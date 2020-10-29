package schema_registry

import (
	"github.com/confluentinc/cli/internal/pkg/cmd"
	"github.com/spf13/pflag"
)

var SchemaClusterSubcommandFlags = map[string]*pflag.FlagSet {
	"enable" 	: cmd.EnvironmentContextSet(),
	"describe" 	: cmd.CombineFlagSet(cmd.EnvironmentContextSet(), cmd.KeySecretSet()),
	"update" 	: cmd.CombineFlagSet(cmd.EnvironmentContextSet(), cmd.KeySecretSet()),
}

var SchemaSubjectSubcommandFlags = map[string]*pflag.FlagSet {
	"subject"	: cmd.CombineFlagSet(cmd.EnvironmentContextSet(), cmd.KeySecretSet()),
}

var SchemaSubcommandFlags = map[string]*pflag.FlagSet {
	"schema"	: cmd.CombineFlagSet(cmd.EnvironmentContextSet(), cmd.KeySecretSet()),
}


