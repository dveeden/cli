package featureflags

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	v1 "github.com/confluentinc/cli/internal/pkg/config/v1"
	dynamicconfig "github.com/confluentinc/cli/internal/pkg/dynamic-config"
	"github.com/confluentinc/cli/internal/pkg/utils"
)

const deprecationPrefix = "DEPRECATED: "

const (
	Announcements      = "cli.announcements"
	DeprecationNotices = "cli.deprecation_notices"
)

type Messages struct {
	CommandMessage string
	Flags          []string
	FlagMessages   []string
}

func NewMessages() *Messages {
	return &Messages{
		Flags:        []string{},
		FlagMessages: []string{},
	}
}

func GetAnnouncementsOrDeprecation(resp interface{}) map[string]*Messages {
	commandToMessages := make(map[string]*Messages)

	list, ok := resp.([]interface{})
	if !ok {
		fmt.Println("A")
		return commandToMessages
	}

	for _, data := range list {
		pair, ok := data.(map[string]interface{})
		if !ok {
			continue
		}
		message, ok := pair["message"].(string)
		if !ok {
			continue
		}
		pattern, ok := pair["pattern"].(string)
		if !ok {
			continue
		}

		subpatterns := strings.Split(pattern, " ")

		idx := 0
		for _, subpattern := range subpatterns {
			if strings.HasPrefix(subpattern, "-") {
				break
			}
			idx++
		}

		command := strings.Join(subpatterns[:idx], " ")

		if _, ok := commandToMessages[command]; !ok {
			commandToMessages[command] = NewMessages()
		}

		if idx == len(subpatterns) {
			commandToMessages[command].CommandMessage = message
		} else {
			for _, subpattern := range subpatterns[idx:] {
				flag := strings.TrimLeft(subpattern, "-")
				commandToMessages[command].Flags = append(commandToMessages[command].Flags, flag)
				commandToMessages[command].FlagMessages = append(commandToMessages[command].FlagMessages, message)
			}
		}
	}

	return commandToMessages
}

func DeprecateCommandTree(cmd *cobra.Command) {
	if cmd.Long != "" {
		cmd.Long = deprecationPrefix + cmd.Long
	}
	if cmd.Short != "" {
		cmd.Short = deprecationPrefix + cmd.Short
	}
	for _, subcommand := range cmd.Commands() {
		DeprecateCommandTree(subcommand)
	}
}

func DeprecateFlags(cmd *cobra.Command, flags []string) {
	for _, flag := range flags {
		if len(flag) == 1 {
			flag = cmd.Flags().ShorthandLookup(flag).Name
		}
		if cmd.Flag(flag) != nil {
			cmd.Flag(flag).Usage = deprecationPrefix + cmd.Flag(flag).Usage
		}
	}
	for _, subcommand := range cmd.Commands() {
		DeprecateFlags(subcommand, flags)
	}
}

func PrintAnnouncements(featureFlag string, ctx *dynamicconfig.DynamicContext, cmd *cobra.Command) {
	flagResponse := Manager.JsonVariation(featureFlag, ctx, v1.CliLaunchDarklyClient, true, []interface{}{})
	cmdToFlagsAndMsg := GetAnnouncementsOrDeprecation(flagResponse)
	for name, flagsAndMsg := range cmdToFlagsAndMsg {
		if strings.HasPrefix(cmd.CommandPath(), "confluent "+name) {
			if len(flagsAndMsg.Flags) == 0 {
				if featureFlag == DeprecationNotices {
					utils.ErrPrintln(cmd, fmt.Sprintf("`confluent %s` is deprecated: %s", name, flagsAndMsg.CommandMessage))
				} else {
					utils.ErrPrintln(cmd, flagsAndMsg.CommandMessage)
				}
			} else {
				for i, flag := range flagsAndMsg.Flags {
					if !cmd.Flags().Changed(flag) {
						continue
					}

					var msg string
					if featureFlag == DeprecationNotices {
						msg = fmt.Sprintf("The `--%s` flag is deprecated", flag)
						if flagsAndMsg.FlagMessages[i] == "" {
							msg = fmt.Sprintf("%s.", msg)
						} else {
							msg = fmt.Sprintf("%s: %s", msg, flagsAndMsg.FlagMessages[i])
						}
					} else {
						msg = flagsAndMsg.FlagMessages[i]
					}
					utils.ErrPrintln(cmd, msg)
				}
			}
		}
	}
}
