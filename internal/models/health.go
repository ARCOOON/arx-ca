package models

// HealthReport aggregates runtime and subsystem status for the health endpoint.
type HealthReport struct {
	Uptime    UptimeInfo       `json:"uptime"`
	Memory    MemoryStats      `json:"memory"`
	API       APIStatus        `json:"api"`
	CABackend CABackendStatus  `json:"ca_backend"`
}

// UptimeInfo describes how long the server process has been running.
type UptimeInfo struct {
	Seconds int64  `json:"seconds"`
	Human   string `json:"human"`
}

// MemoryStats captures Go runtime memory allocator metrics.
type MemoryStats struct {
	Alloc        uint64 `json:"alloc_bytes"`
	TotalAlloc   uint64 `json:"total_alloc_bytes"`
	Sys          uint64 `json:"sys_bytes"`
	HeapAlloc    uint64 `json:"heap_alloc_bytes"`
	HeapInuse    uint64 `json:"heap_inuse_bytes"`
	HeapObjects  uint64 `json:"heap_objects"`
	StackInuse   uint64 `json:"stack_inuse_bytes"`
	NumGC        uint32 `json:"num_gc"`
	LastGCUnix   int64  `json:"last_gc_unix"`
	Goroutines   int    `json:"goroutines"`
}

// APIStatus reports the operational state of the HTTP API layer.
type APIStatus struct {
	Status  string `json:"status"`
	Version string `json:"version"`
}

// CABackendStatus reports the state of the PKI engine (step-ca integration).
type CABackendStatus struct {
	Status      string `json:"status"`
	Message     string `json:"message"`
	Engine      string `json:"engine"`
	Initialized bool   `json:"initialized"`
}
