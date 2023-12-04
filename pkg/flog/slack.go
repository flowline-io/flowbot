package flog

var SlackLogger = &slackLogger{}

type slackLogger struct{}

func (s *slackLogger) Output(i int, s2 string) error {
	l.Debug().Caller(i).Msg(s2)
	return nil
}
