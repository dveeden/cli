package metrics

import (
	"context"
	"encoding/json"
	"github.com/confluentinc/ccloud-sdk-go-v1"
	pcmd "github.com/confluentinc/cli/internal/pkg/cmd"
	"github.com/confluentinc/cli/internal/pkg/utils"
	"github.com/spf13/cobra"
)

func (c *command) newQueryCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "query",
		Short: "Query Confluent Cloud metrics.",
		Args:  cobra.NoArgs,
		RunE:  pcmd.NewCLIRunE(c.query),
	}

	pcmd.AddOutputFlag(cmd)

	return cmd
}

func (c *command) query(cmd *cobra.Command, _ []string) error {
	ctx := context.Background()
	query := buildQuery()
	metricsResponse, _ := c.Client.MetricsApi.QueryV2(ctx, "cloud", query, "")
	data, _ := json.MarshalIndent(metricsResponse, "", "  ")
	utils.Printf(cmd, "%s\n", data)
	return nil
}

func buildQuery() *ccloud.MetricsApiRequest {
	return &ccloud.MetricsApiRequest{
		Aggregations: []ccloud.ApiAggregation{
			{
				Metric: "io.confluent.kafka.server/active_connection_count",
			},
		},
		Filter: ccloud.ApiFilter{
			Field: "resource.kafka.id",
			Op:    "EQ",
			Value: "lkc-l9k5v",
		},
		Granularity: "PT1M",
		Intervals:   []string{"PT5M/now-2m|m"},
	}
}
