package config

import (
	"fmt"
	"os"
	"strings"
)

type ConfigurationV1 struct {
	Username string    `json:"username" yaml:"username"`
	Password string    `json:"password" yaml:"password"`
	TeamID   string    `json:"team_id" yaml:"team_id"`
	Packages []Package `json:"packages" yaml:"packages"`
}

func (config *ConfigurationV1) GetPackages() []Package {
	return config.Packages
}

func (config *ConfigurationV1) ToV2() (*ConfigurationV2, error) {
	password := config.Password

	if envKey, found := strings.CutPrefix(password, "ENV:"); found {
		password = os.Getenv(envKey)
	} else if keychainLabel, found := strings.CutPrefix(password, "@keychain:"); found {
		var err error

		if password, err = getPasswordFromKeychain(keychainLabel); err != nil {
			return nil, fmt.Errorf("get password from keychain: %w", err)
		}
	}

	return &ConfigurationV2{
		NotaryAuth: &ConfigurationV2_NotaryAuth{
			username:            config.Username,
			appSpecificPassword: password,
			teamId:              config.TeamID,
		},
		Packages: config.Packages,
	}, nil
}
