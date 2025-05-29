package workflow

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/types/protocol"
	"github.com/go-resty/resty/v2"
	"github.com/goccy/go-yaml"
	"github.com/urfave/cli/v3"
)

func ImportAction(ctx context.Context, c *cli.Command) error {
	// api url
	conffile := c.String("config")

	file, err := os.Open(filepath.Clean(conffile))
	if err != nil {
		flog.Panic(err.Error())
	}

	config := configType{}

	data, err := io.ReadAll(file)
	if err != nil {
		flog.Panic(err.Error())
	}
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		flog.Panic(err.Error())
	}

	// args
	token := c.String("token")
	if token == "" {
		return errors.New("token args error")
	}
	path := c.String("path")
	yamlFile, err := os.Open(filepath.Clean(path))
	if err != nil {
		flog.Panic(err.Error())
	}
	yamlData, err := io.ReadAll(yamlFile)
	if err != nil {
		flog.Panic(err.Error())
	}
	_, _ = fmt.Println(string(yamlData))

	// call api
	client := resty.New()
	client.SetBaseURL(config.Flowbot.Url)
	client.SetTimeout(time.Minute)
	client.SetAuthToken(token)

	resp, err := client.R().
		SetResult(&protocol.Response{}).
		SetBody(map[string]any{
			"lang":    "yaml",
			"code":    string(yamlData),
			"version": 1,
		}).
		Post("/service/workflow/workflow")
	if err != nil {
		flog.Panic(err.Error())
	}

	_, _ = fmt.Printf("%+v\n", resp)
	return nil
}

type configType struct {
	Flowbot struct {
		Url string `json:"url" yaml:"url"`
	} `json:"flowbot" yaml:"flowbot"`
}
