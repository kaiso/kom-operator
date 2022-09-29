/*
Copyright 2021 Kais OMRI.

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

//Status reports status of the reconcile
type Status string

// Resource defines the resouce targeted by a Scaler
type Resource string

// ScalerType represents whether the metric type is Utilization, Value, or AverageValue
type ScalerType string

const (
	//Success reconcile success
	Success Status = "Success"
	//Failure reconcile failure
	Failure Status = "Failure"
	// CPU the CPU resource type
	CPU Resource = "CPU"
	// Memory the Memory resource type
	Memory Resource = "Memory"
	// Utilization scaler type
	Utilization ScalerType = "Utilization"
	// Value scaler type
	Value ScalerType = "Value"
	// AverageValue scaler type
	AverageValue ScalerType = "AverageValue"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// HTTP http-based routes
type HTTP struct {
	Port corev1.ContainerPort `json:"port,omitempty"`
	Rule string               `json:"rule,omitempty"`
}

// Routing define router rules
type Routing struct {
	HTTP []HTTP `json:"http,omitempty"`
}

// Scaler define a scaler based on a metric
type Scaler struct {
	Resource Resource   `json:"resource"`
	Type     ScalerType `json:"type"`
	Value    string     `json:"value"`
}

// MicroservicesAutoscaling defines the autoscaling strategy configuration
type MicroservicesAutoscaling struct {
	Scaler []Scaler `json:"scaler,omitempty"`
	Max    int32    `json:"max"`
	Min    int32    `json:"min"`
}

// Container the container that will run the microservice
type Container struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book-v1.book.kubebuilder.io/beyond_basics/generating_crd.html

	Image           string                      `json:"image"`
	Command         []string                    `json:"command,omitempty"`
	Args            []string                    `json:"args,omitempty"`
	Routing         Routing                     `json:"routing,omitempty"`
	ImagePullPolicy corev1.PullPolicy           `json:"imagePullPolicy,omitempty"`
	VolumeMounts    []corev1.VolumeMount        `json:"volumeMounts,omitempty"`
	Resources       corev1.ResourceRequirements `json:"resources,omitempty"`
}

// MicroserviceSpec defines the desired state of Microservice
type MicroserviceSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	Container          Container                  `json:"container"`
	Autoscaling        MicroservicesAutoscaling   `json:"autoscaling,omitempty"`
	Volumes            []corev1.Volume            `json:"volumes,omitempty"`
	ServiceAccountName string                     `json:"serviceAccountName,omitempty"`
	NodeSelector       map[string]string          `json:"nodeSelector,omitempty" protobuf:"bytes,7,rep,name=nodeSelector"`
	SecurityContext    *corev1.PodSecurityContext `json:"securityContext,omitempty" protobuf:"bytes,15,opt,name=securityContext"`
}

// MicroserviceStatus defines the observed state of Microservice
type MicroserviceStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	Routing    Routing     `json:"routing,omitempty"`
	Status     Status      `json:"status,omitempty"`
	LastUpdate metav1.Time `json:"lastUpdate,omitempty"`
	Reason     string      `json:"reason,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// Microservice is the Schema for the microservices API
type Microservice struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   MicroserviceSpec   `json:"spec,omitempty"`
	Status MicroserviceStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// MicroserviceList contains a list of Microservice
type MicroserviceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Microservice `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Microservice{}, &MicroserviceList{})
}
