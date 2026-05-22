package pipeline

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestRealClock(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
	}{
		{name: "now returns current time"},
		{name: "after fires within tolerance"},
		{name: "now always advances forward"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := NewRealClock()
			now1 := c.Now()
			assert.WithinDuration(t, time.Now(), now1, 100*time.Millisecond)

			if tt.name == "after fires within tolerance" {
				ch := c.After(10 * time.Millisecond)
				st := time.Now()
				<-ch
				assert.WithinDuration(t, st.Add(10*time.Millisecond), time.Now(), 50*time.Millisecond)
			}

			if tt.name == "now always advances forward" {
				time.Sleep(1 * time.Millisecond)
				now2 := c.Now()
				assert.True(t, now2.After(now1))
			}
		})
	}
}

func TestFakeClock(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
	}{
		{name: "now returns seed time"},
		{name: "after fires on advance"},
		{name: "advance triggers pending timers in order"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			seed := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
			c := NewFakeClock(seed)
			assert.Equal(t, seed, c.Now())

			if tt.name == "after fires on advance" {
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
			}

			if tt.name == "advance triggers pending timers in order" {
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
			}
		})
	}
}
