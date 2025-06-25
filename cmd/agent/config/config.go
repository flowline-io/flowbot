package config

import (
	"context"
	"log"
	"os"
	"runtime"

	"github.com/flowline-io/flowbot/pkg/utils"
	"github.com/flowline-io/flowbot/version"
	"github.com/fsnotify/fsnotify"
	"github.com/samber/lo"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"go.uber.org/fx"
)

var App Type

type Type struct {
	// logging level
	LogLevel string `json:"log_level" yaml:"log_level" mapstructure:"log_level"`
	// bot enabled
	BotsEnabled []string `json:"bots_enabled" yaml:"bots_enabled" mapstructure:"bots_enabled"`
	// API
	Api Api `json:"api" yaml:"api" mapstructure:"api"`
	// Updater
	Updater Updater `json:"updater" yaml:"updater" mapstructure:"updater"`
	// script engine
	ScriptEngine ScriptEngine `json:"script_engine" yaml:"script_engine" mapstructure:"script_engine"`
}

type Api struct {
	// server api url
	Url string `json:"url" yaml:"url" mapstructure:"url"`
	// api token
	Token string `json:"token" yaml:"token" mapstructure:"token"`
}

type Updater struct {
	// github token used for upgrade check and download binary
	GithubToken string `json:"github_token" yaml:"github_token" mapstructure:"github_token"`
}

type ScriptEngine struct {
	// Enabled
	Enabled bool `json:"enabled" yaml:"enabled" mapstructure:"enabled"`
	// script path
	ScriptPath string `json:"script_path" yaml:"script_path" mapstructure:"script_path"`
	// User ID
	UID string `json:"uid" yaml:"uid" mapstructure:"uid"`
	// Group ID
	GID string `json:"gid" yaml:"gid" mapstructure:"gid"`
	// skip watch dir
	SkipWatchDir []string `json:"skip_watch_dir" yaml:"skip_watch_dir" mapstructure:"skip_watch_dir"`
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
					log.Printf("[config] Failed to unmarshal config: %v", err)
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

func BotEnabled(id string) bool {
	return lo.Count(App.BotsEnabled, id) > 0
}
