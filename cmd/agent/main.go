package main

import (
	_ "go.uber.org/automaxprocs"
	"go.uber.org/fx"
)

func main() {
	fx.New(
		Modules,
	).Run()
}
