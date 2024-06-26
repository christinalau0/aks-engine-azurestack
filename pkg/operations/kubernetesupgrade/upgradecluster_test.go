// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT license.

package kubernetesupgrade

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/Azure/aks-engine-azurestack/pkg/api"
	"github.com/Azure/aks-engine-azurestack/pkg/api/common"
	"github.com/Azure/aks-engine-azurestack/pkg/armhelpers"
	"github.com/Azure/aks-engine-azurestack/pkg/helpers/to"
	"github.com/Azure/aks-engine-azurestack/pkg/i18n"
	mock "github.com/Azure/aks-engine-azurestack/pkg/kubernetes/mock_kubernetes"
	compute "github.com/Azure/azure-sdk-for-go/profile/p20200901/resourcemanager/compute/armcompute"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	logtest "github.com/sirupsen/logrus/hooks/test"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/wait"
)

const TestAKSEngineVersion = "1.0.0"

type fakeUpgradeWorkflow struct {
	RunUpgradeError error
	ValidateError   error
}

func (workflow fakeUpgradeWorkflow) RunUpgrade() error {
	if workflow.RunUpgradeError != nil {
		return workflow.RunUpgradeError
	}
	return nil
}

func (workflow fakeUpgradeWorkflow) Validate() error {
	if workflow.ValidateError != nil {
		return workflow.ValidateError
	}
	return nil
}

func TestUpgradeCluster(t *testing.T) {
	RegisterFailHandler(Fail)
	_, reporterConfig := GinkgoConfiguration()
	reporterConfig.JUnitReport = "junit.xml"
	RunSpecs(t, "Server Suite", reporterConfig)
}

var _ = Describe("Upgrade Kubernetes cluster tests", Serial, func() {
	initialVersion := common.RationalizeReleaseAndVersion(common.Kubernetes, "", "", false, false, false)
	versionSplit := strings.Split(initialVersion, ".")
	minorVersion, _ := strconv.Atoi(versionSplit[1])
	minorVersionPlusOne := minorVersion + 1
	upgradeVersion := common.RationalizeReleaseAndVersion(common.Kubernetes, versionSplit[0]+"."+strconv.Itoa(minorVersionPlusOne), "", false, false, false)
	AfterEach(func() {
		// delete temp template directory
		os.RemoveAll("_output")
	})

	It("Should succeed when cluster VMs are missing expected tags during upgrade operation", func() {
		cs := api.CreateMockContainerService("testcluster", upgradeVersion, 1, 1, false)
		uc := UpgradeCluster{
			Translator: &i18n.Translator{},
			Logger:     log.NewEntry(log.New()),
		}

		mockClient := armhelpers.MockAKSEngineClient{}
		mockClient.FailListVirtualMachinesTags = true
		uc.Client = &mockClient

		uc.ClusterTopology = ClusterTopology{}
		uc.SubscriptionID = "DEC923E3-1EF1-4745-9516-37906D56DEC4"
		uc.ResourceGroup = "TestRg"
		uc.DataModel = cs
		uc.NameSuffix = "12345678"
		uc.AgentPoolsToUpgrade = map[string]bool{"agentpool1": true}

		err := uc.UpgradeCluster(&mockClient, "kubeConfig", TestAKSEngineVersion)
		Expect(err).NotTo(HaveOccurred())
		Expect(uc.ClusterTopology.AgentPools).NotTo(BeEmpty())

		// Clean up
		os.RemoveAll("./translations")
	})

	It("Should return error message when failing to list VMs during upgrade operation", func() {
		cs := api.CreateMockContainerService("testcluster", upgradeVersion, 1, 1, false)
		uc := UpgradeCluster{
			Translator: &i18n.Translator{},
			Logger:     log.NewEntry(log.New()),
		}

		mockClient := armhelpers.MockAKSEngineClient{}
		mockClient.FailListVirtualMachines = true
		uc.Client = &mockClient

		uc.ClusterTopology = ClusterTopology{}
		uc.SubscriptionID = "DEC923E3-1EF1-4745-9516-37906D56DEC4"
		uc.ResourceGroup = "TestRg"
		uc.DataModel = cs
		uc.NameSuffix = "12345678"
		uc.AgentPoolsToUpgrade = map[string]bool{"agentpool1": true}

		err := uc.UpgradeCluster(&mockClient, "kubeConfig", TestAKSEngineVersion)
		Expect(err).To(HaveOccurred())
		Expect(err).To(MatchError("Error while querying ARM for resources: ListVirtualMachines failed"))

		// Clean up
		os.RemoveAll("./translations")
	})

	It("Should return error message when failing to delete VMs during upgrade operation", func() {
		cs := api.CreateMockContainerService("testcluster", upgradeVersion, 1, 1, false)
		uc := UpgradeCluster{
			Translator: &i18n.Translator{},
			Logger:     log.NewEntry(log.New()),
		}

		mockClient := armhelpers.MockAKSEngineClient{}
		mockClient.FailDeleteVirtualMachine = true
		uc.Client = &mockClient

		uc.ClusterTopology = ClusterTopology{}
		uc.SubscriptionID = "DEC923E3-1EF1-4745-9516-37906D56DEC4"
		uc.ResourceGroup = "TestRg"
		uc.DataModel = cs
		uc.NameSuffix = "12345678"
		uc.AgentPoolsToUpgrade = map[string]bool{"agentpool1": true}

		err := uc.UpgradeCluster(&mockClient, "kubeConfig", TestAKSEngineVersion)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(Equal("DeleteVirtualMachine failed"))
	})

	It("Should return error message when failing to deploy template during upgrade operation", func() {
		cs := api.CreateMockContainerService("testcluster", upgradeVersion, 1, 1, false)
		uc := UpgradeCluster{
			Translator: &i18n.Translator{},
			Logger:     log.NewEntry(log.New()),
		}

		mockClient := armhelpers.MockAKSEngineClient{}
		mockClient.FailDeployTemplate = true
		uc.Client = &mockClient

		uc.ClusterTopology = ClusterTopology{}
		uc.SubscriptionID = "DEC923E3-1EF1-4745-9516-37906D56DEC4"
		uc.ResourceGroup = "TestRg"
		uc.DataModel = cs
		uc.NameSuffix = "12345678"
		uc.AgentPoolsToUpgrade = map[string]bool{"agentpool1": true}

		err := uc.UpgradeCluster(&mockClient, "kubeConfig", TestAKSEngineVersion)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(Equal("DeployTemplate failed"))
	})

	It("Should return error message when failing to get a virtual machine during upgrade operation", func() {
		cs := api.CreateMockContainerService("testcluster", upgradeVersion, 1, 6, false)
		uc := UpgradeCluster{
			Translator: &i18n.Translator{},
			Logger:     log.NewEntry(log.New()),
		}

		mockClient := armhelpers.MockAKSEngineClient{}
		mockClient.FailGetVirtualMachine = true
		uc.Client = &mockClient

		uc.ClusterTopology = ClusterTopology{}
		uc.SubscriptionID = "DEC923E3-1EF1-4745-9516-37906D56DEC4"
		uc.ResourceGroup = "TestRg"
		uc.DataModel = cs
		uc.NameSuffix = "12345678"
		uc.AgentPoolsToUpgrade = map[string]bool{"agentpool1": true}

		err := uc.UpgradeCluster(&mockClient, "kubeConfig", TestAKSEngineVersion)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(Equal("GetVirtualMachine failed"))
	})

	It("Should return error message when failing to delete network interface during upgrade operation", func() {
		cs := api.CreateMockContainerService("testcluster", upgradeVersion, 3, 2, false)
		uc := UpgradeCluster{
			Translator: &i18n.Translator{},
			Logger:     log.NewEntry(log.New()),
		}

		mockClient := armhelpers.MockAKSEngineClient{}
		mockClient.FailDeleteNetworkInterface = true

		uc.Client = &mockClient
		uc.ClusterTopology = ClusterTopology{}
		uc.SubscriptionID = "DEC923E3-1EF1-4745-9516-37906D56DEC4"
		uc.ResourceGroup = "TestRg"
		uc.DataModel = cs
		uc.NameSuffix = "12345678"
		uc.AgentPoolsToUpgrade = map[string]bool{"agentpool1": true}

		err := uc.UpgradeCluster(&mockClient, "kubeConfig", TestAKSEngineVersion)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(Equal("DeleteNetworkInterface failed"))
	})

	Context("When upgrading a cluster with AvailibilitySets VMs", func() {
		var (
			cs               *api.ContainerService
			uc               UpgradeCluster
			mockClient       armhelpers.MockAKSEngineClient
			versionMapBackup map[string]bool
		)

		AfterEach(func() {
			common.AllKubernetesSupportedVersions = versionMapBackup
		})

		BeforeEach(func() {
			versionMapBackup = common.AllKubernetesSupportedVersions
			mockClient = armhelpers.MockAKSEngineClient{}
			cs = api.CreateMockContainerService("testcluster", "1.9.10", 3, 3, false)
			uc = UpgradeCluster{
				Translator: &i18n.Translator{},
				Logger:     log.NewEntry(log.New()),
			}

			uc.Client = &mockClient
			uc.ClusterTopology = ClusterTopology{}
			uc.SubscriptionID = "DEC923E3-1EF1-4745-9516-37906D56DEC4"
			uc.ResourceGroup = "TestRg"
			uc.DataModel = cs
			uc.NameSuffix = "12345678"
			uc.UpgradeWorkFlow = fakeUpgradeWorkflow{}
		})
		It("Should skip VMs that are already on desired version", func() {
			vm := mockClient.MakeFakeVirtualMachine("k8s-agentpool1-12345678-0", "Kubernetes:1.9.10")
			mockClient.FakeListVirtualMachineResult = func() []*compute.VirtualMachine {
				return []*compute.VirtualMachine{&vm}
			}
			uc.AgentPoolsToUpgrade = map[string]bool{"agentpool1": true}
			uc.Force = false

			err := uc.UpgradeCluster(&mockClient, "kubeConfig", TestAKSEngineVersion)
			Expect(err).NotTo(HaveOccurred())
			Expect(*uc.AgentPools["agentpool1"].AgentVMs).To(HaveLen(0))
		})
		It("Should fail when desired version target is not supported", func() {
			desiredVersion := "1.9.10"
			common.AllKubernetesSupportedVersions = map[string]bool{
				"1.9.7":        true,
				desiredVersion: false,
			}
			vm := mockClient.MakeFakeVirtualMachine("k8s-agentpool1-12345678-0", "Kubernetes:1.9.7")
			mockClient.FakeListVirtualMachineResult = func() []*compute.VirtualMachine {
				return []*compute.VirtualMachine{&vm}
			}
			uc.AgentPoolsToUpgrade = map[string]bool{"agentpool1": true}
			uc.Force = false
			uc.DataModel.Properties.OrchestratorProfile.OrchestratorVersion = desiredVersion
			err := uc.UpgradeCluster(&mockClient, "kubeConfig", TestAKSEngineVersion)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("1.9.7 cannot be upgraded to 1.9.10"))
		})
		It("Should not fail when desired version target is not supported and force true", func() {
			desiredVersion := "1.9.10"
			common.AllKubernetesSupportedVersions = map[string]bool{
				"1.9.7":        true,
				desiredVersion: false,
			}
			vm := mockClient.MakeFakeVirtualMachine("k8s-agentpool1-12345678-0", "Kubernetes:1.9.7")
			mockClient.FakeListVirtualMachineResult = func() []*compute.VirtualMachine {
				return []*compute.VirtualMachine{&vm}
			}
			uc.AgentPoolsToUpgrade = map[string]bool{"agentpool1": true}
			uc.Force = true
			uc.DataModel.Properties.OrchestratorProfile.OrchestratorVersion = desiredVersion
			err := uc.UpgradeCluster(&mockClient, "kubeConfig", TestAKSEngineVersion)
			Expect(err).NotTo(HaveOccurred())
			Expect(*uc.AgentPools["agentpool1"].AgentVMs).To(HaveLen(1))

		})
		It("Should not skip VMs that are already on desired version when Force true", func() {
			vm := mockClient.MakeFakeVirtualMachine("k8s-agentpool1-12345678-0", "Kubernetes:1.9.10")
			mockClient.FakeListVirtualMachineResult = func() []*compute.VirtualMachine {
				return []*compute.VirtualMachine{&vm}
			}
			uc.AgentPoolsToUpgrade = map[string]bool{"agentpool1": true}
			uc.Force = true

			err := uc.UpgradeCluster(&mockClient, "kubeConfig", TestAKSEngineVersion)
			Expect(err).NotTo(HaveOccurred())
			Expect(*uc.AgentPools["agentpool1"].AgentVMs).To(HaveLen(1))
		})
		It("Should skip master VMS that are already on desired version", func() {
			vm := mockClient.MakeFakeVirtualMachine(fmt.Sprintf("%s-12345678-0", common.LegacyControlPlaneVMPrefix), "Kubernetes:1.9.10")
			mockClient.FakeListVirtualMachineResult = func() []*compute.VirtualMachine {
				return []*compute.VirtualMachine{&vm}
			}
			uc.Force = false

			err := uc.UpgradeCluster(&mockClient, "kubeConfig", TestAKSEngineVersion)
			Expect(err).NotTo(HaveOccurred())
			Expect(*uc.MasterVMs).To(HaveLen(0))
			Expect(*uc.UpgradedMasterVMs).To(HaveLen(1))

		})
		It("Should not skip master VMS that are already on desired version when Force is true", func() {
			vm := mockClient.MakeFakeVirtualMachine(fmt.Sprintf("%s-12345678-0", common.LegacyControlPlaneVMPrefix), "Kubernetes:1.9.10")
			mockClient.FakeListVirtualMachineResult = func() []*compute.VirtualMachine {
				return []*compute.VirtualMachine{&vm}
			}
			uc.Force = true

			err := uc.UpgradeCluster(&mockClient, "kubeConfig", TestAKSEngineVersion)
			Expect(err).NotTo(HaveOccurred())
			Expect(*uc.MasterVMs).To(HaveLen(1))
			Expect(*uc.UpgradedMasterVMs).To(HaveLen(0))
		})
		It("Should leave platform fault domain count nil", func() {
			cs := api.CreateMockContainerService("testcluster", "", 3, 2, false)
			cs.Properties.OrchestratorProfile.KubernetesConfig = &api.KubernetesConfig{}
			cs.Properties.OrchestratorProfile.KubernetesConfig.UseManagedIdentity = to.BoolPtr(true)
			cs.Properties.MasterProfile.AvailabilityProfile = "AvailabilitySet"
			uc := UpgradeCluster{
				Translator: &i18n.Translator{},
				Logger:     log.NewEntry(log.New()),
			}

			mockClient := armhelpers.MockAKSEngineClient{}
			uc.Client = &mockClient

			uc.ClusterTopology = ClusterTopology{}
			uc.SubscriptionID = "DEC923E3-1EF1-4745-9516-37906D56DEC4"
			uc.ResourceGroup = "TestRg"
			uc.DataModel = cs
			uc.NameSuffix = "12345678"
			uc.AgentPoolsToUpgrade = map[string]bool{"agentpool1": true}

			err := uc.UpgradeCluster(&mockClient, "kubeConfig", TestAKSEngineVersion)
			Expect(err).To(BeNil())
			Expect(cs.Properties.MasterProfile.PlatformFaultDomainCount).To(BeNil())
			for _, pool := range cs.Properties.AgentPoolProfiles {
				Expect(pool.PlatformFaultDomainCount).To(BeNil())
				Expect(pool.ProximityPlacementGroupID).To(BeEmpty())
			}
		})
	})

	It("Should not fail if no managed identity is returned by azure during upgrade operation", func() {
		cs := api.CreateMockContainerService("testcluster", "", 3, 2, false)
		cs.Properties.OrchestratorProfile.KubernetesConfig = &api.KubernetesConfig{}
		cs.Properties.OrchestratorProfile.KubernetesConfig.UseManagedIdentity = to.BoolPtr(true)
		uc := UpgradeCluster{
			Translator: &i18n.Translator{},
			Logger:     log.NewEntry(log.New()),
		}

		mockClient := armhelpers.MockAKSEngineClient{}
		uc.Client = &mockClient

		uc.ClusterTopology = ClusterTopology{}
		uc.SubscriptionID = "DEC923E3-1EF1-4745-9516-37906D56DEC4"
		uc.ResourceGroup = "TestRg"
		uc.DataModel = cs
		uc.NameSuffix = "12345678"
		uc.AgentPoolsToUpgrade = map[string]bool{"agentpool1": true}

		err := uc.UpgradeCluster(&mockClient, "kubeConfig", TestAKSEngineVersion)
		Expect(err).NotTo(HaveOccurred())
	})

	It("Should not fail if a Kubernetes client cannot be created", func() {
		cs := api.CreateMockContainerService("testcluster", "", 3, 2, false)
		uc := UpgradeCluster{
			Translator: &i18n.Translator{},
			Logger:     log.NewEntry(log.New()),
			Force:      true,
		}

		mockClient := armhelpers.MockAKSEngineClient{
			FailGetKubernetesClient: true,
		}
		uc.Client = &mockClient
		uc.DataModel = cs

		logger, hook := logtest.NewNullLogger()
		uc.Logger.Logger = logger
		defer hook.Reset()
		err := uc.UpgradeCluster(&mockClient, "kubeConfig", TestAKSEngineVersion)
		Expect(err).NotTo(HaveOccurred())
		// check log messages to see that we logged the failure
		messages := []string{
			"failed to get a kubernetes client",
		}
		for _, m := range messages {
			found := false
			for _, entry := range hook.Entries {
				if strings.Contains(strings.ToLower(entry.Message), m) {
					found = true
					break
				}
			}
			Expect(found).To(BeTrue())
		}
	})

	It("Should fail if cluster-autoscaler cannot be paused unless --force is specified", func() {
		cs := api.CreateMockContainerService("testcluster", "", 3, 2, false)
		enabled := true
		addon := api.KubernetesAddon{
			Name:    "cluster-autoscaler",
			Enabled: &enabled,
		}
		cs.Properties.OrchestratorProfile.KubernetesConfig = &api.KubernetesConfig{
			Addons: []api.KubernetesAddon{
				addon,
			},
		}

		uc := UpgradeCluster{
			Translator: &i18n.Translator{},
			Logger:     log.NewEntry(log.New()),
		}

		mockClient := armhelpers.MockAKSEngineClient{}
		mockK8sClient := armhelpers.MockKubernetesClient{
			FailGetDeploymentCount: 10,
		}
		mockClient.MockKubernetesClient = &mockK8sClient
		uc.Client = &mockClient
		uc.DataModel = cs

		err := uc.UpgradeCluster(&mockClient, "kubeConfig", TestAKSEngineVersion)
		Expect(err).To(MatchError("GetDeployment failed"))

		logger, hook := logtest.NewNullLogger()
		uc.Logger.Logger = logger
		defer hook.Reset()
		uc.Force = true
		mockK8sClient.FailUpdateDeploymentCount = 10
		err = uc.UpgradeCluster(&mockClient, "kubeConfig", TestAKSEngineVersion)
		Expect(err).NotTo(HaveOccurred())
		// check log messages to see that we logged the failure
		messages := []string{
			"failed to pause cluster-autoscaler",
		}
		for _, m := range messages {
			found := false
			for _, entry := range hook.Entries {
				if strings.Contains(strings.ToLower(entry.Message), m) {
					found = true
					break
				}
			}
			Expect(found).To(BeTrue())
		}
	})

	It("Should pause cluster-autoscaler during upgrade operation", func() {
		cs := api.CreateMockContainerService("testcluster", "", 3, 2, false)
		enabled := true
		addon := api.KubernetesAddon{
			Name:    "cluster-autoscaler",
			Enabled: &enabled,
		}
		cs.Properties.OrchestratorProfile.KubernetesConfig = &api.KubernetesConfig{
			Addons: []api.KubernetesAddon{
				addon,
			},
		}

		uc := UpgradeCluster{
			Translator: &i18n.Translator{},
			Logger:     log.NewEntry(log.New()),
		}

		mockClient := armhelpers.MockAKSEngineClient{}
		uc.Client = &mockClient

		uc.DataModel = cs

		logger, hook := logtest.NewNullLogger()
		uc.Logger.Logger = logger
		defer hook.Reset()
		err := uc.UpgradeCluster(&mockClient, "kubeConfig", TestAKSEngineVersion)
		Expect(err).NotTo(HaveOccurred())
		// check log messages to see that we paused the cluster-autoscaler
		messages := []string{
			"pausing cluster autoscaler",
			"resuming cluster autoscaler",
		}
		for _, m := range messages {
			found := false
			for _, entry := range hook.Entries {
				if strings.Contains(strings.ToLower(entry.Message), m) {
					found = true
					break
				}
			}
			Expect(found).To(BeTrue())
		}
	})

	It("Should not pause cluster-autoscaler if only control plane is upgraded", func() {
		cs := api.CreateMockContainerService("testcluster", "", 3, 2, false)
		enabled := true
		addon := api.KubernetesAddon{
			Name:    "cluster-autoscaler",
			Enabled: &enabled,
		}
		cs.Properties.OrchestratorProfile.KubernetesConfig = &api.KubernetesConfig{
			Addons: []api.KubernetesAddon{
				addon,
			},
		}

		uc := UpgradeCluster{
			Translator: &i18n.Translator{},
			Logger:     log.NewEntry(log.New()),
		}

		mockClient := armhelpers.MockAKSEngineClient{}
		uc.Client = &mockClient
		uc.ControlPlaneOnly = true
		uc.DataModel = cs

		logger, hook := logtest.NewNullLogger()
		uc.Logger.Logger = logger
		defer hook.Reset()
		err := uc.UpgradeCluster(&mockClient, "kubeConfig", TestAKSEngineVersion)
		Expect(err).NotTo(HaveOccurred())
		// messages we do not expect to see
		messages := []string{
			"pausing cluster autoscaler",
			"resuming cluster autoscaler",
		}
		for _, m := range messages {
			found := false
			for _, entry := range hook.Entries {
				if strings.Contains(strings.ToLower(entry.Message), m) {
					found = true
					break
				}
			}
			Expect(found).To(BeFalse())
		}
	})

	It("Tests SetClusterAutoscalerReplicaCount", func() {
		uc := UpgradeCluster{
			Translator: &i18n.Translator{},
			Logger:     log.NewEntry(log.New()),
		}

		// test with nil KubernetesClient
		count, err := uc.SetClusterAutoscalerReplicaCount(nil, 10)
		Expect(count).To(Equal(int32(0)))
		Expect(err).To(MatchError("no kubernetes client"))

		// test happy path
		mockClient := armhelpers.MockKubernetesClient{}
		count, err = uc.SetClusterAutoscalerReplicaCount(&mockClient, 10)
		Expect(count).To(Equal(int32(1)))
		Expect(err).NotTo(HaveOccurred())

		// test retrying with some KubernetesClient errors
		mockClient = armhelpers.MockKubernetesClient{
			FailGetDeploymentCount:    1,
			FailUpdateDeploymentCount: 3,
		}
		logger, hook := logtest.NewNullLogger()
		uc.Logger.Logger = logger
		defer hook.Reset()
		count, err = uc.SetClusterAutoscalerReplicaCount(&mockClient, 10)
		Expect(count).To(Equal(int32(1)))
		Expect(err).NotTo(HaveOccurred())
		// check log messages to see that we retried after errors
		messages := []string{
			"failed to update cluster-autoscaler",
			"retry updating cluster-autoscaler",
		}
		for _, m := range messages {
			found := false
			for _, entry := range hook.Entries {
				if strings.Contains(strings.ToLower(entry.Message), m) {
					found = true
					break
				}
			}
			Expect(found).To(BeTrue())
		}
	})

	It("Tests CopyCustomPropertiesToNewNode", func() {
		u := &Upgrader{}

		mockClient := &armhelpers.MockAKSEngineClient{MockKubernetesClient: &armhelpers.MockKubernetesClient{}}
		mockClient.MockKubernetesClient.FailGetNode = true

		u.Init(&i18n.Translator{}, log.NewEntry(log.New()), ClusterTopology{}, nil, "", nil, nil, TestAKSEngineVersion, false)
		err := u.copyCustomPropertiesToNewNode(mockClient.MockKubernetesClient, "oldNodeName", "newNodeName")
		Expect(err).To(HaveOccurred())

		oldNode := &v1.Node{}
		oldNode.Annotations = map[string]string{}
		oldNode.Annotations["ann1"] = "val1"
		oldNode.Annotations["ann2"] = "val2"
		oldNode.Annotations["customAnn"] = "customVal"

		oldNode.Labels = map[string]string{}
		oldNode.Labels["label1"] = "val1"
		oldNode.Labels["label2"] = "val2"
		oldNode.Labels["customLabel"] = "customVal"

		oldNode.Spec.Taints = []v1.Taint{
			{
				Key:    "key1",
				Value:  "val1",
				Effect: "NoSchedule",
			},
			{
				Key:    "key2",
				Value:  "val2",
				Effect: "NoSchedule",
			},
		}

		newNode := &v1.Node{}
		newNode.Annotations = map[string]string{}
		newNode.Annotations["ann1"] = "newval1"
		newNode.Annotations["ann2"] = "newval2"

		newNode.Labels = map[string]string{}
		newNode.Labels["label1"] = "newval1"
		newNode.Labels["label2"] = "newval2"

		mockClient.MockKubernetesClient.UpdateNodeFunc = func(node *v1.Node) (*v1.Node, error) {
			return node, nil
		}

		err = u.copyCustomNodeProperties(mockClient.MockKubernetesClient, "oldnode", oldNode, "newnode", newNode)
		Expect(err).NotTo(HaveOccurred())
		Expect(len(newNode.Annotations)).To(Equal(3))
		Expect(newNode.Annotations["ann1"]).To(Equal("newval1"))
		Expect(newNode.Annotations["ann2"]).To(Equal("newval2"))
		Expect(newNode.Annotations["customAnn"]).To(Equal("customVal"))

		Expect(len(newNode.Labels)).To(Equal(3))
		Expect(newNode.Labels["label1"]).To(Equal("newval1"))
		Expect(newNode.Labels["label2"]).To(Equal("newval2"))
		Expect(newNode.Labels["customLabel"]).To(Equal("customVal"))

		Expect(len(newNode.Spec.Taints)).To(Equal(2))
	})
})

func TestCheckControlPlaneNodesStatus(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	statusByName := func(name string) v1.NodeCondition {
		if strings.HasPrefix(name, "ok") {
			return v1.NodeCondition{
				Type:   v1.NodeReady,
				Status: v1.ConditionTrue,
			}
		}
		return v1.NodeCondition{
			Type:   v1.NodeReady,
			Status: v1.ConditionFalse,
		}
	}
	nodeList := func(names ...string) *v1.NodeList {
		nodes := make([]v1.Node, 0)
		for _, name := range names {
			node := v1.Node{}
			node.Name = name
			node.Status.Conditions = []v1.NodeCondition{statusByName(name)}
			nodes = append(nodes, node)
		}
		return &v1.NodeList{Items: nodes}
	}
	upgradedVMs := func(names ...string) *[]*compute.VirtualMachine {
		vms := make([]*compute.VirtualMachine, 0)
		for _, name := range names {
			n := name
			vm := &compute.VirtualMachine{}
			vm.Name = &n
			vms = append(vms, vm)
		}
		return &vms
	}
	upgradedNotReadyStream := func(nodes []string) <-chan []string {
		stream := make(chan []string)
		go func() {
			defer close(stream)
			time.Sleep(1 * time.Second)
			stream <- nodes
		}()
		return stream
	}
	errAPIGeneric := errors.New("error")

	t.Run("cannot fetch node status", func(t *testing.T) {
		allnodes := []string{}
		upgradedNodes := []string{}
		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()
		mock := mock.NewMockClient(mockCtrl)
		mock.EXPECT().ListNodesByOptions(gomock.Any()).Return(nodeList(allnodes...), errAPIGeneric).Times(1)

		uc := &UpgradeCluster{Logger: log.NewEntry(log.New())}
		uc.UpgradedMasterVMs = upgradedVMs(upgradedNodes...)
		res, err := uc.getUpgradedNotReady(mock, upgradedNodes)
		g.Expect(res).To(BeNil())
		g.Expect(err).To(HaveOccurred())
	})

	t.Run("timeout, use last value", func(t *testing.T) {
		upgradedNodes := []string{"nok1"}

		uc := &UpgradeCluster{Logger: log.NewEntry(log.New())}
		uc.UpgradedMasterVMs = upgradedVMs(upgradedNodes...)

		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
		defer cancel()
		err := uc.checkControlPlaneNodesStatus(ctx, upgradedNotReadyStream(upgradedNodes))
		g.Expect(err).NotTo(HaveOccurred())
	})

	t.Run("all nodes ready, no node upgraded", func(t *testing.T) {
		allnodes := []string{"ok1", "ok2", "ok3", "ok4", "ok5"}
		upgradedNodes := []string{}
		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()
		mock := mock.NewMockClient(mockCtrl)
		mock.EXPECT().ListNodesByOptions(gomock.Any()).Return(nodeList(allnodes...), nil).Times(1)

		uc := &UpgradeCluster{Logger: log.NewEntry(log.New())}
		uc.UpgradedMasterVMs = upgradedVMs(upgradedNodes...)
		res, err := uc.getUpgradedNotReady(mock, upgradedNodes)
		g.Expect(res).To(BeEmpty())
		g.Expect(err).NotTo(HaveOccurred())

		err = uc.checkControlPlaneNodesStatus(context.Background(), upgradedNotReadyStream(res))
		g.Expect(err).NotTo(HaveOccurred())
	})

	t.Run("upgraded nodes ready, not upgraded nodes not ready", func(t *testing.T) {
		allnodes := []string{"ok1", "ok2", "nok3", "nok4", "nok5"}
		upgradedNodes := []string{"ok1", "ok2"}
		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()
		mock := mock.NewMockClient(mockCtrl)
		mock.EXPECT().ListNodesByOptions(gomock.Any()).Return(nodeList(allnodes...), nil).Times(1)

		uc := &UpgradeCluster{Logger: log.NewEntry(log.New())}
		uc.UpgradedMasterVMs = upgradedVMs(upgradedNodes...)
		res, err := uc.getUpgradedNotReady(mock, upgradedNodes)
		g.Expect(res).To(BeEmpty())
		g.Expect(err).NotTo(HaveOccurred())

		err = uc.checkControlPlaneNodesStatus(context.Background(), upgradedNotReadyStream(res))
		g.Expect(err).NotTo(HaveOccurred())
	})

	t.Run("1 upgraded node not ready", func(t *testing.T) {
		allnodes := []string{"nok1", "ok2", "ok3", "ok4", "ok5"}
		upgradedNodes := []string{"nok1"}
		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()
		mock := mock.NewMockClient(mockCtrl)
		mock.EXPECT().ListNodesByOptions(gomock.Any()).Return(nodeList(allnodes...), nil).Times(1)

		uc := &UpgradeCluster{Logger: log.NewEntry(log.New())}
		uc.UpgradedMasterVMs = upgradedVMs(upgradedNodes...)
		res, err := uc.getUpgradedNotReady(mock, upgradedNodes)
		g.Expect(res).To(Equal([]string{"nok1"}))
		g.Expect(err).NotTo(HaveOccurred())

		err = uc.checkControlPlaneNodesStatus(context.Background(), upgradedNotReadyStream(res))
		g.Expect(err).NotTo(HaveOccurred())
	})

	t.Run("more than 1 upgraded node not ready, return error", func(t *testing.T) {
		allnodes := []string{"nok1", "nok2", "ok3", "ok4", "ok5"}
		upgradedNodes := []string{"nok1", "nok2"}
		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()
		mock := mock.NewMockClient(mockCtrl)
		mock.EXPECT().ListNodesByOptions(gomock.Any()).Return(nodeList(allnodes...), nil).Times(1)

		uc := &UpgradeCluster{Logger: log.NewEntry(log.New())}
		uc.UpgradedMasterVMs = upgradedVMs(upgradedNodes...)
		res, err := uc.getUpgradedNotReady(mock, upgradedNodes)
		g.Expect(res).To(Equal([]string{"nok1", "nok2"}))
		g.Expect(err).NotTo(HaveOccurred())

		err = uc.checkControlPlaneNodesStatus(context.Background(), upgradedNotReadyStream(res))
		g.Expect(err).To(HaveOccurred())
	})

	t.Run("more than 1 alternated upgraded node not ready, return error", func(t *testing.T) {
		allnodes := []string{"nok1", "ok2", "nok3", "ok4", "ok5"}
		upgradedNodes := []string{"nok1", "ok2", "nok3"}
		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()
		mock := mock.NewMockClient(mockCtrl)
		mock.EXPECT().ListNodesByOptions(gomock.Any()).Return(nodeList(allnodes...), nil).Times(1)

		uc := &UpgradeCluster{Logger: log.NewEntry(log.New())}
		uc.UpgradedMasterVMs = upgradedVMs(upgradedNodes...)
		res, err := uc.getUpgradedNotReady(mock, upgradedNodes)
		g.Expect(res).To(Equal([]string{"nok1", "nok3"}))
		g.Expect(err).NotTo(HaveOccurred())

		err = uc.checkControlPlaneNodesStatus(context.Background(), upgradedNotReadyStream(res))
		g.Expect(err).To(HaveOccurred())
	})

	t.Run("all upgraded nodes not ready", func(t *testing.T) {
		allnodes := []string{"nok1", "nok2", "nok3", "nok4", "nok5"}
		upgradedNodes := []string{"nok1", "nok2", "nok3", "nok4", "nok5"}
		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()
		mock := mock.NewMockClient(mockCtrl)
		mock.EXPECT().ListNodesByOptions(gomock.Any()).Return(nodeList(allnodes...), nil).Times(1)

		uc := &UpgradeCluster{Logger: log.NewEntry(log.New())}
		uc.UpgradedMasterVMs = upgradedVMs(upgradedNodes...)
		res, err := uc.getUpgradedNotReady(mock, upgradedNodes)
		g.Expect(res).To(Equal([]string{"nok1", "nok2", "nok3", "nok4", "nok5"}))
		g.Expect(err).NotTo(HaveOccurred())

		err = uc.checkControlPlaneNodesStatus(context.Background(), upgradedNotReadyStream(res))
		g.Expect(err).To(HaveOccurred())
	})

	t.Run("retry client.ListNodes after API error", func(t *testing.T) {
		allnodes := []string{"ok1", "ok2", "ok3", "ok4", "ok5"}
		upgradedNodes := []string{}
		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()
		mock := mock.NewMockClient(mockCtrl)
		gomock.InOrder(
			mock.EXPECT().ListNodesByOptions(gomock.Any()).Return(nodeList([]string{}...), errAPIGeneric),
			mock.EXPECT().ListNodesByOptions(gomock.Any()).Return(nodeList(allnodes...), nil),
		)

		uc := &UpgradeCluster{Logger: log.NewEntry(log.New())}
		uc.UpgradedMasterVMs = upgradedVMs(upgradedNodes...)
		for range uc.upgradedNotReadyStream(mock, wait.Backoff{Steps: 2, Duration: 500 * time.Millisecond}) {
		}
	})

	t.Run("retry client.ListNodes until backoff", func(t *testing.T) {
		allnodes := []string{"nok1", "nok2", "ok3", "ok4", "ok5"}
		upgradedNodes := []string{"nok1", "nok2"}
		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()
		mock := mock.NewMockClient(mockCtrl)
		gomock.InOrder(
			mock.EXPECT().ListNodesByOptions(gomock.Any()).Return(nodeList(allnodes...), nil),
			mock.EXPECT().ListNodesByOptions(gomock.Any()).Return(nodeList(allnodes...), nil),
		)

		uc := &UpgradeCluster{Logger: log.NewEntry(log.New())}
		uc.UpgradedMasterVMs = upgradedVMs(upgradedNodes...)
		for range uc.upgradedNotReadyStream(mock, wait.Backoff{Steps: 2, Duration: 500 * time.Millisecond}) {
		}
	})

	t.Run("do not retry if no NotReady", func(t *testing.T) {
		allnodes := []string{"ok1", "ok2", "ok3", "ok4", "ok5"}
		upgradedNodes := []string{}
		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()
		mock := mock.NewMockClient(mockCtrl)
		gomock.InOrder(
			mock.EXPECT().ListNodesByOptions(gomock.Any()).Return(nodeList(allnodes...), nil),
		)

		uc := &UpgradeCluster{Logger: log.NewEntry(log.New())}
		uc.UpgradedMasterVMs = upgradedVMs(upgradedNodes...)
		for range uc.upgradedNotReadyStream(mock, wait.Backoff{Steps: 1, Duration: 500 * time.Millisecond}) {
		}
	})
}
