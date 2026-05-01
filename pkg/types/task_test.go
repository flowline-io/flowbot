package types

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestTaskState_IsActive(t *testing.T) {
	assert.True(t, TaskStatePending.IsActive())
	assert.True(t, TaskStateRunning.IsActive())
	assert.False(t, TaskStateCancelled.IsActive())
	assert.False(t, TaskStateStopped.IsActive())
	assert.False(t, TaskStateCompleted.IsActive())
	assert.False(t, TaskStateFailed.IsActive())
}

func TestTask_Clone_AllFields(t *testing.T) {
	createdAt := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	startedAt := time.Date(2024, 1, 1, 1, 0, 0, 0, time.UTC)

	task := &Task{
		ID:        "task-1",
		State:     TaskStateRunning,
		CreatedAt: &createdAt,
		StartedAt: &startedAt,
		Run:       "echo hello",
		Image:     "alpine:latest",
		Env:       map[string]string{"KEY": "VAL"},
		Files:     map[string]string{"file.txt": "content"},
		Error:     "",
		Networks:  []string{"bridge"},
		Retry:     &TaskRetry{Limit: 3, Attempts: 1},
		Limits:    &TaskLimits{CPUs: "2", Memory: "512m"},
		Registry:  &Registry{Username: "user", Password: "pass"},
		Timeout:   "60s",
		Result:    "success",
		GPUs:      "1",
		Mounts:    []Mount{{Type: MountTypeBind, Source: "/src", Target: "/dst"}},
	}

	clone := task.Clone()

	assert.Equal(t, task.ID, clone.ID)
	assert.Equal(t, task.State, clone.State)
	assert.Equal(t, *task.CreatedAt, *clone.CreatedAt)
	assert.Equal(t, task.Run, clone.Run)
	assert.Equal(t, task.Image, clone.Image)
	assert.Equal(t, task.Env, clone.Env)
	assert.Equal(t, task.Files, clone.Files)
	assert.Equal(t, task.Retry.Limit, clone.Retry.Limit)
	assert.NotSame(t, task.Retry, clone.Retry)
	assert.Equal(t, task.Limits.CPUs, clone.Limits.CPUs)
	assert.NotSame(t, task.Limits, clone.Limits)
	assert.Equal(t, task.Registry.Username, clone.Registry.Username)
	assert.NotSame(t, task.Registry, clone.Registry)
	assert.Equal(t, task.Mounts, clone.Mounts)
	assert.Equal(t, task.Networks, clone.Networks)
}

func TestTask_Clone_NilOptionalFields(t *testing.T) {
	task := &Task{
		ID:    "minimal",
		State: TaskStatePending,
	}

	clone := task.Clone()

	assert.Equal(t, task.ID, clone.ID)
	assert.Nil(t, clone.Retry)
	assert.Nil(t, clone.Limits)
	assert.Nil(t, clone.Registry)
	assert.Nil(t, clone.Env)
	assert.Nil(t, clone.Files)
}

func TestTask_Clone_NilEnvSafe(t *testing.T) {
	task := &Task{ID: "x", Env: nil, Files: nil}
	clone := task.Clone()
	assert.Nil(t, clone.Env)
	assert.Nil(t, clone.Files)
}

func TestTask_Clone_PrePostTasks(t *testing.T) {
	pre := &Task{ID: "pre-1", State: TaskStateCompleted}
	post := &Task{ID: "post-1", State: TaskStatePending}
	task := &Task{
		ID:   "main",
		Pre:  []*Task{pre},
		Post: []*Task{post},
	}

	clone := task.Clone()

	require := assert.New(t)
	require.Len(clone.Pre, 1)
	require.Equal("pre-1", clone.Pre[0].ID)
	require.NotSame(task.Pre[0], clone.Pre[0])
	require.Len(clone.Post, 1)
	require.Equal("post-1", clone.Post[0].ID)
	require.NotSame(task.Post[0], clone.Post[0])
}

func TestCloneTasks(t *testing.T) {
	tasks := []*Task{
		{ID: "a", State: TaskStatePending},
		{ID: "b", State: TaskStateRunning},
	}
	cloned := CloneTasks(tasks)

	assert.Len(t, cloned, 2)
	assert.Equal(t, "a", cloned[0].ID)
	assert.NotSame(t, tasks[0], cloned[0])
	assert.Equal(t, "b", cloned[1].ID)
	assert.NotSame(t, tasks[1], cloned[1])
}

func TestTaskRetry_Clone(t *testing.T) {
	r := &TaskRetry{Limit: 5, Attempts: 2}
	c := r.Clone()
	assert.Equal(t, r.Limit, c.Limit)
	assert.Equal(t, r.Attempts, c.Attempts)
	assert.NotSame(t, r, c)
}

func TestTaskLimits_Clone(t *testing.T) {
	l := &TaskLimits{CPUs: "4", Memory: "1g"}
	c := l.Clone()
	assert.Equal(t, l.CPUs, c.CPUs)
	assert.Equal(t, l.Memory, c.Memory)
	assert.NotSame(t, l, c)
}

func TestRegistry_Clone(t *testing.T) {
	r := &Registry{Username: "admin", Password: "s3cret"}
	c := r.Clone()
	assert.Equal(t, r.Username, c.Username)
	assert.Equal(t, r.Password, c.Password)
	assert.NotSame(t, r, c)
}

func TestMountTypeConstants(t *testing.T) {
	assert.Equal(t, "volume", MountTypeVolume)
	assert.Equal(t, "bind", MountTypeBind)
	assert.Equal(t, "tmpfs", MountTypeTmpfs)
}
