package ability

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/pkg/metrics"
)

// resetPool cleans up the global event pool instance between tests.
func resetPool() {
	epMu.Lock()
	if epInst != nil {
		epInst.pool.Release()
	}
	epInst = nil
	epMu.Unlock()
}

func TestInitEventPool(t *testing.T) {
	tests := []struct {
		name           string
		size           int
		expiryDuration string
		wantErr        bool
	}{
		{
			name:           "initializes pool with valid config",
			size:           5,
			expiryDuration: "30s",
			wantErr:        false,
		},
		{
			name:           "double init is safe",
			size:           5,
			expiryDuration: "30s",
			wantErr:        false,
		},
		{
			name:           "init with zero size uses ants default",
			size:           0,
			expiryDuration: "30s",
			wantErr:        false,
		},
		{
			name:           "init with invalid expiry duration falls back to default 30s",
			size:           5,
			expiryDuration: "invalid",
			wantErr:        false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetPool()

			var err error
			if tt.name == "double init is safe" {
				err = InitEventPool(5, "30s", nil)
				require.NoError(t, err)
				err = InitEventPool(5, "30s", nil)
			} else {
				err = InitEventPool(tt.size, tt.expiryDuration, nil)
			}

			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			epMu.Lock()
			inst := epInst
			epMu.Unlock()
			require.NotNil(t, inst)
			require.NotNil(t, inst.pool)

			if tt.name == "init with invalid expiry duration falls back to default 30s" {
				assert.Equal(t, 30*time.Second, inst.config.expiry)
			}

			resetPool()
		})
	}
}

func TestSubmitEvent(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "submits and executes task via pool"},
		{name: "submits multiple tasks concurrently"},
		{name: "falls back to direct execution when pool is nil"},
		{name: "task function captures correct capability and operation context"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetPool()
			defer resetPool()

			switch tt.name {
			case "submits and executes task via pool":
				err := InitEventPool(5, "30s", nil)
				require.NoError(t, err)

				done := make(chan struct{}, 1)
				submitEvent("bookmark", "list", func() {
					done <- struct{}{}
				})
				select {
				case <-done:
				case <-time.After(5 * time.Second):
					t.Fatal("task did not execute within timeout")
				}

			case "submits multiple tasks concurrently":
				err := InitEventPool(20, "30s", nil)
				require.NoError(t, err)

				var counter atomic.Int32
				const numTasks = 20
				var wg sync.WaitGroup
				wg.Add(numTasks)
				for range numTasks {
					go func() {
						defer wg.Done()
						submitEvent("test", "op", func() {
							counter.Add(1)
						})
					}()
				}
				wg.Wait()
				time.Sleep(200 * time.Millisecond)
				assert.Equal(t, int32(numTasks), counter.Load())

			case "falls back to direct execution when pool is nil":
				done := make(chan struct{}, 1)
				submitEvent("bookmark", "list", func() {
					done <- struct{}{}
				})
				select {
				case <-done:
				case <-time.After(1 * time.Second):
					t.Fatal("direct execution did not happen")
				}

			case "task function captures correct capability and operation context":
				err := InitEventPool(5, "30s", nil)
				require.NoError(t, err)

				done := make(chan struct{}, 1)
				var gotCap, gotOp string
				testCap := "bookmark"
				testOp := "list"
				submitEvent(testCap, testOp, func() {
					gotCap = testCap
					gotOp = testOp
					done <- struct{}{}
				})
				select {
				case <-done:
					assert.Equal(t, "bookmark", gotCap)
					assert.Equal(t, "list", gotOp)
				case <-time.After(5 * time.Second):
					t.Fatal("task did not execute within timeout")
				}
			}
		})
	}
}

func TestSubmitEventDropOnFull(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "drops event when pool is full"},
		{name: "drops event when pool is closed"},
		{name: "increments dropped metric on drop without panic"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetPool()
			defer resetPool()

			switch tt.name {
			case "drops event when pool is full":
				err := InitEventPool(2, "30s", nil)
				require.NoError(t, err)

				started := make(chan struct{}, 2)
				release := make(chan struct{})
				blockingTask := func() {
					started <- struct{}{}
					<-release
				}

				submitEvent("test", "op", blockingTask)
				submitEvent("test", "op", blockingTask)
				<-started
				<-started

				executed := make(chan struct{}, 1)
				submitEvent("test", "op", func() {
					executed <- struct{}{}
				})

				select {
				case <-executed:
					t.Fatal("task should have been dropped")
				case <-time.After(200 * time.Millisecond):
				}

				close(release)

			case "drops event when pool is closed":
				err := InitEventPool(2, "30s", nil)
				require.NoError(t, err)

				epMu.Lock()
				epInst.pool.Release()
				epMu.Unlock()

				assert.NotPanics(t, func() {
					submitEvent("test", "op", func() {})
				})

			case "increments dropped metric on drop without panic":
				mc := metrics.NewAbilityCollector(nil)
				err := InitEventPool(2, "30s", mc)
				require.NoError(t, err)

				epMu.Lock()
				epInst.pool.Release()
				epMu.Unlock()

				assert.NotPanics(t, func() {
					submitEvent("bookmark", "list", func() {})
					submitEvent("bookmark", "list", func() {})
					submitEvent("kanban", "create", func() {})
				})
			}
		})
	}
}

func TestShutdownEventPool(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "shutdown releases pool and sets instance to nil"},
		{name: "shutdown when pool is nil does not panic"},
		{name: "shutdown waits for in-flight tasks"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetPool()
			defer resetPool()

			switch tt.name {
			case "shutdown releases pool and sets instance to nil":
				err := InitEventPool(5, "30s", nil)
				require.NoError(t, err)

				epMu.Lock()
				assert.NotNil(t, epInst)
				epMu.Unlock()

				ShutdownEventPool()

				epMu.Lock()
				assert.Nil(t, epInst)
				epMu.Unlock()

			case "shutdown when pool is nil does not panic":
				epMu.Lock()
				epInst = nil
				epMu.Unlock()

				assert.NotPanics(t, func() {
					ShutdownEventPool()
				})

			case "shutdown waits for in-flight tasks":
				err := InitEventPool(5, "30s", nil)
				require.NoError(t, err)

				var completed atomic.Int32
				submitEvent("test", "op", func() {
					time.Sleep(100 * time.Millisecond)
					completed.Store(1)
				})

				time.Sleep(20 * time.Millisecond)
				ShutdownEventPool()

				assert.Equal(t, int32(1), completed.Load())
			}
		})
	}
}
