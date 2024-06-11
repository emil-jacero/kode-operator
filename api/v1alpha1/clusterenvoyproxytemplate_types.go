/*
Copyright 2024.

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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ClusterEnvoyProxyTemplateSpec defines the desired state of ClusterEnvoyProxyTemplate
type ClusterEnvoyProxyTemplateSpec struct {
	SharedEnvoyProxyTemplateSpec `json:",inline"`
}

// ClusterEnvoyProxyTemplateStatus defines the observed state of ClusterEnvoyProxyTemplate
type ClusterEnvoyProxyTemplateStatus struct {
	SharedEnvoyProxyTemplateStatus `json:",inline"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster

// ClusterEnvoyProxyTemplate is the Schema for the clusterenvoyproxytemplates API
type ClusterEnvoyProxyTemplate struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ClusterEnvoyProxyTemplateSpec   `json:"spec,omitempty"`
	Status ClusterEnvoyProxyTemplateStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// ClusterEnvoyProxyTemplateList contains a list of ClusterEnvoyProxyTemplate
type ClusterEnvoyProxyTemplateList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ClusterEnvoyProxyTemplate `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ClusterEnvoyProxyTemplate{}, &ClusterEnvoyProxyTemplateList{})
}