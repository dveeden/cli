package ccloudv2

import (
	"context"

	cloudkafkarestv3 "github.com/confluentinc/ccloud-sdk-go-v2/kafkarest/v3"
)

type CloudKafkaRESTProvider func() (*CloudKafkaREST, error)

type CloudKafkaREST struct {
	Client  *cloudkafkarestv3.APIClient
	Context context.Context
}
