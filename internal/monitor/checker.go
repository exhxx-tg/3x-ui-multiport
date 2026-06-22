package monitor

import (
	"context"
	"fmt"
	"net"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/exhxx-tg/3x-ui-multiport/internal/logger"
	xuiProtocol "github.com/exhxx-tg/3x-ui-multiport/internal/protocol"
	"github.com/exhxx-tg/3x-ui-multiport/internal/servicemanager"
)

type HealthChecker struct {
	registry *xuiProtocol.Registry
	sm       *servicemanager.ServiceManager
	mu       sync.RWMutex
	results  map[string]*CheckResult
	interval time.Duration
}

func NewHealthChecker(registry *xuiProtocol.Registry, sm *servicemanager.ServiceManager) *HealthChecker {
	return &HealthChecker{
		registry: registry,
		sm:       sm,
		results:  make(map[string]*CheckResult),
		interval: 30 * time.Second,
	}
}

func (hc *HealthChecker) SetInterval(d time.Duration) {
	hc.mu.Lock()
	defer hc.mu.Unlock()
	hc.interval = d
}

func (hc *HealthChecker) Interval() time.Duration {
	hc.mu.RLock()
	defer hc.mu.RUnlock()
	return hc.interval
}

func (hc *HealthChecker) CheckProtocol(ctx context.Context, id xuiProtocol.ProtocolID) CheckResult {
	start := time.Now()
	result := CheckResult{
		ProtocolID: string(id),
		CheckedAt:  start,
		Details:    make(map[string]any),
	}

	p := hc.registry.Get(id)
	if p == nil {
		result.Error = "protocol not registered"
		return result
	}

	status := p.Status()
	result.Details["status"] = string(status)

	switch status {
	case xuiProtocol.StatusRunning:
		if ss, ok := p.(xuiProtocol.StandaloneService); ok {
			if err := ss.HealthCheck(); err != nil {
				result.Error = err.Error()
				return result
			}
		}
		result.Healthy = true
		result.Latency = time.Since(start)
		return result

	case xuiProtocol.StatusStopped, xuiProtocol.StatusUnknown:
		result.Healthy = false
		return result

	case xuiProtocol.StatusError:
		result.Error = "protocol in error state"
		result.Healthy = false
		return result
	}

	result.Healthy = false
	result.Error = fmt.Sprintf("unexpected status: %s", status)
	return result
}

func (hc *HealthChecker) CheckAll(ctx context.Context) HealthSummary {
	summary := HealthSummary{
		GeneratedAt: time.Now(),
	}

	var mu sync.Mutex
	var wg sync.WaitGroup

	for _, info := range xuiProtocol.AllProtocols {
		wg.Add(1)
		go func(id xuiProtocol.ProtocolID) {
			defer wg.Done()
			result := hc.CheckProtocol(ctx, id)

			mu.Lock()
			summary.Results = append(summary.Results, result)
			summary.TotalProtocols++
			if result.Healthy {
				summary.Healthy++
			} else if result.Error != "" || !result.Healthy {
				summary.Unhealthy++
			} else {
				summary.Unknown++
			}
			hc.results[string(id)] = &result
			mu.Unlock()
		}(info.ID)
	}

	wg.Wait()
	return summary
}

func (hc *HealthChecker) GetLastResult(id xuiProtocol.ProtocolID) *CheckResult {
	hc.mu.RLock()
	defer hc.mu.RUnlock()
	return hc.results[string(id)]
}

func (hc *HealthChecker) GetAllResults() []CheckResult {
	hc.mu.RLock()
	defer hc.mu.RUnlock()
	out := make([]CheckResult, 0, len(hc.results))
	for _, r := range hc.results {
		out = append(out, *r)
	}
	return out
}

func TCPPortCheck(host string, port int, timeout time.Duration) bool {
	addr := net.JoinHostPort(host, fmt.Sprintf("%d", port))
	conn, err := net.DialTimeout("tcp", addr, timeout)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

func cmdHealthCheck(binary string, args ...string) error {
	if runtime.GOOS == "windows" {
		bin := strings.ReplaceAll(binary, "/", "\\")
		if _, err := exec.LookPath(bin); err != nil {
			return fmt.Errorf("binary not found: %w", err)
		}
		return nil
	}
	cmd := exec.Command(binary, args...)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("health cmd failed: %w", err)
	}
	return nil
}

func getCheckInterval(intervalStr string) time.Duration {
	switch strings.ToLower(intervalStr) {
	case "fast":
		return 10 * time.Second
	case "normal":
		return 30 * time.Second
	case "slow":
		return 60 * time.Second
	default:
		d, err := time.ParseDuration(intervalStr)
		if err != nil {
			return 30 * time.Second
		}
		return d
	}
}

func (hc *HealthChecker) SyncResult(id xuiProtocol.ProtocolID) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	result := hc.CheckProtocol(ctx, id)
	hc.mu.Lock()
	hc.results[string(id)] = &result
	hc.mu.Unlock()

	logger.Debugf("health check [%s]: healthy=%v latency=%v", id, result.Healthy, result.Latency)
}

// AutoRecover attempts to restart protocols that are unhealthy.
// Returns the number of protocols successfully recovered.
func (hc *HealthChecker) AutoRecover() int {
	results := hc.GetAllResults()
	recovered := 0

	for _, r := range results {
		if r.Healthy {
			continue
		}
		id := xuiProtocol.ProtocolID(r.ProtocolID)
		p := hc.registry.Get(id)
		if p == nil {
			continue
		}
		logger.Infof("auto-recovery: attempting restart of protocol [%s]", id)
		if err := p.Restart(); err != nil {
			logger.Warningf("auto-recovery: restart of [%s] failed: %v", id, err)
			continue
		}
		ResetUptime(id)
		recovered++
		logger.Infof("auto-recovery: successfully restarted protocol [%s]", id)
	}

	return recovered
}

// GetUnhealthyProtocols returns IDs of protocols that are currently unhealthy.
func (hc *HealthChecker) GetUnhealthyProtocols() []xuiProtocol.ProtocolID {
	results := hc.GetAllResults()
	var ids []xuiProtocol.ProtocolID
	for _, r := range results {
		if !r.Healthy {
			ids = append(ids, xuiProtocol.ProtocolID(r.ProtocolID))
		}
	}
	return ids
}
