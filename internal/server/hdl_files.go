/******************************************************************************
 *
 *  Description :
 *
 *    Handler of large file uploads/downloads. Validates request first then calls
 *    a handler.
 *
 *****************************************************************************/

package server

import (
	"github.com/flowline-io/flowbot/internal/store"
	"math/rand"
	"time"

	"github.com/flowline-io/flowbot/pkg/logs"
)

// largeFileRunGarbageCollection runs every 'period' and deletes up to 'blockSize' unused files.
// Returns channel which can be used to stop the process.
func largeFileRunGarbageCollection(period time.Duration, blockSize int) chan<- bool {
	// Unbuffered stop channel. Whomever stops the gc must wait for the process to finish.
	stop := make(chan bool)
	go func() {
		// Add some randomness to the tick period to desynchronize runs on cluster nodes:
		// 0.75 * period + rand(0, 0.5) * period.
		period = (period >> 1) + (period >> 2) + time.Duration(rand.Intn(int(period>>1)))
		gcTicker := time.Tick(period)
		for {
			select {
			case <-gcTicker:
				if _, err := store.Chatbot.FileDeleteUnused(time.Now().Add(-time.Hour), blockSize); err != nil {
					logs.Warn.Println("media gc:", err)
				}
			case <-stop:
				return
			}
		}
	}()

	return stop
}
