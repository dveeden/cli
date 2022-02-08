package kafka

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"strings"
	"text/template"

	productv1 "github.com/confluentinc/cc-structs/kafka/product/core/v1"
	schedv1 "github.com/confluentinc/cc-structs/kafka/scheduler/v1"
	"github.com/spf13/cobra"

	pcmd "github.com/confluentinc/cli/internal/pkg/cmd"
	v1 "github.com/confluentinc/cli/internal/pkg/config/v1"
	"github.com/confluentinc/cli/internal/pkg/errors"
	"github.com/confluentinc/cli/internal/pkg/examples"
	"github.com/confluentinc/cli/internal/pkg/form"
	"github.com/confluentinc/cli/internal/pkg/output"
	"github.com/confluentinc/cli/internal/pkg/utils"
)

const (
	skuBasic     = "basic"
	skuStandard  = "standard"
	skuDedicated = "dedicated"
)

var encryptionKeyPolicy = template.Must(template.New("encryptionKey").Parse(`{{range  $i, $accountID := .}}{{if $i}},{{end}}{
    "Sid" : "Allow Confluent account ({{$accountID}}) to use the key",
    "Effect" : "Allow",
    "Principal" : {
      "AWS" : ["arn:aws:iam::{{$accountID}}:root"]
    },
    "Action" : [ "kms:Encrypt", "kms:Decrypt", "kms:ReEncrypt*", "kms:GenerateDataKey*", "kms:DescribeKey" ],
    "Resource" : "*"
  }, {
    "Sid" : "Allow Confluent account ({{$accountID}}) to attach persistent resources",
    "Effect" : "Allow",
    "Principal" : {
      "AWS" : ["arn:aws:iam::{{$accountID}}:root"]
    },
    "Action" : [ "kms:CreateGrant", "kms:ListGrants", "kms:RevokeGrant" ],
    "Resource" : "*"
}{{end}}`))

var permitBYOKGCP = template.Must(template.New("byok_gcp_permissions").Parse(`Create a role with these permissions, add the identity as a member of your key, and grant your role to the member:

Permissions:
  - cloudkms.cryptoKeyVersions.useToDecrypt
  - cloudkms.cryptoKeyVersions.useToEncrypt
  - cloudkms.cryptoKeys.get

Identity:
  {{.ExternalIdentity}}`))

type validateEncryptionKeyInput struct {
	Cloud          string
	MetadataClouds []*schedv1.CloudMetadata
	AccountID      string
}

func (c *clusterCommand) newCreateCommand(cfg *v1.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create <name>",
		Short: "Create a Kafka cluster.",
		Long:  "Create a Kafka cluster.\n\nNote: You cannot use this command to create a cluster that is configured with AWS PrivateLink. You must use the UI to create a cluster of that configuration.",
		Args:  cobra.ExactArgs(1),
		RunE: pcmd.NewCLIRunE(func(cmd *cobra.Command, args []string) error {
			return c.create(cmd, args, form.NewPrompt(os.Stdin))
		}),
		Annotations: map[string]string{pcmd.RunRequirement: pcmd.RequireNonAPIKeyCloudLogin},
		Example: examples.BuildExampleString(
			examples.Example{
				Text: "Create a new dedicated cluster that uses a customer-managed encryption key in AWS:",
				Code: `confluent kafka cluster create sales092020 --cloud "aws" --region "us-west-2" --type "dedicated" --cku 1 --encryption-key "arn:aws:kms:us-west-2:111122223333:key/1234abcd-12ab-34cd-56ef-1234567890ab"`,
			},
			examples.Example{
				Text: "For more information, see https://docs.confluent.io/current/cloud/clusters/byok-encrypted-clusters.html.",
			},
		),
	}

	pcmd.AddCloudFlag(cmd)
	pcmd.AddRegionFlag(cmd, c.AuthenticatedCLICommand)
	cmd.Flags().String("availability", singleZone, fmt.Sprintf("Availability of the cluster. Allowed Values: %s, %s.", singleZone, multiZone))
	cmd.Flags().String("type", skuBasic, fmt.Sprintf("Type of the Kafka cluster. Allowed values: %s, %s, %s.", skuBasic, skuStandard, skuDedicated))
	cmd.Flags().Int("cku", 0, "Number of Confluent Kafka Units (non-negative). Required for Kafka clusters of type 'dedicated'.")
	cmd.Flags().String("encryption-key", "", "Encryption Key ID (e.g. for Amazon Web Services, the Amazon Resource Name of the key).")
	pcmd.AddContextFlag(cmd, c.CLICommand)
	if cfg.IsCloudLogin() {
		pcmd.AddEnvironmentFlag(cmd, c.AuthenticatedCLICommand)
	}
	pcmd.AddOutputFlag(cmd)

	_ = cmd.MarkFlagRequired("cloud")
	_ = cmd.MarkFlagRequired("region")

	return cmd
}

func (c *clusterCommand) create(cmd *cobra.Command, args []string, prompt form.Prompt) error {
	cloud, err := cmd.Flags().GetString("cloud")
	if err != nil {
		return err
	}

	region, err := cmd.Flags().GetString("region")
	if err != nil {
		return err
	}

	clouds, err := c.Client.EnvironmentMetadata.Get(context.Background())
	if err != nil {
		return err
	}

	if err := checkCloudAndRegion(cloud, region, clouds); err != nil {
		return err
	}

	availabilityString, err := cmd.Flags().GetString("availability")
	if err != nil {
		return err
	}

	availability, err := stringToAvailability(availabilityString)
	if err != nil {
		return err
	}

	typeString, err := cmd.Flags().GetString("type")
	if err != nil {
		return err
	}

	sku, err := stringToSku(typeString)
	if err != nil {
		return err
	}

	encryptionKeyID, err := cmd.Flags().GetString("encryption-key")
	if err != nil {
		return err
	}

	if encryptionKeyID != "" {
		input := validateEncryptionKeyInput{
			Cloud:          cloud,
			MetadataClouds: clouds,
			AccountID:      c.EnvironmentId(),
		}
		if err := c.validateEncryptionKey(cmd, prompt, input); err != nil {
			return err
		}
	}

	cfg := &schedv1.KafkaClusterConfig{
		AccountId:       c.EnvironmentId(),
		Name:            args[0],
		ServiceProvider: cloud,
		Region:          region,
		Durability:      availability,
		Deployment:      &schedv1.Deployment{Sku: sku},
		EncryptionKeyId: encryptionKeyID,
	}

	if cmd.Flags().Changed("cku") {
		cku, err := cmd.Flags().GetInt("cku")
		if err != nil {
			return err
		}
		if sku != productv1.Sku_DEDICATED {
			return errors.New(errors.CKUOnlyForDedicatedErrorMsg)
		}
		if cku <= 0 {
			return errors.New(errors.CKUMoreThanZeroErrorMsg)
		}
		cfg.Cku = int32(cku)
	}

	cluster, err := c.Client.Kafka.Create(context.Background(), cfg)
	if err != nil {
		// TODO: don't swallow validation errors (reportedly separately)
		return err
	}

	outputFormat, err := cmd.Flags().GetString(output.FlagName)
	if err != nil {
		return err
	}

	if outputFormat == output.Human.String() {
		utils.ErrPrintln(cmd, getKafkaProvisionEstimate(sku))
	}

	return outputKafkaClusterDescription(cmd, cluster)
}

func checkCloudAndRegion(cloudId string, regionId string, clouds []*schedv1.CloudMetadata) error {
	for _, cloud := range clouds {
		if cloudId == cloud.Id {
			for _, region := range cloud.Regions {
				if regionId == region.Id {
					if region.IsSchedulable {
						return nil
					} else {
						break
					}
				}
			}
			return errors.NewErrorWithSuggestions(fmt.Sprintf(errors.CloudRegionNotAvailableErrorMsg, regionId, cloudId),
				fmt.Sprintf(errors.CloudRegionNotAvailableSuggestions, cloudId, cloudId))
		}
	}
	return errors.NewErrorWithSuggestions(fmt.Sprintf(errors.CloudProviderNotAvailableErrorMsg, cloudId),
		errors.CloudProviderNotAvailableSuggestions)
}

func (c *clusterCommand) validateEncryptionKey(cmd *cobra.Command, prompt form.Prompt, input validateEncryptionKeyInput) error {
	switch input.Cloud {
	case "aws":
		return c.validateAWSEncryptionKey(cmd, prompt, input)
	case "gcp":
		return c.validateGCPEncryptionKey(cmd, prompt, input)
	default:
		return errors.New(errors.BYOKSupportErrorMsg)
	}
}

func (c *clusterCommand) validateGCPEncryptionKey(cmd *cobra.Command, prompt form.Prompt, input validateEncryptionKeyInput) error {
	ctx := context.Background()
	// The call is idempotent so repeated create commands return the same ID for the same account.
	externalID, err := c.Client.ExternalIdentity.CreateExternalIdentity(ctx, input.Cloud, input.AccountID)
	if err != nil {
		return err
	}
	buf := new(bytes.Buffer)
	err = permitBYOKGCP.Execute(buf, struct {
		ExternalIdentity string
	}{
		ExternalIdentity: externalID,
	})
	if err != nil {
		return err
	}
	buf.WriteString("\n\n")
	utils.Println(cmd, buf.String())

	promptMsg := "Please confirm you've authorized the key for this identity: " + externalID
	f := form.New(
		form.Field{ID: "authorized",
			Prompt:    promptMsg,
			IsYesOrNo: true})
	for {
		if err := f.Prompt(cmd, prompt); err != nil {
			utils.ErrPrintln(cmd, errors.FailedToReadConfirmationErrorMsg)
			continue
		}
		if !f.Responses["authorized"].(bool) {
			return errors.Errorf(errors.AuthorizeIdentityErrorMsg, externalID)

		}
		return nil
	}
}

func (c *clusterCommand) validateAWSEncryptionKey(cmd *cobra.Command, prompt form.Prompt, input validateEncryptionKeyInput) error {
	accounts := getEnvironmentsForCloud(input.Cloud, input.MetadataClouds)

	buf := new(bytes.Buffer)
	buf.WriteString(errors.CopyBYOKAWSPermissionsHeaderMsg)
	buf.WriteString("\n\n")
	if err := encryptionKeyPolicy.Execute(buf, accounts); err != nil {
		return errors.New(errors.FailedToRenderKeyPolicyErrorMsg)
	}
	buf.WriteString("\n\n")
	utils.Println(cmd, buf.String())

	promptMsg := "Please confirm you've authorized the key for these accounts: " + strings.Join(accounts, ", ")
	if len(accounts) == 1 {
		promptMsg = "Please confirm you've authorized the key for this account: " + accounts[0]
	}

	f := form.New(form.Field{ID: "authorized", Prompt: promptMsg, IsYesOrNo: true})
	for {
		if err := f.Prompt(cmd, prompt); err != nil {
			utils.ErrPrintln(cmd, errors.FailedToReadConfirmationErrorMsg)
			continue
		}
		if !f.Responses["authorized"].(bool) {
			return errors.Errorf(errors.AuthorizeAccountsErrorMsg, strings.Join(accounts, ", "))
		}
		return nil
	}
}

func getEnvironmentsForCloud(cloudId string, clouds []*schedv1.CloudMetadata) []string {
	var environments []string
	for _, cloud := range clouds {
		if cloudId == cloud.Id {
			for _, environment := range cloud.Accounts {
				environments = append(environments, environment.Id)
			}
			break
		}
	}
	return environments
}

func stringToAvailability(s string) (schedv1.Durability, error) {
	if s == singleZone {
		return schedv1.Durability_LOW, nil
	} else if s == multiZone {
		return schedv1.Durability_HIGH, nil
	}
	return schedv1.Durability_LOW, errors.NewErrorWithSuggestions(fmt.Sprintf(errors.InvalidAvailableFlagErrorMsg, s),
		fmt.Sprintf(errors.InvalidAvailableFlagSuggestions, singleZone, multiZone))
}

func stringToSku(s string) (productv1.Sku, error) {
	sku := productv1.Sku(productv1.Sku_value[strings.ToUpper(s)])
	switch sku {
	case productv1.Sku_BASIC, productv1.Sku_STANDARD, productv1.Sku_DEDICATED:
		break
	default:
		return productv1.Sku_UNKNOWN, errors.NewErrorWithSuggestions(fmt.Sprintf(errors.InvalidTypeFlagErrorMsg, s),
			fmt.Sprintf(errors.InvalidTypeFlagSuggestions, skuBasic, skuStandard, skuDedicated))
	}
	return sku, nil
}

func getKafkaProvisionEstimate(sku productv1.Sku) string {
	fmtEstimate := "It may take up to %s for the Kafka cluster to be ready."

	switch sku {
	case productv1.Sku_DEDICATED:
		return fmt.Sprintf(fmtEstimate, "1 hour") + " The organization admin will receive an email once the dedicated cluster is provisioned."
	default:
		return fmt.Sprintf(fmtEstimate, "5 minutes")
	}
}