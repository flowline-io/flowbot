package chatagent

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/flc1125/go-cron/v4"
	"go.opentelemetry.io/otel/attribute"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/ent/gen"
	"github.com/flowline-io/flowbot/internal/store/ent/schema"
	"github.com/flowline-io/flowbot/pkg/cronutil"
	"github.com/flowline-io/flowbot/pkg/flog"
	fbtrace "github.com/flowline-io/flowbot/pkg/trace"
)

// TaskScheduler registers chat scheduled tasks with cron and one-shot timers.
type TaskScheduler struct {
	mu         sync.Mutex
	cron       *cron.Cron
	cronIDs    map[string]cron.EntryID
	onceTimers map[string]*time.Timer
	taskLocks  map[string]*sync.Mutex
	now        func() time.Time
	started    bool
	shutdown   atomic.Bool
}

// NewTaskScheduler creates a scheduler that uses wall-clock time.
func NewTaskScheduler() *TaskScheduler {
	return NewTaskSchedulerWithClock(time.Now)
}

// NewTaskSchedulerWithClock creates a scheduler with an injectable clock for tests.
func NewTaskSchedulerWithClock(now func() time.Time) *TaskScheduler {
	if now == nil {
		now = time.Now
	}
	return &TaskScheduler{
		cronIDs:    make(map[string]cron.EntryID),
		onceTimers: make(map[string]*time.Timer),
		taskLocks:  make(map[string]*sync.Mutex),
		now:        now,
	}
}

var (
	globalScheduler   *TaskScheduler
	globalSchedulerMu sync.RWMutex
)

// SetGlobalScheduler wires the process-wide scheduler used by schedule tools.
func SetGlobalScheduler(s *TaskScheduler) {
	globalSchedulerMu.Lock()
	globalScheduler = s
	globalSchedulerMu.Unlock()
}

// GlobalScheduler returns the process-wide scheduler, if started.
func GlobalScheduler() *TaskScheduler {
	globalSchedulerMu.RLock()
	defer globalSchedulerMu.RUnlock()
	return globalScheduler
}

// Start loads active tasks from the database and begins scheduling.
func (s *TaskScheduler) Start(_ context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.started {
		return nil
	}
	s.shutdown.Store(false)
	s.cron = cron.New(
		cron.WithSeconds(),
		cron.WithParser(cron.NewParser(cron.Minute|cron.Hour|cron.Dom|cron.Month|cron.Dow|cron.Descriptor)),
	)
	s.cron.Start()
	s.started = true

	if store.Database == nil {
		return nil
	}
	if err := store.Database.FailStaleChatScheduledTaskRuns(context.Background()); err != nil {
		flog.Warn("[chat-agent] scheduler stale run cleanup: %v", err)
	}
	tasks, err := store.Database.ListChatScheduledTasks(context.Background(), store.ListChatScheduledTasksOptions{
		States: []string{string(schema.ChatScheduledTaskStateActive)},
	})
	if err != nil {
		return err
	}
	for _, task := range tasks {
		if err := s.registerLocked(task); err != nil {
			flog.Warn("[chat-agent] scheduler skip task=%s: %v", task.Flag, err)
		}
	}
	flog.Info("[chat-agent] scheduler started tasks=%d", len(tasks))
	return nil
}

// Stop shuts down cron jobs and one-shot timers.
func (s *TaskScheduler) Stop(_ context.Context) error {
	s.shutdown.Store(true)
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.started {
		return nil
	}
	ctx := s.cron.Stop()
	<-ctx.Done()
	for id, timer := range s.onceTimers {
		timer.Stop()
		delete(s.onceTimers, id)
	}
	s.started = false
	flog.Info("[chat-agent] scheduler stopped")
	return nil
}

// RegisterTask schedules one active task.
func (s *TaskScheduler) RegisterTask(task *gen.ChatScheduledTask) error {
	if task == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.registerLocked(task)
}

// UpdateTask re-registers a task after schedule changes under one lock.
func (s *TaskScheduler) UpdateTask(task *gen.ChatScheduledTask) error {
	if task == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.unregisterLocked(task.Flag)
	return s.registerLocked(task)
}

// UnregisterTask removes scheduling hooks for one task id.
func (s *TaskScheduler) UnregisterTask(taskID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.unregisterLocked(taskID)
}

func (s *TaskScheduler) unregisterLocked(taskID string) {
	if entryID, ok := s.cronIDs[taskID]; ok && s.cron != nil {
		s.cron.Remove(entryID)
		delete(s.cronIDs, taskID)
	}
	if timer, ok := s.onceTimers[taskID]; ok {
		timer.Stop()
		delete(s.onceTimers, taskID)
	}
}

func (s *TaskScheduler) registerLocked(task *gen.ChatScheduledTask) error {
	if task.State != string(schema.ChatScheduledTaskStateActive) {
		return nil
	}
	s.ensureTaskLock(task.Flag)
	switch task.ScheduleKind {
	case string(schema.ChatScheduledTaskKindCron):
		return s.registerCronLocked(task)
	case string(schema.ChatScheduledTaskKindOnce):
		return s.registerOnceLocked(task)
	default:
		return errInvalidScheduleKind
	}
}

func (s *TaskScheduler) registerCronLocked(task *gen.ChatScheduledTask) error {
	if s.cron == nil {
		return nil
	}
	taskID := task.Flag
	entryID, err := s.cron.AddFunc(task.Cron, func(_ context.Context) error {
		s.runTask(taskID)
		return nil
	})
	if err != nil {
		return err
	}
	s.cronIDs[taskID] = entryID
	if store.Database != nil {
		next, nerr := cronutil.NextRun(task.Cron, s.now())
		if nerr != nil {
			flog.Warn("[chat-agent] scheduler next_run_at task=%s: %v", taskID, nerr)
		} else if err := store.Database.UpdateChatScheduledTask(context.Background(), taskID, store.UpdateChatScheduledTaskParams{
			NextRunAt: &next,
		}); err != nil {
			flog.Warn("[chat-agent] scheduler next_run_at update task=%s: %v", taskID, err)
		}
	}
	return nil
}

func (s *TaskScheduler) registerOnceLocked(task *gen.ChatScheduledTask) error {
	if task.RunAt == nil {
		return errRunAtRequired
	}
	now := s.now().UTC()
	runAt := task.RunAt.UTC()
	if runAt.After(now) {
		delay := runAt.Sub(now)
		taskID := task.Flag
		timer := time.AfterFunc(delay, func() {
			if s.shutdown.Load() {
				return
			}
			s.runTask(taskID)
		})
		s.onceTimers[task.Flag] = timer
		if store.Database != nil {
			next := runAt
			if err := store.Database.UpdateChatScheduledTask(context.Background(), taskID, store.UpdateChatScheduledTaskParams{
				NextRunAt: &next,
			}); err != nil {
				flog.Warn("[chat-agent] scheduler next_run_at update task=%s: %v", taskID, err)
			}
		}
		return nil
	}
	if now.Sub(runAt) <= onceGraceWindow {
		if s.shutdown.Load() {
			return nil
		}
		taskID := task.Flag
		go func() {
			if s.shutdown.Load() {
				return
			}
			s.runTask(taskID)
		}()
		return nil
	}
	if store.Database != nil {
		missed := string(schema.ChatScheduledTaskStateMissed)
		if err := store.Database.UpdateChatScheduledTask(context.Background(), task.Flag, store.UpdateChatScheduledTaskParams{
			State: &missed,
		}); err != nil {
			flog.Warn("[chat-agent] scheduler mark missed task=%s: %v", task.Flag, err)
		}
	}
	return nil
}

func (s *TaskScheduler) runTask(taskID string) {
	if s.shutdown.Load() {
		return
	}
	lock := s.taskLock(taskID)
	if !lock.TryLock() {
		flog.Info("[chat-agent] scheduled task skipped (overlap) task=%s", taskID)
		return
	}
	defer lock.Unlock()

	if store.Database == nil {
		return
	}
	task, err := store.Database.GetChatScheduledTask(context.Background(), taskID)
	if err != nil {
		flog.Warn("[chat-agent] scheduled task load failed task=%s: %v", taskID, err)
		return
	}
	if task.State != string(schema.ChatScheduledTaskStateActive) {
		return
	}

	runCtx, span := fbtrace.StartSpan(context.Background(), "chatagent.scheduled_task",
		attribute.String("task.id", task.Flag),
		attribute.String("task.name", task.Name),
		attribute.String("session.id", task.SourceSessionID),
	)
	defer span.End()
	executeScheduledTask(runCtx, task)
}

func (s *TaskScheduler) ensureTaskLock(taskID string) {
	if _, ok := s.taskLocks[taskID]; !ok {
		s.taskLocks[taskID] = &sync.Mutex{}
	}
}

func (s *TaskScheduler) taskLock(taskID string) *sync.Mutex {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ensureTaskLock(taskID)
	return s.taskLocks[taskID]
}

// syncTaskWithScheduler registers or unregisters one task based on its persisted state.
func syncTaskWithScheduler(task *gen.ChatScheduledTask) error {
	sched := GlobalScheduler()
	if sched == nil || task == nil {
		return nil
	}
	switch task.State {
	case string(schema.ChatScheduledTaskStateActive):
		return sched.UpdateTask(task)
	default:
		sched.UnregisterTask(task.Flag)
		return nil
	}
}
