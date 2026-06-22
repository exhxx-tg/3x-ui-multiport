package protocol

import (
	"testing"
)

func TestAllProtocolsDefined(t *testing.T) {
	if len(AllProtocols) != 13 {
		t.Errorf("expected 13 protocols in ecosystem, got %d", len(AllProtocols))
	}
}

func TestAllProtocolsUnique(t *testing.T) {
	seen := make(map[ProtocolID]bool)
	for _, p := range AllProtocols {
		if seen[p.ID] {
			t.Errorf("duplicate protocol ID: %s", p.ID)
		}
		seen[p.ID] = true
	}
}

func TestCategoryCounts(t *testing.T) {
	base := GetByCategory(CategoryBase)
	standalone := GetByCategory(CategoryStandalone)
	wrappers := GetByCategory(CategoryWrapper)

	if len(base) != 5 {
		t.Errorf("expected 5 base protocols, got %d", len(base))
	}
	if len(standalone) != 3 {
		t.Errorf("expected 3 standalone services, got %d", len(standalone))
	}
	if len(wrappers) != 5 {
		t.Errorf("expected 5 transport wrappers, got %d", len(wrappers))
	}
}

func TestGetProtocolInfo(t *testing.T) {
	info := GetProtocolInfo(ProtocolVMess)
	if info == nil {
		t.Fatal("expected VMess info")
	}
	if info.ID != ProtocolVMess {
		t.Errorf("expected ID vmess, got %s", info.ID)
	}
	if info.Category != CategoryBase {
		t.Errorf("expected category base, got %s", info.Category)
	}

	info = GetProtocolInfo(ProtocolOpenVPN)
	if info == nil {
		t.Fatal("expected OpenVPN info")
	}
	if info.Category != CategoryStandalone {
		t.Errorf("expected category standalone, got %s", info.Category)
	}

	info = GetProtocolInfo(WrapperWebSocket)
	if info == nil {
		t.Fatal("expected WebSocket wrapper info")
	}
	if info.Category != CategoryWrapper {
		t.Errorf("expected category wrapper, got %s", info.Category)
	}
}

func TestGetXrayNativeProtocols(t *testing.T) {
	for _, p := range AllProtocols {
		if p.Category == CategoryBase && !p.XrayNative {
			t.Errorf("%s is CategoryBase but not XrayNative", p.ID)
		}
		if p.Category == CategoryWrapper && p.ID != WrapperNaive && !p.XrayNative {
			t.Errorf("%s is CategoryWrapper but not XrayNative", p.ID)
		}
	}
}

func TestProtocolInfoConsistency(t *testing.T) {
	for _, p := range AllProtocols {
		info := GetProtocolInfo(p.ID)
		if info == nil {
			t.Errorf("GetProtocolInfo returned nil for %s", p.ID)
			continue
		}
		if info.Name == "" {
			t.Errorf("%s has empty Name", p.ID)
		}
		if info.Description == "" {
			t.Errorf("%s has empty Description", p.ID)
		}
		if info.Source == "" {
			t.Errorf("%s has empty Source", p.ID)
		}
	}
}

func TestRegistryLifecycle(t *testing.T) {
	reg := NewRegistry()

	if len(reg.List()) != 0 {
		t.Error("expected empty registry")
	}

	p := &mockProtocol{id: ProtocolVMess}
	reg.Register(p)

	if reg.Get(ProtocolVMess) == nil {
		t.Error("expected protocol to be registered")
	}

	reg.Unregister(ProtocolVMess)
	if reg.Get(ProtocolVMess) != nil {
		t.Error("expected protocol to be unregistered")
	}
}

func TestManagerInitialization(t *testing.T) {
	reg := NewRegistry()
	mgr := NewManager(reg)

	if err := mgr.Initialize(); err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	allIDs := reg.ListAll()
	if len(allIDs) != 13 {
		t.Errorf("expected 13 protocols after init, got %d: %v", len(allIDs), allIDs)
	}
}

func TestManagerInitializationIdempotent(t *testing.T) {
	reg := NewRegistry()
	mgr := NewManager(reg)

	if err := mgr.Initialize(); err != nil {
		t.Fatalf("first Initialize failed: %v", err)
	}

	if err := mgr.Initialize(); err != nil {
		t.Fatalf("second Initialize failed (not idempotent): %v", err)
	}

	if len(reg.ListAll()) != 13 {
		t.Errorf("expected 13 protocols, got %d", len(reg.ListAll()))
	}
}

func TestGlobalRegistrySingleton(t *testing.T) {
	g1 := Global()
	g2 := Global()

	if g1 != g2 {
		t.Error("Global() should return the same instance")
	}
}

func TestGlobalManagerSingleton(t *testing.T) {
	if err := InitGlobal(); err != nil {
		t.Fatalf("InitGlobal failed: %v", err)
	}

	mgr1 := GlobalManager()
	mgr2 := GlobalManager()

	if mgr1 == nil {
		t.Fatal("GlobalManager() should not be nil after InitGlobal")
	}
	if mgr1 != mgr2 {
		t.Error("GlobalManager() should return the same instance")
	}
}

func TestProtocolStatusConstants(t *testing.T) {
	statuses := []Status{StatusRunning, StatusStopped, StatusError, StatusInstalling, StatusUnknown}
	for _, s := range statuses {
		if s == "" {
			t.Error("status constant should not be empty")
		}
	}
}

func TestEcosystemVersion(t *testing.T) {
	if EcosystemVersion != "1.0.0" {
		t.Errorf("expected EcosystemVersion 1.0.0, got %s", EcosystemVersion)
	}
}

func TestBaseProtocolIDs(t *testing.T) {
	baseIDs := []ProtocolID{
		ProtocolVMess,
		ProtocolVLESS,
		ProtocolTrojan,
		ProtocolShadowsocks,
		ProtocolHysteria,
	}
	for _, id := range baseIDs {
		info := GetProtocolInfo(id)
		if info == nil {
			t.Errorf("no info for base protocol %s", id)
			continue
		}
		if info.Category != CategoryBase {
			t.Errorf("%s should be CategoryBase, got %s", id, info.Category)
		}
		if !info.XrayNative {
			t.Errorf("%s should be XrayNative", id)
		}
	}
}

func TestStandaloneProtocolIDs(t *testing.T) {
	saIDs := []ProtocolID{
		ProtocolOpenVPN,
		ProtocolWireGuard,
		ProtocolDropbear,
	}
	for _, id := range saIDs {
		info := GetProtocolInfo(id)
		if info == nil {
			t.Errorf("no info for standalone protocol %s", id)
			continue
		}
		if info.Category != CategoryStandalone {
			t.Errorf("%s should be CategoryStandalone, got %s", id, info.Category)
		}
		if info.XrayNative {
			t.Errorf("%s should NOT be XrayNative", id)
		}
	}
}

func TestWrapperProtocolIDs(t *testing.T) {
	wrapperIDs := []ProtocolID{
		WrapperWebSocket,
		WrapperTLS,
		WrapperHTTP2,
		WrapperGRPC,
		WrapperNaive,
	}
	for _, id := range wrapperIDs {
		info := GetProtocolInfo(id)
		if info == nil {
			t.Errorf("no info for wrapper protocol %s", id)
			continue
		}
		if info.Category != CategoryWrapper {
			t.Errorf("%s should be CategoryWrapper, got %s", id, info.Category)
		}
	}
}

func TestListByCategory(t *testing.T) {
	reg := NewRegistry()

	reg.Register(&mockProtocol{id: ProtocolVMess})
	reg.Register(&mockProtocol{id: ProtocolVLESS})
	reg.Register(&mockProtocol{id: ProtocolOpenVPN})

	base := reg.ListByCategory(CategoryBase)
	if len(base) != 2 {
		t.Errorf("expected 2 base protocols, got %d", len(base))
	}

	sa := reg.ListByCategory(CategoryStandalone)
	if len(sa) != 1 {
		t.Errorf("expected 1 standalone, got %d", len(sa))
	}

	all := reg.ListAll()
	if len(all) != 13 {
		t.Errorf("expected 13 total known, got %d", len(all))
	}
}

type mockProtocol struct {
	id     ProtocolID
	status Status
	cat    Category
}

func (m *mockProtocol) ID() ProtocolID          { return m.id }
func (m *mockProtocol) Info() ProtocolInfo      { return ProtocolInfo{ID: m.id, Name: string(m.id), Category: m.cat} }
func (m *mockProtocol) Status() Status          { return m.status }
func (m *mockProtocol) Start() error            { m.status = StatusRunning; return nil }
func (m *mockProtocol) Stop() error             { m.status = StatusStopped; return nil }
func (m *mockProtocol) Restart() error          { return m.Start() }
func (m *mockProtocol) Config() (any, error)    { return nil, nil }
func (m *mockProtocol) ApplyConfig(any) error   { return nil }

type mockWrappedProtocol struct {
	id     ProtocolID
	status Status
}

func (m *mockWrappedProtocol) ID() ProtocolID             { return m.id }
func (m *mockWrappedProtocol) Info() ProtocolInfo         { return ProtocolInfo{ID: m.id, Name: string(m.id)} }
func (m *mockWrappedProtocol) Status() Status             { return m.status }
func (m *mockWrappedProtocol) Start() error               { m.status = StatusRunning; return nil }
func (m *mockWrappedProtocol) Stop() error                { m.status = StatusStopped; return nil }
func (m *mockWrappedProtocol) Restart() error             { return m.Start() }
func (m *mockWrappedProtocol) Config() (any, error)       { return nil, nil }
func (m *mockWrappedProtocol) ApplyConfig(any) error      { return nil }
func (m *mockWrappedProtocol) Port() int                  { return 0 }
func (m *mockWrappedProtocol) SetPort(int)                {}

type mockWrapperProtocol struct {
	id        ProtocolID
	status    Status
	supported []ProtocolID
}

func (m *mockWrapperProtocol) ID() ProtocolID                        { return m.id }
func (m *mockWrapperProtocol) Info() ProtocolInfo                    { return ProtocolInfo{ID: m.id, Name: string(m.id), Category: CategoryWrapper} }
func (m *mockWrapperProtocol) Status() Status                        { return m.status }
func (m *mockWrapperProtocol) Start() error                          { m.status = StatusRunning; return nil }
func (m *mockWrapperProtocol) Stop() error                           { m.status = StatusStopped; return nil }
func (m *mockWrapperProtocol) Restart() error                        { return m.Start() }
func (m *mockWrapperProtocol) Config() (any, error)                  { return nil, nil }
func (m *mockWrapperProtocol) ApplyConfig(any) error                 { return nil }
func (m *mockWrapperProtocol) SupportedProtocols() []ProtocolID      { return m.supported }
func (m *mockWrapperProtocol) WrapConfig(any, any) (any, error)      { return nil, nil }

func TestManagerProtocolOperations(t *testing.T) {
	reg := NewRegistry()
	mgr := NewManager(reg)

	p := &mockProtocol{id: ProtocolVMess}
	reg.Register(p)

	if err := mgr.StartProtocol(ProtocolVMess); err != nil {
		t.Fatalf("StartProtocol failed: %v", err)
	}
	if p.status != StatusRunning {
		t.Error("expected running status")
	}

	status, err := mgr.GetProtocolStatus(ProtocolVMess)
	if err != nil {
		t.Fatalf("GetProtocolStatus failed: %v", err)
	}
	if status != StatusRunning {
		t.Errorf("expected StatusRunning, got %s", status)
	}

	if err := mgr.StopProtocol(ProtocolVMess); err != nil {
		t.Fatalf("StopProtocol failed: %v", err)
	}
	if p.status != StatusStopped {
		t.Error("expected stopped status")
	}

	_, err = mgr.GetProtocolStatus("nonexistent")
	if err != ErrProtocolNotFound {
		t.Errorf("expected ErrProtocolNotFound, got %v", err)
	}
}

func TestManagerListAndCategory(t *testing.T) {
	reg := NewRegistry()
	reg.Register(&mockProtocol{id: ProtocolVMess, cat: CategoryBase})
	reg.Register(&mockProtocol{id: ProtocolTrojan, cat: CategoryBase})
	reg.Register(&mockProtocol{id: ProtocolOpenVPN, cat: CategoryStandalone})
	mgr := NewManager(reg)

	all := mgr.ListProtocols()
	if len(all) != 13 {
		t.Errorf("expected 13 all known, got %d", len(all))
	}

	base := mgr.ListByCategory(CategoryBase)
	if len(base) != 2 {
		t.Errorf("expected 2 base, got %d", len(base))
	}

	sa := mgr.ListByCategory(CategoryStandalone)
	if len(sa) != 1 {
		t.Errorf("expected 1 standalone, got %d", len(sa))
	}
}

func TestGetSupportedWrappers(t *testing.T) {
	reg := NewRegistry()
	mgr := NewManager(reg)

	reg.Register(&mockWrappedProtocol{id: ProtocolVLESS})
	reg.Register(&mockWrapperProtocol{
		id:        WrapperWebSocket,
		supported: []ProtocolID{ProtocolVLESS, ProtocolVMess},
	})

	vlessWrappers := mgr.GetSupportedWrappers(ProtocolVLESS)
	if len(vlessWrappers) == 0 {
		t.Error("VLESS should have supported wrappers")
	}

	hasWS := false
	for _, w := range vlessWrappers {
		if w == WrapperWebSocket {
			hasWS = true
			break
		}
	}
	if !hasWS {
		t.Error("VLESS should support WebSocket wrapper")
	}
}
