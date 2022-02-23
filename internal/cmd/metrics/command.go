package metrics

import (
	"github.com/spf13/cobra"

	pcmd "github.com/confluentinc/cli/internal/pkg/cmd"
)

type command struct {
	*pcmd.AuthenticatedCLICommand
}

func New(prerunner pcmd.PreRunner) *cobra.Command {
	c := &command{
		pcmd.NewAuthenticatedCLICommand(
			&cobra.Command{
				Use:   "metrics",
				Short: "Query Confluent Cloud metrics.",
			},
			prerunner,
		),
	}

	c.AddCommand(c.newQueryCommand())

	return c.Command
}
