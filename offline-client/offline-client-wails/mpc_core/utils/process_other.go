//go:build !windows

package utils

import "os/exec"

func configureHiddenProcess(cmd *exec.Cmd) {}
