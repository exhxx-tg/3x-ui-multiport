package service

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

// SshSswsManager handles the interaction with the host OS for managing SSH and SSWS users/configs.
type SshSswsManager struct{}

var usernameRegex = regexp.MustCompile(`^[a-z_][a-z0-9_-]*$`)

// CreateOSUser creates a restricted Linux user for SSH.
func (m *SshSswsManager) CreateOSUser(username, password string) error {
	if !usernameRegex.MatchString(username) {
		return fmt.Errorf("invalid username: must start with a letter/underscore and contain only lowercase, numbers, underscores, or hyphens")
	}

	if _, err := exec.Command("getent", "passwd", username).Output(); err != nil {
		cmd := exec.Command("useradd", "-M", "-s", "/bin/false", username)
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("useradd failed: %w: %s", err, string(out))
		}
	} else {
		_ = exec.Command("usermod", "-s", "/bin/false", username).Run()
	}

	passwdCmd := exec.Command("chpasswd")
	passwdCmd.Stdin = strings.NewReader(fmt.Sprintf("%s:%s\n", username, password))
	if out, err := passwdCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("chpasswd failed: %w: %s", err, string(out))
	}

	return nil
}

// DeleteOSUser removes the Linux user.
func (m *SshSswsManager) DeleteOSUser(username string) error {
	if !usernameRegex.MatchString(username) {
		return fmt.Errorf("invalid username")
	}
	cmd := exec.Command("userdel", "-f", username)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("userdel failed: %w: %s", err, string(out))
	}
	return nil
}

// UpdateOSUserPassword updates the password for an existing user.
func (m *SshSswsManager) UpdateOSUserPassword(username, password string) error {
	if !usernameRegex.MatchString(username) {
		return fmt.Errorf("invalid username")
	}
	passwdCmd := exec.Command("chpasswd")
	passwdCmd.Stdin = strings.NewReader(fmt.Sprintf("%s:%s\n", username, password))
	if out, err := passwdCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("chpasswd failed: %w: %s", err, string(out))
	}
	return nil
}

// ManageSSWSConfig handles the credential files for SSWS.
// Implementation depends on the specific SSWS daemon used.
func (m *SshSswsManager) ManageSSWSConfig(username, password string, action string) error {
	configPath := "/etc/ssws/users.conf"
	if err := os.MkdirAll("/etc/ssws", 0755); err != nil {
		return err
	}

	switch action {
	case "add", "update":
		return m.writeSswsUser(configPath, username, password)
	case "delete":
		return m.deleteSswsUser(configPath, username)
	}
	return nil
}

func (m *SshSswsManager) writeSswsUser(path, username, password string) error {
	users := map[string]string{}
	if raw, err := os.ReadFile(path); err == nil {
		for _, line := range strings.Split(string(raw), "\n") {
			name, pass, ok := strings.Cut(strings.TrimSpace(line), ":")
			if ok && usernameRegex.MatchString(name) {
				users[name] = pass
			}
		}
	}
	users[username] = password
	var b strings.Builder
	for name, pass := range users {
		b.WriteString(name)
		b.WriteByte(':')
		b.WriteString(pass)
		b.WriteByte('\n')
	}
	return os.WriteFile(path, []byte(b.String()), 0600)
}

func (m *SshSswsManager) deleteSswsUser(path, username string) error {
	users := map[string]string{}
	if raw, err := os.ReadFile(path); err == nil {
		for _, line := range strings.Split(string(raw), "\n") {
			name, pass, ok := strings.Cut(strings.TrimSpace(line), ":")
			if ok && name != username && usernameRegex.MatchString(name) {
				users[name] = pass
			}
		}
	}
	var b strings.Builder
	for name, pass := range users {
		b.WriteString(name)
		b.WriteByte(':')
		b.WriteString(pass)
		b.WriteByte('\n')
	}
	return os.WriteFile(path, []byte(b.String()), 0600)
}
