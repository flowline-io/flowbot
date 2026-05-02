package flog

var SlackLogger = &slackLogger{}

type slackLogger struct{}

func (s *slackLogger) Output(i int, s2 string) error {
	evt := l.Debug()
	if mustCaller() {
		evt = evt.Caller(i)
	}
	evt.Msg(s2)
	return nil
}
