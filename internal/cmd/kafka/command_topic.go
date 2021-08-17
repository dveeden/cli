package kafka

import (
	"bufio"
	"context"
	"encoding/binary"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"sort"
	"strings"

	v1 "github.com/confluentinc/cli/internal/pkg/config/v1"

	"github.com/c-bata/go-prompt"

	"github.com/confluentinc/kafka-rest-sdk-go/kafkarestv3"
	srsdk "github.com/confluentinc/schema-registry-sdk-go"

	"github.com/antihax/optional"
	schedv1 "github.com/confluentinc/cc-structs/kafka/scheduler/v1"
	ckafka "github.com/confluentinc/confluent-kafka-go/kafka"
	"github.com/confluentinc/go-printer"
	"github.com/google/uuid"
	"github.com/spf13/cobra"

	sr "github.com/confluentinc/cli/internal/cmd/schema-registry"
	serdes "github.com/confluentinc/cli/internal/pkg/serdes"

	pcmd "github.com/confluentinc/cli/internal/pkg/cmd"
	"github.com/confluentinc/cli/internal/pkg/errors"
	"github.com/confluentinc/cli/internal/pkg/examples"
	kafka "github.com/confluentinc/cli/internal/pkg/kafka"
	"github.com/confluentinc/cli/internal/pkg/log"
	"github.com/confluentinc/cli/internal/pkg/output"
	"github.com/confluentinc/cli/internal/pkg/utils"
)

const (
	defaultReplicationFactor = 3
)

type kafkaTopicCommand struct {
	*hasAPIKeyTopicCommand
	*authenticatedTopicCommand
}

type hasAPIKeyTopicCommand struct {
	*pcmd.HasAPIKeyCLICommand
	prerunner pcmd.PreRunner
	logger    *log.Logger
	clientID  string
}
type authenticatedTopicCommand struct {
	*pcmd.AuthenticatedStateFlagCommand
	logger              *log.Logger
	clientID            string
	completableChildren []*cobra.Command
}

type partitionDescribeDisplay struct {
	Topic     string   `json:"topic" yaml:"topic"`
	Partition uint32   `json:"partition" yaml:"partition"`
	Leader    uint32   `json:"leader" yaml:"leader"`
	Replicas  []uint32 `json:"replicas" yaml:"replicas"`
	ISR       []uint32 `json:"isr" yaml:"isr"`
}

type structuredDescribeDisplay struct {
	TopicName         string                     `json:"topic_name" yaml:"topic_name"`
	PartitionCount    int                        `json:"partition_count" yaml:"partition_count"`
	ReplicationFactor int                        `json:"replication_factor" yaml:"replication_factor"`
	Partitions        []partitionDescribeDisplay `json:"partitions" yaml:"partitions"`
	Config            map[string]string          `json:"config" yaml:"config"`
}

type partitionData struct {
	TopicName              string  `json:"topic" yaml:"topic"`
	PartitionId            int32   `json:"partition" yaml:"partition"`
	LeaderBrokerId         int32   `json:"leader" yaml:"leader"`
	ReplicaBrokerIds       []int32 `json:"replicas" yaml:"replicas"`
	InSyncReplicaBrokerIds []int32 `json:"isr" yaml:"isr"`
}

type topicData struct {
	TopicName         string            `json:"topic_name" yaml:"topic_name"`
	PartitionCount    int               `json:"partition_count" yaml:"partition_count"`
	ReplicationFactor int               `json:"replication_factor" yaml:"replication_factor"`
	Partitions        []partitionData   `json:"partitions" yaml:"partitions"`
	Configs           map[string]string `json:"config" yaml:"config"`
}

// NewTopicCommand returns the Cobra command for Kafka topic.
func NewTopicCommand(isAPIKeyLogin bool, prerunner pcmd.PreRunner, logger *log.Logger, clientID string) *kafkaTopicCommand {
	command := &cobra.Command{
		Use:   "topic",
		Short: "Manage Kafka topics.",
	}
	hasAPIKeyCmd := &hasAPIKeyTopicCommand{
		HasAPIKeyCLICommand: pcmd.NewHasAPIKeyCLICommand(command, prerunner, ProduceAndConsumeFlags),
		prerunner:           prerunner,
		logger:              logger,
		clientID:            clientID,
	}
	hasAPIKeyCmd.init()
	kafkaTopicCommand := &kafkaTopicCommand{
		hasAPIKeyTopicCommand: hasAPIKeyCmd,
	}
	if !isAPIKeyLogin {
		authenticatedCmd := &authenticatedTopicCommand{
			AuthenticatedStateFlagCommand: pcmd.NewAuthenticatedStateFlagCommand(command, prerunner, TopicSubcommandFlags),
			logger:                        logger,
			clientID:                      clientID,
		}
		authenticatedCmd.init()
		kafkaTopicCommand.authenticatedTopicCommand = authenticatedCmd
	}
	return kafkaTopicCommand
}

func (k *kafkaTopicCommand) Cmd() *cobra.Command {
	return k.hasAPIKeyTopicCommand.Command
}

func (k *kafkaTopicCommand) ServerComplete() []prompt.Suggest {
	var suggestions []prompt.Suggest
	cmd := k.authenticatedTopicCommand
	if cmd == nil {
		return suggestions
	}
	topics, err := cmd.getTopics(cmd.Command)
	if err != nil {
		return suggestions
	}
	for _, topic := range topics {
		description := ""
		if topic.Internal {
			description = "Internal"
		}
		suggestions = append(suggestions, prompt.Suggest{
			Text:        topic.Name,
			Description: description,
		})
	}
	return suggestions
}

func (k *kafkaTopicCommand) ServerCompletableChildren() []*cobra.Command {
	return k.completableChildren
}

func (h *hasAPIKeyTopicCommand) init() {
	cmd := &cobra.Command{
		Use:   "produce <topic>",
		Short: "Produce messages to a Kafka topic.",
		Args:  cobra.ExactArgs(1),
		RunE:  pcmd.NewCLIRunE(h.produce),
	}
	cmd.Flags().String("delimiter", ":", "The key/value delimiter.")
	cmd.Flags().String("value-format", "string", "Format of message value as string, avro, protobuf, or jsonschema.")
	cmd.Flags().String("schema", "", "The path to the schema file.")
	cmd.Flags().Bool("parse-key", false, "Parse key from the message.")
	cmd.Flags().String("sr-endpoint", "", "Endpoint for Schema Registry cluster.")
	cmd.Flags().StringP(output.FlagName, output.ShortHandFlag, output.DefaultValue, output.Usage)
	cmd.Flags().SortFlags = false
	h.AddCommand(cmd)

	cmd = &cobra.Command{
		Use:   "consume <topic>",
		Short: "Consume messages from a Kafka topic.",
		Args:  cobra.ExactArgs(1),
		RunE:  pcmd.NewCLIRunE(h.consume),
		Example: examples.BuildExampleString(
			examples.Example{
				Text: "Consume items from the `my_topic` topic and press `Ctrl+C` to exit.",
				Code: "ccloud kafka topic consume -b my_topic",
			},
		),
	}
	cmd.Flags().String("group", fmt.Sprintf("confluent_cli_consumer_%s", uuid.New()), "Consumer group ID.")
	cmd.Flags().BoolP("from-beginning", "b", false, "Consume from beginning of the topic.")
	cmd.Flags().String("value-format", "string", "Format of message value as string, avro, protobuf, or jsonschema.")
	cmd.Flags().Bool("print-key", false, "Print key of the message.")
	cmd.Flags().String("delimiter", "\t", "The key/value delimiter.")
	cmd.Flags().String("sr-endpoint", "", "Endpoint for Schema Registry cluster.")
	cmd.Flags().SortFlags = false
	h.AddCommand(cmd)
}

func (a *authenticatedTopicCommand) init() {
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List Kafka topics.",
		Args:  cobra.NoArgs,
		RunE:  pcmd.NewCLIRunE(a.list),
		Example: examples.BuildExampleString(
			examples.Example{
				Text: "List all topics.",
				Code: "ccloud kafka topic list",
			},
		),
	}
	listCmd.Flags().StringP(output.FlagName, output.ShortHandFlag, output.DefaultValue, output.Usage)
	listCmd.Flags().SortFlags = false
	a.AddCommand(listCmd)

	createCmd = &cobra.Command{
		Use:   "create <topic>",
		Short: "Create a Kafka topic.",
		Args:  cobra.ExactArgs(1),
		RunE:  pcmd.NewCLIRunE(a.create),
		Example: examples.BuildExampleString(
			examples.Example{
				Text: "Create a topic named `my_topic` with default options.",
				Code: "ccloud kafka topic create my_topic",
			},
		),
	}
	createCmd.Flags().Int32("partitions", 6, "Number of topic partitions.")
	createCmd.Flags().StringSlice("config", nil, "A comma-separated list of configuration overrides ('key=value') for the topic being created.")
	createCmd.Flags().Bool("dry-run", false, "Run the command without committing changes to Kafka.")
	createCmd.Flags().Bool("if-not-exists", false, "Exit gracefully if topic already exists.")
	createCmd.Flags().SortFlags = false
	a.AddCommand(createCmd)

	describeCmd := &cobra.Command{
		Use:   "describe <topic>",
		Short: "Describe a Kafka topic.",
		Args:  cobra.ExactArgs(1),
		RunE:  pcmd.NewCLIRunE(a.describe),
		Example: examples.BuildExampleString(
			examples.Example{
				Text: "Describe the `my_topic` topic.",
				Code: "ccloud kafka topic describe my_topic",
			},
		),
	}
	describeCmd.Flags().StringP(output.FlagName, output.ShortHandFlag, output.DefaultValue, output.Usage)
	describeCmd.Flags().SortFlags = false
	a.AddCommand(describeCmd)

	updateCmd := &cobra.Command{
		Use:   "update <topic>",
		Short: "Update a Kafka topic.",
		Args:  cobra.ExactArgs(1),
		RunE:  pcmd.NewCLIRunE(a.update),
		Example: examples.BuildExampleString(
			examples.Example{
				Text: "Modify the `my_topic` topic to have a retention period of 3 days (259200000 milliseconds).",
				Code: `ccloud kafka topic update my_topic --config="retention.ms=259200000"`,
			},
		),
	}
	updateCmd.Flags().StringSlice("config", nil, "A comma-separated list of topics. Configuration ('key=value') overrides for the topic being created.")
	updateCmd.Flags().Bool("dry-run", false, "Execute request without committing changes to Kafka.")
	updateCmd.Flags().SortFlags = false
	a.AddCommand(updateCmd)

	deleteCmd := &cobra.Command{
		Use:   "delete <topic>",
		Short: "Delete a Kafka topic.",
		Args:  cobra.ExactArgs(1),
		RunE:  pcmd.NewCLIRunE(a.delete),
		Example: examples.BuildExampleString(
			examples.Example{
				Text: "Delete the topics `my_topic` and `my_topic_avro`. Use this command carefully as data loss can occur.",
				Code: "ccloud kafka topic delete my_topic\nccloud kafka topic delete my_topic_avro",
			},
		),
	}
	a.AddCommand(deleteCmd)

	a.completableChildren = []*cobra.Command{describeCmd, updateCmd, deleteCmd}
}

func (a *authenticatedTopicCommand) list(cmd *cobra.Command, _ []string) error {
	kafkaREST, _ := a.GetKafkaREST()
	if kafkaREST != nil {
		kafkaClusterConfig, err := a.AuthenticatedCLICommand.Context.GetKafkaClusterForCommand(cmd)
		if err != nil {
			return err
		}
		lkc := kafkaClusterConfig.ID

		topicGetResp, httpResp, err := kafkaREST.Client.TopicApi.ClustersClusterIdTopicsGet(kafkaREST.Context, lkc)

		if err != nil && httpResp != nil {
			// Kafka REST is available, but an error occurred
			return kafkaRestError(kafkaREST.Client.GetConfig().BasePath, err, httpResp)
		}

		if err == nil && httpResp != nil {
			if httpResp.StatusCode != http.StatusOK {
				return errors.NewErrorWithSuggestions(
					fmt.Sprintf(errors.KafkaRestUnexpectedStatusMsg, httpResp.Request.URL, httpResp.StatusCode),
					errors.InternalServerErrorSuggestions)
			}
			// Kafka REST is available and there was no error
			outputWriter, err := output.NewListOutputWriter(cmd, []string{"TopicName"}, []string{"Name"}, []string{"name"})
			if err != nil {
				return err
			}
			for _, topicData := range topicGetResp.Data {
				outputWriter.AddElement(&topicData)
			}
			return outputWriter.Out()
		}
	}

	// Kafka REST is not available, fall back to KafkaAPI

	resp, err := a.getTopics(cmd)
	if err != nil {
		return err
	}
	outputWriter, err := output.NewListOutputWriter(cmd, []string{"Name"}, []string{"Name"}, []string{"name"})
	if err != nil {
		return err
	}
	for _, topic := range resp {
		outputWriter.AddElement(topic)
	}
	return outputWriter.Out()
}

func (a *authenticatedTopicCommand) create(cmd *cobra.Command, args []string) error {
	topicName := args[0]

	numPartitions, err := cmd.Flags().GetInt32("partitions")
	if err != nil {
		return err
	}

	configs, err := cmd.Flags().GetStringSlice("config")
	if err != nil {
		return err
	}
	topicConfigsMap, err := kafka.ToMap(configs)
	if err != nil {
		return err
	}

	dryRun, err := cmd.Flags().GetBool("dry-run")
	if err != nil {
		return err
	}

	ifNotExistsFlag, err := cmd.Flags().GetBool("if-not-exists")
	if err != nil {
		return err
	}

	kafkaREST, _ := a.GetKafkaREST()
	if kafkaREST != nil && !dryRun {
		topicConfigs := make([]kafkarestv3.CreateTopicRequestDataConfigs, len(topicConfigsMap))
		i := 0
		for k, v := range topicConfigsMap {
			val := v
			topicConfigs[i] = kafkarestv3.CreateTopicRequestDataConfigs{
				Name:  k,
				Value: &val,
			}
			i++
		}

		kafkaClusterConfig, err := a.AuthenticatedCLICommand.Context.GetKafkaClusterForCommand(cmd)
		if err != nil {
			return err
		}
		lkc := kafkaClusterConfig.ID

		_, httpResp, err := kafkaREST.Client.TopicApi.ClustersClusterIdTopicsPost(kafkaREST.Context, lkc, &kafkarestv3.ClustersClusterIdTopicsPostOpts{
			CreateTopicRequestData: optional.NewInterface(kafkarestv3.CreateTopicRequestData{
				TopicName:         topicName,
				PartitionsCount:   numPartitions,
				ReplicationFactor: defaultReplicationFactor,
				Configs:           topicConfigs,
			}),
		})

		if err != nil && httpResp != nil {
			// Kafka REST is available, but there was an error
			restErr, parseErr := parseOpenAPIError(err)
			if parseErr == nil {
				if restErr.Code == KafkaRestBadRequestErrorCode {
					// Ignore or pretty print topic exists error
					if !ifNotExistsFlag {
						return errors.NewErrorWithSuggestions(
							fmt.Sprintf(errors.TopicExistsErrorMsg, topicName, lkc),
							fmt.Sprintf(errors.TopicExistsSuggestions, lkc, lkc))
					}
					return nil
				}
			}
			return kafkaRestError(kafkaREST.Client.GetConfig().BasePath, err, httpResp)
		}

		if err == nil && httpResp != nil {
			if httpResp.StatusCode != http.StatusCreated {
				return errors.NewErrorWithSuggestions(
					fmt.Sprintf(errors.KafkaRestUnexpectedStatusMsg, httpResp.Request.URL, httpResp.StatusCode),
					errors.InternalServerErrorSuggestions)
			}
			// Kafka REST is available and there was no error
			utils.Printf(cmd, errors.CreatedTopicMsg, topicName)
			return nil
		}
	}

	// Kafka REST is not available, fall back to KafkaAPI

	cluster, err := pcmd.KafkaCluster(cmd, a.Context)
	if err != nil {
		return err
	}

	topic := &schedv1.Topic{
		Spec: &schedv1.TopicSpecification{
			Configs: make(map[string]string)},
		Validate: false,
	}

	topic.Spec.Name = topicName
	topic.Spec.NumPartitions = numPartitions
	topic.Spec.ReplicationFactor = defaultReplicationFactor
	topic.Validate = dryRun
	topic.Spec.Configs = topicConfigsMap

	if err := a.Client.Kafka.CreateTopic(context.Background(), cluster, topic); err != nil {
		err = errors.CatchTopicExistsError(err, cluster.Id, topic.Spec.Name, ifNotExistsFlag)
		err = errors.CatchClusterNotReadyError(err, cluster.Id)
		return err
	}
	utils.Printf(cmd, errors.CreatedTopicMsg, topic.Spec.Name)
	return nil
}

func (a *authenticatedTopicCommand) describe(cmd *cobra.Command, args []string) error {
	topicName := args[0]

	outputOption, err := cmd.Flags().GetString(output.FlagName)
	if err != nil {
		return err
	}

	if !output.IsValidFormatString(outputOption) {
		return output.NewInvalidOutputFormatFlagError(outputOption)
	}

	kafkaREST, _ := a.GetKafkaREST()
	if kafkaREST != nil {
		kafkaClusterConfig, err := a.AuthenticatedCLICommand.Context.GetKafkaClusterForCommand(cmd)
		if err != nil {
			return err
		}
		lkc := kafkaClusterConfig.ID

		partitionsResp, httpResp, err := kafkaREST.Client.PartitionApi.ClustersClusterIdTopicsTopicNamePartitionsGet(kafkaREST.Context, lkc, topicName)

		if err != nil && httpResp != nil {
			// Kafka REST is available, but there was an error
			restErr, parseErr := parseOpenAPIError(err)
			if parseErr == nil {
				if restErr.Code == KafkaRestUnknownTopicOrPartitionErrorCode {
					return fmt.Errorf(errors.UnknownTopicMsg, topicName)
				}
			}
			return kafkaRestError(kafkaREST.Client.GetConfig().BasePath, err, httpResp)
		}

		if err == nil && httpResp != nil {
			if httpResp.StatusCode != http.StatusOK {
				return errors.NewErrorWithSuggestions(
					fmt.Sprintf(errors.KafkaRestUnexpectedStatusMsg, httpResp.Request.URL, httpResp.StatusCode),
					errors.InternalServerErrorSuggestions)
			}

			// Kafka REST is available and there was no error. Fetch partition and config information.

			topicData := &topicData{}
			topicData.TopicName = topicName
			topicData.PartitionCount = len(partitionsResp.Data)
			topicData.Partitions = make([]partitionData, len(partitionsResp.Data))
			replicaStatusDataList, httpResp, err := kafkaREST.Client.ReplicaStatusApi.ClustersClusterIdTopicsTopicNamePartitionsReplicaStatusGet(kafkaREST.Context, lkc, topicName)
			if err != nil {
				return kafkaRestError(kafkaREST.Client.GetConfig().BasePath, err, httpResp)
			} else if replicaStatusDataList.Data == nil {
				return errors.NewErrorWithSuggestions(errors.EmptyResponseMsg, errors.InternalServerErrorSuggestions)
			}
			partitionIdToData := make(map[int32]partitionData)
			for _, replica := range replicaStatusDataList.Data {
				if _, ok := partitionIdToData[replica.PartitionId]; !ok {
					partitionIdToData[replica.PartitionId] = partitionData{
						TopicName:              replica.TopicName,
						PartitionId:            replica.PartitionId,
						ReplicaBrokerIds:       []int32{replica.BrokerId},
						InSyncReplicaBrokerIds: []int32{},
					}
				} else {
					tmp := partitionIdToData[replica.PartitionId]
					tmp.ReplicaBrokerIds = append(partitionIdToData[replica.PartitionId].ReplicaBrokerIds, replica.BrokerId)
					partitionIdToData[replica.PartitionId] = tmp
				}
				if replica.IsLeader {
					tmp := partitionIdToData[replica.PartitionId]
					tmp.LeaderBrokerId = replica.BrokerId
					partitionIdToData[replica.PartitionId] = tmp
				}
				if replica.IsInIsr {
					tmp := partitionIdToData[replica.PartitionId]
					tmp.InSyncReplicaBrokerIds = append(partitionIdToData[replica.PartitionId].InSyncReplicaBrokerIds, replica.BrokerId)
					partitionIdToData[replica.PartitionId] = tmp
				}
			}

			for i, data := range partitionsResp.Data {
				if i == 0 {
					topicData.ReplicationFactor = len(partitionIdToData[data.PartitionId].ReplicaBrokerIds)
				}
				topicData.Partitions[i] = partitionIdToData[data.PartitionId]
			}

			// Get topic config
			configsResp, httpResp, err := kafkaREST.Client.ConfigsApi.ClustersClusterIdTopicsTopicNameConfigsGet(kafkaREST.Context, lkc, topicName)
			if err != nil {
				return kafkaRestError(kafkaREST.Client.GetConfig().BasePath, err, httpResp)
			} else if configsResp.Data == nil {
				return errors.NewErrorWithSuggestions(errors.EmptyResponseMsg, errors.InternalServerErrorSuggestions)
			}
			topicData.Configs = make(map[string]string)
			for _, config := range configsResp.Data {
				topicData.Configs[config.Name] = *config.Value
			}

			if outputOption == output.Human.String() {
				return printHumanDescribe(cmd, topicData)
			}

			return output.StructuredOutput(outputOption, topicData)
		}
	}
	// Kafka REST is not available, fallback to KafkaAPI
	cluster, err := pcmd.KafkaCluster(cmd, a.Context)
	if err != nil {
		return err
	}

	topic := &schedv1.TopicSpecification{Name: topicName}
	resp, err := a.Client.Kafka.DescribeTopic(context.Background(), cluster, &schedv1.Topic{Spec: topic, Validate: false})
	if err != nil {
		return err
	}

	if outputOption == output.Human.String() {
		return printHumanTopicDescription(cmd, resp)
	} else {
		return printStructuredTopicDescription(resp, outputOption)
	}
}

func (a *authenticatedTopicCommand) update(cmd *cobra.Command, args []string) error {
	topicName := args[0]

	configStrings, err := cmd.Flags().GetStringSlice("config")
	if err != nil {
		return err
	}
	configsMap, err := kafka.ToMap(configStrings)
	if err != nil {
		return err
	}

	dryRun, err := cmd.Flags().GetBool("dry-run")
	if err != nil {
		return err
	}

	kafkaREST, _ := a.GetKafkaREST()
	if kafkaREST != nil && !dryRun {
		kafkaRestConfigs := toAlterConfigBatchRequestData(configsMap)

		kafkaClusterConfig, err := a.AuthenticatedCLICommand.Context.GetKafkaClusterForCommand(cmd)
		if err != nil {
			return err
		}
		lkc := kafkaClusterConfig.ID

		httpResp, err := kafkaREST.Client.ConfigsApi.ClustersClusterIdTopicsTopicNameConfigsalterPost(kafkaREST.Context, lkc, topicName,
			&kafkarestv3.ClustersClusterIdTopicsTopicNameConfigsalterPostOpts{
				AlterConfigBatchRequestData: optional.NewInterface(kafkarestv3.AlterConfigBatchRequestData{Data: kafkaRestConfigs}),
			})

		if err != nil && httpResp != nil {
			// Kafka REST is available, but an error occurred
			restErr, parseErr := parseOpenAPIError(err)
			if parseErr == nil {
				if restErr.Code == KafkaRestUnknownTopicOrPartitionErrorCode {
					return fmt.Errorf(errors.UnknownTopicMsg, topicName)
				}
			}
			return kafkaRestError(kafkaREST.Client.GetConfig().BasePath, err, httpResp)
		}

		if err == nil && httpResp != nil {
			if httpResp.StatusCode != http.StatusNoContent {
				return errors.NewErrorWithSuggestions(
					fmt.Sprintf(errors.KafkaRestUnexpectedStatusMsg, httpResp.Request.URL, httpResp.StatusCode),
					errors.InternalServerErrorSuggestions)
			}
			// Kafka REST is available and there was no error
			utils.Printf(cmd, errors.UpdateTopicConfigMsg, topicName)
			tableLabels := []string{"Name", "Value"}
			tableEntries := make([][]string, len(kafkaRestConfigs))
			for i, config := range kafkaRestConfigs {
				tableEntries[i] = printer.ToRow(
					&struct {
						Name  string
						Value string
					}{Name: config.Name, Value: *config.Value}, []string{"Name", "Value"})
			}
			sort.Slice(tableEntries, func(i int, j int) bool {
				return tableEntries[i][0] < tableEntries[j][0]
			})
			printer.RenderCollectionTable(tableEntries, tableLabels)
			return nil
		}
	}

	// Kafka REST is not available, fallback to KafkaAPI

	cluster, err := pcmd.KafkaCluster(cmd, a.Context)
	if err != nil {
		return err
	}

	topic := &schedv1.TopicSpecification{Name: args[0], Configs: make(map[string]string)}

	configs, err := cmd.Flags().GetStringSlice("config")
	if err != nil {
		return err
	}

	configMap, err := kafka.ToMap(configs)
	if err != nil {
		return err
	}
	topic.Configs = copyMap(configMap)

	err = a.Client.Kafka.UpdateTopic(context.Background(), cluster, &schedv1.Topic{Spec: topic, Validate: dryRun})
	if err != nil {
		err = errors.CatchClusterNotReadyError(err, cluster.Id)
		return err
	}
	utils.Printf(cmd, errors.UpdateTopicConfigMsg, args[0])
	var entries [][]string
	titleRow := []string{"Name", "Value"}
	for name, value := range configMap {
		record := &struct {
			Name  string
			Value string
		}{
			name,
			value,
		}
		entries = append(entries, printer.ToRow(record, titleRow))
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i][0] < entries[j][0]
	})
	printer.RenderCollectionTable(entries, titleRow)
	return nil
}

func (a *authenticatedTopicCommand) delete(cmd *cobra.Command, args []string) error {
	topicName := args[0]

	kafkaREST, _ := a.GetKafkaREST()
	if kafkaREST != nil {
		kafkaClusterConfig, err := a.AuthenticatedCLICommand.Context.GetKafkaClusterForCommand(cmd)
		if err != nil {
			return err
		}
		lkc := kafkaClusterConfig.ID

		httpResp, err := kafkaREST.Client.TopicApi.ClustersClusterIdTopicsTopicNameDelete(kafkaREST.Context, lkc, topicName)
		if err != nil && httpResp != nil {
			// Kafka REST is available, but an error occurred
			restErr, parseErr := parseOpenAPIError(err)
			if parseErr == nil {
				if restErr.Code == KafkaRestUnknownTopicOrPartitionErrorCode {
					return fmt.Errorf(errors.UnknownTopicMsg, topicName)
				}
			}
			return kafkaRestError(kafkaREST.Client.GetConfig().BasePath, err, httpResp)
		}

		if err == nil && httpResp != nil {
			if httpResp.StatusCode != http.StatusNoContent {
				return errors.NewErrorWithSuggestions(
					fmt.Sprintf(errors.KafkaRestUnexpectedStatusMsg, httpResp.Request.URL, httpResp.StatusCode),
					errors.InternalServerErrorSuggestions)
			}
			// Topic succesfully deleted
			utils.Printf(cmd, errors.DeletedTopicMsg, topicName)
			return nil
		}
	}

	// Kafka REST is not available, fallback to KafkaAPI
	cluster, err := pcmd.KafkaCluster(cmd, a.Context)
	if err != nil {
		return err
	}

	topic := &schedv1.TopicSpecification{Name: topicName}
	err = a.Client.Kafka.DeleteTopic(context.Background(), cluster, &schedv1.Topic{Spec: topic, Validate: false})
	if err != nil {
		err = errors.CatchClusterNotReadyError(err, cluster.Id)
		return err
	}
	utils.Printf(cmd, errors.DeletedTopicMsg, topicName)
	return nil
}

func (h *hasAPIKeyTopicCommand) registerSchema(cmd *cobra.Command, subject string, valueFormat string, schemaPath string) ([]byte, error) {
	schema, err := ioutil.ReadFile(schemaPath)
	if err != nil {
		return nil, err
	}
	var refs []srsdk.SchemaReference

	srClient, ctx, err := sr.GetApiClient(cmd, nil, h.Config, h.Version)
	if err != nil {
		if err.Error() == "ccloud" {
			return nil, &errors.SRNotAuthenticatedError{CLIName: err.Error()}
		} else {
			return nil, err
		}
	}

	response, _, err := srClient.DefaultApi.Register(ctx, subject, srsdk.RegisterSchemaRequest{Schema: string(schema), SchemaType: valueFormat, References: refs})
	if err != nil {
		return nil, err
	}

	outputFormat, err := cmd.Flags().GetString(output.FlagName)
	if err != nil {
		return nil, err
	}
	if outputFormat == output.Human.String() {
		utils.Printf(cmd, errors.RegisteredSchemaMsg, response.Id)
	} else {
		err = output.StructuredOutput(outputFormat, &struct {
			Id int32 `json:"id" yaml:"id"`
		}{response.Id})
		if err != nil {
			return nil, err
		}
	}

	metaInfo := []byte{0x0}
	schemaIdBuffer := make([]byte, 4)
	binary.BigEndian.PutUint32(schemaIdBuffer, uint32(response.Id))
	metaInfo = append(metaInfo, schemaIdBuffer...)
	return metaInfo, nil
}

func (h *hasAPIKeyTopicCommand) produce(cmd *cobra.Command, args []string) error {
	topic := args[0]
	cluster, err := h.Context.GetKafkaClusterForCommand(cmd)
	if err != nil {
		return err
	}

	producer, err := NewProducer(cluster, h.clientID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create producer: %s\n", err)
		os.Exit(1)
	}
	defer producer.Close()
	adminClient, err := ckafka.NewAdminClientFromProducer(producer)
	if err != nil {
		return err
	}
	err = validateTopic(adminClient, topic, cluster)
	if err != nil {
		return err
	}

	delim, err := cmd.Flags().GetString("delimiter")
	if err != nil {
		return err
	}

	valueFormat, err := cmd.Flags().GetString("value-format")
	if err != nil {
		return err
	}

	schemaPath, err := cmd.Flags().GetString("schema")
	if err != nil {
		return err
	}

	parseKey, err := cmd.Flags().GetBool("parse-key")
	if err != nil {
		return err
	}

	subject := topic + "-value"
	serializationProvider, err := serdes.GetSerializationProvider(valueFormat)
	if err != nil {
		return err
	}
	err = serializationProvider.LoadSchema(schemaPath)
	if err != nil {
		return err
	}

	// Meta info contains magic byte and schema ID (4 bytes).
	// For plain string encoding, meta info is empty.
	metaInfo := []byte{}

	// Registering schema when specified, and fill metaInfo array.
	if valueFormat != "string" && len(schemaPath) > 0 {
		info, err := h.registerSchema(cmd, subject, serializationProvider.GetSchemaName(), schemaPath)
		if err != nil {
			return err
		}
		metaInfo = info
	}

	utils.ErrPrintln(cmd, errors.StartingProducerMsg)

	// Line reader for producer input.
	scanner := bufio.NewScanner(os.Stdin)
	// CCloud Kafka messageMaxBytes:
	// https://github.com/confluentinc/cc-spec-kafka/blob/9f0af828d20e9339aeab6991f32d8355eb3f0776/plugins/kafka/kafka.go#L43.
	const maxScanTokenSize = 1024*1024*2 + 12
	scanner.Buffer(nil, maxScanTokenSize)
	input := make(chan string, 1)
	// Avoid blocking in for loop so ^C or ^D can exit immediately.
	var scanErr error
	scan := func() {
		hasNext := scanner.Scan()
		if !hasNext {
			// Actual error.
			if scanner.Err() != nil {
				scanErr = scanner.Err()
			}
			// Otherwise just EOF.
			close(input)
		} else {
			input <- scanner.Text()
		}
	}

	// Trap SIGINT to trigger a shutdown.
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt)
	go func() {
		<-signals
		close(input)
	}()
	// Prime reader
	go scan()

	var key, valueString string
	deliveryChan := make(chan ckafka.Event)
	for data := range input {
		if len(data) == 0 {
			go scan()
			continue
		}
		if parseKey {
			record := strings.SplitN(data, delim, 2)
			valueString = strings.TrimSpace(record[len(record)-1])

			if len(record) == 2 {
				key = strings.TrimSpace(record[0]) // sarama StringEncoder.
			} else {
				return errors.New(errors.MissingKeyErrorMsg)
			}
		} else {
			valueString = strings.TrimSpace(data)
		}
		encodedMessage, err := serdes.Serialize(serializationProvider, valueString)
		if err != nil {
			return err
		}
		encoded := append(metaInfo, encodedMessage...)
		value := string(encoded) // sarama StringEncoder.

		msg := &ckafka.Message{
			TopicPartition: ckafka.TopicPartition{Topic: &topic, Partition: ckafka.PartitionAny},
			Key:            []byte(key),
			Value:          []byte(value),
		}
		err = producer.Produce(msg, deliveryChan)
		if err != nil {
			isTopicNotExistError, err := errors.CatchTopicNotExistError(err, topic, cluster.ID)
			if isTopicNotExistError {
				scanErr = err
				close(input)
				break
			}
			isProduceToCompactedTopicError, err := errors.CatchProduceToCompactedTopicError(err, topic)
			if isProduceToCompactedTopicError {
				scanErr = err
				close(input)
				break
			}
		}

		// Reset key prior to reuse
		e := <-deliveryChan
		m := e.(*ckafka.Message)
		if m.TopicPartition.Error != nil { // catch all other errors
			utils.ErrPrintf(cmd, errors.FailedToProduceErrorMsg, m.TopicPartition.Offset, m.TopicPartition.Error)
		}
		key = ""
		go scan()
	}
	close(deliveryChan)
	if scanErr != nil {
		return scanErr
	}
	return nil
}

func (h *hasAPIKeyTopicCommand) consume(cmd *cobra.Command, args []string) error {
	topic := args[0]
	beginning, err := cmd.Flags().GetBool("from-beginning")
	if err != nil {
		return err
	}

	valueFormat, err := cmd.Flags().GetString("value-format")
	if err != nil {
		return err
	}

	cluster, err := h.Context.GetKafkaClusterForCommand(cmd)
	if err != nil {
		return err
	}

	group, err := cmd.Flags().GetString("group")
	if err != nil {
		return err
	}

	printKey, err := cmd.Flags().GetBool("print-key")
	if err != nil {
		return err
	}

	delimiter, err := cmd.Flags().GetString("delimiter")
	if err != nil {
		return err
	}

	var srClient *srsdk.APIClient
	var ctx context.Context
	if valueFormat != "string" {
		// Only initialize client and context when schema is specified.
		srClient, ctx, err = sr.GetApiClient(cmd, nil, h.Config, h.Version)
		if err != nil {
			if err.Error() == "ccloud" {
				return &errors.SRNotAuthenticatedError{CLIName: err.Error()}
			} else {
				return err
			}
		}
	} else {
		srClient, ctx = nil, nil
	}
	fmt.Println("srClient, ctx", srClient, ctx)

	consumer, err := NewConsumer(group, cluster, h.clientID, beginning)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create consumer: %s\n", err)
		os.Exit(1)
	}
	adminClient, err := ckafka.NewAdminClientFromConsumer(consumer)
	if err != nil {
		return err
	}
	err = validateTopic(adminClient, topic, cluster)
	if err != nil {
		return err
	}

	utils.ErrPrintln(cmd, errors.StartingConsumerMsg)

	dir := filepath.Join(os.TempDir(), "ccloud-schema")
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		err = os.Mkdir(dir, 0755)
		if err != nil {
			return err
		}
	}

	err = consumer.Subscribe(topic, nil)
	_, err = errors.CatchTopicNotExistError(err, topic, cluster.ID) // should have been validated by err = validateTopic(adminClient, topic, cluster)
	if err != nil {
		return err
	}
	// start consuming msg
	run := true
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt)
	for run {
		select {
		case <-signals: // Trap SIGINT to trigger a shutdown.
			utils.ErrPrintln(cmd, errors.StoppingConsumer)
			consumer.Close()
			run = false
		default:
			ev := consumer.Poll(100)
			if ev == nil {
				continue
			}
			switch e := ev.(type) {
			case *ckafka.Message:
				if printKey {
					key := e.Key
					var keyString string
					if len(key) == 0 {
						keyString = "null"
					} else {
						keyString = string(key)
					}
					fmt.Printf("%s", keyString)
				}

				if delimiter != "" {
					fmt.Printf("%s", delimiter)
				}

				fmt.Printf("%s", string(e.Value))

				if e.Headers != nil {
					fmt.Printf("%% Headers: %v\n", e.Headers)
				}
			case ckafka.Error:
				fmt.Fprintf(os.Stderr, "%% Error: %v: %v\n", e.Code(), e)
				if e.Code() == ckafka.ErrAllBrokersDown {
					run = false
				}
			}
		}
	}
	err = os.RemoveAll(dir)
	return err
}

// validate that a topic exists before attempting to produce/consume messages
func validateTopic(client *ckafka.AdminClient, topic string, cluster *v1.KafkaClusterConfig) error {
	metaData, err := client.GetMetadata(nil, true, 60*1000)
	if err != nil {
		return err
	}
	var foundTopic bool
	for _, t := range metaData.Topics {
		if topic == t.Topic {
			foundTopic = true
			break
		}
	}
	if !foundTopic {
		return errors.NewErrorWithSuggestions(fmt.Sprintf(errors.TopicNotExistsErrorMsg, topic), fmt.Sprintf(errors.TopicNotExistsSuggestions, cluster.ID, cluster.ID))
	}
	return nil
}

func printHumanDescribe(cmd *cobra.Command, topicData *topicData) error {
	utils.Printf(cmd, "Topic: %s PartitionCount: %d ReplicationFactor: %d\n",
		topicData.TopicName, topicData.PartitionCount, topicData.ReplicationFactor)
	partitionsTableLabels := []string{"Topic", "Partition", "Leader", "Replicas", "ISR"}
	partitionsTableEntries := make([][]string, topicData.PartitionCount)
	for i, partition := range topicData.Partitions {
		partitionsTableEntries[i] = printer.ToRow(&partition, []string{"TopicName", "PartitionId", "LeaderBrokerId", "ReplicaBrokerIds", "InSyncReplicaBrokerIds"})
	}
	printer.RenderCollectionTable(partitionsTableEntries, partitionsTableLabels)

	utils.Print(cmd, "\nConfiguration\n\n")
	configsTableLabels := []string{"Name", "Value"}
	configsTableEntries := make([][]string, len(topicData.Configs))
	i := 0
	for name, value := range topicData.Configs {
		configsTableEntries[i] = printer.ToRow(&struct {
			name  string
			value string
		}{name: name, value: value}, []string{"name", "value"})
		i++
	}
	sort.Slice(configsTableEntries, func(i int, j int) bool {
		return configsTableEntries[i][0] < configsTableEntries[j][0]
	})
	printer.RenderCollectionTable(configsTableEntries, configsTableLabels)
	return nil
}

func printHumanTopicDescription(cmd *cobra.Command, resp *schedv1.TopicDescription) error {
	utils.Printf(cmd, "Topic: %s PartitionCount: %d ReplicationFactor: %d\n",
		resp.Name, len(resp.Partitions), len(resp.Partitions[0].Replicas))

	var partitions [][]string
	titleRow := []string{"Topic", "Partition", "Leader", "Replicas", "ISR"}
	for _, partition := range resp.Partitions {
		partitions = append(partitions, printer.ToRow(getPartitionDisplay(partition, resp.Name), titleRow))
	}

	printer.RenderCollectionTable(partitions, titleRow)

	var entries [][]string
	titleRow = []string{"Name", "Value"}
	for _, entry := range resp.Config {
		record := &struct {
			Name  string
			Value string
		}{
			entry.Name,
			entry.Value,
		}
		entries = append(entries, printer.ToRow(record, titleRow))
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i][0] < entries[j][0]
	})
	utils.Print(cmd, "\nConfiguration\n\n")
	printer.RenderCollectionTable(entries, titleRow)
	return nil
}

func printStructuredTopicDescription(resp *schedv1.TopicDescription, format string) error {
	structuredDisplay := &structuredDescribeDisplay{Config: make(map[string]string)}
	structuredDisplay.TopicName = resp.Name
	structuredDisplay.PartitionCount = len(resp.Partitions)
	structuredDisplay.ReplicationFactor = len(resp.Partitions[0].Replicas)

	var partitionList []partitionDescribeDisplay
	for _, partition := range resp.Partitions {
		partitionList = append(partitionList, *getPartitionDisplay(partition, resp.Name))
	}
	structuredDisplay.Partitions = partitionList

	for _, entry := range resp.Config {
		structuredDisplay.Config[entry.Name] = entry.Value
	}
	return output.StructuredOutput(format, structuredDisplay)
}

func getPartitionDisplay(partition *schedv1.TopicPartitionInfo, topicName string) *partitionDescribeDisplay {
	var replicas []uint32
	for _, replica := range partition.Replicas {
		replicas = append(replicas, replica.Id)
	}

	var isr []uint32
	for _, replica := range partition.Isr {
		isr = append(isr, replica.Id)
	}

	return &partitionDescribeDisplay{
		Topic:     topicName,
		Partition: partition.Partition,
		Leader:    partition.Leader.Id,
		Replicas:  replicas,
		ISR:       isr,
	}
}

func (a *authenticatedTopicCommand) getTopics(cmd *cobra.Command) ([]*schedv1.TopicDescription, error) {
	cluster, err := pcmd.KafkaCluster(cmd, a.Context)
	if err != nil {
		return []*schedv1.TopicDescription{}, err
	}
	resp, err := a.Client.Kafka.ListTopics(context.Background(), cluster)
	if err != nil {
		err = errors.CatchClusterNotReadyError(err, cluster.Id)
	}

	return resp, err
}
