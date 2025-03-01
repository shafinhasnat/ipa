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

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// IPASpec defines the desired state of IPA.
type IPASpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Foo is an example field of IPA. Edit ipa_types.go to remove/update
	Metadata Metadata `json:"metadata"`
}

type Metadata struct {
	PrometheusUri string     `json:"prometheusUri"`
	LLMAgent      string     `json:"llmAgent"`
	IPAGroup      []IPAGroup `json:"ipaGroup"`
}

type IPAGroup struct {
	Deployment string `json:"deployment"`
	Namespace  string `json:"namespace"`
	Ingress    string `json:"ingress,omitempty"`
}

// IPAStatus defines the observed state of IPA.
type IPAStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	Status string `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// IPA is the Schema for the ipas API.
type IPA struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   IPASpec   `json:"spec,omitempty"`
	Status IPAStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// IPAList contains a list of IPA.
type IPAList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []IPA `json:"items"`
}

func init() {
	SchemeBuilder.Register(&IPA{}, &IPAList{})
}
