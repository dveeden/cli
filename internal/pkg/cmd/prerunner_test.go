package cmd_test

import (
	"fmt"
	v0 "github.com/confluentinc/cli/internal/pkg/config/v0"
	"github.com/spf13/pflag"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/jonboulle/clockwork"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"

	"github.com/confluentinc/cli/internal/pkg/auth"
	pcmd "github.com/confluentinc/cli/internal/pkg/cmd"
	"github.com/confluentinc/cli/internal/pkg/config/load"
	v3 "github.com/confluentinc/cli/internal/pkg/config/v3"
	"github.com/confluentinc/cli/internal/pkg/errors"
	"github.com/confluentinc/cli/internal/pkg/log"
	pmock "github.com/confluentinc/cli/internal/pkg/mock"
	"github.com/confluentinc/cli/internal/pkg/update/mock"
	cliMock "github.com/confluentinc/cli/mock"
)

var (
	expiredAuthTokenForDevCLoud = "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJvcmdhbml6YXRpb25JZCI6MT" +
		"U5NCwidXNlcklkIjoxNTM3MiwiZXhwIjoxNTc0NzIwODgzLCJqdGkiOiJkMzFlYjc2OC0zNzIzLTQ4MTEtYjg3" +
		"Zi1lMTQ2YTQyYmMyMjciLCJpYXQiOjE1NzQ3MTcyODMsImlzcyI6IkNvbmZsdWVudCIsInN1YiI6IjE1MzcyIn" +
		"0.r9o6HEaacidXV899sjYDajCfVd_Tczyfk541jzidw8r0TRGz74RxL2UFK0aGyR4tNrJRSOJlYHSEBNMV7" +
		"J1sEzdGj_mYbvdAL8feH3Sj0uOf1BSKEdhOLsZbQRPn1TnUwUI0ujxjvY3V4l9unXjdRcNceQx1RcAIm8JEo" +
		"Vjpgsb5MRQWYHlTTEwJe5MVY-dZZEsq40YzlchmFi8LVYCxY3rtwEtINbFJx7K-0rW-GJWyek2zRMiUDtmXI" +
		"o8C87TmR9JfLAhLGYKYB-sMnX1FWQs1GSEf9CBGerhZ6T4wwTu_GCVEqg_kDZpGxx1V3nTr0K_lHN8QxFHoJA" +
		"ccbtRHKFuQZaXkJjhsq4i6q9OV-wgL_G7y003Z-hRiBvdBPoEqecXOfI6HKYbzfv9P9N2p0UnfPF2fZWitcmd" +
		"55IgHZ15zwDkFqixoV1hY_tG7dWtQNZIlPDabgm5UH0mc7GS2dh9Z5spZTvqH8xZ0SFF6T5-iFqpJjm6wkzMd6" +
		"1u9UuWTTTNG-Nr_8abS0cYfChZIXde3D1so2KhG4r6uAB1onlNWK4Gq2Lc9uT_r2tKcGDqyZWFPvVtAepr8duW" +
		"ts27QsDs7BvMnwSkUjGv6scSJZWX1fMZbXh7zd0Khg_13dWshAyE935n46T4S7VJm9JhZLEwUcoOPOhWmVcJn5xSJ-YQ"
	validAuthToken = "eyJ0eXAiOiJKV1QiLCJhbGciOiJIUzI1NiJ9.eyJpc3MiOiJPbmxpbmUgSldUIEJ1aWxkZXIiLCJpYXQiO" +
		"jE1NjE2NjA4NTcsImV4cCI6MjUzMzg2MDM4NDU3LCJhdWQiOiJ3d3cuZXhhbXBsZS5jb20iLCJzdWIiOiJqcm9ja2V0QGV4YW1w" +
		"bGUuY29tIn0.G6IgrFm5i0mN7Lz9tkZQ2tZvuZ2U7HKnvxMuZAooPmE"
)

func TestPreRun_Anonymous_SetLoggingLevel(t *testing.T) {
	type fields struct {
		Logger  *log.Logger
		Command string
	}
	tests := []struct {
		name   string
		fields fields
		want   log.Level
	}{
		{
			name: "default logging level",
			fields: fields{
				Logger:  log.New(),
				Command: "help",
			},
			want: log.ERROR,
		},
		{
			name: "warn logging level",
			fields: fields{
				Logger:  log.New(),
				Command: "help -v",
			},
			want: log.WARN,
		},
		{
			name: "info logging level",
			fields: fields{
				Logger:  log.New(),
				Command: "help -vv",
			},
			want: log.INFO,
		},
		{
			name: "debug logging level",
			fields: fields{
				Logger:  log.New(),
				Command: "help -vvv",
			},
			want: log.DEBUG,
		},
		{
			name: "trace logging level",
			fields: fields{
				Logger:  log.New(),
				Command: "help -vvvv",
			},
			want: log.TRACE,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ver := pmock.NewVersionMock()
			cfg := v3.New(nil)
			cfg, err := load.LoadAndMigrate(cfg)
			require.NoError(t, err)
			r := &pcmd.PreRun{
				Version: ver,
				Logger:  tt.fields.Logger,
				UpdateClient: &mock.Client{
					CheckForUpdatesFunc: func(n, v string, f bool) (bool, string, error) {
						return false, "", nil
					},
				},
				FlagResolver: &pcmd.FlagResolverImpl{
					Prompt: &pcmd.RealPrompt{},
					Out:    os.Stdout,
				},
				Analytics:          cliMock.NewDummyAnalyticsMock(),
				Clock:              clockwork.NewRealClock(),
				UpdateTokenHandler: auth.NewUpdateTokenHandler(auth.NewNetrcHandler("")),
				Config:             cfg,
			}

			root := &cobra.Command{Run: func(cmd *cobra.Command, args []string) {}}
			root.Flags().CountP("verbose", "v", "Increase verbosity")
			rootCmd := pcmd.NewAnonymousCLICommand(root, r)

			args := strings.Split(tt.fields.Command, " ")
			_, err = pcmd.ExecuteCommand(rootCmd.Command, args...)
			require.NoError(t, err)

			got := tt.fields.Logger.GetLevel()
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("PreRun.HasAPIKey() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPreRun_HasAPIKey_SetupLoggingAndCheckForUpdates(t *testing.T) {
	ver := pmock.NewVersionMock()

	calledAnonymous := false
	r := &pcmd.PreRun{
		Version: ver,
		Logger:  log.New(),
		UpdateClient: &mock.Client{
			CheckForUpdatesFunc: func(n, v string, f bool) (bool, string, error) {
				calledAnonymous = true
				return false, "", nil
			},
		},
		FlagResolver: &pcmd.FlagResolverImpl{
			Prompt: &pcmd.RealPrompt{},
			Out:    os.Stdout,
		},
		Analytics:          cliMock.NewDummyAnalyticsMock(),
		Clock:              clockwork.NewRealClock(),
		UpdateTokenHandler: auth.NewUpdateTokenHandler(auth.NewNetrcHandler("")),
	}

	root := &cobra.Command{Run: func(cmd *cobra.Command, args []string) {}}
	root.Flags().CountP("verbose", "v", "Increase verbosity")
	rootCmd := pcmd.NewAnonymousCLICommand(root, r)
	args := strings.Split("help", " ")
	_, err := pcmd.ExecuteCommand(rootCmd.Command, args...)
	require.NoError(t, err)

	if !calledAnonymous {
		t.Errorf("PreRun.HasAPIKey() didn't call the Anonymous() helper to set logging level and updates")
	}
}

func TestPreRun_CallsAnalyticsTrackCommand(t *testing.T) {
	ver := pmock.NewVersionMock()
	analyticsClient := cliMock.NewDummyAnalyticsMock()

	r := &pcmd.PreRun{
		Version: ver,
		Logger:  log.New(),
		UpdateClient: &mock.Client{
			CheckForUpdatesFunc: func(n, v string, f bool) (bool, string, error) {
				return false, "", nil
			},
		},
		FlagResolver: &pcmd.FlagResolverImpl{
			Prompt: &pcmd.RealPrompt{},
			Out:    os.Stdout,
		},
		Analytics:          analyticsClient,
		Clock:              clockwork.NewRealClock(),
		UpdateTokenHandler: auth.NewUpdateTokenHandler(auth.NewNetrcHandler("")),
	}

	root := &cobra.Command{
		Run: func(cmd *cobra.Command, args []string) {},
	}
	rootCmd := pcmd.NewAnonymousCLICommand(root, r)
	root.Flags().CountP("verbose", "v", "Increase verbosity")

	_, err := pcmd.ExecuteCommand(rootCmd.Command)
	require.NoError(t, err)

	require.True(t, analyticsClient.TrackCommandCalled())
}

func TestPreRun_TokenExpires(t *testing.T) {
	cfg := v3.AuthenticatedCloudConfigMock()
	cfg.Context().State.AuthToken = expiredAuthTokenForDevCLoud

	ver := pmock.NewVersionMock()
	analyticsClient := cliMock.NewDummyAnalyticsMock()

	r := &pcmd.PreRun{
		Version: ver,
		Logger:  log.New(),
		UpdateClient: &mock.Client{
			CheckForUpdatesFunc: func(n, v string, f bool) (bool, string, error) {
				return false, "", nil
			},
		},
		FlagResolver: &pcmd.FlagResolverImpl{
			Prompt: &pcmd.RealPrompt{},
			Out:    os.Stdout,
		},
		Analytics:          analyticsClient,
		Clock:              clockwork.NewRealClock(),
		UpdateTokenHandler: auth.NewUpdateTokenHandler(auth.NewNetrcHandler("")),
		Config:             cfg,
	}

	root := &cobra.Command{
		Run: func(cmd *cobra.Command, args []string) {},
	}
	rootCmd := pcmd.NewAnonymousCLICommand(root, r)
	root.Flags().CountP("verbose", "v", "Increase verbosity")

	_, err := pcmd.ExecuteCommand(rootCmd.Command)
	require.NoError(t, err)

	// Check auth is nil for now, until there is a better to create a fake logged in user and check if it's logged out
	require.Nil(t, cfg.Context().State.Auth)
	require.True(t, analyticsClient.SessionTimedOutCalled())
}

func Test_UpdateToken(t *testing.T) {
	jwtWithNoExp := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c"
	tests := []struct {
		name      string
		cliName   string
		authToken string
	}{
		{
			name:      "ccloud expired token",
			cliName:   "ccloud",
			authToken: expiredAuthTokenForDevCLoud,
		},
		{
			name:      "ccloud empty token",
			cliName:   "ccloud",
			authToken: "",
		},
		{
			name:      "ccloud invalid token",
			cliName:   "ccloud",
			authToken: "jajajajaja",
		},
		{
			name:      "ccloud jwt with no exp claim",
			cliName:   "ccloud",
			authToken: jwtWithNoExp,
		},
		{
			name:      "confluent expired token",
			cliName:   "confluent",
			authToken: expiredAuthTokenForDevCLoud,
		},
		{
			name:      "confluent empty token",
			cliName:   "confluent",
			authToken: "",
		},
		{
			name:      "confluent invalid token",
			cliName:   "confluent",
			authToken: "jajajajaja",
		},
		{
			name:      "confluent jwt with no exp claim",
			cliName:   "confluent",
			authToken: jwtWithNoExp,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var cfg *v3.Config
			if tt.cliName == "ccloud" {
				cfg = v3.AuthenticatedCloudConfigMock()
			} else {
				cfg = v3.AuthenticatedConfluentConfigMock()
			}

			cfg.Context().State.AuthToken = tt.authToken

			ver := pmock.NewVersionMock()

			updateTokenHandler := &cliMock.MockUpdateTokenHandler{
				UpdateCCloudAuthTokenUsingNetrcCredentialsFunc: func(ctx *v3.Context, userAgent string, logger *log.Logger) error {
					return nil
				},
				UpdateConfluentAuthTokenUsingNetrcCredentialsFunc: func(ctx *v3.Context, logger *log.Logger) error {
					return nil
				},
			}
			r := &pcmd.PreRun{
				CLIName: tt.cliName,
				Version: ver,
				Logger:  log.New(),
				UpdateClient: &mock.Client{
					CheckForUpdatesFunc: func(n, v string, f bool) (bool, string, error) {
						return false, "", nil
					},
				},
				FlagResolver: &pcmd.FlagResolverImpl{
					Prompt: &pcmd.RealPrompt{},
					Out:    os.Stdout,
				},
				Analytics:          cliMock.NewDummyAnalyticsMock(),
				Clock:              clockwork.NewRealClock(),
				UpdateTokenHandler: updateTokenHandler,
				Config:             cfg,
			}

			root := &cobra.Command{
				Run: func(cmd *cobra.Command, args []string) {},
			}
			rootCmd := pcmd.NewAnonymousCLICommand(root, r)
			root.Flags().CountP("verbose", "v", "Increase verbosity")

			_, err := pcmd.ExecuteCommand(rootCmd.Command)
			require.NoError(t, err)
			if tt.cliName == "ccloud" {
				require.True(t, updateTokenHandler.UpdateCCloudAuthTokenUsingNetrcCredentialsCalled())
			} else {
				require.True(t, updateTokenHandler.UpdateConfluentAuthTokenUsingNetrcCredentialsCalled())
			}
		})
	}
}

// Test that when context is of username login type it should check auth token and login state
// And when context is of API key credential then it should not ask for user to login
func TestPreRun_HasAPIKeyCommand(t *testing.T) {
	userNameConfigLoggedIn := v3.AuthenticatedCloudConfigMock()
	userNameConfigLoggedIn.Context().State.AuthToken = validAuthToken

	userNameCfgCorruptedAuthToken := v3.AuthenticatedCloudConfigMock()
	userNameCfgCorruptedAuthToken.Context().State.AuthToken = "corrupted.auth.token"

	userNotLoggedIn := v3.AuthenticatedCloudConfigMock()
	userNotLoggedIn.Context().State.Auth = nil

	usernameClusterWithoutKeyOrSecret := v3.AuthenticatedCloudConfigMock()
	usernameClusterWithoutKeyOrSecret.Context().State.AuthToken = validAuthToken
	usernameClusterWithoutKeyOrSecret.Context().KafkaClusterContext.GetKafkaClusterConfig("lkc-0000", "").APIKey = ""

	usernameClusterWithStoredSecret := v3.AuthenticatedCloudConfigMock()
	usernameClusterWithStoredSecret.Context().State.AuthToken = validAuthToken
	usernameClusterWithStoredSecret.Context().KafkaClusterContext.GetKafkaClusterConfig("lkc-0000", "").APIKeys["miles"] = &v0.APIKeyPair{
		Key:    "miles",
		Secret: "secret",
	}
	usernameClusterWithoutSecret := v3.AuthenticatedCloudConfigMock()
	usernameClusterWithoutSecret.Context().State.AuthToken = validAuthToken
	tests := []struct {
		name           string
		config         *v3.Config
		errMsg         string
		suggestionsMsg string
		key            string
		secret         string
	}{
		{
			name:   "username logged in user",
			config: userNameConfigLoggedIn,
		},
		{
			name:           "not logged in user",
			config:         userNotLoggedIn,
			errMsg:         errors.NotLoggedInErrorMsg,
			suggestionsMsg: fmt.Sprintf(errors.NotLoggedInSuggestions, "ccloud"),
		},
		{
			name:           "username context corrupted auth token",
			config:         userNameCfgCorruptedAuthToken,
			errMsg:         errors.CorruptedTokenErrorMsg,
			suggestionsMsg: errors.CorruptedTokenSuggestions,
		},
		{
			name:   "api credential context",
			config: v3.APICredentialConfigMock(),
		},
		{
			name:   "api key and secret passed via flags",
			key:    "miles",
			secret: "shhhh",
			config: usernameClusterWithoutKeyOrSecret,
		},
		{
			name: "api key passed via flag with stored secret",
			key: "miles",
			config: usernameClusterWithStoredSecret,
		},
		{
			name: "api key passed via flag without stored secret",
			key: "miles",
			errMsg: errors.NoAPISecretStoredOrPassedMsg,
			suggestionsMsg: errors.NoAPISecretStoredOrPassedSuggestions,
			config: usernameClusterWithoutSecret,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ver := pmock.NewVersionMock()
			analyticsClient := cliMock.NewDummyAnalyticsMock()

			r := &pcmd.PreRun{
				Version: ver,
				Logger:  log.New(),
				UpdateClient: &mock.Client{
					CheckForUpdatesFunc: func(n, v string, f bool) (bool, string, error) {
						return false, "", nil
					},
				},
				FlagResolver: &pcmd.FlagResolverImpl{
					Prompt: &pcmd.RealPrompt{},
					Out:    os.Stdout,
				},
				Analytics:          analyticsClient,
				Clock:              clockwork.NewRealClock(),
				UpdateTokenHandler: auth.NewUpdateTokenHandler(auth.NewNetrcHandler("")),
				Config:             tt.config,
			}

			root := &cobra.Command{
				Run: func(cmd *cobra.Command, args []string) {},
			}
			rootCmd := pcmd.NewHasAPIKeyCLICommand(root, r)
			root.Flags().CountP("verbose", "v", "Increase verbosity")
			root.Flags().String("api-key", "", "Kafka cluster API key.")
			root.Flags().String("api-secret", "", "API key secret.")
			root.Flags().String("cluster", "", "Kafka cluster ID.")

			_, err := pcmd.ExecuteCommand(rootCmd.Command, "--api-key", tt.key, "--api-secret", tt.secret)
			if tt.errMsg != "" {
				require.Error(t, err)
				require.Equal(t, tt.errMsg, err.Error())
				if tt.suggestionsMsg != "" {
					errors.VerifyErrorAndSuggestions(require.New(t), err, tt.errMsg, tt.suggestionsMsg)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestAuthenticatedStateFlagCommand_AddCommand(t *testing.T) {
	userNameConfigLoggedIn := v3.AuthenticatedCloudConfigMock()
	userNameConfigLoggedIn.Context().State.AuthToken = validAuthToken

	subcommandFlags := map[string]*pflag.FlagSet {
		"root"	:	pcmd.ContextSet(),
		"one"	:	pcmd.EnvironmentContextSet(),
		"two"	:	pcmd.KeySecretSet(),
	}
	r := &pcmd.PreRun{
		CLIName: "ccloud",
		Version: pmock.NewVersionMock(),
		Logger:  log.New(),
		UpdateClient: &mock.Client{
			CheckForUpdatesFunc: func(n, v string, f bool) (bool, string, error) {
				return false, "", nil
			},
		},
		FlagResolver: &pcmd.FlagResolverImpl{
			Prompt: &pcmd.RealPrompt{},
			Out:    os.Stdout,
		},
		Analytics:          cliMock.NewDummyAnalyticsMock(),
		Clock:              clockwork.NewRealClock(),
		UpdateTokenHandler: auth.NewUpdateTokenHandler(auth.NewNetrcHandler("")),
		Config:             userNameConfigLoggedIn,
	}
	cmdRoot := &cobra.Command{Use: "root"}
	root := pcmd.NewAuthenticatedStateFlagCommand(cmdRoot, r, subcommandFlags)

	for subcommand, _ := range subcommandFlags {
		t.Run(subcommand, func(t *testing.T) {
			cmd := &cobra.Command{Use: subcommand}
			root.AddCommand(cmd)
			//create flagset of all flags that should be included
			shouldHaveFlags := subcommandFlags["root"]
			shouldHaveFlags.AddFlagSet(subcommandFlags[subcommand])
			//iterate through shouldHaveFlags and make sure they are all attached to cmd
			shouldHaveFlags.VisitAll(func(flag *pflag.Flag) {
				f := cmd.Flag(flag.Name)
				require.NotNil(t, f)
			})
		})
	}
}
