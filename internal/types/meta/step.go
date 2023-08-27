package meta

import (
	"time"
)

type StepInfo struct {
	Step       *Step
	ParseError error
}

func NewStepInfo(step *Step) *StepInfo {
	return &StepInfo{Step: step}
}

func (pi *StepInfo) Update(step *Step) {
	if step != nil && pi.Step != nil && pi.Step.UID == step.UID {
		// StepInfo includes immutable information, and so it is safe to update the step in place if it is
		// the exact same step
		pi.Step = step
		return
	}

	pi.Step = step
}

// QueuedStepInfo is a Step wrapper with additional information related to
// the step's status in the scheduling queue, such as the timestamp when
// it's added to the queue.
type QueuedStepInfo struct {
	*StepInfo
	// The time step added to the scheduling queue.
	Timestamp time.Time
	// Number of schedule attempts before successfully scheduled.
	// It's used to record the # attempts metric.
	Attempts int
	// The time when the step is added to the queue for the first time. The step may be added
	// back to the queue multiple times before it's successfully scheduled.
	// It shouldn't be updated once initialized. It's used to record the e2e scheduling
	// latency for a step.
	InitialAttemptTimestamp time.Time
	// If a Step failed in a scheduling cycle, record the plugin names it failed by.
	UnschedulablePlugins map[string]struct{}
}

type NominatingMode int

const (
	ModeNoop NominatingMode = iota
	ModeOverride
)

type NominatingInfo struct {
	NominatedWorkerName string
	NominatingMode      NominatingMode
}

func (ni *NominatingInfo) Mode() NominatingMode {
	if ni == nil {
		return ModeNoop
	}
	return ni.NominatingMode
}

// StepNominator abstracts operations to maintain nominated Steps.
type StepNominator interface {
	// AddNominatedStep adds the given step to the nominator or
	// updates it if it already exists.
	AddNominatedStep(step *StepInfo, nominatingInfo *NominatingInfo)
	// DeleteNominatedStepIfExists deletes nominatedStep from internal cache. It's a no-op if it doesn't exist.
	DeleteNominatedStepIfExists(step *Step)
	// UpdateNominatedStep updates the <oldStep> with <newStep>.
	UpdateNominatedStep(oldStep *Step, newStepInfo *StepInfo)
	// NominatedStepsForWorker returns nominatedSteps on the given worker.
	NominatedStepsForWorker(workerName string) []*StepInfo
}
