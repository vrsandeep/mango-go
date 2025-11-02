// This file defines the configuration structure for the application.
package config

import (
	// use Viper for loading the config.yml file.
	"strings"

	"github.com/spf13/viper"
)

// Config holds all configuration settings for the application.
// It maps directly to the structure of config.yml.
type Config struct {
	Port         int `mapstructure:"port"`
	ScanInterval int `mapstructure:"scan_interval"`
	Database     struct {
		Path string `mapstructure:"path"`
	} `mapstructure:"database"`
	Library struct {
		Path string `mapstructure:"path"`
	} `mapstructure:"library"`
	Plugins struct {
		Path string `mapstructure:"path"`
	} `mapstructure:"plugins"`
}

// Load reads configuration from a file named "config.yml" in the
// current directory and unmarshals it into a Config struct.
func Load() (*Config, error) {
	viper.SetConfigName("config") // name of config file (without extension)
	viper.SetConfigType("yml")    // or "yaml"
	viper.AddConfigPath(".")      // looking for config in the current directory

	// --- Environment Variable Overrides ---
	// This tells Viper to look for environment variables with a "MANGO_" prefix.
	// e.g., MANGO_DATABASE_PATH will override the `database.path` key.
	viper.SetEnvPrefix("MANGO")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	// Set default values
	viper.SetDefault("port", 8080)
	viper.SetDefault("scan_interval", 60)
	viper.SetDefault("database.path", "./mango.db")
	viper.SetDefault("library.path", "./manga")
	viper.SetDefault("plugins.path", "../mango-go-plugins")

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// Config file not found; ignore error and use defaults
		} else {
			// Config file was found but another error was produced
			return nil, err
		}
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, err
	}

	return &config, nil
}
