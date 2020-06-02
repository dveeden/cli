package auth

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	pauth "github.com/confluentinc/cli/internal/pkg/auth"
	"math/big"
	"net/http"
	"os"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"

	orgv1 "github.com/confluentinc/cc-structs/kafka/org/v1"
	"github.com/confluentinc/ccloud-sdk-go"
	sdkMock "github.com/confluentinc/ccloud-sdk-go/mock"

	mds "github.com/confluentinc/mds-sdk-go/mdsv1"
	mdsMock "github.com/confluentinc/mds-sdk-go/mdsv1/mock"

	pcmd "github.com/confluentinc/cli/internal/pkg/cmd"
	"github.com/confluentinc/cli/internal/pkg/config"
	v0 "github.com/confluentinc/cli/internal/pkg/config/v0"
	v1 "github.com/confluentinc/cli/internal/pkg/config/v1"
	v3 "github.com/confluentinc/cli/internal/pkg/config/v3"
	"github.com/confluentinc/cli/internal/pkg/log"
	cliMock "github.com/confluentinc/cli/mock"
)

func TestCredentialsOverride(t *testing.T) {
	req := require.New(t)
	currentEmail := os.Getenv("XX_CCLOUD_EMAIL")
	currentPassword := os.Getenv("XX_CCLOUD_PASSWORD")

	os.Setenv("XX_CCLOUD_EMAIL", "test-email")
	os.Setenv("XX_CCLOUD_PASSWORD", "test-password")

	prompt := prompt("cody@confluent.io", "iambatman")
	auth := &sdkMock.Auth{
		LoginFunc: func(ctx context.Context, idToken string, username string, password string) (string, error) {
			return "y0ur.jwt.T0kEn", nil
		},
		UserFunc: func(ctx context.Context) (*orgv1.GetUserReply, error) {
			return &orgv1.GetUserReply{
				User: &orgv1.User{
					Id:        23,
					Email:     "test-email",
					FirstName: "Cody",
				},
				Accounts: []*orgv1.Account{{Id: "a-595", Name: "Default"}},
			}, nil
		},
	}
	user := &sdkMock.User{
		CheckEmailFunc: func(ctx context.Context, user *orgv1.User) (*orgv1.User, error) {
			return &orgv1.User{
				Email: "test-email",
			}, nil
		},
	}
	cmds, cfg := newAuthCommand(prompt, auth, user, "ccloud", req)

	output, err := pcmd.ExecuteCommand(cmds.Commands[0].Command)
	req.NoError(err)
	req.Contains(output, "Logged in as test-email")
	ctx := cfg.Context()
	req.NotNil(ctx)

	req.Equal("y0ur.jwt.T0kEn", ctx.State.AuthToken)
	req.Equal(&orgv1.User{Id: 23, Email: "test-email", FirstName: "Cody"}, ctx.State.Auth.User)

	os.Setenv("XX_CCLOUD_EMAIL", currentEmail)
	os.Setenv("XX_CCLOUD_PASSWORD", currentPassword)
}

func TestLoginSuccess(t *testing.T) {
	req := require.New(t)

	prompt := prompt("cody@confluent.io", "iambatman")
	auth := &sdkMock.Auth{
		LoginFunc: func(ctx context.Context, idToken string, username string, password string) (string, error) {
			return "y0ur.jwt.T0kEn", nil
		},
		UserFunc: func(ctx context.Context) (*orgv1.GetUserReply, error) {
			return &orgv1.GetUserReply{
				User: &orgv1.User{
					Id:        23,
					Email:     "cody@confluent.io",
					FirstName: "Cody",
				},
				Accounts: []*orgv1.Account{{Id: "a-595", Name: "Default"}},
			}, nil
		},
	}
	user := &sdkMock.User{
		CheckEmailFunc: func(ctx context.Context, user *orgv1.User) (*orgv1.User, error) {
			return &orgv1.User{
				Email: "test-email",
			}, nil
		},
	}

	suite := []struct {
		cliName string
		args    []string
	}{
		{
			cliName: "ccloud",
			args:    []string{},
		},
		{
			cliName: "confluent",
			args: []string{
				"--url=http://localhost:8090",
			},
		},
	}

	for _, s := range suite {
		// Login to the CLI control plane
		cmds, cfg := newAuthCommand(prompt, auth, user, s.cliName, req)
		output, err := pcmd.ExecuteCommand(cmds.Commands[LoginIndex].Command, s.args...)
		req.NoError(err)
		req.Contains(output, "Logged in as cody@confluent.io")
		verifyLoggedInState(t, cfg, s.cliName)
	}
}

func TestLoginFail(t *testing.T) {
	req := require.New(t)

	prompt := prompt("cody@confluent.io", "iamrobin")
	auth := &sdkMock.Auth{
		LoginFunc: func(ctx context.Context, idToken string, username string, password string) (string, error) {
			return "", &ccloud.InvalidLoginError{}
		},
	}
	user := &sdkMock.User{
		CheckEmailFunc: func(ctx context.Context, user *orgv1.User) (*orgv1.User, error) {
			return &orgv1.User{
				Email: "test-email",
			}, nil
		},
	}
	cmds, _ := newAuthCommand(prompt, auth, user, "ccloud", req)

	_, err := pcmd.ExecuteCommand(cmds.Commands[0].Command)
	req.Contains(err.Error(), "You have entered an incorrect username or password.")
}

func TestURLRequiredWithMDS(t *testing.T) {
	req := require.New(t)

	prompt := prompt("cody@confluent.io", "iamrobin")
	auth := &sdkMock.Auth{
		LoginFunc: func(ctx context.Context, idToken string, username string, password string) (string, error) {
			return "", &ccloud.InvalidLoginError{}
		},
	}
	cmds, _ := newAuthCommand(prompt, auth, nil, "confluent", req)

	_, err := pcmd.ExecuteCommand(cmds.Commands[0].Command)
	req.Contains(err.Error(), "required flag(s) \"url\" not set")
}

func TestLogout(t *testing.T) {
	req := require.New(t)

	prompt := prompt("cody@confluent.io", "iamrobin")
	auth := &sdkMock.Auth{}
	cmds, _ := newAuthCommand(prompt, auth, nil, "ccloud", req)
	cmds.config = v3.AuthenticatedCloudConfigMock()
	output, err := pcmd.ExecuteCommand(cmds.Commands[1].Command)
	req.NoError(err)
	req.Contains(output, "You are now logged out")
	verifyLoggedOutState(t, cmds.config)
}

func Test_credentials_NoSpacesAroundEmail_ShouldSupportSpacesAtBeginOrEnd(t *testing.T) {
	req := require.New(t)

	prompt := prompt(" cody@confluent.io ", " iamrobin ")
	auth := &sdkMock.Auth{}
	cmds, _ := newAuthCommand(prompt, auth, nil, "ccloud", req)

	user, pass, err := cmds.credentials(cmds.Commands[0].Command, "Email", nil)
	req.NoError(err)
	req.Equal("cody@confluent.io", user)
	req.Equal(" iamrobin ", pass)
}

func Test_SelfSignedCerts(t *testing.T) {
	req := require.New(t)
	mdsConfig := mds.NewConfiguration()
	mdsClient := mds.NewAPIClient(mdsConfig)
	cfg := v3.New(&config.Params{
		CLIName:    "confluent",
		MetricSink: nil,
		Logger:     log.New(),
	})
	prompt := prompt("cody@confluent.io", "iambatman")
	prerunner := cliMock.NewPreRunnerMock(nil, nil)

	// Create a test certificate to be read in by the command
	ca := &x509.Certificate{
		SerialNumber: big.NewInt(1234),
		Subject:      pkix.Name{Organization: []string{"testorg"}},
	}
	priv, err := rsa.GenerateKey(rand.Reader, 512)
	req.NoError(err, "Couldn't generate private key")
	ca_b, err := x509.CreateCertificate(rand.Reader, ca, ca, &priv.PublicKey, priv)
	req.NoError(err, "Couldn't generate certificate from private key")
	pemBytes := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: ca_b})
	certReader := bytes.NewReader(pemBytes)

	cert, err := x509.ParseCertificate(ca_b)
	req.NoError(err, "Couldn't reparse certificate")
	expectedSubject := cert.RawSubject
	mdsClient.TokensAndAuthenticationApi = &mdsMock.TokensAndAuthenticationApi{
		GetTokenFunc: func(ctx context.Context) (mds.AuthenticationResponse, *http.Response, error) {
			req.NotEqual(http.DefaultClient, mdsClient)
			transport, ok := mdsClient.GetConfig().HTTPClient.Transport.(*http.Transport)
			req.True(ok)
			req.NotEqual(http.DefaultTransport, transport)
			found := false
			for _, actualSubject := range transport.TLSClientConfig.RootCAs.Subjects() {
				if bytes.Equal(expectedSubject, actualSubject) {
					found = true
					break
				}
			}
			req.True(found, "Certificate not found in client.")
			return mds.AuthenticationResponse{
				AuthToken: "y0ur.jwt.T0kEn",
				TokenType: "JWT",
				ExpiresIn: 100,
			}, nil, nil
		},
	}
	mdsClientManager := &cliMock.MockMDSClientManager{
		GetMDSClientFunc: func(ctx *v3.Context, caCertPath string, flagChanged bool, url string, logger *log.Logger) (client *mds.APIClient, e error) {
			mdsClient.GetConfig().HTTPClient, err = pauth.SelfSignedCertClient(certReader, logger)
			if err != nil {
				return nil, err
			}
			return mdsClient, nil
		},
	}
	cmds := newCommands(prerunner, cfg, log.New(), prompt, nil, nil, mdsClientManager, cliMock.NewDummyAnalyticsMock(), nil)
	for _, c := range cmds.Commands {
		c.PersistentFlags().CountP("verbose", "v", "Increase output verbosity")
	}
	_, err = pcmd.ExecuteCommand(cmds.Commands[0].Command, "--url=http://localhost:8090", "--ca-cert-path=testcert.pem")
	req.NoError(err)
}

func TestLoginWithExistingContext(t *testing.T) {
	req := require.New(t)

	prompt := prompt("cody@confluent.io", "iambatman")
	auth := &sdkMock.Auth{
		LoginFunc: func(ctx context.Context, idToken string, username string, password string) (string, error) {
			return "y0ur.jwt.T0kEn", nil
		},
		UserFunc: func(ctx context.Context) (*orgv1.GetUserReply, error) {
			return &orgv1.GetUserReply{
				User: &orgv1.User{
					Id:        23,
					Email:     "cody@confluent.io",
					FirstName: "Cody",
				},
				Accounts: []*orgv1.Account{{Id: "a-595", Name: "Default"}},
			}, nil
		},
	}
	user := &sdkMock.User{
		CheckEmailFunc: func(ctx context.Context, user *orgv1.User) (*orgv1.User, error) {
			return &orgv1.User{
				Email: "test-email",
			}, nil
		},
	}

	suite := []struct {
		cliName string
		args    []string
	}{
		{
			cliName: "ccloud",
			args:    []string{},
		},
		{
			cliName: "confluent",
			args: []string{
				"--url=http://localhost:8090",
			},
		},
	}

	activeApiKey := "bo"
	kafkaCluster := &v1.KafkaClusterConfig{
		ID:          "lkc-0000",
		Name:        "bob",
		Bootstrap:   "http://bobby",
		APIEndpoint: "http://bobbyboi",
		APIKeys: map[string]*v0.APIKeyPair{
			activeApiKey: {
				Key:    activeApiKey,
				Secret: "bo",
			},
		},
		APIKey: activeApiKey,
	}

	for _, s := range suite {
		cmds, cfg := newAuthCommand(prompt, auth, user, s.cliName, req)

		// Login to the CLI control plane
		output, err := pcmd.ExecuteCommand(cmds.Commands[LoginIndex].Command, s.args...)
		req.NoError(err)
		req.Contains(output, "Logged in as cody@confluent.io")
		verifyLoggedInState(t, cfg, s.cliName)

		// Set kafka related states for the logged in context
		ctx := cfg.Context()
		ctx.KafkaClusterContext.AddKafkaClusterConfig(kafkaCluster)
		ctx.KafkaClusterContext.SetActiveKafkaCluster(kafkaCluster.ID)

		// Executing logout
		output, err = pcmd.ExecuteCommand(cmds.Commands[1].Command)
		req.NoError(err)
		req.Contains(output, "You are now logged out")
		verifyLoggedOutState(t, cfg)

		// logging back in the the same context
		output, err = pcmd.ExecuteCommand(cmds.Commands[LoginIndex].Command, s.args...)
		req.NoError(err)
		req.Contains(output, "Logged in as cody@confluent.io")
		verifyLoggedInState(t, cfg, s.cliName)

		// verify that kafka cluster info persists between logging back in again
		req.Equal(kafkaCluster.ID, ctx.KafkaClusterContext.GetActiveKafkaClusterId())
		reflect.DeepEqual(kafkaCluster, ctx.KafkaClusterContext.GetKafkaClusterConfig(kafkaCluster.ID))
	}
}

func verifyLoggedInState(t *testing.T, cfg *v3.Config, cliName string) {
	req := require.New(t)
	ctx := cfg.Context()
	req.NotNil(ctx)
	req.Equal("y0ur.jwt.T0kEn", ctx.State.AuthToken)
	contextName := fmt.Sprintf("login-cody@confluent.io-%s", ctx.Platform.Server)
	credName := fmt.Sprintf("username-%s", ctx.Credential.Username)
	req.Contains(cfg.Platforms, ctx.Platform.Name)
	req.Equal(ctx.Platform, cfg.Platforms[ctx.PlatformName])
	req.Contains(cfg.Credentials, credName)
	req.Equal("cody@confluent.io", cfg.Credentials[credName].Username)
	req.Contains(cfg.Contexts, contextName)
	req.Equal(ctx.Platform, cfg.Contexts[contextName].Platform)
	req.Equal(ctx.Credential, cfg.Contexts[contextName].Credential)
	if cliName == "ccloud" {
		// MDS doesn't set some things like cfg.Auth.User since e.g. an MDS user != an orgv1 (ccloud) User
		req.Equal(&orgv1.User{Id: 23, Email: "cody@confluent.io", FirstName: "Cody"}, ctx.State.Auth.User)
	} else {
		req.Equal("http://localhost:8090", ctx.Platform.Server)
	}
}

func verifyLoggedOutState(t *testing.T, cfg *v3.Config) {
	req := require.New(t)
	state := cfg.Context().State
	req.Empty(state.AuthToken)
	req.Empty(state.Auth)
}

func prompt(username, password string) *cliMock.Prompt {
	return &cliMock.Prompt{
		ReadStringFunc: func(delim byte) (string, error) {
			return "cody@confluent.io", nil
		},
		ReadPasswordFunc: func() ([]byte, error) {
			return []byte(" iamrobin "), nil
		},
	}
}

func newAuthCommand(prompt pcmd.Prompt, auth *sdkMock.Auth, user *sdkMock.User, cliName string, req *require.Assertions) (*commands, *v3.Config) {
	var mockAnonHTTPClientFactory = func(baseURL string, logger *log.Logger) *ccloud.Client {
		req.Equal("https://confluent.cloud", baseURL)
		return &ccloud.Client{Auth: auth, User: user}
	}
	var mockJwtHTTPClientFactory = func(ctx context.Context, jwt, baseURL string, logger *log.Logger) *ccloud.Client {
		return &ccloud.Client{Auth: auth, User: user}
	}
	cfg := v3.New(&config.Params{
		CLIName:    cliName,
		MetricSink: nil,
		Logger:     nil,
	})
	var mdsClient *mds.APIClient
	if cliName == "confluent" {
		mdsConfig := mds.NewConfiguration()
		mdsClient = mds.NewAPIClient(mdsConfig)
		mdsClient.TokensAndAuthenticationApi = &mdsMock.TokensAndAuthenticationApi{
			GetTokenFunc: func(ctx context.Context) (mds.AuthenticationResponse, *http.Response, error) {
				return mds.AuthenticationResponse{
					AuthToken: "y0ur.jwt.T0kEn",
					TokenType: "JWT",
					ExpiresIn: 100,
				}, nil, nil
			},
		}
	}
	mdsClientManager := &cliMock.MockMDSClientManager{
		GetMDSClientFunc: func(ctx *v3.Context, caCertPath string, flagChanged bool, url string, logger *log.Logger) (client *mds.APIClient, e error) {
			return mdsClient, nil
		},
	}
	commands := newCommands(cliMock.NewPreRunnerMock(mockAnonHTTPClientFactory("https://confluent.cloud", nil), mdsClient),
		cfg, log.New(), prompt, mockAnonHTTPClientFactory, mockJwtHTTPClientFactory, mdsClientManager,
		cliMock.NewDummyAnalyticsMock(), nil)
	for _, c := range commands.Commands {
		c.PersistentFlags().CountP("verbose", "v", "Increase output verbosity")
	}
	return commands, cfg
}
