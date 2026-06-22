package servicemanager

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
)

type ServiceDefinition struct {
	Name        string
	DisplayName string
	BinaryName  string
	BinaryPath  string
	ConfigPath  string
	LogPath     string
	DefaultPort int
	Args        []string
	InstallHint string
}

type ServiceManager struct {
	mu       sync.RWMutex
	services map[string]*ServiceInstance
}

type ServiceInstance struct {
	Def     ServiceDefinition
	Process *Process
	State   State
}

var global *ServiceManager

func init() {
	global = NewServiceManager()
}

func NewServiceManager() *ServiceManager {
	return &ServiceManager{
		services: make(map[string]*ServiceInstance),
	}
}

func Global() *ServiceManager {
	return global
}

func (sm *ServiceManager) Register(def ServiceDefinition) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	binPath := def.BinaryPath
	if binPath == "" && def.BinaryName != "" {
		if p, err := execLookPath(def.BinaryName); err == nil {
			binPath = p
		}
	}

	info := ProcessInfo{
		BinaryPath: binPath,
		ConfigPath: def.ConfigPath,
		LogPath:    def.LogPath,
		Args:       def.Args,
	}

	svc := &ServiceInstance{
		Def:     def,
		Process: NewProcess(info),
		State:   StateUnknown,
	}
	sm.services[def.Name] = svc
}

func execLookPath(file string) (string, error) {
	return exec.LookPath(file)
}

func (sm *ServiceManager) Get(name string) (*ServiceInstance, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	svc, ok := sm.services[name]
	if !ok {
		return nil, fmt.Errorf("service %s not found", name)
	}
	return svc, nil
}

func (sm *ServiceManager) Start(name string) error {
	svc, err := sm.Get(name)
	if err != nil {
		return err
	}

	if HasSystemd() {
		if state, _ := SystemdUnitStatus(name); state == StateRunning {
			return nil
		}
		if err := exec.Command("systemctl", "start", name).Run(); err == nil {
			svc.State = StateRunning
			return nil
		}
	}

	if err := svc.Process.Start(); err != nil {
		svc.State = StateError
		return err
	}
	svc.State = StateRunning
	return nil
}

func (sm *ServiceManager) Stop(name string) error {
	svc, err := sm.Get(name)
	if err != nil {
		return err
	}

	if HasSystemd() {
		if err := exec.Command("systemctl", "stop", name).Run(); err == nil {
			svc.State = StateStopped
			return nil
		}
	}

	if err := svc.Process.Stop(); err != nil {
		return err
	}
	svc.State = StateStopped
	return nil
}

func (sm *ServiceManager) Restart(name string) error {
	_ = sm.Stop(name)
	return sm.Start(name)
}

func (sm *ServiceManager) Status(name string) State {
	svc, err := sm.Get(name)
	if err != nil {
		return StateUnknown
	}

	if HasSystemd() {
		if state, err := SystemdUnitStatus(name); err == nil && state != StateUnknown {
			svc.State = state
			return state
		}
	}

	if svc.Process != nil && svc.Process.IsRunning() {
		svc.State = StateRunning
		return StateRunning
	}

	if svc.Def.BinaryPath != "" {
		if _, err := os.Stat(svc.Def.BinaryPath); os.IsNotExist(err) {
			svc.State = StateError
			return StateError
		}
	}

	svc.State = StateStopped
	return StateStopped
}

func (sm *ServiceManager) IsInstalled(name string) bool {
	svc, err := sm.Get(name)
	if err != nil {
		return false
	}
	if HasSystemd() {
		path := fmt.Sprintf("/etc/systemd/system/%s.service", name)
		if _, err := os.Stat(path); err == nil {
			return true
		}
	}
	if svc.Def.BinaryPath != "" {
		if _, err := os.Stat(svc.Def.BinaryPath); err == nil {
			return true
		}
	}
	if svc.Def.BinaryName != "" {
		if _, err := exec.LookPath(svc.Def.BinaryName); err == nil {
			return true
		}
	}
	return false
}

func (sm *ServiceManager) Install(name string) error {
	svc, err := sm.Get(name)
	if err != nil {
		return err
	}
	if sm.IsInstalled(name) {
		return nil
	}

	if HasSystemd() && svc.Def.BinaryPath != "" {
		unit := DefaultSystemdUnit(name, svc.Def.BinaryPath)
		unit.Description = svc.Def.DisplayName
		unit.Args = svc.Def.Args
		if svc.Def.LogPath != "" {
			logDir := filepath.Dir(svc.Def.LogPath)
			if err := os.MkdirAll(logDir, 0755); err != nil {
				return fmt.Errorf("failed to create log dir: %w", err)
			}
		}
		return InstallSystemdUnit(unit)
	}

	_ = os.MkdirAll(filepath.Dir(svc.Def.ConfigPath), 0755)
	return nil
}

func (sm *ServiceManager) Uninstall(name string) error {
	_ = sm.Stop(name)
	if HasSystemd() {
		return UninstallSystemdUnit(name)
	}
	return nil
}

func (sm *ServiceManager) List() []string {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	names := make([]string, 0, len(sm.services))
	for n := range sm.services {
		names = append(names, n)
	}
	return names
}

func (sm *ServiceManager) PID(name string) int {
	svc, err := sm.Get(name)
	if err != nil || svc.Process == nil {
		return 0
	}
	return svc.Process.PID()
}

func EnsureBinDir() error {
	if runtime.GOOS == "windows" {
		return nil
	}
	dirs := []string{
		"/usr/local/bin",
		"/usr/bin",
		"/opt/x-ui-pro/bin",
	}
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0755); err == nil {
			return nil
		}
	}
	return os.MkdirAll("/opt/x-ui-pro/bin", 0755)
}
