// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT license.

package engine

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/Azure/aks-engine-azurestack/pkg/api"
	"github.com/Azure/aks-engine-azurestack/pkg/api/common"
	"github.com/Azure/aks-engine-azurestack/pkg/helpers/to"
	"github.com/Azure/azure-sdk-for-go/profiles/2020-09-01/compute"
)

func CreateCustomScriptExtension(cs *api.ContainerService) VirtualMachineExtensionARM {
	location := "[variables('location')]"
	name := "[concat(variables('masterVMNamePrefix'), copyIndex(variables('masterOffset')),'/cse', '-master-', copyIndex(variables('masterOffset')))]"
	var userAssignedIDEnabled bool
	if cs.Properties.OrchestratorProfile != nil && cs.Properties.OrchestratorProfile.KubernetesConfig != nil {
		userAssignedIDEnabled = cs.Properties.OrchestratorProfile.KubernetesConfig.UserAssignedIDEnabled()
	} else {
		userAssignedIDEnabled = false
	}
	isVHD := "false"
	if cs.Properties.MasterProfile != nil {
		isVHD = strconv.FormatBool(cs.Properties.MasterProfile.IsVHDDistro())
	}

	vmExtension := compute.VirtualMachineExtension{
		Location: to.StringPtr(location),
		Name:     to.StringPtr(name),
		VirtualMachineExtensionProperties: &compute.VirtualMachineExtensionProperties{
			Publisher:               to.StringPtr("Microsoft.Azure.Extensions"),
			Type:                    to.StringPtr("CustomScript"),
			TypeHandlerVersion:      to.StringPtr("2.0"),
			AutoUpgradeMinorVersion: to.BoolPtr(true),
			Settings:                &map[string]interface{}{},
			ProtectedSettings: &map[string]interface{}{
				//note that any change to this property will trigger a CSE rerun on upgrade as well as reimage.
				"commandToExecute": fmt.Sprintf("[concat('echo $(date),$(hostname); for i in $(seq 1 1200); do grep -Fq \"EOF\" /opt/azure/containers/provision.sh && break; if [ $i -eq 1200 ]; then exit 100; else sleep 1; fi; done; ', variables('provisionScriptParametersCommon'),%s,variables('provisionScriptParametersMaster'), ' IS_VHD=%s /usr/bin/nohup /bin/bash -c \"/bin/bash /opt/azure/containers/provision.sh >> %s 2>&1\"')]", generateUserAssignedIdentityClientIDParameter(userAssignedIDEnabled), isVHD, linuxCSELogPath),
			},
		},
		Type: to.StringPtr("Microsoft.Compute/virtualMachines/extensions"),
		Tags: map[string]*string{},
	}
	return VirtualMachineExtensionARM{
		ARMResource: ARMResource{
			APIVersion: "[variables('apiVersionCompute')]",
			Copy: map[string]string{
				"count": "[sub(variables('masterCount'), variables('masterOffset'))]",
				"name":  "vmLoopNode",
			},
			DependsOn: []string{"[concat('Microsoft.Compute/virtualMachines/', variables('masterVMNamePrefix'), copyIndex(variables('masterOffset')))]"},
		},
		VirtualMachineExtension: vmExtension,
	}
}

func createAgentVMASCustomScriptExtension(cs *api.ContainerService, profile *api.AgentPoolProfile) VirtualMachineExtensionARM {
	location := "[variables('location')]"
	name := fmt.Sprintf("[concat(variables('%[1]sVMNamePrefix'), copyIndex(variables('%[1]sOffset')),'/cse', '-agent-', copyIndex(variables('%[1]sOffset')))]", profile.Name)
	var userAssignedIDEnabled bool
	if cs.Properties.OrchestratorProfile != nil && cs.Properties.OrchestratorProfile.KubernetesConfig != nil {
		userAssignedIDEnabled = cs.Properties.OrchestratorProfile.KubernetesConfig.UserAssignedIDEnabled()
	} else {
		userAssignedIDEnabled = false
	}

	runInBackground := ""

	if cs.Properties.FeatureFlags.IsFeatureEnabled("CSERunInBackground") {
		runInBackground = " &"
	}

	nVidiaEnabled := strconv.FormatBool(common.IsNvidiaEnabledSKU(profile.VMSize))
	sgxEnabled := strconv.FormatBool(common.IsSgxEnabledSKU(profile.VMSize))
	auditDEnabled := strconv.FormatBool(to.Bool(profile.AuditDEnabled))
	isVHD := strconv.FormatBool(profile.IsVHDDistro())

	vmExtension := compute.VirtualMachineExtension{
		Location: to.StringPtr(location),
		Name:     to.StringPtr(name),
		VirtualMachineExtensionProperties: &compute.VirtualMachineExtensionProperties{
			AutoUpgradeMinorVersion: to.BoolPtr(true),
			Settings:                &map[string]interface{}{},
		},
		Type: to.StringPtr("Microsoft.Compute/virtualMachines/extensions"),
	}

	if profile.IsWindows() {
		vmExtension.Publisher = to.StringPtr("Microsoft.Compute")
		vmExtension.VirtualMachineExtensionProperties.Type = to.StringPtr("CustomScriptExtension")
		vmExtension.TypeHandlerVersion = to.StringPtr("1.8")
		commandExec := fmt.Sprintf("[concat('echo %s && powershell.exe -ExecutionPolicy Unrestricted -command \"', '$arguments = ', variables('singleQuote'),'-MasterIP ',variables('kubernetesAPIServerIP'),' -KubeDnsServiceIp ',parameters('kubeDnsServiceIp'),%s' -MasterFQDNPrefix ',variables('masterFqdnPrefix'),' -Location ',variables('location'),' -TargetEnvironment ',parameters('targetEnvironment'),' -AgentKey ',parameters('clientPrivateKey'),' -AADClientId ',variables('servicePrincipalClientId'),' -AADClientSecret ',variables('singleQuote'),variables('singleQuote'),base64(variables('servicePrincipalClientSecret')),variables('singleQuote'),variables('singleQuote'),' -NetworkAPIVersion ',variables('apiVersionNetwork'),' ',variables('singleQuote'), ' ; ', variables('windowsCustomScriptSuffix'), '\" > %s 2>&1 ; exit $LASTEXITCODE')]", "%DATE%,%TIME%,%COMPUTERNAME%", generateUserAssignedIdentityClientIDParameterForWindows(userAssignedIDEnabled), "%SYSTEMDRIVE%\\AzureData\\CustomDataSetupScript.log")
		vmExtension.ProtectedSettings = &map[string]interface{}{
			"commandToExecute": commandExec,
		}
	} else {
		vmExtension.Publisher = to.StringPtr("Microsoft.Azure.Extensions")
		vmExtension.VirtualMachineExtensionProperties.Type = to.StringPtr("CustomScript")
		vmExtension.TypeHandlerVersion = to.StringPtr("2.0")
		commandExec := fmt.Sprintf("[concat('echo $(date),$(hostname); for i in $(seq 1 1200); do grep -Fq \"EOF\" /opt/azure/containers/provision.sh && break; if [ $i -eq 1200 ]; then exit 100; else sleep 1; fi; done; ', variables('provisionScriptParametersCommon'),%s,' IS_VHD=%s GPU_NODE=%s SGX_NODE=%s AUDITD_ENABLED=%s /usr/bin/nohup /bin/bash -c \"/bin/bash /opt/azure/containers/provision.sh >> %s 2>&1%s\"')]", generateUserAssignedIdentityClientIDParameter(userAssignedIDEnabled), isVHD, nVidiaEnabled, sgxEnabled, auditDEnabled, linuxCSELogPath, runInBackground)
		vmExtension.ProtectedSettings = &map[string]interface{}{
			"commandToExecute": commandExec,
		}
	}

	dependency := fmt.Sprintf("[concat('Microsoft.Compute/virtualMachines/', variables('%[1]sVMNamePrefix'), copyIndex(variables('%[1]sOffset')))]", profile.Name)

	return VirtualMachineExtensionARM{
		ARMResource: ARMResource{
			APIVersion: "[variables('apiVersionCompute')]",
			Copy: map[string]string{
				"count": fmt.Sprintf("[sub(variables('%[1]sCount'), variables('%[1]sOffset'))]", profile.Name),
				"name":  "vmLoopNode",
			},
			DependsOn: []string{dependency},
		},
		VirtualMachineExtension: vmExtension,
	}
}

// CreateCustomExtensions returns a list of DeploymentARM objects for the custom extensions to be deployed
func CreateCustomExtensions(properties *api.Properties) []DeploymentARM {
	var extensionsARM []DeploymentARM

	if properties.MasterProfile != nil {
		// The first extension needs to depend on the master cse created for all nodes
		// Each proceeding extension needs to depend on the previous one to avoid ARM conflicts in the Compute RP
		nextDependsOn := "[concat('Microsoft.Compute/virtualMachines/', variables('masterVMNamePrefix'), copyIndex(variables('masterOffset')), '/extensions/cse-master-', copyIndex(variables('masterOffset')))]"

		for _, extensionProfile := range properties.ExtensionProfiles {
			masterOptedForExtension, singleOrAll := validateProfileOptedForExtension(extensionProfile.Name, properties.MasterProfile.Extensions)
			if masterOptedForExtension {
				data, e := getMasterLinkedTemplateText(properties.OrchestratorProfile.OrchestratorType, extensionProfile, singleOrAll)
				if e != nil {
					fmt.Println(e.Error())
				}
				var ext DeploymentARM
				e = json.Unmarshal([]byte(data), &ext)
				if e != nil {
					fmt.Println(e.Error())
				}
				ext.DependsOn = []string{nextDependsOn}
				nextDependsOn = *ext.Name
				extensionsARM = append(extensionsARM, ext)
			}
		}
	}

	for _, agentPoolProfile := range properties.AgentPoolProfiles {
		// The first extension needs to depend on the agent cse created for all nodes
		// Each proceeding extension needs to depend on the previous one to avoid ARM conflicts in the Compute RP
		nextDependsOn := fmt.Sprintf("[concat('Microsoft.Compute/virtualMachines/', variables('%[1]sVMNamePrefix'), copyIndex(variables('%[1]sOffset')), '/extensions/cse-agent-', copyIndex(variables('%[1]sOffset')))]", agentPoolProfile.Name)

		for _, extensionProfile := range properties.ExtensionProfiles {
			poolOptedForExtension, singleOrAll := validateProfileOptedForExtension(extensionProfile.Name, agentPoolProfile.Extensions)
			if poolOptedForExtension {
				data, e := getAgentPoolLinkedTemplateText(agentPoolProfile, properties.OrchestratorProfile.OrchestratorType, extensionProfile, singleOrAll)
				if e != nil {
					fmt.Println(e.Error())
				}
				var ext DeploymentARM
				e = json.Unmarshal([]byte(data), &ext)
				if e != nil {
					fmt.Println(e.Error())
				}
				ext.DependsOn = []string{nextDependsOn}
				nextDependsOn = *ext.Name
				extensionsARM = append(extensionsARM, ext)
			}
		}
	}

	return extensionsARM
}
