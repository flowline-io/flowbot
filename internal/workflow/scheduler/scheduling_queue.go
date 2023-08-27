package scheduler

import (
	"github.com/sysatom/flowbot/internal/types/meta"
	"github.com/sysatom/flowbot/pkg/logs"
	"github.com/sysatom/flowbot/pkg/utils/clock"
	"github.com/sysatom/flowbot/pkg/utils/heap"
	"github.com/sysatom/flowbot/pkg/utils/parallelizer"
	"golang.org/x/xerrors"
	"reflect"
	"sync"
	"time"
)

var (
	// AssignedStepAdd is the event when a step is added that causes steps with matching affinity terms
	// to be more schedulable.
	AssignedStepAdd = "AssignedStepAdd"
	// WorkerAdd is the event when a new worker is added to the cluster.
	WorkerAdd = "WorkerAdd"
	// AssignedStepUpdate is the event when a step is updated that causes steps with matching affinity
	// terms to be more schedulable.
	AssignedStepUpdate = "AssignedStepUpdate"
	// AssignedStepDelete is the event when a step is deleted that causes steps with matching affinity
	// terms to be more schedulable.
	AssignedStepDelete = "AssignedStepDelete"
	// UnschedulableTimeout is the event when a step stays in unschedulable for longer than timeout.
	UnschedulableTimeout = "UnschedulableTimeout"
	// WorkerStateChange is the event when worker label is changed.
	WorkerStateChange = "WorkerStateChange"
)

// PreEnqueueCheck is a function type. It's used to build functions that
// run against a Step and the caller can choose to enqueue or skip the Step
// by the checking result.
type PreEnqueueCheck func(step *meta.Step) bool

type SchedulingQueue interface {
	Add(step *meta.Step) error
	// Activate moves the given steps to activeQ iff they're in unschedulableSteps or backoffQ.
	// The passed-in steps are originally compiled from plugins that want to activate Steps,
	// by injecting the steps through a reserved CycleState struct (StepsToActivate).
	Activate(steps map[string]*meta.Step)
	// AddUnschedulableIfNotPresent adds an unschedulable step back to scheduling queue.
	// The stepSchedulingCycle represents the current scheduling cycle number which can be
	// returned by calling SchedulingCycle().
	AddUnschedulableIfNotPresent(step *meta.QueuedStepInfo, stepSchedulingCycle int64) error
	// SchedulingCycle returns the current number of scheduling cycle which is
	// cached by scheduling queue. Normally, incrementing this number whenever
	// a step is popped (e.g. called Pop()) is enough.
	SchedulingCycle() int64
	// Pop removes the head of the queue and returns it. It blocks if the
	// queue is empty and waits until a new item is added to the queue.
	Pop() (*meta.QueuedStepInfo, error)
	Update(oldStep, newStep *meta.Step) error
	Delete(step *meta.Step) error
	MoveAllToActiveOrBackoffQueue(event string, preCheck PreEnqueueCheck)
	AssignedStepAdded(step *meta.Step)
	AssignedStepUpdated(step *meta.Step)
	PendingSteps() []*meta.Step
	// Close closes the SchedulingQueue so that the goroutine which is
	// waiting to pop items can exit gracefully.
	Close()
	// Run starts the goroutines managing the queue.
	Run()
}

type priorityQueueOptions struct {
	clock                               clock.Clock
	stepInitialBackoffDuration          time.Duration
	stepMaxBackoffDuration              time.Duration
	stepMaxInUnschedulableStepsDuration time.Duration
	stepNominator                       meta.StepNominator
	clusterEventMap                     map[string]map[string]struct{}
}

const (
	// DefaultStepMaxInUnschedulableStepsDuration is the default value for the maximum
	// time a step can stay in unschedulableSteps. If a step stays in unschedulableSteps
	// for longer than this value, the step will be moved from unschedulableSteps to
	// backoffQ or activeQ. If this value is empty, the default value (5min)
	// will be used.
	DefaultStepMaxInUnschedulableStepsDuration = 5 * time.Minute

	queueClosed = "scheduling queue is closed"
)

const (
	// DefaultStepInitialBackoffDuration is the default value for the initial backoff duration
	// for unschedulable steps. To change the default stepInitialBackoffDurationSeconds used by the
	// scheduler, update the ComponentConfig value in defaults.go
	DefaultStepInitialBackoffDuration = 1 * time.Second
	// DefaultStepMaxBackoffDuration is the default value for the max backoff duration
	// for unschedulable steps. To change the default stepMaxBackoffDurationSeconds used by the
	// scheduler, update the ComponentConfig value in defaults.go
	DefaultStepMaxBackoffDuration = 10 * time.Second
)

var defaultPriorityQueueOptions = priorityQueueOptions{
	clock:                               clock.RealClock{},
	stepInitialBackoffDuration:          DefaultStepInitialBackoffDuration,
	stepMaxBackoffDuration:              DefaultStepMaxBackoffDuration,
	stepMaxInUnschedulableStepsDuration: DefaultStepMaxInUnschedulableStepsDuration,
}

// LessFunc is the function to sort step info
type LessFunc func(stepInfo1, stepInfo2 *meta.QueuedStepInfo) bool

// Option configures a PriorityQueue
type Option func(*priorityQueueOptions)

// NewSchedulingQueue initializes a priority queue as a new scheduling queue.
func NewSchedulingQueue(
	lessFn LessFunc,
	opts ...Option) SchedulingQueue {
	return NewPriorityQueue(lessFn, opts...)
}

func MakeNextStepFunc(queue SchedulingQueue) func() *meta.QueuedStepInfo {
	return func() *meta.QueuedStepInfo {
		stepInfo, err := queue.Pop()
		if err == nil {
			logs.Info.Printf("About to try and schedule step %s %s", stepInfo.Step.Name, stepInfo.Step.UID)
			return stepInfo
		}
		logs.Err.Printf("%s Error while retrieving next step from scheduling queue", err)
		return nil
	}
}

// newQueuedStepInfoForLookup builds a QueuedStepInfo object for a lookup in the queue.
func newQueuedStepInfoForLookup(step *meta.Step, plugins ...string) *meta.QueuedStepInfo {
	sets := make(map[string]struct{})
	for _, plugin := range plugins {
		sets[plugin] = struct{}{}
	}
	// Since this is only used for a lookup in the queue, we only need to set the Step,
	// and so we avoid creating a full StepInfo, which is expensive to instantiate frequently.
	return &meta.QueuedStepInfo{
		StepInfo:             &meta.StepInfo{Step: step},
		UnschedulablePlugins: sets,
	}
}

// PriorityQueue implements a scheduling queue.
// The head of PriorityQueue is the highest priority pending step. This structure
// has two sub queues and a additional data structure, namely: activeQ,
// backoffQ and unschedulableSteps.
//   - activeQ holds steps that are being considered for scheduling.
//   - backoffQ holds steps that moved from unschedulableSteps and will move to
//     activeQ when their backoff periods complete.
//   - unschedulableSteps holds steps that were already attempted for scheduling and
//     are currently determined to be unschedulable.
type PriorityQueue struct {
	// StepNominator abstracts the operations to maintain nominated Steps.
	meta.StepNominator

	stop  chan struct{}
	clock clock.Clock

	// step initial backoff duration.
	stepInitialBackoffDuration time.Duration
	// step maximum backoff duration.
	stepMaxBackoffDuration time.Duration
	// the maximum time a step can stay in the unschedulableSteps.
	stepMaxInUnschedulableStepsDuration time.Duration

	lock sync.RWMutex
	cond sync.Cond

	// activeQ is heap structure that scheduler actively looks at to find steps to
	// schedule. Head of heap is the highest priority step.
	activeQ *heap.Heap
	// stepBackoffQ is a heap ordered by backoff expiry. Steps which have completed backoff
	// are popped from this heap before the scheduler looks at activeQ
	stepBackoffQ *heap.Heap
	// unschedulableSteps holds steps that have been tried and determined unschedulable.
	unschedulableSteps *UnschedulableSteps
	// schedulingCycle represents sequence number of scheduling cycle and is incremented
	// when a step is popped.
	schedulingCycle int64
	// moveRequestCycle caches the sequence number of scheduling cycle when we
	// received a move request. Unschedulable steps in and before this scheduling
	// cycle will be put back to activeQueue if we were trying to schedule them
	// when we received move request.
	moveRequestCycle int64

	clusterEventMap map[string]map[string]struct{}

	// closed indicates that the queue is closed.
	// It is mainly used to let Pop() exit its control loop while waiting for an item.
	closed bool
}

// newQueuedStepInfo builds a QueuedStepInfo object.
func (p *PriorityQueue) newQueuedStepInfo(step *meta.Step, plugins ...string) *meta.QueuedStepInfo {
	sets := make(map[string]struct{})
	for _, plugin := range plugins {
		sets[plugin] = struct{}{}
	}
	now := p.clock.Now()
	return &meta.QueuedStepInfo{
		StepInfo:                meta.NewStepInfo(step),
		Timestamp:               now,
		InitialAttemptTimestamp: now,
		UnschedulablePlugins:    sets,
	}
}

func (p *PriorityQueue) activate(step *meta.Step) bool {
	// Verify if the step is present in activeQ.
	if _, exists, _ := p.activeQ.Get(newQueuedStepInfoForLookup(step)); exists {
		// No need to activate if it's already present in activeQ.
		return false
	}
	var pInfo *meta.QueuedStepInfo
	// Verify if the step is present in unschedulableSteps or backoffQ.
	if pInfo = p.unschedulableSteps.get(step); pInfo == nil {
		// If the step doesn't belong to unschedulableSteps or backoffQ, don't activate it.
		if obj, exists, _ := p.stepBackoffQ.Get(newQueuedStepInfoForLookup(step)); !exists {
			logs.Err.Printf("To-activate step does not exist in unschedulableSteps or backoffQ, %v", step)
			return false
		} else {
			pInfo = obj.(*meta.QueuedStepInfo)
		}
	}

	if pInfo == nil {
		// Redundant safe check. We shouldn't reach here.
		logs.Err.Printf("Internal error: cannot obtain pInfo")
		return false
	}

	if err := p.activeQ.Add(pInfo); err != nil {
		logs.Err.Printf("Error adding step to the scheduling queue, %v", step)
		return false
	}
	p.unschedulableSteps.delete(step)
	_ = p.stepBackoffQ.Delete(pInfo)
	//p.StepNominator.AddNominatedStep(pInfo.StepInfo, nil)
	return true
}

func (p *PriorityQueue) Add(step *meta.Step) error {
	p.lock.Lock()
	defer p.lock.Unlock()
	pInfo := p.newQueuedStepInfo(step)
	if err := p.activeQ.Add(pInfo); err != nil {
		logs.Err.Printf("Error adding step to the active queue, %v", step)
		return err
	}
	if p.unschedulableSteps.get(step) != nil {
		logs.Err.Printf("Error: step is already in the unschedulable queue, %v", step)
		p.unschedulableSteps.delete(step)
	}
	// Delete step from backoffQ if it is backing off
	if err := p.stepBackoffQ.Delete(pInfo); err == nil {
		logs.Err.Printf("Error: step is already in the stepBackoff queue, %v", step)
	}
	//p.StepNominator.AddNominatedStep(pInfo.StepInfo, nil)
	p.cond.Broadcast()

	return nil
}

func (p *PriorityQueue) Activate(steps map[string]*meta.Step) {
	p.lock.Lock()
	defer p.lock.Unlock()

	activated := false
	for _, step := range steps {
		if p.activate(step) {
			activated = true
		}
	}

	if activated {
		p.cond.Broadcast()
	}
}

func (p *PriorityQueue) AddUnschedulableIfNotPresent(pInfo *meta.QueuedStepInfo, stepSchedulingCycle int64) error {
	p.lock.Lock()
	defer p.lock.Unlock()

	step := pInfo.Step
	if p.unschedulableSteps.get(step) != nil {
		return xerrors.Errorf("step %v %s is already present in unschedulable queue", step.Name, step.UID)
	}

	if _, exists, _ := p.activeQ.Get(pInfo); exists {
		return xerrors.Errorf("step %v is already present in the active queue", step)
	}
	if _, exists, _ := p.stepBackoffQ.Get(pInfo); exists {
		return xerrors.Errorf("step %v is already present in the backoff queue", step)
	}

	// Refresh the timestamp since the step is re-added.
	pInfo.Timestamp = p.clock.Now()

	// If a move request has been received, move it to the BackoffQ, otherwise move
	// it to unschedulableSteps.
	if p.moveRequestCycle >= stepSchedulingCycle {
		if err := p.stepBackoffQ.Add(pInfo); err != nil {
			return xerrors.Errorf("error adding step %v to the backoff queue: %v", step.Name, err)
		}
	} else {
		p.unschedulableSteps.addOrUpdate(pInfo)
	}

	//p.StepNominator.AddNominatedStep(pInfo.StepInfo, nil)
	return nil
}

func (p *PriorityQueue) SchedulingCycle() int64 {
	p.lock.Lock()
	defer p.lock.Unlock()

	return p.schedulingCycle
}

func (p *PriorityQueue) Pop() (*meta.QueuedStepInfo, error) {
	p.lock.Lock()
	defer p.lock.Unlock()

	for p.activeQ.Len() == 0 {
		// When the queue is empty, invocation of Pop() is blocked until new item is enqueued.
		// When Close() is called, the p.closed is set and the condition is broadcast,
		// which causes this loop to continue and return from the Pop().
		if p.closed {
			return nil, xerrors.Errorf(queueClosed)
		}
		p.cond.Wait()
	}
	obj, err := p.activeQ.Pop()
	if err != nil {
		return nil, err
	}
	pInfo := obj.(*meta.QueuedStepInfo)
	pInfo.Attempts++
	p.schedulingCycle++
	return pInfo, nil
}

func updateStep(oldStepInfo interface{}, newStep *meta.Step) *meta.QueuedStepInfo {
	pInfo := oldStepInfo.(*meta.QueuedStepInfo)
	pInfo.Update(newStep)
	return pInfo
}

// isStepUpdated checks if the step is updated in a way that it may have become
// schedulable. It drops status of the step and compares it with old version.
func isStepUpdated(oldStep, newStep *meta.Step) bool {
	strip := func(step *meta.Step) *meta.Step {
		// DeepCopyObject
		p := step
		p.ResourceVersion = ""
		p.Generation = 0
		//p.Status = v1.StepStatus{}
		//p.ManagedFields = nil
		p.Finalizers = nil
		return p
	}
	return !reflect.DeepEqual(strip(oldStep), strip(newStep))
}

// isStepBackingoff returns true if a step is still waiting for its backoff timer.
// If this returns true, the step should not be re-tried.
func (p *PriorityQueue) isStepBackingoff(stepInfo *meta.QueuedStepInfo) bool {
	boTime := p.getBackoffTime(stepInfo)
	return boTime.After(p.clock.Now())
}

func (p *PriorityQueue) Update(oldStep, newStep *meta.Step) error {
	p.lock.Lock()
	defer p.lock.Unlock()

	if oldStep != nil {
		oldStepInfo := newQueuedStepInfoForLookup(oldStep)
		// If the step is already in the active queue, just update it there.
		if oldStepInfo, exists, _ := p.activeQ.Get(oldStepInfo); exists {
			pInfo := updateStep(oldStepInfo, newStep)
			//p.StepNominator.UpdateNominatedStep(oldStep, pInfo.StepInfo)
			return p.activeQ.Update(pInfo)
		}

		// If the step is in the backoff queue, update it there.
		if oldStepInfo, exists, _ := p.stepBackoffQ.Get(oldStepInfo); exists {
			pInfo := updateStep(oldStepInfo, newStep)
			//p.StepNominator.UpdateNominatedStep(oldStep, pInfo.StepInfo)
			return p.stepBackoffQ.Update(pInfo)
		}
	}

	// If the step is in the unschedulable queue, updating it may make it schedulable.
	if usStepInfo := p.unschedulableSteps.get(newStep); usStepInfo != nil {
		pInfo := updateStep(usStepInfo, newStep)
		//p.StepNominator.UpdateNominatedStep(oldStep, pInfo.StepInfo)
		if isStepUpdated(oldStep, newStep) {
			if p.isStepBackingoff(usStepInfo) {
				if err := p.stepBackoffQ.Add(pInfo); err != nil {
					return err
				}
				p.unschedulableSteps.delete(usStepInfo.Step)
			} else {
				if err := p.activeQ.Add(pInfo); err != nil {
					return err
				}
				p.unschedulableSteps.delete(usStepInfo.Step)
				p.cond.Broadcast()
			}
		} else {
			// Step update didn't make it schedulable, keep it in the unschedulable queue.
			p.unschedulableSteps.addOrUpdate(pInfo)
		}

		return nil
	}
	// If step is not in any of the queues, we put it in the active queue.
	pInfo := p.newQueuedStepInfo(newStep)
	if err := p.activeQ.Add(pInfo); err != nil {
		return err
	}
	//p.StepNominator.AddNominatedStep(pInfo.StepInfo, nil)
	p.cond.Broadcast()
	return nil
}

func (p *PriorityQueue) Delete(step *meta.Step) error {
	p.lock.Lock()
	defer p.lock.Unlock()

	//p.StepNominator.DeleteNominatedStepIfExists(step)
	if err := p.activeQ.Delete(newQueuedStepInfoForLookup(step)); err != nil {
		// The item was probably not found in the activeQ.
		_ = p.stepBackoffQ.Delete(newQueuedStepInfoForLookup(step))
		p.unschedulableSteps.delete(step)
	}
	return nil
}

func (p *PriorityQueue) MoveAllToActiveOrBackoffQueue(event string, preCheck PreEnqueueCheck) {
	p.lock.Lock()
	defer p.lock.Unlock()
	unschedulableSteps := make([]*meta.QueuedStepInfo, 0, len(p.unschedulableSteps.stepInfoMap))
	for _, pInfo := range p.unschedulableSteps.stepInfoMap {
		if preCheck == nil || preCheck(pInfo.Step) {
			unschedulableSteps = append(unschedulableSteps, pInfo)
		}
	}
	p.moveStepsToActiveOrBackoffQueue(unschedulableSteps, event)
}

// NOTE: this function assumes lock has been acquired in caller
func (p *PriorityQueue) moveStepsToActiveOrBackoffQueue(stepInfoList []*meta.QueuedStepInfo, event string) {
	activated := false
	for _, pInfo := range stepInfoList {
		// If the event doesn't help making the Step schedulable, continue.
		// Note: we don't run the check if pInfo.UnschedulablePlugins is nil, which denotes
		// either there is some abnormal error, or scheduling the step failed by plugins other than PreFilter, Filter and Permit.
		// In that case, it's desired to move it anyways.
		if len(pInfo.UnschedulablePlugins) != 0 && !p.stepMatchesEvent(pInfo, event) {
			continue
		}
		step := pInfo.Step
		if p.isStepBackingoff(pInfo) {
			if err := p.stepBackoffQ.Add(pInfo); err != nil {
				logs.Err.Printf(err.Error())
				logs.Err.Printf("Error adding step to the backoff queue, %v", step)
			} else {
				p.unschedulableSteps.delete(step)
			}
		} else {
			if err := p.activeQ.Add(pInfo); err != nil {
				logs.Err.Printf(err.Error())
				logs.Err.Printf("Error adding step to the scheduling queue, %v", step)
			} else {
				activated = true
				p.unschedulableSteps.delete(step)
			}
		}
	}
	p.moveRequestCycle = p.schedulingCycle
	if activated {
		p.cond.Broadcast()
	}
}

// Checks if the Step may become schedulable upon the event.
// This is achieved by looking up the global clusterEventMap registry.
func (p *PriorityQueue) stepMatchesEvent(stepInfo *meta.QueuedStepInfo, clusterEvent string) bool {
	if clusterEvent == "*" {
		return true
	}

	for evt, nameSet := range p.clusterEventMap {
		// Firstly verify if the two ClusterEvents match:
		// - either the registered event from plugin side is a WildCardEvent,
		// - or the two events have identical Resource fields and *compatible* ActionType.
		//   Note the ActionTypes don't need to be *identical*. We check if the ANDed value
		//   is zero or not. In this way, it's easy to tell Update&Delete is not compatible,
		//   but Update&All is.
		evtMatch := evt == "*" || evt == clusterEvent

		// Secondly verify the plugin name matches.
		// Note that if it doesn't match, we shouldn't continue to search.
		if evtMatch && intersect(nameSet, stepInfo.UnschedulablePlugins) {
			return true
		}
	}

	return false
}

// getUnschedulableStepsWithMatchingAffinityTerm returns unschedulable steps which have
// any affinity term that matches "step".
// NOTE: this function assumes lock has been acquired in caller.
func (p *PriorityQueue) getUnschedulableStepsWithMatchingAffinityTerm(_ *meta.Step) []*meta.QueuedStepInfo {
	var stepsToMove []*meta.QueuedStepInfo
	for _, pInfo := range p.unschedulableSteps.stepInfoMap {
		///for _, term := range pInfo.RequiredAffinityTerms {
		//	if term.Matches(step, nsLabels) {
		stepsToMove = append(stepsToMove, pInfo) // todo
		//		break
		//	}
		//}
	}
	return stepsToMove
}

func (p *PriorityQueue) AssignedStepAdded(step *meta.Step) {
	p.lock.Lock()
	p.moveStepsToActiveOrBackoffQueue(p.getUnschedulableStepsWithMatchingAffinityTerm(step), AssignedStepAdd)
	p.lock.Unlock()
}

func (p *PriorityQueue) AssignedStepUpdated(step *meta.Step) {
	p.lock.Lock()
	p.moveStepsToActiveOrBackoffQueue(p.getUnschedulableStepsWithMatchingAffinityTerm(step), AssignedStepUpdate)
	p.lock.Unlock()
}

func (p *PriorityQueue) PendingSteps() []*meta.Step {
	p.lock.RLock()
	defer p.lock.RUnlock()

	var result []*meta.Step
	for _, pInfo := range p.activeQ.List() {
		result = append(result, pInfo.(*meta.QueuedStepInfo).Step)
	}
	for _, pInfo := range p.stepBackoffQ.List() {
		result = append(result, pInfo.(*meta.QueuedStepInfo).Step)
	}
	for _, pInfo := range p.unschedulableSteps.stepInfoMap {
		result = append(result, pInfo.Step)
	}
	return result
}

func (p *PriorityQueue) Close() {
	p.lock.Lock()
	defer p.lock.Unlock()

	close(p.stop)
	p.closed = true
	p.cond.Broadcast()
}

// flushBackoffQCompleted Moves all steps from backoffQ which have completed backoff in to activeQ
func (p *PriorityQueue) flushBackoffQCompleted() {
	p.lock.Lock()
	defer p.lock.Unlock()
	activated := false
	for {
		rawStepInfo := p.stepBackoffQ.Peek()
		if rawStepInfo == nil {
			break
		}
		step := rawStepInfo.(*meta.QueuedStepInfo).Step
		boTime := p.getBackoffTime(rawStepInfo.(*meta.QueuedStepInfo))
		if boTime.After(p.clock.Now()) {
			break
		}
		_, err := p.stepBackoffQ.Pop()
		if err != nil {
			logs.Err.Printf(err.Error())
			logs.Err.Printf("Unable to pop step from backoff queue despite backoff completion, %v", step)
			break
		}
		_ = p.activeQ.Add(rawStepInfo)
		activated = true
	}

	if activated {
		p.cond.Broadcast()
	}
}

// flushUnschedulableStepsLeftover moves steps which stay in unschedulableSteps
// longer than stepMaxInUnschedulableStepsDuration to backoffQ or activeQ.
func (p *PriorityQueue) flushUnschedulableStepsLeftover() {
	p.lock.Lock()
	defer p.lock.Unlock()

	var stepsToMove []*meta.QueuedStepInfo
	currentTime := p.clock.Now()
	for _, pInfo := range p.unschedulableSteps.stepInfoMap {
		lastScheduleTime := pInfo.Timestamp
		if currentTime.Sub(lastScheduleTime) > p.stepMaxInUnschedulableStepsDuration {
			stepsToMove = append(stepsToMove, pInfo)
		}
	}

	if len(stepsToMove) > 0 {
		p.moveStepsToActiveOrBackoffQueue(stepsToMove, UnschedulableTimeout)
	}
}

// NewStepNominator creates a nominator as a backing of framework.StepNominator.
// A stepLister is passed in to check if the step exists
// before adding its nominatedWorker info.
func NewStepNominator() meta.StepNominator {
	return &nominator{
		nominatedSteps:        make(map[string][]*meta.StepInfo),
		nominatedStepToWorker: make(map[string]string),
	}
}

type nominator struct {
	// nominatedSteps is a map keyed by a worker name and the value is a list of
	// steps which are nominated to run on the worker. These are steps which can be in
	// the activeQ or unschedulableSteps.
	nominatedSteps map[string][]*meta.StepInfo
	// nominatedStepToWorker is map keyed by a Step UID to the worker name where it is
	// nominated.
	nominatedStepToWorker map[string]string

	sync.RWMutex
}

func (npm *nominator) AddNominatedStep(step *meta.StepInfo, nominatingInfo *meta.NominatingInfo) {
	npm.Lock()
	npm.add(step, nominatingInfo)
	npm.Unlock()
}

func (npm *nominator) delete(p *meta.Step) {
	nnn, ok := npm.nominatedStepToWorker[p.UID]
	if !ok {
		return
	}
	for i, np := range npm.nominatedSteps[nnn] {
		if np.Step.UID == p.UID {
			npm.nominatedSteps[nnn] = append(npm.nominatedSteps[nnn][:i], npm.nominatedSteps[nnn][i+1:]...)
			if len(npm.nominatedSteps[nnn]) == 0 {
				delete(npm.nominatedSteps, nnn)
			}
			break
		}
	}
	delete(npm.nominatedStepToWorker, p.UID)
}

func (npm *nominator) add(pi *meta.StepInfo, nominatingInfo *meta.NominatingInfo) {
	// Always delete the step if it already exists, to ensure we never store more than
	// one instance of the step.
	npm.delete(pi.Step)

	var workerName string
	if nominatingInfo.Mode() == meta.ModeOverride {
		workerName = nominatingInfo.NominatedWorkerName
	} else if nominatingInfo.Mode() == meta.ModeNoop {
		//if pi.Step.Status.NominatedWorkerName == "" {
		//	return
		//}
		//workerName = pi.Step.Status.NominatedWorkerName
		logs.Info.Printf("%+v", nominatingInfo.Mode())
	}

	//if npm.stepLister != nil {
	//	//If the step was removed or if it was already scheduled, don't nominate it.
	//	updatedStep, err := npm.stepLister.Get(pi.Step.UID)
	//	if err != nil {
	//		logs.Err.Printf("Step doesn't exist in stepLister, aborted adding it to the nominator %T %s", pi.Step, pi.Step.UID)
	//		return
	//	}
	//	if updatedStep.WorkerUID != "" {
	//		logs.Info.Printf("Step is already scheduled to a worker, aborted adding it to the nominator, %T %s", pi.Step, updatedStep.WorkerUID)
	//		return
	//	}
	//}

	npm.nominatedStepToWorker[pi.Step.UID] = workerName
	for _, npi := range npm.nominatedSteps[workerName] {
		if npi.Step.UID == pi.Step.UID {
			logs.Info.Printf("Step already exists in the nominator, %v", npi.Step)
			return
		}
	}
	npm.nominatedSteps[workerName] = append(npm.nominatedSteps[workerName], pi)
}

func (npm *nominator) DeleteNominatedStepIfExists(step *meta.Step) {
	npm.Lock()
	npm.delete(step)
	npm.Unlock()
}

func NominatedWorkerName(step *meta.Step) string {
	return step.WorkerUID
}

func (npm *nominator) UpdateNominatedStep(oldStep *meta.Step, newStepInfo *meta.StepInfo) {
	npm.Lock()
	defer npm.Unlock()
	// In some cases, an Update event with no "NominatedWorker" present is received right
	// after a worker("NominatedWorker") is reserved for this step in memory.
	// In this case, we need to keep reserving the NominatedWorker when updating the step pointer.
	var nominatingInfo *meta.NominatingInfo
	// We won't fall into below `if` block if the Update event represents:
	// (1) NominatedWorker info is added
	// (2) NominatedWorker info is updated
	// (3) NominatedWorker info is removed
	if NominatedWorkerName(oldStep) == "" && NominatedWorkerName(newStepInfo.Step) == "" {
		if nnn, ok := npm.nominatedStepToWorker[oldStep.UID]; ok {
			// This is the only case we should continue reserving the NominatedWorker
			nominatingInfo = &meta.NominatingInfo{
				NominatingMode:      meta.ModeOverride,
				NominatedWorkerName: nnn,
			}
		}
	}
	// We update irrespective of the nominatedWorkerName changed or not, to ensure
	// that step pointer is updated.
	npm.delete(oldStep)
	npm.add(newStepInfo, nominatingInfo)
}

func (npm *nominator) NominatedStepsForWorker(workerName string) []*meta.StepInfo {
	npm.RLock()
	defer npm.RUnlock()
	// Make a copy of the nominated Steps so the caller can mutate safely.
	steps := make([]*meta.StepInfo, len(npm.nominatedSteps[workerName]))
	for i := 0; i < len(steps); i++ {
		// DeepCopyObject
		steps[i] = npm.nominatedSteps[workerName][i]
	}
	return steps
}

func (p *PriorityQueue) Run() {
	go parallelizer.JitterUntil(p.flushBackoffQCompleted, 1.0*time.Second, 0.0, true, p.stop)
	go parallelizer.JitterUntil(p.flushUnschedulableStepsLeftover, 30*time.Second, 0.0, true, p.stop)
}

// NewPriorityQueue creates a PriorityQueue object.
func NewPriorityQueue(
	lessFn LessFunc,
	//informerFactory informers.SharedInformerFactory,
	opts ...Option,
) *PriorityQueue {
	options := defaultPriorityQueueOptions
	for _, opt := range opts {
		opt(&options)
	}

	comp := func(stepInfo1, stepInfo2 interface{}) bool {
		pInfo1 := stepInfo1.(*meta.QueuedStepInfo)
		pInfo2 := stepInfo2.(*meta.QueuedStepInfo)
		return lessFn(pInfo1, pInfo2)
	}

	//if options.stepNominator == nil {
	//	options.stepNominator = NewStepNominator(informerFactory.Core().V1().Steps().Lister())
	//}

	pq := &PriorityQueue{
		StepNominator:                       options.stepNominator,
		clock:                               options.clock,
		stop:                                make(chan struct{}),
		stepInitialBackoffDuration:          options.stepInitialBackoffDuration,
		stepMaxBackoffDuration:              options.stepMaxBackoffDuration,
		stepMaxInUnschedulableStepsDuration: options.stepMaxInUnschedulableStepsDuration,
		activeQ:                             heap.NewWithRecorder(stepInfoKeyFunc, comp),
		unschedulableSteps:                  newUnschedulableSteps(),
		moveRequestCycle:                    -1,
		clusterEventMap:                     options.clusterEventMap,
	}
	pq.cond.L = &pq.lock
	pq.stepBackoffQ = heap.NewWithRecorder(stepInfoKeyFunc, pq.stepsCompareBackoffCompleted)

	return pq
}

func (p *PriorityQueue) stepsCompareBackoffCompleted(stepInfo1, stepInfo2 interface{}) bool {
	pInfo1 := stepInfo1.(*meta.QueuedStepInfo)
	pInfo2 := stepInfo2.(*meta.QueuedStepInfo)
	bo1 := p.getBackoffTime(pInfo1)
	bo2 := p.getBackoffTime(pInfo2)
	return bo1.Before(bo2)
}

func (p *PriorityQueue) getBackoffTime(stepInfo *meta.QueuedStepInfo) time.Time {
	duration := p.calculateBackoffDuration(stepInfo)
	backoffTime := stepInfo.Timestamp.Add(duration)
	return backoffTime
}

func (p *PriorityQueue) calculateBackoffDuration(stepInfo *meta.QueuedStepInfo) time.Duration {
	duration := p.stepInitialBackoffDuration
	for i := 1; i < stepInfo.Attempts; i++ {
		// Use subtraction instead of addition or multiplication to avoid overflow.
		if duration > p.stepMaxBackoffDuration-duration {
			return p.stepMaxBackoffDuration
		}
		duration += duration
	}
	return duration
}

// UnschedulableSteps holds steps that cannot be scheduled. This data structure
// is used to implement unschedulableSteps.
type UnschedulableSteps struct {
	// stepInfoMap is a map key by a step's full-name and the value is a pointer to the QueuedStepInfo.
	stepInfoMap map[string]*meta.QueuedStepInfo
	keyFunc     func(step *meta.Step) string
}

// Add adds a step to the unschedulable stepInfoMap.
func (u *UnschedulableSteps) addOrUpdate(pInfo *meta.QueuedStepInfo) {
	stepID := u.keyFunc(pInfo.Step)
	u.stepInfoMap[stepID] = pInfo
}

// Delete deletes a step from the unschedulable stepInfoMap.
func (u *UnschedulableSteps) delete(step *meta.Step) {
	stepID := u.keyFunc(step)
	delete(u.stepInfoMap, stepID)
}

// Get returns the QueuedStepInfo if a step with the same key as the key of the given "step"
// is found in the map. It returns nil otherwise.
func (u *UnschedulableSteps) get(step *meta.Step) *meta.QueuedStepInfo {
	stepKey := u.keyFunc(step)
	if pInfo, exists := u.stepInfoMap[stepKey]; exists {
		return pInfo
	}
	return nil
}

func stepInfoKeyFunc(obj interface{}) (string, error) {
	return MetaNamespaceKeyFunc(obj.(*meta.QueuedStepInfo).Step)
}

type ExplicitKey string

func MetaNamespaceKeyFunc(obj interface{}) (string, error) {
	if key, ok := obj.(ExplicitKey); ok {
		return string(key), nil
	}
	//m, err := meta.Accessor(obj)
	//if err != nil {
	//	return "", xerrors.Errorf("object has no meta: %v", err)
	//}
	//return m.GetName(), nil
	return "", nil
}

func newUnschedulableSteps() *UnschedulableSteps {
	return &UnschedulableSteps{
		stepInfoMap: make(map[string]*meta.QueuedStepInfo),
		keyFunc:     GetStepFullName,
	}
}

func GetStepFullName(step *meta.Step) string {
	return step.UID
}

func intersect(x, y map[string]struct{}) bool {
	if len(x) > len(y) {
		x, y = y, x
	}
	for v := range x {
		if _, ok := y[v]; ok {
			return true
		}
	}
	return false
}
