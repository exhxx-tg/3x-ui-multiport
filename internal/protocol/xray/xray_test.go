package xray

import (
	"testing"

	xuiProtocol "github.com/exhxx-tg/3x-ui-multiport/internal/protocol"
)

func TestNewXrayBaseProtocol(t *testing.T) {
	p := NewXrayBaseProtocol(xuiProtocol.ProtocolVMess)
	if p == nil {
		t.Fatal("expected non-nil protocol")
	}
	if p.ID() != xuiProtocol.ProtocolVMess {
		t.Errorf("expected VMess, got %s", p.ID())
	}
}

func TestProtoToXrayProtocol(t *testing.T) {
	tests := []struct {
		id       xuiProtocol.ProtocolID
		expected string
	}{
		{xuiProtocol.ProtocolVMess, "vmess"},
		{xuiProtocol.ProtocolVLESS, "vless"},
		{xuiProtocol.ProtocolTrojan, "trojan"},
		{xuiProtocol.ProtocolShadowsocks, "shadowsocks"},
		{xuiProtocol.ProtocolHysteria, "hysteria"},
		{xuiProtocol.ProtocolOpenVPN, "vless"},
	}

	for _, tt := range tests {
		t.Run(string(tt.id), func(t *testing.T) {
			got := ProtoToXrayProtocol(tt.id)
			if string(got) != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, got)
			}
		})
	}
}

func TestStatusTransitions(t *testing.T) {
	p := NewXrayBaseProtocol(xuiProtocol.ProtocolVLESS)

	if p.Status() != xuiProtocol.StatusUnknown {
		t.Errorf("expected unknown status initially")
	}

	_ = p.Start()
	if p.Status() != xuiProtocol.StatusRunning {
		t.Errorf("expected running status")
	}

	_ = p.Stop()
	if p.Status() != xuiProtocol.StatusStopped {
		t.Errorf("expected stopped status")
	}

	_ = p.Restart()
	if p.Status() != xuiProtocol.StatusRunning {
		t.Errorf("expected running after restart")
	}
}

func TestApplyConfig(t *testing.T) {
	p := NewXrayBaseProtocol(xuiProtocol.ProtocolTrojan)

	err := p.ApplyConfig(map[string]any{"port": 443})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.Port() != 443 {
		t.Errorf("expected port 443, got %d", p.Port())
	}

	err = p.ApplyConfig("invalid")
	if err == nil {
		t.Error("expected error for invalid config type")
	}
}

func TestAllBaseProtocolIDs(t *testing.T) {
	ids := []xuiProtocol.ProtocolID{
		xuiProtocol.ProtocolVMess,
		xuiProtocol.ProtocolVLESS,
		xuiProtocol.ProtocolTrojan,
		xuiProtocol.ProtocolShadowsocks,
		xuiProtocol.ProtocolHysteria,
	}
	for _, id := range ids {
		t.Run(string(id), func(t *testing.T) {
			p := NewXrayBaseProtocol(id)
			info := p.Info()
			if info.ID != id {
				t.Errorf("expected ID %s, got %s", id, info.ID)
			}
			if info.Category != xuiProtocol.CategoryBase {
				t.Errorf("expected category base, got %s", info.Category)
			}
		})
	}
}
