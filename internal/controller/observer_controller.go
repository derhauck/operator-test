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
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"strconv"
	"time"
)

// ObserverReconciler reconciles a Observer object
type ObserverReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

const observerAnnotationTimeToDelete = "observer.crd.test.kateops.com/ttd"
const observerAnnotationProcessed = "observer.crd.test.kateops.com/processed"
const observerFinalizer = "crd.test.kateops.com/finalizer"

func (r *ObserverReconciler) reFetchObserver(ctx context.Context, observer *crdv1.Observer) error {
	updatedObserver := crdv1.Observer{}
	logger := log.FromContext(ctx)
	namespacedName := types.NamespacedName{Name: observer.Name, Namespace: observer.Namespace}
	if err := r.Get(ctx, namespacedName, &updatedObserver); err != nil {
		logger.Error(err, "Failed to re-fetch Observer")
		return err
	}
	observer = &updatedObserver

	return nil
}

func (r *ObserverReconciler) checkConditions(ctx context.Context, observer *crdv1.Observer) (reconcile.Result, error) {
	if observer.Status.Conditions == nil || len(observer.Status.Conditions) == 0 {
		logger := log.FromContext(ctx)

		if err := r.reFetchObserver(ctx, observer); err != nil {
			return ctrl.Result{}, err
		}

		observer.Status.SetStatusCondition(
			crdv1.TypeAvailableObserver,
			metav1.ConditionUnknown,
			"Reconciling",
			fmt.Sprintf("Starting reconciliation for observer: %s ", observer.Name),
		)
		observer.Status.SetStatusCondition(
			crdv1.TypeSuccessLastStatusObserver,
			metav1.ConditionUnknown,
			"Finalizing",
			fmt.Sprintf("Starting reconciliation for observer: %s ", observer.Name),
		)
		observer.Status.SetStatusCondition(
			crdv1.TypeSuccessStatusForLastFiveObserver,
			metav1.ConditionUnknown,
			"Finalizing",
			fmt.Sprintf("Starting reconciliation for observer: %s ", observer.Name),
		)
		observer.Status.InitIterationResults()
		if err := r.Status().Update(ctx, observer); err != nil {
			logger.Error(err, "Failed to update Observer status")
			return ctrl.Result{}, err
		}
	}
	return reconcile.Result{}, nil
}

func (r *ObserverReconciler) addFinalizer(ctx context.Context, observer *crdv1.Observer) (reconcile.Result, error) {
	logger := log.FromContext(ctx)

	if !controllerutil.ContainsFinalizer(observer, observerFinalizer) {
		logger.Info("Adding Finalizer to Observer")

		if err := r.reFetchObserver(ctx, observer); err != nil {
			return ctrl.Result{}, err
		}

		if ok := controllerutil.AddFinalizer(observer, observerFinalizer); !ok {
			logger.Info("Failed to add finalizer into the custom resource")
			return ctrl.Result{Requeue: true}, nil
		}

		if err := r.Update(ctx, observer); err != nil {
			logger.Error(err, "Failed to update custom resource to add finalizer")
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

func (r *ObserverReconciler) deleteObserver(ctx context.Context, observer *crdv1.Observer) (reconcile.Result, error) {
	if controllerutil.ContainsFinalizer(observer, observerFinalizer) {
		logger := log.FromContext(ctx)
		logger.Info("Performing Finalizer Operations for Observer before deleting CR")

		if err := r.reFetchObserver(ctx, observer); err != nil {
			return ctrl.Result{}, err
		}

		observer.Status.SetStatusCondition(
			crdv1.TypeDegradedObserver,
			metav1.ConditionUnknown,
			"Finalizing",
			fmt.Sprintf("Performing finalizer operations for the custom resource: %s ", observer.Name),
		)

		if err := r.Status().Update(ctx, observer); err != nil {
			logger.Error(err, "Failed to update Observer status")
			return ctrl.Result{}, err
		}

		if err := r.doFinalizerOperationsForObserver(observer); err != nil {
			logger.Error(err, "Failed to execute Finalizers")
			return ctrl.Result{Requeue: true}, err
		}

		if err := r.reFetchObserver(ctx, observer); err != nil {
			return ctrl.Result{}, err
		}
		observer.Status.SetStatusCondition(
			crdv1.TypeDegradedObserver,
			metav1.ConditionTrue,
			"Finalizing",
			fmt.Sprintf("Finalizer operations for Observer %s name were successfully accomplished", observer.Name),
		)

		if err := r.Status().Update(ctx, observer); err != nil {
			logger.Error(err, "Failed to update Observer status")
			return ctrl.Result{}, err
		}

		logger.Info("Removing Finalizer for Observer after successfully perform the operations")
		if ok := controllerutil.RemoveFinalizer(observer, observerFinalizer); !ok {
			logger.Info("Failed to remove finalizer for Observer")
			return ctrl.Result{Requeue: true}, nil
		}

		if err := r.Update(ctx, observer); err != nil {
			logger.Error(err, "Failed to remove finalizer for Observer")
			return ctrl.Result{}, err
		}
	}
	return ctrl.Result{}, nil
}

func (r *ObserverReconciler) deletePod(ctx context.Context, observer *crdv1.Observer, found *v1.Pod) (ctrl.Result, error) {
	if found.Status.Phase == "Succeeded" || found.Status.Phase == "Failed" {
		logger := log.FromContext(ctx)

		if observer.GetLabels()[observerAnnotationTimeToDelete] == "" {
			labels := found.GetLabels()
			if labels == nil {
				labels = make(map[string]string, 1)
			}
			labels[observerAnnotationTimeToDelete] = "true"

			if err := r.reFetchObserver(ctx, observer); err != nil {
				return ctrl.Result{}, nil
			}
			observer.Labels = labels
			if err := r.Update(ctx, observer); err != nil {
				//logger.Error(err, "Can not update Observer labels")
				return reconcile.Result{Requeue: true}, nil
			}
			logger.Info("Updated Observer Labels")
			return reconcile.Result{RequeueAfter: time.Duration(time.Second.Seconds() * float64(observer.Spec.RetryAfterSeconds))}, nil
		}

		if found.GetAnnotations()[observerAnnotationTimeToDelete] != "" {
			ttd, err := strconv.ParseInt(found.GetAnnotations()[observerAnnotationTimeToDelete], 10, 64)
			if err != nil {
				logger.Error(err, "can not get time from pod annotation")
				ttd = time.Now().Add(-time.Second).Unix()
			}
			now := time.Now().Unix()
			if now >= time.Unix(ttd, 0).Unix() {
				if err := r.reFetchObserver(ctx, observer); err != nil {
					return ctrl.Result{}, err
				}
				observer.Status.SetStatusCondition(
					crdv1.TypeAvailableObserver,
					metav1.ConditionFalse,
					"Reconciling",
					fmt.Sprintf("Deleting old Pod for custom resource (%s)", observer.Name),
				)
				if err := r.Status().Update(ctx, observer); err != nil {
					logger.Error(err, "Failed to update Observer status")
					return ctrl.Result{}, nil
				}
				logger.Info("Deleting old Pod", "now", now, "ttd", ttd)
				if err = r.Delete(ctx, found); err != nil {
					logger.Error(err, "Can not deleteObserver old Pod")
					return ctrl.Result{}, err
				}
				if err := r.reFetchObserver(ctx, observer); err != nil {
					return ctrl.Result{}, err
				}

				labels := observer.GetLabels()
				labels[observerAnnotationTimeToDelete] = ""
				observer.Labels = labels
				if err = r.Update(ctx, observer); err != nil {
					logger.Error(err, "Can not update Observer labels")
					return reconcile.Result{Requeue: true}, nil
				}
				if err := r.reFetchObserver(ctx, observer); err != nil {
					return ctrl.Result{}, err
				}
				observer.Status.SetStatusCondition(
					crdv1.TypeAvailableObserver,
					metav1.ConditionTrue,
					"Reconciling",
					fmt.Sprintf("Waiting for new reconciliation for custom resource (%s)", observer.Name),
				)
				if err := r.Status().Update(ctx, observer); err != nil {
					logger.Error(err, "Failed to update Observer status")
					return ctrl.Result{}, err
				}

				// increment the current item to prepare the next pod
				result, err := r.iterateCurrentItem(ctx, observer)
				if err != nil {
					logger.Error(err, "Can not iterate current result Item")
					return result, nil
				}
			}
		}
	}
	return reconcile.Result{}, nil
}

func (r *ObserverReconciler) createPod(ctx context.Context, observer *crdv1.Observer) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	pod, err := observer.NewPod()
	if err != nil {
		return ctrl.Result{}, err
	}
	logger.Info("Creating new Pod", "Pod.Namespace", pod.Namespace, "Pod.Name", pod.Name)
	if err := ctrl.SetControllerReference(observer, pod, r.Scheme); err != nil {
		return ctrl.Result{}, err
	}
	if err = r.Create(ctx, pod); err != nil {
		logger.Error(err, "Failed to create new Pod", "Pod.Namespace", pod.Namespace, "Pod.Name", pod.Name)
		return ctrl.Result{}, err
	}
	observer.Status.SetStatusCondition(
		crdv1.TypeAvailableObserver,
		metav1.ConditionTrue,
		"Reconciling",
		fmt.Sprintf("Created new Pod for crd %s", observer.Name),
	)

	if err := r.Status().Update(ctx, observer); err != nil {
		logger.Error(err, "Failed to update Observer status")
		return ctrl.Result{}, err
	}
	logger.Info("Created new Pod", "Pod.Namespace", pod.Namespace, "Pod.Name", pod.Name)
	return ctrl.Result{}, nil
}

func (r *ObserverReconciler) iterateCurrentItem(ctx context.Context, observer *crdv1.Observer) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	if err := r.reFetchObserver(ctx, observer); err != nil {
		return ctrl.Result{}, err
	}

	logger.Info("updating current item", "current", observer.Status.CurrentItem)
	maxLength := len(observer.Spec.Entries) - 1
	item := observer.Status.CurrentItem
	item++
	if item > maxLength {
		item = 0
	}
	logger.Info("updating current item", "number", item)
	observer.Status.CurrentItem = item

	if err := r.Status().Update(ctx, observer); err != nil {
		logger.Error(err, "Failed to update Observer status")
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}

//+kubebuilder:rbac:groups=crd.test.kateops.com,resources=observers,verbs=get;list;watch;create;update;patch;deleteObserver
//+kubebuilder:rbac:groups=crd.test.kateops.com,resources=observers/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=crd.test.kateops.com,resources=observers/finalizers,verbs=update

func (r *ObserverReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	var observer = crdv1.Observer{}
	//logger.Info("Observer Reconciling")
	if err := r.Get(ctx, req.NamespacedName, &observer); err != nil {
		if errors.IsNotFound(err) {
			logger.Info("Observer resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		logger.Error(err, "Failed to get Observer")
		return ctrl.Result{}, err
	}

	if result, err := r.checkConditions(ctx, &observer); err != nil {
		return result, err
	}

	if result, err := r.addFinalizer(ctx, &observer); err != nil {
		return result, err
	}

	isObserverToBeDeleted := observer.DeletionTimestamp != nil
	if isObserverToBeDeleted {
		if result, err := r.deleteObserver(ctx, &observer); err != nil {
			return result, err
		}
	}

	// check if observer is currently not available - in process => available=false
	if observer.IsStatusConditionFalse(crdv1.TypeAvailableObserver) {
		logger.Info("Observer already busy, will try again in 5s")
		return reconcile.Result{RequeueAfter: time.Second * 5}, nil
	}

	// create new Pod if not exists
	found := &v1.Pod{}
	if err := r.Get(ctx, types.NamespacedName{Name: observer.GetCurrentPodName(), Namespace: observer.Namespace}, found); err != nil {
		if errors.IsNotFound(err) {
			if result, err := r.createPod(ctx, &observer); err != nil {
				return result, err
			}
		} else {
			return reconcile.Result{}, err
		}
	}
	//logger.Info("Observer Reconcile Logic")

	if result, err := r.deletePod(ctx, &observer, found); err != nil {
		return result, err
	}

	if err := r.reFetchObserver(ctx, &observer); err != nil {
		return ctrl.Result{}, err
	}
	observer.Status.SetStatusCondition(
		crdv1.TypeAvailableObserver,
		metav1.ConditionTrue,
		"Reconciling",
		fmt.Sprintf("Reconciling done for custom resource (%s) successfully", observer.Name),
	)

	if err := r.Status().Update(ctx, &observer); err != nil {
		//logger.Error(err, "Failed to update Observer status")
		return ctrl.Result{}, nil
	}
	return ctrl.Result{RequeueAfter: time.Second}, nil
}

func (r *ObserverReconciler) HandlePodEvents(ctx context.Context, obj client.Object) []reconcile.Request {
	namespacedName := types.NamespacedName{
		Namespace: obj.GetNamespace(),
		Name:      obj.GetName(),
	}
	logger := log.FromContext(ctx)
	var pod v1.Pod
	if err := r.Get(ctx, namespacedName, &pod); err != nil {
		return []reconcile.Request{}
	}

	annotations := pod.GetAnnotations()
	if annotations[observerAnnotationProcessed] != "" {
		return []reconcile.Request{}
	}
	if annotations == nil {
		annotations = make(map[string]string, 1)
	}

	annotations[observerAnnotationProcessed] = "true"

	var crdNamespaced types.NamespacedName
	for _, owner := range pod.OwnerReferences {
		if *owner.Controller {
			crdNamespaced = types.NamespacedName{
				Namespace: pod.Namespace,
				Name:      owner.Name,
			}
		}
	}
	observer := crdv1.Observer{}
	if err := r.Get(ctx, crdNamespaced, &observer); err != nil {
		logger.Error(err, "Can not get Observer")
		return []reconcile.Request{}
	}
	interval := time.Now()

	// when last element of entries then add interval to pause
	if observer.Status.CurrentItem == len(observer.Spec.Entries)-1 {
		interval = interval.Add(time.Second * time.Duration(observer.Spec.RetryAfterSeconds))
	}

	if observer.Status.CurrentItem == 0 {
		observer.Status.AddNewIterationResult()
	}

	if pod.Status.Phase == v1.PodSucceeded {
		observer.Status.SetStatusCondition(
			crdv1.TypeSuccessLastStatusObserver,
			metav1.ConditionTrue,
			"Finalizing",
			fmt.Sprintf("Last status was success for observer: %s ", observer.Name),
		)
	} else {
		observer.Status.SetStatusCondition(
			crdv1.TypeSuccessLastStatusObserver,
			metav1.ConditionFalse,
			"Finalizing",
			fmt.Sprintf("Last status was failure for observer: %s ", observer.Name),
		)
	}

	observer.Status.AddPodStatusToLatestIterationResult(crdv1.PodStatus{
		Status: pod.Status.Phase,
		Name:   observer.GetCurrentEntry().Name,
		Time:   time.Now().UTC().Format("2006-01-02 15:04:05"),
	}).EvaluateStatus()
	observer.Status.GetLatestIterationResult().Evaluate()
	annotations[observerAnnotationTimeToDelete] = fmt.Sprintf("%d", interval.Unix())
	pod.Annotations = annotations
	if err := r.Update(ctx, &pod); err != nil {
		return []reconcile.Request{}
	}
	_ = r.Status().Update(ctx, &observer)

	return []reconcile.Request{
		{NamespacedName: crdNamespaced},
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *ObserverReconciler) SetupWithManager(mgr ctrl.Manager) error {

	return ctrl.NewControllerManagedBy(mgr).
		For(&crdv1.Observer{}, builder.WithPredicates(predicate.GenerationChangedPredicate{})).
		Watches(
			&v1.Pod{},
			handler.EnqueueRequestsFromMapFunc(r.HandlePodEvents),
			builder.WithPredicates(predicate.Funcs{
				DeleteFunc:  func(e event.DeleteEvent) bool { return false },
				CreateFunc:  func(e event.CreateEvent) bool { return false },
				GenericFunc: func(e event.GenericEvent) bool { return false },
				UpdateFunc: func(e event.UpdateEvent) bool {
					newObject := e.ObjectNew.(*v1.Pod)
					oldObject := e.ObjectOld.(*v1.Pod)
					if (newObject.Status.Phase == "Succeeded" || newObject.Status.Phase == "Failed") && (oldObject.Status.Phase != "Succeeded" && oldObject.Status.Phase != "Failed") {
						return true
					}
					return false

				},
			},
			),
		).
		Complete(r)
}

func (r *ObserverReconciler) doFinalizerOperationsForObserver(_ *crdv1.Observer) error {
	return nil
}
