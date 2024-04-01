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
	"fmt"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// ObserverSpec defines the desired state of Observer
type ObserverSpecEntry struct {
	Endpoint  string              `json:"endpoint"`
	SecretRef *v1.SecretReference `json:"secretRef,omitempty"`
	Name      string              `json:"name"`
}

func (o *ObserverSpecEntry) NewPod(namespace string) (*v1.Pod, error) {
	labels := map[string]string{
		"app.kubernetes.io/managed-by": "ObserverController",
	}
	container := v1.Container{
		Name:    "check",
		Image:   "curlimages/curl:7.78.0",
		Command: []string{"/bin/sh", "-c", fmt.Sprintf("curl -vvv -L %s", o.Endpoint)},
	}
	pod := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      o.Name,
			Namespace: namespace,
			Labels:    labels,
		},
		Spec: v1.PodSpec{
			Containers: []v1.Container{
				container,
			},
			RestartPolicy: v1.RestartPolicyNever,
		},
	}

	if o.SecretRef != nil {
		container.EnvFrom = []v1.EnvFromSource{
			{
				SecretRef: &v1.SecretEnvSource{
					LocalObjectReference: v1.LocalObjectReference{
						Name: o.SecretRef.Name,
					},
				},
			},
		}
	}
	return pod, nil
}

type ObserverSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Foo is an example field of Observer. Edit observer_types.go to remove/update

	Entries  []ObserverSpecEntry `json:"entries"`
	Interval int                 `json:"interval"`
}

// ObserverStatus defines the observed state of Observer

type Status struct {
	Status int    `json:"status,omitempty"`
	Time   string `json:"time"`
}

type ObserverStatus struct {

	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	Conditions  []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,1,rep,name=conditions"`
	CurrentItem int                `json:"currentItem"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// Observer is the Schema for the observers API
type Observer struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ObserverSpec   `json:"spec,omitempty"`
	Status ObserverStatus `json:"status,omitempty"`
}

func (o *Observer) SetStatusCondition(conditionType ConditionType, status metav1.ConditionStatus, reason string, message string) bool {
	return meta.SetStatusCondition(&o.Status.Conditions, metav1.Condition{
		Type:    string(conditionType),
		Status:  status,
		Reason:  reason,
		Message: message,
	})
}

func (o *Observer) IsStatusConditionFalse(conditionType ConditionType) bool {
	return meta.IsStatusConditionFalse(o.Status.Conditions, string(conditionType))
}
func (o *Observer) IsStatusConditionTrue(conditionType ConditionType) bool {
	return meta.IsStatusConditionTrue(o.Status.Conditions, string(conditionType))
}

//+kubebuilder:object:root=true

// ObserverList contains a list of Observer
type ObserverList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Observer `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Observer{}, &ObserverList{})
}
