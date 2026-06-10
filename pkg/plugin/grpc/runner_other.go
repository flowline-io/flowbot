//go:build !linux

package grpc

import "os/exec"

func setPdeathsig(_ *exec.Cmd) {}
