package postgres

import (
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

// Asset represents a monitored service
type Asset struct {
	ID              string             `db:"id"`
	Name            string             `db:"name"`
	Address         string             `db:"address"`
	IntervalSeconds int                `db:"interval_seconds"`
	TimeoutSeconds  int                `db:"timeout_seconds"`
	CreatedAt       pgtype.Timestamptz `db:"created_at"`
	UpdatedAt       pgtype.Timestamptz `db:"updated_at"`
}

// ProbeResult contains the outcome of a single health check
type ProbeResult struct {
	ID         string             `db:"id"`
	Success    bool               `db:"success"`
	Code       int                `db:"code"`
	ErrMessage *string            `db:"err_message"`
	CreatedAt  pgtype.Timestamptz `db:"created_at"`
	AssetID    string             `db:"asset_id"`
}

// Incident tracks a service outage
type Incident struct {
	ID        string             `db:"id"`
	Severity  *string            `db:"severity"`
	Summary   *string            `db:"summary"`
	StartedAt pgtype.Timestamptz `db:"started_at"`
	EndedAt   pgtype.Timestamptz `db:"ended_at"`
	AssetID   string             `db:"asset_id"`
}

type Event string

const (
	UP       Event = "up"
	DOWN     Event = "down"
	RECOVERY Event = "recovery"
)

func ToDetails(event Event, probe *ProbeResult) string {
	switch event {
	case UP:
		return "service is reachable"
	case DOWN:
		if probe.ErrMessage != nil && *probe.ErrMessage != "" {
			return *probe.ErrMessage
		}
		if probe.Code > 0 {
			return fmt.Sprintf("received status code %d", probe.Code)
		}
		return "probe failed"
	case RECOVERY:
		if probe.Code > 0 {
			return fmt.Sprintf("service recovered with status code %d", probe.Code)
		}
		return "service recovered"
	default:
		return ""
	}
}

// Notification is what we send to Telegram
type Notification struct {
	AssetID   string
	AssetName string
	Event     Event
	Timestamp time.Time
	Details   string
}

// ProbeProcessingResult captures everything persisted/derived from a probe execution.
type ProbeProcessingResult struct {
	Probe        *ProbeResult
	Incident     *Incident
	Notification *Notification
}
