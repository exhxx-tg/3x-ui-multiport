package protocol

import (
	"os"
	"sync"
)

var (
	globalManager *Manager
	globalOnce    sync.Once
)

func InitGlobal() error {
	var initErr error
	globalOnce.Do(func() {
		reg := Global()
		mgr := NewManager(reg)
		if err := mgr.Initialize(); err != nil {
			initErr = err
			return
		}
		globalManager = mgr
		if os.Getenv("XUI_DEV") != "" {
			reg.mu.RLock()
			count := len(reg.protocols)
			reg.mu.RUnlock()
			println("protocol: initialized", count, "protocols in global registry")
		}
	})
	return initErr
}

func GlobalManager() *Manager {
	return globalManager
}

func IsInitialized() bool {
	return globalManager != nil
}
