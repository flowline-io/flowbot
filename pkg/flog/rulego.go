package flog

import (
	"fmt"
)

var RulegoLogger = &rulegoLogger{}

type rulegoLogger struct{}

func (a *rulegoLogger) Printf(format string, v ...interface{}) {
	l.Info().Caller(2).Msg(fmt.Sprintf(format, v...))
}
