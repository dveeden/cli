package environment

import (
	"fmt"

	"github.com/spf13/cobra"

	pcmd "github.com/confluentinc/cli/internal/pkg/cmd"
	"github.com/confluentinc/cli/internal/pkg/output"
)

func (c *command) newListCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List Confluent Cloud environments.",
		Args:  cobra.NoArgs,
		RunE:  c.list,
	}

	pcmd.AddContextFlag(cmd, c.CLICommand)
	pcmd.AddOutputFlag(cmd)

	return cmd
}

func (c *command) list(cmd *cobra.Command, _ []string) error {
	environments, err := c.V2Client.ListOrgEnvironments()
	if err != nil {
		return err
	}

	list := output.NewList(cmd)
	for _, environment := range environments {
		id := *environment.Id
		if output.GetFormat(cmd) == output.Human {
			if *environment.Id == c.EnvironmentId() {
				id = fmt.Sprintf("* %s", id)
			} else {
				id = fmt.Sprintf("  %s", id)
			}
		}

		list.Add(&out{
			Id:   id,
			Name: *environment.DisplayName,
		})
	}
	return list.Print()
}
