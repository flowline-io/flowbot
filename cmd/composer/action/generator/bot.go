package generator

import (
	"bytes"
	_ "embed"
	"errors"
	"fmt"
	"os"
	"text/template"

	"github.com/urfave/cli/v2"
)

//go:embed tmpl/main.tmpl
var mainTemple string

//go:embed tmpl/agent.tmpl
var agentTemple string

//go:embed tmpl/agent_func.tmpl
var agentFuncTemple string

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

func BotAction(c *cli.Context) error {
	// args
	bot := c.String("name")
	rule := c.StringSlice("rule") // input,group,agent,command,condition,cron,form
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
		panic("bots NotExist")
	}
	dir := fmt.Sprintf("%s/%s", BasePath, data.BotName)
	_, err = os.Stat(dir)
	if os.IsNotExist(err) {
		err = os.Mkdir(dir, os.ModePerm)
		if err != nil {
			return err
		}

		err = os.WriteFile(filePath(data.BotName, "bot.go"), parseTemplate(mainTemple, data), os.ModePerm)
		if err != nil {
			return err
		}
		if data.HasAgent {
			err = os.WriteFile(filePath(data.BotName, "agent.go"), parseTemplate(agentTemple, data), os.ModePerm)
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
			panic("dir exist, but bot.go file not exist")
		}
		if data.HasInput {
			// append
			appendFileContent(filePath(data.BotName, "bot.go"), parseTemplate(inputFuncTemple, data))
		}
		if !fileExist(data.BotName, "agent.go") {
			if data.HasAgent {
				err = os.WriteFile(filePath(data.BotName, "agent.go"), parseTemplate(agentTemple, data), os.ModePerm)
				if err != nil {
					return err
				}
				// append
				appendFileContent(filePath(data.BotName, "bot.go"), parseTemplate(agentFuncTemple, data))
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
	HasAgent    bool
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

func parseTemplate(text string, data interface{}) []byte {
	buf := bytes.NewBufferString("")
	t, err := template.New("tmpl").Parse(text)
	if err != nil {
		panic(err)
	}
	err = t.Execute(buf, data)
	if err != nil {
		panic(err)
	}
	return buf.Bytes()
}

func parseRule(rules []string, data *schema) {
	data.HasCommand = true
	for _, item := range rules {
		switch item {
		case "input":
			data.HasInput = true
		case "agent":
			data.HasAgent = true
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
		panic(err)
	}

	_, err = file.Write(content)
	if err != nil {
		panic(err)
	}

	_ = file.Close()
}
