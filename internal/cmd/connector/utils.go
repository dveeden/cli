package connector

import (
	"io/ioutil"
	"strings"

	"github.com/ghodss/yaml"
	"github.com/spf13/cobra"

	"github.com/confluentinc/cli/internal/pkg/errors"
)

func FormatDescription(description string, cliName string) string {
	return strings.ReplaceAll(description, "{{.CLIName}}", cliName)
}

func getConfig(cmd *cobra.Command) (*map[string]string, error) {
	filename, err := cmd.Flags().GetString("config")
	if err != nil {
		return nil, errors.Wrap(err, "error reading --config as string")
	}
	var options map[string]string
	yamlFile, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, errors.Wrapf(err, "unable to read config file %s", filename)
	}
	if len(yamlFile) == 0 {
		return nil, errors.Wrap(errors.ErrEmptyConfigFile, "empty file")
	}
	err = yaml.Unmarshal(yamlFile, &options)
	if err != nil {
		return nil, errors.Wrapf(err, "unable to parse config %s", filename)
	}
	_, nameExists := options["name"]
	_, classExists := options["connector.class"]
	if !nameExists || !classExists {
		return nil, errors.Wrapf(errors.ErrEmptyConfigFile, "name and connector.class are required")
	}
	return &options, nil
}