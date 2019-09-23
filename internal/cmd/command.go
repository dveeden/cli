package cmd

import (
	"context"
	"os"

	"github.com/DABH/go-basher"
	"github.com/jonboulle/clockwork"
	"github.com/spf13/cobra"

	"github.com/confluentinc/ccloud-sdk-go"
	"github.com/confluentinc/mds-sdk-go"

	"github.com/confluentinc/cli/internal/cmd/apikey"
	"github.com/confluentinc/cli/internal/cmd/auth"
	"github.com/confluentinc/cli/internal/cmd/completion"
	"github.com/confluentinc/cli/internal/cmd/config"
	"github.com/confluentinc/cli/internal/cmd/environment"
	"github.com/confluentinc/cli/internal/cmd/iam"
	initcontext "github.com/confluentinc/cli/internal/cmd/init-context"
	"github.com/confluentinc/cli/internal/cmd/kafka"
	"github.com/confluentinc/cli/internal/cmd/ksql"
	"github.com/confluentinc/cli/internal/cmd/local"
	ps1 "github.com/confluentinc/cli/internal/cmd/prompt"
	"github.com/confluentinc/cli/internal/cmd/schema-registry"
	"github.com/confluentinc/cli/internal/cmd/secret"
	"github.com/confluentinc/cli/internal/cmd/service-account"
	"github.com/confluentinc/cli/internal/cmd/update"
	"github.com/confluentinc/cli/internal/cmd/version"
	pcmd "github.com/confluentinc/cli/internal/pkg/cmd"
	configs "github.com/confluentinc/cli/internal/pkg/config"
	"github.com/confluentinc/cli/internal/pkg/errors"
	"github.com/confluentinc/cli/internal/pkg/help"
	"github.com/confluentinc/cli/internal/pkg/io"
	"github.com/confluentinc/cli/internal/pkg/keystore"
	"github.com/confluentinc/cli/internal/pkg/log"
	pps1 "github.com/confluentinc/cli/internal/pkg/ps1"
	apikeys "github.com/confluentinc/cli/internal/pkg/sdk/apikey"
	environments "github.com/confluentinc/cli/internal/pkg/sdk/environment"
	kafkas "github.com/confluentinc/cli/internal/pkg/sdk/kafka"
	ksqls "github.com/confluentinc/cli/internal/pkg/sdk/ksql"
	users "github.com/confluentinc/cli/internal/pkg/sdk/user"
	secrets "github.com/confluentinc/cli/internal/pkg/secret"
	versions "github.com/confluentinc/cli/internal/pkg/version"
)

func NewConfluentCommand(cliName string, cfg *configs.Config, ver *versions.Version, logger *log.Logger) (*cobra.Command, error) {
	cli := &cobra.Command{
		Use:               cliName,
		Version:           ver.Version,
		DisableAutoGenTag: true,
	}
	cli.SetUsageFunc(func(cmd *cobra.Command) error {
		return help.ResolveReST(cmd.UsageTemplate(), cmd)
	})
	cli.SetHelpFunc(func(cmd *cobra.Command, args []string) {
		_ = help.ResolveReST(cmd.HelpTemplate(), cmd)
	})
	if cliName == "ccloud" {
		cli.Short = "Confluent Cloud CLI."
		cli.Long = "Manage your Confluent Cloud."
	} else {
		cli.Short = "Confluent CLI."
		cli.Long = "Manage your Confluent Platform."
	}
	cli.PersistentFlags().CountP("verbose", "v",
		"Increase verbosity (-v for warn, -vv for info, -vvv for debug, -vvvv for trace).")

	prompt := pcmd.NewPrompt(os.Stdin)

	updateClient, err := update.NewClient(cliName, logger)
	if err != nil {
		return nil, err
	}

	client := ccloud.NewClientWithJWT(context.Background(), cfg.AuthToken, &ccloud.Params{
		BaseURL: cfg.AuthURL, Logger: cfg.Logger, UserAgent: ver.UserAgent,
	})

	ch := &pcmd.ConfigHelper{Config: cfg, Client: client, Version: ver}
	fs := &io.RealFileSystem{}

	prerunner := &pcmd.PreRun{
		UpdateClient: updateClient,
		CLIName:      cliName,
		Version:      ver.Version,
		Logger:       logger,
		Config:       cfg,
		ConfigHelper: ch,
		Clock:        clockwork.NewRealClock(),
	}

	cli.PersistentPreRunE = prerunner.Anonymous()

	mdsConfig := mds.NewConfiguration()
	mdsConfig.BasePath = cfg.AuthURL
	mdsConfig.UserAgent = ver.UserAgent

	mdsClient := mds.NewAPIClient(mdsConfig)

	cli.Version = ver.Version
	cli.AddCommand(version.NewVersionCmd(prerunner, ver))

	conn := config.New(cfg)
	conn.Hidden = true // The config/context feature isn't finished yet, so let's hide it
	cli.AddCommand(conn)

	cli.AddCommand(completion.NewCompletionCmd(cli, cliName))
	cli.AddCommand(update.New(cliName, cfg, ver, prompt, updateClient))
	cli.AddCommand(auth.New(prerunner, cfg, logger, mdsClient, ver.UserAgent)...)

	resolver := &pcmd.FlagResolverImpl{Prompt: prompt, Out: os.Stdout}

	if cliName == "ccloud" {
		kafkaClient := kafkas.New(client, logger)
		cmd, err := kafka.New(prerunner, cfg, kafkaClient, ch)
		if err != nil {
			return nil, err
		}
		cli.AddCommand(cmd)
		cli.AddCommand(initcontext.New(prerunner, cfg, prompt, resolver))
		credType, err := cfg.CredentialType()
		if _, ok := err.(*errors.UnspecifiedCredentialError); ok {
			return nil, err
		}
		if credType == configs.APIKey {
			return cli, nil
		}
		cli.AddCommand(ps1.NewPromptCmd(cfg, &pps1.Prompt{Config: cfg}, logger))
		userClient := users.New(client, logger)
		ks := &keystore.ConfigKeyStore{Config: cfg, Helper: ch}
		cli.AddCommand(environment.New(prerunner, cfg, environments.New(client, logger), cliName))
		cli.AddCommand(service_account.New(prerunner, cfg, userClient))
		cli.AddCommand(apikey.New(prerunner, cfg, apikeys.New(client, logger), ch, ks))

		// Schema Registry
		// If srClient is nil, the function will look it up after prerunner verifies authentication. Exposed so tests can pass mocks
		sr := schema_registry.New(prerunner, cfg, client.SchemaRegistry, ch, nil, client.Metrics, logger)
		cli.AddCommand(sr)

		conn = ksql.New(prerunner, cfg, ksqls.New(client, logger), kafkaClient, userClient, ch)
		conn.Hidden = true // The ksql feature isn't finished yet, so let's hide it
		cli.AddCommand(conn)

		//conn = connect.New(prerunner, cfg, connects.New(client, logger))
		//conn.Hidden = true // The connect feature isn't finished yet, so let's hide it
		//cli.AddCommand(conn)
	} else if cliName == "confluent" {
		ch := &pcmd.ConfigHelper{Config: cfg, Client: client}

		cli.AddCommand(iam.New(prerunner, cfg, ch, ver, mdsClient))

		bash, err := basher.NewContext("/bin/bash", false)
		if err != nil {
			return nil, err
		}
		shellRunner := &local.BashShellRunner{BasherContext: bash}
		cli.AddCommand(local.New(cli, prerunner, shellRunner, logger, fs))

		cli.AddCommand(secret.New(prerunner, cfg, prompt, resolver, secrets.NewPasswordProtectionPlugin(logger)))
	}
	return cli, nil
}
