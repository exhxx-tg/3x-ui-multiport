package protocol

import (
	"fmt"
)

type (
	// BaseProtocolFactory creates a BaseProtocol for the given protocol ID.
	BaseProtocolFactory func(ProtocolID) BaseProtocol
	// StandaloneFactory creates a StandaloneService for the given protocol ID.
	StandaloneFactory func(ProtocolID) StandaloneService
	// WrapperFactory creates a TransportWrapper for the given protocol ID.
	WrapperFactory func(ProtocolID) TransportWrapper
)

var (
	baseFactories      []BaseProtocolFactory
	standaloneFactories []StandaloneFactory
	wrapperFactories    []WrapperFactory
)

// RegisterBaseFactory adds a factory that can create base protocol instances.
// Called during init() from implementation packages.
func RegisterBaseFactory(fn BaseProtocolFactory) {
	baseFactories = append(baseFactories, fn)
}

// RegisterStandaloneFactory adds a factory that can create standalone service
// instances. Called during init() from implementation packages.
func RegisterStandaloneFactory(fn StandaloneFactory) {
	standaloneFactories = append(standaloneFactories, fn)
}

// RegisterWrapperFactory adds a factory that can create transport wrapper
// instances. Called during init() from implementation packages.
func RegisterWrapperFactory(fn WrapperFactory) {
	wrapperFactories = append(wrapperFactories, fn)
}

type Manager struct {
	registry *Registry
}

func NewManager(registry *Registry) *Manager {
	return &Manager{registry: registry}
}

func (m *Manager) Initialize() error {
	for _, p := range AllProtocols {
		if m.registry.Get(p.ID) != nil {
			continue
		}

		switch p.Category {
		case CategoryBase:
			adapter := m.createBaseProtocol(p.ID)
			if adapter != nil {
				m.registry.Register(adapter)
			}

		case CategoryStandalone:
			svc := m.createStandaloneService(p.ID)
			if svc != nil {
				m.registry.Register(svc)
			}

		case CategoryWrapper:
			w := m.createWrapper(p.ID)
			if w != nil {
				m.registry.Register(w)
			}
		}
	}

	return m.validateNoPortConflicts()
}

func (m *Manager) createBaseProtocol(id ProtocolID) BaseProtocol {
	for _, fn := range baseFactories {
		if p := fn(id); p != nil {
			return p
		}
	}
	return nil
}

func (m *Manager) createStandaloneService(id ProtocolID) StandaloneService {
	for _, fn := range standaloneFactories {
		if s := fn(id); s != nil {
			return s
		}
	}
	return nil
}

func (m *Manager) createWrapper(id ProtocolID) TransportWrapper {
	for _, fn := range wrapperFactories {
		if w := fn(id); w != nil {
			return w
		}
	}
	return nil
}

func (m *Manager) validateNoPortConflicts() error {
	usedPorts := make(map[int]ProtocolID)

	for _, id := range m.registry.ListAll() {
		p := m.registry.Get(id)
		if p == nil {
			continue
		}

		if bp, ok := p.(BaseProtocol); ok {
			port := bp.Port()
			if port > 0 {
				if existing, ok := usedPorts[port]; ok {
					return fmt.Errorf("port %d conflict between %s and %s", port, id, existing)
				}
				usedPorts[port] = id
			}
		}
	}

	return nil
}

func (m *Manager) StartProtocol(id ProtocolID) error {
	p := m.registry.Get(id)
	if p == nil {
		return ErrProtocolNotFound
	}
	return p.Start()
}

func (m *Manager) StopProtocol(id ProtocolID) error {
	p := m.registry.Get(id)
	if p == nil {
		return ErrProtocolNotFound
	}
	return p.Stop()
}

func (m *Manager) RestartProtocol(id ProtocolID) error {
	p := m.registry.Get(id)
	if p == nil {
		return ErrProtocolNotFound
	}
	return p.Restart()
}

func (m *Manager) GetProtocolStatus(id ProtocolID) (Status, error) {
	p := m.registry.Get(id)
	if p == nil {
		return StatusUnknown, ErrProtocolNotFound
	}
	return p.Status(), nil
}

func (m *Manager) GetProtocolConfig(id ProtocolID) (any, error) {
	p := m.registry.Get(id)
	if p == nil {
		return nil, ErrProtocolNotFound
	}
	return p.Config()
}

func (m *Manager) ApplyProtocolConfig(id ProtocolID, cfg any) error {
	p := m.registry.Get(id)
	if p == nil {
		return ErrProtocolNotFound
	}
	return p.ApplyConfig(cfg)
}

func (m *Manager) ListProtocols() []ProtocolID {
	return m.registry.ListAll()
}

func (m *Manager) ListByCategory(cat Category) []ProtocolID {
	return m.registry.ListByCategory(cat)
}

func (m *Manager) GetSupportedWrappers(baseID ProtocolID) []ProtocolID {
	var result []ProtocolID
	for _, id := range m.registry.ListByCategory(CategoryWrapper) {
		w := m.registry.GetWrapper(id)
		if w == nil {
			continue
		}
		for _, supported := range w.SupportedProtocols() {
			if supported == baseID {
				result = append(result, id)
				break
			}
		}
	}
	return result
}
