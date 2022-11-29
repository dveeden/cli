package streamshare

import (
	"github.com/spf13/cobra"

	pcmd "github.com/confluentinc/cli/internal/pkg/cmd"
	"github.com/confluentinc/cli/internal/pkg/errors"
	"github.com/confluentinc/cli/internal/pkg/examples"
	"github.com/confluentinc/cli/internal/pkg/resource"
	"github.com/confluentinc/cli/internal/pkg/utils"
)

func (c *command) newConsumerShareDeleteCommand() *cobra.Command {
	return &cobra.Command{
		Use:               "delete <id>",
		Short:             "Delete a consumer share.",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: pcmd.NewValidArgsFunction(c.validConsumerShareArgs),
		RunE:              c.deleteConsumerShare,
		Example: examples.BuildExampleString(
			examples.Example{
				Text: `Delete consumer share "ss-12345":`,
				Code: "confluent stream-share consumer share delete ss-12345",
			},
		),
	}
}

func (c *command) deleteConsumerShare(cmd *cobra.Command, args []string) error {
	isDeleteShareConfirmed, err := confirmDeleteShare(cmd, DeleteConsumerShareMsg)
	if err != nil {
		return err
	}

	if !isDeleteShareConfirmed {
		utils.Println(cmd, "Operation terminated.")
		return nil
	}

	shareId := args[0]

	err = c.V2Client.DeleteConsumerShare(shareId)
	if err != nil {
		return err
	}

	utils.Printf(cmd, errors.DeletedResourceMsg, resource.ConsumerShare, shareId)
	return nil
}
