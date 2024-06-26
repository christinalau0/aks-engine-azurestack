// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT license.

package engine

import (
	"strings"

	"github.com/Azure/aks-engine-azurestack/pkg/api"
	"github.com/Azure/aks-engine-azurestack/pkg/helpers/to"
)

func createKubernetesMasterResourcesVMAS(cs *api.ContainerService) []interface{} {
	var masterResources []interface{}

	p := cs.Properties

	if p.MasterProfile.HasCosmosEtcd() {
		masterResources = append(masterResources, createCosmosDBAccount())
	}

	if p.HasManagedDisks() {
		if !p.HasAvailabilityZones() {
			masterResources = append(masterResources, CreateAvailabilitySet(cs, true))
		}
	} else if p.MasterProfile.IsStorageAccount() {
		availabilitySet := CreateAvailabilitySet(cs, false)
		storageAccount := createStorageAccount(cs)
		masterResources = append(masterResources, availabilitySet, storageAccount)
	}

	if !p.MasterProfile.IsCustomVNET() {
		virtualNetwork := CreateVirtualNetwork(cs)
		masterResources = append(masterResources, virtualNetwork)
	}

	masterNsg := CreateNetworkSecurityGroup(cs)
	masterResources = append(masterResources, masterNsg)

	if cs.Properties.RequireRouteTable() {
		masterResources = append(masterResources, createRouteTable())
	}

	kubernetesConfig := cs.Properties.OrchestratorProfile.KubernetesConfig

	if kubernetesConfig.PrivateJumpboxProvision() {
		jumpboxVM := createJumpboxVirtualMachine(cs)
		masterResources = append(masterResources, jumpboxVM)
		jumpboxIsManagedDisks :=
			kubernetesConfig.PrivateJumpboxProvision() &&
				kubernetesConfig.PrivateCluster.JumpboxProfile.StorageProfile == api.ManagedDisks
		if !jumpboxIsManagedDisks {
			jumpBoxStorage := createJumpboxStorageAccount()
			masterResources = append(masterResources, jumpBoxStorage)
		}
		jumpboxNSG := createJumpboxNSG()
		jumpboxNIC := createJumpboxNetworkInterface(cs)
		jumpboxPublicIP := createJumpboxPublicIPAddress()
		masterResources = append(masterResources, jumpboxNSG, jumpboxNIC, jumpboxPublicIP)
	}

	var masterNic NetworkInterfaceARM
	if cs.Properties.OrchestratorProfile.IsPrivateCluster() {
		masterNic = createPrivateClusterMasterVMNetworkInterface(cs)
	} else {
		masterNic = CreateMasterVMNetworkInterfaces(cs)
	}
	masterResources = append(masterResources, masterNic)

	// We don't create a master load balancer in a private cluster + single master vm scenario
	if !(cs.Properties.OrchestratorProfile.IsPrivateCluster() && !p.MasterProfile.HasMultipleNodes()) &&
		// And we don't create a master load balancer in a private cluster + Basic LB scenario
		!(cs.Properties.OrchestratorProfile.IsPrivateCluster() && cs.Properties.OrchestratorProfile.KubernetesConfig.LoadBalancerSku == api.BasicLoadBalancerSku) {
		loadBalancer := CreateMasterLoadBalancer(cs.Properties)
		// In a private cluster scenario, the master NIC spec is different,
		// and the master LB is for outbound access only and doesn't require a DNS record for the public IP
		includeDNS := !cs.Properties.OrchestratorProfile.IsPrivateCluster()
		publicIPAddress := CreatePublicIPAddressForMaster(includeDNS)
		masterResources = append(masterResources, publicIPAddress, loadBalancer)
	}

	if p.MasterProfile.HasMultipleNodes() {
		internalLB := CreateMasterInternalLoadBalancer(cs)
		masterResources = append(masterResources, internalLB)
	}

	var isKMSEnabled bool
	if kubernetesConfig != nil {
		isKMSEnabled = to.Bool(kubernetesConfig.EnableEncryptionWithExternalKms)
	}

	if isKMSEnabled {
		keyVaultStorageAccount := createKeyVaultStorageAccount()
		keyVault := CreateKeyVaultVMAS(cs)
		keyVaultKey := CreateKeyVaultKey(cs)
		masterResources = append(masterResources, keyVaultStorageAccount, keyVault, keyVaultKey)
	}

	if cs.Properties.FeatureFlags.IsFeatureEnabled("EnableIPv6DualStack") {
		// for standard lb sku, the loadbalancer and ipv4 FE is already created
		if cs.Properties.OrchestratorProfile.KubernetesConfig.LoadBalancerSku != api.StandardLoadBalancerSku {
			clusterIPv4PublicIPAddress := CreateClusterPublicIPAddress()
			clusterLB := CreateClusterLoadBalancerForIPv6()

			masterResources = append(masterResources, clusterIPv4PublicIPAddress, clusterLB)
		}
	}

	masterVM := CreateMasterVM(cs)
	masterResources = append(masterResources, masterVM)

	var useManagedIdentity, userAssignedIDEnabled bool
	useManagedIdentity = to.Bool(kubernetesConfig.UseManagedIdentity)
	userAssignedIDEnabled = kubernetesConfig.UserAssignedIDEnabled()

	if useManagedIdentity && !userAssignedIDEnabled {
		vmasRoleAssignment := createVMASRoleAssignment()
		masterResources = append(masterResources, vmasRoleAssignment)
	}

	masterCSE := CreateCustomScriptExtension(cs)
	if isKMSEnabled {
		masterCSE.ARMResource.DependsOn = append(masterCSE.ARMResource.DependsOn, "[concat('Microsoft.KeyVault/vaults/', variables('clusterKeyVaultName'))]")
	}

	// If the control plane is in a discrete VNET
	if hasDistinctControlPlaneVNET(cs) {
		// TODO: This is only necessary if the resource group of the masters is different from the RG of the node pool
		// subnet. But when we generate the template we don't know to which RG it will be deployed to. To solve this we
		// would have to add the necessary condition into the template. For the resources we can use the `condition` field
		// but how can we conditionally declare the dependencies? Perhaps by creating a variable for the dependency array
		// and conditionally adding more dependencies.
		if kubernetesConfig.SystemAssignedIDEnabled() &&
			// The fix for ticket 2373 is only available for individual VMs / AvailabilitySet.
			cs.Properties.MasterProfile.IsAvailabilitySet() {
			masterRoleAssignmentForAgentPools := createKubernetesMasterRoleAssignmentForAgentPools(cs.Properties.MasterProfile, cs.Properties.AgentPoolProfiles)

			for _, assignmentForAgentPool := range masterRoleAssignmentForAgentPools {
				masterResources = append(masterResources, assignmentForAgentPool)
				masterCSE.ARMResource.DependsOn = append(masterCSE.ARMResource.DependsOn, *assignmentForAgentPool.Name)
			}
		}
	}

	masterResources = append(masterResources, masterCSE)

	customExtensions := CreateCustomExtensions(cs.Properties)
	for _, ext := range customExtensions {
		masterResources = append(masterResources, ext)
	}

	return masterResources
}

// hasDistinctControlPlaneVNET returns whether or not the VNET config of the control plane is distinct from any one node pool
// If the VnetSubnetID string is malformed in either the MasterProfile or any AgentPoolProfile, we return false
func hasDistinctControlPlaneVNET(cs *api.ContainerService) bool {
	var controlPlaneVNETResourceURI string
	if cs.Properties.MasterProfile.VnetSubnetID != "" {
		controlPlaneSubnetElements := strings.Split(cs.Properties.MasterProfile.VnetSubnetID, "/")
		if len(controlPlaneSubnetElements) >= 9 {
			controlPlaneVNETResourceURI = strings.Join(controlPlaneSubnetElements[:9], "/")
		} else {
			return false
		}
	}
	for _, agentPool := range cs.Properties.AgentPoolProfiles {
		if agentPool.VnetSubnetID != "" {
			nodePoolSubnetElements := strings.Split(agentPool.VnetSubnetID, "/")
			if len(nodePoolSubnetElements) < 9 {
				return false
			}
			if strings.Join(nodePoolSubnetElements[:9], "/") != controlPlaneVNETResourceURI {
				return true
			}
		} else {
			if cs.Properties.MasterProfile.VnetSubnetID != "" {
				return true
			}
		}
	}
	return false
}
