package servicemanager

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/exhxx-tg/3x-ui-multiport/internal/logger"
)

var (
	ErrNotInstalled   = errors.New("service not installed")
	ErrAlreadyRunning = errors.New("service already running")
	ErrNotRunning     = errors.New("service not running")
	ErrTimeout        = errors.New("service operation timed out")
)

type State string

const (
	StateRunning    State = "running"
	StateStopped    State = "stopped"
	StateError      State = "error"
	StateInstalling State = "installing"
	StateUnknown    State = "unknown"
)

type ProcessInfo struct {
	PID         int
	BinaryPath  string
	ConfigPath  string
	LogPath     string
	Args        []string
	Env         []string
	WorkDir     string
}

type Process struct {
	info     ProcessInfo
	cmd      *exec.Cmd
	done     chan struct{}
	mu       sync.RWMutex
	state    atomic.Value
	exitErr  error
	stopFlag atomic.Bool
}

func NewProcess(info ProcessInfo) *Process {
	p := &Process{
		info: info,
		done: make(chan struct{}),
	}
	p.state.Store(StateStopped)
	return p
}

func (p *Process) State() State {
	return p.state.Load().(State)
}

func (p *Process) PID() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if p.cmd != nil && p.cmd.Process != nil {
		return p.cmd.Process.Pid
	}
	return 0
}

func (p *Process) IsRunning() bool {
	p.mu.RLock()
	cmd, done := p.cmd, p.done
	p.mu.RUnlock()
	if cmd == nil || cmd.Process == nil {
		return false
	}
	if done != nil {
		select {
		case <-done:
			return false
		default:
		}
	}
	return true
}

func (p *Process) Start() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.IsRunning() {
		return ErrAlreadyRunning
	}

	if p.info.BinaryPath != "" {
		if _, err := os.Stat(p.info.BinaryPath); os.IsNotExist(err) {
			return fmt.Errorf("%w: binary %s not found", ErrNotInstalled, p.info.BinaryPath)
		}
	}

	p.done = make(chan struct{})
	p.stopFlag.Store(false)
	p.exitErr = nil

	cmd := exec.Command(p.info.BinaryPath, p.info.Args...)
	if p.info.WorkDir != "" {
		cmd.Dir = p.info.WorkDir
	}
	if len(p.info.Env) > 0 {
		cmd.Env = append(os.Environ(), p.info.Env...)
	}

	if p.info.LogPath != "" {
		logDir := filepath.Dir(p.info.LogPath)
		if err := os.MkdirAll(logDir, 0755); err != nil {
			return fmt.Errorf("failed to create log dir: %w", err)
		}
		f, err := os.OpenFile(p.info.LogPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return fmt.Errorf("failed to open log file: %w", err)
		}
		cmd.Stdout = f
		cmd.Stderr = f
	}

	attachChildLifetime(cmd)

	if err := cmd.Start(); err != nil {
		p.state.Store(StateError)
		return fmt.Errorf("failed to start process: %w", err)
	}

	p.cmd = cmd
	p.state.Store(StateRunning)

	go p.waitForCommand(cmd)
	return nil
}

func (p *Process) Stop() error {
	p.mu.Lock()
	cmd := p.cmd
	p.mu.Unlock()

	if cmd == nil || cmd.Process == nil {
		return ErrNotRunning
	}

	p.stopFlag.Store(true)

	if runtime.GOOS != "windows" {
		_ = cmd.Process.Signal(syscall.SIGTERM)
		done := make(chan struct{})
		go func() {
			select {
			case <-p.done:
			case <-time.After(5 * time.Second):
			}
			close(done)
		}()
		<-done
	}

	if p.IsRunning() {
		_ = cmd.Process.Kill()
	}

	<-p.done
	return nil
}

func (p *Process) Restart() error {
	_ = p.Stop()
	time.Sleep(500 * time.Millisecond)
	return p.Start()
}

func (p *Process) Signal(sig os.Signal) error {
	p.mu.RLock()
	cmd := p.cmd
	p.mu.RUnlock()
	if cmd == nil || cmd.Process == nil {
		return ErrNotRunning
	}
	return cmd.Process.Signal(sig)
}

func (p *Process) ExitErr() error {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.exitErr
}

func (p *Process) waitForCommand(cmd *exec.Cmd) {
	err := cmd.Wait()
	p.mu.Lock()
	p.exitErr = err
	p.mu.Unlock()

	if !p.stopFlag.Load() && err != nil {
		p.state.Store(StateError)
		logger.Warningf("process %s exited unexpectedly: %v", p.info.BinaryPath, err)
	} else {
		p.state.Store(StateStopped)
	}
	close(p.done)
}

func (p *Process) Info() ProcessInfo {
	return p.info
}
