package xray

import (
	"encoding/json"
	"fmt"

	"github.com/mhsanaei/3x-ui/v3/database"
	"github.com/mhsanaei/3x-ui/v3/web/entity"
)

// InjectExtraProtocolsFallbacks injects SSH and SSWS fallbacks into the main Xray inbound.
// It reads the current extra_settings from the database and appends them to the fallback slice.
func InjectExtraProtocolsFallbacks(config *Config) error {
	db := database.GetDB()
	var extraSettings []entity.ExtraSetting
	if err := db.Find(&extraSettings).Error; err != nil {
		return fmt.Errorf("failed to fetch extra settings for fallbacks: %w", err)
	}

	// We target the first inbound that looks like a "Main" inbound (usually port 443 or configured web port).
	// In a more robust system, we'd use a specific tag or configuration.
	if len(config.InboundConfigs) == 0 {
		return nil
	}

	// targetInbound is typically the first one.
	targetInbound := &config.InboundConfigs[0]

	// Fallbacks in Xray are part of the streamSettings.
	// We need to handle streamSettings as a RawMessage/JSON.
	var streamSettings map[string]any
	if err := json.Unmarshal(targetInbound.StreamSettings, &streamSettings); err != nil {
		return fmt.Errorf("failed to unmarshal streamSettings: %w", err)
	}

	// Access the fallback list.
	var fallbacks []map[string]any
	if fs, ok := streamSettings["fallbacks"].([]any); ok {
		for _, f := range fs {
			if fm, ok := f.(map[string]any); ok {
				fallbacks = append(fallbacks, fm)
			}
		}
	}

	// Update or Append fallbacks for Extra Protocols
	for _, setting := range extraSettings {
		if !setting.IsEnabled {
			continue
		}

		// Find existing fallback for this protocol
		found := false
		for i, f := range fallbacks {
			// We use a custom field or check the destination port to identify the fallback.
			if f["dest"] == fmt.Sprintf("127.0.0.1:%d", setting.ListeningPort) {
				fallbacks[i]["dest"] = fmt.Sprintf("127.0.0.1:%d", setting.ListeningPort)
				found = true
				break
			}
		}

		if !found {
			// Add new fallback. 
			// For SSH, we typically fallback based on protocol or a specific path.
			// Since we are multiplexing, we'll add it as a generic fallback.
			fallbacks = append(fallbacks, map[string]any{
				"dest": fmt.Sprintf("127.0.0.1:%d", setting.ListeningPort),
			})
		}
	}

	streamSettings["fallbacks"] = fallbacks
	updatedStream, err := json.Marshal(streamSettings)
	if err != nil {
		return fmt.Errorf("failed to marshal updated streamSettings: %w", err)
	}
	targetInbound.StreamSettings = string(updatedStream)

	return nil
}
