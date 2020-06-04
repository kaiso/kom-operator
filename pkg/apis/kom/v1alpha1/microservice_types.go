package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// Http http-based routes
type Http struct {
	Port corev1.ContainerPort `json:"port,omitempty"`
	Rule string               `json:"rule,omitempty"`
}

// Routing define router rules
type Routing struct {
	Http []Http `json:"http,omitempty"`
}

// MicroserviceSpec defines the desired state of Microservice
type MicroserviceSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book-v1.book.kubebuilder.io/beyond_basics/generating_crd.html

	Image       string                   `json:"image"`
	Command     []string                 `json:"command,omitempty"`
	Autoscaling MicroservicesAutoscaling `json:"autoscaling"`
	Routing     Routing                  `json:"routing,omitempty"`
}

// MicroservicesAutoscaling defines the autoscaling strategy configuration
type MicroservicesAutoscaling struct {
	Max int32 `json:"max"`
	Min int32 `json:"min"`
}

// MicroserviceStatus defines the observed state of Microservice
type MicroserviceStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book-v1.book.kubebuilder.io/beyond_basics/generating_crd.html
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Microservice is the Schema for the microservices API
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=microservices,scope=Namespaced
type Microservice struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   MicroserviceSpec   `json:"spec,omitempty"`
	Status MicroserviceStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// MicroserviceList contains a list of Microservice
type MicroserviceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Microservice `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Microservice{}, &MicroserviceList{})
}
