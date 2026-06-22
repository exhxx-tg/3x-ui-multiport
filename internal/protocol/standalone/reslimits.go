package standalone

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

type ResourceLimits struct {
	MaxMemory    int    `json:"maxMemory"`    // MB
	MaxCPU       int    `json:"maxCPU"`       // percent * 100 (10000 = 100%)
	MaxProcesses int    `json:"maxProcesses"` // nproc limit
	MaxOpenFiles int    `json:"maxOpenFiles"` // nofile limit
	Nice         int    `json:"nice"`         // -20 to 19
	IOClass      string `json:"ioClass"`      // idle, best-effort, realtime
}

func DefaultResourceLimits() ResourceLimits {
	return ResourceLimits{
		MaxMemory:    256,
		MaxCPU:       5000,
		MaxProcesses: 100,
		MaxOpenFiles: 1024,
		Nice:         0,
		IOClass:      "best-effort",
	}
}

func ApplyResourceLimits(serviceName string, limits ResourceLimits) error {
	if runtime.GOOS == "windows" {
		return nil
	}
	if !hasSystemd() {
		return nil
	}
	dropInDir := fmt.Sprintf("/etc/systemd/system/%s.service.d", serviceName)
	if err := os.MkdirAll(dropInDir, 0755); err != nil {
		return fmt.Errorf("failed to create drop-in dir: %w", err)
	}
	var sb strings.Builder
	sb.WriteString("[Service]\n")
	if limits.MaxMemory > 0 {
		sb.WriteString(fmt.Sprintf("MemoryMax=%dM\n", limits.MaxMemory))
	}
	if limits.MaxCPU > 0 {
		sb.WriteString(fmt.Sprintf("CPUQuota=%d%%\n", limits.MaxCPU/100))
	}
	if limits.MaxProcesses > 0 {
		sb.WriteString(fmt.Sprintf("TasksMax=%d\n", limits.MaxProcesses))
	}
	if limits.MaxOpenFiles > 0 {
		sb.WriteString(fmt.Sprintf("LimitNOFILE=%d\n", limits.MaxOpenFiles))
	}
	dropIn := filepath.Join(dropInDir, "50-x-ui-pro-limits.conf")
	if err := os.WriteFile(dropIn, []byte(sb.String()), 0644); err != nil {
		return fmt.Errorf("failed to write drop-in: %w", err)
	}
	return exec.Command("systemctl", "daemon-reload").Run()
}

func RemoveResourceLimits(serviceName string) error {
	if runtime.GOOS == "windows" {
		return nil
	}
	dropIn := fmt.Sprintf("/etc/systemd/system/%s.service.d/50-x-ui-pro-limits.conf", serviceName)
	if err := os.Remove(dropIn); err != nil && !os.IsNotExist(err) {
		return err
	}
	return exec.Command("systemctl", "daemon-reload").Run()
}

func hasSystemd() bool {
	if _, err := os.Stat("/run/systemd/system"); os.IsNotExist(err) {
		return false
	}
	return true
}

func ApplyUlimit(limits ResourceLimits, existingArgs []string) []string {
	args := make([]string, len(existingArgs))
	copy(args, existingArgs)
	if limits.Nice != 0 {
		args = append([]string{"-n", strconv.Itoa(limits.Nice)}, args...)
	}
	return args
}
