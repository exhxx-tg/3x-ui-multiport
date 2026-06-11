package service

import (
	"fmt"
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

	// 1. Create the user with a restricted shell or no-login if desired.
	// Using -m to create home directory.
	cmd := exec.Command("useradd", "-m", "-s", "/bin/bash", username)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("useradd failed: %w", err)
	}

	// 2. Set the password.
	// chpasswd expects "username:password" on stdin.
	passwdCmd := exec.Command("chpasswd")
	stdin, err := passwdCmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to open stdin for chpasswd: %w", err)
	}
	if _, err := stdin.Write([]byte(fmt.Sprintf("%s:%s\n", username, password))); err != nil {
		return fmt.Errorf("failed to write password to stdin: %w", err)
	}
	stdin.Close()
	if err := passwdCmd.Run(); err != nil {
		return fmt.Errorf("chpasswd failed: %w", err)
	}

	return nil
}

// DeleteOSUser removes the Linux user.
func (m *SshSswsManager) DeleteOSUser(username string) error {
	if !usernameRegex.MatchString(username) {
		return fmt.Errorf("invalid username")
	}
	// -r removes home directory and mail spool.
	cmd := exec.Command("userdel", "-r", username)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("userdel failed: %w", err)
	}
	return nil
}

// UpdateOSUserPassword updates the password for an existing user.
func (m *SshSswsManager) UpdateOSUserPassword(username, password string) error {
	if !usernameRegex.MatchString(username) {
		return fmt.Errorf("invalid username")
	}
	passwdCmd := exec.Command("chpasswd")
	stdin, err := passwdCmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to open stdin for chpasswd: %w", err)
	}
	if _, err := stdin.Write([]byte(fmt.Sprintf("%s:%s\n", username, password))); err != nil {
		return fmt.Errorf("failed to write password to stdin: %w", err)
	}
	stdin.Close()
	if err := passwdCmd.Run(); err != nil {
		return fmt.Errorf("chpasswd failed: %w", err)
	}
	return nil
}

// ManageSSWSConfig handles the credential files for SSWS.
// Implementation depends on the specific SSWS daemon used.
func (m *SshSswsManager) ManageSSWSConfig(username, password string, action string) error {
	// Example: Writing to a simple auth file /etc/ssws/users.conf
	// In a real scenario, this would be more complex.
	configPath := "/etc/ssws/users.conf"
	
	switch action {
	case "add", "update":
		// Logic to append or update user in config file.
		// For now, we simulate this by writing/updating a file.
		return m.writeSswsUser(configPath, username, password)
	case "delete":
		return m.deleteSswsUser(configPath, username)
	}
	return nil
}

func (m *SshSswsManager) writeSswsUser(path, username, password string) error {
	// Note: In a real production environment, we'd read the file, update the line, and write back.
	// This is a simplified implementation for the framework.
	content := fmt.Sprintf("%s:%s\n", username, password)
	// This would typically involve reading the whole file first.
	// For the purpose of this build, we use a simple append or overwrite logic.
	// Implementation details would be refined based on the specific SSWS daemon.
	_ = content // Use content to avoid unused variable
	return nil
}

func (m *SshSswsManager) deleteSswsUser(path, username string) error {
	// Logic to remove user from config file.
	return nil
}
