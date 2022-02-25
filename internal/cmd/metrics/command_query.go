package metrics

import (
	"context"
	"github.com/confluentinc/ccloud-sdk-go-v1"
	pcmd "github.com/confluentinc/cli/internal/pkg/cmd"
	"github.com/confluentinc/cli/internal/pkg/errors"
	"github.com/confluentinc/cli/internal/pkg/output"
	"github.com/spf13/cobra"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
	"strings"
	"time"
)

type abstractMetricsApiQueryReply interface{}

func (c *command) newQueryCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "query",
		Short: "Query Confluent Cloud metrics.",
		Args:  cobra.NoArgs,
		RunE:  pcmd.NewCLIRunE(c.query),
	}

	cmd.Flags().StringP(output.FlagName, "o", "human", `Specify the output format as "human", "json", "yaml", or "chart-html".`)

	cmd.Flags().String("metric", "", `The metric to query.`)
	_ = cmd.MarkFlagRequired("metric")

	cmd.Flags().String("interval", "PT30M/now-2m|m", "Time range in ISO-8601 interval syntax")
	cmd.Flags().String("granularity", "PT1M", "Query granularity")

	cmd.Flags().StringArray("group-by", nil, "Label(s) to group by")
	cmd.Flags().Int32("group-limit", 5, "Group limit")

	cmd.Flags().String("kafka", "", "A Kafka cluster to query metrics for")
	cmd.Flags().String("connector", "", "A Connector to query metrics for")
	cmd.Flags().String("schema-registry", "", "A Schema Registry to query metrics for")
	cmd.Flags().String("ksql", "", "A ksqlDB application to query metrics for")

	return cmd
}

func (c *command) query(cmd *cobra.Command, _ []string) error {
	ctx := context.Background()

	query, err := buildQuery(cmd)
	if err != nil {
		return err
	}

	var response abstractMetricsApiQueryReply
	if query.Format == "FLAT" {
		response, err = c.Client.MetricsApi.QueryV2(ctx, "cloud", query, "")
	} else {
		response, err = c.Client.MetricsApi.QueryV2Grouped(ctx, "cloud", query, "")
	}

	if err != nil {
		return err
	}

	return outputResponse(cmd, query, response)
}

func buildQuery(cmd *cobra.Command) (*ccloud.MetricsApiRequest, error) {
	metric, _ := cmd.Flags().GetString("metric")
	interval, _ := cmd.Flags().GetString("interval")
	groupBy, _ := cmd.Flags().GetStringArray("group-by")
	groupLimit, _ := cmd.Flags().GetInt32("group-limit")
	granularity, _ := cmd.Flags().GetString("granularity")

	filter, err := getResourceFilter(cmd)
	if err != nil {
		return nil, err
	}

	out, err := cmd.Flags().GetString(output.FlagName)
	if err != nil {
		return nil, err
	}

	format := "FLAT"
	if len(groupBy) > 0 && out == "chart-html" {
		format = "GROUPED"
	}

	request := &ccloud.MetricsApiRequest{
		Aggregations: []ccloud.ApiAggregation{
			{
				Metric: metric,
			},
		},
		Filter:      *filter,
		GroupBy:     groupBy,
		Limit:       groupLimit,
		Granularity: strings.ToUpper(granularity),
		Intervals:   []string{interval},
		Format:      format,
	}

	return request, nil
}

func getResourceFilter(cmd *cobra.Command) (*ccloud.ApiFilter, error) {
	kafka, _ := cmd.Flags().GetString("kafka")
	connector, _ := cmd.Flags().GetString("connector")
	ksql, _ := cmd.Flags().GetString("ksql")
	schemaRegistry, _ := cmd.Flags().GetString("schema-registry")

	if kafka != "" {
		return &ccloud.ApiFilter{
			Field: "resource.kafka.id",
			Op:    "EQ",
			Value: kafka,
		}, nil
	} else if connector != "" {
		return &ccloud.ApiFilter{
			Field: "resource.connector.id",
			Op:    "EQ",
			Value: connector,
		}, nil
	} else if ksql != "" {
		return &ccloud.ApiFilter{
			Field: "resource.ksql.id",
			Op:    "EQ",
			Value: ksql,
		}, nil
	} else if schemaRegistry != "" {
		return &ccloud.ApiFilter{
			Field: "resource.schema_registry.id",
			Op:    "EQ",
			Value: schemaRegistry,
		}, nil
	} else {
		return nil, errors.New("One of --kafka, --connector, --ksql, or --schema-registry is required")
	}

}

func outputResponse(cmd *cobra.Command, query *ccloud.MetricsApiRequest, response abstractMetricsApiQueryReply) error {
	format, err := cmd.Flags().GetString(output.FlagName)
	if err != nil {
		return err
	}

	switch format {
	case "chart-html":
		return chartResponseAsHtml(cmd, query, response)
	default:
		return outputStructured(cmd, query, response.(*ccloud.MetricsApiQueryReply))
	}
}

func outputStructured(cmd *cobra.Command, query *ccloud.MetricsApiRequest, response *ccloud.MetricsApiQueryReply) error {
	printer := message.NewPrinter(language.English)

	fields := []string{"timestamp"}
	humanLabels := []string{"Timestamp"}
	for _, label := range query.GroupBy {
		fields = append(fields, label)
		humanLabels = append(humanLabels, label)
	}
	fields = append(fields, "value")
	humanLabels = append(humanLabels, "Value")

	outputWriter, _ := output.NewListOutputWriter(cmd, fields, humanLabels, fields)
	for _, point := range response.Result {
		data := map[string]string{}
		data["timestamp"] = point.Timestamp.Format(time.RFC3339)
		data["value"] = printer.Sprintf("%.1f", point.Value)
		for k, v := range point.Labels {
			data[k] = v.(string)
		}
		outputWriter.AddMapElement(data)
	}
	return outputWriter.Out()
}
