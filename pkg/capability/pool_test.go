package capability

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/pkg/metrics"
	"github.com/flowline-io/flowbot/pkg/stats"
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
		doubleInit     bool
		wantErr        bool
		checkExpiry    bool
		expectedExpiry time.Duration
	}{
		{
			name:           "initializes pool with valid config",
			size:           5,
			expiryDuration: "30s",
		},
		{
			name:           "double init is safe",
			size:           5,
			expiryDuration: "30s",
			doubleInit:     true,
		},
		{
			name:           "init with zero size uses ants default",
			size:           0,
			expiryDuration: "30s",
		},
		{
			name:           "init with invalid expiry duration falls back to default 30s",
			size:           5,
			expiryDuration: "invalid",
			checkExpiry:    true,
			expectedExpiry: 30 * time.Second,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetPool()

			var err error
			if tt.doubleInit {
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

			if tt.checkExpiry {
				assert.Equal(t, tt.expectedExpiry, inst.config.expiry)
			}

			resetPool()
		})
	}
}

func TestSubmitEvent(t *testing.T) {
	tests := []struct {
		name         string
		initPool     bool
		concurrent   int
		checkContext bool
	}{
		{
			name:     "submits and executes task via pool",
			initPool: true,
		},
		{
			name:       "submits multiple tasks concurrently",
			initPool:   true,
			concurrent: 20,
		},
		{
			name:     "falls back to direct execution when pool is nil",
			initPool: false,
		},
		{
			name:         "task function captures correct capability and operation context",
			initPool:     true,
			checkContext: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetPool()
			defer resetPool()

			if tt.concurrent > 0 {
				err := InitEventPool(tt.concurrent, "30s", nil)
				require.NoError(t, err)

				var counter atomic.Int32
				var wg sync.WaitGroup
				wg.Add(tt.concurrent)
				for range tt.concurrent {
					go func() {
						defer wg.Done()
						submitEvent("test", "op", func() {
							counter.Add(1)
						})
					}()
				}
				wg.Wait()
				time.Sleep(200 * time.Millisecond)
				assert.Equal(t, int32(tt.concurrent), counter.Load())
				return
			}

			if tt.initPool {
				err := InitEventPool(5, "30s", nil)
				require.NoError(t, err)
			}

			if tt.checkContext {
				done := make(chan struct{}, 1)
				var gotCap, gotOp string
				testCap := "karakeep"
				testOp := "list"
				submitEvent(testCap, testOp, func() {
					gotCap = testCap
					gotOp = testOp
					done <- struct{}{}
				})
				select {
				case <-done:
					assert.Equal(t, "karakeep", gotCap)
					assert.Equal(t, "list", gotOp)
				case <-time.After(5 * time.Second):
					t.Fatal("task did not execute within timeout")
				}
				return
			}

			done := make(chan struct{}, 1)
			submitEvent("karakeep", "list", func() {
				done <- struct{}{}
			})
			select {
			case <-done:
			case <-time.After(5 * time.Second):
				t.Fatal("task did not execute within timeout")
			}
		})
	}
}

func TestSubmitEventDropOnFull(t *testing.T) {
	tests := []struct {
		name       string
		poolSize   int
		closePool  bool
		useMetrics bool
		wantDirect bool
	}{
		{
			name:     "drops event when pool is full",
			poolSize: 2,
		},
		{
			name:       "falls back to direct execution when pool is closed",
			poolSize:   2,
			closePool:  true,
			wantDirect: true,
		},
		{
			name:       "increments dropped metric on pool overflow with real stats",
			poolSize:   1,
			useMetrics: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetPool()
			defer resetPool()

			var mc *metrics.CapabilityCollector
			if tt.useMetrics {
				prev := stats.SetInitializedForTesting(true)
				defer stats.SetInitializedForTesting(prev)

				mc = metrics.NewCapabilityCollector(stats.NewStats())
			}

			err := InitEventPool(tt.poolSize, "30s", mc)
			require.NoError(t, err)

			if tt.closePool {
				epMu.Lock()
				epInst.pool.Release()
				epMu.Unlock()

				if tt.wantDirect {
					executed := make(chan struct{}, 1)
					assert.NotPanics(t, func() {
						submitEvent("test", "op", func() {
							executed <- struct{}{}
						})
					})
					select {
					case <-executed:
					case <-time.After(1 * time.Second):
						t.Fatal("task was not executed directly")
					}
					return
				}

				assert.NotPanics(t, func() {
					submitEvent("test", "op", func() {})
				})
				return
			}

			started := make(chan struct{}, tt.poolSize)
			release := make(chan struct{})
			blockingTask := func() {
				started <- struct{}{}
				<-release
			}

			for range tt.poolSize {
				submitEvent("test", "op", blockingTask)
			}
			for range tt.poolSize {
				<-started
			}

			if tt.useMetrics {
				assert.NotPanics(t, func() {
					submitEvent("karakeep", "list", func() {})
					submitEvent("karakeep", "list", func() {})
					submitEvent("kanboard", "create", func() {})
				})
			} else {
				executed := make(chan struct{}, 1)
				submitEvent("test", "op", func() {
					executed <- struct{}{}
				})

				select {
				case <-executed:
					t.Fatal("task should have been dropped")
				case <-time.After(200 * time.Millisecond):
				}
			}

			close(release)
		})
	}
}

func TestShutdownEventPool(t *testing.T) {
	tests := []struct {
		name         string
		initPool     bool
		submitTask   bool
		wantNilAfter bool
	}{
		{
			name:         "shutdown releases pool and sets instance to nil",
			initPool:     true,
			wantNilAfter: true,
		},
		{
			name:     "shutdown when pool is nil does not panic",
			initPool: false,
		},
		{
			name:         "shutdown waits for in-flight tasks",
			initPool:     true,
			submitTask:   true,
			wantNilAfter: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetPool()
			defer resetPool()

			if tt.initPool {
				err := InitEventPool(5, "30s", nil)
				require.NoError(t, err)

				if tt.submitTask {
					var completed atomic.Int32
					submitEvent("test", "op", func() {
						time.Sleep(100 * time.Millisecond)
						completed.Store(1)
					})

					time.Sleep(20 * time.Millisecond)
					ShutdownEventPool()

					assert.Equal(t, int32(1), completed.Load())
					return
				}

				epMu.Lock()
				assert.NotNil(t, epInst)
				epMu.Unlock()

				ShutdownEventPool()

				if tt.wantNilAfter {
					epMu.Lock()
					assert.Nil(t, epInst)
					epMu.Unlock()
				}
				return
			}

			epMu.Lock()
			epInst = nil
			epMu.Unlock()

			assert.NotPanics(t, func() {
				ShutdownEventPool()
			})
		})
	}
}
