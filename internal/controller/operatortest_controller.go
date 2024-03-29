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

package controller

import (
	"context"
	crdv1 "derhauck/operator-test/api/v1"
	"fmt"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// OperatorTestReconciler reconciles a OperatorTest object
type OperatorTestReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=crd.test.kateops.com,resources=operatortests,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=crd.test.kateops.com,resources=operatortests/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=crd.test.kateops.com,resources=operatortests/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the OperatorTest object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.17.0/pkg/reconcile
func (r *OperatorTestReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Info("ctr Reconcile - not doing anything")

	return ctrl.Result{}, nil
}

func (r *OperatorTestReconciler) HandleEvents(ctx context.Context, crd client.Object) []reconcile.Request {
	logger := log.FromContext(ctx)
	if crd.GetNamespace() != "default" {
		logger.Info(fmt.Sprintf("ctr event - wrong namespace - %s", crd.GetNamespace()))
		return []reconcile.Request{}
	}
	//logger.Info(pod.GetObjectKind().GroupVersionKind().Kind)
	namespacedName := types.NamespacedName{
		Namespace: crd.GetNamespace(),
		Name:      crd.GetName(),
	}
	var podObject crdv1.OperatorTest
	err := r.Get(context.Background(), namespacedName, &podObject)
	if err != nil {
		log.Log.Error(err, err.Error())
		return []reconcile.Request{}
	}
	if podObject.GetAnnotations()["new"] == "" {
		podObject.Annotations["new"] = podObject.CreationTimestamp.String()
	}
	annotations := podObject.GetAnnotations()
	annotations["new"] = podObject.CreationTimestamp.String()
	podObject.SetAnnotations(annotations)
	if podObject.Spec.Shell == "bash" {
		podObject.Spec.Shell = "/bin/bash"
	}

	if err := r.Update(context.TODO(), &podObject); err != nil {
		logger.Error(err, err.Error())
	}
	logger.Info(fmt.Sprintf("handling events - %s - shell: %s - %s", podObject.GetGenerateName(), podObject.Spec.Shell, podObject.CreationTimestamp.String()))
	return []reconcile.Request{
		{NamespacedName: namespacedName},
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *OperatorTestReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&crdv1.OperatorTest{}).
		Watches(
			&crdv1.OperatorTest{},
			handler.EnqueueRequestsFromMapFunc(r.HandleEvents),
			builder.WithPredicates(predicate.AnnotationChangedPredicate{}),
		).
		Complete(r)
}
