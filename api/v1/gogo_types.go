/*
Copyright 2023 Jon Keam.

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

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// GogoSpec defines the desired state of Gogo
type GogoSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	Size int32  `json:"size"`
	Host string `json:"host"`
}

// GogoStatus defines the observed state of Gogo
type GogoStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	Pods []string `json:"pods"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// Gogo is the Schema for the gogoes API
type Gogo struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   GogoSpec   `json:"spec,omitempty"`
	Status GogoStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// GogoList contains a list of Gogo
type GogoList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Gogo `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Gogo{}, &GogoList{})
}
