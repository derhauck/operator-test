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
	v12 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"time"
)

// ObserverReconciler reconciles a Observer object
type ObserverReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

const SEC = 1000000000

//+kubebuilder:rbac:groups=crd.test.kateops.com,resources=observers,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=crd.test.kateops.com,resources=observers/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=crd.test.kateops.com,resources=observers/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Observer object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.17.0/pkg/reconcile
func (r *ObserverReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Info("reconcile observer")

	var pod = &v12.Pod{}
	var crd = crdv1.Observer{}
	err := r.Get(ctx, req.NamespacedName, &crd)
	if err != nil {
		logger.Error(err, err.Error())
		return ctrl.Result{}, err
	}
	newPod := NewPodForCR(&crd)

	err = r.Get(ctx, req.NamespacedName, pod)
	if err != nil {
		logger.Info(fmt.Sprintf("pod not found '%s' - recreating", req.NamespacedName))

		err = ctrl.SetControllerReference(&crd, newPod, r.Scheme)
		if err != nil {
			return ctrl.Result{}, err
		}
		err = r.Create(ctx, newPod)
		if err != nil {
			logger.Error(err, err.Error())
			return ctrl.Result{}, nil
		}
		return ctrl.Result{Requeue: true}, nil
	} else {
		if pod.Status.Phase == "Running" || pod.Status.Phase == "Pending" {
			logger.Info(fmt.Sprintf("pod found '%s' - still running", req.NamespacedName))
			return ctrl.Result{
				Requeue:      true,
				RequeueAfter: time.Duration(2 * SEC),
			}, nil
		}

		logger.Info(fmt.Sprintf("pod found '%s' - deleting", req.NamespacedName))
		err = r.Delete(ctx, newPod)
		if err != nil {
			logger.Error(err, err.Error())

		}
		logger.Info(fmt.Sprintf("interval '%d'", crd.Spec.Interval*SEC))
		return ctrl.Result{
			RequeueAfter: time.Duration(crd.Spec.Interval * SEC),
		}, nil
	}
}

func (r *ObserverReconciler) HandlePodEvents(ctx context.Context, pod client.Object) []reconcile.Request {
	namespacedName := types.NamespacedName{
		Namespace: pod.GetNamespace(),
		Name:      pod.GetName(),
	}
	logger := log.FromContext(ctx)
	logger.Info(fmt.Sprintf("handle update event - %s - ", namespacedName))
	var getPod v12.Pod
	err := r.Get(ctx, namespacedName, &getPod)
	if err != nil {
		return []reconcile.Request{
			{NamespacedName: namespacedName},
		}
	}

	return []reconcile.Request{}
}

// SetupWithManager sets up the controller with the Manager.
func (r *ObserverReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&crdv1.Observer{}).
		Watches(
			&v12.Pod{},
			handler.EnqueueRequestsFromMapFunc(r.HandlePodEvents),
			builder.WithPredicates(predicate.Funcs{
				DeleteFunc:  func(e event.DeleteEvent) bool { return false },
				CreateFunc:  func(e event.CreateEvent) bool { return false },
				GenericFunc: func(e event.GenericEvent) bool { return false },
			}),
		).
		Complete(r)
}

func NewPodForCR(cr *crdv1.Observer) *v12.Pod {
	labels := map[string]string{
		"app": cr.Name,
	}

	return &v12.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cr.Name,
			Namespace: cr.Namespace,
			Labels:    labels,
		},
		Spec: v12.PodSpec{
			Containers: []v12.Container{
				{
					Name:    "busybox",
					Image:   "curlimages/curl:7.78.0",
					Command: []string{"/bin/sh", "-c", fmt.Sprintf("curl -vvv -L %s", cr.Spec.Endpoint)},
				},
			},
			RestartPolicy: v12.RestartPolicyOnFailure,
		},
	}
}
