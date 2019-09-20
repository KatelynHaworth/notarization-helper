package main

import (
	"encoding/json"
	"flag"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

var (
	usernameFlag = flag.String("username", "", "Specifies the Apple Developer account username for notarization")
	passwordFlag = flag.String("password", "", "Specifies the password for the Apple Developer account")
	teamIDFlag = flag.String("team-id", "", "Optional flag that specifies a specific Apple Developer team")
	fileFlag = flag.String("file", "", "Specifies the file to be notarized, if the file is a JSON or YAML file all other flags are ignored")
	bundleIDFlag = flag.String("bundle-id", "", "Specifies the primary bundle ID of the package to be notarized")
	stapleFlag = flag.Bool("staple", false, "Optional flag that specifies the file should be stapled after notarization")

	errBadFlags = errors.New("missing command line flags")
)

type Configuration struct {
	Username string `json:"username" yaml:"username"`
	Password string `json:"password" yaml:"password"`
	TeamID   string `json:"team_id" yaml:"team_id"`
	Packages []Package `json:"packages" yaml:"packages"`
}

type Package struct {
	BundleID string `json:"bundle_id" yaml:"bundle_id"`
	File string `json:"file" yaml:"file"`
	Staple   bool `json:"staple" yaml:"staple"`
}

func loadConfiguration() (*Configuration, error) {
	flag.Parse()
	config := new(Configuration)

	switch {
	case len(*fileFlag) > 0 && filepath.Ext(*fileFlag) == ".json":
		if err := loadConfigurationFromJSON(config, *fileFlag); err != nil {
			return nil, errors.Wrap(err, "parse configuration from JSON file")
		}

	case len(*fileFlag) > 0 && (filepath.Ext(*fileFlag) == ".yaml" || filepath.Ext(*fileFlag) == ".yml"):
		if err := loadConfigurationFromYAML(config, *fileFlag); err != nil {
			return nil, errors.Wrap(err, "parse configuration from YAML file")
		}

	case len(*fileFlag) == 0 || len(*usernameFlag) == 0 || len(*passwordFlag) == 0 || len(*bundleIDFlag) == 0:
		return nil, errBadFlags

	default:
		config.Username = *usernameFlag
		config.Password = *passwordFlag
		config.TeamID = *teamIDFlag
		config.Packages = append(config.Packages, Package{
			BundleID: *bundleIDFlag,
			File:     *fileFlag,
			Staple:   *stapleFlag,
		})
	}

	return config, nil
}

func loadConfigurationFromJSON(config *Configuration, path string) error {
	source, err := os.OpenFile(path, os.O_RDONLY, 0000)
	if err != nil {
		return errors.Wrap(err, "open configuration file")
	}
	defer source.Close()

	return json.NewDecoder(source).Decode(config)
}

func loadConfigurationFromYAML(config *Configuration, path string) error {
	source, err := os.OpenFile(path, os.O_RDONLY, 0000)
	if err != nil {
		return errors.Wrap(err, "open configuration file")
	}
	defer source.Close()

	return yaml.NewDecoder(source).Decode(config)
}