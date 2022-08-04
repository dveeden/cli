package schemaregistry

import (
	"context"
	"os"

	srsdk "github.com/confluentinc/schema-registry-sdk-go"
	"github.com/spf13/cobra"

	dynamicconfig "github.com/confluentinc/cli/internal/pkg/dynamic-config"
	"github.com/confluentinc/cli/internal/pkg/log"

	pauth "github.com/confluentinc/cli/internal/pkg/auth"
	v1 "github.com/confluentinc/cli/internal/pkg/config/v1"
	"github.com/confluentinc/cli/internal/pkg/errors"
	"github.com/confluentinc/cli/internal/pkg/form"
	"github.com/confluentinc/cli/internal/pkg/utils"
	"github.com/confluentinc/cli/internal/pkg/version"
)

func promptSchemaRegistryCredentials(command *cobra.Command) (string, string, error) {
	f := form.New(
		form.Field{ID: "api-key", Prompt: "Enter your Schema Registry API key"},
		form.Field{ID: "secret", Prompt: "Enter your Schema Registry API secret", IsHidden: true},
	)
	if err := f.Prompt(command, form.NewPrompt(os.Stdin)); err != nil {
		return "", "", err
	}
	return f.Responses["api-key"].(string), f.Responses["secret"].(string), nil
}

func getSchemaRegistryAuth(cmd *cobra.Command, srCredentials *v1.APIKeyPair, shouldPrompt bool) (*srsdk.BasicAuth, bool, error) {
	auth := &srsdk.BasicAuth{}
	didPromptUser := false

	if srCredentials != nil {
		auth.UserName = srCredentials.Key
		auth.Password = srCredentials.Secret
	}

	if auth.UserName == "" || auth.Password == "" || shouldPrompt {
		var err error
		auth.UserName, auth.Password, err = promptSchemaRegistryCredentials(cmd)
		if err != nil {
			return nil, false, err
		}
		didPromptUser = true
	}

	return auth, didPromptUser, nil
}

func getApiClient(cmd *cobra.Command, srClient *srsdk.APIClient, cfg *dynamicconfig.DynamicConfig, ver *version.Version) (*srsdk.APIClient, context.Context, error) {
	if srClient != nil {
		// Tests/mocks
		return srClient, nil, nil
	}
	return GetSchemaRegistryClientWithApiKey(cmd, cfg, ver, "", "")
}

func GetSrApiClientWithToken(cmd *cobra.Command, ver *version.Version, mdsToken string) (*srsdk.APIClient, context.Context, error) {
	return getSchemaRegistryClientWithToken(cmd, ver, mdsToken)
}

func GetSchemaRegistryClientWithApiKey(cmd *cobra.Command, cfg *dynamicconfig.DynamicConfig, ver *version.Version, srAPIKey, srAPISecret string) (*srsdk.APIClient, context.Context, error) {
	srConfig := srsdk.NewConfiguration()
	srConfig.Debug = log.CliLogger.Level >= log.DEBUG

	ctx := cfg.Context()

	srCluster := &v1.SchemaRegistryCluster{}
	endpoint, _ := cmd.Flags().GetString("sr-endpoint")
	if len(endpoint) == 0 {
		cluster, err := ctx.SchemaRegistryCluster(cmd)
		if err != nil {
			log.CliLogger.Debug("failed to find active schema registry cluster")
			return nil, nil, err
		}
		srCluster = cluster
	}
	// Check if --api-key and --api-secret flags were set, if so, insert them as the credentials for the sr cluster
	key, secret, err := ctx.KeyAndSecretFlags(cmd)
	if err != nil {
		return nil, nil, err
	}
	if key != "" {
		if srCluster.SrCredentials == nil {
			srCluster.SrCredentials = &v1.APIKeyPair{}
		}
		srCluster.SrCredentials.Key = key
		if secret != "" {
			srCluster.SrCredentials.Secret = secret
		}
	}

	// First examine existing credentials. If check fails(saved credentials no longer works or user enters
	// incorrect information), shouldPrompt becomes true and prompt users to enter credentials again.
	shouldPrompt := false

	for {
		// Get credentials as Schema Registry BasicAuth
		if srAPIKey != "" && srAPISecret != "" {
			srCluster.SrCredentials = &v1.APIKeyPair{
				Key:    srAPIKey,
				Secret: srAPISecret,
			}
		} else if srAPISecret != "" {
			utils.ErrPrintln(cmd, "No schema registry API key specified.")
		} else if srAPIKey != "" {
			utils.ErrPrintln(cmd, "No schema registry API key secret specified.")
		}
		srAuth, didPromptUser, err := getSchemaRegistryAuth(cmd, srCluster.SrCredentials, shouldPrompt)
		if err != nil {
			return nil, nil, err
		}
		srCtx := context.WithValue(context.Background(), srsdk.ContextBasicAuth, *srAuth)

		if len(endpoint) == 0 {
			envId, err := ctx.AuthenticatedEnvId()
			if err != nil {
				return nil, nil, err
			}

			if srCluster, ok := ctx.SchemaRegistryClusters[envId]; ok {
				srConfig.BasePath = srCluster.SchemaRegistryEndpoint
			} else {
				srCluster, err := ctx.FetchSchemaRegistryByAccountId(srCtx, envId)
				if err != nil {
					return nil, nil, err
				}
				srConfig.BasePath = srCluster.Endpoint
			}
		} else {
			srConfig.BasePath = endpoint
		}
		srConfig.UserAgent = ver.UserAgent
		srConfig.HTTPClient = utils.DefaultClient()
		srClient := srsdk.NewAPIClient(srConfig)

		// Test credentials
		if _, _, err = srClient.DefaultApi.Get(srCtx); err != nil {
			utils.ErrPrintln(cmd, errors.SRCredsValidationFailedErrorMsg)
			// Prompt users to enter new credentials if validation fails.
			shouldPrompt = true
			continue
		}

		if didPromptUser {
			// Save credentials
			srCluster.SrCredentials = &v1.APIKeyPair{
				Key:    srAuth.UserName,
				Secret: srAuth.Password,
			}
			if err := ctx.Save(); err != nil {
				return nil, nil, err
			}
		}

		return srClient, srCtx, nil
	}
}

func getSchemaRegistryClientWithToken(cmd *cobra.Command, ver *version.Version, mdsToken string) (*srsdk.APIClient, context.Context, error) {
	srConfig := srsdk.NewConfiguration()

	caCertPath, err := cmd.Flags().GetString("ca-location")
	if err != nil {
		return nil, nil, err
	}
	if caCertPath == "" {
		caCertPath = pauth.GetEnvWithFallback(pauth.ConfluentPlatformCACertPath, pauth.DeprecatedConfluentPlatformCACertPath)
	}
	endpoint, err := cmd.Flags().GetString("sr-endpoint")
	if err != nil {
		return nil, nil, err
	}
	if len(endpoint) == 0 {
		return nil, nil, errors.New(errors.SREndpointNotSpecifiedErrorMsg)
	}

	srCtx := context.WithValue(context.Background(), srsdk.ContextAccessToken, mdsToken)

	srConfig.BasePath = endpoint
	srConfig.UserAgent = ver.UserAgent
	srConfig.HTTPClient, err = utils.GetCAClient(caCertPath)
	if err != nil {
		return nil, nil, err
	}
	srClient := srsdk.NewAPIClient(srConfig)

	if _, _, err = srClient.DefaultApi.Get(srCtx); err != nil { // validate client
		return nil, nil, errors.New(errors.SRClientNotValidatedErrorMsg)
	}
	return srClient, srCtx, nil
}
