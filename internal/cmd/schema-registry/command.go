package schemaregistry

import (
	srsdk "github.com/confluentinc/schema-registry-sdk-go"
	"github.com/spf13/cobra"

	pcmd "github.com/confluentinc/cli/internal/pkg/cmd"
	v1 "github.com/confluentinc/cli/internal/pkg/config/v1"
)

func New(cfg *v1.Config, prerunner pcmd.PreRunner, srClient *srsdk.APIClient) *cobra.Command {
	cmd := &cobra.Command{
		Use:         "schema-registry",
		Aliases:     []string{"sr"},
		Short:       "Manage Schema Registry.",
		Annotations: map[string]string{pcmd.RunRequirement: pcmd.RequireNonAPIKeyCloudLoginOrOnPremLogin},
	}

	c := pcmd.NewAuthenticatedCLICommand(cmd, prerunner)

	c.AddCommand(newClusterCommand(cfg, prerunner, srClient))
	c.AddCommand(newCompatibilityCommand(cfg, prerunner, srClient))
	c.AddCommand(newConfigCommand(cfg, prerunner, srClient))
	c.AddCommand(newExporterCommand(cfg, prerunner, srClient))
	c.AddCommand(newSchemaCommand(cfg, prerunner, srClient))
	c.AddCommand(newSubjectCommand(cfg, prerunner, srClient))
	return c.Command
}

func addCompatibilityFlag(cmd *cobra.Command) {
	cmd.Flags().String("compatibility", "", "Can be BACKWARD, BACKWARD_TRANSITIVE, FORWARD, FORWARD_TRANSITIVE, FULL, FULL_TRANSITIVE, or NONE.")
	pcmd.RegisterFlagCompletionFunc(cmd, "compatibility", func(_ *cobra.Command, _ []string) []string {
		return []string{"BACKWARD", "BACKWARD_TRANSITIVE", "FORWARD", "FORWARD_TRANSITIVE", "FULL", "FULL_TRANSITIVE", "NONE"}
	})
}

func addModeFlag(cmd *cobra.Command) {
	cmd.Flags().String("mode", "", "Can be READWRITE, READONLY, OR IMPORT.")
	pcmd.RegisterFlagCompletionFunc(cmd, "mode", func(_ *cobra.Command, _ []string) []string {
		return []string{"READWRITE", "READONLY", "IMPORT"}
	})
}
