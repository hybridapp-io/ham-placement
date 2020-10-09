// Copyright 2019 The Kubernetes Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package placementrule

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic"
	"k8s.io/klog"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	corev1alpha1 "github.com/hybridapp-io/ham-placement/pkg/apis/core/v1alpha1"
)

var PlacementDecisionMaker DecisionMaker

/**
* USER ACTION REQUIRED: This is a scaffold file intended for the user to modify with their own Controller
* business logic.  Delete these comments after modifying this file.*
 */

// Add creates a new PlacementRule Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	if PlacementDecisionMaker == nil {
		PlacementDecisionMaker = &DefaultDecisionMaker{}
	}

	rec := &ReconcilePlacementRule{
		client:        mgr.GetClient(),
		scheme:        mgr.GetScheme(),
		dynamicClient: dynamic.NewForConfigOrDie(mgr.GetConfig()),
		decisionMaker: PlacementDecisionMaker,
	}

	return rec
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("placementrule-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource PlacementRule
	err = c.Watch(&source.Kind{Type: &corev1alpha1.PlacementRule{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	return nil
}

// blank assignment to verify that ReconcilePlacementRule implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcilePlacementRule{}

// ReconcilePlacementRule reconciles a PlacementRule object
type ReconcilePlacementRule struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client        client.Client
	scheme        *runtime.Scheme
	dynamicClient dynamic.Interface
	decisionMaker DecisionMaker
}

// Reconcile reads that state of the cluster for a PlacementRule object and makes changes based on the state read
// and what is in the PlacementRule.Spec
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcilePlacementRule) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	klog.Info("Reconciling PlacementRule ", request.NamespacedName)

	// Fetch the PlacementRule instance
	instance := &corev1alpha1.PlacementRule{}

	err := r.client.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	// Step 1: generate new candidates from spec
	ncans, err := r.generateCandidates(instance)
	if err != nil {
		klog.Error("Failed to generate candidates for decision with error: ", err)
	}

	// if spec has been changed, reset it
	if instance.Status.ObservedGeneration != instance.GetGeneration() || !isSameCandidateList(ncans, instance) {
		err = r.resetDecisionMakingProcess(ncans, instance)
		if err != nil {
			klog.Error("Following error occurred during resetDecisionMakingProcess: ", err)
		}

		return reconcile.Result{}, err
	}

	return reconcile.Result{}, r.continueDecisionMakingProcess(instance)
}

func (r *ReconcilePlacementRule) resetDecisionMakingProcess(candidates []corev1.ObjectReference, instance *corev1alpha1.PlacementRule) error {
	instance.Status.ObservedGeneration = instance.GetGeneration()
	now := metav1.Now()
	instance.Status.LastUpdateTime = &now
	instance.Status.Candidates = candidates
	instance.Status.Recommendations = nil
	instance.Status.Eliminators = nil

	r.decisionMaker.ResetDecisionMakingProcess(candidates, instance)

	return r.client.Status().Update(context.TODO(), instance)
}

func (r *ReconcilePlacementRule) continueDecisionMakingProcess(instance *corev1alpha1.PlacementRule) error {
	readytodecide := true

	for _, adv := range instance.Spec.Advisors {
		if instance.Status.Recommendations == nil {
			readytodecide = false
			break
		}

		if _, ok := instance.Status.Recommendations[adv.Name]; !ok {
			readytodecide = false
			break
		}
	}

	if readytodecide && r.decisionMaker.ContinueDecisionMakingProcess(instance) {
		return r.client.Status().Update(context.TODO(), instance)
	}

	return nil
}
