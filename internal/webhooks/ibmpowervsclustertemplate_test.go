/*
Copyright 2023 The Kubernetes Authors.

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

	infrav1 "sigs.k8s.io/cluster-api-provider-ibmcloud/api/v1beta2"

	. "github.com/onsi/gomega"
)

func TestIBMPowerVSClusterTemplate_ValidateUpdate(t *testing.T) {
	g := NewWithT(t)

	tests := []struct {
		name        string
		newTemplate *infrav1.IBMPowerVSClusterTemplate
		oldTemplate *infrav1.IBMPowerVSClusterTemplate
		wantErr     bool
	}{
		{
			name: "IBMPowerVSClusterTemplate with immutable spec",
			newTemplate: &infrav1.IBMPowerVSClusterTemplate{
				Spec: infrav1.IBMPowerVSClusterTemplateSpec{
					Template: infrav1.IBMPowerVSClusterTemplateResource{
						Spec: infrav1.IBMPowerVSClusterSpec{
							ServiceInstanceID: "test-instance1",
						},
					},
				},
			},
			oldTemplate: &infrav1.IBMPowerVSClusterTemplate{
				Spec: infrav1.IBMPowerVSClusterTemplateSpec{
					Template: infrav1.IBMPowerVSClusterTemplateResource{
						Spec: infrav1.IBMPowerVSClusterSpec{
							ServiceInstanceID: "test-instance1",
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: " IBMPowerVSClusterTemplate with mutable spec",
			newTemplate: &infrav1.IBMPowerVSClusterTemplate{
				Spec: infrav1.IBMPowerVSClusterTemplateSpec{
					Template: infrav1.IBMPowerVSClusterTemplateResource{
						Spec: infrav1.IBMPowerVSClusterSpec{
							ServiceInstanceID: "test-instance1",
						},
					},
				},
			},
			oldTemplate: &infrav1.IBMPowerVSClusterTemplate{
				Spec: infrav1.IBMPowerVSClusterTemplateSpec{
					Template: infrav1.IBMPowerVSClusterTemplateResource{
						Spec: infrav1.IBMPowerVSClusterSpec{
							ServiceInstanceID: "test-instance2",
						},
					},
				},
			},
			wantErr: true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(_ *testing.T) {
			ibmPowerVSClusterTemplate := IBMPowerVSClusterTemplate{}
			_, err := ibmPowerVSClusterTemplate.ValidateUpdate(ctx, test.oldTemplate, test.newTemplate)
			if test.wantErr {
				g.Expect(err).To(HaveOccurred())
			} else {
				g.Expect(err).NotTo(HaveOccurred())
			}
		})
	}
}
