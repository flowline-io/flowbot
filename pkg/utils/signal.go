package utils

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/flowline-io/flowbot/pkg/flog"
)

func SignalHandler() <-chan bool {
	stop := make(chan bool)

	signchan := make(chan os.Signal, 1)
	signal.Notify(signchan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)

	go func() {
		// Wait for a signal. Don't care which signal it is
		sig := <-signchan
		flog.Info("Signal received: '%s', shutting down", sig)
		stop <- true
	}()

	return stop
}
