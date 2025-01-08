package server

import (
	"crypto/rand"
	"math/big"
	"time"

	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/pkg/flog"
)

// largeFileRunGarbageCollection runs every 'period' and deletes up to 'blockSize' unused files.
// Returns channel which can be used to stop the process.
func largeFileRunGarbageCollection(period time.Duration, blockSize int) chan<- bool {
	// Unbuffered stop channel. Whoever stops the gc must wait for the process to finish.
	stop := make(chan bool)
	go func() {
		// Add some randomness to the tick period to desynchronize runs on cluster nodes:
		// 0.75 * period + rand(0, 0.5) * period.
		n, _ := rand.Int(rand.Reader, big.NewInt(int64(period>>1)))
		period = (period >> 1) + (period >> 2) + time.Duration(n.Int64())
		gcTicker := time.NewTicker(period)
		for {
			select {
			case <-gcTicker.C:
				if _, err := store.Database.FileDeleteUnused(time.Now().Add(-time.Hour), blockSize); err != nil {
					flog.Warn("media gc: %v", err)
				}
				// todo delete unused
			case <-stop:
				return
			}
		}
	}()

	return stop
}
