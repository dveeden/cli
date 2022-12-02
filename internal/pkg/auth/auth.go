package auth

import (
	"context"
	"fmt"
	"os"
	"strings"

	ccloudv1 "github.com/confluentinc/ccloud-sdk-go-v1-public"
	"github.com/dghubble/sling"

	v1 "github.com/confluentinc/cli/internal/pkg/config/v1"
	"github.com/confluentinc/cli/internal/pkg/errors"
)

const (
	CCloudURL = "https://confluent.cloud"

	ConfluentCloudEmail          = "CONFLUENT_CLOUD_EMAIL"
	ConfluentCloudPassword       = "CONFLUENT_CLOUD_PASSWORD"
	ConfluentCloudOrganizationId = "CONFLUENT_CLOUD_ORGANIZATION_ID"
	ConfluentPlatformUsername    = "CONFLUENT_PLATFORM_USERNAME"
	ConfluentPlatformPassword    = "CONFLUENT_PLATFORM_PASSWORD"
	ConfluentPlatformMDSURL      = "CONFLUENT_PLATFORM_MDS_URL"
	ConfluentPlatformCACertPath  = "CONFLUENT_PLATFORM_CA_CERT_PATH"

	DeprecatedConfluentCloudEmail         = "CCLOUD_EMAIL"
	DeprecatedConfluentCloudPassword      = "CCLOUD_PASSWORD"
	DeprecatedConfluentPlatformUsername   = "CONFLUENT_USERNAME"
	DeprecatedConfluentPlatformPassword   = "CONFLUENT_PASSWORD"
	DeprecatedConfluentPlatformMDSURL     = "CONFLUENT_MDS_URL"
	DeprecatedConfluentPlatformCACertPath = "CONFLUENT_CA_CERT_PATH"
)

// GetEnvWithFallback calls os.GetEnv() twice, once for the current var and once for the deprecated var.
func GetEnvWithFallback(current, deprecated string) string {
	if val := os.Getenv(current); val != "" {
		return val
	}

	if val := os.Getenv(deprecated); val != "" {
		_, _ = fmt.Fprintf(os.Stderr, errors.DeprecatedEnvVarWarningMsg, deprecated, current)
		return val
	}

	return ""
}

func PersistLogoutToConfig(config *v1.Config) error {
	ctx := config.Context()
	if ctx == nil {
		return nil
	}
	err := ctx.DeleteUserAuth()
	if err != nil {
		return err
	}
	ctx.Config.CurrentContext = ""
	return config.Save()
}

func PersistConfluentLoginToConfig(config *v1.Config, username, url, token, caCertPath string, isLegacyContext bool) error {
	state := &v1.ContextState{AuthToken: token}
	var ctxName string
	if isLegacyContext {
		ctxName = GenerateContextName(username, url, "")
	} else {
		ctxName = GenerateContextName(username, url, caCertPath)
	}
	return addOrUpdateContext(config, ctxName, username, url, state, caCertPath, "")
}

func PersistCCloudCredentialsToConfig(config *v1.Config, client *ccloudv1.Client, url string, credentials *Credentials) (*ccloudv1.Account, *ccloudv1.Organization, error) {
	ctxName := GenerateCloudContextName(credentials.Username, url)
	user, err := client.Auth.User(context.Background())
	if err != nil {
		return nil, nil, err
	}
	state := getCCloudContextState(config, ctxName, credentials.AuthToken, credentials.AuthRefreshToken, user)

	err = addOrUpdateContext(config, ctxName, credentials.Username, url, state, "", user.GetOrganization().GetResourceId())
	return state.Auth.Account, user.GetOrganization(), err
}

func addOrUpdateContext(config *v1.Config, ctxName, username, url string, state *v1.ContextState, caCertPath, orgResourceId string) error {
	credName := generateCredentialName(username)
	platform := &v1.Platform{
		Name:       strings.TrimPrefix(url, "https://"),
		Server:     url,
		CaCertPath: caCertPath,
	}
	credential := &v1.Credential{
		Name:     credName,
		Username: username,
		// don't save password if they entered it interactively.
	}

	if err := config.SavePlatform(platform); err != nil {
		return err
	}

	if err := config.SaveCredential(credential); err != nil {
		return err
	}

	if ctx, ok := config.Contexts[ctxName]; ok {
		config.ContextStates[ctxName] = state
		ctx.State = state

		ctx.Platform = platform
		ctx.PlatformName = platform.Name

		ctx.Credential = credential
		ctx.CredentialName = credential.Name
		ctx.LastOrgId = orgResourceId
	} else {
		if err := config.AddContext(ctxName, platform.Name, credential.Name, map[string]*v1.KafkaClusterConfig{}, "", nil, state, orgResourceId); err != nil {
			return err
		}
	}

	return config.UseContext(ctxName)
}

func getCCloudContextState(config *v1.Config, ctxName, token, refreshToken string, user *ccloudv1.GetMeReply) *v1.ContextState {
	state := new(v1.ContextState)
	if ctx, err := config.FindContext(ctxName); err == nil {
		state = ctx.State
	}

	if state.Auth == nil {
		state.Auth = &v1.AuthConfig{}
	}
	state.AuthToken = token
	state.AuthRefreshToken = refreshToken

	// Always overwrite the user, organization, and list of accounts when logging in -- but don't necessarily
	// overwrite `Account` (current/active environment) since we want that to be remembered
	// between CLI sessions.
	state.Auth.User = user.GetUser()
	state.Auth.Accounts = user.GetAccounts()
	state.Auth.Organization = user.GetOrganization()

	// Default to 0th environment if no suitable environment is already configured
	hasGoodEnv := false
	if state.Auth.Account != nil {
		for _, account := range state.Auth.Accounts {
			if account.Id == state.Auth.Account.Id {
				hasGoodEnv = true
			}
		}
	}
	if !hasGoodEnv && len(state.Auth.Accounts) > 0 {
		state.Auth.Account = state.Auth.Accounts[0]
	}

	return state
}

func GenerateCloudContextName(username string, url string) string {
	return GenerateContextName(username, url, "")
}

// if CP users use cacertpath then include that in the context name
// (legacy CP users may still have context without cacertpath in the name but have cacertpath stored)
func GenerateContextName(username string, url string, caCertPath string) string {
	if caCertPath == "" {
		return fmt.Sprintf("login-%s-%s", username, url)
	}
	return fmt.Sprintf("login-%s-%s?cacertpath=%s", username, url, caCertPath)
}

func generateCredentialName(username string) string {
	return fmt.Sprintf("username-%s", username)
}

type response struct {
	Error string `json:"error"`
	Token string `json:"token"`
}

func GetBearerToken(authenticatedState *v1.ContextState, server, clusterId string) (string, error) {
	bearerSessionToken := "Bearer " + authenticatedState.AuthToken
	accessTokenEndpoint := strings.Trim(server, "/") + "/api/access_tokens"
	clusterIds := map[string][]string{"clusterIds": {clusterId}}

	// Configure and send post request with session token to Auth Service to get access token
	responses := new(response)
	_, err := sling.New().Add("content", "application/json").Add("Content-Type", "application/json").Add("Authorization", bearerSessionToken).BodyJSON(clusterIds).Post(accessTokenEndpoint).ReceiveSuccess(responses)
	if err != nil {
		return "", err
	}
	return responses.Token, nil
}

func GetJwtTokenForV2Client(authenticatedState *v1.ContextState, server string) (string, error) {
	bearerSessionToken := "Bearer " + authenticatedState.AuthToken
	accessTokenEndpoint := strings.Trim(server, "/") + "/api/access_tokens"

	responses := new(response)
	_, err := sling.New().Add("content", "application/json").Add("Content-Type", "application/json").Add("Authorization", bearerSessionToken).Body(strings.NewReader("{}")).Post(accessTokenEndpoint).ReceiveSuccess(responses)
	if err != nil {
		return "", err
	}
	if responses.Error != "" {
		return "", errors.New(responses.Error)
	}
	return responses.Token, nil
}
