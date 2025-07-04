/*
Copyright 2021 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"fmt"
	"time"

	"github.com/go-logr/logr"

	"github.com/IBM/vpc-go-sdk/vpcv1"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	clusterv1beta1 "sigs.k8s.io/cluster-api/api/core/v1beta1" //nolint:staticcheck
	clusterv1 "sigs.k8s.io/cluster-api/api/core/v1beta2"
	"sigs.k8s.io/cluster-api/util"
	v1beta1conditions "sigs.k8s.io/cluster-api/util/deprecated/v1beta1/conditions" //nolint:staticcheck

	infrav1 "sigs.k8s.io/cluster-api-provider-ibmcloud/api/v1beta2"
	"sigs.k8s.io/cluster-api-provider-ibmcloud/cloud/scope"
	"sigs.k8s.io/cluster-api-provider-ibmcloud/pkg/endpoints"
	capibmrecord "sigs.k8s.io/cluster-api-provider-ibmcloud/pkg/record"
)

// IBMVPCMachineReconciler reconciles a IBMVPCMachine object.
type IBMVPCMachineReconciler struct {
	client.Client
	Log             logr.Logger
	Recorder        record.EventRecorder
	ServiceEndpoint []endpoints.ServiceEndpoint
	Scheme          *runtime.Scheme
}

// +kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=ibmvpcmachines,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=ibmvpcmachines/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=cluster.x-k8s.io,resources=machines;machines/status,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=secrets;,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=events,verbs=get;list;watch;create;update;patch

// Reconcile implements controller runtime Reconciler interface and handles reconcileation logic for IBMVPCMachine.
func (r *IBMVPCMachineReconciler) Reconcile(ctx context.Context, req ctrl.Request) (_ ctrl.Result, reterr error) {
	log := r.Log.WithValues("ibmvpcmachine", req.NamespacedName)

	// Fetch the IBMVPCMachine instance.

	ibmVpcMachine := &infrav1.IBMVPCMachine{}
	err := r.Get(ctx, req.NamespacedName, ibmVpcMachine)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}
	// Fetch the Machine.
	machine, err := util.GetOwnerMachine(ctx, r.Client, ibmVpcMachine.ObjectMeta)
	if err != nil {
		return ctrl.Result{}, err
	}
	if machine == nil {
		log.Info("Machine Controller has not yet set OwnerRef")
		return ctrl.Result{}, nil
	}

	// Fetch the Cluster.
	cluster, err := util.GetClusterFromMetadata(ctx, r.Client, ibmVpcMachine.ObjectMeta)
	if err != nil {
		log.Info("Machine is missing cluster label or cluster does not exist")
		return ctrl.Result{}, nil
	}

	log = log.WithValues("cluster", cluster.Name)

	ibmCluster := &infrav1.IBMVPCCluster{}
	ibmVpcClusterName := client.ObjectKey{
		Namespace: ibmVpcMachine.Namespace,
		Name:      cluster.Spec.InfrastructureRef.Name,
	}
	if err := r.Client.Get(ctx, ibmVpcClusterName, ibmCluster); err != nil {
		log.Info("IBMVPCCluster is not available yet")
		return ctrl.Result{}, nil
	}

	// Create the machine scope.
	machineScope, err := scope.NewMachineScope(scope.MachineScopeParams{
		Client:          r.Client,
		Logger:          log,
		Cluster:         cluster,
		IBMVPCCluster:   ibmCluster,
		Machine:         machine,
		IBMVPCMachine:   ibmVpcMachine,
		ServiceEndpoint: r.ServiceEndpoint,
	})
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to create scope: %w", err)
	}

	// Always close the scope when exiting this function, so we can persist any IBMVPCMachine changes.
	defer func() {
		if machineScope != nil {
			if err := machineScope.Close(); err != nil && reterr == nil {
				reterr = err
			}
		}
	}()

	// Handle deleted machines.
	if !ibmVpcMachine.DeletionTimestamp.IsZero() {
		return r.reconcileDelete(machineScope)
	}

	// Handle non-deleted machines.
	return r.reconcileNormal(machineScope)
}

// SetupWithManager creates a new IBMVPCMachine controller for a manager.
func (r *IBMVPCMachineReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&infrav1.IBMVPCMachine{}).
		Complete(r)
}

func (r *IBMVPCMachineReconciler) reconcileNormal(machineScope *scope.MachineScope) (ctrl.Result, error) { //nolint:gocyclo
	if controllerutil.AddFinalizer(machineScope.IBMVPCMachine, infrav1.MachineFinalizer) {
		return ctrl.Result{}, nil
	}

	// Make sure bootstrap data is available and populated.
	if machineScope.Machine.Spec.Bootstrap.DataSecretName == nil {
		machineScope.Info("Bootstrap data secret reference is not yet available")
		return ctrl.Result{RequeueAfter: 1 * time.Minute}, nil
	}

	if machineScope.IBMVPCCluster.Status.Subnet.ID != nil {
		machineScope.IBMVPCMachine.Spec.PrimaryNetworkInterface = infrav1.NetworkInterface{
			Subnet: *machineScope.IBMVPCCluster.Status.Subnet.ID,
		}
	}

	instance, err := r.getOrCreate(machineScope)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to reconcile VSI for IBMVPCMachine %s/%s: %w", machineScope.IBMVPCMachine.Namespace, machineScope.IBMVPCMachine.Name, err)
	}

	machineRunning := false
	if instance != nil {
		// Attempt to tag the Instance.
		if err := machineScope.TagResource(machineScope.IBMVPCCluster.Name, *instance.CRN); err != nil {
			return ctrl.Result{}, fmt.Errorf("error failed to tag machine: %w", err)
		}

		// Set available status' for Machine.
		machineScope.SetInstanceID(*instance.ID)
		if err := machineScope.SetProviderID(instance.ID); err != nil {
			return ctrl.Result{}, fmt.Errorf("error failed to set machine provider id: %w", err)
		}
		machineScope.SetAddresses(instance)
		machineScope.SetInstanceStatus(*instance.Status)

		// Depending on the state of the Machine, update status, conditions, etc.
		switch machineScope.GetInstanceStatus() {
		case vpcv1.InstanceStatusPendingConst:
			machineScope.SetNotReady()
			v1beta1conditions.MarkFalse(machineScope.IBMVPCMachine, infrav1.InstanceReadyCondition, infrav1.InstanceNotReadyReason, clusterv1beta1.ConditionSeverityWarning, "")
		case vpcv1.InstanceStatusStoppedConst:
			machineScope.SetNotReady()
			v1beta1conditions.MarkFalse(machineScope.IBMVPCMachine, infrav1.InstanceReadyCondition, infrav1.InstanceStoppedReason, clusterv1beta1.ConditionSeverityError, "")
		case vpcv1.InstanceStatusFailedConst:
			msg := ""
			healthReasonsLen := len(instance.HealthReasons)
			if healthReasonsLen > 0 {
				// Create a failure message using the last entry's Code and Message fields.
				// TODO(cjschaef): Consider adding the MoreInfo field as well, as it contains a link to IBM Cloud docs.
				msg = fmt.Sprintf("%s: %s", *instance.HealthReasons[healthReasonsLen-1].Code, *instance.HealthReasons[healthReasonsLen-1].Message)
			}
			machineScope.SetNotReady()
			machineScope.SetFailureReason(infrav1.UpdateMachineError)
			machineScope.SetFailureMessage(msg)
			v1beta1conditions.MarkFalse(machineScope.IBMVPCMachine, infrav1.InstanceReadyCondition, infrav1.InstanceErroredReason, clusterv1beta1.ConditionSeverityError, "%s", msg)
			capibmrecord.Warnf(machineScope.IBMVPCMachine, "FailedBuildInstance", "Failed to build the instance - %s", msg)
			return ctrl.Result{}, nil
		case vpcv1.InstanceStatusRunningConst:
			machineRunning = true
		default:
			machineScope.SetNotReady()
			machineScope.V(3).Info("unexpected vpc instance status", "instanceStatus", *instance.Status, "instanceID", machineScope.GetInstanceID())
			v1beta1conditions.MarkUnknown(machineScope.IBMVPCMachine, infrav1.InstanceReadyCondition, "", "")
		}
	} else {
		machineScope.SetNotReady()
		v1beta1conditions.MarkUnknown(machineScope.IBMVPCMachine, infrav1.InstanceReadyCondition, infrav1.InstanceStateUnknownReason, "")
	}

	// Check if the Machine is running.
	if !machineRunning {
		// Requeue after 1 minute if machine is not running.
		return ctrl.Result{RequeueAfter: 1 * time.Minute}, nil
	}

	// Rely on defined VPC Load Balancer Pool Members first before falling back to hardcoded defaults.
	if len(machineScope.IBMVPCMachine.Spec.LoadBalancerPoolMembers) > 0 {
		needsRequeue := false
		for _, poolMember := range machineScope.IBMVPCMachine.Spec.LoadBalancerPoolMembers {
			requeue, err := machineScope.ReconcileVPCLoadBalancerPoolMember(poolMember)
			if err != nil {
				return ctrl.Result{}, fmt.Errorf("error failed to reconcile machine's pool member: %w", err)
			} else if requeue {
				needsRequeue = true
			}
		}

		// If any VPC Load Balancer Pool Member needs reconciliation, requeue.
		if needsRequeue {
			return ctrl.Result{RequeueAfter: 1 * time.Minute}, nil
		}
	} else {
		// Otherwise, default to previous Load Balancer Pool Member configuration.
		_, ok := machineScope.IBMVPCMachine.Labels[clusterv1.MachineControlPlaneNameLabel]
		if err = machineScope.SetProviderID(instance.ID); err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to set provider id IBMVPCMachine %s/%s: %w", machineScope.IBMVPCMachine.Namespace, machineScope.IBMVPCMachine.Name, err)
		}
		if ok {
			if instance.PrimaryNetworkInterface.PrimaryIP.Address == nil || *instance.PrimaryNetworkInterface.PrimaryIP.Address == "0.0.0.0" {
				return ctrl.Result{}, fmt.Errorf("invalid primary ip address")
			}
			internalIP := instance.PrimaryNetworkInterface.PrimaryIP.Address
			port := int64(machineScope.APIServerPort())
			poolMember, err := machineScope.CreateVPCLoadBalancerPoolMember(internalIP, port)
			if err != nil {
				return ctrl.Result{}, fmt.Errorf("failed to bind port %d to control plane %s/%s: %w", port, machineScope.IBMVPCMachine.Namespace, machineScope.IBMVPCMachine.Name, err)
			}
			if poolMember != nil && *poolMember.ProvisioningStatus != string(infrav1.VPCLoadBalancerStateActive) {
				return ctrl.Result{RequeueAfter: 1 * time.Minute}, nil
			}
		}
	}

	// With a running machine and all Load Balancer Pool Members reconciled, mark machine as ready.
	machineScope.SetReady()
	v1beta1conditions.MarkTrue(machineScope.IBMVPCMachine, infrav1.InstanceReadyCondition)
	return ctrl.Result{}, nil
}

func (r *IBMVPCMachineReconciler) getOrCreate(scope *scope.MachineScope) (*vpcv1.Instance, error) {
	instance, err := scope.CreateMachine()
	return instance, err
}

func (r *IBMVPCMachineReconciler) reconcileDelete(scope *scope.MachineScope) (_ ctrl.Result, reterr error) {
	scope.Info("Handling deleted IBMVPCMachine")

	if _, ok := scope.IBMVPCMachine.Labels[clusterv1.MachineControlPlaneNameLabel]; ok {
		if err := scope.DeleteVPCLoadBalancerPoolMember(); err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to delete loadBalancer pool member: %w", err)
		}
	}

	if err := scope.DeleteMachine(); err != nil {
		scope.Info("error deleting IBMVPCMachine")
		return ctrl.Result{}, fmt.Errorf("error deleting IBMVPCMachine %s/%s: %w", scope.IBMVPCMachine.Namespace, scope.IBMVPCMachine.Spec.Name, err)
	}

	defer func() {
		if reterr == nil {
			// VSI is deleted so remove the finalizer.
			controllerutil.RemoveFinalizer(scope.IBMVPCMachine, infrav1.MachineFinalizer)
		}
	}()

	return ctrl.Result{}, nil
}
