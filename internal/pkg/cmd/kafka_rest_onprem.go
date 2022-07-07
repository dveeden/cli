package cmd

import (
	"context"

	"github.com/confluentinc/kafka-rest-sdk-go/kafkarestv3"
)

type OnPremKafkaREST struct {
	Client  *kafkarestv3.APIClient
	Context context.Context
}

func NewKafkaREST(client *kafkarestv3.APIClient, context context.Context) *OnPremKafkaREST {
	return &OnPremKafkaREST{
		Client:  client,
		Context: context,
	}
}
