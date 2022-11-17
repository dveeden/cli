package iam

import (
	"context"
	"encoding/json"
	"os"

	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
	"github.com/tidwall/pretty"

	mdsv2alpha1 "github.com/confluentinc/ccloud-sdk-go-v1-public/mdsv2alpha1"
	mds "github.com/confluentinc/mds-sdk-go/mdsv1"

	pcmd "github.com/confluentinc/cli/internal/pkg/cmd"
	v1 "github.com/confluentinc/cli/internal/pkg/config/v1"
)

var (
	roleFields = []string{"Name", "AccessPolicy"}
	roleLabels = []string{"Name", "Access Policy"}
)

type roleCommand struct {
	*pcmd.AuthenticatedStateFlagCommand
	cfg *v1.Config
}

type prettyRole struct {
	Name         string
	AccessPolicy string
}

func newRoleCommand(cfg *v1.Config, prerunner pcmd.PreRunner) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "role",
		Short: "Manage RBAC and IAM roles.",
		Long:  "Manage Role-Based Access Control (RBAC) and Identity and Access Management (IAM) roles.",
	}

	c := &roleCommand{cfg: cfg}

	if cfg.IsOnPremLogin() {
		c.AuthenticatedStateFlagCommand = pcmd.NewAuthenticatedWithMDSStateFlagCommand(cmd, prerunner)
	} else {
		c.AuthenticatedStateFlagCommand = pcmd.NewAuthenticatedStateFlagCommand(cmd, prerunner)
	}

	c.AddCommand(c.newDescribeCommand())
	c.AddCommand(c.newListCommand())

	return c.Command
}

func (c *roleCommand) createContext() context.Context {
	if c.cfg.IsCloudLogin() {
		return context.WithValue(context.Background(), mdsv2alpha1.ContextAccessToken, c.Context.GetAuthToken())
	} else {
		return context.WithValue(context.Background(), mds.ContextAccessToken, c.Context.GetAuthToken())
	}
}

func createPrettyRole(role mds.Role) (*prettyRole, error) {
	marshalled, err := json.Marshal(role.AccessPolicy)
	if err != nil {
		return nil, err
	}

	return &prettyRole{
		role.Name,
		string(pretty.Pretty(marshalled)),
	}, nil
}

func createPrettyRoleV2(role mdsv2alpha1.Role) (*prettyRole, error) {
	marshalled, err := json.Marshal(role.Policies)
	if err != nil {
		return nil, err
	}

	return &prettyRole{
		role.Name,
		string(pretty.Pretty(marshalled)),
	}, nil
}

func outputTable(data [][]string) {
	tablePrinter := tablewriter.NewWriter(os.Stdout)
	tablePrinter.SetAutoWrapText(false)
	tablePrinter.SetAutoFormatHeaders(false)
	tablePrinter.SetHeader(roleLabels)
	tablePrinter.AppendBulk(data)
	tablePrinter.SetBorder(false)
	tablePrinter.Render()
}
