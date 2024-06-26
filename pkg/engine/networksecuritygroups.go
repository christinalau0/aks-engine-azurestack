// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT license.

package engine

import (
	"github.com/Azure/aks-engine-azurestack/pkg/api"
	"github.com/Azure/aks-engine-azurestack/pkg/helpers/to"
	"github.com/Azure/azure-sdk-for-go/profiles/2020-09-01/network/mgmt/network"
)

func CreateNetworkSecurityGroup(cs *api.ContainerService) NetworkSecurityGroupARM {
	armResource := ARMResource{
		APIVersion: "[variables('apiVersionNetwork')]",
	}

	sshRule := network.SecurityRule{
		Name: to.StringPtr("allow_ssh"),
		SecurityRulePropertiesFormat: &network.SecurityRulePropertiesFormat{
			Access:                   network.SecurityRuleAccessAllow,
			Description:              to.StringPtr("Allow SSH traffic to master"),
			DestinationAddressPrefix: to.StringPtr("*"),
			DestinationPortRange:     to.StringPtr("22-22"),
			Direction:                network.SecurityRuleDirectionInbound,
			Priority:                 to.Int32Ptr(101),
			Protocol:                 network.SecurityRuleProtocolTCP,
			SourceAddressPrefix:      to.StringPtr("*"),
			SourcePortRange:          to.StringPtr("*"),
		},
	}

	kubeTLSRule := network.SecurityRule{
		Name: to.StringPtr("allow_kube_tls"),
		SecurityRulePropertiesFormat: &network.SecurityRulePropertiesFormat{
			Access:                   network.SecurityRuleAccessAllow,
			Description:              to.StringPtr("Allow kube-apiserver (tls) traffic to master"),
			DestinationAddressPrefix: to.StringPtr("*"),
			DestinationPortRange:     to.StringPtr("443-443"),
			Direction:                network.SecurityRuleDirectionInbound,
			Priority:                 to.Int32Ptr(100),
			Protocol:                 network.SecurityRuleProtocolTCP,
			SourceAddressPrefix:      to.StringPtr("*"),
			SourcePortRange:          to.StringPtr("*"),
		},
	}

	if cs.Properties.OrchestratorProfile.IsPrivateCluster() {
		source := "VirtualNetwork"
		kubeTLSRule.SourceAddressPrefix = &source
	}

	securityRules := []network.SecurityRule{
		sshRule,
		kubeTLSRule,
	}

	if cs.Properties.HasWindows() {
		rdpRule := network.SecurityRule{
			Name: to.StringPtr("allow_rdp"),
			SecurityRulePropertiesFormat: &network.SecurityRulePropertiesFormat{
				Access:                   network.SecurityRuleAccessAllow,
				Description:              to.StringPtr("Allow RDP traffic to master"),
				DestinationAddressPrefix: to.StringPtr("*"),
				DestinationPortRange:     to.StringPtr("3389-3389"),
				Direction:                network.SecurityRuleDirectionInbound,
				Priority:                 to.Int32Ptr(102),
				Protocol:                 network.SecurityRuleProtocolTCP,
				SourceAddressPrefix:      to.StringPtr("*"),
				SourcePortRange:          to.StringPtr("*"),
			},
		}

		securityRules = append(securityRules, rdpRule)
	}

	if cs.Properties.FeatureFlags.IsFeatureEnabled("BlockOutboundInternet") {
		vnetRule := network.SecurityRule{
			Name: to.StringPtr("allow_vnet"),
			SecurityRulePropertiesFormat: &network.SecurityRulePropertiesFormat{
				Access:                   network.SecurityRuleAccessAllow,
				Description:              to.StringPtr("Allow outbound internet to vnet"),
				DestinationAddressPrefix: to.StringPtr("[parameters('masterSubnet')]"),
				DestinationPortRange:     to.StringPtr("*"),
				Direction:                network.SecurityRuleDirectionOutbound,
				Priority:                 to.Int32Ptr(110),
				Protocol:                 network.SecurityRuleProtocolAsterisk,
				SourceAddressPrefix:      to.StringPtr("VirtualNetwork"),
				SourcePortRange:          to.StringPtr("*"),
			},
		}

		blockOutBoundRule := network.SecurityRule{
			Name: to.StringPtr("block_outbound"),
			SecurityRulePropertiesFormat: &network.SecurityRulePropertiesFormat{
				Access:                   network.SecurityRuleAccessDeny,
				Description:              to.StringPtr("Block outbound internet from master"),
				DestinationAddressPrefix: to.StringPtr("*"),
				DestinationPortRange:     to.StringPtr("*"),
				Direction:                network.SecurityRuleDirectionOutbound,
				Priority:                 to.Int32Ptr(120),
				Protocol:                 network.SecurityRuleProtocolAsterisk,
				SourceAddressPrefix:      to.StringPtr("*"),
				SourcePortRange:          to.StringPtr("*"),
			},
		}

		allowARMRule := network.SecurityRule{
			Name: to.StringPtr("allow_ARM"),
			SecurityRulePropertiesFormat: &network.SecurityRulePropertiesFormat{
				Access:                   network.SecurityRuleAccessAllow,
				Description:              to.StringPtr("Allow outbound internet to ARM"),
				DestinationAddressPrefix: to.StringPtr("AzureResourceManager"),
				DestinationPortRange:     to.StringPtr("443"),
				Direction:                network.SecurityRuleDirectionOutbound,
				Priority:                 to.Int32Ptr(100),
				Protocol:                 network.SecurityRuleProtocolTCP,
				SourceAddressPrefix:      to.StringPtr("*"),
				SourcePortRange:          to.StringPtr("*"),
			},
		}

		securityRules = append(securityRules, vnetRule)
		securityRules = append(securityRules, blockOutBoundRule)
		securityRules = append(securityRules, allowARMRule)
	}

	nsg := network.SecurityGroup{
		Location: to.StringPtr("[variables('location')]"),
		Name:     to.StringPtr("[variables('nsgName')]"),
		Type:     to.StringPtr("Microsoft.Network/networkSecurityGroups"),
		SecurityGroupPropertiesFormat: &network.SecurityGroupPropertiesFormat{
			SecurityRules: &securityRules,
		},
	}

	return NetworkSecurityGroupARM{
		ARMResource:   armResource,
		SecurityGroup: nsg,
	}
}

func createJumpboxNSG() NetworkSecurityGroupARM {
	armResource := ARMResource{
		APIVersion: "[variables('apiVersionNetwork')]",
	}

	securityRules := []network.SecurityRule{
		{
			Name: to.StringPtr("default-allow-ssh"),
			SecurityRulePropertiesFormat: &network.SecurityRulePropertiesFormat{
				Priority:                 to.Int32Ptr(1000),
				Protocol:                 network.SecurityRuleProtocolTCP,
				Access:                   network.SecurityRuleAccessAllow,
				Direction:                network.SecurityRuleDirectionInbound,
				SourceAddressPrefix:      to.StringPtr("*"),
				SourcePortRange:          to.StringPtr("*"),
				DestinationAddressPrefix: to.StringPtr("*"),
				DestinationPortRange:     to.StringPtr("22"),
			},
		},
	}
	nsg := network.SecurityGroup{
		Location: to.StringPtr("[variables('location')]"),
		Name:     to.StringPtr("[variables('jumpboxNetworkSecurityGroupName')]"),
		Type:     to.StringPtr("Microsoft.Network/networkSecurityGroups"),
		SecurityGroupPropertiesFormat: &network.SecurityGroupPropertiesFormat{
			SecurityRules: &securityRules,
		},
	}
	return NetworkSecurityGroupARM{
		ARMResource:   armResource,
		SecurityGroup: nsg,
	}
}
