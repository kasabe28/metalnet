/*
Copyright 2022 The Metal Authors.

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

// LoadBalancerSpec defines the desired state of LoadBalancer
type LoadBalancerSpec struct {
	// NetworkRef is the Network this LoadBalancer is connected to
	NetworkRef corev1.LocalObjectReference `json:"networkRef"`
	// IPFamily defines which IPFamily this LoadBalancer is supporting
	IPFamily corev1.IPFamily `json:"ipFamily"`
	// IP is the provided IP which should be loadbalanced by this LoadBalancer
	IP IP `json:"ip,omitempty"`
	// Ports are the provided ports
	Ports []LBPort `json:"ports,omitempty"`
	// NodeName is the name of the node on which the LoadBalancer should be created.
	NodeName *string `json:"nodeName,omitempty"`
}

// LoadBalancerStatus defines the observed state of LoadBalancer
type LoadBalancerStatus struct {
	// State is the LoadBalancerState of the LoadBalancer.
	State LoadBalancerState `json:"state,omitempty"`
}

// LoadBalancerState is the binding state of a LoadBalancer.
type LoadBalancerState string

const (
	// LoadBalancerStateReady is used for any LoadBalancer that is ready.
	LoadBalancerStateReady LoadBalancerState = "Ready"
	// LoadBalancerStatePending is used for any LoadBalancer that is in an intermediate state.
	LoadBalancerStatePending LoadBalancerState = "Pending"
	// LoadBalancerStateError is used for any LoadBalancer that is some error occurred.
	LoadBalancerStateError LoadBalancerState = "Error"
)

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Status",type=string,description="Status of the loadbalancer.",JSONPath=`.status.state`,priority=0
// +kubebuilder:printcolumn:name="NodeName",type=string,description="Node the loadbalancer is running on.",JSONPath=`.spec.nodeName`,priority=0
// LoadBalancer is the Schema for the loadbalancers API
type LoadBalancer struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   LoadBalancerSpec   `json:"spec,omitempty"`
	Status LoadBalancerStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// LoadBalancerList contains a list of LoadBalancer
type LoadBalancerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []LoadBalancer `json:"items"`
}

func init() {
	SchemeBuilder.Register(&LoadBalancer{}, &LoadBalancerList{})
}