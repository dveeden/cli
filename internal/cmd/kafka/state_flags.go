package kafka

import (
	"github.com/confluentinc/cli/internal/pkg/cmd"
	"github.com/spf13/pflag"
)

var ClusterCloudSubcommandStateFlags = map[string]*pflag.FlagSet {
	"cluster"	:	cmd.EnvironmentContextSet(),
}

var ACLSubcommandStateFlags = map[string]*pflag.FlagSet {
	"acl"	:	cmd.ClusterEnvironmentContextSet(),
}

var TopicSubcommandStateFlags = map[string]*pflag.FlagSet {
	"topic"		:	cmd.ClusterEnvironmentContextSet(),
}

var LinkSubcommandStateFlags = map[string]*pflag.FlagSet {
	"link"		:	cmd.ClusterEnvironmentContextSet(),
}
