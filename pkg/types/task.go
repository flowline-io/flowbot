package types // import "https://github.com/runabol/tork"

import (
	"slices"
	"time"

	"golang.org/x/exp/maps"
)

// TaskState State defines the list of states that a
// task can be in, at any given moment.
type TaskState string

const (
	TaskStatePending   TaskState = "PENDING"
	TaskStateRunning   TaskState = "RUNNING"
	TaskStateCancelled TaskState = "CANCELED"
	TaskStateStopped   TaskState = "STOPPED"
	TaskStateCompleted TaskState = "COMPLETED"
	TaskStateFailed    TaskState = "FAILED"
)

// Task is the basic unit of work that a Worker can handle.
type Task struct {
	ID          string            `json:"id,omitempty"`
	State       TaskState         `json:"state,omitempty"`
	CreatedAt   *time.Time        `json:"created_at,omitempty"`
	StartedAt   *time.Time        `json:"started_at,omitempty"`
	CompletedAt *time.Time        `json:"completed_at,omitempty"`
	FailedAt    *time.Time        `json:"failed_at,omitempty"`
	CMD         []string          `json:"cmd,omitempty"`
	Entrypoint  []string          `json:"entrypoint,omitempty"`
	Run         string            `json:"run,omitempty"`
	Image       string            `json:"image,omitempty"`
	Registry    *Registry         `json:"registry,omitempty"`
	Env         map[string]string `json:"env,omitempty"`
	Files       map[string]string `json:"files,omitempty"`
	Error       string            `json:"error,omitempty"`
	Pre         []*Task           `json:"pre,omitempty"`
	Post        []*Task           `json:"post,omitempty"`
	Mounts      []Mount           `json:"mounts,omitempty"`
	Networks    []string          `json:"networks,omitempty"`
	Retry       *TaskRetry        `json:"retry,omitempty"`
	Limits      *TaskLimits       `json:"limits,omitempty"`
	Timeout     string            `json:"timeout,omitempty"`
	Result      string            `json:"result,omitempty"`
	GPUs        string            `json:"gpus,omitempty"`
}

type TaskRetry struct {
	Limit    int `json:"limit,omitempty"`
	Attempts int `json:"attempts,omitempty"`
}

type TaskLimits struct {
	CPUs   string `json:"cpus,omitempty"`
	Memory string `json:"memory,omitempty"`
}

type Registry struct {
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
}

func (s TaskState) IsActive() bool {
	return s == TaskStatePending ||
		s == TaskStateRunning
}

func (t *Task) Clone() *Task {
	var retry *TaskRetry
	if t.Retry != nil {
		retry = t.Retry.Clone()
	}
	var limits *TaskLimits
	if t.Limits != nil {
		limits = t.Limits.Clone()
	}
	var registry *Registry
	if t.Registry != nil {
		registry = t.Registry.Clone()
	}
	return &Task{
		ID:          t.ID,
		State:       t.State,
		CreatedAt:   t.CreatedAt,
		StartedAt:   t.StartedAt,
		CompletedAt: t.CompletedAt,
		FailedAt:    t.FailedAt,
		CMD:         t.CMD,
		Entrypoint:  t.Entrypoint,
		Run:         t.Run,
		Image:       t.Image,
		Registry:    registry,
		Env:         maps.Clone(t.Env),
		Files:       maps.Clone(t.Files),
		Error:       t.Error,
		Pre:         CloneTasks(t.Pre),
		Post:        CloneTasks(t.Post),
		Mounts:      slices.Clone(t.Mounts),
		Networks:    t.Networks,
		Retry:       retry,
		Limits:      limits,
		Timeout:     t.Timeout,
		Result:      t.Result,
		GPUs:        t.GPUs,
	}
}

func CloneTasks(tasks []*Task) []*Task {
	c := make([]*Task, len(tasks))
	for i, t := range tasks {
		c[i] = t.Clone()
	}
	return c
}

func (r *TaskRetry) Clone() *TaskRetry {
	return &TaskRetry{
		Limit:    r.Limit,
		Attempts: r.Attempts,
	}
}

func (l *TaskLimits) Clone() *TaskLimits {
	return &TaskLimits{
		CPUs:   l.CPUs,
		Memory: l.Memory,
	}
}

func (r *Registry) Clone() *Registry {
	return &Registry{
		Username: r.Username,
		Password: r.Password,
	}
}

const (
	MountTypeVolume string = "volume"
	MountTypeBind   string = "bind"
	MountTypeTmpfs  string = "tmpfs"
)

type Mount struct {
	Type   string `json:"type,omitempty"`
	Source string `json:"source,omitempty"`
	Target string `json:"target,omitempty"`
}
