package common

import (
	"testing"

	"github.com/stretchr/testify/require"

	terminal "github.com/confluentinc/cli/command"
)

func TestCompletionBash(t *testing.T) {
	req := require.New(t)

	root, prompt := terminal.BuildRootCommand()
	cmd := NewCompletionCmd(root, prompt)
	root.AddCommand(cmd)

	output, err := terminal.ExecuteCommand(root, "completion", "bash")
	req.NoError(err)
	req.Contains(output, "bash completion for")
}

func TestCompletionZsh(t *testing.T) {
	req := require.New(t)

	root, prompt := terminal.BuildRootCommand()
	cmd := NewCompletionCmd(root, prompt)
	root.AddCommand(cmd)

	output, err := terminal.ExecuteCommand(root, "completion", "zsh")
	req.NoError(err)
	req.Contains(output, "#compdef")
}

func TestCompletionUnknown(t *testing.T) {
	req := require.New(t)

	root, prompt := terminal.BuildRootCommand()
	cmd := NewCompletionCmd(root, prompt)
	root.AddCommand(cmd)

	output, err := terminal.ExecuteCommand(root, "completion", "newsh")
	req.Error(err)
	req.Contains(output, "Error: unsupported shell type \"newsh\"")
}

func TestCompletionNone(t *testing.T) {
	req := require.New(t)

	root, prompt := terminal.BuildRootCommand()
	cmd := NewCompletionCmd(root, prompt)
	root.AddCommand(cmd)

	output, err := terminal.ExecuteCommand(root, "completion")
	req.Error(err)
	req.Contains(output, "Error: accepts 1 arg(s), received 0")
}
