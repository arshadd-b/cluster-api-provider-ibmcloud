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

package webhooks

import (
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	infrav1 "sigs.k8s.io/cluster-api-provider-ibmcloud/api/v1beta2"
)

func TestIBMPowerVSCluster_create(t *testing.T) {
	tests := []struct {
		name           string
		powervsCluster *infrav1.IBMPowerVSCluster
		wantErr        bool
	}{
		{
			name: "Should allow if either Network ID or name is set",
			powervsCluster: &infrav1.IBMPowerVSCluster{
				Spec: infrav1.IBMPowerVSClusterSpec{
					ServiceInstanceID: "capi-si-id",
					Network: infrav1.IBMPowerVSResourceReference{
						ID: ptr.To("capi-net-id"),
					},
				},
			},
			wantErr: false,
		},
		{
			name: "Should error if both Network ID and name are set",
			powervsCluster: &infrav1.IBMPowerVSCluster{
				Spec: infrav1.IBMPowerVSClusterSpec{
					ServiceInstanceID: "capi-si-id",
					Network: infrav1.IBMPowerVSResourceReference{
						ID:   ptr.To("capi-net-id"),
						Name: ptr.To("capi-net"),
					},
				},
			},
			wantErr: true,
		},
		{
			name: "Should error if all Network ID, name and regex are set",
			powervsCluster: &infrav1.IBMPowerVSCluster{
				Spec: infrav1.IBMPowerVSClusterSpec{
					ServiceInstanceID: "capi-si-id",
					Network: infrav1.IBMPowerVSResourceReference{
						ID:    ptr.To("capi-net-id"),
						Name:  ptr.To("capi-net"),
						RegEx: ptr.To("^capi$"),
					},
				},
			},
			wantErr: true,
		},
		{
			name: "Should error if both Network name and DHCP name are set",
			powervsCluster: &infrav1.IBMPowerVSCluster{
				Spec: infrav1.IBMPowerVSClusterSpec{
					ServiceInstanceID: "capi-si-id",
					Network: infrav1.IBMPowerVSResourceReference{
						Name: ptr.To("capi-net"),
					},
					DHCPServer: &infrav1.DHCPServer{
						Name: ptr.To("capi-dhcp"),
					},
				},
			},
			wantErr: true,
		},
		{
			name: "Should error if both Network id and DHCP name are set",
			powervsCluster: &infrav1.IBMPowerVSCluster{
				Spec: infrav1.IBMPowerVSClusterSpec{
					ServiceInstanceID: "capi-si-id",
					Network: infrav1.IBMPowerVSResourceReference{
						ID: ptr.To("capi-net-id"),
					},
					DHCPServer: &infrav1.DHCPServer{
						Name: ptr.To("capi-dhcp"),
					},
				},
			},
			wantErr: true,
		},
		{
			name: "Should error if both Network name and DHCP id are set",
			powervsCluster: &infrav1.IBMPowerVSCluster{
				Spec: infrav1.IBMPowerVSClusterSpec{
					ServiceInstanceID: "capi-si-id",
					Network: infrav1.IBMPowerVSResourceReference{
						Name: ptr.To("capi-net"),
					},
					DHCPServer: &infrav1.DHCPServer{
						ID: ptr.To("capi-dhcp-id"),
					},
				},
			},
			wantErr: true,
		},
		{
			name: "Should error if both Network id and DHCP id are set",
			powervsCluster: &infrav1.IBMPowerVSCluster{
				Spec: infrav1.IBMPowerVSClusterSpec{
					ServiceInstanceID: "capi-si-id",
					Network: infrav1.IBMPowerVSResourceReference{
						ID: ptr.To("capi-net-id"),
					},
					DHCPServer: &infrav1.DHCPServer{
						ID: ptr.To("capi-dhcp-id"),
					},
				},
			},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cluster := tc.powervsCluster.DeepCopy()
			cluster.ObjectMeta = metav1.ObjectMeta{
				GenerateName: "capi-cluster-",
				Namespace:    "default",
			}

			if err := testEnv.Create(ctx, cluster); (err != nil) != tc.wantErr {
				t.Errorf("ValidateCreate() error = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}

func TestIBMPowerVSCluster_update(t *testing.T) {
	tests := []struct {
		name              string
		oldPowervsCluster *infrav1.IBMPowerVSCluster
		newPowervsCluster *infrav1.IBMPowerVSCluster
		wantErr           bool
	}{
		{
			name: "Should allow if either Network ID or name is set",
			oldPowervsCluster: &infrav1.IBMPowerVSCluster{
				Spec: infrav1.IBMPowerVSClusterSpec{
					ServiceInstanceID: "capi-si-id",
					Network: infrav1.IBMPowerVSResourceReference{
						ID: ptr.To("capi-net-id"),
					},
				},
			},
			newPowervsCluster: &infrav1.IBMPowerVSCluster{
				Spec: infrav1.IBMPowerVSClusterSpec{
					ServiceInstanceID: "capi-si-id",
					Network: infrav1.IBMPowerVSResourceReference{
						ID: ptr.To("capi-net-id"),
					},
				},
			},
			wantErr: false,
		},
		{
			name: "Should error if both Network ID and name are set",
			oldPowervsCluster: &infrav1.IBMPowerVSCluster{
				Spec: infrav1.IBMPowerVSClusterSpec{
					ServiceInstanceID: "capi-si-id",
					Network: infrav1.IBMPowerVSResourceReference{
						ID: ptr.To("capi-net-id"),
					},
				},
			},
			newPowervsCluster: &infrav1.IBMPowerVSCluster{
				Spec: infrav1.IBMPowerVSClusterSpec{
					ServiceInstanceID: "capi-si-id",
					Network: infrav1.IBMPowerVSResourceReference{
						ID:   ptr.To("capi-net-id"),
						Name: ptr.To("capi-net-name"),
					},
				},
			},
			wantErr: true,
		},
		{
			name: "Should allow if Network ID is set",
			oldPowervsCluster: &infrav1.IBMPowerVSCluster{
				Spec: infrav1.IBMPowerVSClusterSpec{
					ServiceInstanceID: "capi-si-id",
					Network: infrav1.IBMPowerVSResourceReference{
						RegEx: ptr.To("^capi-net-id$"),
					},
				},
			},
			newPowervsCluster: &infrav1.IBMPowerVSCluster{
				Spec: infrav1.IBMPowerVSClusterSpec{
					ServiceInstanceID: "capi-si-id",
					Network: infrav1.IBMPowerVSResourceReference{
						RegEx: ptr.To("^capi-net-id$"),
					},
				},
			},
			wantErr: false,
		},
		{
			name: "Should error if all Network ID, name and regex are set",
			oldPowervsCluster: &infrav1.IBMPowerVSCluster{
				Spec: infrav1.IBMPowerVSClusterSpec{
					ServiceInstanceID: "capi-si-id",
					Network: infrav1.IBMPowerVSResourceReference{
						ID: ptr.To("capi-net-id"),
					},
				},
			},
			newPowervsCluster: &infrav1.IBMPowerVSCluster{
				Spec: infrav1.IBMPowerVSClusterSpec{
					ServiceInstanceID: "capi-si-id",
					Network: infrav1.IBMPowerVSResourceReference{
						ID:    ptr.To("capi-net-id"),
						Name:  ptr.To("capi-net-name"),
						RegEx: ptr.To("^capi-net-id$"),
					},
				},
			},
			wantErr: true,
		},
		{
			name: "Should error if the additionalListener selector is changed for same port and protocol",
			oldPowervsCluster: &infrav1.IBMPowerVSCluster{
				Spec: infrav1.IBMPowerVSClusterSpec{
					ServiceInstanceID: "capi-si-id",
					Network: infrav1.IBMPowerVSResourceReference{
						ID: ptr.To("capi-net-id"),
					},
					LoadBalancers: []infrav1.VPCLoadBalancerSpec{
						{
							Name: "load-balancer-1",
							AdditionalListeners: []infrav1.AdditionalListenerSpec{
								{
									Port:     23,
									Protocol: &infrav1.VPCLoadBalancerListenerProtocolTCP,
									Selector: metav1.LabelSelector{
										MatchLabels: map[string]string{
											"listener-selector": "port-23",
										},
									},
								},
							},
						},
					},
				},
			},
			newPowervsCluster: &infrav1.IBMPowerVSCluster{
				Spec: infrav1.IBMPowerVSClusterSpec{
					ServiceInstanceID: "capi-si-id",
					Network: infrav1.IBMPowerVSResourceReference{
						ID: ptr.To("capi-net-id"),
					},
					LoadBalancers: []infrav1.VPCLoadBalancerSpec{
						{
							Name: "load-balancer-1",
							AdditionalListeners: []infrav1.AdditionalListenerSpec{
								{
									Port:     23,
									Protocol: &infrav1.VPCLoadBalancerListenerProtocolTCP,
									Selector: metav1.LabelSelector{
										MatchLabels: map[string]string{
											"listener-selector": "port-23-1",
										},
									},
								},
							},
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "Should work if there is an additional listener added",
			oldPowervsCluster: &infrav1.IBMPowerVSCluster{
				Spec: infrav1.IBMPowerVSClusterSpec{
					ServiceInstanceID: "capi-si-id",
					Network: infrav1.IBMPowerVSResourceReference{
						ID: ptr.To("capi-net-id"),
					},
					LoadBalancers: []infrav1.VPCLoadBalancerSpec{
						{
							Name: "load-balancer-1",
							AdditionalListeners: []infrav1.AdditionalListenerSpec{
								{
									Port:     23,
									Protocol: &infrav1.VPCLoadBalancerListenerProtocolTCP,
									Selector: metav1.LabelSelector{
										MatchLabels: map[string]string{
											"listener-selector": "port-23",
										},
									},
								},
							},
						},
					},
				},
			},
			newPowervsCluster: &infrav1.IBMPowerVSCluster{
				Spec: infrav1.IBMPowerVSClusterSpec{
					ServiceInstanceID: "capi-si-id",
					Network: infrav1.IBMPowerVSResourceReference{
						ID: ptr.To("capi-net-id"),
					},
					LoadBalancers: []infrav1.VPCLoadBalancerSpec{
						{
							Name: "load-balancer-1",
							AdditionalListeners: []infrav1.AdditionalListenerSpec{
								{
									Port:     23,
									Protocol: &infrav1.VPCLoadBalancerListenerProtocolTCP,
									Selector: metav1.LabelSelector{
										MatchLabels: map[string]string{
											"listener-selector": "port-23",
										},
									},
								},
								{
									Port:     25,
									Protocol: &infrav1.VPCLoadBalancerListenerProtocolTCP,
									Selector: metav1.LabelSelector{
										MatchLabels: map[string]string{
											"listener-selector": "port-25",
										},
									},
								},
							},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "Should work if the additionalListener selector is updated with new port and protocol",
			oldPowervsCluster: &infrav1.IBMPowerVSCluster{
				Spec: infrav1.IBMPowerVSClusterSpec{
					ServiceInstanceID: "capi-si-id",
					Network: infrav1.IBMPowerVSResourceReference{
						ID: ptr.To("capi-net-id"),
					},
					LoadBalancers: []infrav1.VPCLoadBalancerSpec{
						{
							Name: "load-balancer-1",
							AdditionalListeners: []infrav1.AdditionalListenerSpec{
								{
									Port:     23,
									Protocol: &infrav1.VPCLoadBalancerListenerProtocolTCP,
									Selector: metav1.LabelSelector{
										MatchLabels: map[string]string{
											"listener-selector": "port-23",
										},
									},
								},
							},
						},
					},
				},
			},
			newPowervsCluster: &infrav1.IBMPowerVSCluster{
				Spec: infrav1.IBMPowerVSClusterSpec{
					ServiceInstanceID: "capi-si-id",
					Network: infrav1.IBMPowerVSResourceReference{
						ID: ptr.To("capi-net-id"),
					},
					LoadBalancers: []infrav1.VPCLoadBalancerSpec{
						{
							Name: "load-balancer-1",
							AdditionalListeners: []infrav1.AdditionalListenerSpec{
								{
									Port:     25,
									Protocol: &infrav1.VPCLoadBalancerListenerProtocolTCP,
									Selector: metav1.LabelSelector{
										MatchLabels: map[string]string{
											"listener-selector": "port-25",
										},
									},
								},
							},
						},
					},
				},
			},
			wantErr: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cluster := tc.oldPowervsCluster.DeepCopy()
			cluster.ObjectMeta = metav1.ObjectMeta{
				GenerateName: "capi-cluster-",
				Namespace:    "default",
			}

			if err := testEnv.Create(ctx, cluster); err != nil {
				t.Errorf("failed to create cluster: %v", err)
			}

			cluster.Spec = tc.newPowervsCluster.Spec
			if err := testEnv.Update(ctx, cluster); (err != nil) != tc.wantErr {
				t.Errorf("ValidateUpdate() error = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}
