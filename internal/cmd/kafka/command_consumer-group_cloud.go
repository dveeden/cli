package kafka

import (
	"fmt"
	"github.com/c-bata/go-prompt"
	pcmd "github.com/confluentinc/cli/internal/pkg/cmd"
	"github.com/confluentinc/cli/internal/pkg/errors"
	"github.com/confluentinc/cli/internal/pkg/examples"
	"github.com/confluentinc/cli/internal/pkg/output"
	"github.com/confluentinc/cli/internal/pkg/utils"
	"github.com/confluentinc/go-printer"
	"github.com/confluentinc/go-printer/tables"
	"github.com/confluentinc/kafka-rest-sdk-go/kafkarestv3"
	"github.com/spf13/cobra"
	"os"
	"strings"
)

// ahu: description should state 'max lag consumer ID', 'max lag instance ID', etc
var (
	groupListFields  = []string{"ClusterId", "ConsumerGroupId", "IsSimple", "State"}
	groupListHumanLabels = []string{"Cluster", "ConsumerGroup", "Simple", "State"}
	groupListStructuredLabels = []string{"cluster", "consumer_group", "simple", "state"}
	// groupDescribe vars and struct used for human output
	groupDescribeFields = []string{"ClusterId", "ConsumerGroupId", "Coordinator", "IsSimple", "PartitionAssignor", "State"}
	groupDescribeHumanRenames = map[string]string{
		"ClusterId":	   "Cluster",
		"ConsumerGroupId": "ConsumerGroup",
		"IsSimple":        "Simple"}
	groupDescribeConsumersFields = []string{"ConsumerGroupId", "ConsumerId", "InstanceId", "ClientId"}
	groupDescribeConsumerTableLabels = []string{"Consumer Group", "Consumer", "Instance", "Client"}
	lagSummaryFields = []string{"ClusterId", "ConsumerGroupId", "TotalLag", "MaxLag", "MaxLagConsumerId", "MaxLagInstanceId", "MaxLagClientId", "MaxLagTopicName", "MaxLagPartitionId"}
	lagSummaryHumanRenames = map[string]string{
		"ClusterId":		 "Cluster",
		"ConsumerGroupId": 	 "ConsumerGroup",
		"MaxLagConsumerId":  "MaxLagConsumer",
		"MaxLagInstanceId":  "MaxLagInstance",
		"MaxLagClientId":    "MaxLagClient",
		"MaxLagTopicName":   "MaxLagTopic",
		"MaxLagPartitionId": "MaxLagPartition"}
	lagSummaryStructuredRenames = map[string]string{
		"ClusterId":		 "cluster",
		"ConsumerGroupId": 	 "consumer_group",
		"TotalLag":          "total_lag",
		"MaxLag":            "max_lag",
		"MaxLagConsumerId":  "max_lag_consumer",
		"MaxLagInstanceId":  "max_lag_instance",
		"MaxLagClientId":    "max_lag_client",
		"MaxLagTopicName":   "max_lag_topic",
		"MaxLagPartitionId": "max_lag_partition"}
	lagFields = []string{"ClusterId", "ConsumerGroupId", "Lag", "LogEndOffset", "CurrentOffset", "ConsumerId", "InstanceId", "ClientId", "TopicName", "PartitionId"}
	lagListHumanLabels = []string{"Cluster", "ConsumerGroup", "Lag", "LogEndOffset", "CurrentOffset", "Consumer", "Instance", "Client", "Topic", "Partition"}
	lagListStructuredLabels = []string{"cluster", "consumer_group", "lag", "log_end_offset", "current_offset", "consumer", "instance", "client", "topic", "partition"}
	lagGetHumanRenames = map[string]string{
		"ClusterId":	   "Cluster",
		"ConsumerGroupId": "ConsumerGroup",
		"ConsumerId":      "Consumer",
		"InstanceId":      "Instance",
		"ClientId":        "Client",
		"TopicName":       "Topic",
		"PartitionId":     "Partition"}
	lagGetStructuredRenames = map[string]string{
		"ClusterId":	   "cluster",
		"ConsumerGroupId": "consumer_group",
		"Lag":             "lag",
		"LogEndOffset":    "log_end_offset",
		"CurrentOffset":   "current_offset",
		"ConsumerId":      "consumer",
		"InstanceId":      "instance",
		"ClientId":        "client"}
)

type groupCommand struct {
	*pcmd.AuthenticatedStateFlagCommand
	prerunner			pcmd.PreRunner
	completableChildren []*cobra.Command
}

type consumerData struct {
	ConsumerGroupId string `json:"consumer_group" yaml:"consumer_group"`
	ConsumerId	    string `json:"consumer" yaml:"consumer"`
	InstanceId      string `json:"instance" yaml:"instance"`
	ClientId        string `json:"client" yaml:"client"`
}

type groupData struct {
	ClusterId         string		 `json:"cluster" yaml:"cluster"`
	ConsumerGroupId   string		 `json:"consumer_group" yaml:"consumer_group"`
	Coordinator       string		 `json:"coordinator" yaml:"coordinator"`
	IsSimple		  bool			 `json:"simple" yaml:"simple"`
	PartitionAssignor string		 `json:"partition_assignor" yaml:"partition_assignor"`
	State			  string		 `json:"state" yaml:"state"`
	Consumers         []consumerData `json:"consumers" yaml:"consumers"`
}

type groupDescribeStruct struct {
	ClusterId         string
	ConsumerGroupId   string
	Coordinator       string
	IsSimple		  bool
	PartitionAssignor string
	State			  string
}

type lagCommand struct {
	*pcmd.AuthenticatedStateFlagCommand
	prerunner			pcmd.PreRunner
	completableChildren	[]*cobra.Command
}

type lagSummaryStruct struct {
	ClusterId 		  string
	ConsumerGroupId   string
	TotalLag          int32
	MaxLag            int32
	MaxLagConsumerId  string
	MaxLagInstanceId  string
	MaxLagClientId    string
	MaxLagTopicName   string
	MaxLagPartitionId int32
}

type lagDataStruct struct {
	ClusterId       string
	ConsumerGroupId string
	Lag             int32
	LogEndOffset    int32
	CurrentOffset   int32
	ConsumerId      string
	InstanceId      string
	ClientId        string
	TopicName       string
	PartitionId     int32
}

//func NewGroupCommand(prerunner pcmd.PreRunner) *cobra.Command {
//	cliCmd := pcmd.NewAuthenticatedStateFlagCommand(
//		&cobra.Command{
//			Use:	"consumer-group",
//			Short:	"Manage Kafka consumer-groups.",
//		}, prerunner, GroupSubcommandFlags)
//	groupCommand := &groupCommand{
//		AuthenticatedStateFlagCommand:	cliCmd,
//		prerunner:						prerunner,
//	}
//	groupCommand.init()
//	return groupCommand.Command
//}

func NewGroupCommand(prerunner pcmd.PreRunner) *groupCommand {
	command := &cobra.Command{
		Use:   "consumer-group",
		Short: "Manage Kafka consumer-groups",
	}
	groupCmd := &groupCommand{
		AuthenticatedStateFlagCommand: pcmd.NewAuthenticatedStateFlagCommand(command, prerunner, GroupSubcommandFlags),
		prerunner:                     prerunner,
	}
	groupCmd.init()
	return groupCmd
}

func (g *groupCommand) init() {
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List Kafka consumer groups.",
		Args:  cobra.NoArgs,
		RunE:  pcmd.NewCLIRunE(g.list),
		//RunE: g.list,
	}
	listCmd.Flags().StringP(output.FlagName, output.ShortHandFlag, output.DefaultValue, output.Usage)
	listCmd.Flags().SortFlags = false
	g.AddCommand(listCmd)

	describeCmd := &cobra.Command{
		Use:   "describe <consumer-group>",
		Short: "Describe a Kafka consumer group.",
		Args:  cobra.ExactArgs(1),
		RunE:  pcmd.NewCLIRunE(g.describe),
	}
	describeCmd.Flags().StringP(output.FlagName, output.ShortHandFlag, output.DefaultValue, output.Usage)
	describeCmd.Flags().SortFlags = false
	g.AddCommand(describeCmd)

	lagCmd := NewLagCommand(g.prerunner)
	g.AddCommand(lagCmd.Command)

	g.completableChildren = append(lagCmd.completableChildren, listCmd, describeCmd)
	//g.completableChildren = lagCmd.completableChildren
}

func (g *groupCommand) list(cmd *cobra.Command, args []string) error {
	kafkaREST, err := g.GetKafkaREST()
	if err != nil {
		return err
	}
	if kafkaREST == nil {
		return errors.New(errors.RestProxyNotAvailable)
	}
	// Kafka REST is available
	kafkaClusterConfig, err := g.AuthenticatedCLICommand.Context.GetKafkaClusterForCommand(cmd)
	if err != nil {
		return err
	}
	lkc := kafkaClusterConfig.ID
	groupCmdResp, _, err :=
		kafkaREST.Client.ConsumerGroupApi.ClustersClusterIdConsumerGroupsGet(
			kafkaREST.Context,
			lkc)
	if err != nil {
		return err
	}
	outputWriter, err := output.NewListOutputWriter(cmd, groupListFields, groupListHumanLabels, groupListStructuredLabels)
	if err != nil {
		return err
	}
	for _, groupData := range groupCmdResp.Data {
		outputWriter.AddElement(&groupData)
	}
	return outputWriter.Out()
}

func (g *groupCommand) describe(cmd *cobra.Command, args []string) error {
	consumerGroupId := args[0]

	kafkaREST, err := g.GetKafkaREST()
	if err != nil {
		return err
	}
	if kafkaREST == nil {
		return errors.New(errors.RestProxyNotAvailable)
	}
	// Kafka REST is available
	kafkaClusterConfig, err := g.AuthenticatedCLICommand.Context.GetKafkaClusterForCommand(cmd)
	if err != nil {
		return err
	}
	lkc := kafkaClusterConfig.ID
	groupCmdResp, _, err :=
		kafkaREST.Client.ConsumerGroupApi.ClustersClusterIdConsumerGroupsConsumerGroupIdGet(
			kafkaREST.Context,
			lkc,
			consumerGroupId)
	if err != nil {
		return err
	}
	groupCmdConsumersResp, _, err :=
		kafkaREST.Client.ConsumerGroupApi.ClustersClusterIdConsumerGroupsConsumerGroupIdConsumersGet(
			kafkaREST.Context,
			lkc,
			consumerGroupId)
	if err != nil {
		return err
	}
	groupData := &groupData{}
	groupData.ClusterId = groupCmdResp.ClusterId
	groupData.ConsumerGroupId = groupCmdResp.ConsumerGroupId
	groupData.Coordinator = getStringBroker(groupCmdResp.Coordinator)
	groupData.IsSimple = groupCmdResp.IsSimple
	groupData.PartitionAssignor = groupCmdResp.PartitionAssignor
	groupData.State = getStringState(groupCmdResp.State)
	groupData.Consumers = make([]consumerData, len(groupCmdConsumersResp.Data))
	for i, consumerResp := range groupCmdConsumersResp.Data {
		instanceId := ""
		if consumerResp.InstanceId != nil {
			instanceId = *consumerResp.InstanceId
		}
		consumerData := consumerData{
			ConsumerGroupId: consumerGroupId,
			ConsumerId: consumerResp.ConsumerId,
			InstanceId: instanceId,
			ClientId: consumerResp.ClientId,
		}
		groupData.Consumers[i] = consumerData
	}
	outputOption, err := cmd.Flags().GetString(output.FlagName)
	if err != nil {
		return err
	}
	if outputOption == output.Human.String() {
		return printGroupHumanDescribe(cmd, groupData)
	}
	return output.StructuredOutput(outputOption, groupData)
}

func getStringBroker(relationship kafkarestv3.Relationship) string {
	splitString := strings.SplitAfter(relationship.Related, "brokers/")
	// if relationship was an empty string or did not contain "brokers/"
	if len(splitString) < 2 {
		return ""
	}
	// returning brokerId
	return splitString[1]
}

func getStringState(state kafkarestv3.ConsumerGroupState) string {
	return fmt.Sprintf("%+v", state)
}

func printGroupHumanDescribe(cmd *cobra.Command, groupData *groupData) error {
	// printing non-consumer information in table format first
	err := tables.RenderTableOut(convertGroupToDescribeStruct(groupData), groupDescribeFields, groupDescribeHumanRenames, os.Stdout)
	if err != nil {
		return err
	}
	utils.Printf(cmd, "\nConsumers\n\n")
	// printing consumer information in list table format
	consumerTableEntries := make([][]string, len(groupData.Consumers))
	for i, consumer := range groupData.Consumers {
		consumerTableEntries[i] = printer.ToRow(&consumer, groupDescribeConsumersFields)
	}
	printer.RenderCollectionTable(consumerTableEntries, groupDescribeConsumerTableLabels)
	return nil
}

func convertGroupToDescribeStruct(groupData *groupData) *groupDescribeStruct {
	return &groupDescribeStruct{
		ClusterId:         groupData.ClusterId,
		ConsumerGroupId:   groupData.ConsumerGroupId,
		Coordinator:       groupData.Coordinator,
		IsSimple:          groupData.IsSimple,
		PartitionAssignor: groupData.PartitionAssignor,
		State:             groupData.State,
	}
}

func NewLagCommand(prerunner pcmd.PreRunner) *lagCommand {
	cliCmd := pcmd.NewAuthenticatedStateFlagCommand(
		&cobra.Command{
			Use:   "lag",
			Short: "View consumer lag.",
		}, prerunner, LagSubcommandFlags)
	lagCmd := &lagCommand{
		AuthenticatedStateFlagCommand: cliCmd,
		prerunner:                     prerunner,
	}
	lagCmd.init()
	return lagCmd
}

func (lagCmd *lagCommand) init() {
    summarizeLagCmd := &cobra.Command{
    	Use:	"summarize <id>",
    	Short:	"Summarize consumer lag for a Kafka consumer-group.",
    	Args:	cobra.ExactArgs(1),
    	RunE:	pcmd.NewCLIRunE(lagCmd.summarizeLag),
    	Example: examples.BuildExampleString(
    		examples.Example{
    			Text: "Summarize the lag for consumer-group ``consumer-group-1``.",
    			// ahu: should the examples include the other required flag(s)? --cluster
    			Code: "ccloud kafka consumer-group lag summarize consumer-group-1",
    		},
    	),
    }
    summarizeLagCmd.Flags().StringP(output.FlagName, output.ShortHandFlag, output.DefaultValue, output.Usage)
    summarizeLagCmd.Flags().SortFlags = false
    lagCmd.AddCommand(summarizeLagCmd)

    listLagCmd := &cobra.Command{
    	Use:	"list <id>",
      	Short:	"List consumer lags for a Kafka consumer-group.",
     	Args:	cobra.ExactArgs(1),
      	RunE:	pcmd.NewCLIRunE(lagCmd.listLag),
      	Example: examples.BuildExampleString(
      		examples.Example{
      			Text: "List all consumer lags for consumers in consumer-group ``consumer-group-1``.",
      			Code: "ccloud kafka consumer-group lag list consumer-group-1",
      		},
      	),
    }
	listLagCmd.Flags().StringP(output.FlagName, output.ShortHandFlag, output.DefaultValue, output.Usage)
	listLagCmd.Flags().SortFlags = false
    lagCmd.AddCommand(listLagCmd)

   	getLagCmd := &cobra.Command{
    	Use:	"get <id>",
      	Short:	"Get consumer lag for a partition consumed by a Kafka consumer-group.",
     	Args:	cobra.ExactArgs(1),
      	RunE:	pcmd.NewCLIRunE(lagCmd.getLag),
      	Example: examples.BuildExampleString(
      		examples.Example{
      			Text: "Get the consumer lag for topic ``my_topic`` partition ``0`` consumed by consumer-group ``consumer-group-1``.",
      			Code: "ccloud kafka consumer-group lag get consumer-group-1 --topic my_topic --partition 0",
      		},
      	),
   	}
   	// ahu: handle defaults
	getLagCmd.Flags().StringP(output.FlagName, output.ShortHandFlag, output.DefaultValue, output.Usage)
   	getLagCmd.Flags().String("topic", "", "Topic name.")
   	getLagCmd.Flags().Int32("partition", -1, "Partition ID.")
   	check(getLagCmd.MarkFlagRequired("topic"))
   	check(getLagCmd.MarkFlagRequired("partition"))
   	getLagCmd.Flags().SortFlags = false
   	lagCmd.AddCommand(getLagCmd)

   	lagCmd.completableChildren = []*cobra.Command{summarizeLagCmd, listLagCmd, getLagCmd}
	//lagCmd.completableChildren = []*cobra.Command{summarizeLagCmd}
}

func (lagCmd *lagCommand) summarizeLag(cmd *cobra.Command, args []string) error {
	consumerGroupId := args[0]

	kafkaREST, err := lagCmd.GetKafkaREST()
	if err != nil {
		return err
	}
	if kafkaREST == nil {
		return errors.New(errors.RestProxyNotAvailable)
	}
	// Kafka REST is available
	kafkaClusterConfig, err := lagCmd.AuthenticatedCLICommand.Context.GetKafkaClusterForCommand(cmd)
	if err != nil {
		return err
	}
	lkc := kafkaClusterConfig.ID
	fmt.Print("got the lkc ")
	fmt.Println(lkc)
	lagSummaryResp, _, err :=
		kafkaREST.Client.ConsumerGroupApi.ClustersClusterIdConsumerGroupsConsumerGroupIdLagSummaryGet(
			kafkaREST.Context,
			lkc,
			consumerGroupId)

	if err != nil {
		return err
	}

	return output.DescribeObject(
		cmd,
		convertLagSummaryToStruct(lagSummaryResp),
		lagSummaryFields,
		lagSummaryHumanRenames,
		lagSummaryStructuredRenames)

	//lagSummaryResp, httpResp, err :=
	//	kafkaREST.Client.ConsumerGroupApi.ClustersClusterIdConsumerGroupsConsumerGroupIdLagSummaryGet(
	//		kafkaREST.Context,
	//		lkc,
	//		consumerGroupId)
	//
	//if httpResp != nil {
	//	fmt.Print("httpResp received ")
	//	if err != nil {
	//		fmt.Print("error getting lag response ")
	//		restErr, parseErr := parseOpenAPIError(err)
	//		if parseErr == nil && restErr.Code == KafkaRestUnknownConsumerGroupErrorCode {
	//			return fmt.Errorf(errors.UnknownGroupMsg, consumerGroupId)
	//		}
	//		// ahu: check if this will be descriptive enough to cover parse errors (if we can remove the preceding check)
	//		return kafkaRestError(kafkaREST.Client.GetConfig().BasePath, err, httpResp)
	//	}
	//	if httpResp.StatusCode != http.StatusOK {
	//		fmt.Print("got a status code that wasn't OK ")
	//		return errors.NewErrorWithSuggestions(
	//			fmt.Sprintf(errors.KafkaRestUnexpectedStatusMsg, httpResp.Request.URL, httpResp.StatusCode),
	//			errors.InternalServerErrorSuggestions)
	//	}
	//	// Kafka REST returns StatusOK
	//	fmt.Print("we got status OK ")
	//	return output.DescribeObject(
	//		cmd,
	//		convertLagSummaryToStruct(lagSummaryResp),
	//		lagSummaryFields,
	//		lagSummaryHumanRenames,
	//		lagSummaryStructuredRenames)
	//}
	//fmt.Print("no httpResp received ")
	//return err

}

func convertLagSummaryToStruct(lagSummaryData kafkarestv3.ConsumerGroupLagSummaryData) *lagSummaryStruct {
	maxLagInstanceId := ""
	if lagSummaryData.MaxLagInstanceId != nil {
		maxLagInstanceId = *lagSummaryData.MaxLagInstanceId
	}
	return &lagSummaryStruct{
		ClusterId:		   lagSummaryData.ClusterId,
		ConsumerGroupId:   lagSummaryData.ConsumerGroupId,
		TotalLag:          lagSummaryData.TotalLag,
		MaxLag:            lagSummaryData.MaxLag,
		MaxLagConsumerId:  lagSummaryData.MaxLagConsumerId,
		MaxLagInstanceId:  maxLagInstanceId,
		MaxLagClientId:    lagSummaryData.MaxLagClientId,
		MaxLagTopicName:   lagSummaryData.MaxLagTopicName,
		MaxLagPartitionId: lagSummaryData.MaxLagPartitionId,
	}
}

func (lagCmd *lagCommand) listLag(cmd *cobra.Command, args []string) error {
	consumerGroupId := args[0]

	kafkaREST, err := lagCmd.GetKafkaREST()
	if err != nil {
		return err
	}
	if kafkaREST == nil {
		return errors.New(errors.RestProxyNotAvailable)
	}
	// Kafka REST is available
	kafkaClusterConfig, err := lagCmd.AuthenticatedCLICommand.Context.GetKafkaClusterForCommand(cmd)
	if err != nil {
		return err
	}
	lkc := kafkaClusterConfig.ID
	lagSummaryResp, _, err :=
		kafkaREST.Client.ConsumerGroupApi.ClustersClusterIdConsumerGroupsConsumerGroupIdLagsGet(
			kafkaREST.Context,
			lkc,
			consumerGroupId)
	if err != nil {
		return err
	}
	outputWriter, err := output.NewListOutputWriter(cmd, lagFields, lagListHumanLabels, lagListStructuredLabels)
	if err != nil {
		return err
	}
	for _, lagData := range lagSummaryResp.Data {
		outputWriter.AddElement(convertLagToStruct(lagData))
	}
	return outputWriter.Out()
}

func (lagCmd *lagCommand) getLag(cmd *cobra.Command, args []string) error {
	consumerGroupId := args[0]
	topicName, err := cmd.Flags().GetString("topic")
	if err != nil {
		return err
	}
	partitionId, err := cmd.Flags().GetInt32("partition")
	if err != nil {
		return err
	}
	kafkaREST, err := lagCmd.GetKafkaREST()
	if err != nil {
		return err
	}
	if kafkaREST == nil {
		return errors.New(errors.RestProxyNotAvailable)
	}
	// Kafka REST is available
	kafkaClusterConfig, err := lagCmd.AuthenticatedCLICommand.Context.GetKafkaClusterForCommand(cmd)
	if err != nil {
		return err
	}
	lkc := kafkaClusterConfig.ID
	lagGetResp, _, err :=
		kafkaREST.Client.PartitionApi.ClustersClusterIdConsumerGroupsConsumerGroupIdLagsTopicNamePartitionsPartitionIdGet(
			kafkaREST.Context,
			lkc,
			consumerGroupId,
			topicName,
			partitionId)
	if err != nil {
		return err
	}
	return output.DescribeObject(
		cmd,
		convertLagToStruct(lagGetResp),
		lagFields,
		lagGetHumanRenames,
		lagGetStructuredRenames)
}

func convertLagToStruct(lagData kafkarestv3.ConsumerLagData) *lagDataStruct {
	instanceId := ""
	if lagData.InstanceId != nil {
		instanceId = *lagData.InstanceId
	}

	return &lagDataStruct{
		ClusterId:       lagData.ClusterId,
		ConsumerGroupId: lagData.ConsumerGroupId,
		Lag:             lagData.Lag,
		LogEndOffset:    lagData.LogEndOffset,
		CurrentOffset:   lagData.CurrentOffset,
		ConsumerId:      lagData.ConsumerId,
		InstanceId:      instanceId,
		ClientId:        lagData.ClientId,
		TopicName:       lagData.TopicName,
		PartitionId:     lagData.PartitionId,
	}
}

func (g *groupCommand) Cmd() *cobra.Command {
	return g.Command
}

func (g *groupCommand) ServerComplete() []prompt.Suggest {
	var suggestions []prompt.Suggest
	return suggestions
}

func (g *groupCommand) ServerCompletableChildren() []*cobra.Command {
	return g.completableChildren
}