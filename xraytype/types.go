package xraytype

import (
	"bytes"

	"github.com/mhsanaei/3x-ui/v3/util/json_util"
)

// Config represents the full Xray configuration.
type Config struct {
	LogConfig        json_util.RawMessage `json:"log,omitempty"`
	RouterConfig     json_util.RawMessage `json:"routing,omitempty"`
	OutboundConfigs  json_util.RawMessage `json:"outbounds,omitempty"`
	DNSConfig        json_util.RawMessage `json:"dns,omitempty"`
	Transport        json_util.RawMessage `json:"transport,omitempty"`
	Policy           json_util.RawMessage `json:"policy,omitempty"`
	API              json_util.RawMessage `json:"api,omitempty"`
	Stats            json_util.RawMessage `json:"stats,omitempty"`
	Metrics          json_util.RawMessage `json:"metrics,omitempty"`
	Reverse          json_util.RawMessage `json:"reverse,omitempty"`
	FakeDNS          json_util.RawMessage `json:"fakedns,omitempty"`
	BurstObservatory json_util.RawMessage `json:"burstObservatory,omitempty"`
	InboundConfigs   []InboundConfig      `json:"inbounds"`
}

// Equals compares two Config instances for deep equality.
func (c *Config) Equals(other *Config) bool {
	if !bytes.Equal(c.LogConfig, other.LogConfig) {
		return false
	}
	if !bytes.Equal(c.RouterConfig, other.RouterConfig) {
		return false
	}
	if !bytes.Equal(c.OutboundConfigs, other.OutboundConfigs) {
		return false
	}
	if !bytes.Equal(c.DNSConfig, other.DNSConfig) {
		return false
	}
	if !bytes.Equal(c.Transport, other.Transport) {
		return false
	}
	if !bytes.Equal(c.Policy, other.Policy) {
		return false
	}
	if !bytes.Equal(c.API, other.API) {
		return false
	}
	if !bytes.Equal(c.Stats, other.Stats) {
		return false
	}
	if !bytes.Equal(c.Metrics, other.Metrics) {
		return false
	}
	if !bytes.Equal(c.Reverse, other.Reverse) {
		return false
	}
	if !bytes.Equal(c.FakeDNS, other.FakeDNS) {
		return false
	}
	if len(c.InboundConfigs) != len(other.InboundConfigs) {
		return false
	}
	for i := range c.InboundConfigs {
		if !c.InboundConfigs[i].Equals(&other.InboundConfigs[i]) {
			return false
		}
	}
	return true
}

// InboundConfig represents an Xray inbound configuration.
// It defines how Xray accepts incoming connections including protocol, port, and settings.
type InboundConfig struct {
	Listen         json_util.RawMessage `json:"listen"` // listen cannot be an empty string
	Port           int                  `json:"port"`
	Protocol       string               `json:"protocol"`
	Settings       json_util.RawMessage `json:"settings"`
	StreamSettings json_util.RawMessage `json:"streamSettings,omitempty"`
	Tag            string               `json:"tag"`
	Sniffing       json_util.RawMessage `json:"sniffing,omitempty"`
}

// Equals compares two InboundConfig instances for deep equality.
func (c *InboundConfig) Equals(other *InboundConfig) bool {
	if !bytes.Equal(c.Listen, other.Listen) {
		return false
	}
	if c.Port != other.Port {
		return false
	}
	if c.Protocol != other.Protocol {
		return false
	}
	if !bytes.Equal(c.Settings, other.Settings) {
		return false
	}
	if !bytes.Equal(c.StreamSettings, other.StreamSettings) {
		return false
	}
	if c.Tag != other.Tag {
		return false
	}
	if !bytes.Equal(c.Sniffing, other.Sniffing) {
		return false
	}
	return true
}

// ClientTraffic represents traffic statistics and limits for a specific client.
// It tracks upload/download usage, expiry times, and online status for inbound clients.
type ClientTraffic struct {
	Id         int    `json:"id" form:"id" gorm:"primaryKey;autoIncrement"`
	InboundId  int    `json:"inboundId" form:"inboundId"`
	Enable     bool   `json:"enable" form:"enable"`
	Email      string `json:"email" form:"email" gorm:"unique"`
	UUID       string `json:"uuid" form:"uuid" gorm:"-"`
	SubId      string `json:"subId" form:"subId" gorm:"-"`
	Up         int64  `json:"up" form:"up"`
	Down       int64  `json:"down" form:"down"`
	ExpiryTime int64  `json:"expiryTime" form:"expiryTime"`
	Total      int64  `json:"total" form:"total"`
	Reset      int    `json:"reset" form:"reset" gorm:"default:0"`
	LastOnline int64  `json:"lastOnline" form:"lastOnline" gorm:"default:0"`
}
