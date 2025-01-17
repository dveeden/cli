package netrc

import (
	"strings"

	"github.com/confluentinc/cli/internal/pkg/errors"
)

type MachineContextInfo struct {
	CredentialType string
	Username       string
	URL            string
	CaCertPath     string
}

func ParseNetrcMachineName(machineName string) (*MachineContextInfo, error) {
	if !strings.HasPrefix(machineName, netrcCredentialsPrefix) {
		return nil, errors.New("Incorrect machine name format")
	}

	// example: machinename = confluent-cli:ccloud-username-password:login-caas-team+integ-cli@confluent.io-https://devel.cpdev.cloud
	credTypeAndContextNameString := suffixFromIndex(machineName, len(netrcCredentialsPrefix)+1)

	// credTypeAndContextName = ccloud-username-password:login-caas-team+integ-cli@confluent.io-https://devel.cpdev.cloud
	credType, contextNameString, err := extractCredentialType(credTypeAndContextNameString)
	if err != nil {
		return nil, err
	}

	// contextNameString = login-caas-team+integ-cli@confluent.io-https://devel.cpdev.cloud
	username, url, caCertPath, err := parseContextName(contextNameString)
	if err != nil {
		return nil, err
	}

	// username = caas-team+integ-cli@confluent.io
	// url = https://devel.cpdev.cloud
	return &MachineContextInfo{
		CredentialType: credType,
		Username:       username,
		URL:            url,
		CaCertPath:     caCertPath,
	}, nil

}

func extractCredentialType(nameSubstring string) (credType string, rest string, err error) {
	if strings.HasPrefix(nameSubstring, mdsUsernamePasswordString) {
		credType = mdsUsernamePasswordString
	} else if strings.HasPrefix(nameSubstring, ccloudUsernamePasswordString) {
		credType = ccloudUsernamePasswordString
	} else {
		return "", "", errors.New("Incorrect machine name format")
	}
	// +1 to remove the character ":"
	rest = suffixFromIndex(nameSubstring, len(credType)+1)
	return
}

func parseContextName(nameSubstring string) (username string, url string, caCertPath string, err error) {
	contextNamePrefix := "login-"
	if !strings.HasPrefix(nameSubstring, contextNamePrefix) {
		return "", "", "", errors.New("Incorrect context name format")
	}

	contextName := suffixFromIndex(nameSubstring, len(contextNamePrefix))

	urlIndex := strings.Index(contextName, "http")

	// -1 to exclude "-"
	username = prefixToIndex(contextName, urlIndex-1)

	// +1 to exclude "-"
	rest := suffixFromIndex(contextName, len(username)+1)

	questionMarkIndex := strings.Index(rest, "?")
	if questionMarkIndex == -1 {
		url = rest
	} else {
		url = prefixToIndex(rest, questionMarkIndex)
		caCertPath = suffixFromIndex(rest, questionMarkIndex+len("cacertpath")+2)
	}
	return
}

func suffixFromIndex(s string, index int) string {
	runes := []rune(s)
	return string(runes[index:])
}

func prefixToIndex(s string, index int) string {
	runes := []rune(s)
	return string(runes[:index])
}
