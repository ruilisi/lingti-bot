package cron

import (
	"time"

	"github.com/robfig/cron/v3"
)

// Job represents a scheduled task
type Job struct {
	ID        string         `json:"id"`                  // Unique identifier
	Name      string         `json:"name"`                // Human-readable name
	Schedule  string         `json:"schedule"`            // Cron expression
	Tool      string         `json:"tool"`                // MCP tool to execute
	Arguments map[string]any `json:"arguments,omitempty"` // Tool arguments
	Enabled   bool                   `json:"enabled"`             // Whether job is active
	CreatedAt time.Time              `json:"created_at"`          // Job creation timestamp
	LastRun   *time.Time             `json:"last_run,omitempty"`  // Last execution timestamp
	LastError string                 `json:"last_error,omitempty"` // Last error message

	// Runtime fields (not persisted)
	EntryID cron.EntryID `json:"-"` // Cron scheduler entry ID
}

// Clone creates a deep copy of the job
func (j *Job) Clone() *Job {
	clone := &Job{
		ID:        j.ID,
		Name:      j.Name,
		Schedule:  j.Schedule,
		Tool:      j.Tool,
		Enabled:   j.Enabled,
		CreatedAt: j.CreatedAt,
		LastError: j.LastError,
		EntryID:   j.EntryID,
	}

	if j.LastRun != nil {
		lastRun := *j.LastRun
		clone.LastRun = &lastRun
	}

	if j.Arguments != nil {
		clone.Arguments = make(map[string]any, len(j.Arguments))
		for k, v := range j.Arguments {
			clone.Arguments[k] = v
		}
	}

	return clone
}
