package performance

import (
	"context"
	"fmt"
	"math"
	"sync"
	"sync/atomic"
	"time"

	"github.com/exhxx-tg/3x-ui-multiport/internal/logger"
)

type LoadProfile struct {
	Name        string        `json:"name"`
	Concurrency int           `json:"concurrency"`
	Duration    time.Duration `json:"duration"`
	RampUp      time.Duration `json:"rampUp"`
	MaxErrors   int64         `json:"maxErrors"`
}

type LoadTestResult struct {
	Name           string        `json:"name"`
	Concurrency    int           `json:"concurrency"`
	Duration       time.Duration `json:"duration"`
	TotalOps       int64         `json:"totalOps"`
	SuccessOps     int64         `json:"successOps"`
	ErrorOps       int64         `json:"errorOps"`
	OpsPerSecond   float64       `json:"opsPerSecond"`
	AvgLatency     time.Duration `json:"avgLatency"`
	P50Latency     time.Duration `json:"p50Latency"`
	P90Latency     time.Duration `json:"p90Latency"`
	P99Latency     time.Duration `json:"p99Latency"`
	MaxLatency     time.Duration `json:"maxLatency"`
	MinLatency     time.Duration `json:"minLatency"`
}

type LoadTester struct {
	profile    LoadProfile
	latencies  []time.Duration
	mu         sync.Mutex
	successOps atomic.Int64
	errorOps   atomic.Int64
	startTime  time.Time
}

func NewLoadTester(profile LoadProfile) *LoadTester {
	return &LoadTester{
		profile:   profile,
		latencies: make([]time.Duration, 0, 100000),
	}
}

func (lt *LoadTester) Run(operation func(ctx context.Context) error) LoadTestResult {
	ctx, cancel := context.WithTimeout(context.Background(), lt.profile.Duration+lt.profile.RampUp)
	defer cancel()

	lt.startTime = time.Now()

	var wg sync.WaitGroup
	concurrencyCh := make(chan struct{}, 1)

	rampStep := time.Duration(0)
	if lt.profile.RampUp > 0 && lt.profile.Concurrency > 1 {
		rampStep = lt.profile.RampUp / time.Duration(lt.profile.Concurrency)
	}

	for i := 0; i < lt.profile.Concurrency; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			if rampStep > 0 {
				time.Sleep(rampStep * time.Duration(id))
			}

			lt.worker(ctx, concurrencyCh, operation)
		}(i)
	}

	wg.Wait()
	cancel()

	return lt.computeResults()
}

func (lt *LoadTester) worker(ctx context.Context, ch chan struct{}, operation func(ctx context.Context) error) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		if lt.profile.MaxErrors > 0 && lt.errorOps.Load() >= lt.profile.MaxErrors {
			return
		}

		start := time.Now()
		err := operation(ctx)
		latency := time.Since(start)

		lt.addLatency(latency)

		if err != nil {
			lt.errorOps.Add(1)
		} else {
			lt.successOps.Add(1)
		}
	}
}

func (lt *LoadTester) addLatency(latency time.Duration) {
	lt.mu.Lock()
	lt.latencies = append(lt.latencies, latency)
	lt.mu.Unlock()
}

func (lt *LoadTester) computeResults() LoadTestResult {
	elapsed := time.Since(lt.startTime)
	totalOps := lt.successOps.Load() + lt.errorOps.Load()
	opsPerSec := float64(0)
	if elapsed.Seconds() > 0 {
		opsPerSec = float64(totalOps) / elapsed.Seconds()
	}

	lt.mu.Lock()
	lats := make([]time.Duration, len(lt.latencies))
	copy(lats, lt.latencies)
	lt.mu.Unlock()

	result := LoadTestResult{
		Name:         lt.profile.Name,
		Concurrency:  lt.profile.Concurrency,
		Duration:     elapsed,
		TotalOps:     totalOps,
		SuccessOps:   lt.successOps.Load(),
		ErrorOps:     lt.errorOps.Load(),
		OpsPerSecond: opsPerSec,
	}

	if len(lats) == 0 {
		return result
	}

	sortDurations(lats)

	var totalLatency time.Duration
	for _, l := range lats {
		totalLatency += l
	}
	result.AvgLatency = totalLatency / time.Duration(len(lats))
	result.MinLatency = lats[0]
	result.MaxLatency = lats[len(lats)-1]

	result.P50Latency = lats[int(float64(len(lats))*0.50)]
	result.P90Latency = lats[int(float64(len(lats))*0.90)]
	result.P99Latency = lats[int(float64(len(lats))*0.99)]

	return result
}

func sortDurations(durations []time.Duration) {
	n := len(durations)
	for i := 0; i < n-1; i++ {
		for j := 0; j < n-i-1; j++ {
			if durations[j] > durations[j+1] {
				durations[j], durations[j+1] = durations[j+1], durations[j]
			}
		}
	}
}

var DefaultLoadProfiles = []LoadProfile{
	{
		Name:        "light",
		Concurrency: 10,
		Duration:    10 * time.Second,
		RampUp:      2 * time.Second,
		MaxErrors:   50,
	},
	{
		Name:        "medium",
		Concurrency: 50,
		Duration:    30 * time.Second,
		RampUp:      5 * time.Second,
		MaxErrors:   100,
	},
	{
		Name:        "heavy",
		Concurrency: 200,
		Duration:    60 * time.Second,
		RampUp:      10 * time.Second,
		MaxErrors:   500,
	},
	{
		Name:        "stress",
		Concurrency: 500,
		Duration:    120 * time.Second,
		RampUp:      20 * time.Second,
		MaxErrors:   1000,
	},
}

type ConnectionSimulator struct {
	mu          sync.Mutex
	connections int
	maxSeen     int
	active      map[int]bool
	nextID      int
}

func NewConnectionSimulator() *ConnectionSimulator {
	return &ConnectionSimulator{
		active: make(map[int]bool),
	}
}

func (cs *ConnectionSimulator) Connect() int {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	cs.nextID++
	id := cs.nextID
	cs.active[id] = true
	cs.connections++
	if cs.connections > cs.maxSeen {
		cs.maxSeen = cs.connections
	}
	return id
}

func (cs *ConnectionSimulator) Disconnect(id int) {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	if cs.active[id] {
		delete(cs.active, id)
		cs.connections--
	}
}

func (cs *ConnectionSimulator) Stats() (current, max int) {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	return cs.connections, cs.maxSeen
}

func (cs *ConnectionSimulator) SimulateTraffic(ctx context.Context, numConnections int, opsPerConn int) {
	var wg sync.WaitGroup
	for i := 0; i < numConnections; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			connID := cs.Connect()
			defer cs.Disconnect(connID)

			for j := 0; j < opsPerConn; j++ {
				select {
				case <-ctx.Done():
					return
				default:
				}
				time.Sleep(time.Millisecond * time.Duration(50+id%50))
			}
		}(i)
	}
	wg.Wait()
}

func FormatLoadTestResult(result LoadTestResult) string {
	return fmt.Sprintf(
		"  %-20s  total=%d  success=%d  errors=%d  %.0f ops/s\n"+
			"  %-20s  avg=%v  p50=%v  p90=%v  p99=%v  min=%v  max=%v\n",
		result.Name,
		result.TotalOps, result.SuccessOps, result.ErrorOps,
		result.OpsPerSecond,
		"",
		result.AvgLatency.Round(time.Millisecond),
		result.P50Latency.Round(time.Millisecond),
		result.P90Latency.Round(time.Millisecond),
		result.P99Latency.Round(time.Millisecond),
		result.MinLatency.Round(time.Millisecond),
		result.MaxLatency.Round(time.Millisecond),
	)
}

func RunLoadTestSuite(operations map[string]func(ctx context.Context) error) {
	logger.Info("═══════════════════════════════════════════════════════════════")
	logger.Info("  LOAD TEST SUITE")
	logger.Info("═══════════════════════════════════════════════════════════════")

	for name, op := range operations {
		profile := LoadProfile{
			Name:        name,
			Concurrency: 25,
			Duration:    15 * time.Second,
			RampUp:      3 * time.Second,
			MaxErrors:   100,
		}

		tester := NewLoadTester(profile)
		result := tester.Run(op)

		logger.Infof("\n%s", FormatLoadTestResult(result))
	}
}

type ThroughputResult struct {
	BytesPerSecond float64 `json:"bytesPerSecond"`
	TotalBytes     int64   `json:"totalBytes"`
	Duration       time.Duration `json:"duration"`
}

func MeasureThroughput(fn func() (int, error), duration time.Duration) ThroughputResult {
	start := time.Now
	end := start().Add(duration)
	totalBytes := atomic.Int64{}

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for time.Now().Before(end) {
				n, err := fn()
				if err == nil {
					totalBytes.Add(int64(n))
				}
			}
		}()
	}
	wg.Wait()

	elapsed := time.Since(start())
	return ThroughputResult{
		BytesPerSecond: float64(totalBytes.Load()) / elapsed.Seconds(),
		TotalBytes:     totalBytes.Load(),
		Duration:       elapsed,
	}
}

func SimulateHighConcurrency(ctx context.Context, workers int, duration time.Duration, task func(id int) error) map[int]error {
	ctx, cancel := context.WithTimeout(ctx, duration)
	defer cancel()

	results := make(map[int]error)
	var mu sync.Mutex
	var wg sync.WaitGroup

	sem := make(chan struct{}, int(math.Min(float64(workers), 100)))

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			select {
			case <-ctx.Done():
				mu.Lock()
				results[id] = ctx.Err()
				mu.Unlock()
				return
			default:
			}

			err := task(id)
			mu.Lock()
			results[id] = err
			mu.Unlock()
		}(i)
	}

	wg.Wait()

	return results
}
