package servicemanager

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

type SystemdUnit struct {
	Name        string
	Description string
	BinaryPath  string
	Args        []string
	WorkDir     string
	User        string
	Group       string
	Restart     string
	RestartSec  string
	Environment []string
}

func DefaultSystemdUnit(name, binaryPath string) SystemdUnit {
	return SystemdUnit{
		Name:        name,
		Description: fmt.Sprintf("%s service", name),
		BinaryPath:  binaryPath,
		Restart:     "on-failure",
		RestartSec:  "5",
		User:        "root",
	}
}

func (u SystemdUnit) UnitFilePath() string {
	return fmt.Sprintf("/etc/systemd/system/%s.service", u.Name)
}

func (u SystemdUnit) GenerateUnitFile() string {
	var b strings.Builder
	b.WriteString("[Unit]\n")
	b.WriteString(fmt.Sprintf("Description=%s\n", u.Description))
	b.WriteString("After=network.target\n\n")

	b.WriteString("[Service]\n")
	b.WriteString(fmt.Sprintf("Type=simple\n"))
	b.WriteString(fmt.Sprintf("User=%s\n", u.User))
	if u.Group != "" {
		b.WriteString(fmt.Sprintf("Group=%s\n", u.Group))
	}
	b.WriteString(fmt.Sprintf("ExecStart=%s %s\n", u.BinaryPath, strings.Join(u.Args, " ")))
	if u.WorkDir != "" {
		b.WriteString(fmt.Sprintf("WorkingDirectory=%s\n", u.WorkDir))
	}
	for _, env := range u.Environment {
		b.WriteString(fmt.Sprintf("Environment=%s\n", env))
	}
	b.WriteString(fmt.Sprintf("Restart=%s\n", u.Restart))
	b.WriteString(fmt.Sprintf("RestartSec=%s\n", u.RestartSec))
	b.WriteString("StandardOutput=journal\n")
	b.WriteString("StandardError=journal\n\n")

	b.WriteString("[Install]\n")
	b.WriteString("WantedBy=multi-user.target\n")
	return b.String()
}

func HasSystemd() bool {
	if runtime.GOOS == "windows" {
		return false
	}
	if _, err := os.Stat("/run/systemd/system"); os.IsNotExist(err) {
		return false
	}
	return true
}

func InstallSystemdUnit(unit SystemdUnit) error {
	if !HasSystemd() {
		return fmt.Errorf("systemd not available")
	}
	content := unit.GenerateUnitFile()
	path := unit.UnitFilePath()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("failed to create systemd dir: %w", err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write unit file: %w", err)
	}
	if err := exec.Command("systemctl", "daemon-reload").Run(); err != nil {
		return fmt.Errorf("failed to reload systemd: %w", err)
	}
	return nil
}

func UninstallSystemdUnit(name string) error {
	if !HasSystemd() {
		return fmt.Errorf("systemd not available")
	}
	_ = exec.Command("systemctl", "stop", name).Run()
	_ = exec.Command("systemctl", "disable", name).Run()
	path := fmt.Sprintf("/etc/systemd/system/%s.service", name)
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove unit file: %w", err)
	}
	return exec.Command("systemctl", "daemon-reload").Run()
}

func EnableSystemdUnit(name string) error {
	return exec.Command("systemctl", "enable", name).Run()
}

func DisableSystemdUnit(name string) error {
	return exec.Command("systemctl", "disable", name).Run()
}

func SystemdUnitStatus(name string) (State, error) {
	if !HasSystemd() {
		return StateUnknown, fmt.Errorf("systemd not available")
	}
	cmd := exec.Command("systemctl", "is-active", name)
	output, err := cmd.Output()
	if err != nil {
		return StateUnknown, nil
	}
	status := strings.TrimSpace(string(output))
	switch status {
	case "active":
		return StateRunning, nil
	case "inactive", "dead":
		return StateStopped, nil
	default:
		return StateUnknown, nil
	}
}
