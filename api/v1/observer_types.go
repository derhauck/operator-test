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

type ConditionType string

const (
	// TypeAvailableObserver represents the status of the Deployment reconciliation
	TypeAvailableObserver ConditionType = "Available"
	// TypeDegradedObserver represents the status used when the custom resource is deleted and the finalizer operations are yet to occur.
	TypeDegradedObserver ConditionType = "Degraded"
	// TypeSuccessLastStatusObserver represents the status used when the custom resource is deleted and the finalizer operations are yet to occur.
	TypeSuccessLastStatusObserver ConditionType = "SuccessLastStatus"
	// TypeSuccessStatusForLastFiveObserver represents the status used when the custom resource is deleted and the finalizer operations are yet to occur.
	TypeSuccessStatusForLastFiveObserver ConditionType = "SuccessStatusForLastFive"
)

// ObserverSpec defines the desired state of Observer
type ObserverSpecEntry struct {
	// Endpoint will directly use 'curl' on the specified endpoint
	Endpoint string `json:"endpoint,omitempty"`
	// SecretRef allows to expose all keys inside a referenced secret to the pod environment
	SecretRef *v1.SecretReference `json:"secretRef,omitempty"`
	// Name of the entry to check
	Name string `json:"name"`
	// OpenApi will use OpenAPI config to validate with schemathesis
	SchemaEndpoint string `json:"schemaEndpoint,omitempty"`
}

func (o *Observer) NewPod() (*v1.Pod, error) {
	current := o.Spec.Entries[o.Status.CurrentItem]
	image := "curlimages/curl:7.78.0"
	command := fmt.Sprintf("curl -vvv -L %s", current.Endpoint)

	if current.SchemaEndpoint != "" {
		image = "schemathesis/schemathesis:stable"
		command = fmt.Sprintf("schemathesis run --checks all  %s", current.SchemaEndpoint)
		if current.Endpoint != "" {
			command = fmt.Sprintf("%s --base-url %s", command, current.Endpoint)
		}
	}

	labels := map[string]string{
		"app.kubernetes.io/managed-by": "ObserverController",
		"app":                          o.Name,
		"observer.crd.test.kateops.com/entry-item": current.Name,
	}

	container := v1.Container{
		Name:    "check",
		Image:   image,
		Command: []string{"/bin/sh", "-c", command},
	}

	if current.SecretRef != nil {
		container.EnvFrom = []v1.EnvFromSource{
			{
				SecretRef: &v1.SecretEnvSource{
					LocalObjectReference: v1.LocalObjectReference{
						Name: current.SecretRef.Name,
					},
				},
			},
		}
	}
	pod := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      o.GetCurrentPodName(),
			Namespace: o.Namespace,
			Labels:    labels,
		},
		Spec: v1.PodSpec{
			Containers: []v1.Container{
				container,
			},
			RestartPolicy: v1.RestartPolicyNever,
		},
	}

	return pod, nil
}

type ObserverSpec struct {
	// Entries list of configurations for the observing process
	Entries []ObserverSpecEntry `json:"entries"`
	// RetryAfterSeconds how long the process will wait until it starts the process again in seconds
	RetryAfterSeconds int `json:"retryAfterSeconds"`
}

func AddWithMaxLen[T any](slice []T, item T, max int) []T {
	if len(slice) >= max && max != -1 {
		slice = slice[1:]
	}

	result := append(slice, item)

	return result
}

type PodStatus struct {
	Status v1.PodPhase `json:"status"`
	Time   string      `json:"time"`
	Name   string      `json:"name"`
}

type IterationResult struct {
	Status        v1.PodPhase  `json:"status"`
	PodStatusList *[]PodStatus `json:"podStatusList,omitempty"`
}

func (i *IterationResult) Evaluate() {
	i.Status = v1.PodSucceeded
	var result v1.PodPhase
	for _, podStatus := range *i.PodStatusList {
		if podStatus.Status != v1.PodSucceeded {
			result = podStatus.Status
		}
	}
	if result != v1.PodSucceeded && result != "" {
		i.Status = result
	}
}

func (i *IterationResult) Add(item *PodStatus) {
	if i.PodStatusList == nil {
		temp := make([]PodStatus, 0)
		i.PodStatusList = &temp
	}
	temp := append(*i.PodStatusList, *item)
	i.PodStatusList = &temp
}

// ObserverStatus defines the observed state of Observer
type ObserverStatus struct {

	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	Conditions  []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,1,rep,name=conditions"`
	CurrentItem int                `json:"currentItem"`
	// results of status checks
	IterationResults *[]IterationResult `json:"iterationResults,omitempty" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,1,rep,name=iterationResults"`
}

func (o *ObserverStatus) InitIterationResults() {
	o.IterationResults = &[]IterationResult{}
}

func (o *ObserverStatus) AddNewIterationResult() *ObserverStatus {
	temp := AddWithMaxLen(*o.IterationResults, IterationResult{
		PodStatusList: &[]PodStatus{},
	}, 4)
	o.IterationResults = &temp
	return o
}
func (o *ObserverStatus) AddPodStatusToLatestIterationResult(item *PodStatus) *ObserverStatus {
	o.GetLatestIterationResult().Add(item)
	return o
}

func (o *ObserverStatus) GetLatestIterationResult() *IterationResult {
	var temp = IterationResult{PodStatusList: &[]PodStatus{}, Status: v1.PodPending}
	if o.IterationResults != nil && len(*o.IterationResults) > 0 {
		results := *o.IterationResults
		current := &results[len(*o.IterationResults)-1]
		if current != nil {
			temp = *current
		}
	}
	return &temp
}

func (o *ObserverStatus) SetStatusCondition(conditionType ConditionType, status metav1.ConditionStatus, reason string, message string) bool {
	return meta.SetStatusCondition(&o.Conditions, metav1.Condition{
		Type:    string(conditionType),
		Status:  status,
		Reason:  reason,
		Message: message,
	})
}

func (o *ObserverStatus) EvaluateStatus() {
	var failure = false
	var status v1.PodPhase = v1.PodSucceeded
	for _, iteration := range *o.IterationResults {
		if iteration.Status != v1.PodSucceeded && iteration.Status != "" {
			failure = true
			status = iteration.Status

		}
	}

	if failure {
		o.SetStatusCondition(
			TypeSuccessStatusForLastFiveObserver,
			metav1.ConditionFalse,
			fmt.Sprintf("pod:%s", string(status)),
			fmt.Sprintf("Evaluate observer to contain recent failure"),
		)
	} else {
		o.SetStatusCondition(
			TypeSuccessStatusForLastFiveObserver,
			metav1.ConditionTrue,
			fmt.Sprintf("pod:%s", string(status)),
			fmt.Sprintf("Evaluate observer to contain no recent failure"),
		)
	}
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

func (o *Observer) GetCurrentPodName() string {
	return fmt.Sprintf("%s-%d", o.Name, o.Status.CurrentItem)
}

func (o *Observer) GetCurrentEntry() ObserverSpecEntry {
	return o.Spec.Entries[o.Status.CurrentItem]
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
