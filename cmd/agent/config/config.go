package config

import (
	"context"
	"github.com/flowline-io/flowbot/pkg/utils"
	"github.com/flowline-io/flowbot/version"
	"github.com/fsnotify/fsnotify"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"go.uber.org/fx"
	"log"
	"os"
	"runtime"
)

var App Type

type Type struct {
	LogLevel string `json:"log_level" yaml:"log_level" mapstructure:"log_level"`
	ApiUrl   string `json:"api_url" yaml:"api_url" mapstructure:"api_url"`
	ApiToken string `json:"api_token" yaml:"api_token" mapstructure:"api_token"`
}

func Load(path ...string) {
	err := viper.BindPFlags(pflag.CommandLine)
	if err != nil {
		log.Fatalf("[config] Failed to bind flags: %v", err)
	}
	for _, p := range path {
		viper.AddConfigPath(p)
	}
	viper.SetConfigName("flowbot-agent")
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

func NewConfig(lc fx.Lifecycle) Type {
	executable, _ := os.Executable()

	curwd, err := os.Getwd()
	if err != nil {
		log.Fatalf("Couldn't get current working directory: %v", err)
	}

	log.Printf("version %s:%s:%s; pid %d; %d process(es)\n",
		version.Buildtags, executable, version.Buildstamp,
		os.Getpid(), runtime.GOMAXPROCS(runtime.NumCPU()))

	configFile := utils.ToAbsolutePath(curwd, "flowbot.yaml")
	log.Printf("Using config from '%s'\n", configFile)

	// Load config
	Load(".", curwd)

	// fx hooks
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			// Watch config
			viper.OnConfigChange(func(e fsnotify.Event) {
				log.Printf("Config file changed: %s\n", e.String())

				// Reload
				err := viper.Unmarshal(&App)
				if err != nil {
					log.Fatalf("[config] Failed to unmarshal config: %v", err)
				}
			})
			viper.WatchConfig()

			return nil
		},
		OnStop: func(ctx context.Context) error {
			return nil
		},
	})

	return App
}
