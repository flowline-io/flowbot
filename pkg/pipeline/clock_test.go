package pipeline

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestRealClock(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		extra func(t *testing.T, c *RealClock, now1 time.Time)
	}{
		{
			name: "now returns current time",
			extra: func(t *testing.T, c *RealClock, _ time.Time) {
				assert.WithinDuration(t, time.Now(), c.Now(), 100*time.Millisecond)
			},
		},
		{
			name: "after fires within tolerance",
			extra: func(t *testing.T, c *RealClock, _ time.Time) {
				ch := c.After(50 * time.Millisecond)
				st := time.Now()
				<-ch
				assert.WithinDuration(t, st.Add(50*time.Millisecond), time.Now(), 200*time.Millisecond)
			},
		},
		{
			name: "now always advances forward",
			extra: func(t *testing.T, c *RealClock, now1 time.Time) {
				time.Sleep(1 * time.Millisecond)
				now2 := c.Now()
				assert.True(t, now2.After(now1))
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := NewRealClock()
			now1 := c.Now()
			assert.WithinDuration(t, time.Now(), now1, 100*time.Millisecond)
			tt.extra(t, c, now1)
		})
	}
}

func TestFakeClock(t *testing.T) {
	t.Parallel()
	seed := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	tests := []struct {
		name  string
		extra func(t *testing.T, c *FakeClock)
	}{
		{
			name: "now returns seed time",
			extra: func(t *testing.T, c *FakeClock) {
				assert.Equal(t, seed, c.Now())
			},
		},
		{
			name: "after fires on advance",
			extra: func(t *testing.T, c *FakeClock) {
				ch := c.After(1 * time.Hour)
				done := make(chan time.Time, 1)
				go func() { done <- <-ch }()
				c.Advance(1 * time.Hour)
				select {
				case fired := <-done:
					assert.Equal(t, seed.Add(1*time.Hour), fired)
				case <-time.After(100 * time.Millisecond):
					t.Fatal("After channel did not fire")
				}
			},
		},
		{
			name: "advance triggers pending timers in order",
			extra: func(t *testing.T, c *FakeClock) {
				ch1 := c.After(2 * time.Hour)
				ch2 := c.After(1 * time.Hour)
				done1 := make(chan time.Time, 1)
				done2 := make(chan time.Time, 1)
				go func() { done1 <- <-ch1 }()
				go func() { done2 <- <-ch2 }()
				c.Advance(3 * time.Hour)
				r1 := <-done2
				r2 := <-done1
				assert.Equal(t, seed.Add(1*time.Hour), r1)
				assert.Equal(t, seed.Add(2*time.Hour), r2)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := NewFakeClock(seed)
			tt.extra(t, c)
		})
	}
}
