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
	"errors"
	"fmt"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// log is for logging in this package.
var operatortestlog = logf.Log.WithName("operatortest-resource")

// SetupWebhookWithManager will setup the manager to manage the webhooks
func (r *OperatorTest) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

// TODO(user): EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

//+kubebuilder:webhook:path=/mutate-crd-test-kateops-com-v1-operatortest,mutating=true,failurePolicy=fail,sideEffects=None,groups=crd.test.kateops.com,resources=operatortests,verbs=create;update,versions=v1,name=moperatortest.kb.io,admissionReviewVersions=v1

var _ webhook.Defaulter = &OperatorTest{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *OperatorTest) Default() {
	operatortestlog.Info("default", "name", r.Name)

	// TODO(user): fill in your defaulting logic.
	if r.Spec.Shell == "" {
		r.Spec.Shell = "/bin/ash"
	}
}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
//+kubebuilder:webhook:path=/validate-crd-test-kateops-com-v1-operatortest,mutating=false,failurePolicy=fail,sideEffects=None,groups=crd.test.kateops.com,resources=operatortests,verbs=create;update,versions=v1,name=voperatortest.kb.io,admissionReviewVersions=v1

var _ webhook.Validator = &OperatorTest{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *OperatorTest) ValidateCreate() (admission.Warnings, error) {
	operatortestlog.Info("validate create", "name", r.Name)

	if r.Spec.Name == "" {
		operatortestlog.Info("validate create - no Name set", "name", r.Name)
		return admission.Warnings{"no Name set"}, errors.New(fmt.Sprintf("no Name set crd: %s", r.Name))
	}
	if r.Spec.Shell == "" {
		operatortestlog.Info("validate create - no Shell set", "name", r.Name)
		return admission.Warnings{"no Shell set"}, errors.New(fmt.Sprintf("no Shell set crd: %s", r.Name))
	}
	// TODO(user): fill in your validation logic upon object creation.
	return nil, nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *OperatorTest) ValidateUpdate(old runtime.Object) (admission.Warnings, error) {
	operatortestlog.Info("validate update", "name", r.Name)

	if r.Spec.Name == "" {
		operatortestlog.Info("validate update - no Name set", "name", r.Name)
		return admission.Warnings{"no Name set"}, errors.New(fmt.Sprintf("no Name set crd: %s", r.Name))
	}
	if r.Spec.Shell == "" {
		operatortestlog.Info("validate update - no Shell set", "name", r.Name)
		return admission.Warnings{"no Shell set"}, errors.New(fmt.Sprintf("no Shell set crd: %s", r.Name))
	}

	// TODO(user): fill in your validation logic upon object update.
	return nil, nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *OperatorTest) ValidateDelete() (admission.Warnings, error) {
	operatortestlog.Info("validate delete", "name", r.Name)

	// TODO(user): fill in your validation logic upon object deletion.
	return nil, nil
}
