// Copyright (C) 2025 Crash Override, Inc.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the FSF, either version 3 of the License, or (at your option) any later version.
// See the LICENSE file in the root of this repository for full license text or
// visit: <https://www.gnu.org/licenses/gpl-3.0.html>.

// Package config provides the global configuration for Ocular.
package config

import (
	"errors"
	"io"
	"os"
	"strings"

	"github.com/spf13/viper"
	"go.uber.org/zap"
)

// Config is the structure for the global configuration file for Ocular.
// It is loaded from a config file at startup time, and values can be overridden
// by environment variables. The config file is expected to be in YAML format.
// Environment variables are expected to be prefixed with "OCULAR_", all capital
// and use underscores to separate nested keys. For example, the key
// "api.tls.enabled" can be overridden by the environment variable "OCULAR_API_TLS_ENABLED".
type Config struct {
	// Environment is the environment that Ocular is running in.
	Environment string `json:"environment" yaml:"environment"`

	// Logging is the configuration for the logger.
	Logging struct {
		// Level is the logging level.
		Level  string `json:"level"`
		Format string `json:"format"` // TODO(bryce): add format support
	} `json:"logging" yaml:"logging"`
}

// State is the global configuration state for Ocular.
var State Config

func Init() {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("/etc/ocular/")
	viper.AddConfigPath("$HOME/.ocular")
	viper.AddConfigPath(".")

	if configPath, exists := os.LookupEnv("OCULAR_CONFIG_PATH"); exists {
		// If the OCULAR_CONFIG_PATH environment variable is set, add it as a config path.
		viper.AddConfigPath(configPath)
	}

	// have to use something that will most likely not be a
	// key anywhere in the config file, so that we can
	// use it as a delimiter for the viper keys.
	// By default viper uses "." as a delimiter, which is not
	// suitable for Ocular, as when we have labels (or annotations)
	// that we want to parse into a map, it creates sub-maps for each ".", i.e.
	// "my.custom.label: value" becomes {"my": {"custom": {"label": "value"}}}
	delimiter := "%"
	viper.SetOptions(viper.KeyDelimiter(delimiter))

	viper.SetEnvPrefix("ocular")
	viper.SetEnvKeyReplacer(strings.NewReplacer(delimiter, "_"))
	viper.SetDefault("Environment", "production")

	viper.SetDefault("Logging%Level", "info")
	viper.SetDefault("Logging%Format", "json")

	err := viper.ReadInConfig()
	if err != nil {
		var configFileNotFoundError viper.ConfigFileNotFoundError
		if !errors.As(err, &configFileNotFoundError) {
			zap.L().Error("error reading config", zap.Error(err))
			return
		} else if err != nil {
			zap.L().Info("config file not found, using defaults")
		}
	}
	viper.AutomaticEnv()

	if err = viper.Unmarshal(&State); err != nil {
		zap.L().Error("error unmarshalling config", zap.Error(err))
	}
	InitEnv()
	InitLogger(State.Logging.Level, State.Logging.Format,
		zap.String("environment", State.Environment),
		zap.Any("build_metadata", map[string]string{
			"version":    Version,
			"build_time": BuildTime,
			"commit":     Commit,
		}))
}

func WriteConfig(w io.Writer) error {
	if err := viper.WriteConfigTo(w); err != nil {
		return err
	}
	return nil
}
