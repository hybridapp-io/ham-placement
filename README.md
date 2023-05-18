# ham-placement

[![Build](http://prow.purple-chesterfield.com/badge.svg?jobs=build-ham-placement-amd64-postsubmit)](http://prow.purple-chesterfield.com/?job=build-ham-placement-amd64-postsubmit)
[![GoDoc](https://godoc.org/github.com/hybridapp-io/ham-placement?status.svg)](https://godoc.org/github.com/hybridapp-io/ham-placement)
[![Go Report Card](https://goreportcard.com/badge/github.com/hybridapp-io/ham-placement)](https://goreportcard.com/report/github.com/hybridapp-io/ham-placement)
[![Code Coverage](https://codecov.io/gh/hybridapp-io/ham-placement/branch/master/graphs/badge.svg?branch=master)](https://codecov.io/gh/hybridapp-io/ham-placement?branch=master)
[![License](https://img.shields.io/:license-apache-blue.svg)](http://www.apache.org/licenses/LICENSE-2.0.html)
[![Image](https://quay.io/repository/cicdtest/ham-placementrule/status)](https://quay.io/repository/cicdtest/ham-placementrule?tab=tags)

<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->
**Table of Contents**  *generated with [DocToc](https://github.com/thlorenz/doctoc)*

- [ham-placement](#ham-placement)
  - [What is the PlacementRule in Hybrid Application Model](#what-is-the-placementrule-in-hybrid-application-model)
  - [Community, discussion, contribution, and support](#community-discussion-contribution-and-support)
  - [Getting Started](#getting-started)
    - [Prerequisites](#prerequisites)
    - [Quick Start](#quick-start)
      - [Clone PlacementRule Repository](#clone-placementrule-repository)
      - [Build Deployable Operator](#build-deployable-operator)
      - [Install Deployable Operator](#install-deployable-operator)
      - [Play with Examples](#play-with-examples)
      - [Uninstall Deployable Operator](#uninstall-deployable-operator)
    - [Troubleshooting](#troubleshooting)
  - [References](#references)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

PlacementRule Operator of Hybrid Application Model to decide target to deploy

## What is the PlacementRule in Hybrid Application Model

It is a resource to let user define where to deploy their application components.

## Community, discussion, contribution, and support

Check the [CONTRIBUTING Doc](CONTRIBUTING.md) for how to contribute to the repo.

------

## Getting Started

### Prerequisites

- git v2.18+
- Go v1.13.4+
- operator-sdk v0.17.0
- Kubernetes v1.14+
- kubectl v1.14+

Check the [Development Doc](docs/development.md) for how to contribute to the repo.

### Quick Start

#### Clone PlacementRule Repository

```shell
$ mkdir -p "$GOPATH"/src/github.com/hybridapp-io
$ cd "$GOPATH"/src/github.com/hybridapp-io
$ git clone https://github.com/hybridapp-io/ham-placement.git
$ cd "$GOPATH"/src/github.com/hybridapp-io/ham-placement
```

#### Build Deployable Operator

Build the ham-placementrule and push it to a registry.  Modify the example below to reference a container reposistory you have access to.

```shell
$ operator-sdk build quay.io/<user>/ham-placementrule:v0.1.0
$ sed -i 's|REPLACE_IMAGE|quay.io/johndoe/ham-placementrule:v0.1.0|g' deploy/operator.yaml
$ docker push quay.io/johndoe/ham-placementrule:v0.1.0
```

#### Install Deployable Operator

Register the CRD.

```shell
$ kubectl apply -f deploy/crds
```

Setup RBAC and deploy.

```shell
$ kubectl create -f deploy
```

Verify ham-placementrule is up and running.

```shell
$ kubectl get deployment
NAME                        READY   UP-TO-DATE   AVAILABLE   AGE
ham-placementrule   1/1     1            1           2m21s
```

#### Play with Examples

Register cluster CRD and clusters

```shell
% kubectl apply -f hack/test/open-cluster-management.io_managedclusters.crd.yaml
customresourcedefinition.apiextensions.k8s.io/managedclusters.cluster.open-cluster-management.io created
% kubectl apply -f hack/test/crs/clusters.yaml
namespace/raleigh created
managedcluster.cluster.open-cluster-management.io/raleigh created
namespace/toronto created
managedcluster.cluster.open-cluster-management.io/toronto created
namespace/shanghai created
managedcluster.cluster.open-cluster-management.io/shanghai created
% kubectl get clusters --all-namespaces --show-labels
NAMESPACE   NAME      AGE   LABELS
raleigh     raleigh   38s   cloud=IBM,datacenter=raleigh,environment=Dev,name=raleigh,owner=marketing,region=US,vendor=ICP
shanghai    shanghai  38s   cloud=IBM,datacenter=shanghai,environment=Dev,name=shanghai,owner=dev,region=China,vendor=ICP
toronto     toronto   38s   cloud=IBM,datacenter=toronto,environment=Dev,name=toronto,owner=marketing,region=US,vendor=ICP
```

Create the sample board cR.

```shell
% kubectl apply -f examples/board.yaml
placementrule.core.hybridapp.io/board created
% kubectl get placementrule
NAME    AGE
board   21s

 % kubectl describe placementrule
Name:         board
Namespace:    default
API Version:  core.hybridapp.io/v1alpha1
Kind:         PlacementRule
...
Spec:
  Advisors:
    Name:    alphabet
    Weight:  60
    Name:    veto
    Rules:
      Resources:
        Name:       raleigh
        Namespace:  raleigh
    Type:           predicate
    Weight:         50
  Decision Weight:  5
  Replicas:         1
  Target Labels:
    Match Labels:
      Cloud:  IBM
Status:
  Candidates:
    API Version:  cluster.open-cluster-management.io/v1
    Kind:         Cluster
    Name:         shanghai
    Namespace:    shanghai
    UID:          66c83e70-4184-4eed-b593-09abb4e5d7a3
    API Version:  cluster.open-cluster-management.io/v1
    Kind:         Cluster
    Name:         toronto
    Namespace:    toronto
    UID:          f43e6fbe-8b32-4ee6-986c-f87fdbc83f51
  Decisions:
    API Version:  cluster.open-cluster-management.io/v1
    Kind:         Cluster
    Name:         shanghai
    Namespace:    shanghai
    UID:          66c83e70-4184-4eed-b593-09abb4e5d7a3
  Eliminators:
    API Version:        cluster.open-cluster-management.io/v1
    Kind:               Cluster
    Name:               raleigh
    Namespace:          raleigh
    UID:                54777532-d293-44de-b3b5-bfa8ffbaf9a9
  Last Update Time:     2020-06-30T02:36:28Z
  Observed Generation:  1
  Recommendations:
    Alphabet:
      API Version:  cluster.open-cluster-management.io/v1
      Kind:         Cluster
      Name:         shanghai
      Namespace:    shanghai
      UID:          66c83e70-4184-4eed-b593-09abb4e5d7a3
    Veto:
      API Version:  cluster.open-cluster-management.io/v1
      Kind:         Cluster
      Name:         shanghai
      Namespace:    shanghai
      UID:          66c83e70-4184-4eed-b593-09abb4e5d7a3
      API Version:  cluster.open-cluster-management.io/v1
      Kind:         Cluster
      Name:         toronto
      Namespace:    toronto
      UID:          f43e6fbe-8b32-4ee6-986c-f87fdbc83f51
Events:     <none>
```

2 advisors are built-in with placementrule operator: alphabet and veto.

#### Uninstall Deployable Operator

Remove all resources created.

```shell
$ kubectl delete -f deploy
$ kubectl delete -f deploy/crds
```

### Troubleshooting

Please refer to [Troubleshooting documentation](docs/trouble_shooting.md) for further info.

## References
