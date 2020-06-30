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

package veto

import (
	"context"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/klog"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	advisorutils "github.com/hybridapp-io/ham-placement/pkg/advisor/utils"
	corev1alpha1 "github.com/hybridapp-io/ham-placement/pkg/apis/core/v1alpha1"
)

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
	rec := &ReconcileVetoAdvisor{
		client: mgr.GetClient(),
		scheme: mgr.GetScheme(),
	}

	return rec
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("veto-advisor", mgr, controller.Options{Reconciler: r})
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

// blank assignment to verify that ReconcileVetoAdvisor implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileVetoAdvisor{}

// ReconcileVetoAdvisor reconciles a PlacementRule object
type ReconcileVetoAdvisor struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a PlacementRule object and makes changes based on the state read
// and what is in the PlacementRule.Spec
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileVetoAdvisor) Reconcile(request reconcile.Request) (reconcile.Result, error) {
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

	advisor := advisorutils.GetAdvisor(instance, advisorName)
	if advisor == nil || advisorutils.Recommended(instance, advisorName) {
		return reconcile.Result{}, nil
	}

	rec := r.Recommend(instance, advisor)
	klog.Info("Veto advising placementRule ", request.NamespacedName, " targets: ", rec)

	if !advisorutils.IsSameRecommendataion(instance, advisorName, rec) {
		advisorutils.MakeRecommendataion(instance, advisorName, rec)
		err = r.client.Status().Update(context.TODO(), instance)
	}

	if err != nil {
		klog.Error("Veto failed to provide recommendation, error: ", err)
	}

	return reconcile.Result{}, err
}
