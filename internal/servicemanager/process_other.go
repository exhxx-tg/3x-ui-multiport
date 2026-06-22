//go:build !windows

package servicemanager

import "os/exec"

func attachChildLifetime(_ *exec.Cmd) {}
