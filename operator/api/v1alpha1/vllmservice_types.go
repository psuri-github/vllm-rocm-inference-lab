/*
Copyright 2026.

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
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// VLLMServiceSpec defines the desired state of VLLMService
type VLLMServiceSpec struct {
	// model is the Hugging Face model identifier served by vLLM.
	// +kubebuilder:validation:MinLength=1
	// +required
	Model string `json:"model"`

	// image is the vLLM ROCm container image.
	// +kubebuilder:default="vllm/vllm-openai-rocm:latest"
	// +optional
	Image string `json:"image,omitempty"`

	// replicas is the desired number of vLLM Pods.
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:default=1
	// +optional
	Replicas int32 `json:"replicas,omitempty"`

	// port is the HTTP port exposed by vLLM and its Service.
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=65535
	// +kubebuilder:default=8000
	// +optional
	Port int32 `json:"port,omitempty"`

	// gpuCount is the number of AMD GPUs requested by each vLLM Pod.
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:default=1
	// +optional
	GPUCount int32 `json:"gpuCount,omitempty"`

	// modelCacheSize is the requested capacity of the model-cache PVC.
	// +kubebuilder:default="20Gi"
	// +optional
	ModelCacheSize resource.Quantity `json:"modelCacheSize,omitempty"`

	// storageClassName selects the StorageClass for the model-cache PVC.
	// When omitted, Kubernetes uses the cluster's default StorageClass.
	// +optional
	StorageClassName *string `json:"storageClassName,omitempty"`
}

// VLLMServiceStatus defines the observed state of VLLMService.
type VLLMServiceStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// For Kubernetes API conventions, see:
	// https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#typical-status-properties

	// conditions represent the current state of the VLLMService resource.
	// Each condition has a unique type and reflects the status of a specific aspect of the resource.
	//
	// Standard condition types include:
	// - "Available": the resource is fully functional
	// - "Progressing": the resource is being created or updated
	// - "Degraded": the resource failed to reach or maintain its desired state
	//
	// The status of each condition is one of True, False, or Unknown.
	// +listType=map
	// +listMapKey=type
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// VLLMService is the Schema for the vllmservices API
type VLLMService struct {
	metav1.TypeMeta `json:",inline"`

	// metadata is a standard object metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitzero"`

	// spec defines the desired state of VLLMService
	// +required
	Spec VLLMServiceSpec `json:"spec"`

	// status defines the observed state of VLLMService
	// +optional
	Status VLLMServiceStatus `json:"status,omitzero"`
}

// +kubebuilder:object:root=true

// VLLMServiceList contains a list of VLLMService
type VLLMServiceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitzero"`
	Items           []VLLMService `json:"items"`
}

func init() {
	SchemeBuilder.Register(func(s *runtime.Scheme) error {
		s.AddKnownTypes(SchemeGroupVersion, &VLLMService{}, &VLLMServiceList{})
		return nil
	})
}
