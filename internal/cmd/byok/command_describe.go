package byok

import (
	"github.com/spf13/cobra"

	pcmd "github.com/confluentinc/cli/internal/pkg/cmd"
	"github.com/confluentinc/cli/internal/pkg/errors"
	"github.com/confluentinc/cli/internal/pkg/output"
	"github.com/confluentinc/cli/internal/pkg/utils"
)

type describeStruct struct {
	Id        string   `human:"ID" serialzed:"id"`
	Key       string   `human:"Key" serialzed:"key"`
	Roles     []string `human:"Roles" serialzed:"roles"`
	Provider  string   `human:"Provider" serialzed:"provider"`
	State     string   `human:"State" serialzed:"state"`
	CreatedAt string   `human:"Created At" serialzed:"created_at"`
	UpdatedAt string   `human:"Updated At" serialzed:"updated_at"`
	DeletedAt string   `human:"Deleted At" serialzed:"deleted_at"`
}

func (c *command) newDescribeCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "describe <id>",
		Short:             "Describe a self-managed key in Confluent Cloud.",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: pcmd.NewValidArgsFunction(c.validArgs),
		RunE:              c.describe,
	}

	cmd.Flags().Bool("show-policy-command", false, "Print post-creation step to grant Confluent access to the key.")
	pcmd.AddOutputFlag(cmd)

	return cmd
}

func (c *command) describe(cmd *cobra.Command, args []string) error {
	key, httpResp, err := c.V2Client.GetByokKey(args[0])
	if err != nil {
		return errors.CatchEnvironmentNotFoundError(err, httpResp)
	}

	var keyString string
	var roles []string
	switch {
	case key.Key.ByokV1AwsKey != nil:
		keyString = key.Key.ByokV1AwsKey.KeyArn
		roles = key.Key.ByokV1AwsKey.GetRoles()
	case key.Key.ByokV1AzureKey != nil:
		keyString = key.Key.ByokV1AzureKey.KeyId
		roles = append(roles, key.Key.ByokV1AzureKey.GetApplicationId())
	default:
		return errors.New("unknown key type")
	}
	var updatedAt string
	if key.Metadata.UpdatedAt == nil || key.Metadata.UpdatedAt.IsZero() {
		updatedAt = ""
	} else {
		updatedAt = key.Metadata.UpdatedAt.String()
	}
	var deletedAt string
	if key.Metadata.DeletedAt == nil || key.Metadata.UpdatedAt.IsZero() {
		deletedAt = ""
	} else {
		deletedAt = key.Metadata.DeletedAt.String()
	}

	table := output.NewTable(cmd)
	table.Add(&describeStruct{
		Id:        *key.Id,
		Key:       keyString,
		Roles:     roles,
		Provider:  *key.Provider,
		State:     *key.State,
		CreatedAt: key.Metadata.CreatedAt.String(),
		UpdatedAt: updatedAt,
		DeletedAt: deletedAt,
	})
	table.Print()

	// If the user has specified the --show-policy-command flag, print the post-creation step to grant Confluent access to the key.
	showPolicyCommand, err := cmd.Flags().GetBool("show-policy-command")
	if err != nil {
		return err
	}

	if showPolicyCommand {
		postCreationStepInstructions, err := renderPostCreationStepInstructions(&key)
		if err != nil {
			return err
		}

		utils.Println(cmd, postCreationStepInstructions)
	}

	return nil
}
