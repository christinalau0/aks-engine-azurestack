// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT license.

package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Azure/aks-engine-azurestack/pkg/api"
	"github.com/Azure/aks-engine-azurestack/pkg/api/vlabs"
	"github.com/Azure/aks-engine-azurestack/pkg/armhelpers"
	"github.com/Azure/aks-engine-azurestack/pkg/engine"
	"github.com/Azure/aks-engine-azurestack/pkg/engine/transform"
	"github.com/Azure/aks-engine-azurestack/pkg/helpers"
	"github.com/Azure/aks-engine-azurestack/pkg/i18n"
	"github.com/Azure/aks-engine-azurestack/pkg/kubernetes"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/cloud"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	flag "github.com/spf13/pflag"
	ini "gopkg.in/ini.v1"
)

const (
	rootName             = "aks-engine-azurestack"
	rootShortDescription = "AKS Engine deploys and manages Kubernetes clusters in Azure Stack Hub"
	rootLongDescription  = "AKS Engine deploys and manages Kubernetes clusters in Azure Stack Hub"
)

var (
	debug            bool
	dumpDefaultModel bool
)

// NewRootCmd returns the root command for AKS Engine.
func NewRootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   rootName,
		Short: rootShortDescription,
		Long:  rootLongDescription,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			if debug {
				log.SetLevel(log.DebugLevel)
			}
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if dumpDefaultModel {
				return writeDefaultModel(cmd.OutOrStdout())
			}
			return cmd.Usage()
		},
	}

	p := rootCmd.PersistentFlags()
	p.BoolVar(&debug, "debug", false, "enable verbose debug logs")

	f := rootCmd.Flags()
	f.BoolVar(&dumpDefaultModel, "show-default-model", false, "Dump the default API model to stdout")

	rootCmd.AddCommand(newVersionCmd())
	rootCmd.AddCommand(newGenerateCmd())
	rootCmd.AddCommand(newDeployCmd())
	rootCmd.AddCommand(newGetLogsCmd())
	rootCmd.AddCommand(newGetVersionsCmd())
	rootCmd.AddCommand(newOrchestratorsCmd())
	rootCmd.AddCommand(newUpgradeCmd())
	rootCmd.AddCommand(newScaleCmd())
	rootCmd.AddCommand(newRotateCertsCmd())
	rootCmd.AddCommand(newAddPoolCmd())
	rootCmd.AddCommand(getCompletionCmd(rootCmd))

	return rootCmd
}

func writeDefaultModel(out io.Writer) error {
	meta, p := api.LoadDefaultContainerServiceProperties()
	type withMeta struct {
		APIVersion string            `json:"apiVersion"`
		Properties *vlabs.Properties `json:"properties"`
	}

	b, err := json.MarshalIndent(withMeta{APIVersion: meta.APIVersion, Properties: p}, "", "\t")
	if err != nil {
		return errors.Wrap(err, "error encoding model to json")
	}
	b = append(b, '\n')
	if _, err := out.Write(b); err != nil {
		return errors.Wrap(err, "error writing output")
	}
	return nil
}

type authProvider interface {
	getAuthArgs() *authArgs
	getClient(env *api.Environment) (armhelpers.AKSEngineClient, error)
}

type authArgs struct {
	RawAzureEnvironment string
	rawSubscriptionID   string
	SubscriptionID      uuid.UUID
	AuthMethod          string
	rawClientID         string

	ClientID        uuid.UUID
	ClientSecret    string
	CertificatePath string
	PrivateKeyPath  string
	IdentitySystem  string
	language        string
}

func addAuthFlags(authArgs *authArgs, f *flag.FlagSet) {
	f.StringVar(&authArgs.RawAzureEnvironment, "azure-env", "AzurePublicCloud", "the target Azure cloud")
	f.StringVarP(&authArgs.rawSubscriptionID, "subscription-id", "s", "", "azure subscription id (required)")
	f.StringVar(&authArgs.AuthMethod, "auth-method", "client_secret", "auth method (default:`client_secret`, `client_certificate`)")
	f.StringVar(&authArgs.rawClientID, "client-id", "", "client id (used with --auth-method=[client_secret|client_certificate])")
	f.StringVar(&authArgs.ClientSecret, "client-secret", "", "client secret (used with --auth-method=client_secret)")
	f.StringVar(&authArgs.CertificatePath, "certificate-path", "", "path to client certificate (used with --auth-method=client_certificate)")
	f.StringVar(&authArgs.PrivateKeyPath, "private-key-path", "", "path to private key (used with --auth-method=client_certificate)")
	f.StringVar(&authArgs.IdentitySystem, "identity-system", "azure_ad", "identity system (default:`azure_ad`, `adfs`)")
	f.StringVar(&authArgs.language, "language", "en-us", "language to return error messages in")
}

func (authArgs *authArgs) getAuthArgs() *authArgs {
	return authArgs
}

func (authArgs *authArgs) isAzureStackCloud() bool {
	return strings.EqualFold(authArgs.RawAzureEnvironment, api.AzureStackCloud)
}

func (authArgs *authArgs) validateAuthArgs() error {
	var err error

	if authArgs.AuthMethod == "" {
		return errors.New("--auth-method is a required parameter")
	}

	if authArgs.AuthMethod == "client_secret" || authArgs.AuthMethod == "client_certificate" {
		authArgs.ClientID, err = uuid.Parse(authArgs.rawClientID)
		if err != nil {
			return errors.Wrap(err, "parsing --client-id")
		}
		if authArgs.AuthMethod == "client_secret" {
			if authArgs.ClientSecret == "" {
				return errors.New(`--client-secret must be specified when --auth-method="client_secret"`)
			}
		} else if authArgs.AuthMethod == "client_certificate" {
			if authArgs.CertificatePath == "" || authArgs.PrivateKeyPath == "" {
				return errors.New(`--certificate-path and --private-key-path must be specified when --auth-method="client_certificate"`)
			}
		}
	}

	authArgs.SubscriptionID, _ = uuid.Parse(authArgs.rawSubscriptionID)
	if authArgs.SubscriptionID.String() == "00000000-0000-0000-0000-000000000000" {
		var subID uuid.UUID
		subID, err = getSubFromAzDir(filepath.Join(helpers.GetHomeDir(), ".azure"))
		if err != nil || subID.String() == "00000000-0000-0000-0000-000000000000" {
			return errors.New("--subscription-id is required (and must be a valid UUID)")
		}
		log.Infoln("No subscription provided, using selected subscription from azure CLI:", subID.String())
		authArgs.SubscriptionID = subID
	}

	switch strings.ToUpper(authArgs.RawAzureEnvironment) {
	case "AZURESTACKCLOUD":
		// Azure stack cloud environment, verify file path can be read
		fileName := os.Getenv("AZURE_ENVIRONMENT_FILEPATH")
		if fileContents, err := os.ReadFile(fileName); err != nil ||
			json.Unmarshal(fileContents, &api.Environment{}) != nil {
			return fmt.Errorf("failed to read file or unmarshal JSON from file %s: %v", fileName, err)
		}
	case "AZURECHINACLOUD", "AZUREGERMANCLOUD", "AZUREPUBLICCLOUD", "AZUREUSGOVERNMENTCLOUD":
		// Known environment, no action needed
	default:
		return errors.New("failed to parse --azure-env as a valid target Azure cloud environment")
	}
	return nil
}

func getSubFromAzDir(root string) (uuid.UUID, error) {
	subConfig, err := ini.Load(filepath.Join(root, "clouds.config"))
	if err != nil {
		return uuid.UUID{}, errors.Wrap(err, "error decoding cloud subscription config")
	}

	cloudConfig, err := ini.Load(filepath.Join(root, "config"))
	if err != nil {
		return uuid.UUID{}, errors.Wrap(err, "error decoding cloud config")
	}

	cloud := getSelectedCloudFromAzConfig(cloudConfig)
	return getCloudSubFromAzConfig(cloud, subConfig)
}

func getSelectedCloudFromAzConfig(f *ini.File) string {
	selectedCloud := "AzureCloud"
	if cloud, err := f.GetSection("cloud"); err == nil {
		if name, err := cloud.GetKey("name"); err == nil {
			if s := name.String(); s != "" {
				selectedCloud = s
			}
		}
	}
	return selectedCloud
}

func getCloudSubFromAzConfig(cloud string, f *ini.File) (uuid.UUID, error) {
	cfg, err := f.GetSection(cloud)
	if err != nil {
		return uuid.UUID{}, errors.New("could not find user defined subscription id")
	}
	sub, err := cfg.GetKey("subscription")
	if err != nil {
		return uuid.UUID{}, errors.Wrap(err, "error reading subscription id from cloud config")
	}
	return uuid.Parse(sub.String())
}

func (authArgs *authArgs) getClient(env *api.Environment) (armhelpers.AKSEngineClient, error) {
	var cc cloud.Configuration
	switch authArgs.RawAzureEnvironment {
	case api.AzureUSGovernmentCloud:
		cc = cloud.AzureGovernment
	case api.AzureChinaCloud:
		cc = cloud.AzureChina
	default:
		cc = cloud.AzurePublic
	}
	if authArgs.isAzureStackCloud() {
		if env == nil {
			return nil, errors.New("failed to get azure stack cloud client, API model Properties.CustomCloudProfile.Environment cannot be nil")
		}
		cc = cloud.Configuration{
			ActiveDirectoryAuthorityHost: env.ActiveDirectoryEndpoint,
			Services: map[cloud.ServiceName]cloud.ServiceConfiguration{
				cloud.ResourceManager: {
					Audience: env.ServiceManagementEndpoint,
					Endpoint: env.ResourceManagerEndpoint,
				},
			},
		}
	}
	credential, err := authArgs.getCredentials(cc)
	if err != nil {
		return nil, err
	}
	return authArgs.getAzureClient(credential, cc)
}

func (authArgs *authArgs) getCredentials(env cloud.Configuration) (azcore.TokenCredential, error) {
	if !authArgs.isAzureStackCloud() {
		return armhelpers.NewDefaultCredential(env, authArgs.SubscriptionID.String())
	}
	switch authArgs.AuthMethod {
	case "client_secret":
		if authArgs.IdentitySystem == "azure_ad" {
			return armhelpers.NewClientSecretCredential(env, authArgs.SubscriptionID.String(), authArgs.ClientID.String(), authArgs.ClientSecret)
		} else if authArgs.IdentitySystem == "adfs" {
			return armhelpers.NewClientSecretCredentialExternalTenant(env, authArgs.SubscriptionID.String(), authArgs.ClientID.String(), authArgs.ClientSecret)
		} else {
			return nil, errors.Errorf("--auth-method: ERROR: method unsupported. method=%q identitysystem=%q", authArgs.AuthMethod, authArgs.IdentitySystem)
		}
	case "client_certificate":
		if authArgs.IdentitySystem == "azure_ad" {
			return armhelpers.NewClientCertificateCredential(env, authArgs.SubscriptionID.String(), authArgs.ClientID.String(), authArgs.CertificatePath, authArgs.PrivateKeyPath)
		} else if authArgs.IdentitySystem == "adfs" {
			return armhelpers.NewClientCertificateCredentialExternalTenant(env, authArgs.SubscriptionID.String(), authArgs.ClientID.String(), authArgs.CertificatePath, authArgs.PrivateKeyPath)
		}
		fallthrough
	default:
		return nil, errors.Errorf("--auth-method: ERROR: method unsupported. method=%q identitysystem=%q", authArgs.AuthMethod, authArgs.IdentitySystem)
	}
}

func (authArgs *authArgs) getAzureClient(credential azcore.TokenCredential, env cloud.Configuration) (armhelpers.AKSEngineClient, error) {
	client, err := armhelpers.NewAzureClient(authArgs.SubscriptionID.String(), credential, &arm.ClientOptions{
		ClientOptions: azcore.ClientOptions{
			Cloud: env,
		},
	})
	if err != nil {
		return nil, err
	}
	err = client.EnsureProvidersRegistered(authArgs.SubscriptionID.String())
	if err != nil {
		return nil, err
	}
	client.AddAcceptLanguages([]string{authArgs.language})
	return client, nil
}

func getCompletionCmd(root *cobra.Command) *cobra.Command {
	var completionCmd = &cobra.Command{
		Use:   "completion",
		Short: "Generates bash completion scripts",
		Long: `To load completion run

	source <(aks-engine-azurestack completion)

	To configure your bash shell to load completions for each session, add this to your bashrc

	# ~/.bashrc or ~/.profile
	source <(aks-engine-azurestack completion)
	`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return root.GenBashCompletion(os.Stdout)
		},
	}
	return completionCmd
}

func writeCustomCloudProfile(cs *api.ContainerService) error {

	tmpFile, err := os.CreateTemp("", "azurestackcloud.json")
	tmpFileName := tmpFile.Name()
	if err != nil {
		return err
	}
	log.Infoln(fmt.Sprintf("Writing cloud profile to: %s", tmpFileName))

	// Build content for the file
	content, err := cs.Properties.GetCustomEnvironmentJSON(false)
	if err != nil {
		return err
	}
	if err = os.WriteFile(tmpFileName, []byte(content), os.ModeAppend); err != nil {
		return err
	}

	os.Setenv("AZURE_ENVIRONMENT_FILEPATH", tmpFileName)

	return nil
}

func getKubeClient(cs *api.ContainerService, interval, timeout time.Duration) (kubernetes.Client, error) {
	kubeconfig, err := engine.GenerateKubeConfig(cs.Properties, cs.Location)
	if err != nil {
		return nil, errors.Wrap(err, "generating kubeconfig")
	}
	var az *armhelpers.AzureClient
	client, err := az.GetKubernetesClient("", kubeconfig, interval, timeout)
	if err != nil {
		return nil, err
	}
	return client, nil
}

func writeArtifacts(outputDirectory string, cs *api.ContainerService, apiVersion string, translator *i18n.Translator) error {
	ctx := engine.Context{Translator: translator}
	tplgen, err := engine.InitializeTemplateGenerator(ctx)
	if err != nil {
		return errors.Wrap(err, "initializing template generator")
	}
	tpl, params, err := tplgen.GenerateTemplateV2(cs, engine.DefaultGeneratorCode, BuildTag)
	if err != nil {
		return errors.Wrap(err, "generating template")
	}
	if tpl, err = transform.PrettyPrintArmTemplate(tpl); err != nil {
		return errors.Wrap(err, "pretty-printing template")
	}
	if params, err = transform.BuildAzureParametersFile(params); err != nil {
		return errors.Wrap(err, "pretty-printing template parameters")
	}
	w := &engine.ArtifactWriter{Translator: translator}
	return w.WriteTLSArtifacts(cs, apiVersion, tpl, params, outputDirectory, true, false)
}
