package protocol

import "sync"

type Registry struct {
	mu                sync.RWMutex
	protocols         map[ProtocolID]Protocol
	standalone        map[ProtocolID]StandaloneService
	wrappers          map[ProtocolID]TransportWrapper
	baseProtocols     map[ProtocolID]BaseProtocol
}

var globalRegistry *Registry

func init() {
	globalRegistry = NewRegistry()
}

func NewRegistry() *Registry {
	return &Registry{
		protocols:     make(map[ProtocolID]Protocol),
		standalone:    make(map[ProtocolID]StandaloneService),
		wrappers:      make(map[ProtocolID]TransportWrapper),
		baseProtocols: make(map[ProtocolID]BaseProtocol),
	}
}

func Global() *Registry {
	return globalRegistry
}

func (r *Registry) Register(p Protocol) {
	r.mu.Lock()
	defer r.mu.Unlock()
	id := p.ID()
	r.protocols[id] = p

	if bp, ok := p.(BaseProtocol); ok {
		r.baseProtocols[id] = bp
	}
	if ss, ok := p.(StandaloneService); ok {
		r.standalone[id] = ss
	}
	if tw, ok := p.(TransportWrapper); ok {
		r.wrappers[id] = tw
	}
}

func (r *Registry) Get(id ProtocolID) Protocol {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.protocols[id]
}

func (r *Registry) GetBase(id ProtocolID) BaseProtocol {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.baseProtocols[id]
}

func (r *Registry) GetStandalone(id ProtocolID) StandaloneService {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.standalone[id]
}

func (r *Registry) GetWrapper(id ProtocolID) TransportWrapper {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.wrappers[id]
}

func (r *Registry) List() []ProtocolID {
	r.mu.RLock()
	defer r.mu.RUnlock()
	ids := make([]ProtocolID, 0, len(r.protocols))
	for id := range r.protocols {
		ids = append(ids, id)
	}
	return ids
}

func (r *Registry) ListByCategory(cat Category) []ProtocolID {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var result []ProtocolID
	for _, p := range AllProtocols {
		if p.Category == cat {
			if _, ok := r.protocols[p.ID]; ok {
				result = append(result, p.ID)
			}
		}
	}
	return result
}

func (r *Registry) ListAll() []ProtocolID {
	ids := make([]ProtocolID, len(AllProtocols))
	for i, p := range AllProtocols {
		ids[i] = p.ID
	}
	return ids
}

func (r *Registry) Unregister(id ProtocolID) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.protocols, id)
	delete(r.baseProtocols, id)
	delete(r.standalone, id)
	delete(r.wrappers, id)
}
