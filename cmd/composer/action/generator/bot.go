package generator

import (
	"bytes"
	_ "embed"
	"errors"
	"fmt"
	"github.com/urfave/cli/v2"
	"os"
	"text/template"
)

//go:embed tmpl/main.tmpl
var mainTemple string

//go:embed tmpl/agent.tmpl
var agentTemple string

//go:embed tmpl/agent_func.tmpl
var agentFuncTemple string

//go:embed tmpl/command.tmpl
var commandTemple string

//go:embed tmpl/condition.tmpl
var conditionTemple string

//go:embed tmpl/condition_func.tmpl
var conditionFuncTemple string

//go:embed tmpl/cron.tmpl
var cronTemple string

//go:embed tmpl/cron_func.tmpl
var cronFuncTemple string

//go:embed tmpl/form.tmpl
var formTemple string

//go:embed tmpl/form_func.tmpl
var formFuncTemple string

//go:embed tmpl/action.tmpl
var actionTemple string

//go:embed tmpl/action_func.tmpl
var actionFuncTemple string

//go:embed tmpl/session.tmpl
var sessionTemple string

//go:embed tmpl/session_func.tmpl
var sessionFuncTemple string

//go:embed tmpl/instruct.tmpl
var instructTemple string

//go:embed tmpl/instruct_func.tmpl
var instructFuncTemple string

//go:embed tmpl/group.tmpl
var groupTemple string

//go:embed tmpl/group_func.tmpl
var groupFuncTemple string

//go:embed tmpl/input_func.tmpl
var inputFuncTemple string

const BasePath = "./server/extra/bots"

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
		if data.HasCondition {
			err = os.WriteFile(filePath(data.BotName, "condition.go"), parseTemplate(conditionTemple, data), os.ModePerm)
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
		if data.HasAction {
			err = os.WriteFile(filePath(data.BotName, "action.go"), parseTemplate(actionTemple, data), os.ModePerm)
			if err != nil {
				return err
			}
		}
		if data.HasSession {
			err = os.WriteFile(filePath(data.BotName, "session.go"), parseTemplate(sessionTemple, data), os.ModePerm)
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
		if data.HasGroup {
			err = os.WriteFile(filePath(data.BotName, "group.go"), parseTemplate(groupTemple, data), os.ModePerm)
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
		if !fileExist(data.BotName, "group.go") {
			if data.HasGroup {
				err = os.WriteFile(filePath(data.BotName, "group.go"), parseTemplate(groupTemple, data), os.ModePerm)
				if err != nil {
					return err
				}
				// append
				appendFileContent(filePath(data.BotName, "bot.go"), parseTemplate(groupFuncTemple, data))
			}
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
		if !fileExist(data.BotName, "condition.go") {
			if data.HasCondition {
				err = os.WriteFile(filePath(data.BotName, "condition.go"), parseTemplate(conditionTemple, data), os.ModePerm)
				if err != nil {
					return err
				}
				// append
				appendFileContent(filePath(data.BotName, "bot.go"), parseTemplate(conditionFuncTemple, data))
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
		if !fileExist(data.BotName, "action.go") {
			if data.HasAction {
				err = os.WriteFile(filePath(data.BotName, "action.go"), parseTemplate(actionTemple, data), os.ModePerm)
				if err != nil {
					return err
				}
				// append
				appendFileContent(filePath(data.BotName, "bot.go"), parseTemplate(actionFuncTemple, data))
			}
		}
		if !fileExist(data.BotName, "session.go") {
			if data.HasSession {
				err = os.WriteFile(filePath(data.BotName, "session.go"), parseTemplate(sessionTemple, data), os.ModePerm)
				if err != nil {
					return err
				}
				// append
				appendFileContent(filePath(data.BotName, "bot.go"), parseTemplate(sessionFuncTemple, data))
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

	fmt.Println("done")
	return nil
}

type schema struct {
	BotName      string
	HasInput     bool
	HasGroup     bool
	HasCommand   bool
	HasAgent     bool
	HasCondition bool
	HasCron      bool
	HasForm      bool
	HasAction    bool
	HasSession   bool
	HasInstruct  bool
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
		case "group":
			data.HasGroup = true
		case "agent":
			data.HasAgent = true
		case "condition":
			data.HasCondition = true
		case "cron":
			data.HasCron = true
		case "form":
			data.HasForm = true
		case "action":
			data.HasAction = true
		case "session":
			data.HasSession = true
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
