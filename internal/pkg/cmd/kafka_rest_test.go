package cmd

import (
	"testing"

	"github.com/stretchr/testify/suite"

	v2 "github.com/confluentinc/cli/internal/pkg/config/v2"
)

type KafkaRestTestSuite struct {
	suite.Suite
}

func (suite *KafkaRestTestSuite) TestInvalidGetBearerToken() {
	req := suite.Require()
	emptyState := v2.ContextState{}
	_, err := getBearerToken(&emptyState, "invalidhost")
	req.NotNil(err)
}

func TestKafkaRestTestSuite(t *testing.T) {
	suite.Run(t, new(KafkaRestTestSuite))
}
