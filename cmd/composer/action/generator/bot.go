package generator

import (
	"bytes"
	"context"
	_ "embed"
	"errors"
	"fmt"
	"os"
	"text/template"

	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/urfave/cli/v3"
)

//go:embed tmpl/main.tmpl
var mainTemple string

//go:embed tmpl/collect.tmpl
var collectTemple string

//go:embed tmpl/collect_func.tmpl
var collectFuncTemple string

//go:embed tmpl/command.tmpl
var commandTemple string

//go:embed tmpl/cron.tmpl
var cronTemple string

//go:embed tmpl/cron_func.tmpl
var cronFuncTemple string

//go:embed tmpl/form.tmpl
var formTemple string

//go:embed tmpl/form_func.tmpl
var formFuncTemple string

//go:embed tmpl/instruct.tmpl
var instructTemple string

//go:embed tmpl/instruct_func.tmpl
var instructFuncTemple string

//go:embed tmpl/input_func.tmpl
var inputFuncTemple string

const BasePath = "./internal/bots"

func BotAction(ctx context.Context, c *cli.Command) error {
	// args
	bot := c.String("name")
	rule := c.StringSlice("rule") // input,group,collect,command,condition,cron,form
	if bot == "" {
		return errors.New("bot name args error")
	}

	// schema
	data := schema{
		BotName: bot,
	}
	parseRule(rule, &data)

	// check dir
	_, err := os.Stat(BasePath)
	if os.IsNotExist(err) {
		flog.Panic("bots NotExist")
	}
	dir := fmt.Sprintf("%s/%s", BasePath, data.BotName)
	_, err = os.Stat(dir)
	if os.IsNotExist(err) {
		err = os.Mkdir(dir, 0700)
		if err != nil {
			return err
		}

		err = os.WriteFile(filePath(data.BotName, "bot.go"), parseTemplate(mainTemple, data), os.ModePerm)
		if err != nil {
			return err
		}
		if data.HasCollect {
			err = os.WriteFile(filePath(data.BotName, "collect.go"), parseTemplate(collectTemple, data), os.ModePerm)
			if err != nil {
				return err
			}
		}
		if data.HasCommand {
			err = os.WriteFile(filePath(data.BotName, "command.go"), parseTemplate(commandTemple, data), os.ModePerm)
			if err != nil {
				return err
			}
		}
		if data.HasCron {
			err = os.WriteFile(filePath(data.BotName, "cron.go"), parseTemplate(cronTemple, data), os.ModePerm)
			if err != nil {
				return err
			}
		}
		if data.HasForm {
			err = os.WriteFile(filePath(data.BotName, "form.go"), parseTemplate(formTemple, data), os.ModePerm)
			if err != nil {
				return err
			}
		}
		if data.HasInstruct {
			err = os.WriteFile(filePath(data.BotName, "instruct.go"), parseTemplate(instructTemple, data), os.ModePerm)
			if err != nil {
				return err
			}
		}
	} else {
		if !fileExist(data.BotName, "bot.go") {
			flog.Panic("dir exist, but bot.go file not exist")
		}
		if data.HasInput {
			// append
			appendFileContent(filePath(data.BotName, "bot.go"), parseTemplate(inputFuncTemple, data))
		}
		if !fileExist(data.BotName, "collect.go") {
			if data.HasCollect {
				err = os.WriteFile(filePath(data.BotName, "collect.go"), parseTemplate(collectTemple, data), os.ModePerm)
				if err != nil {
					return err
				}
				// append
				appendFileContent(filePath(data.BotName, "bot.go"), parseTemplate(collectFuncTemple, data))
			}
		}
		if !fileExist(data.BotName, "cron.go") {
			if data.HasCron {
				err = os.WriteFile(filePath(data.BotName, "cron.go"), parseTemplate(cronTemple, data), os.ModePerm)
				if err != nil {
					return err
				}
				// append
				appendFileContent(filePath(data.BotName, "bot.go"), parseTemplate(cronFuncTemple, data))
			}
		}
		if !fileExist(data.BotName, "form.go") {
			if data.HasForm {
				err = os.WriteFile(filePath(data.BotName, "form.go"), parseTemplate(formTemple, data), os.ModePerm)
				if err != nil {
					return err
				}
				// append
				appendFileContent(filePath(data.BotName, "bot.go"), parseTemplate(formFuncTemple, data))
			}
		}
		if !fileExist(data.BotName, "instruct.go") {
			if data.HasInstruct {
				err = os.WriteFile(filePath(data.BotName, "instruct.go"), parseTemplate(instructTemple, data), os.ModePerm)
				if err != nil {
					return err
				}
				// append
				appendFileContent(filePath(data.BotName, "bot.go"), parseTemplate(instructFuncTemple, data))
			}
		}
	}

	_, _ = fmt.Println("done")
	return nil
}

type schema struct {
	BotName     string
	HasInput    bool
	HasCommand  bool
	HasCollect  bool
	HasCron     bool
	HasForm     bool
	HasInstruct bool
}

func filePath(botName, fileName string) string {
	return fmt.Sprintf("%s/%s/%s", BasePath, botName, fileName)
}

func fileExist(botName, fileName string) bool {
	_, err := os.Stat(filePath(botName, fileName))
	return !os.IsNotExist(err)
}

func parseTemplate(text string, data any) []byte {
	buf := bytes.NewBufferString("")
	t, err := template.New("tmpl").Parse(text)
	if err != nil {
		flog.Panic("%s", err.Error())
	}
	err = t.Execute(buf, data)
	if err != nil {
		flog.Panic("%s", err.Error())
	}
	return buf.Bytes()
}

func parseRule(rules []string, data *schema) {
	data.HasCommand = true
	for _, item := range rules {
		switch item {
		case "input":
			data.HasInput = true
		case "collect":
			data.HasCollect = true
		case "cron":
			data.HasCron = true
		case "form":
			data.HasForm = true
		case "instruct":
			data.HasInstruct = true
		}
	}
}

func appendFileContent(filePath string, content []byte) {
	file, err := os.OpenFile(filePath, os.O_APPEND|os.O_WRONLY, os.ModePerm)
	if err != nil {
		flog.Panic("%s", err.Error())
	}

	_, err = file.Write(content)
	if err != nil {
		flog.Panic("%s", err.Error())
	}

	_ = file.Close()
}
