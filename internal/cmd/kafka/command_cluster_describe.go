package kafka

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/spf13/cobra"

	cmkv2 "github.com/confluentinc/ccloud-sdk-go-v2/cmk/v2"

	"github.com/confluentinc/cli/internal/pkg/ccloudv2"
	pcmd "github.com/confluentinc/cli/internal/pkg/cmd"
	v1 "github.com/confluentinc/cli/internal/pkg/config/v1"
	dynamicconfig "github.com/confluentinc/cli/internal/pkg/dynamic-config"
	"github.com/confluentinc/cli/internal/pkg/errors"
	"github.com/confluentinc/cli/internal/pkg/kafkarest"
	"github.com/confluentinc/cli/internal/pkg/output"
	"github.com/confluentinc/cli/internal/pkg/resource"
)

var basicDescribeFields = []string{"IsCurrent", "Id", "Name", "Type", "NetworkIngress", "NetworkEgress", "Storage", "ServiceProvider", "Availability", "Region", "Status", "Endpoint", "ApiEndpoint", "RestEndpoint"}

type describeStruct struct {
	IsCurrent          bool   `human:"Current" serialized:"is_current"`
	Id                 string `human:"ID" serialized:"id"`
	Name               string `human:"Name" serialized:"name"`
	Type               string `human:"Type" serialized:"type"`
	ClusterSize        int32  `human:"Cluster Size" serialized:"cluster_size"`
	PendingClusterSize int32  `human:"Pending Cluster Size" serialized:"pending_cluster_size"`
	NetworkIngress     int32  `human:"Ingress" serialized:"ingress"`
	NetworkEgress      int32  `human:"Egress" serialized:"egress"`
	Storage            string `human:"Storage" serialized:"storage"`
	ServiceProvider    string `human:"Provider" serialized:"provider"`
	Region             string `human:"Region" serialized:"region"`
	Availability       string `human:"Availability" serialized:"availability"`
	Status             string `human:"Status" serialized:"status"`
	Endpoint           string `human:"Endpoint" serialized:"endpoint"`
	EncryptionKeyId    string `human:"Encryption Key ID" serialized:"encryption_key_id"`
	RestEndpoint       string `human:"REST Endpoint" serialized:"rest_endpoint"`
	TopicCount         int    `human:"Topic Count" serialized:"topic_count"`
}

func (c *clusterCommand) newDescribeCommand(cfg *v1.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:               "describe [id]",
		Short:             "Describe a Kafka cluster.",
		Long:              "Describe the Kafka cluster specified with the ID argument, or describe the active cluster for the current context.",
		Args:              cobra.MaximumNArgs(1),
		ValidArgsFunction: pcmd.NewValidArgsFunction(c.validArgs),
		RunE:              c.describe,
		Annotations:       map[string]string{pcmd.RunRequirement: pcmd.RequireNonAPIKeyCloudLogin},
	}

	pcmd.AddContextFlag(cmd, c.CLICommand)
	if cfg.IsCloudLogin() {
		pcmd.AddEnvironmentFlag(cmd, c.AuthenticatedCLICommand)
	}
	pcmd.AddOutputFlag(cmd)

	return cmd
}

func (c *clusterCommand) describe(cmd *cobra.Command, args []string) error {
	lkc, err := c.getLkcForDescribe(args)
	if err != nil {
		return err
	}

	ctx := c.AuthenticatedCLICommand.Context.Config.Context()
	c.AuthenticatedCLICommand.Context.Config.SetOverwrittenActiveKafka(ctx.KafkaClusterContext.GetActiveKafkaClusterId())
	ctx.KafkaClusterContext.SetActiveKafkaCluster(lkc)

	cluster, httpResp, err := c.V2Client.DescribeKafkaCluster(lkc, c.EnvironmentId())
	if err != nil {
		return errors.CatchKafkaNotFoundError(err, lkc, httpResp)
	}

	return c.outputKafkaClusterDescription(cmd, &cluster, true)
}

func (c *clusterCommand) getLkcForDescribe(args []string) (string, error) {
	if len(args) > 0 {
		if resource.LookupType(args[0]) != resource.KafkaCluster {
			return "", errors.Errorf(errors.KafkaClusterMissingPrefixErrorMsg, args[0])
		}
		return args[0], nil
	}

	lkc := c.Config.Context().KafkaClusterContext.GetActiveKafkaClusterId()
	if lkc == "" {
		return "", errors.NewErrorWithSuggestions(errors.NoKafkaSelectedErrorMsg, errors.NoKafkaForDescribeSuggestions)
	}

	return lkc, nil
}

func (c *clusterCommand) outputKafkaClusterDescription(cmd *cobra.Command, cluster *cmkv2.CmkV2Cluster, getTopicCount bool) error {
	out := convertClusterToDescribeStruct(cluster, c.Context.Context)

	if getTopicCount {
		topicCount, err := c.getTopicCountForKafkaCluster(cluster)
		if err != nil {
			return err
		}
		out.TopicCount = topicCount
	}

	table := output.NewTable(cmd)
	table.Add(out)
	table.Filter(getKafkaClusterDescribeFields(cluster, basicDescribeFields, getTopicCount))
	return table.Print()
}

func convertClusterToDescribeStruct(cluster *cmkv2.CmkV2Cluster, ctx *v1.Context) *describeStruct {
	clusterStorage := getKafkaClusterStorage(cluster)
	ingress, egress := getCmkClusterIngressAndEgress(cluster)

	return &describeStruct{
		IsCurrent:          *cluster.Id == ctx.KafkaClusterContext.GetActiveKafkaClusterId(),
		Id:                 *cluster.Id,
		Name:               *cluster.Spec.DisplayName,
		Type:               getCmkClusterType(cluster),
		ClusterSize:        getCmkClusterSize(cluster),
		PendingClusterSize: getCmkClusterPendingSize(cluster),
		NetworkIngress:     ingress,
		NetworkEgress:      egress,
		Storage:            clusterStorage,
		ServiceProvider:    strings.ToLower(*cluster.Spec.Cloud),
		Region:             *cluster.Spec.Region,
		Availability:       availabilitiesToHuman[*cluster.Spec.Availability],
		Status:             getCmkClusterStatus(cluster),
		Endpoint:           cluster.Spec.GetKafkaBootstrapEndpoint(),
		EncryptionKeyId:    getCmkEncryptionKey(cluster),
		RestEndpoint:       cluster.Spec.GetHttpEndpoint(),
	}
}

func getKafkaClusterStorage(cluster *cmkv2.CmkV2Cluster) string {
	if !isBasic(cluster) {
		return "Infinite"
	} else {
		return "5 TB"
	}
}

func getKafkaClusterDescribeFields(cluster *cmkv2.CmkV2Cluster, basicFields []string, getTopicCount bool) []string {
	describeFields := basicFields
	if isDedicated(cluster) {
		describeFields = append(describeFields, "ClusterSize")
		if isExpanding(cluster) || isShrinking(cluster) {
			describeFields = append(describeFields, "PendingClusterSize")
		}
		if cluster.Spec.Config.CmkV2Dedicated.EncryptionKey != nil && *cluster.Spec.Config.CmkV2Dedicated.EncryptionKey != "" {
			describeFields = append(describeFields, "EncryptionKeyId")
		}
	}

	if getTopicCount {
		describeFields = append(describeFields, "TopicCount")
	}

	return describeFields
}

func (c *clusterCommand) getTopicCountForKafkaCluster(cluster *cmkv2.CmkV2Cluster) (int, error) {
	if getCmkClusterStatus(cluster) == ccloudv2.StatusProvisioning {
		return 0, nil
	}

	lkc := *cluster.Id
	if kafkaREST, _ := c.GetKafkaREST(); kafkaREST != nil {
		topicGetResp, httpResp, err := kafkaREST.CloudClient.ListKafkaTopics(lkc)
		if err != nil && httpResp != nil {
			// Kafka REST is available, but an error occurred
			return 0, kafkarest.NewError(kafkaREST.CloudClient.GetUrl(), err, httpResp)
		}

		if err == nil && httpResp != nil {
			if httpResp.StatusCode != http.StatusOK {
				return 0, errors.NewErrorWithSuggestions(
					fmt.Sprintf(errors.KafkaRestUnexpectedStatusErrorMsg, httpResp.Request.URL, httpResp.StatusCode),
					errors.InternalServerErrorSuggestions)
			}
			// Kafka REST is available and there was no error
			return len(topicGetResp.Data), nil
		}
	}

	// Kafka REST is not available, fall back to KafkaAPI, to be deprecated
	req, err := dynamicconfig.KafkaCluster(c.Context)
	if err != nil {
		return 0, err
	}
	resp, err := c.Client.Kafka.ListTopics(context.Background(), req)
	return len(resp), err
}
