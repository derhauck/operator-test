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

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// OperatorTestSpec defines the desired state of OperatorTest
type OperatorTestSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Foo is an example field of OperatorTest. Edit operatortest_types.go to remove/update
	Name  string `json:"name,omitempty"`
	Shell string `json:"shell,omitempty"`
}

// OperatorTestStatus defines the observed state of OperatorTest
type OperatorTestStatus struct {

	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	//Pod string `json:"pod"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// OperatorTest is the Schema for the operatortests API
type OperatorTest struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   OperatorTestSpec   `json:"spec,omitempty"`
	Status OperatorTestStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// OperatorTestList contains a list of OperatorTest
type OperatorTestList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []OperatorTest `json:"items"`
}

func init() {
	SchemeBuilder.Register(&OperatorTest{}, &OperatorTestList{})
}
