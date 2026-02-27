// Package config loads and manages the flowbot-app.yaml configuration
// for the Admin PWA application.
package config

import (
	"context"
	"log"
	"os"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"go.uber.org/fx"
)

// App is the global configuration instance loaded from flowbot-app.yaml.
var App Type

// Type is the top-level configuration structure for the Admin PWA server.
type Type struct {
	// HTTP listen address for the PWA server
	Listen string `json:"listen" yaml:"listen" mapstructure:"listen"`
	// Main server API configuration
	API API `json:"api" yaml:"api" mapstructure:"api"`
	// Logging settings
	Log Log `json:"log" yaml:"log" mapstructure:"log"`
}

// API holds the main server API connection configuration.
type API struct {
	// Main server URL (e.g., "http://127.0.0.1:8060")
	URL string `json:"url" yaml:"url" mapstructure:"url"`
	// API route prefix (e.g., "/service/admin")
	Prefix string `json:"prefix" yaml:"prefix" mapstructure:"prefix"`
}

// Log holds logging configuration.
type Log struct {
	// Logging level (debug, info, warn, error)
	Level string `json:"level" yaml:"level" mapstructure:"level"`
}

// Load reads the flowbot-app.yaml configuration file from the given paths
// and unmarshals it into the global App variable.
func Load(path ...string) {
	err := viper.BindPFlags(pflag.CommandLine)
	if err != nil {
		log.Fatalf("[config] Failed to bind flags: %v", err)
	}
	for _, p := range path {
		viper.AddConfigPath(p)
	}
	viper.SetConfigName("flowbot-app")
	viper.SetConfigType("yaml")
	err = viper.ReadInConfig()
	if err != nil {
		log.Fatalf("[config] Failed to read config file: %v", err)
	}
	err = viper.Unmarshal(&App)
	if err != nil {
		log.Fatalf("[config] Failed to unmarshal config: %v", err)
	}
}

// NewConfig is an fx-compatible constructor that loads the configuration
// and sets up hot-reload via file watching.
func NewConfig(lc fx.Lifecycle) Type {
	curwd, err := os.Getwd()
	if err != nil {
		log.Fatalf("Failed to get working directory: %v", err)
	}

	Load(".", curwd)

	lc.Append(fx.Hook{
		OnStart: func(_ context.Context) error {
			viper.OnConfigChange(func(e fsnotify.Event) {
				log.Printf("Config file changed: %s", e.String())
				if err := viper.Unmarshal(&App); err != nil {
					log.Printf("[config] Failed to unmarshal config: %v", err)
				}
			})
			viper.WatchConfig()
			return nil
		},
		OnStop: func(_ context.Context) error {
			return nil
		},
	})

	return App
}
