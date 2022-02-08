package kafka

import (
	"context"
	"fmt"
	"os"

	"github.com/confluentinc/cli/internal/pkg/log"

	schedv1 "github.com/confluentinc/cc-structs/kafka/scheduler/v1"
	"github.com/spf13/cobra"

	pcmd "github.com/confluentinc/cli/internal/pkg/cmd"
	v1 "github.com/confluentinc/cli/internal/pkg/config/v1"
	"github.com/confluentinc/cli/internal/pkg/errors"
	"github.com/confluentinc/cli/internal/pkg/examples"
	"github.com/confluentinc/cli/internal/pkg/form"
	"github.com/confluentinc/cli/internal/pkg/utils"
)

func (c *clusterCommand) newUpdateCommand(cfg *v1.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:               "update <id>",
		Short:             "Update a Kafka cluster.",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: pcmd.NewValidArgsFunction(c.validArgs),
		RunE: pcmd.NewCLIRunE(func(cmd *cobra.Command, args []string) error {
			return c.update(cmd, args, form.NewPrompt(os.Stdin))
		}),
		Annotations: map[string]string{pcmd.RunRequirement: pcmd.RequireNonAPIKeyCloudLogin},
		Example: examples.BuildExampleString(
			examples.Example{
				Text: "Change a cluster's name and expand its CKU count:",
				Code: `confluent kafka cluster update lkc-abc123 --name "Cool Cluster" --cku 3`,
			},
		),
	}

	cmd.Flags().String("name", "", "Name of the Kafka cluster.")
	cmd.Flags().Int("cku", 0, "Number of Confluent Kafka Units (non-negative). For Kafka clusters of type 'dedicated' only. When shrinking a cluster, you can reduce capacity one CKU at a time.")
	pcmd.AddContextFlag(cmd, c.CLICommand)
	if cfg.IsCloudLogin() {
		pcmd.AddEnvironmentFlag(cmd, c.AuthenticatedCLICommand)
	}
	pcmd.AddOutputFlag(cmd)

	return cmd
}

func (c *clusterCommand) update(cmd *cobra.Command, args []string, prompt form.Prompt) error {
	if !cmd.Flags().Changed("name") && !cmd.Flags().Changed("cku") {
		return errors.New(errors.NameOrCKUFlagErrorMsg)
	}

	clusterID := args[0]
	req := &schedv1.KafkaCluster{
		AccountId: c.EnvironmentId(),
		Id:        clusterID,
	}
	currentCluster, err := c.Client.Kafka.Describe(context.Background(), req)
	if err != nil {
		return errors.NewErrorWithSuggestions(fmt.Sprintf(errors.KafkaClusterNotFoundErrorMsg, clusterID), errors.ChooseRightEnvironmentSuggestions)
	}

	if cmd.Flags().Changed("name") {
		name, err := cmd.Flags().GetString("name")
		if err != nil {
			return err
		}
		if name == "" {
			return errors.New(errors.NonEmptyNameErrorMsg)
		}
		req.Name = name
	} else {
		req.Name = currentCluster.Name
	}

	req.Cku, err = c.validateResize(cmd, currentCluster, prompt)
	if err != nil {
		return err
	}

	updatedCluster, err := c.Client.Kafka.Update(context.Background(), req)
	if err != nil {
		return errors.NewErrorWithSuggestions(err.Error(), errors.KafkaClusterUpdateFailedSuggestions)
	}

	return outputKafkaClusterDescription(cmd, updatedCluster)
}

func (c *clusterCommand) validateResize(cmd *cobra.Command, currentCluster *schedv1.KafkaCluster, prompt form.Prompt) (int32, error) {
	if cmd.Flags().Changed("cku") {
		cku, err := cmd.Flags().GetInt("cku")
		if err != nil {
			return currentCluster.Cku, err
		}
		// Ensure the cluster is a Dedicated Cluster
		if !isDedicated(currentCluster) {
			return currentCluster.Cku, errors.Errorf("error updating kafka cluster: %v", errors.ClusterResizeNotSupported)
		}
		// Durability Checks
		if currentCluster.Durability == schedv1.Durability_HIGH && cku <= 1 {
			return currentCluster.Cku, errors.New(errors.CKUMoreThanOneErrorMsg)
		}
		if cku <= 0 {
			return currentCluster.Cku, errors.New(errors.CKUMoreThanZeroErrorMsg)
		}
		// Cluster can't be resized while it's provisioning or being expanded already.
		// Name _can_ be changed during these times, though.
		err = isClusterResizeInProgress(currentCluster)
		if err != nil {
			return currentCluster.Cku, err
		}
		//If shrink
		if int32(cku) < currentCluster.Cku {
			// metrics api auth via jwt
			shouldPrompt, errFromSmallWindowMetrics := c.validateKafkaClusterMetrics(context.Background(), int32(cku), currentCluster, true)
			if errFromSmallWindowMetrics != nil && !shouldPrompt {
				return currentCluster.Cku, fmt.Errorf("cluster shrink validation error: \n%v", errFromSmallWindowMetrics)
			}
			promptMessage := ""
			if shouldPrompt {
				promptMessage = fmt.Sprintf("\n%v\n", errFromSmallWindowMetrics)
			}
			_, errFromLargeWindowMetrics := c.validateKafkaClusterMetrics(context.Background(), int32(cku), currentCluster, false)
			if errFromLargeWindowMetrics != nil {
				promptMessage += fmt.Sprintf("\n%v\n", errFromLargeWindowMetrics)
			}
			if promptMessage != "" {
				ok, err := confirmShrink(cmd, prompt, promptMessage)
				if !ok || err != nil {
					return currentCluster.Cku, err
				} else {
					return int32(cku), nil
				}
			}
		}
		return int32(cku), nil
	}
	return currentCluster.Cku, nil
}

func (c *clusterCommand) validateKafkaClusterMetrics(ctx context.Context, cku int32, currentCluster *schedv1.KafkaCluster, isLatestMetric bool) (bool, error) {
	var window string
	if isLatestMetric {
		window = "15 min"
	} else {
		window = "3 days"
	}
	requiredPartitionCount, requiredStorageLimit, err := c.getUsageLimit(ctx, uint32(cku))
	if err != nil {
		log.CliLogger.Warn("Could not retrieve usage limits ", err)
		return false, errors.New("Could not retrieve usage limits to validate request to shrink cluster.")
	}
	errorMessage := errors.Errorf("Looking at metrics in the last %s window:", window)
	shouldPrompt := true
	isValidPartitionCountErr := c.validatePartitionCount(currentCluster.Id, requiredPartitionCount, isLatestMetric, cku)
	if isValidPartitionCountErr != nil {
		errorMessage = errors.Errorf("%v \n %v", errorMessage.Error(), isValidPartitionCountErr.Error())
		shouldPrompt = false
	}
	var isValidStorageLimitErr error
	if !currentCluster.InfiniteStorage {
		isValidStorageLimitErr = c.validateStorageLimit(currentCluster.Id, requiredStorageLimit, isLatestMetric, cku)
		if isValidStorageLimitErr != nil {
			errorMessage = errors.Errorf("%v \n %v", errorMessage.Error(), isValidStorageLimitErr.Error())
			shouldPrompt = false
		}
	}
	// Get Cluster Load Metric
	isValidLoadErr := c.validateClusterLoad(currentCluster.Id, isLatestMetric)
	if isValidLoadErr != nil {
		errorMessage = errors.Errorf("%v \n %v", errorMessage.Error(), isValidLoadErr)
	}
	if isValidStorageLimitErr == nil && isValidLoadErr == nil && isValidPartitionCountErr == nil {
		return false, nil
	}
	return shouldPrompt, errorMessage
}

func (c *clusterCommand) getUsageLimit(ctx context.Context, cku uint32) (int32, int32, error) {
	usageReply, err := c.Client.UsageLimits.GetUsageLimits(ctx)
	if err != nil || usageReply.UsageLimits == nil || len(usageReply.UsageLimits.GetCkuLimits()) == 0 || usageReply.UsageLimits.GetCkuLimits()[cku] == nil {
		return 0, 0, errors.Wrap(err, "Could not retrieve partition count usage limits. Please try again or contact support.")
	}
	partitionCount := usageReply.UsageLimits.GetCkuLimits()[cku].GetNumPartitions().GetValue()
	storageLimit := usageReply.UsageLimits.GetCkuLimits()[cku].Storage.GetValue()
	return partitionCount, storageLimit, nil
}

func confirmShrink(cmd *cobra.Command, prompt form.Prompt, promptMessage string) (bool, error) {
	f := form.New(form.Field{ID: "proceed", Prompt: fmt.Sprintf("Validated cluster metrics and found that: %s\nDo you want to proceed with shrinking your kafka cluster?", promptMessage), IsYesOrNo: true})
	if err := f.Prompt(cmd, prompt); err != nil {
		return false, errors.New(errors.FailedToReadClusterResizeConfirmationErrorMsg)
	}
	if !f.Responses["proceed"].(bool) {
		utils.Println(cmd, "Not proceeding with kafka cluster shrink")
		return false, nil
	}
	return true, nil
}