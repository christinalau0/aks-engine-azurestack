// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT license.

package engine

import (
	"github.com/Azure/aks-engine-azurestack/pkg/api"
	"github.com/Azure/aks-engine-azurestack/pkg/api/common"
	"github.com/Azure/aks-engine-azurestack/pkg/helpers/to"
	"github.com/Azure/azure-sdk-for-go/profiles/2020-09-01/network/mgmt/network"
)

func CreateVirtualNetwork(cs *api.ContainerService) VirtualNetworkARM {

	dependencies := []string{
		"[concat('Microsoft.Network/networkSecurityGroups/', variables('nsgName'))]",
	}

	requireRouteTable := cs.Properties.RequireRouteTable()
	if requireRouteTable {
		dependencies = append(dependencies, "[concat('Microsoft.Network/routeTables/', variables('routeTableName'))]")
	}

	armResource := ARMResource{
		APIVersion: "[variables('apiVersionNetwork')]",
		DependsOn:  dependencies,
	}

	subnet := network.Subnet{
		Name: to.StringPtr("[variables('subnetName')]"),
		SubnetPropertiesFormat: &network.SubnetPropertiesFormat{
			AddressPrefix: to.StringPtr("[parameters('masterSubnet')]"),
			NetworkSecurityGroup: &network.SecurityGroup{
				ID: to.StringPtr("[variables('nsgID')]"),
			},
		},
	}

	masterAddressPrefixes := []string{"[parameters('masterSubnet')]"}
	// add ipv6 vnet cidr if dual stack enabled
	if cs.Properties.FeatureFlags.IsFeatureEnabled("EnableIPv6DualStack") ||
		cs.Properties.FeatureFlags.IsFeatureEnabled("EnableIPv6Only") {
		masterAddressPrefixes = append(masterAddressPrefixes, "[parameters('masterSubnetIPv6')]")
		subnet.AddressPrefix = nil
		subnet.AddressPrefixes = &masterAddressPrefixes
	}

	if requireRouteTable {
		subnet.RouteTable = &network.RouteTable{
			ID: to.StringPtr("[variables('routeTableID')]"),
		}
	}

	addressPrefixes := []string{"[parameters('vnetCidr')]"}
	// add ipv6 vnet cidr if dual stack enabled
	if cs.Properties.FeatureFlags.IsFeatureEnabled("EnableIPv6DualStack") ||
		cs.Properties.FeatureFlags.IsFeatureEnabled("EnableIPv6Only") {
		addressPrefixes = append(addressPrefixes, "[parameters('vnetCidrIPv6')]")
	}

	virtualNetwork := network.VirtualNetwork{
		Location: to.StringPtr("[variables('location')]"),
		Name:     to.StringPtr("[variables('virtualNetworkName')]"),
		Type:     to.StringPtr("Microsoft.Network/virtualNetworks"),
		VirtualNetworkPropertiesFormat: &network.VirtualNetworkPropertiesFormat{
			AddressSpace: &network.AddressSpace{
				AddressPrefixes: &addressPrefixes,
			},
			Subnets: &[]network.Subnet{
				subnet,
			},
		},
	}

	if cs.Properties.OrchestratorProfile.KubernetesConfig.IsAddonEnabled(common.AppGwIngressAddonName) {
		subnetAppGw := network.Subnet{
			Name: to.StringPtr("[variables('appGwSubnetName')]"),
			SubnetPropertiesFormat: &network.SubnetPropertiesFormat{
				AddressPrefix: to.StringPtr("[parameters('appGwSubnet')]"),
			},
		}

		subnets := append(*virtualNetwork.VirtualNetworkPropertiesFormat.Subnets, subnetAppGw)
		virtualNetwork.VirtualNetworkPropertiesFormat.Subnets = &subnets
	}

	return VirtualNetworkARM{
		ARMResource:    armResource,
		VirtualNetwork: virtualNetwork,
	}
}

func createVirtualNetworkVMSS(cs *api.ContainerService) VirtualNetworkARM {

	dependencies := []string{
		"[concat('Microsoft.Network/networkSecurityGroups/', variables('nsgName'))]",
	}

	requireRouteTable := cs.Properties.RequireRouteTable()
	if requireRouteTable {
		dependencies = append(dependencies, "[concat('Microsoft.Network/routeTables/', variables('routeTableName'))]")
	}

	armResource := ARMResource{
		APIVersion: "[variables('apiVersionNetwork')]",
		DependsOn:  dependencies,
	}

	subnetMaster := network.Subnet{
		Name: to.StringPtr("subnetmaster"),
		SubnetPropertiesFormat: &network.SubnetPropertiesFormat{
			AddressPrefix: to.StringPtr("[parameters('masterSubnet')]"),
			NetworkSecurityGroup: &network.SecurityGroup{
				ID: to.StringPtr("[variables('nsgID')]"),
			},
		},
	}
	masterAddressPrefixes := []string{"[parameters('masterSubnet')]"}
	// add ipv6 vnet cidr if dual stack enabled
	if cs.Properties.FeatureFlags.IsFeatureEnabled("EnableIPv6DualStack") ||
		cs.Properties.FeatureFlags.IsFeatureEnabled("EnableIPv6Only") {
		masterAddressPrefixes = append(masterAddressPrefixes, "[parameters('masterSubnetIPv6')]")
		subnetMaster.AddressPrefix = nil
		subnetMaster.AddressPrefixes = &masterAddressPrefixes
	}

	if requireRouteTable {
		subnetMaster.RouteTable = &network.RouteTable{
			ID: to.StringPtr("[variables('routeTableID')]"),
		}
	}

	subnetAgent := network.Subnet{
		Name: to.StringPtr("subnetagent"),
		SubnetPropertiesFormat: &network.SubnetPropertiesFormat{
			AddressPrefix: to.StringPtr("[parameters('agentSubnet')]"),
			NetworkSecurityGroup: &network.SecurityGroup{
				ID: to.StringPtr("[variables('nsgID')]"),
			},
		},
	}

	if requireRouteTable {
		subnetAgent.RouteTable = &network.RouteTable{
			ID: to.StringPtr("[variables('routeTableID')]"),
		}
	}

	addressPrefixes := []string{"[parameters('vnetCidr')]"}
	// add ipv6 vnet cidr if dual stack enabled
	if cs.Properties.FeatureFlags.IsFeatureEnabled("EnableIPv6DualStack") ||
		cs.Properties.FeatureFlags.IsFeatureEnabled("EnableIPv6Only") {
		addressPrefixes = append(addressPrefixes, "[parameters('vnetCidrIPv6')]")
	}

	virtualNetwork := network.VirtualNetwork{
		Location: to.StringPtr("[variables('location')]"),
		Name:     to.StringPtr("[variables('virtualNetworkName')]"),
		Type:     to.StringPtr("Microsoft.Network/virtualNetworks"),
		VirtualNetworkPropertiesFormat: &network.VirtualNetworkPropertiesFormat{
			AddressSpace: &network.AddressSpace{
				AddressPrefixes: &addressPrefixes,
			},
			Subnets: &[]network.Subnet{
				subnetMaster,
				subnetAgent,
			},
		},
	}

	if cs.Properties.OrchestratorProfile.KubernetesConfig.IsAddonEnabled(common.AppGwIngressAddonName) {
		subnetAppGw := network.Subnet{
			Name: to.StringPtr("[variables('appGwSubnetName')]"),
			SubnetPropertiesFormat: &network.SubnetPropertiesFormat{
				AddressPrefix: to.StringPtr("[parameters('appGwSubnet')]"),
			},
		}

		subnets := append(*virtualNetwork.VirtualNetworkPropertiesFormat.Subnets, subnetAppGw)
		virtualNetwork.VirtualNetworkPropertiesFormat.Subnets = &subnets
	}

	return VirtualNetworkARM{
		ARMResource:    armResource,
		VirtualNetwork: virtualNetwork,
	}
}
