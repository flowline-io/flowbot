package config

import (
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

var App configType

type configType struct {
	LogLevel string `json:"log_level" yaml:"log_level" mapstructure:"log_level"`
	ApiUrl   string `json:"api_url" yaml:"api_url" mapstructure:"api_url"`
	ApiToken string `json:"api_token" yaml:"api_token" mapstructure:"api_token"`
}

func Load(path ...string) {
	err := viper.BindPFlags(pflag.CommandLine)
	if err != nil {
		flog.Fatal("Failed to bind flags: %v", err)
	}
	for _, p := range path {
		viper.AddConfigPath(p)
	}
	viper.SetConfigName("flowbot-agent")
	viper.SetConfigType("yaml")
	err = viper.ReadInConfig()
	if err != nil {
		flog.Fatal("Failed to read config file: %v", err)
	}
	err = viper.Unmarshal(&App)
	if err != nil {
		flog.Fatal("Failed to unmarshal config: %v", err)
	}
}
