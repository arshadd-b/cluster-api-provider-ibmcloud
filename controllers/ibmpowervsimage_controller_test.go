/*
Copyright 2022 The Kubernetes Authors.

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
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/IBM-Cloud/power-go-client/power/models"
	"go.uber.org/mock/gomock"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog/v2"
	"k8s.io/utils/ptr"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	clusterv1beta1 "sigs.k8s.io/cluster-api/api/core/v1beta1" //nolint:staticcheck
	"sigs.k8s.io/cluster-api/util"
	v1beta1conditions "sigs.k8s.io/cluster-api/util/deprecated/v1beta1/conditions" //nolint:staticcheck

	infrav1 "sigs.k8s.io/cluster-api-provider-ibmcloud/api/v1beta2"
	"sigs.k8s.io/cluster-api-provider-ibmcloud/cloud/scope"
	"sigs.k8s.io/cluster-api-provider-ibmcloud/pkg/cloud/services/powervs/mock"

	. "github.com/onsi/gomega"
)

func TestIBMPowerVSImageReconciler_Reconcile(t *testing.T) {
	testCases := []struct {
		name           string
		powervsCluster *infrav1.IBMPowerVSCluster
		powervsImage   *infrav1.IBMPowerVSImage
		expectError    bool
	}{
		{
			name:        "Should Reconcile successfully if IBMPowerVSImage is not found",
			expectError: false,
		},
		{
			name: "Should not Reconcile if failed to find IBMPowerVSCluster",
			powervsImage: &infrav1.IBMPowerVSImage{
				ObjectMeta: metav1.ObjectMeta{
					Name: "capi-image",
				},
				Spec: infrav1.IBMPowerVSImageSpec{
					ClusterName: "capi-powervs-cluster",
					Object:      ptr.To("capi-image.ova.gz"),
					Region:      ptr.To("us-south"),
					Bucket:      ptr.To("capi-bucket"),
				},
			},
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			g := NewWithT(t)
			reconciler := &IBMPowerVSImageReconciler{
				Client: testEnv.Client,
			}

			ns, err := testEnv.CreateNamespace(ctx, fmt.Sprintf("namespace-%s", util.RandomString(5)))
			g.Expect(err).To(BeNil())

			createObject(g, tc.powervsImage, ns.Name)
			defer cleanupObject(g, tc.powervsImage)

			if tc.powervsImage != nil {
				g.Eventually(func() bool {
					machine := &infrav1.IBMPowerVSImage{}
					key := client.ObjectKey{
						Name:      tc.powervsImage.Name,
						Namespace: ns.Name,
					}
					err = testEnv.Get(ctx, key, machine)
					return err == nil
				}, 10*time.Second).Should(Equal(true))

				_, err := reconciler.Reconcile(ctx, ctrl.Request{
					NamespacedName: client.ObjectKey{
						Namespace: tc.powervsImage.Namespace,
						Name:      tc.powervsImage.Name,
					},
				})
				if tc.expectError {
					g.Expect(err).ToNot(BeNil())
				} else {
					g.Expect(err).To(BeNil())
				}
			} else {
				_, err = reconciler.Reconcile(ctx, ctrl.Request{
					NamespacedName: client.ObjectKey{
						Namespace: "default",
						Name:      "test",
					},
				})
				g.Expect(err).To(BeNil())
			}
		})
	}
}

func TestIBMPowerVSImageReconciler_reconcile(t *testing.T) {
	var (
		mockpowervs *mock.MockPowerVS
		mockCtrl    *gomock.Controller
		reconciler  IBMPowerVSImageReconciler
	)

	setup := func(t *testing.T) {
		t.Helper()
		mockCtrl = gomock.NewController(t)
		mockpowervs = mock.NewMockPowerVS(mockCtrl)
		recorder := record.NewFakeRecorder(2)
		reconciler = IBMPowerVSImageReconciler{
			Client:   testEnv.Client,
			Recorder: recorder,
		}
	}
	teardown := func() {
		mockCtrl.Finish()
	}

	t.Run("Reconciling IBMPowerVSImage ", func(t *testing.T) {
		t.Run("Should reconcile by setting the owner reference", func(t *testing.T) {
			g := NewWithT(t)
			setup(t)
			t.Cleanup(teardown)
			powervsCluster := &infrav1.IBMPowerVSCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: "capi-powervs-cluster"},
			}
			imageScope := &scope.PowerVSImageScope{
				Logger: klog.Background(),
				IBMPowerVSImage: &infrav1.IBMPowerVSImage{
					ObjectMeta: metav1.ObjectMeta{
						Name: "capi-image",
					},
					Spec: infrav1.IBMPowerVSImageSpec{
						ClusterName: "capi-powervs-cluster",
						Object:      ptr.To("capi-image.ova.gz"),
						Region:      ptr.To("us-south"),
						Bucket:      ptr.To("capi-bucket"),
					},
				},
			}
			_, err := reconciler.reconcile(powervsCluster, imageScope)
			g.Expect(err).To(BeNil())
		})
		t.Run("Reconciling an image import job", func(t *testing.T) {
			g := NewWithT(t)
			setup(t)
			t.Cleanup(teardown)
			const jobID = "job-1"
			powervsCluster := &infrav1.IBMPowerVSCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: "capi-powervs-cluster",
					UID:  "1",
				},
				Spec: infrav1.IBMPowerVSClusterSpec{
					ServiceInstanceID: "service-instance-1",
				},
			}
			powervsImage := &infrav1.IBMPowerVSImage{
				ObjectMeta: metav1.ObjectMeta{
					Name: "capi-image",
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion: infrav1.GroupVersion.String(),
							Kind:       "IBMPowerVSCluster",
							Name:       "capi-powervs-cluster",
							UID:        "1",
						},
					},
					Finalizers: []string{infrav1.IBMPowerVSImageFinalizer},
				},
				Spec: infrav1.IBMPowerVSImageSpec{
					ClusterName: "capi-powervs-cluster",
					Object:      ptr.To("capi-image.ova.gz"),
					Region:      ptr.To("us-south"),
					Bucket:      ptr.To("capi-bucket"),
				},
			}

			mockclient := fake.NewClientBuilder().WithObjects([]client.Object{powervsCluster, powervsImage}...).Build()
			imageScope := &scope.PowerVSImageScope{
				Logger:           klog.Background(),
				Client:           mockclient,
				IBMPowerVSImage:  powervsImage,
				IBMPowerVSClient: mockpowervs,
			}

			imageScope.IBMPowerVSImage.Status.JobID = jobID
			t.Run("When failed to get the import job using jobID", func(_ *testing.T) {
				mockpowervs.EXPECT().GetJob(gomock.AssignableToTypeOf(jobID)).Return(nil, errors.New("Error finding the job"))
				result, err := reconciler.reconcile(powervsCluster, imageScope)
				g.Expect(err).To(Not(BeNil()))
				g.Expect(result.RequeueAfter).To(Not(BeZero()))
				g.Expect(imageScope.IBMPowerVSImage.Finalizers).To(ContainElement(infrav1.IBMPowerVSImageFinalizer))
			})
			job := &models.Job{
				ID: ptr.To(jobID),
				Status: &models.Status{
					State: ptr.To("queued"),
				},
			}
			t.Run("When import job status is queued", func(_ *testing.T) {
				mockpowervs.EXPECT().GetJob(gomock.AssignableToTypeOf(jobID)).Return(job, nil)
				result, err := reconciler.reconcile(powervsCluster, imageScope)
				g.Expect(err).To(BeNil())
				g.Expect(imageScope.IBMPowerVSImage.Finalizers).To(ContainElement(infrav1.IBMPowerVSImageFinalizer))
				g.Expect(imageScope.IBMPowerVSImage.Status.Ready).To(Equal(false))
				g.Expect(imageScope.IBMPowerVSImage.Status.ImageState).To(BeEquivalentTo(infrav1.PowerVSImageStateQue))
				expectConditionsImage(g, imageScope.IBMPowerVSImage, []conditionAssertion{{infrav1.ImageImportedCondition, corev1.ConditionFalse, clusterv1beta1.ConditionSeverityInfo, string(infrav1.PowerVSImageStateQue)}})
				g.Expect(result.RequeueAfter).To(Not(BeZero()))
			})
			t.Run("When importing image is still in progress", func(_ *testing.T) {
				job.Status.State = ptr.To("")
				mockpowervs.EXPECT().GetJob(gomock.AssignableToTypeOf("job-1")).Return(job, nil)
				result, err := reconciler.reconcile(powervsCluster, imageScope)
				g.Expect(err).To(BeNil())
				g.Expect(imageScope.IBMPowerVSImage.Finalizers).To(ContainElement(infrav1.IBMPowerVSImageFinalizer))
				g.Expect(imageScope.IBMPowerVSImage.Status.Ready).To(Equal(false))
				g.Expect(imageScope.IBMPowerVSImage.Status.ImageState).To(BeEquivalentTo(infrav1.PowerVSImageStateImporting))
				expectConditionsImage(g, imageScope.IBMPowerVSImage, []conditionAssertion{{infrav1.ImageImportedCondition, corev1.ConditionFalse, clusterv1beta1.ConditionSeverityInfo, *job.Status.State}})
				g.Expect(result.RequeueAfter).To(Not(BeZero()))
			})
			t.Run("When import job status is failed", func(_ *testing.T) {
				job.Status.State = ptr.To("failed")
				mockpowervs.EXPECT().GetJob(gomock.AssignableToTypeOf("job-1")).Return(job, nil)
				result, err := reconciler.reconcile(powervsCluster, imageScope)
				g.Expect(err).To(Not(BeNil()))
				g.Expect(imageScope.IBMPowerVSImage.Finalizers).To(ContainElement(infrav1.IBMPowerVSImageFinalizer))
				g.Expect(imageScope.IBMPowerVSImage.Status.Ready).To(Equal(false))
				g.Expect(imageScope.IBMPowerVSImage.Status.ImageState).To(BeEquivalentTo(infrav1.PowerVSImageStateFailed))
				expectConditionsImage(g, imageScope.IBMPowerVSImage, []conditionAssertion{{infrav1.ImageImportedCondition, corev1.ConditionFalse, clusterv1beta1.ConditionSeverityError, infrav1.ImageImportFailedReason}})
				g.Expect(result.RequeueAfter).To(Not(BeZero()))
			})
			job.Status.State = ptr.To("completed")
			images := &models.Images{
				Images: []*models.ImageReference{
					{
						Name:    ptr.To("capi-image"),
						ImageID: ptr.To("capi-image-id"),
					},
				},
			}
			t.Run("When import job status is completed and fails to get the image details", func(_ *testing.T) {
				mockpowervs.EXPECT().GetJob(gomock.AssignableToTypeOf("job-1")).Return(job, nil)
				mockpowervs.EXPECT().GetAllImage().Return(images, nil)
				mockpowervs.EXPECT().GetImage(gomock.AssignableToTypeOf("capi-image-id")).Return(nil, errors.New("Failed to the image details"))
				result, err := reconciler.reconcile(powervsCluster, imageScope)
				g.Expect(err).To(Not(BeNil()))
				g.Expect(result.RequeueAfter).To(BeZero())
				expectConditionsImage(g, imageScope.IBMPowerVSImage, []conditionAssertion{{conditionType: infrav1.ImageImportedCondition, status: corev1.ConditionTrue}})
				g.Expect(imageScope.IBMPowerVSImage.Finalizers).To(ContainElement(infrav1.IBMPowerVSImageFinalizer))
			})
			image := &models.Image{
				Name:    ptr.To("capi-image"),
				ImageID: ptr.To("capi-image-id"),
				State:   "queued",
			}
			t.Run("When import job status is completed and image state is queued", func(_ *testing.T) {
				mockpowervs.EXPECT().GetJob(gomock.AssignableToTypeOf("job-1")).Return(job, nil)
				mockpowervs.EXPECT().GetAllImage().Return(images, nil)
				mockpowervs.EXPECT().GetImage(gomock.AssignableToTypeOf("capi-image-id")).Return(image, nil)
				result, err := reconciler.reconcile(powervsCluster, imageScope)
				g.Expect(err).To(BeNil())
				g.Expect(imageScope.IBMPowerVSImage.Finalizers).To(ContainElement(infrav1.IBMPowerVSImageFinalizer))
				g.Expect(imageScope.IBMPowerVSImage.Status.Ready).To(Equal(false))
				expectConditionsImage(g, imageScope.IBMPowerVSImage, []conditionAssertion{{infrav1.ImageReadyCondition, corev1.ConditionFalse, clusterv1beta1.ConditionSeverityWarning, infrav1.ImageNotReadyReason}})
				g.Expect(result.RequeueAfter).To(Not(BeZero()))
			})
			t.Run("When import job status is completed and image state is undefined", func(_ *testing.T) {
				image.State = "unknown"
				mockpowervs.EXPECT().GetJob(gomock.AssignableToTypeOf("job-1")).Return(job, nil)
				mockpowervs.EXPECT().GetAllImage().Return(images, nil)
				mockpowervs.EXPECT().GetImage(gomock.AssignableToTypeOf("capi-image-id")).Return(image, nil)
				result, err := reconciler.reconcile(powervsCluster, imageScope)
				g.Expect(err).To(BeNil())
				g.Expect(imageScope.IBMPowerVSImage.Finalizers).To(ContainElement(infrav1.IBMPowerVSImageFinalizer))
				expectConditionsImage(g, imageScope.IBMPowerVSImage, []conditionAssertion{{infrav1.ImageReadyCondition, corev1.ConditionUnknown, "", ""}})
				g.Expect(imageScope.IBMPowerVSImage.Status.Ready).To(Equal(false))
				g.Expect(result.RequeueAfter).To(Not(BeZero()))
			})
			t.Run("When import job status is completed and image state is active", func(_ *testing.T) {
				image.State = "active"
				mockpowervs.EXPECT().GetJob(gomock.AssignableToTypeOf("job-1")).Return(job, nil)
				mockpowervs.EXPECT().GetAllImage().Return(images, nil)
				mockpowervs.EXPECT().GetImage(gomock.AssignableToTypeOf("capi-image-id")).Return(image, nil)
				result, err := reconciler.reconcile(powervsCluster, imageScope)
				g.Expect(err).To(BeNil())
				expectConditionsImage(g, imageScope.IBMPowerVSImage, []conditionAssertion{{conditionType: infrav1.ImageReadyCondition, status: corev1.ConditionTrue}})
				g.Expect(imageScope.IBMPowerVSImage.Finalizers).To(ContainElement(infrav1.IBMPowerVSImageFinalizer))
				g.Expect(imageScope.IBMPowerVSImage.Status.Ready).To(Equal(true))
				g.Expect(result.RequeueAfter).To(BeZero())
			})
		})
	})
}

func TestIBMPowerVSImageReconciler_delete(t *testing.T) {
	var (
		mockpowervs *mock.MockPowerVS
		mockCtrl    *gomock.Controller
		reconciler  IBMPowerVSImageReconciler
		imageScope  *scope.PowerVSImageScope
	)

	setup := func(t *testing.T) {
		t.Helper()
		mockCtrl = gomock.NewController(t)
		mockpowervs = mock.NewMockPowerVS(mockCtrl)
		recorder := record.NewFakeRecorder(2)
		reconciler = IBMPowerVSImageReconciler{
			Client:   testEnv.Client,
			Recorder: recorder,
		}
		imageScope = &scope.PowerVSImageScope{
			Logger:           klog.Background(),
			IBMPowerVSImage:  &infrav1.IBMPowerVSImage{},
			IBMPowerVSClient: mockpowervs,
		}
	}
	teardown := func() {
		mockCtrl.Finish()
	}

	t.Run("Reconcile deleting IBMPowerVSImage ", func(t *testing.T) {
		t.Run("Should not delete IBMPowerVSImage is neither job ID nor image ID are set", func(t *testing.T) {
			g := NewWithT(t)
			setup(t)
			t.Cleanup(teardown)
			imageScope.IBMPowerVSImage.Finalizers = []string{infrav1.IBMPowerVSImageFinalizer}
			_, err := reconciler.reconcileDelete(imageScope)
			g.Expect(err).To(BeNil())
			g.Expect(imageScope.IBMPowerVSImage.Finalizers).To(Not(ContainElement(infrav1.IBMPowerVSImageFinalizer)))
		})
		t.Run("Should fail to delete the import image job", func(t *testing.T) {
			g := NewWithT(t)
			setup(t)
			t.Cleanup(teardown)
			imageScope.IBMPowerVSImage.Status.JobID = "job-1"
			imageScope.IBMPowerVSImage.Finalizers = []string{infrav1.IBMPowerVSImageFinalizer}
			mockpowervs.EXPECT().DeleteJob(gomock.AssignableToTypeOf("job-1")).Return(errors.New("Failed to deleted the import job"))
			_, err := reconciler.reconcileDelete(imageScope)
			g.Expect(err).To(Not(BeNil()))
			g.Expect(imageScope.IBMPowerVSImage.Finalizers).To(ContainElement(infrav1.IBMPowerVSImageFinalizer))
		})
		t.Run("Should delete the import image job", func(t *testing.T) {
			g := NewWithT(t)
			setup(t)
			t.Cleanup(teardown)
			imageScope.IBMPowerVSImage.Status.JobID = "job-1"
			imageScope.IBMPowerVSImage.Finalizers = []string{infrav1.IBMPowerVSImageFinalizer}
			mockpowervs.EXPECT().DeleteJob(gomock.AssignableToTypeOf("job-1")).Return(nil)
			_, err := reconciler.reconcileDelete(imageScope)
			g.Expect(err).To(BeNil())
			g.Expect(imageScope.IBMPowerVSImage.Finalizers).To(Not(ContainElement(infrav1.IBMPowerVSImageFinalizer)))
		})
		t.Run("Should fail to delete the image using ID when delete policy is not to retain it", func(t *testing.T) {
			g := NewWithT(t)
			setup(t)
			t.Cleanup(teardown)
			imageScope.IBMPowerVSImage.Status.ImageID = "capi-image-id"
			imageScope.IBMPowerVSImage.Finalizers = []string{infrav1.IBMPowerVSImageFinalizer}
			mockpowervs.EXPECT().DeleteImage(gomock.AssignableToTypeOf("capi-image-id")).Return(errors.New("Failed to delete the image"))
			_, err := reconciler.reconcileDelete(imageScope)
			g.Expect(err).To(Not(BeNil()))
			g.Expect(imageScope.IBMPowerVSImage.Finalizers).To(ContainElement(infrav1.IBMPowerVSImageFinalizer))
		})
		t.Run("Should not delete the image using ID when delete policy is to retain it", func(t *testing.T) {
			g := NewWithT(t)
			setup(t)
			t.Cleanup(teardown)
			imageScope.IBMPowerVSImage.Status.ImageID = "capi-image-id"
			imageScope.IBMPowerVSImage.Finalizers = []string{infrav1.IBMPowerVSImageFinalizer}
			imageScope.IBMPowerVSImage.Spec.DeletePolicy = "retain"
			_, err := reconciler.reconcileDelete(imageScope)
			g.Expect(err).To(BeNil())
			g.Expect(imageScope.IBMPowerVSImage.Finalizers).To(Not(ContainElement(infrav1.IBMPowerVSImageFinalizer)))
		})
	})
}

func expectConditionsImage(g *WithT, m *infrav1.IBMPowerVSImage, expected []conditionAssertion) {
	g.Expect(len(m.Status.Conditions)).To(BeNumerically(">=", len(expected)))
	for _, c := range expected {
		actual := v1beta1conditions.Get(m, c.conditionType)
		g.Expect(actual).To(Not(BeNil()))
		g.Expect(actual.Type).To(Equal(c.conditionType))
		g.Expect(actual.Status).To(Equal(c.status))
		g.Expect(actual.Severity).To(Equal(c.severity))
		g.Expect(actual.Reason).To(Equal(c.reason))
	}
}
