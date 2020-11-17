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
const defaultReplicas = 1

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

	// managed cluster 1
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

	// managed cluster 2
	mc2Name = "mc2"
	mc2     = &clusterv1alpha1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      mc2Name,
			Namespace: mc2Name,
		},
	}

	mc2NS = corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: mc2Name,
		},
	}

	// managed cluster 3
	mc3Name = "mc3"
	mc3     = &clusterv1alpha1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      mc3Name,
			Namespace: mc3Name,
		},
	}

	mc3NS = corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: mc3Name,
		},
	}

	AdvisorTypePriority  = corev1alpha1.AdvisorTypePriority
	AdvisorTypePredicate = corev1alpha1.AdvisorTypePredicate

	RHACMPriority = int16(120)
	rhacmAdvisor  = corev1alpha1.Advisor{
		Name:   "rhacm",
		Type:   &AdvisorTypePriority,
		Weight: &RHACMPriority,
	}

	CostPriority = int16(50)
	costAdvisor  = corev1alpha1.Advisor{
		Name:   "cost",
		Type:   &AdvisorTypePriority,
		Weight: &CostPriority,
	}

	grcAdvisor = corev1alpha1.Advisor{
		Name: "grc",
		Type: &AdvisorTypePredicate,
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

	ns1 := mc1NS.DeepCopy()
	g.Expect(c.Create(context.TODO(), ns1)).To(Succeed())

	ns2 := mc2NS.DeepCopy()
	g.Expect(c.Create(context.TODO(), ns2)).To(Succeed())

	ns3 := mc3NS.DeepCopy()
	g.Expect(c.Create(context.TODO(), ns3)).To(Succeed())

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

func TestSingleTarget(t *testing.T) {
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

	pr := placementRule.DeepCopy()
	// targets: cl1
	cl1Target := corev1.ObjectReference{
		Name:       cl1.Name,
		Namespace:  cl1.Namespace,
		APIVersion: cl1.APIVersion,
	}
	pr.Spec.Targets = []corev1.ObjectReference{
		cl1Target,
	}
	defer func() {
		if err = c.Delete(context.TODO(), pr); err != nil {
			klog.Error(err)
			t.Fail()
		}
	}()

	cl2 := mc2.DeepCopy()
	g.Expect(c.Create(context.TODO(), cl2)).NotTo(HaveOccurred())
	// reload the cluster object
	//g.Expect(c.Get(context.TODO(), mc2Key, cl2)).NotTo(HaveOccurred())

	defer func() {
		if err = c.Delete(context.TODO(), cl2); err != nil {
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

	// candidate cl1
	g.Expect(len(pr.Status.Candidates)).To(Equal(1))
	g.Expect(pr.Status.Candidates[0].Name).To(Equal(cl1.Name))
	g.Expect(pr.Status.Candidates[0].Namespace).To(Equal(cl1.Namespace))
	g.Expect(pr.Status.Candidates[0].APIVersion).To(Equal(cl1.APIVersion))

	// decision cl1
	g.Expect(len(pr.Status.Decisions)).To(Equal(1))
	g.Expect(pr.Status.Decisions[0].Name).To(Equal(cl1.Name))
	g.Expect(pr.Status.Decisions[0].Namespace).To(Equal(cl1.Namespace))
}

func TestTwoTargetsSingleReplica(t *testing.T) {
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

	cl1 := mc1.DeepCopy()
	g.Expect(c.Create(context.TODO(), cl1)).NotTo(HaveOccurred())
	// reload the cluster object
	//g.Expect(c.Get(context.TODO(), mc1Key, cl1)).NotTo(HaveOccurred())

	defer func() {
		if err = c.Delete(context.TODO(), cl1); err != nil {
			klog.Error(err)
			t.Fail()
		}
	}()

	cl2 := mc2.DeepCopy()
	g.Expect(c.Create(context.TODO(), cl2)).NotTo(HaveOccurred())
	// reload the cluster object
	//g.Expect(c.Get(context.TODO(), mc2Key, cl2)).NotTo(HaveOccurred())

	defer func() {
		if err = c.Delete(context.TODO(), cl2); err != nil {
			klog.Error(err)
			t.Fail()
		}
	}()

	pr := placementRule.DeepCopy()
	cl1Target := corev1.ObjectReference{
		Name:       cl1.Name,
		Namespace:  cl1.Namespace,
		APIVersion: cl1.APIVersion,
	}
	cl2Target := corev1.ObjectReference{
		Name:       cl2.Name,
		Namespace:  cl2.Namespace,
		APIVersion: cl2.APIVersion,
	}
	pr.Spec.Targets = []corev1.ObjectReference{
		cl1Target,
		cl2Target,
	}
	var replica int16 = int16(defaultReplicas)
	pr.Spec.Replicas = &replica
	defer func() {
		if err = c.Delete(context.TODO(), pr); err != nil {
			klog.Error(err)
			t.Fail()
		}
	}()

	g.Expect(c.Create(context.TODO(), pr)).To(Succeed())

	// wait for main reconciliation
	g.Eventually(requests, timeout, interval).Should(Receive(Equal(expectedRequest)))
	g.Expect(c.Get(context.TODO(), prKey, pr)).NotTo(HaveOccurred())

	g.Expect(len(pr.Status.Candidates)).To(Equal(2))
	g.Expect(len(pr.Status.Decisions)).To(Equal(0))

	g.Eventually(requests, timeout, interval).Should(Receive(Equal(expectedRequest)))
	g.Expect(c.Get(context.TODO(), prKey, pr)).NotTo(HaveOccurred())

	g.Expect(len(pr.Status.Candidates)).To(Equal(1))
	g.Expect(len(pr.Status.Decisions)).To(Equal(0))

	g.Eventually(requests, timeout, interval).Should(Receive(Equal(expectedRequest)))
	g.Expect(c.Get(context.TODO(), prKey, pr)).NotTo(HaveOccurred())

	g.Expect(len(pr.Status.Candidates)).To(Equal(1))
	g.Expect(len(pr.Status.Decisions)).To(Equal(1))
	g.Expect(pr.Status.Candidates[0].Name).To(Equal(pr.Status.Decisions[0].Name))
	g.Expect(pr.Status.Candidates[0].Namespace).To(Equal(pr.Status.Decisions[0].Namespace))
}

func TestTwoTargetsTwoReplicas(t *testing.T) {
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

	cl1 := mc1.DeepCopy()
	g.Expect(c.Create(context.TODO(), cl1)).NotTo(HaveOccurred())
	// reload the cluster object
	//g.Expect(c.Get(context.TODO(), mc1Key, cl1)).NotTo(HaveOccurred())

	defer func() {
		if err = c.Delete(context.TODO(), cl1); err != nil {
			klog.Error(err)
			t.Fail()
		}
	}()

	cl2 := mc2.DeepCopy()
	g.Expect(c.Create(context.TODO(), cl2)).NotTo(HaveOccurred())
	// reload the cluster object
	//g.Expect(c.Get(context.TODO(), mc2Key, cl2)).NotTo(HaveOccurred())

	defer func() {
		if err = c.Delete(context.TODO(), cl2); err != nil {
			klog.Error(err)
			t.Fail()
		}
	}()

	pr := placementRule.DeepCopy()
	cl1Target := corev1.ObjectReference{
		Name:       cl1.Name,
		Namespace:  cl1.Namespace,
		APIVersion: cl1.APIVersion,
	}
	cl2Target := corev1.ObjectReference{
		Name:       cl2.Name,
		Namespace:  cl2.Namespace,
		APIVersion: cl2.APIVersion,
	}
	pr.Spec.Targets = []corev1.ObjectReference{
		cl1Target,
		cl2Target,
	}
	var replica int16 = int16(2)
	pr.Spec.Replicas = &replica
	defer func() {
		if err = c.Delete(context.TODO(), pr); err != nil {
			klog.Error(err)
			t.Fail()
		}
	}()

	g.Expect(c.Create(context.TODO(), pr)).To(Succeed())

	// wait for main reconciliation
	g.Eventually(requests, timeout, interval).Should(Receive(Equal(expectedRequest)))
	g.Expect(c.Get(context.TODO(), prKey, pr)).NotTo(HaveOccurred())

	g.Expect(len(pr.Status.Candidates)).To(Equal(2))
	g.Expect(len(pr.Status.Decisions)).To(Equal(0))

	g.Eventually(requests, timeout, interval).Should(Receive(Equal(expectedRequest)))
	g.Expect(c.Get(context.TODO(), prKey, pr)).NotTo(HaveOccurred())

	g.Expect(len(pr.Status.Candidates)).To(Equal(2))
	g.Expect(len(pr.Status.Decisions)).To(Equal(2))
}

func TestTwoTargetsWithLabelsTwoReplicas(t *testing.T) {
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

	labelsMap := make(map[string]string)
	labelsMap["test_label"] = "test"

	cl1 := mc1.DeepCopy()
	cl1.Labels = labelsMap
	g.Expect(c.Create(context.TODO(), cl1)).NotTo(HaveOccurred())
	// reload the cluster object
	//g.Expect(c.Get(context.TODO(), mc1Key, cl1)).NotTo(HaveOccurred())

	defer func() {
		if err = c.Delete(context.TODO(), cl1); err != nil {
			klog.Error(err)
			t.Fail()
		}
	}()

	cl2 := mc2.DeepCopy()
	cl2.Labels = labelsMap
	g.Expect(c.Create(context.TODO(), cl2)).NotTo(HaveOccurred())
	// reload the cluster object
	//g.Expect(c.Get(context.TODO(), mc2Key, cl2)).NotTo(HaveOccurred())

	defer func() {
		if err = c.Delete(context.TODO(), cl2); err != nil {
			klog.Error(err)
			t.Fail()
		}
	}()

	pr := placementRule.DeepCopy()
	cl1Target := corev1.ObjectReference{
		Name:       cl1.Name,
		Namespace:  cl1.Namespace,
		APIVersion: cl1.APIVersion,
	}
	cl2Target := corev1.ObjectReference{
		Name:       cl2.Name,
		Namespace:  cl2.Namespace,
		APIVersion: cl2.APIVersion,
	}
	pr.Spec.Targets = []corev1.ObjectReference{
		cl1Target,
		cl2Target,
	}
	var lSelector = metav1.LabelSelector{
		MatchLabels: labelsMap,
	}
	pr.Spec.TargetLabels = &lSelector
	var replica int16 = int16(2)
	pr.Spec.Replicas = &replica
	defer func() {
		if err = c.Delete(context.TODO(), pr); err != nil {
			klog.Error(err)
			t.Fail()
		}
	}()

	g.Expect(c.Create(context.TODO(), pr)).To(Succeed())

	// wait for main reconciliation
	g.Eventually(requests, timeout, interval).Should(Receive(Equal(expectedRequest)))
	g.Expect(c.Get(context.TODO(), prKey, pr)).NotTo(HaveOccurred())

	g.Expect(len(pr.Status.Candidates)).To(Equal(2))
	g.Expect(len(pr.Status.Decisions)).To(Equal(0))

	g.Eventually(requests, timeout, interval).Should(Receive(Equal(expectedRequest)))
	g.Expect(c.Get(context.TODO(), prKey, pr)).NotTo(HaveOccurred())

	g.Expect(len(pr.Status.Candidates)).To(Equal(2))
	g.Expect(len(pr.Status.Decisions)).To(Equal(2))
}

func TestCandidates(t *testing.T) {
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

	cl1 := mc1.DeepCopy()
	g.Expect(c.Create(context.TODO(), cl1)).NotTo(HaveOccurred())
	// reload the cluster object
	//g.Expect(c.Get(context.TODO(), mc1Key, cl1)).NotTo(HaveOccurred())

	defer func() {
		if err = c.Delete(context.TODO(), cl1); err != nil {
			klog.Error(err)
			t.Fail()
		}
	}()

	cl2 := mc2.DeepCopy()
	g.Expect(c.Create(context.TODO(), cl2)).NotTo(HaveOccurred())
	// reload the cluster object
	//g.Expect(c.Get(context.TODO(), mc2Key, cl2)).NotTo(HaveOccurred())

	defer func() {
		if err = c.Delete(context.TODO(), cl2); err != nil {
			klog.Error(err)
			t.Fail()
		}
	}()

	cl3 := mc3.DeepCopy()
	g.Expect(c.Create(context.TODO(), cl3)).NotTo(HaveOccurred())
	// reload the cluster object
	//g.Expect(c.Get(context.TODO(), mc3Key, cl3)).NotTo(HaveOccurred())

	defer func() {
		if err = c.Delete(context.TODO(), cl3); err != nil {
			klog.Error(err)
			t.Fail()
		}
	}()

	/**
		- no deployer type with 3 managed clusters, weights and no target
		- input :
			3 managed clusters
			no deployers
			placement rule with empty specs and replica = 1
		- expected output after hpr reconciliation:
			3 candidates
			one decision
	**/

	pr := placementRule.DeepCopy()
	var replica int16 = int16(defaultReplicas)
	pr.Spec.Replicas = &replica
	defer func() {
		if err = c.Delete(context.TODO(), pr); err != nil {
			klog.Error(err)
			t.Fail()
		}
	}()

	g.Expect(c.Create(context.TODO(), pr)).To(Succeed())

	// wait for main reconciliation
	g.Eventually(requests, timeout, interval).Should(Receive(Equal(expectedRequest)))

	// 3 candidates and no decisions
	g.Expect(c.Get(context.TODO(), prKey, pr)).NotTo(HaveOccurred())
	g.Expect(len(pr.Status.Candidates)).To(Equal(3))

	// wait for reconciliation triggered by status update
	g.Eventually(requests, timeout, interval).Should(Receive(Equal(expectedRequest)))
	// 2 candidates and no decisions
	g.Expect(c.Get(context.TODO(), prKey, pr)).NotTo(HaveOccurred())
	g.Expect(len(pr.Status.Candidates)).To(Equal(2))

	// wait for reconciliation triggered by status update
	g.Eventually(requests, timeout, interval).Should(Receive(Equal(expectedRequest)))
	// 2 candidates and no decisions
	g.Expect(c.Get(context.TODO(), prKey, pr)).NotTo(HaveOccurred())
	g.Expect(len(pr.Status.Candidates)).To(Equal(1))

	// wait for reconciliation triggered by status update
	g.Eventually(requests, timeout, interval).Should(Receive(Equal(expectedRequest)))
	// 2 candidates and one decisions
	g.Expect(c.Get(context.TODO(), prKey, pr)).NotTo(HaveOccurred())
	g.Expect(len(pr.Status.Candidates)).To(Equal(1))
	g.Expect(len(pr.Status.Decisions)).To(Equal(1))

}

func TestScoredAndUnscoredRecommendations(t *testing.T) {
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

	/**
	- add 3 advisors and change the hpr recommendations:
		rhacm(priority, 60), recommendation: cluster3
		cost(priority, 50), recommendation: cluster2
		grc(predicate), recommendation: cluster1, cluster3

	- expected output after hpr reconciliation:
		2 candidates: cluster1(weight 150), cluster3(60)
		1 decision (cluster1)
	**/

	advisor1 := rhacmAdvisor.DeepCopy()
	advisor2 := costAdvisor.DeepCopy()
	advisor3 := grcAdvisor.DeepCopy()

	cl1 := mc1.DeepCopy()
	g.Expect(c.Create(context.TODO(), cl1)).NotTo(HaveOccurred())
	// reload the cluster object
	//g.Expect(c.Get(context.TODO(), mc1Key, cl1)).NotTo(HaveOccurred())

	defer func() {
		if err = c.Delete(context.TODO(), cl1); err != nil {
			klog.Error(err)
			t.Fail()
		}
	}()

	pr := placementRule.DeepCopy()
	var replica int16 = int16(defaultReplicas)
	pr.Spec.Replicas = &replica
	pr.Spec.Advisors = []corev1alpha1.Advisor{
		*advisor1,
		*advisor2,
		*advisor3,
	}
	defer func() {
		if err = c.Delete(context.TODO(), pr); err != nil {
			klog.Error(err)
			t.Fail()
		}
	}()

	g.Expect(c.Create(context.TODO(), pr)).To(Succeed())

	// wait for main reconciliation
	g.Eventually(requests, timeout, interval).Should(Receive(Equal(expectedRequest)))

	// wait for status update reconciliation
	g.Eventually(requests, timeout, interval).Should(Receive(Equal(expectedRequest)))

	// 1 candidates and 0 decisions, as advisors have not sent in any recommendations yet
	g.Expect(c.Get(context.TODO(), prKey, pr)).NotTo(HaveOccurred())
	g.Expect(len(pr.Status.Candidates)).To(Equal(1))
	g.Expect(pr.Status.Candidates[0].Name).To(Equal(mc1.Name))
	g.Expect(len(pr.Status.Decisions)).To(Equal(0))

	cl2 := mc2.DeepCopy()
	g.Expect(c.Create(context.TODO(), cl2)).NotTo(HaveOccurred())

	defer func() {
		if err = c.Delete(context.TODO(), cl2); err != nil {
			klog.Error(err)
			t.Fail()
		}
	}()

	cl3 := mc3.DeepCopy()
	g.Expect(c.Create(context.TODO(), cl3)).NotTo(HaveOccurred())

	defer func() {
		if err = c.Delete(context.TODO(), cl3); err != nil {
			klog.Error(err)
			t.Fail()
		}
	}()

	// cluster3
	rhacmRecommendations := []corev1alpha1.ScoredObjectReference{
		{
			ObjectReference: corev1.ObjectReference{
				Name:       cl3.Name,
				Namespace:  cl3.Namespace,
				APIVersion: cl3.APIVersion,
				Kind:       cl3.Kind,
				UID:        cl3.UID,
			},
		},
	}

	// cluster2
	costRecommendations := []corev1alpha1.ScoredObjectReference{
		{
			ObjectReference: corev1.ObjectReference{
				Name:       cl2.Name,
				Namespace:  cl2.Namespace,
				APIVersion: cl2.APIVersion,
				Kind:       cl2.Kind,
				UID:        cl2.UID,
			},
		},
	}

	// cluster1 and cluster3
	grcRecommendations := []corev1alpha1.ScoredObjectReference{
		{
			ObjectReference: corev1.ObjectReference{
				Name:       cl1.Name,
				Namespace:  cl1.Namespace,
				APIVersion: cl1.APIVersion,
				Kind:       cl1.Kind,
				UID:        cl1.UID,
			},
		},
		{
			ObjectReference: corev1.ObjectReference{
				Name:       cl3.Name,
				Namespace:  cl3.Namespace,
				APIVersion: cl3.APIVersion,
				Kind:       cl3.Kind,
				UID:        cl3.UID,
			},
		},
	}

	pr.Status.Decisions = []corev1.ObjectReference{
		{
			Name:       cl1.Name,
			Namespace:  cl1.Namespace,
			APIVersion: cl1.APIVersion,
			Kind:       cl1.Kind,
			UID:        cl1.UID,
		},
	}
	g.Expect(c.Status().Update(context.TODO(), pr)).NotTo(HaveOccurred())
	// wait for reconciliation triggered by status update
	g.Eventually(requests, timeout, interval).Should(Receive(Equal(expectedRequest)))
	g.Expect(c.Get(context.TODO(), prKey, pr)).NotTo(HaveOccurred())

	pr.Status.Recommendations = map[string]corev1alpha1.Recommendation{
		advisor1.Name: rhacmRecommendations,
		advisor2.Name: costRecommendations,
		advisor3.Name: grcRecommendations,
	}
	g.Expect(c.Status().Update(context.TODO(), pr)).NotTo(HaveOccurred())

	// wait for reconciliation triggered by status update
	g.Eventually(requests, timeout, interval).Should(Receive(Equal(expectedRequest)))

	// wait for reconciliation triggered by status update
	g.Eventually(requests, timeout, interval).Should(Receive(Equal(expectedRequest)))

	// wait for reconciliation triggered by status update
	g.Eventually(requests, timeout, interval).Should(Receive(Equal(expectedRequest)))

	// simulate the advisor reconciliation, as the recommendations have been cleaned up during hpr reconciliation
	g.Expect(c.Get(context.TODO(), prKey, pr)).NotTo(HaveOccurred())
	pr.Status.Recommendations = map[string]corev1alpha1.Recommendation{
		advisor1.Name: rhacmRecommendations,
		advisor2.Name: costRecommendations,
		advisor3.Name: grcRecommendations,
	}
	g.Expect(c.Status().Update(context.TODO(), pr)).NotTo(HaveOccurred())

	// wait for reconciliation triggered by status update
	g.Eventually(requests, timeout, interval).Should(Receive(Equal(expectedRequest)))

	// wait for reconciliation triggered by status update
	g.Eventually(requests, timeout, interval).Should(Receive(Equal(expectedRequest)))

	g.Expect(c.Get(context.TODO(), prKey, pr)).NotTo(HaveOccurred())
	pr.Status.Recommendations = map[string]corev1alpha1.Recommendation{
		advisor1.Name: rhacmRecommendations,
		advisor2.Name: costRecommendations,
		advisor3.Name: grcRecommendations,
	}
	g.Expect(c.Status().Update(context.TODO(), pr)).NotTo(HaveOccurred())

	// wait for reconciliation triggered by status update
	g.Eventually(requests, timeout, interval).Should(Receive(Equal(expectedRequest)))

	g.Expect(c.Get(context.TODO(), prKey, pr)).NotTo(HaveOccurred())

	g.Expect(len(pr.Status.Candidates)).To(Equal(1))
	g.Expect(len(pr.Status.Decisions)).To(Equal(1))
	g.Expect(pr.Status.Decisions[0].Name).To(Equal(cl3.Name))

	/**
	- simulate advisor reconciliation and change hpr recommendations:
			rhacm(priority, 20), recommendation: cluster3
			cost(priority, 140), recommendation: cluster1
			grc(predicate), recommendation: cluster1, cluster 3

		- expected result after hpr reconciliation:
			candidates: cluster1 (weight 140), cluster3 (weight 120)
			decisions: cluster1
	**/

	var newRHACMWeight = int16(20)
	var newCostWeight = int16(140)
	pr.Spec.Advisors[0].Weight = &newRHACMWeight
	pr.Spec.Advisors[1].Weight = &newCostWeight

	g.Expect(c.Update(context.TODO(), pr)).NotTo(HaveOccurred())
	// wait for reconciliation triggered by status update
	g.Eventually(requests, timeout, interval).Should(Receive(Equal(expectedRequest)))
	g.Eventually(requests, timeout, interval).Should(Receive(Equal(expectedRequest)))
	g.Eventually(requests, timeout, interval).Should(Receive(Equal(expectedRequest)))

	g.Expect(c.Get(context.TODO(), prKey, pr)).NotTo(HaveOccurred())

	// simulate advisor reconciliation
	costRecommendations = []corev1alpha1.ScoredObjectReference{
		{
			ObjectReference: corev1.ObjectReference{
				Name:       cl1.Name,
				Namespace:  cl1.Namespace,
				APIVersion: cl1.APIVersion,
				Kind:       cl1.Kind,
				UID:        cl1.UID,
			},
		},
	}
	pr.Status.Recommendations = map[string]corev1alpha1.Recommendation{
		advisor1.Name: rhacmRecommendations,
		advisor2.Name: costRecommendations,
		advisor3.Name: grcRecommendations,
	}

	g.Expect(c.Status().Update(context.TODO(), pr)).NotTo(HaveOccurred())

	// wait for reconciliation triggered by status update
	g.Eventually(requests, timeout, interval).Should(Receive(Equal(expectedRequest)))
	g.Eventually(requests, timeout, interval).Should(Receive(Equal(expectedRequest)))
	g.Expect(c.Get(context.TODO(), prKey, pr)).NotTo(HaveOccurred())

	// simulate the second advisor reconciliation
	pr.Status.Recommendations = map[string]corev1alpha1.Recommendation{
		advisor1.Name: rhacmRecommendations,
		advisor2.Name: costRecommendations,
		advisor3.Name: grcRecommendations,
	}

	g.Expect(c.Status().Update(context.TODO(), pr)).NotTo(HaveOccurred())

	// wait for reconciliation triggered by status update
	g.Eventually(requests, timeout, interval).Should(Receive(Equal(expectedRequest)))
	g.Eventually(requests, timeout, interval).Should(Receive(Equal(expectedRequest)))
	g.Expect(c.Get(context.TODO(), prKey, pr)).NotTo(HaveOccurred())

	// simulate the third advisor reconciliation
	pr.Status.Recommendations = map[string]corev1alpha1.Recommendation{
		advisor1.Name: rhacmRecommendations,
		advisor2.Name: costRecommendations,
		advisor3.Name: grcRecommendations,
	}

	g.Expect(c.Status().Update(context.TODO(), pr)).NotTo(HaveOccurred())

	// wait for reconciliation triggered by status update
	g.Eventually(requests, timeout, interval).Should(Receive(Equal(expectedRequest)))
	g.Eventually(requests, timeout, interval).Should(Receive(Equal(expectedRequest)))
	g.Expect(c.Get(context.TODO(), prKey, pr)).NotTo(HaveOccurred())

	g.Expect(len(pr.Status.Candidates)).To(Equal(1))
	g.Expect(len(pr.Status.Decisions)).To(Equal(1))
	g.Expect(pr.Status.Decisions[0].Name).To(Equal(cl1.Name))

	/**
	- simulate advisor reconciliation and change hpr recommendations:
				rhacm(priority 180), recommendation: cluster3
				cost(priority, weight 140, score 50), recommendation: cluster1
				grc(predicate), recommendation: cluster1, cluster 3

			- expected result after hpr reconciliation:
				candidates: cluster1 (weight 100 + 140 * 50/100 = 170), cluster3 (weight 180)
				decisions: cluster3
	**/

	newRHACMWeight = int16(180)
	newCostWeight = int16(140)
	var newCostScore = int16(50)
	pr.Spec.Advisors[0].Weight = &newRHACMWeight
	pr.Spec.Advisors[1].Weight = &newCostWeight

	g.Expect(c.Update(context.TODO(), pr)).NotTo(HaveOccurred())
	// wait for reconciliation triggered by status update
	g.Eventually(requests, timeout, interval).Should(Receive(Equal(expectedRequest)))
	g.Eventually(requests, timeout, interval).Should(Receive(Equal(expectedRequest)))

	g.Expect(c.Get(context.TODO(), prKey, pr)).NotTo(HaveOccurred())

	// simulate advisor reconciliation
	costRecommendations = []corev1alpha1.ScoredObjectReference{
		{
			ObjectReference: corev1.ObjectReference{
				Name:       cl1.Name,
				Namespace:  cl1.Namespace,
				APIVersion: cl1.APIVersion,
				Kind:       cl1.Kind,
				UID:        cl1.UID,
			},
			Score: &newCostScore,
		},
	}
	pr.Status.Recommendations = map[string]corev1alpha1.Recommendation{
		advisor1.Name: rhacmRecommendations,
		advisor2.Name: costRecommendations,
		advisor3.Name: grcRecommendations,
	}

	g.Expect(c.Status().Update(context.TODO(), pr)).NotTo(HaveOccurred())

	// wait for reconciliation triggered by status update
	g.Eventually(requests, timeout, interval).Should(Receive(Equal(expectedRequest)))
	g.Eventually(requests, timeout, interval).Should(Receive(Equal(expectedRequest)))
	g.Expect(c.Get(context.TODO(), prKey, pr)).NotTo(HaveOccurred())

	// simulate the second advisor reconciliation
	pr.Status.Recommendations = map[string]corev1alpha1.Recommendation{
		advisor1.Name: rhacmRecommendations,
		advisor2.Name: costRecommendations,
		advisor3.Name: grcRecommendations,
	}

	g.Expect(c.Status().Update(context.TODO(), pr)).NotTo(HaveOccurred())

	// wait for reconciliation triggered by status update
	g.Eventually(requests, timeout, interval).Should(Receive(Equal(expectedRequest)))
	g.Eventually(requests, timeout, interval).Should(Receive(Equal(expectedRequest)))
	g.Expect(c.Get(context.TODO(), prKey, pr)).NotTo(HaveOccurred())

	// simulate the third advisor reconciliation
	pr.Status.Recommendations = map[string]corev1alpha1.Recommendation{
		advisor1.Name: rhacmRecommendations,
		advisor2.Name: costRecommendations,
		advisor3.Name: grcRecommendations,
	}

	g.Expect(c.Status().Update(context.TODO(), pr)).NotTo(HaveOccurred())

	// wait for reconciliation triggered by status update
	g.Eventually(requests, timeout, interval).Should(Receive(Equal(expectedRequest)))
	g.Eventually(requests, timeout, interval).Should(Receive(Equal(expectedRequest)))
	g.Expect(c.Get(context.TODO(), prKey, pr)).NotTo(HaveOccurred())

	g.Expect(len(pr.Status.Candidates)).To(Equal(1))
	g.Expect(len(pr.Status.Decisions)).To(Equal(1))
	g.Expect(pr.Status.Decisions[0].Name).To(Equal(cl3.Name))

}
