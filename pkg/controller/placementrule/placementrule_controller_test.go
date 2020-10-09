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
	"testing"
	"time"

	. "github.com/onsi/gomega"

	corev1alpha1 "github.com/hybridapp-io/ham-placement/pkg/apis/core/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	clusterv1alpha1 "k8s.io/cluster-registry/pkg/apis/clusterregistry/v1alpha1"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const timeout = time.Second * 30
const interval = time.Second * 1

var (
	prName      = "testhpr"
	prNamespace = "default"

	prKey = types.NamespacedName{
		Name:      prName,
		Namespace: prNamespace,
	}

	expectedRequest = reconcile.Request{
		NamespacedName: prKey,
	}

	placementRule = &corev1alpha1.PlacementRule{
		ObjectMeta: metav1.ObjectMeta{
			Name:      prName,
			Namespace: prNamespace,
		},
	}

	kubernetesDeployerName = "kubernetes"
	kubernetesDeployerKey  = types.NamespacedName{
		Name:      kubernetesDeployerName,
		Namespace: "default",
	}
	kubernetesDeployer = &corev1alpha1.Deployer{
		TypeMeta: metav1.TypeMeta{
			Kind: "Deployer",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        kubernetesDeployerKey.Name,
			Namespace:   kubernetesDeployerKey.Namespace,
			Annotations: map[string]string{corev1alpha1.DeployerInCluster: "true"},
		},
		Spec: corev1alpha1.DeployerSpec{
			Type:  kubernetesDeployerName,
			Scope: apiextensions.ClusterScoped,
		},
	}

	ibminfraDeployerName = "ibminfra"
	ibminfraDeployerKey  = types.NamespacedName{
		Name:      ibminfraDeployerName,
		Namespace: "default",
	}
	ibminfraDeployer = &corev1alpha1.Deployer{
		TypeMeta: metav1.TypeMeta{
			Kind: "Deployer",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        ibminfraDeployerKey.Name,
			Namespace:   ibminfraDeployerKey.Namespace,
			Annotations: map[string]string{corev1alpha1.DeployerInCluster: "true"},
		},
		Spec: corev1alpha1.DeployerSpec{
			Type:  ibminfraDeployerName,
			Scope: apiextensions.ClusterScoped,
		},
	}

	mc1Name = "mc1"
	mc1Key  = types.NamespacedName{
		Name:      mc1Name,
		Namespace: mc1Name,
	}
	mc1 = &clusterv1alpha1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      mc1Name,
			Namespace: mc1Name,
		},
	}

	mc1NS = corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: mc1Name,
		},
	}
)

func TestReconcile(t *testing.T) {
	g := NewWithT(t)

	var c client.Client

	// Setup the Manager and Controller.  Wrap the Controller Reconcile function so it writes each request to a
	// channel when it is finished.
	mgr, err := manager.New(cfg, manager.Options{MetricsBindAddress: "0"})
	g.Expect(err).NotTo(HaveOccurred())

	c = mgr.GetClient()

	rec := newReconciler(mgr)
	recFn, requests := SetupTestReconcile(rec)

	g.Expect(add(mgr, recFn)).To(Succeed())

	stopMgr, mgrStopped := StartTestManager(mgr, g)

	defer func() {
		close(stopMgr)
		mgrStopped.Wait()
	}()

	pl := &corev1alpha1.PlacementRule{}
	pl.Name = prName
	pl.Namespace = prNamespace
	g.Expect(c.Create(context.TODO(), pl)).To(Succeed())
	defer func() {
		if err = c.Delete(context.TODO(), pl); err != nil {
			klog.Error(err)
			t.Fail()
		}
	}()

	g.Eventually(requests, timeout, interval).Should(Receive(Equal(expectedRequest)))
}

func TestEmptyDecisions(t *testing.T) {
	g := NewWithT(t)

	var c client.Client

	// Setup the Manager and Controller.  Wrap the Controller Reconcile function so it writes each request to a
	// channel when it is finished.
	mgr, err := manager.New(cfg, manager.Options{MetricsBindAddress: "0"})
	g.Expect(err).NotTo(HaveOccurred())

	c = mgr.GetClient()

	rec := newReconciler(mgr)
	recFn, requests := SetupTestReconcile(rec)

	g.Expect(add(mgr, recFn)).To(Succeed())

	stopMgr, mgrStopped := StartTestManager(mgr, g)

	defer func() {
		close(stopMgr)
		mgrStopped.Wait()
	}()

	pr := placementRule.DeepCopy()
	defer func() {
		if err = c.Delete(context.TODO(), pr); err != nil {
			klog.Error(err)
			t.Fail()
		}
	}()

	g.Expect(c.Create(context.TODO(), pr)).To(Succeed())

	// wait for maiin reconciliation
	g.Eventually(requests, timeout, interval).Should(Receive(Equal(expectedRequest)))

	// wait for reconciliation triggered by status update
	g.Eventually(requests, timeout, interval).Should(Receive(Equal(expectedRequest)))

	// reload the object, as status has been populated in the meantime
	g.Expect(c.Get(context.TODO(), prKey, pr)).NotTo(HaveOccurred())

	// no candidates and no decisions
	g.Expect(pr.Status.Candidates).To(BeEmpty())
	g.Expect(pr.Status.Decisions).To(BeEmpty())

}

func TestDefaultCandidates(t *testing.T) {
	g := NewWithT(t)

	var c client.Client

	// Setup the Manager and Controller.  Wrap the Controller Reconcile function so it writes each request to a
	// channel when it is finished.
	mgr, err := manager.New(cfg, manager.Options{MetricsBindAddress: "0"})
	g.Expect(err).NotTo(HaveOccurred())

	c = mgr.GetClient()

	rec := newReconciler(mgr)
	recFn, requests := SetupTestReconcile(rec)

	g.Expect(add(mgr, recFn)).To(Succeed())

	stopMgr, mgrStopped := StartTestManager(mgr, g)

	defer func() {
		close(stopMgr)
		mgrStopped.Wait()
	}()
	ns1 := mc1NS.DeepCopy()
	g.Expect(c.Create(context.TODO(), ns1)).To(Succeed())

	pr := placementRule.DeepCopy()
	defer func() {
		if err = c.Delete(context.TODO(), pr); err != nil {
			klog.Error(err)
			t.Fail()
		}
	}()

	cl1 := mc1.DeepCopy()
	g.Expect(c.Create(context.TODO(), cl1)).NotTo(HaveOccurred())
	// reload the cluster object
	g.Expect(c.Get(context.TODO(), mc1Key, cl1)).NotTo(HaveOccurred())

	defer func() {
		if err = c.Delete(context.TODO(), cl1); err != nil {
			klog.Error(err)
			t.Fail()
		}
	}()

	g.Expect(c.Create(context.TODO(), pr)).To(Succeed())

	// wait for maiin reconciliation
	g.Eventually(requests, timeout, interval).Should(Receive(Equal(expectedRequest)))

	// wait for reconciliation triggered by status update
	g.Eventually(requests, timeout, interval).Should(Receive(Equal(expectedRequest)))

	// reload the object, as status has been populated in the meantime
	g.Expect(c.Get(context.TODO(), prKey, pr)).NotTo(HaveOccurred())

	// no candidates and no decisions
	g.Expect(len(pr.Status.Candidates)).To(Equal(1))
	g.Expect(pr.Status.Candidates[0].Name).To(Equal(cl1.Name))
	g.Expect(pr.Status.Candidates[0].Namespace).To(Equal(cl1.Namespace))
	g.Expect(pr.Status.Candidates[0].APIVersion).To(Equal(cl1.APIVersion))

	g.Expect(len(pr.Status.Decisions)).To(Equal(1))
	g.Expect(pr.Status.Decisions[0].Name).To(Equal(cl1.Name))
	g.Expect(pr.Status.Decisions[0].Namespace).To(Equal(cl1.Namespace))
	g.Expect(pr.Status.Decisions[0].APIVersion).To(Equal(cl1.APIVersion))
}

func TestExplicitDeployerOnHub(t *testing.T) {
	g := NewWithT(t)

	var c client.Client

	// Setup the Manager and Controller.  Wrap the Controller Reconcile function so it writes each request to a
	// channel when it is finished.
	mgr, err := manager.New(cfg, manager.Options{MetricsBindAddress: "0"})
	g.Expect(err).NotTo(HaveOccurred())

	c = mgr.GetClient()

	rec := newReconciler(mgr)
	recFn, requests := SetupTestReconcile(rec)

	g.Expect(add(mgr, recFn)).To(Succeed())

	stopMgr, mgrStopped := StartTestManager(mgr, g)

	defer func() {
		close(stopMgr)
		mgrStopped.Wait()
	}()

	deployer := kubernetesDeployer.DeepCopy()
	defer func() {
		if err = c.Delete(context.TODO(), deployer); err != nil {
			klog.Error(err)
			t.Fail()
		}
	}()
	g.Expect(c.Create(context.TODO(), deployer)).To(Succeed())
	// reload the deployer object
	g.Expect(c.Get(context.TODO(), kubernetesDeployerKey, deployer)).NotTo(HaveOccurred())

	pr := placementRule.DeepCopy()
	pr.Spec.DeployerType = &kubernetesDeployerName
	defer func() {
		if err = c.Delete(context.TODO(), pr); err != nil {
			klog.Error(err)
			t.Fail()
		}
	}()

	g.Expect(c.Create(context.TODO(), pr)).To(Succeed())

	// wait for maiin reconciliation
	g.Eventually(requests, timeout, interval).Should(Receive(Equal(expectedRequest)))

	// wait for reconciliation triggered by status update
	g.Eventually(requests, timeout, interval).Should(Receive(Equal(expectedRequest)))

	// reload the object, as status has been populated in the meantime
	g.Expect(c.Get(context.TODO(), prKey, pr)).NotTo(HaveOccurred())

	// no candidates and no decisions
	g.Expect(len(pr.Status.Candidates)).To(Equal(1))
	g.Expect(pr.Status.Candidates[0].Name).To(Equal(deployer.Name))
	g.Expect(pr.Status.Candidates[0].Namespace).To(Equal(deployer.Namespace))
	g.Expect(pr.Status.Candidates[0].APIVersion).To(Equal(deployer.APIVersion))

	g.Expect(len(pr.Status.Decisions)).To(Equal(1))
	g.Expect(pr.Status.Decisions[0].Name).To(Equal(deployer.Name))
	g.Expect(pr.Status.Decisions[0].Namespace).To(Equal(deployer.Namespace))
	g.Expect(pr.Status.Decisions[0].APIVersion).To(Equal(deployer.APIVersion))
}

func TestExplicitDeployersOnHub(t *testing.T) {
	g := NewWithT(t)

	var c client.Client

	// Setup the Manager and Controller.  Wrap the Controller Reconcile function so it writes each request to a
	// channel when it is finished.
	mgr, err := manager.New(cfg, manager.Options{MetricsBindAddress: "0"})
	g.Expect(err).NotTo(HaveOccurred())

	c = mgr.GetClient()

	rec := newReconciler(mgr)
	recFn, requests := SetupTestReconcile(rec)

	g.Expect(add(mgr, recFn)).To(Succeed())

	stopMgr, mgrStopped := StartTestManager(mgr, g)

	defer func() {
		close(stopMgr)
		mgrStopped.Wait()
	}()

	kubernetesDeployer := kubernetesDeployer.DeepCopy()
	defer func() {
		if err = c.Delete(context.TODO(), kubernetesDeployer); err != nil {
			klog.Error(err)
			t.Fail()
		}
	}()
	g.Expect(c.Create(context.TODO(), kubernetesDeployer)).To(Succeed())

	ibminfraDeployer := ibminfraDeployer.DeepCopy()
	defer func() {
		if err = c.Delete(context.TODO(), ibminfraDeployer); err != nil {
			klog.Error(err)
			t.Fail()
		}
	}()
	g.Expect(c.Create(context.TODO(), ibminfraDeployer)).To(Succeed())
	// reload the deployer object
	g.Expect(c.Get(context.TODO(), ibminfraDeployerKey, ibminfraDeployer)).NotTo(HaveOccurred())

	pr := placementRule.DeepCopy()
	pr.Spec.DeployerType = &ibminfraDeployerName
	defer func() {
		if err = c.Delete(context.TODO(), pr); err != nil {
			klog.Error(err)
			t.Fail()
		}
	}()

	g.Expect(c.Create(context.TODO(), pr)).To(Succeed())

	// wait for maiin reconciliation
	g.Eventually(requests, timeout, interval).Should(Receive(Equal(expectedRequest)))

	// wait for reconciliation triggered by status update
	g.Eventually(requests, timeout, interval).Should(Receive(Equal(expectedRequest)))

	// reload the object, as status has been populated in the meantime
	g.Expect(c.Get(context.TODO(), prKey, pr)).NotTo(HaveOccurred())

	// no candidates and no decisions
	g.Expect(len(pr.Status.Candidates)).To(Equal(1))
	g.Expect(pr.Status.Candidates[0].Name).To(Equal(ibminfraDeployer.Name))
	g.Expect(pr.Status.Candidates[0].Namespace).To(Equal(ibminfraDeployer.Namespace))
	g.Expect(pr.Status.Candidates[0].APIVersion).To(Equal(ibminfraDeployer.APIVersion))

	g.Expect(len(pr.Status.Decisions)).To(Equal(1))
	g.Expect(pr.Status.Decisions[0].Name).To(Equal(ibminfraDeployer.Name))
	g.Expect(pr.Status.Decisions[0].Namespace).To(Equal(ibminfraDeployer.Namespace))
	g.Expect(pr.Status.Decisions[0].APIVersion).To(Equal(ibminfraDeployer.APIVersion))
}
