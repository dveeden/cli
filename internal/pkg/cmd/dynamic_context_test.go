package cmd_test

import (
	orgv1 "github.com/confluentinc/cc-structs/kafka/org/v1"
	pcmd "github.com/confluentinc/cli/internal/pkg/cmd"
	v1 "github.com/confluentinc/cli/internal/pkg/config/v1"
	v3 "github.com/confluentinc/cli/internal/pkg/config/v3"
	"github.com/confluentinc/cli/internal/pkg/errors"
	pmock "github.com/confluentinc/cli/internal/pkg/mock"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
	"os"
	"testing"
)

func TestGetKafkaClusterForCommand(t *testing.T) {
	config := v3.AuthenticatedCloudConfigMock()
	context := pcmd.NewDynamicContext(config.Context(), &pcmd.FlagResolverImpl{
		Prompt: &pcmd.RealPrompt{},
		Out:    os.Stdout,
	}, pmock.NewClientMock())

	flagContext := pcmd.NewDynamicContext(config.Context(), &pcmd.FlagResolverImpl{
		Prompt: &pcmd.RealPrompt{},
		Out:    os.Stdout,
	}, pmock.NewClientMock())

	// create cluster that will be used in "--cluster" flag value
	flagContext.KafkaClusterContext.KafkaEnvContexts["testAccount"].KafkaClusterConfigs["lkc-0001"] = &v1.KafkaClusterConfig{
		ID:          "lkc-0001",
		Name:        "miles",
	}

	tests := []struct {
		name			string
		ctx				*pcmd.DynamicContext
		cluster			string
		errMsg			string
		suggestionsMsg	string
	}{
		{
			name:	"read cluster from config",
			ctx:	context,
		},
		{
			name:	"read cluster from flag",
			ctx:	flagContext,
			cluster:	"lkc-0001",

		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func (t *testing.T) {
			cmd := &cobra.Command{
				Run:                        func(cmd *cobra.Command, args []string) {},
			}
			cmd.Flags().String("cluster", "", "Kafka cluster ID.")
			//execute so that flags will be parsed
			_, err := pcmd.ExecuteCommand(cmd, "--cluster", tt.cluster)
			if tt.errMsg != "" {
				require.Error(t, err)
				require.Equal(t, tt.errMsg, err.Error())
				if tt.suggestionsMsg != "" {
					errors.VerifyErrorAndSuggestions(require.New(t), err, tt.errMsg, tt.suggestionsMsg)
				}
			} else {
				require.NoError(t, err)
				if tt.cluster != "" {
					cluster, _ := tt.ctx.GetKafkaClusterForCommand(cmd)
					require.Equal(t, cluster.ID, tt.cluster)
				}
			}
		})
	}
}

func TestAuthenticatedStateSetsEnvironmentFromFlag(t *testing.T) {
	config := v3.AuthenticatedCloudConfigMock()
	ctx := pcmd.NewDynamicContext(config.Context(), &pcmd.FlagResolverImpl{
		Prompt: &pcmd.RealPrompt{},
		Out:    os.Stdout,
	}, pmock.NewClientMock())

	config = v3.AuthenticatedCloudConfigMock()
	flagContext := pcmd.NewDynamicContext(config.Context(), &pcmd.FlagResolverImpl{
		Prompt: &pcmd.RealPrompt{},
		Out:    os.Stdout,
	}, pmock.NewClientMock())
	flagContext.State.Auth.Accounts = append(flagContext.State.Auth.Accounts, &orgv1.Account{Name: "env-test", Id: "env-test"})

	tests := []struct {
		name			string
		env				string
		errMsg			string
		suggestionsMsg	string
		context			*pcmd.DynamicContext
	}{
		{
			name:	"read environment from config",
			context: ctx,
		},
		{
			name:	"read environment from flag",
			env:	"env-test",
			context: flagContext,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func (t *testing.T) {
			cmd := &cobra.Command{
				Run:                        func(cmd *cobra.Command, args []string) {},
			}
			cmd.Flags().String("environment", "", "Environment ID.")
			cmd.Flags().CountP("verbose", "v", "Increase verbosity")
			//execute command to have flags parsed
			_, err := pcmd.ExecuteCommand(cmd, "--environment", tt.env)
			require.NoError(t, err)
			state, err := tt.context.AuthenticatedState(cmd)
			if tt.errMsg != "" {
				require.Error(t, err)
				require.Equal(t, tt.errMsg, err.Error())
				if tt.suggestionsMsg != "" {
					errors.VerifyErrorAndSuggestions(require.New(t), err, tt.errMsg, tt.suggestionsMsg)
				}
			} else {
				require.NoError(t, err)
				if tt.env != "" {
					require.Equal(t, tt.env, state.Auth.Account.Id)
				}
			}
		})
	}
}
