package monitor

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type PrometheusExporter struct {
	checker    *HealthChecker
	collector  *MetricsCollector
	httpClient *http.Client
}

func NewPrometheusExporter(checker *HealthChecker, collector *MetricsCollector) *PrometheusExporter {
	return &PrometheusExporter{
		checker:   checker,
		collector: collector,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

func (e *PrometheusExporter) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; version=0.0.4")

	results := e.checker.GetAllResults()
	metrics := e.collector.GetAllLatest()

	var sb strings.Builder

	sb.WriteString("# HELP xui_protocol_healthy Protocol health status (1=healthy, 0=unhealthy)\n")
	sb.WriteString("# TYPE xui_protocol_healthy gauge\n")
	for _, res := range results {
		v := 0
		if res.Healthy {
			v = 1
		}
		sb.WriteString(fmt.Sprintf("xui_protocol_healthy{protocol=\"%s\"} %d\n", res.ProtocolID, v))
	}

	sb.WriteString("# HELP xui_protocol_up_bytes_total Total bytes uploaded per protocol\n")
	sb.WriteString("# TYPE xui_protocol_up_bytes_total counter\n")
	for id, m := range metrics {
		sb.WriteString(fmt.Sprintf("xui_protocol_up_bytes_total{protocol=\"%s\"} %d\n", id, m.UpBytes))
	}

	sb.WriteString("# HELP xui_protocol_down_bytes_total Total bytes downloaded per protocol\n")
	sb.WriteString("# TYPE xui_protocol_down_bytes_total counter\n")
	for id, m := range metrics {
		sb.WriteString(fmt.Sprintf("xui_protocol_down_bytes_total{protocol=\"%s\"} %d\n", id, m.DownBytes))
	}

	sb.WriteString("# HELP xui_protocol_active_connections Current active connections per protocol\n")
	sb.WriteString("# TYPE xui_protocol_active_connections gauge\n")
	for id, m := range metrics {
		sb.WriteString(fmt.Sprintf("xui_protocol_active_connections{protocol=\"%s\"} %d\n", id, m.Connections))
	}

	sb.WriteString("# HELP xui_protocol_active_users Current active users per protocol\n")
	sb.WriteString("# TYPE xui_protocol_active_users gauge\n")
	for id, m := range metrics {
		sb.WriteString(fmt.Sprintf("xui_protocol_active_users{protocol=\"%s\"} %d\n", id, m.ActiveUsers))
	}

	sb.WriteString("# HELP xui_protocol_error_count Total errors per protocol\n")
	sb.WriteString("# TYPE xui_protocol_error_count counter\n")
	for id, m := range metrics {
		sb.WriteString(fmt.Sprintf("xui_protocol_error_count{protocol=\"%s\"} %d\n", id, m.ErrorCount))
	}

	sb.WriteString("# HELP xui_protocol_uptime_seconds Protocol uptime in seconds\n")
	sb.WriteString("# TYPE xui_protocol_uptime_seconds gauge\n")
	for id, m := range metrics {
		sb.WriteString(fmt.Sprintf("xui_protocol_uptime_seconds{protocol=\"%s\"} %d\n", id, m.UptimeSeconds))
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(sb.String()))
}

func (e *PrometheusExporter) Run(ctx context.Context, addr string) error {
	mux := http.NewServeMux()
	mux.Handle("/metrics", e)

	server := &http.Server{
		Addr:        addr,
		Handler:     mux,
		ReadTimeout: 5 * time.Second,
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		server.Shutdown(shutdownCtx)
	}()

	return server.ListenAndServe()
}
