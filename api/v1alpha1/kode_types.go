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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// KodeSpec defines the desired state of Kode
type KodeSpec struct {
	// TemplateRef is the reference to the KodeTemplate configuration
	// +kubebuilder:validation:Description="Reference to the KodeTemplate configuration."
	// +kubebuilder:validation:Required
	TemplateRef KodeTemplateReference `json:"templateRef"`

	// User is the HTTP Basic auth username or the user the container should run as, abc is default.
	// +kubebuilder:validation:Description="User is the HTTP Basic auth username or the user the container should run as. Defaults to 'abc'."
	// +kubebuilder:default="abc"
	User string `json:"user,omitempty"`

	// Password HTTP Basic auth password. If unset there will be no auth
	// +kubebuilder:validation:Description="HTTP Basic auth password. If unset, there will be no authentication."
	Password string `json:"password,omitempty"`

	// ExistingSecret is a reference to an existing secret containing user and password.
	// +kubebuilder:validation:Description="ExistingSecret is a reference to an existing secret containing user and password. If set, User and Password fields are ignored."
	ExistingSecret string `json:"existingSecret,omitempty"`

	// Storage specifies the storage configuration
	// +kubebuilder:validation:Description="Storage configuration."
	Storage KodeStorageSpec `json:"storage,omitempty"`

	// Home is the path to the directory for the user data
	// +kubebuilder:validation:Description="Home is the path to the directory for the user data. Defaults to '/config'."
	// +kubebuilder:validation:MinLength=3
	// +kubebuilder:default=/config
	Home string `json:"home,omitempty"`

	// Workspace is the user specified workspace directory (e.g. my-workspace)
	// +kubebuilder:validation:Description="User specified workspace directory (e.g. my-workspace)."
	// +kubebuilder:validation:MinLength=3
	// +kubebuilder:validation:Pattern="^[^/].*$"
	Workspace string `json:"workspace,omitempty"`

	// UserConfig specifies a git repository URL to get user configuration from
	// +kubebuilder:validation:Description="Git repository URL to get user configuration from."
	UserConfig string `json:"userConfig,omitempty"`

	// Privileged specifies if the container should run in privileged mode. Will only work if the KodeTemplate allows it. Only set to true if you know what you are doing.
	// +kubebuilder:validation:Description="Specifies if the container should run in privileged mode. Will only work if the KodeTemplate allows it. Only set to true if you know what you are doing."
	// +kubebuilder:default=false
	Privileged *bool `json:"privileged,omitempty"`

	// InitPlugins specifies the OCI containers to be run as InitContainers. These containers can be used to prepare the workspace or run some setup scripts. It is an ordered list.
	// +kubebuilder:validation:Description="OCI containers to be run as InitContainers. These containers can be used to prepare the workspace or run some setup scripts. It is an ordered list."
	InitPlugins []InitPluginSpec `json:"initPlugins,omitempty"`

	// // Ingress contains the Ingress configuration for the Kode resource. It will override the KodeTemplate Ingress configuration.
	// // +kubebuilder:validation:Description="Contains the Ingress configuration for the Kode resource. It will override the KodeTemplate Ingress configuration."
	// Ingress *IngressSpec `json:"ingress,omitempty"`

	// // Gateway contains the Gateway configuration for the Kode resource. It will override the KodeTemplate Gateway configuration.
	// // +kubebuilder:validation:Description="Contains the Gateway configuration for the Kode resource. It will override the KodeTemplate Gateway configuration."
	// Gateway *GatewaySpec `json:"gateway,omitempty"`
}

// KodeStorageSpec defines the storage configuration
type KodeStorageSpec struct {
	// AccessModes specifies the access modes for the persistent volume
	AccessModes []corev1.PersistentVolumeAccessMode `json:"accessModes,omitempty"`

	// StorageClassName specifies the storage class name for the persistent volume
	StorageClassName *string `json:"storageClassName,omitempty"`

	// Resources specifies the resource requirements for the persistent volume
	Resources corev1.VolumeResourceRequirements `json:"resources,omitempty"`

	// KeepVolume specifies if the volume should be kept when the kode is recycled. Defaults to false.
	// +kubebuilder:validation:Description="Specifies if the volume should be kept when the kode is recycled. Defaults to false."
	// +kubebuilder:default=false
	KeepVolume *bool `json:"keepVolume,omitempty"`
}

type ConditionType string

const (
	// Created means the code server has been accepted by the system.
	Created ConditionType = "Created"
	// Ready means the code server has been ready for usage.
	Ready ConditionType = "Ready"
	// Recycled means the code server has been recycled totally.
	Recycled ConditionType = "Recycled"
	// Inactive means the code server will be marked inactive if `InactiveAfterSeconds` elapsed
	Inactive ConditionType = "Inactive"
)

type Condition struct {
	// Type of code server condition.
	Type ConditionType `json:"type"`
	// Status of the condition, one of True, False, Unknown.
	Status corev1.ConditionStatus `json:"status"`
	// The reason for the condition's last transition.
	Reason string `json:"reason,omitempty"`
	// A human readable message indicating details about the transition.
	Message string `json:"message,omitempty"`
	// The last time this condition was updated.
	LastUpdateTime metav1.Time `json:"lastUpdateTime,omitempty"`
	// Last time the condition transitioned from one status to another.
	LastTransitionTime metav1.Time `json:"lastTransitionTime,omitempty"`
}

// KodeStatus defines the observed state of Kode
type KodeStatus struct {
	Conditions []Condition `json:"conditions,omitempty" protobuf:"bytes,1,opt,name=conditions"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// Kode is the Schema for the kodes API
type Kode struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   KodeSpec   `json:"spec,omitempty"`
	Status KodeStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// KodeList contains a list of Kode
type KodeList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Kode `json:"items"`
}

type KodeTemplateReference struct {
	// Kind is the resource kind
	// +kubebuilder:validation:Description="Resource kind"
	// +kubebuilder:validation:Enum=KodeTemplate;KodeClusterTemplate
	Kind string `json:"kind"`

	// Name is the name of the KodeTemplate
	// +kubebuilder:validation:Description="Name of the KodeTemplate"
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// Namespace is the namespace of the KodeTemplate
	// +kubebuilder:validation:Description="Namespace of the KodeTemplate"
	Namespace string `json:"namespace,omitempty"`
}

type InitPluginSpec struct {
	// Name is the name of the container
	// +kubebuilder:validation:Description="Name of the container."
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// Image is the OCI image for the container
	// +kubebuilder:validation:Description="OCI image for the container."
	// +kubebuilder:validation:Required
	Image string `json:"image"`

	// Args are the arguments to the container
	// +kubebuilder:validation:Description="Arguments to the container."
	Args []string `json:"args,omitempty"`

	// EnvVars are the environment variables to the container
	// +kubebuilder:validation:Description="Environment variables to the container."
	EnvVars []corev1.EnvVar `json:"envVars,omitempty"`

	// Command is the command to run in the container
	// +kubebuilder:validation:Description="Command to run in the container."
	Command []string `json:"command,omitempty"`
}

func init() {
	SchemeBuilder.Register(&Kode{}, &KodeList{})
}
