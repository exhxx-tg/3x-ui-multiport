package service

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"unicode"
)

// SshSswsManager handles the interaction with the host OS for managing SSH and SSWS users/configs.
type SshSswsManager struct{}

var usernameRegex = regexp.MustCompile(`^[a-z_][a-z0-9_]*$`)

// SanitizeLinuxUsername normalizes user-provided account names before any
// useradd/usermod/userdel call. Linux useradd rejects spaces, uppercase
// letters, and many punctuation characters, so we convert to a conservative
// lowercase POSIX-safe form used consistently by the DB, payloads, and OS.
func SanitizeLinuxUsername(username string) string {
	username = strings.TrimSpace(strings.ToLower(username))
	if username == "" {
		return ""
	}

	var b strings.Builder
	lastUnderscore := false
	for _, r := range username {
		valid := (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_'
		if valid {
			b.WriteRune(r)
			lastUnderscore = r == '_'
			continue
		}

		// Treat any space/separator/punctuation/non-ASCII/invalid character as a
		// single underscore so names like "Ali VPN 1" become "ali_vpn_1".
		if !lastUnderscore || unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteByte('_')
			lastUnderscore = true
		}
	}

	sanitized := strings.Trim(b.String(), "_")
	if sanitized == "" {
		return "u"
	}
	if sanitized[0] >= '0' && sanitized[0] <= '9' {
		sanitized = "u_" + sanitized
	}
	return sanitized
}

// CreateOSUser creates a restricted Linux user for SSH.
func (m *SshSswsManager) CreateOSUser(username, password string) error {
	username = SanitizeLinuxUsername(username)
	if !usernameRegex.MatchString(username) {
		return fmt.Errorf("invalid username after sanitization")
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
	username = SanitizeLinuxUsername(username)
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
	username = SanitizeLinuxUsername(username)
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
	username = SanitizeLinuxUsername(username)
	if !usernameRegex.MatchString(username) {
		return fmt.Errorf("invalid username")
	}
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
