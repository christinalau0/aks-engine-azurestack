// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT license.

package rotatecerts

import (
	"context"
	"time"

	"github.com/Azure/aks-engine-azurestack/cmd/rotatecerts/internal"
	"github.com/Azure/aks-engine-azurestack/pkg/helpers/ssh"
	"github.com/Azure/aks-engine-azurestack/pkg/kubernetes"
	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
)

const defaultSuccessesNeeded int = 5

type nodesCondition func(*v1.NodeList) bool

// waitForNodesCondition checks that nodesCondition is met for every node in the cluster
func waitForNodesCondition(client internal.KubeClient, condition nodesCondition, successesNeeded int, interval, timeout time.Duration) (*v1.NodeList, error) {
	var nl *v1.NodeList
	var err error
	var successesCount int
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	err = wait.PollUntilContextCancel(ctx, interval, true, func(ctx context.Context) (bool, error) {
		nl, err = client.ListNodes()
		if err != nil {
			return false, err
		}
		if !condition(nl) {
			return false, nil
		}
		successesCount++
		if successesCount < successesNeeded {
			return false, nil
		}
		return true, nil
	})
	return nl, err
}

// WaitForNodesReady returns true if all requiredNodes reached the Ready state
func WaitForNodesReady(client internal.KubeClient, requiredNodes []string, interval, timeout time.Duration) error {
	_, err := waitForNodesCondition(client, allNodesReadyCondition(requiredNodes), defaultSuccessesNeeded, interval, timeout)
	return err
}

func allNodesReadyCondition(requiredNodes []string) nodesCondition {
	return func(nl *v1.NodeList) bool {
		requiredReady := make(map[string]bool)
		for _, name := range requiredNodes {
			requiredReady[name] = false
		}
		for _, nli := range nl.Items {
			_, ok := requiredReady[nli.ObjectMeta.Name]
			if !ok {
				continue
			}
			ready := kubernetes.IsNodeReady(&nli)
			if !ready {
				return false
			}
			requiredReady[nli.ObjectMeta.Name] = ready
		}
		for _, ready := range requiredReady {
			if !ready {
				return false
			}
		}
		return true
	}
}

type podsCondition func(*v1.PodList) error

// waitForPodsCondition checks that podsCondition is met for every pod in the specified namespace
func waitForPodsCondition(client internal.KubeClient, namespace string, condition podsCondition, successesNeeded int, interval, timeout time.Duration) error {
	var listErr, condErr error
	var successesCount int
	var pl *v1.PodList
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	err := wait.PollUntilContextCancel(ctx, interval, true, func(ctx context.Context) (bool, error) {
		pl, listErr = client.ListPods(namespace, metav1.ListOptions{})
		if listErr != nil {
			return false, listErr
		}
		if condErr = condition(pl); condErr != nil {
			return false, nil
		}
		successesCount++
		if successesCount < successesNeeded {
			return false, nil
		}
		return true, nil
	})
	if listErr != nil {
		return errors.Wrapf(listErr, "condition successesCount: %d", successesCount)
	}
	if condErr != nil {
		return errors.Wrapf(condErr, "condition successesCount: %d", successesCount)
	}
	return err
}

// WaitForAllInNamespaceReady returns true if all containers in a given namespace reached the Ready state
func WaitForAllInNamespaceReady(client internal.KubeClient, namespace string, interval, timeout time.Duration, nodes map[string]*ssh.RemoteHost) error {
	if err := waitForDaemonSetCondition(client, namespace, allDaemontSetReplicasUpdatedCondition, defaultSuccessesNeeded, interval, timeout); err != nil {
		return err
	}
	if err := waitForDeploymentCondition(client, namespace, allDeploymentReplicasUpdatedCondition, defaultSuccessesNeeded, interval, timeout); err != nil {
		return err
	}
	return waitForPodsCondition(client, namespace, allListedPodsReadyCondition, defaultSuccessesNeeded, interval, timeout)
}

func allListedPodsReadyCondition(pl *v1.PodList) error {
	podsNotReady := make([]string, 0)
	for _, pli := range pl.Items {
		ready := pli.Status.Phase == v1.PodRunning
		for _, c := range pli.Status.ContainerStatuses {
			ready = ready && c.State.Running != nil && c.Ready
		}
		if !ready {
			podsNotReady = append(podsNotReady, pli.Name)
		}
	}
	if len(podsNotReady) != 0 {
		return errors.Errorf("at least one pod did not reach the Ready state: %s", podsNotReady)
	}
	return nil
}

// WaitForReady returns true if all containers in a given pod list reached the Ready state
func WaitForReady(client internal.KubeClient, namespace string, pods []string, interval, timeout time.Duration, nodes map[string]*ssh.RemoteHost) error {
	waitFor := allExpectedPodsReadyCondition(pods)
	return waitForPodsCondition(client, namespace, waitFor, defaultSuccessesNeeded, interval, timeout)
}

func allExpectedPodsReadyCondition(expectedPods []string) podsCondition {
	return func(pl *v1.PodList) error {
		podReady := make(map[string]bool, len(expectedPods))
		for _, n := range expectedPods {
			podReady[n] = false
		}
		for _, pli := range pl.Items {
			_, ok := podReady[pli.ObjectMeta.Name]
			if !ok {
				continue
			}
			ready := pli.Status.Phase == v1.PodRunning
			for _, c := range pli.Status.ContainerStatuses {
				ready = ready && c.State.Running != nil && c.Ready
			}
			podReady[pli.ObjectMeta.Name] = ready
		}
		podsNotReady := make([]string, 0)
		for pod, ready := range podReady {
			if !ready {
				podsNotReady = append(podsNotReady, pod)
			}
		}
		if len(podsNotReady) != 0 {
			return errors.Errorf("at least one pod did not reach the Ready state: %s", podsNotReady)
		}
		return nil
	}
}

type daemonsetCondition func(*appsv1.DaemonSetList) error

// waitForDaemonSetCondition fetches the ds in a namespace and checks that daemonsetCondition is met for every ds in the cluster
func waitForDaemonSetCondition(client internal.KubeClient, namespace string, condition daemonsetCondition, successesNeeded int, interval, timeout time.Duration) error {
	var listErr, condErr error
	var successesCount int
	var dsl *appsv1.DaemonSetList
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	err := wait.PollUntilContextCancel(ctx, interval, true, func(ctx context.Context) (bool, error) {
		dsl, listErr = client.ListDaemonSets(namespace, metav1.ListOptions{})
		if listErr != nil {
			return false, listErr
		}
		if condErr = condition(dsl); condErr != nil {
			return false, nil
		}
		successesCount++
		if successesCount < successesNeeded {
			return false, nil
		}
		return true, nil
	})
	if listErr != nil {
		return errors.Wrapf(listErr, "condition successesCount: %d", successesCount)
	}
	if condErr != nil {
		return errors.Wrapf(condErr, "condition successesCount: %d", successesCount)
	}
	return err
}

func allDaemontSetReplicasUpdatedCondition(dsl *appsv1.DaemonSetList) error {
	dsNotReady := make([]string, 0)
	for _, dsli := range dsl.Items {
		desired := dsli.Status.DesiredNumberScheduled
		current := dsli.Status.CurrentNumberScheduled
		updated := dsli.Status.UpdatedNumberScheduled
		if desired != current || desired != updated {
			dsNotReady = append(dsNotReady, dsli.Name)
		}
	}
	if len(dsNotReady) != 0 {
		return errors.Errorf("at least one daemonset is still updating replicas: %s", dsNotReady)
	}
	return nil
}

type deploymentCondition func(*appsv1.DeploymentList) error

// waitForDeploymentCondition fetches the deployment in a namespace and checks that deployCondition is met for every deployment in the cluster
func waitForDeploymentCondition(client internal.KubeClient, namespace string, condition deploymentCondition, successesNeeded int, interval, timeout time.Duration) error {
	var listErr, condErr error
	var successesCount int
	var dl *appsv1.DeploymentList
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	err := wait.PollUntilContextCancel(ctx, interval, true, func(ctx context.Context) (bool, error) {
		dl, listErr = client.ListDeployments(namespace, metav1.ListOptions{})
		if listErr != nil {
			return false, listErr
		}
		if condErr = condition(dl); condErr != nil {
			return false, nil
		}
		successesCount++
		if successesCount < successesNeeded {
			return false, nil
		}
		return true, nil
	})
	if listErr != nil {
		return errors.Wrapf(listErr, "condition successesCount: %d", successesCount)
	}
	if condErr != nil {
		return errors.Wrapf(condErr, "condition successesCount: %d", successesCount)
	}
	return err
}

func allDeploymentReplicasUpdatedCondition(dsl *appsv1.DeploymentList) error {
	deployNotReady := make([]string, 0)
	for _, dli := range dsl.Items {
		desired := dli.Status.Replicas
		current := dli.Status.AvailableReplicas
		updated := dli.Status.UpdatedReplicas
		if desired != current || desired != updated {
			deployNotReady = append(deployNotReady, dli.Name)
		}
	}
	if len(deployNotReady) != 0 {
		return errors.Errorf("at least one deployment is still updating replicas: %s", deployNotReady)
	}
	return nil
}

// WaitForVMsRunning checks that all requiredVMs are running
func WaitForVMsRunning(client internal.ARMClient, resourceGroupName string, requiredVMs []string, interval, timeout time.Duration) error {
	var err error
	var successesCount int
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	err = wait.PollUntilContextCancel(ctx, interval, true, func(ctx context.Context) (bool, error) {
		allRunning := true
		for _, vm := range requiredVMs {
			var state string
			state, err = client.GetVirtualMachinePowerState(resourceGroupName, vm)
			if err != nil {
				return false, nil
			}
			running := isVirtualMachineRunning(state)
			if err != nil {
				return false, err
			}
			allRunning = allRunning && running
		}
		if !allRunning {
			return false, nil
		}
		successesCount++
		if successesCount < 1 {
			return false, nil
		}
		return true, nil
	})
	return err
}
