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

// Spec for the EnvoyProxyConfig.
type SharedEnvoyProxyConfigSpec struct {
	// Image is the Docker image for the Envoy proxy
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:Required
	// +kubebuilder:default="envoyproxy/envoy:v1.30-latest"
	Image string `json:"image"`

	// AuthType is the type of authentication to use
	// +kubebuilder:validation:Description="Type of authentication to use"
	// +kubebuilder:validation:Enum=basic
	AuthType string `json:"authType,omitempty"`

	// HTTPFilters is a list of Envoy HTTP filters to be applied
	// +kubebuilder:validation:Description="HTTP filters to be applied"
	HTTPFilters []HTTPFilter `json:"httpFilters,omitempty"`

	// Clusters is a list of Envoy clusters
	// +kubebuilder:validation:Description="Envoy clusters"
	Clusters []Cluster `json:"clusters,omitempty"`
}

// EnvoyProxyStatus defines the observed state of EnvoyProxy
type SharedEnvoyProxyStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// EnvoyProxyReference is a reference to an EnvoyProxyConfig or EnvoyProxyClusterConfig
type EnvoyProxyReference struct {
	// Kind is the resource kind
	// +kubebuilder:validation:Description="Resource kind"
	// +kubebuilder:validation:Enum=EnvoyProxyConfig;EnvoyProxyClusterConfig
	// +kubebuilder:validation:Required
	Kind string `json:"kind"`

	// Name is the name of the EnvoyProxyConfig or EnvoyProxyClusterConfig
	// +kubebuilder:validation:Description="Name of the config"
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// Namespace is the namespace of the EnvoyProxyConfig or EnvoyProxyClusterConfig
	// +kubebuilder:validation:Description="Namespace of the Envoy Proxy config"
	Namespace string `json:"namespace,omitempty"`
}
