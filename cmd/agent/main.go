package main

import (
	"log"
	"time"

	"github.com/flowline-io/flowbot/pkg/utils/reexec"
	"go.uber.org/automaxprocs/maxprocs"
	"go.uber.org/fx"
)

func main() {
	// reexec if requested
	if reexec.Init() {
		return
	} else {
		_, _ = maxprocs.Set(maxprocs.Logger(log.Printf))
	}

	fx.New(
		fx.StartTimeout(time.Minute),
		fx.StopTimeout(time.Minute),
		Modules,
	).Run()
}
