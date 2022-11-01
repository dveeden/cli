package kafka

import (
	"fmt"
	"net/http"

	"github.com/confluentinc/kafka-rest-sdk-go/kafkarestv3"
	"github.com/spf13/cobra"

	pcmd "github.com/confluentinc/cli/internal/pkg/cmd"
	"github.com/confluentinc/cli/internal/pkg/examples"
	"github.com/confluentinc/cli/internal/pkg/kafkarest"
	"github.com/confluentinc/cli/internal/pkg/output"
	"github.com/confluentinc/cli/internal/pkg/utils"
)

type replicaOut struct {
	ClusterId          string `human:"Cluster ID" serialized:"cluster_id"`
	TopicName          string `human:"Topic Name" serialized:"topic_name"`
	BrokerId           int32  `human:"Broker ID" serialized:"broker_id"`
	PartitionId        int32  `human:"Partition ID" serialized:"partition_id"`
	IsLeader           bool   `human:"Leader" serialized:"is_leader"`
	IsObserver         bool   `human:"Observer" serialized:"is_observer"`
	IsIsrEligible      bool   `human:"ISR Eligible" serialized:"is_isr_eligible"`
	IsInIsr            bool   `human:"In ISR" serialized:"is_in_isr"`
	IsCaughtUp         bool   `human:"Caught Up" serialized:"is_caught_up"`
	LogStartOffset     int64  `human:"Log Start Offset" serialized:"log_start_offset"`
	LogEndOffset       int64  `human:"Log End Offset" serialized:"log_end_offset"`
	LastCaughtUpTimeMs string `human:"Last Caught Up Time Ms" serialized:"last_caught_up_time_ms"`
	LastFetchTimeMs    string `human:"Last Fetch Time Ms" serialized:"last_fetch_time_ms"`
	LinkName           string `human:"Link Name" serialized:"link_name"`
}

func (c *replicaCommand) newListCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List Kafka replica statuses.",
		Long:  "List partition-replicas statuses filtered by topic and partition via Confluent Kafka REST.",
		Args:  cobra.NoArgs,
		RunE:  c.list,
		Example: examples.BuildExampleString(
			examples.Example{
				Text: `List the replica statuses for partition 1 of topic "my_topic".`,
				Code: "confluent kafka replica list --topic my_topic --partition 1",
			},
			examples.Example{
				Text: `List the replica statuses for topic "my_topic".`,
				Code: "confluent kafka replica list --topic my_topic",
			},
		),
	}

	cmd.Flags().String("topic", "", "Topic name.")
	cmd.Flags().Int32("partition", -1, "Partition ID.")
	cmd.Flags().AddFlagSet(pcmd.OnPremKafkaRestSet())
	pcmd.AddOutputFlag(cmd)

	_ = cmd.MarkFlagRequired("topic")

	return cmd
}

func (c *replicaCommand) list(cmd *cobra.Command, _ []string) error {
	topic, partitionId, err := readFlagValues(cmd)
	if err != nil {
		return err
	}

	restClient, restContext, err := initKafkaRest(c.AuthenticatedCLICommand, cmd)
	if err != nil {
		return err
	}

	clusterId, err := getClusterIdForRestRequests(restClient, restContext)
	if err != nil {
		return err
	}

	var replicaStatusDataList kafkarestv3.ReplicaStatusDataList
	var resp *http.Response
	if partitionId != -1 {
		replicaStatusDataList, resp, err = restClient.ReplicaStatusApi.ClustersClusterIdTopicsTopicNamePartitionsPartitionIdReplicaStatusGet(restContext, clusterId, topic, partitionId)
	} else {
		replicaStatusDataList, resp, err = restClient.ReplicaStatusApi.ClustersClusterIdTopicsTopicNamePartitionsReplicaStatusGet(restContext, clusterId, topic)
	}
	if err != nil {
		return kafkarest.NewError(restClient.GetConfig().BasePath, err, resp)
	}

	list := output.NewList(cmd)
	for _, data := range replicaStatusDataList.Data {
		out := &replicaOut{
			ClusterId:          data.ClusterId,
			TopicName:          data.TopicName,
			BrokerId:           data.BrokerId,
			PartitionId:        data.PartitionId,
			IsLeader:           data.IsLeader,
			IsObserver:         data.IsObserver,
			IsIsrEligible:      data.IsIsrEligible,
			IsInIsr:            data.IsInIsr,
			IsCaughtUp:         data.IsCaughtUp,
			LogStartOffset:     data.LogStartOffset,
			LogEndOffset:       data.LogEndOffset,
			LastCaughtUpTimeMs: fmt.Sprint(data.LastCaughtUpTimeMs),
			LastFetchTimeMs:    fmt.Sprint(data.LastFetchTimeMs),
			LinkName:           data.LinkName,
		}
		if output.GetFormat(cmd) == output.Human {
			out.LastCaughtUpTimeMs = utils.FormatUnixTime(data.LastCaughtUpTimeMs)
			out.LastFetchTimeMs = utils.FormatUnixTime(data.LastFetchTimeMs)
		}
		list.Add(out)
	}
	return list.Print()
}

func readFlagValues(cmd *cobra.Command) (string, int32, error) {
	topic, err := cmd.Flags().GetString("topic")
	if err != nil {
		return "", -1, err
	}
	partition, err := cmd.Flags().GetInt32("partition")
	if err != nil {
		return "", -1, err
	}
	return topic, partition, nil
}
