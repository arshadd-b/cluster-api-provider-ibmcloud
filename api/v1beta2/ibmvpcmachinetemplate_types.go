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

package v1beta2

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// IBMVPCMachineTemplateSpec defines the desired state of IBMVPCMachineTemplate.
type IBMVPCMachineTemplateSpec struct {
	Template IBMVPCMachineTemplateResource `json:"template"`
}

// IBMVPCMachineTemplateResource describes the data needed to create am IBMVPCMachine from a template.
type IBMVPCMachineTemplateResource struct {
	// Spec is the specification of the desired behavior of the machine.
	Spec IBMVPCMachineSpec `json:"spec"`
}

// IBMVPCMachineTemplateStatus defines the observed state of IBMVPCMachineTemplate.
type IBMVPCMachineTemplateStatus struct {
	// Capacity defines the resource capacity for this machine.
	// This value is used for autoscaling from zero operations as defined in:
	// https://github.com/kubernetes-sigs/cluster-api/blob/main/docs/proposals/20210310-opt-in-autoscaling-from-zero.md
	// +optional
	Capacity corev1.ResourceList `json:"capacity,omitempty"`
}

//+kubebuilder:subresource:status
// +kubebuilder:object:root=true
// +kubebuilder:resource:path=ibmvpcmachinetemplates,scope=Namespaced,categories=cluster-api
// +kubebuilder:storageversion

// IBMVPCMachineTemplate is the Schema for the ibmvpcmachinetemplates API.
type IBMVPCMachineTemplate struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   IBMVPCMachineTemplateSpec   `json:"spec,omitempty"`
	Status IBMVPCMachineTemplateStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// IBMVPCMachineTemplateList contains a list of IBMVPCMachineTemplate.
type IBMVPCMachineTemplateList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []IBMVPCMachineTemplate `json:"items"`
}

func init() {
	objectTypes = append(objectTypes, &IBMVPCMachineTemplate{}, &IBMVPCMachineTemplateList{})
}
