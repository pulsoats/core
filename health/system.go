package health

import "time"

type ServiceStatus int

const (
	ServiceStatusHealthy  ServiceStatus = iota
	ServiceStatusDegraded ServiceStatus = iota
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
