package handlers

import (
	"fmt"
	"net/http"
	"runtime"
	"time"

	"github.com/your-org/ca-api/internal/api"
	"github.com/your-org/ca-api/internal/ca"
	"github.com/your-org/ca-api/internal/models"
)

const apiVersion = "v1"

// HealthHandler serves the GET /api/v1/health endpoint.
type HealthHandler struct {
	startTime time.Time
	engine    *ca.PKIEngine
}

// NewHealthHandler constructs a health handler bound to the server start instant.
func NewHealthHandler(startTime time.Time, engine *ca.PKIEngine) *HealthHandler {
	return &HealthHandler{
		startTime: startTime,
		engine:    engine,
	}
}

// ServeHTTP implements http.Handler.
func (h *HealthHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		api.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	report := h.buildReport()
	api.WriteSuccess(w, http.StatusOK, report)
}

func (h *HealthHandler) buildReport() models.HealthReport {
	return models.HealthReport{
		Uptime:    h.uptime(),
		Memory:    h.memoryStats(),
		API:       h.apiStatus(),
		CABackend: h.engine.Status(),
	}
}

func (h *HealthHandler) uptime() models.UptimeInfo {
	elapsed := time.Since(h.startTime)
	seconds := int64(elapsed.Seconds())

	return models.UptimeInfo{
		Seconds: seconds,
		Human:   formatDuration(elapsed),
	}
}

func (h *HealthHandler) memoryStats() models.MemoryStats {
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)

	lastGC := int64(0)
	if mem.LastGC > 0 {
		lastGC = time.Unix(0, int64(mem.LastGC)).Unix()
	}

	return models.MemoryStats{
		Alloc:       mem.Alloc,
		TotalAlloc:  mem.TotalAlloc,
		Sys:         mem.Sys,
		HeapAlloc:   mem.HeapAlloc,
		HeapInuse:   mem.HeapInuse,
		HeapObjects: mem.HeapObjects,
		StackInuse:  mem.StackInuse,
		NumGC:       mem.NumGC,
		LastGCUnix:  lastGC,
		Goroutines:  runtime.NumGoroutine(),
	}
}

func (h *HealthHandler) apiStatus() models.APIStatus {
	return models.APIStatus{
		Status:  "healthy",
		Version: apiVersion,
	}
}

func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm %ds", int(d.Minutes()), int(d.Seconds())%60)
	}
	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60
	seconds := int(d.Seconds()) % 60
	return fmt.Sprintf("%dh %dm %ds", hours, minutes, seconds)
}
