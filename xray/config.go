package xray

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/mhsanaei/3x-ui/v3/database"
	"github.com/mhsanaei/3x-ui/v3/util/json_util"
	"github.com/mhsanaei/3x-ui/v3/web/entity"
	"github.com/mhsanaei/3x-ui/v3/xraytype"
)

// InjectExtraProtocolsFallbacks injects SSH and SSWS fallbacks into the main Xray inbound.
// It reads the current extra_settings from the database and appends them to the fallback slice.
func InjectExtraProtocolsFallbacks(config *xraytype.Config) error {
	if config == nil || len(config.InboundConfigs) == 0 {
		return nil
	}

	db := database.GetDB()
	if db == nil {
		return nil
	}
	var extraSettings []entity.ExtraSetting
	if err := db.Find(&extraSettings).Error; err != nil {
		return fmt.Errorf("failed to fetch extra settings for fallbacks: %w", err)
	}

	// Fallbacks for VLESS/Trojan live inside inbound settings, not streamSettings.
	// Pick the first enabled TLS-capable inbound that can host them and never
	// create malformed RawMessage values when settings are empty or invalid.
	var targetInbound *xraytype.InboundConfig
	for i := range config.InboundConfigs {
		protocol := strings.ToLower(strings.TrimSpace(config.InboundConfigs[i].Protocol))
		if protocol == "vless" || protocol == "trojan" {
			targetInbound = &config.InboundConfigs[i]
			break
		}
	}
	if targetInbound == nil {
		return nil
	}

	var settings map[string]any
	if len(targetInbound.Settings) > 0 {
		if err := json.Unmarshal(targetInbound.Settings, &settings); err != nil {
			return fmt.Errorf("failed to unmarshal inbound settings for extra fallbacks: %w", err)
		}
	}
	if settings == nil {
		settings = make(map[string]any)
	}

	// Access the fallback list.
	var fallbacks []map[string]any
	if fs, ok := settings["fallbacks"].([]any); ok {
		for _, f := range fs {
			if fm, ok := f.(map[string]any); ok {
				fallbacks = append(fallbacks, fm)
			}
		}
	}

	// Update or append only protocols that are intended to be multiplexed by
	// Xray fallbacks. Skip bad database values instead of poisoning config.json.
	for _, setting := range extraSettings {
		protocol := strings.ToUpper(strings.TrimSpace(setting.ProtocolName))
		if !setting.IsEnabled || (protocol != "SSH" && protocol != "SSWS") {
			continue
		}
		if setting.ListeningPort <= 0 || setting.ListeningPort > 65535 || setting.ListeningPort == targetInbound.Port {
			continue
		}
		dest := fmt.Sprintf("127.0.0.1:%d", setting.ListeningPort)

		// Find existing fallback for this destination.
		found := false
		for i, f := range fallbacks {
			if f["dest"] == dest {
				fallbacks[i]["dest"] = dest
				found = true
				break
			}
		}

		if !found {
			// Add new fallback.
			// For SSH, we typically fallback based on protocol or a specific path.
			// Since we are multiplexing, we'll add it as a generic fallback.
			fallbacks = append(fallbacks, map[string]any{
				"dest": dest,
			})
		}
	}

	if len(fallbacks) == 0 {
		return nil
	}

	settings["fallbacks"] = fallbacks
	updatedSettings, err := json.Marshal(settings)
	if err != nil {
		return fmt.Errorf("failed to marshal updated inbound settings: %w", err)
	}
	targetInbound.Settings = json_util.RawMessage(updatedSettings)

	return nil
}
