package service

import "testing"

func TestSanitizeLinuxUsername(t *testing.T) {
	tests := map[string]string{
		"Ali VPN 1":        "ali_vpn_1",
		"  User.Name-01  ": "user_name_01",
		"123 Client":       "u_123_client",
		"___":              "u",
		"Valid_user":       "valid_user",
	}

	for input, want := range tests {
		if got := SanitizeLinuxUsername(input); got != want {
			t.Fatalf("SanitizeLinuxUsername(%q) = %q, want %q", input, got, want)
		}
		if got := SanitizeLinuxUsername(input); got != "" && !usernameRegex.MatchString(got) {
			t.Fatalf("sanitized username %q does not match Linux username regex", got)
		}
	}
}
