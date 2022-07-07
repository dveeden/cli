package cmd

import (
	"context"
	"net/http"

	cloudkafkarestv3 "github.com/confluentinc/ccloud-sdk-go-v2/kafkarest/v3"
	plog "github.com/confluentinc/cli/internal/pkg/log"
	testserver "github.com/confluentinc/cli/test/test-server"
	"github.com/hashicorp/go-retryablehttp"
)

type CloudKafkaREST struct {
	Client  *cloudkafkarestv3.APIClient
	Context context.Context
}

func createKafkaRESTClient(kafkaRestURL, userAgent string, isTest bool) (*cloudkafkarestv3.APIClient, error) {
	cfg := cloudkafkarestv3.NewConfiguration()
	cfg.Debug = plog.CliLogger.Level >= plog.DEBUG
	cfg.HTTPClient = newRetryableHttpClient()
	cfg.Servers = cloudkafkarestv3.ServerConfigurations{{URL: getServerUrl(kafkaRestURL, isTest), Description: "Confluent Cloud Kafka Rest"}}
	cfg.UserAgent = userAgent
	return cloudkafkarestv3.NewAPIClient(cfg), nil
}

func newRetryableHttpClient() *http.Client {
	client := retryablehttp.NewClient()
	client.Logger = new(plog.LeveledLogger)
	return client.StandardClient()
}

func getServerUrl(baseURL string, isTest bool) string {
	if isTest {
		return testserver.TestV2CloudURL.String()
	}
	return baseURL + "/kafka/v3"
}
