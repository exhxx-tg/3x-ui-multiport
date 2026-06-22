package xray

import (
	"fmt"
	"sync"

	"github.com/exhxx-tg/3x-ui-multiport/internal/database"
	"github.com/exhxx-tg/3x-ui-multiport/internal/database/model"
	xuiProtocol "github.com/exhxx-tg/3x-ui-multiport/internal/protocol"
	"gorm.io/gorm"
)

func init() {
	xuiProtocol.RegisterBaseFactory(func(id xuiProtocol.ProtocolID) xuiProtocol.BaseProtocol {
		switch id {
		case xuiProtocol.ProtocolVMess,
			xuiProtocol.ProtocolVLESS,
			xuiProtocol.ProtocolTrojan,
			xuiProtocol.ProtocolShadowsocks,
			xuiProtocol.ProtocolHysteria:
			return NewXrayBaseProtocol(id)
		default:
			return nil
		}
	})
}

type XrayBaseProtocol struct {
	mu       sync.RWMutex
	id       xuiProtocol.ProtocolID
	info     xuiProtocol.ProtocolInfo
	status   xuiProtocol.Status
	port     int
	db       *gorm.DB
	stopChan chan struct{}
}

func NewXrayBaseProtocol(id xuiProtocol.ProtocolID) *XrayBaseProtocol {
	info := xuiProtocol.GetProtocolInfo(id)
	if info == nil {
		info = &xuiProtocol.ProtocolInfo{ID: id, Name: string(id), Category: xuiProtocol.CategoryBase}
	}
	return &XrayBaseProtocol{
		id:       id,
		info:     *info,
		status:   xuiProtocol.StatusUnknown,
		db:       database.GetDB(),
		stopChan: make(chan struct{}),
	}
}

func (p *XrayBaseProtocol) ID() xuiProtocol.ProtocolID { return p.id }

func (p *XrayBaseProtocol) Info() xuiProtocol.ProtocolInfo { return p.info }

func (p *XrayBaseProtocol) Port() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.port
}

func (p *XrayBaseProtocol) SetPort(port int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.port = port
}

func (p *XrayBaseProtocol) Status() xuiProtocol.Status {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.status
}

func (p *XrayBaseProtocol) hasEnabledInbounds() bool {
	if p.db == nil {
		return false
	}
	var count int64
	p.db.Model(&model.Inbound{}).
		Where("protocol = ? AND enable = ?", string(p.id), true).
		Count(&count)
	return count > 0
}

func (p *XrayBaseProtocol) firstInboundPort() int {
	if p.db == nil {
		return 0
	}
	var inbound model.Inbound
	result := p.db.Where("protocol = ?", string(p.id)).
		Order("id ASC").
		First(&inbound)
	if result.Error != nil {
		return 0
	}
	return inbound.Port
}

func (p *XrayBaseProtocol) Start() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.status == xuiProtocol.StatusRunning {
		return nil
	}

	if p.db == nil {
		p.status = xuiProtocol.StatusRunning
		return nil
	}

	if !p.hasEnabledInbounds() {
		return fmt.Errorf("no enabled inbounds for %s", p.id)
	}

	p.port = p.firstInboundPort()
	p.status = xuiProtocol.StatusRunning
	return nil
}

func (p *XrayBaseProtocol) Stop() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.status != xuiProtocol.StatusRunning {
		return nil
	}

	p.status = xuiProtocol.StatusStopped
	return nil
}

func (p *XrayBaseProtocol) Restart() error {
	_ = p.Stop()
	return p.Start()
}

func (p *XrayBaseProtocol) Config() (any, error) {
	if p.db == nil {
		return []model.Inbound{}, nil
	}
	var inbounds []model.Inbound
	result := p.db.Where("protocol = ?", string(p.id)).Find(&inbounds)
	if result.Error != nil {
		return nil, result.Error
	}
	return inbounds, nil
}

func (p *XrayBaseProtocol) ApplyConfig(cfg any) error {
	c, ok := cfg.(map[string]any)
	if !ok {
		return xuiProtocol.ErrInvalidConfig
	}
	if port, ok := c["port"].(int); ok {
		p.mu.Lock()
		p.port = port
		p.mu.Unlock()
	}
	return nil
}

func ProtoToXrayProtocol(protoID xuiProtocol.ProtocolID) model.Protocol {
	switch protoID {
	case xuiProtocol.ProtocolVMess:
		return model.VMESS
	case xuiProtocol.ProtocolVLESS:
		return model.VLESS
	case xuiProtocol.ProtocolTrojan:
		return model.Trojan
	case xuiProtocol.ProtocolShadowsocks:
		return model.Shadowsocks
	case xuiProtocol.ProtocolHysteria:
		return model.Hysteria
	default:
		return model.VLESS
	}
}
