package types

import (
	"github.com/flowline-io/flowbot/internal/store/model"
	"github.com/looplab/fsm"
)

type JobInfo struct {
	Job *model.Job
	FSM *fsm.FSM
}

type StepInfo struct {
	Step *model.Step
	FSM  *fsm.FSM
}
