package rules

import (
	"github.com/flowline-io/flowbot/pkg/utils"
	"github.com/rulego/rulego"
)

func InitEngine() error {
	conf, err := NewConfig()
	if err != nil {
		return err
	}

	_, err = rulego.New("test", utils.StringToBytes(testRuleFile), rulego.WithConfig(conf))
	if err != nil {
		return err
	}

	return nil
}
