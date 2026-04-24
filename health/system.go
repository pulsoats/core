package health

import "time"

type ServiceStatus int

const (
	ServiceStatusHealthy = iota
	ServiceStatusDegraded
)

type ServiceInfo struct {
	ID       string
	Name     string
	Exchange string
	Account  string
	Version  string
}

type ServiceMetrics struct {
	ServiceID     string
	Status        ServiceStatus
	CpuPercent    float64
	MemoryPercent float64
	UptimeSeconds int64
	ReportedAt    time.Time
}
