package errors

import (
	"fmt"
	"reflect"

	"github.com/hashicorp/go-multierror"
	"github.com/spf13/cobra"

	corev1 "github.com/confluentinc/cc-structs/kafka/core/v1"
	"github.com/confluentinc/ccloud-sdk-go"
	mds "github.com/confluentinc/mds-sdk-go/mdsv1"
)

var messages = map[error]string{
	ErrNoContext:      UserNotLoggedInErrMsg,
	ErrNotLoggedIn:    UserNotLoggedInErrMsg,
	ErrNotImplemented: "Sorry, this functionality is not yet available in the CLI.",
	ErrNoKafkaContext: "You must pass --cluster or set an active kafka in your context with 'kafka cluster use'",
}

// HandleCommon provides standard error messaging for common errors.
func HandleCommon(err error, cmd *cobra.Command) error {
	// Give an indication of successful completion
	if err == nil {
		return nil
	}
	cmd.SilenceUsage = true

	// [CLI-505] mds.GenericOpenAPIErrors are not hashable so messages[err] panics;
	// so check if the error is hashable before trying to use messages[err]
	// (This is a recommended way of checking whether a variable is hashable, see
	//  https://groups.google.com/forum/#!topic/golang-nuts/fpzQdHBdV3c )
	k := reflect.TypeOf(err).Kind()
	hashable := k < reflect.Array || k == reflect.Ptr || k == reflect.UnsafePointer
	if hashable {
		if msg, ok := messages[err]; ok {
			return fmt.Errorf(msg)
		}
	}

	switch e := err.(type) {
	case mds.GenericOpenAPIError:
		return fmt.Errorf("metadata service backend error: " + e.Error() + ": " + string(e.Body()))
	case *corev1.Error:
		var result error
		result = multierror.Append(result, e)
		for name, msg := range e.GetNestedErrors() {
			result = multierror.Append(result, fmt.Errorf("%s: %s", name, msg))
		}
		return result
	case *UnspecifiedAPIKeyError:
		return fmt.Errorf("no API key selected for %s, please select an api-key first (e.g., with `api-key use`)", e.ClusterID)
	case *UnspecifiedCredentialError:
		// TODO: Add more context to credential error messages (add variable error).
		return fmt.Errorf(ConfigUnspecifiedCredentialError, e.ContextName)
	case *UnspecifiedPlatformError:
		// TODO: Add more context to platform error messages (add variable error).
		return fmt.Errorf(ConfigUnspecifiedPlatformError, e.ContextName)
	case *ccloud.InvalidLoginError:
		return fmt.Errorf("You have entered an incorrect username or password. Please try again.")
	case *ccloud.InvalidTokenError:
		return fmt.Errorf(CorruptedAuthTokenErrorMsg)
	}
	return err
}
