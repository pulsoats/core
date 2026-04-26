package system

import (
	"time"

	"github.com/google/uuid"
)

type ServiceKind string

const (
	ServiceKindAnalysis = "analysis"
	ServiceKindLive     = "live"
)

type ServiceStatus int

const (
	ServiceStatusHealthy  ServiceStatus = iota
	ServiceStatusDegraded ServiceStatus = iota
)

type ServiceInfo struct {
	ID       uuid.UUID
	Kind     ServiceKind
	Name     string
	Exchange string
	Account  string
	Version  string
}

type ServiceMetrics struct {
	ServiceID     uuid.UUID
	Status        ServiceStatus
	CpuPercent    float64
	MemoryPercent float64
	UptimeSeconds int64
	ReportedAt    time.Time
}
