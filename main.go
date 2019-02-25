package main

import (
	"fmt"
	"os"
	"runtime"

	"github.com/confluentinc/cli/command/update"
	"github.com/hashicorp/go-plugin"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/confluentinc/cli/command"
	"github.com/confluentinc/cli/command/auth"
	"github.com/confluentinc/cli/command/common"
	"github.com/confluentinc/cli/command/config"
	"github.com/confluentinc/cli/command/connect"
	"github.com/confluentinc/cli/command/kafka"
	"github.com/confluentinc/cli/command/ksql"
	"github.com/confluentinc/cli/log"
	"github.com/confluentinc/cli/metric"
	"github.com/confluentinc/cli/shared"
	cliVersion "github.com/confluentinc/cli/version"
)

var (
	// Injected from linker flags like `go build -ldflags "-X main.version=$VERSION" -X ...`
	version = "v0.0.0"
	commit  = ""
	date    = ""
	host    = ""
)

func main() {
	viper.AutomaticEnv()

	var logger *log.Logger
	{
		logger = log.New()
		logger.Out = os.Stdout
		logger.Log("msg", "hello")
		defer logger.Log("msg", "goodbye")

		//if viper.GetString("log_level") != "" {
		//	level, err := logrus.ParseLevel(viper.GetString("log_level"))
			level, err := logrus.ParseLevel("TRACE")
			check(err)
			logger.SetLevel(level)
			logger.Log("msg", "set log level", "level", level.String())
		//}
	}

	var metricSink shared.MetricSink
	{
		metricSink = metric.NewSink()
	}

	var cfg *shared.Config
	{
		cfg = shared.NewConfig(&shared.Config{
			MetricSink: metricSink,
			Logger:     logger,
		})
		err := cfg.Load()
		if err != nil && err != shared.ErrNoConfig {
			logger.WithError(err).Errorf("unable to load cfg")
		}
	}

	userAgent := fmt.Sprintf("Confluent/1.0 ccloud/%s (%s/%s)", version, runtime.GOOS, runtime.GOARCH)
	version := cliVersion.NewVersion(version, commit, date, host, userAgent)
	factory := &common.GRPCPluginFactoryImpl{}

	cli := BuildCommand(cfg, version, factory, logger)
	check(cli.Execute())

	plugin.CleanupClients()
	os.Exit(0)
}

func BuildCommand(cfg *shared.Config, version *cliVersion.Version, factory common.GRPCPluginFactory, logger *log.Logger) *cobra.Command {
	cli := &cobra.Command{
		Use:   "ccloud",
		Short: "Welcome to the Confluent Cloud CLI",
	}

	prompt := command.NewTerminalPrompt(os.Stdin)

	cli.Version = version.Version
	cli.AddCommand(common.NewVersionCmd(version, prompt))

	cli.AddCommand(config.New(cfg))

	cli.AddCommand(common.NewCompletionCmd(cli, prompt))

	cli.AddCommand(auth.New(cfg)...)

	var installedPlugins []string
	addPluginCommand := func(cmd *cobra.Command, path string) {
		cli.AddCommand(cmd)
		installedPlugins = append(installedPlugins, path)
	}

	conn, path, err := kafka.New(cfg, factory)
	if err != nil {
		logger.Log("msg", err)
	} else {
		addPluginCommand(conn, path)
	}

	conn, path, err = connect.New(cfg, factory)
	if err != nil {
		logger.Log("msg", err)
	} else {
		addPluginCommand(conn, path)
	}

	conn, path, err = ksql.New(cfg, factory)
	if err != nil {
		logger.Log("msg", err)
	} else {
		addPluginCommand(conn, path)
	}

	cli.AddCommand(update.New("ccloud", installedPlugins, cfg, version))

	return cli
}

func check(err error) {
	if err != nil {
		plugin.CleanupClients()
		os.Exit(1)
	}
}
