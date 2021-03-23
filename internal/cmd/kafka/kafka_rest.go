package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/confluentinc/cli/internal/pkg/cmd"
	"net/http"
	neturl "net/url"
	"os"
	"regexp"
	"strings"

	"github.com/spf13/cobra"

	"github.com/confluentinc/cli/internal/pkg/utils"

	"github.com/antihax/optional"
	schedv1 "github.com/confluentinc/cc-structs/kafka/scheduler/v1"
	"github.com/confluentinc/kafka-rest-sdk-go/kafkarestv3"

	"github.com/confluentinc/cli/internal/pkg/errors"
)

const KafkaRestBadRequestErrorCode = 40002
const KafkaRestUnknownTopicOrPartitionErrorCode = 40403
const CONFLUENT_REST_URL = "CONFLUENT_REST_URL"
const SelfSignedCertError = "x509: certificate is not authorized to sign other certificates"
const UnauthorizedCertError = "x509: certificate signed by unknown authority"

type kafkaRestV3Error struct {
	Code    int    `json:"error_code"`
	Message string `json:"message"`
}

func kafkaRestHttpError(httpResp *http.Response) error {
	return errors.NewErrorWithSuggestions(
		fmt.Sprintf(errors.KafkaRestErrorMsg, httpResp.Request.Method, httpResp.Request.URL, httpResp.Status),
		errors.InternalServerErrorSuggestions)
}

func parseOpenAPIError(err error) (*kafkaRestV3Error, error) {
	if openAPIError, ok := err.(kafkarestv3.GenericOpenAPIError); ok {
		var decodedError kafkaRestV3Error
		err = json.Unmarshal(openAPIError.Body(), &decodedError)
		if err != nil {
			return nil, err
		}
		return &decodedError, nil
	}
	return nil, fmt.Errorf("unexpected type")
}

func kafkaRestError(url string, err error, httpResp *http.Response) error {
	switch err.(type) {
	case *neturl.Error:
		if e, ok := err.(*neturl.Error); ok {
			if strings.Contains(e.Error(), SelfSignedCertError) || strings.Contains(e.Error(), UnauthorizedCertError) {
				return errors.NewErrorWithSuggestions(fmt.Sprintf(errors.KafkaRestConnectionMsg, url, e.Err), errors.KafkaRestCertErrorSuggestions)
			}
			return errors.Errorf(errors.KafkaRestConnectionMsg, url, e.Err)
		}
	case kafkarestv3.GenericOpenAPIError:
		openAPIError, parseErr := parseOpenAPIError(err)
		if parseErr == nil {
			if strings.Contains(openAPIError.Message, "invalid_token") {
				return errors.NewErrorWithSuggestions(errors.InvalidMDSToken, errors.InvalidMDSTokenSuggestions)
			}
			return fmt.Errorf("REST request failed: %v (%v)", openAPIError.Message, openAPIError.Code)
		}
		if httpResp != nil && httpResp.StatusCode >= 400 {
			return kafkaRestHttpError(httpResp)
		}
		return errors.NewErrorWithSuggestions(errors.UnknownErrorMsg, errors.InternalServerErrorSuggestions)
	}
	return err
}

// Used for on-prem KafkaRest commands
// Embedded KafkaRest uses /kafka/v3 and standalone uses /v3
// Relying on users to include the /kafka in the url for embedded instances
func setServerURL(cmd *cobra.Command, client *kafkarestv3.APIClient, url string) {
	url = strings.Trim(url, "/")   // localhost:8091/kafka/v3/ --> localhost:8091/kafka/v3
	url = strings.Trim(url, "/v3") // localhost:8091/kafka/v3 --> localhost:8091/kafka
	protocolRgx, _ := regexp.Compile(`(\w+)://`)
	protocolMatch := protocolRgx.MatchString(url)
	if !protocolMatch {
		var protocolMsg string
		if cmd.Flags().Changed("client-cert-path") || cmd.Flags().Changed("ca-cert-path") { // assume https if client-cert is set since this means we want to use mTLS auth
			url = "https://" + url
			protocolMsg = errors.AssumingHttpsProtocol
		} else {
			url = "http://" + url
			protocolMsg = errors.AssumingHttpProtocol
		}
		if i, _ := cmd.Flags().GetCount("verbose"); i > 0 {
			utils.ErrPrintf(cmd, protocolMsg)
		}
	}
	client.ChangeBasePath(strings.Trim(url, "/") + "/v3")
}

// Used for on-prem KafkaRest commands
func getKafkaRestClientAndContext(cmd *cobra.Command, kafkaRest *cmd.KafkaREST) (*kafkarestv3.APIClient, context.Context, error) {
	url, err := getKafkaRestUrl(cmd)
	if err != nil {
		return nil, nil, err
	}
	kafkaRestClient := kafkaRest.Client
	setServerURL(cmd, kafkaRestClient, url)
	return kafkaRestClient, kafkaRest.Context, nil
}

// Used for on-prem KafkaRest commands
// Fetch rest url from flag, otherwise from CONFLUENT_REST_URL
func getKafkaRestUrl(cmd *cobra.Command) (string, error) {
	if cmd.Flags().Changed("url") {
		url, err := cmd.Flags().GetString("url")
		if err != nil {
			return "", err
		}
		return url, nil
	}
	if restUrl := os.Getenv(CONFLUENT_REST_URL); restUrl != "" {
		return restUrl, nil
	}
	return "", errors.NewErrorWithSuggestions(errors.KafkaRestUrlNotFoundErrorMsg, errors.KafkaRestUrlNotFoundSuggestions)
}

// Converts ACLBinding to Kafka REST ClustersClusterIdAclsGetOpts
func aclBindingToClustersClusterIdAclsGetOpts(acl *schedv1.ACLBinding) kafkarestv3.ClustersClusterIdAclsGetOpts {
	var opts kafkarestv3.ClustersClusterIdAclsGetOpts

	if acl.Pattern.ResourceType != schedv1.ResourceTypes_UNKNOWN {
		opts.ResourceType = optional.NewInterface(kafkarestv3.AclResourceType(acl.Pattern.ResourceType.String()))
	}

	opts.ResourceName = optional.NewString(acl.Pattern.Name)

	if acl.Pattern.PatternType != schedv1.PatternTypes_UNKNOWN {
		opts.PatternType = optional.NewInterface(kafkarestv3.AclPatternType(acl.Pattern.PatternType.String()))
	}

	opts.Principal = optional.NewString(acl.Entry.Principal)
	opts.Host = optional.NewString(acl.Entry.Host)

	if acl.Entry.Operation != schedv1.ACLOperations_UNKNOWN {
		opts.Operation = optional.NewInterface(kafkarestv3.AclOperation(acl.Entry.Operation.String()))
	}

	if acl.Entry.PermissionType != schedv1.ACLPermissionTypes_UNKNOWN {
		opts.Permission = optional.NewInterface(kafkarestv3.AclPermission(acl.Entry.PermissionType.String()))
	}

	return opts
}

// Converts ACLBinding to Kafka REST ClustersClusterIdAclsPostOpts
func aclBindingToClustersClusterIdAclsPostOpts(acl *schedv1.ACLBinding) kafkarestv3.ClustersClusterIdAclsPostOpts {
	var aclRequestData kafkarestv3.CreateAclRequestData

	if acl.Pattern.ResourceType != schedv1.ResourceTypes_UNKNOWN {
		aclRequestData.ResourceType = kafkarestv3.AclResourceType(acl.Pattern.ResourceType.String())
	}

	if acl.Pattern.PatternType != schedv1.PatternTypes_UNKNOWN {
		aclRequestData.PatternType = kafkarestv3.AclPatternType(acl.Pattern.PatternType.String())
	}

	aclRequestData.ResourceName = acl.Pattern.Name
	aclRequestData.Principal = acl.Entry.Principal
	aclRequestData.Host = acl.Entry.Host

	if acl.Entry.Operation != schedv1.ACLOperations_UNKNOWN {
		aclRequestData.Operation = kafkarestv3.AclOperation(acl.Entry.Operation.String())
	}

	if acl.Entry.PermissionType != schedv1.ACLPermissionTypes_UNKNOWN {
		aclRequestData.Permission = kafkarestv3.AclPermission(acl.Entry.PermissionType.String())
	}

	var opts kafkarestv3.ClustersClusterIdAclsPostOpts
	opts.CreateAclRequestData = optional.NewInterface(aclRequestData)

	return opts
}

// Converts ACLFilter to Kafka REST ClustersClusterIdAclsDeleteOpts
func aclFilterToClustersClusterIdAclsDeleteOpts(acl *schedv1.ACLFilter) kafkarestv3.ClustersClusterIdAclsDeleteOpts {
	var opts kafkarestv3.ClustersClusterIdAclsDeleteOpts

	if acl.PatternFilter.ResourceType != schedv1.ResourceTypes_UNKNOWN {
		opts.ResourceType = optional.NewInterface(kafkarestv3.AclResourceType(acl.PatternFilter.ResourceType.String()))
	}

	opts.ResourceName = optional.NewString(acl.PatternFilter.Name)

	if acl.PatternFilter.PatternType != schedv1.PatternTypes_UNKNOWN {
		opts.PatternType = optional.NewInterface(kafkarestv3.AclPatternType(acl.PatternFilter.PatternType.String()))
	}

	opts.Principal = optional.NewString(acl.EntryFilter.Principal)
	opts.Host = optional.NewString(acl.EntryFilter.Host)

	if acl.EntryFilter.Operation != schedv1.ACLOperations_UNKNOWN {
		opts.Operation = optional.NewInterface(kafkarestv3.AclOperation(acl.EntryFilter.Operation.String()))
	}

	if acl.EntryFilter.PermissionType != schedv1.ACLPermissionTypes_UNKNOWN {
		opts.Permission = optional.NewInterface(kafkarestv3.AclPermission(acl.EntryFilter.PermissionType.String()))
	}

	return opts
}
