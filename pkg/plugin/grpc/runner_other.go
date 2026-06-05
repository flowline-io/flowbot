//go:build !linux

package grpc

import "os/exec"

func setPdeathsig(cmd *exec.Cmd) {}
