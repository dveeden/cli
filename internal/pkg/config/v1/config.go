package v1

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/confluentinc/cli/internal/pkg/log"

	orgv1 "github.com/confluentinc/cc-structs/kafka/org/v1"
	"github.com/google/uuid"
	"github.com/hashicorp/go-version"

	"github.com/confluentinc/cli/internal/pkg/config"
	"github.com/confluentinc/cli/internal/pkg/errors"
)

const (
	defaultConfigFileFmt = "%s/.confluent/config.json"
	emptyFieldIndicator  = "EMPTY"
)

var (
	ver, _ = version.NewVersion("1.0.0")
)

// Config represents the CLI configuration.
type Config struct {
	*config.BaseConfig
	DisableUpdateCheck     bool                     `json:"disable_update_check"`
	DisableUpdates         bool                     `json:"disable_updates"`
	NoBrowser              bool                     `json:"no_browser" hcl:"no_browser"`
	Platforms              map[string]*Platform     `json:"platforms,omitempty"`
	Credentials            map[string]*Credential   `json:"credentials,omitempty"`
	Contexts               map[string]*Context      `json:"contexts,omitempty"`
	ContextStates          map[string]*ContextState `json:"context_states,omitempty"`
	CurrentContext         string                   `json:"current_context"`
	AnonymousId            string                   `json:"anonymous_id,omitempty"`
	IsTest                 bool                     `json:"-"`
	overwrittenAccount     *orgv1.Account
	overwrittenCurrContext string
	overwrittenActiveKafka string
}

func (c *Config) SetOverwrittenAccount(acct *orgv1.Account) {
	if c.overwrittenAccount == nil {
		c.overwrittenAccount = acct
	}
}

func (c *Config) SetOverwrittenCurrContext(contextName string) {
	if contextName == "" {
		contextName = emptyFieldIndicator
	}
	if c.overwrittenCurrContext == "" {
		c.overwrittenCurrContext = contextName
	}
}

func (c *Config) SetOverwrittenActiveKafka(clusterId string) {
	if clusterId == "" {
		clusterId = emptyFieldIndicator
	}
	if c.overwrittenActiveKafka == "" {
		c.overwrittenActiveKafka = clusterId
	}
}

func New() *Config {
	return &Config{
		BaseConfig:    config.NewBaseConfig(ver),
		Platforms:     make(map[string]*Platform),
		Credentials:   make(map[string]*Credential),
		Contexts:      make(map[string]*Context),
		ContextStates: make(map[string]*ContextState),
		AnonymousId:   uuid.New().String(),
	}
}

// Load reads the CLI config from disk.
// Save a default version if none exists yet.
func (c *Config) Load() error {
	filename := c.GetFilename()
	input, err := ioutil.ReadFile(filename)
	if err != nil {
		if os.IsNotExist(err) {
			// Save a default version if none exists yet.
			if err := c.Save(); err != nil {
				return errors.Wrapf(err, errors.UnableToCreateConfigErrorMsg)
			}
			return nil
		}
		return errors.Wrapf(err, errors.UnableToReadConfigErrorMsg, filename)
	}
	err = json.Unmarshal(input, c)
	if c.Ver.Compare(ver) < 0 {
		return errors.Errorf(errors.ConfigNotUpToDateErrorMsg, c.Ver, ver)
	} else if c.Ver.Compare(ver) > 0 {
		if c.Ver.Equal(version.Must(version.NewVersion("3.0.0"))) {
			// The user is a CP user who downloaded the v2 CLI instead of running `confluent update --major`,
			// so their config files weren't merged and migrated. Migrate this config to avoid an error.
			c.Ver = config.Version{Version: version.Must(version.NewVersion("1.0.0"))}
			for name := range c.Contexts {
				c.Contexts[name].NetrcMachineName = name
			}
		} else {
			return errors.Errorf(errors.InvalidConfigVersionErrorMsg, c.Ver)
		}
	}
	if err != nil {
		return errors.Wrapf(err, errors.ParseConfigErrorMsg, filename)
	}
	for _, context := range c.Contexts {
		// Some "pre-validation"
		if context.Name == "" {
			return errors.NewCorruptedConfigError(errors.NoNameContextErrorMsg, "", c.Filename)
		}
		if context.CredentialName == "" {
			return errors.NewCorruptedConfigError(errors.UnspecifiedCredentialErrorMsg, context.Name, c.Filename)
		}
		if context.PlatformName == "" {
			return errors.NewCorruptedConfigError(errors.UnspecifiedPlatformErrorMsg, context.Name, c.Filename)
		}
		context.State = c.ContextStates[context.Name]
		context.Credential = c.Credentials[context.CredentialName]
		context.Platform = c.Platforms[context.PlatformName]
		context.Config = c
		if context.KafkaClusterContext == nil {
			return errors.NewCorruptedConfigError(errors.MissingKafkaClusterContextErrorMsg, context.Name, c.Filename)
		}
		context.KafkaClusterContext.Context = context
	}
	return c.Validate()
}

// Save writes the CLI config to disk.
func (c *Config) Save() error {
	tempKafka := c.resolveOverwrittenKafka()
	tempAccount := c.resolveOverwrittenAccount()
	tempContext := c.resolveOverwrittenContext()

	if err := c.Validate(); err != nil {
		return err
	}

	cfg, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return errors.Wrapf(err, errors.MarshalConfigErrorMsg)
	}

	filename := c.GetFilename()

	if err := os.MkdirAll(filepath.Dir(filename), 0700); err != nil {
		return errors.Wrapf(err, errors.CreateConfigDirectoryErrorMsg, filename)
	}

	if err := ioutil.WriteFile(filename, cfg, 0600); err != nil {
		return errors.Wrapf(err, errors.CreateConfigFileErrorMsg, filename)
	}

	c.restoreOverwrittenContext(tempContext)
	c.restoreOverwrittenAccount(tempAccount)
	c.restoreOverwrittenKafka(tempKafka)

	return nil
}

// If active Kafka cluster has been overwritten by flag value; if so, replace with previous active kafka
// Return the flag value so that it can be restored after writing to file so that continued execution uses flag value
// This prevents flags from updating state
func (c *Config) resolveOverwrittenKafka() string {
	ctx := c.Context()
	var tempKafka string
	if c.overwrittenActiveKafka != "" && ctx != nil && ctx.KafkaClusterContext != nil {
		if c.overwrittenActiveKafka == emptyFieldIndicator {
			c.overwrittenActiveKafka = ""
		}
		tempKafka = ctx.KafkaClusterContext.GetActiveKafkaClusterId()
		ctx.KafkaClusterContext.SetActiveKafkaCluster(c.overwrittenActiveKafka)
	}
	return tempKafka
}

// Restore the flag cluster back into the struct so that it is used for any execution after Save()
func (c *Config) restoreOverwrittenKafka(tempKafka string) {
	ctx := c.Context()
	if tempKafka != "" {
		ctx.KafkaClusterContext.SetActiveKafkaCluster(tempKafka)
	}
}

// Switch the initial config context back into the struct so that it is saved and not the flag value
// Return the overwriting flag context value so that it can be restored after writing the file
func (c *Config) resolveOverwrittenContext() string {
	var tempContext string
	if c.overwrittenCurrContext != "" && c != nil {
		if c.overwrittenCurrContext == emptyFieldIndicator {
			c.overwrittenCurrContext = ""
		}
		tempContext = c.CurrentContext
		c.CurrentContext = c.overwrittenCurrContext
	}
	return tempContext
}

// Restore the flag context back into the struct so that it is used for any execution after Save()
func (c *Config) restoreOverwrittenContext(tempContext string) {
	if tempContext != "" {
		c.CurrentContext = tempContext
	}
}

// Switch the initial config account back into the struct so that it is saved and not the flag value
// Return the overwriting flag account value so that it can be restored after writing the file
func (c *Config) resolveOverwrittenAccount() *orgv1.Account {
	ctx := c.Context()
	var tempAccount *orgv1.Account
	if c.overwrittenAccount != nil && ctx != nil && ctx.State != nil && ctx.State.Auth != nil {
		tempAccount = ctx.State.Auth.Account
		ctx.State.Auth.Account = c.overwrittenAccount
	}
	return tempAccount
}

// Restore the flag account back into the struct so that it is used for any execution after Save()
func (c *Config) restoreOverwrittenAccount(tempAccount *orgv1.Account) {
	ctx := c.Context()
	if tempAccount != nil {
		ctx.State.Auth.Account = tempAccount
	}
}

func (c *Config) Validate() error {
	// Validate that current context exists.
	if c.CurrentContext != "" {
		if _, ok := c.Contexts[c.CurrentContext]; !ok {
			log.CliLogger.Trace("current context does not exist")
			return errors.NewCorruptedConfigError(errors.CurrentContextNotExistErrorMsg, c.CurrentContext, c.Filename)
		}
	}
	// Validate that every context:
	// 1. Has no hanging references between the context and the config.
	// 2. Is mapped by name correctly in the config.
	for _, context := range c.Contexts {
		err := context.validate()
		if err != nil {
			log.CliLogger.Trace("context validation error")
			return err
		}
		if _, ok := c.Credentials[context.CredentialName]; !ok {
			log.CliLogger.Trace("unspecified credential error")
			return errors.NewCorruptedConfigError(errors.UnspecifiedCredentialErrorMsg, context.Name, c.Filename)
		}
		if _, ok := c.Platforms[context.PlatformName]; !ok {
			log.CliLogger.Trace("unspecified platform error")
			return errors.NewCorruptedConfigError(errors.UnspecifiedPlatformErrorMsg, context.Name, c.Filename)
		}
		if _, ok := c.ContextStates[context.Name]; !ok {
			c.ContextStates[context.Name] = new(ContextState)
		}
		if *c.ContextStates[context.Name] != *context.State {
			log.CliLogger.Tracef("state of context %s in config does not match actual state of context", context.Name)
			return errors.NewCorruptedConfigError(errors.ContextStateMismatchErrorMsg, context.Name, c.Filename)
		}
	}
	// Validate that all context states are mapped to an existing context.
	for contextName := range c.ContextStates {
		if _, ok := c.Contexts[contextName]; !ok {
			log.CliLogger.Trace("context state mapped to nonexistent context")
			return errors.NewCorruptedConfigError(errors.ContextStateNotMappedErrorMsg, contextName, c.Filename)
		}
	}

	return nil
}

// DeleteContext deletes the specified context, and returns an error if it's not found.
func (c *Config) DeleteContext(name string) error {
	if _, err := c.FindContext(name); err != nil {
		return err
	}
	delete(c.Contexts, name)
	delete(c.ContextStates, name)

	if name == c.CurrentContext {
		c.CurrentContext = ""
	}

	return c.Save()
}

// FindContext finds a context by name, and returns nil if not found.
func (c *Config) FindContext(name string) (*Context, error) {
	context, ok := c.Contexts[name]
	if !ok {
		return nil, fmt.Errorf(errors.ContextDoesNotExistErrorMsg, name)
	}
	return context, nil
}

func (c *Config) AddContext(name, platformName, credentialName string, kafkaClusters map[string]*KafkaClusterConfig, kafka string, schemaRegistryClusters map[string]*SchemaRegistryCluster, state *ContextState, orgResourceId string) error {
	if _, ok := c.Contexts[name]; ok {
		return fmt.Errorf(errors.ContextAlreadyExistsErrorMsg, name)
	}

	credential, ok := c.Credentials[credentialName]
	if !ok {
		return fmt.Errorf(errors.CredentialNotFoundErrorMsg, credentialName)
	}

	platform, ok := c.Platforms[platformName]
	if !ok {
		return fmt.Errorf(errors.PlatformNotFoundErrorMsg, platformName)
	}

	ctx, err := newContext(name, platform, credential, kafkaClusters, kafka, schemaRegistryClusters, state, c, orgResourceId)
	if err != nil {
		return err
	}

	c.Contexts[name] = ctx
	c.ContextStates[name] = ctx.State

	if err := c.Validate(); err != nil {
		return err
	}

	return c.Save()
}

// CreateContext creates a new context.
func (c *Config) CreateContext(name, bootstrapURL, apiKey, apiSecret string) error {
	apiKeyPair := &APIKeyPair{
		Key:    apiKey,
		Secret: apiSecret,
	}
	apiKeys := map[string]*APIKeyPair{
		apiKey: apiKeyPair,
	}
	kafkaClusterCfg := &KafkaClusterConfig{
		ID:        "anonymous-id",
		Name:      "anonymous-cluster",
		Bootstrap: bootstrapURL,
		APIKeys:   apiKeys,
		APIKey:    apiKey,
	}
	kafkaClusters := map[string]*KafkaClusterConfig{
		kafkaClusterCfg.ID: kafkaClusterCfg,
	}
	platform := &Platform{Server: bootstrapURL}

	// Inject credential and platforms name for now, until users can provide custom names.
	platform.Name = strings.TrimPrefix(platform.Server, "https://")

	// Hardcoded for now, since username/password isn't implemented yet.
	credential := &Credential{
		Username:       "",
		Password:       "",
		APIKeyPair:     apiKeyPair,
		CredentialType: APIKey,
	}

	switch credential.CredentialType {
	case Username:
		credential.Name = fmt.Sprintf("%s-%s", &credential.CredentialType, credential.Username)
	case APIKey:
		credential.Name = fmt.Sprintf("%s-%s", &credential.CredentialType, credential.APIKeyPair.Key)
	default:
		return errors.Errorf(errors.UnknownCredentialTypeErrorMsg, credential.CredentialType)
	}

	if err := c.SaveCredential(credential); err != nil {
		return err
	}

	if err := c.SavePlatform(platform); err != nil {
		return err
	}

	return c.AddContext(name, platform.Name, credential.Name, kafkaClusters, kafkaClusterCfg.ID, nil, nil, "")
}

// UseContext sets the current context, if it exists.
func (c *Config) UseContext(name string) error {
	if _, err := c.FindContext(name); err != nil {
		return err
	}
	c.CurrentContext = name
	return c.Save()
}

func (c *Config) SaveCredential(credential *Credential) error {
	if credential.Name == "" {
		return errors.New(errors.NoNameCredentialErrorMsg)
	}
	c.Credentials[credential.Name] = credential
	return c.Save()
}

func (c *Config) SavePlatform(platform *Platform) error {
	if platform.Name == "" {
		return errors.New(errors.NoNamePlatformErrorMsg)
	}
	c.Platforms[platform.Name] = platform
	return c.Save()
}

// Context returns the current context.
func (c *Config) Context() *Context {
	if c == nil {
		return nil
	}
	return c.Contexts[c.CurrentContext]
}

// CredentialType returns the credential type used in the current context: API key, username & password, or neither.
func (c *Config) CredentialType() CredentialType {
	if c.hasAPIKeyLogin() {
		return APIKey
	}

	if c.HasBasicLogin() {
		return Username
	}

	return None
}

// hasAPIKeyLogin returns true if the user has valid API Key credentials.
func (c *Config) hasAPIKeyLogin() bool {
	ctx := c.Context()
	return ctx != nil && ctx.Credential != nil && ctx.Credential.CredentialType == APIKey
}

// HasBasicLogin returns true if the user has valid username & password credentials.
func (c *Config) HasBasicLogin() bool {
	ctx := c.Context()
	if ctx == nil {
		return false
	}

	if c.IsCloudLogin() {
		return ctx.hasBasicCloudLogin()
	} else {
		return ctx.HasBasicMDSLogin()
	}
}

func (c *Config) ResetAnonymousId() error {
	c.AnonymousId = uuid.New().String()
	return c.Save()
}

func (c *Config) GetFilename() string {
	if c.Filename == "" {
		homedir, _ := os.UserHomeDir()
		c.Filename = filepath.FromSlash(fmt.Sprintf(defaultConfigFileFmt, homedir))
	}
	return c.Filename
}

func (c *Config) IsCloud() bool {
	ctx := c.Context()
	if ctx == nil {
		return false
	}

	return ctx.IsCloud(c.IsTest)
}

func (c *Config) IsCloudLogin() bool {
	return c.IsCloud() && !c.IsOrgSuspended()
}

func (c *Config) IsCloudLoginAllowFreeTrialEnded() bool {
	return c.IsCloud() && !c.IsLoginBlockedByOrgSuspension()
}

func (c *Config) IsOnPremLogin() bool {
	ctx := c.Context()
	return ctx != nil && ctx.PlatformName != "" && !c.IsCloud()
}

func (c *Config) IsOrgSuspended() bool {
	ctx := c.Context()
	if ctx.State == nil || ctx.State.Auth == nil || ctx.State.Auth.Organization == nil {
		log.CliLogger.Trace("current context state is not setup properly for checking org suspension status")
		return true
	}

	suspensionStatus := c.Context().GetSuspensionStatus()

	// is org suspended
	return c.isOrgSuspended(suspensionStatus)
}

func (c *Config) IsLoginBlockedByOrgSuspension() bool {
	ctx := c.Context()
	if ctx.State == nil || ctx.State.Auth == nil || ctx.State.Auth.Organization == nil {
		log.CliLogger.Trace("current context state is not setup properly for checking org suspension status")
		return true
	}

	suspensionStatus := c.Context().GetSuspensionStatus()

	// is org suspended
	if c.isOrgSuspended(suspensionStatus) {
		// is org suspended due to end of free trial
		return suspensionStatus.GetEventType() != orgv1.SuspensionEventType_SUSPENSION_EVENT_END_OF_FREE_TRIAL
	}
	return false
}

func (c *Config) isOrgSuspended(suspensionStatus *orgv1.SuspensionStatus) bool {
	return suspensionStatus != nil && (suspensionStatus.GetStatus() == orgv1.SuspensionStatusType_SUSPENSION_IN_PROGRESS || suspensionStatus.Status == orgv1.SuspensionStatusType_SUSPENSION_COMPLETED)
}

func (c *Config) GetLastUsedOrgId() string {
	if ctx := c.Context(); ctx != nil && ctx.LastOrgId != "" {
		return ctx.LastOrgId
	}
	return os.Getenv("CONFLUENT_CLOUD_ORGANIZATION_ID")
}
