package chatagent

import (
	"github.com/flowline-io/flowbot/pkg/agent/loopdetect"
	"github.com/flowline-io/flowbot/pkg/config"
)

func loopDetectConfigFromApp() loopdetect.Config {
	src := config.App.ChatAgent.LoopDetection
	out := loopdetect.DefaultConfig()
	if src.Enabled != nil {
		out.Enabled = *src.Enabled
	}
	if src.Window > 0 {
		out.Window = src.Window
	}
	if src.GenericWarn > 0 {
		out.GenericWarn = src.GenericWarn
	}
	if src.GenericCritical > 0 {
		out.GenericCritical = src.GenericCritical
	}
	if src.NoProgressCritical > 0 {
		out.NoProgressCritical = src.NoProgressCritical
	}
	if src.PingPongWarn > 0 {
		out.PingPongWarn = src.PingPongWarn
	}
	if src.PingPongCritical > 0 {
		out.PingPongCritical = src.PingPongCritical
	}
	if src.GlobalCircuitBreaker > 0 {
		out.GlobalCircuitBreaker = src.GlobalCircuitBreaker
	}
	if src.PostCompactionIdentical > 0 {
		out.PostCompactionIdentical = src.PostCompactionIdentical
	}
	if src.PostCompactionWatch > 0 {
		out.PostCompactionWatch = src.PostCompactionWatch
	}
	return out
}

func newLoopDetector() *loopdetect.Detector {
	return loopdetect.NewDetector(loopDetectConfigFromApp())
}
