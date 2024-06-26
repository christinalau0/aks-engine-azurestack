// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT license.

package engine

import (
	"fmt"

	"github.com/Azure/aks-engine-azurestack/pkg/api"
	"github.com/Azure/aks-engine-azurestack/pkg/helpers/to"
	"github.com/Azure/azure-sdk-for-go/profiles/2020-09-01/storage/mgmt/storage"
)

func createStorageAccount(cs *api.ContainerService) StorageAccountARM {
	armResource := ARMResource{
		APIVersion: "[variables('apiVersionStorage')]",
	}

	if !cs.Properties.OrchestratorProfile.IsPrivateCluster() {
		armResource.DependsOn = []string{
			"[concat('Microsoft.Network/publicIPAddresses/', variables('masterPublicIPAddressName'))]",
		}
	}

	storageAccount := storage.Account{
		Location: to.StringPtr("[variables('location')]"),
		Name:     to.StringPtr("[variables('masterStorageAccountName')]"),
		Type:     to.StringPtr("Microsoft.Storage/storageAccounts"),
		Sku: &storage.Sku{
			Name: storage.SkuName("[variables('vmSizesMap')[parameters('masterVMSize')].storageAccountType]"),
		},
	}

	return StorageAccountARM{
		ARMResource: armResource,
		Account:     storageAccount,
	}
}

func createJumpboxStorageAccount() StorageAccountARM {
	armResource := ARMResource{
		APIVersion: "[variables('apiVersionStorage')]",
	}

	storageAccount := storage.Account{
		Type:     to.StringPtr("Microsoft.Storage/storageAccounts"),
		Name:     to.StringPtr("[variables('jumpboxStorageAccountName')]"),
		Location: to.StringPtr("[variables('location')]"),
		Sku: &storage.Sku{
			Name: storage.SkuName("[variables('vmSizesMap')[parameters('jumpboxVMSize')].storageAccountType]"),
		},
	}

	return StorageAccountARM{
		ARMResource: armResource,
		Account:     storageAccount,
	}
}

func createKeyVaultStorageAccount() StorageAccountARM {
	return StorageAccountARM{
		ARMResource: ARMResource{
			APIVersion: "[variables('apiVersionStorage')]",
		},
		Account: storage.Account{
			Type:     to.StringPtr("Microsoft.Storage/storageAccounts"),
			Name:     to.StringPtr("[variables('clusterKeyVaultName')]"),
			Location: to.StringPtr("[variables('location')]"),
			Sku: &storage.Sku{
				Name: storage.StandardLRS,
			},
		},
	}
}

func createAgentVMASStorageAccount(cs *api.ContainerService, profile *api.AgentPoolProfile, isDataDisk bool) StorageAccountARM {
	var copyName string
	if isDataDisk {
		copyName = "datadiskLoop"
	} else {
		copyName = "loop"
	}

	armResource := ARMResource{
		APIVersion: "[variables('apiVersionStorage')]",
		Copy: map[string]string{
			"count": fmt.Sprintf("[variables('%sStorageAccountsCount')]", profile.Name),
			"name":  copyName,
		},
	}

	if !cs.Properties.OrchestratorProfile.IsPrivateCluster() {
		armResource.DependsOn = []string{
			"[concat('Microsoft.Network/publicIPAddresses/', variables('masterPublicIPAddressName'))]",
		}
	}

	storageAccount := storage.Account{
		Type:     to.StringPtr("Microsoft.Storage/storageAccounts"),
		Location: to.StringPtr("[variables('location')]"),
		Sku: &storage.Sku{
			Name: storage.SkuName(fmt.Sprintf("[variables('vmSizesMap')[variables('%sVMSize')].storageAccountType]", profile.Name)),
		},
	}

	if isDataDisk {
		storageAccount.Name = to.StringPtr(fmt.Sprintf("[concat(variables('storageAccountPrefixes')[mod(add(copyIndex(variables('dataStorageAccountPrefixSeed')),variables('%[1]sStorageAccountOffset')),variables('storageAccountPrefixesCount'))],variables('storageAccountPrefixes')[div(add(copyIndex(variables('dataStorageAccountPrefixSeed')),variables('%[1]sStorageAccountOffset')),variables('storageAccountPrefixesCount'))],variables('%[1]sDataAccountName'))]", profile.Name))
	} else {
		storageAccount.Name = to.StringPtr(fmt.Sprintf("[concat(variables('storageAccountPrefixes')[mod(add(copyIndex(),variables('%[1]sStorageAccountOffset')),variables('storageAccountPrefixesCount'))],variables('storageAccountPrefixes')[div(add(copyIndex(),variables('%[1]sStorageAccountOffset')),variables('storageAccountPrefixesCount'))],variables('%[1]sAccountName'))]", profile.Name))
	}

	return StorageAccountARM{
		ARMResource: armResource,
		Account:     storageAccount,
	}
}
