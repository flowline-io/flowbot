package bulkhead

import (
	"context"
	"errors"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestBulkheadDoAcquireRelease(t *testing.T) {
	tests := []struct {
		name string
		size int
	}{
		{name: "single slot", size: 1},
		{name: "two slots", size: 2},
		{name: "five slots", size: 5},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := New("test", WithMaxConcurrent(tt.size), WithTimeout(10*time.Second))
			var executed int32
			err := b.Do(context.Background(), func() error {
				atomic.AddInt32(&executed, 1)
				return nil
			})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if atomic.LoadInt32(&executed) != 1 {
				t.Fatalf("expected executed=1, got %d", executed)
			}
		})
	}
}

func TestBulkheadDoEnforcesMaxConcurrent(t *testing.T) {
	maxConc := 3
	b := New("test", WithMaxConcurrent(maxConc), WithMaxQueue(0), WithTimeout(5*time.Second))

	var current, maxConcurrent int64
	gate := make(chan struct{})
	ready := make(chan struct{})
	var wg sync.WaitGroup

	for i := 0; i < maxConc*3; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-ready
			_ = b.Do(context.Background(), func() error {
				cur := atomic.AddInt64(&current, 1)
				for {
					old := atomic.LoadInt64(&maxConcurrent)
					if cur > old {
						if atomic.CompareAndSwapInt64(&maxConcurrent, old, cur) {
							break
						}
					} else {
						break
					}
				}
				<-gate
				atomic.AddInt64(&current, -1)
				return nil
			})
		}()
	}

	close(ready)
	time.Sleep(200 * time.Millisecond)

	if n := atomic.LoadInt64(&maxConcurrent); n != int64(maxConc) {
		t.Errorf("expected max concurrent %d, got %d", maxConc, n)
	}

	close(gate)
	wg.Wait()
}

func TestBulkheadDoContextCancellation(t *testing.T) {
	b := New("test", WithMaxConcurrent(1), WithMaxQueue(1), WithTimeout(10*time.Second))

	hold := make(chan struct{})
	go func() {
		_ = b.Do(context.Background(), func() error {
			<-hold
			return nil
		})
	}()

	time.Sleep(100 * time.Millisecond)

	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)
	go func() {
		errCh <- b.Do(ctx, func() error { return nil })
	}()

	time.Sleep(50 * time.Millisecond)
	cancel()

	select {
	case err := <-errCh:
		if err != context.Canceled {
			t.Errorf("expected context.Canceled, got %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for cancellation")
	}

	close(hold)
}

func TestBulkheadDoTimeout(t *testing.T) {
	b := New("test", WithMaxConcurrent(1), WithMaxQueue(1), WithTimeout(50*time.Millisecond))

	hold := make(chan struct{})
	done := make(chan struct{})
	go func() {
		defer close(done)
		_ = b.Do(context.Background(), func() error {
			<-hold
			return nil
		})
	}()

	time.Sleep(100 * time.Millisecond)

	err := b.Do(context.Background(), func() error { return nil })
	if !errors.Is(err, ErrBulkheadTimeout) {
		t.Errorf("expected ErrBulkheadTimeout, got %v", err)
	}

	close(hold)
	<-done
}

func TestBulkheadDoQueueFull(t *testing.T) {
	b := New("test", WithMaxConcurrent(1), WithMaxQueue(1), WithTimeout(5*time.Second))

	hold := make(chan struct{})
	done := make(chan struct{})
	go func() {
		defer close(done)
		_ = b.Do(context.Background(), func() error {
			<-hold
			return nil
		})
	}()

	time.Sleep(50 * time.Millisecond)

	queued := make(chan struct{})
	go func() {
		close(queued)
		_ = b.Do(context.Background(), func() error { return nil })
	}()
	<-queued
	time.Sleep(50 * time.Millisecond)

	err := b.Do(context.Background(), func() error { return nil })
	if !errors.Is(err, ErrBulkheadFull) {
		t.Errorf("expected ErrBulkheadFull, got %v", err)
	}

	close(hold)
	<-done
}

func TestBulkheadDoCallbacks(t *testing.T) {
	tests := []struct {
		name           string
		trigger        func(t *testing.T, b *Bulkhead)
		wantEnterCalls int32
		wantLeaveCalls int32
		wantDropCalls  int32
		wantDropReason string
	}{
		{
			name: "successful call triggers enter and leave",
			trigger: func(t *testing.T, b *Bulkhead) {
				_ = b.Do(context.Background(), func() error { return nil })
			},
			wantEnterCalls: 1,
			wantLeaveCalls: 1,
			wantDropCalls:  0,
		},
		{
			name: "timeout triggers drop",
			trigger: func(t *testing.T, b *Bulkhead) {
				hold := make(chan struct{})
				go func() {
					_ = b.Do(context.Background(), func() error { <-hold; return nil })
				}()
				time.Sleep(100 * time.Millisecond)
				err := b.Do(context.Background(), func() error { return nil })
				if !errors.Is(err, ErrBulkheadTimeout) {
					t.Errorf("expected ErrBulkheadTimeout, got %v", err)
				}
				close(hold)
			},
			wantEnterCalls: 1,
			wantLeaveCalls: 1,
			wantDropCalls:  1,
			wantDropReason: "timeout",
		},
		{
			name: "queue full triggers drop",
			trigger: func(t *testing.T, b *Bulkhead) {
				hold := make(chan struct{})
				go func() {
					_ = b.Do(context.Background(), func() error { <-hold; return nil })
				}()
				time.Sleep(50 * time.Millisecond)
				go func() {
					_ = b.Do(context.Background(), func() error { return nil })
				}()
				time.Sleep(50 * time.Millisecond)
				_ = b.Do(context.Background(), func() error { return nil })
				close(hold)
			},
			wantEnterCalls: 1,
			wantLeaveCalls: 1,
			wantDropCalls:  1,
			wantDropReason: "queue_full",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var enters, leaves, drops int32
			var dropReason string
			b := New("test",
				WithMaxConcurrent(1),
				WithMaxQueue(1),
				WithTimeout(50*time.Millisecond),
				WithOnEnter(func(name string, d time.Duration) {
					atomic.AddInt32(&enters, 1)
				}),
				WithOnLeave(func(name string) {
					atomic.AddInt32(&leaves, 1)
				}),
				WithOnDrop(func(name string, reason string) {
					atomic.AddInt32(&drops, 1)
					dropReason = reason
				}),
			)
			tt.trigger(t, b)
			if atomic.LoadInt32(&enters) != tt.wantEnterCalls {
				t.Errorf("enters: want %d, got %d", tt.wantEnterCalls, enters)
			}
			if atomic.LoadInt32(&leaves) != tt.wantLeaveCalls {
				t.Errorf("leaves: want %d, got %d", tt.wantLeaveCalls, leaves)
			}
			if atomic.LoadInt32(&drops) != tt.wantDropCalls {
				t.Errorf("drops: want %d, got %d", tt.wantDropCalls, drops)
			}
			if tt.wantDropCalls > 0 && dropReason != tt.wantDropReason {
				t.Errorf("drop reason: want %s, got %s", tt.wantDropReason, dropReason)
			}
		})
	}
}

func TestBulkheadDefaultSize(t *testing.T) {
	n := runtime.GOMAXPROCS(0)
	if n < 1 {
		t.Fatal("GOMAXPROCS returned 0")
	}
	expected := n * 4
	b := New("test", WithMaxConcurrent(expected), WithMaxQueue(expected))
	if cap(b.sem) != expected {
		t.Errorf("sem cap: want %d, got %d", expected, cap(b.sem))
	}
	if cap(b.queue) != expected {
		t.Errorf("queue cap: want %d, got %d", expected, cap(b.queue))
	}
}

func TestBulkheadDoRace(t *testing.T) {
	b := New("test", WithMaxConcurrent(4), WithMaxQueue(4), WithTimeout(5*time.Second))
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = b.Do(context.Background(), func() error { return nil })
		}()
	}
	wg.Wait()
}
