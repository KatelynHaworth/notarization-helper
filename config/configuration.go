package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"

	"gopkg.in/yaml.v2"
)

type ConfigFormat uint8

const (
	ConfigFormatJSON ConfigFormat = iota
	ConfigFormatYAML
)

func (format ConfigFormat) decode(src io.Reader, dst any) error {
	switch format {
	case ConfigFormatJSON:
		return json.NewDecoder(src).Decode(dst)

	case ConfigFormatYAML:
		return yaml.NewDecoder(src).Decode(dst)

	default:
		return errors.New("unsupported config format")
	}
}

var ErrUnsupportedVersion = errors.New("unsupported configuration version")

type configuration interface {
	GetPackages() []Package
}

type Package struct {
	File     string `json:"file" yaml:"file"`
	BundleID string `json:"bundle_id" yaml:"bundle_id"`
	Staple   bool   `json:"staple" yaml:"staple"`
}

type configVersion struct {
	ConfigVersion int `json:"config_version" yaml:"config_version"`
}

func (ver *configVersion) getTargetType() (configuration, error) {
	switch ver.ConfigVersion {
	case 0, 1:
		return new(ConfigurationV1), nil

	case 2:
		return new(ConfigurationV2), nil

	default:
		return nil, ErrUnsupportedVersion
	}
}

func LoadConfigurationFromFile(srcFile string, format ConfigFormat) (*ConfigurationV2, error) {
	src, err := os.OpenFile(srcFile, os.O_RDONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("open configuration file: %w", err)
	}
	defer src.Close()

	var configVer configVersion
	if err = format.decode(src, &configVer); err != nil {
		return nil, fmt.Errorf("decode config version: %w", err)
	} else if _, err = src.Seek(0, io.SeekStart); err != nil {
		return nil, fmt.Errorf("seek to start of config: %w", err)
	}

	config, err := configVer.getTargetType()
	if err != nil {
		return nil, err
	} else if err = format.decode(src, config); err != nil {
		return nil, fmt.Errorf("decode configuration file: %w", err)
	}

	switch t := config.(type) {
	case *ConfigurationV1:
		return t.ToV2()

	case *ConfigurationV2:
		return t, nil

	default:
		return nil, ErrUnsupportedVersion
	}
}
