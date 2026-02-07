package cron

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/robfig/cron/v3"
)

// ToolExecutor interface for executing MCP tools
type ToolExecutor interface {
	ExecuteTool(ctx context.Context, toolName string, arguments map[string]any) (any, error)
}

// ChatNotifier interface for sending messages to chat
type ChatNotifier interface {
	NotifyChat(message string) error
}

// Scheduler manages scheduled jobs
type Scheduler struct {
	cron         *cron.Cron
	store        *Store
	toolExecutor ToolExecutor
	chatNotifier ChatNotifier
	jobs         map[string]*Job
	mu           sync.RWMutex
}

// NewScheduler creates a new scheduler
func NewScheduler(store *Store, toolExecutor ToolExecutor, chatNotifier ChatNotifier) *Scheduler {
	return &Scheduler{
		cron:         cron.New(cron.WithSeconds()), // Support second-level precision
		store:        store,
		toolExecutor: toolExecutor,
		chatNotifier: chatNotifier,
		jobs:         make(map[string]*Job),
	}
}

// Start loads jobs from storage and starts the scheduler
func (s *Scheduler) Start() error {
	// Load jobs from disk
	jobs, err := s.store.Load()
	if err != nil {
		return fmt.Errorf("failed to load jobs: %w", err)
	}

	// Schedule enabled jobs
	for _, job := range jobs {
		s.jobs[job.ID] = job
		if job.Enabled {
			if err := s.scheduleJob(job); err != nil {
				log.Printf("[CRON] Failed to schedule job %s (%s): %v", job.ID, job.Name, err)
			}
		}
	}

	// Start the cron scheduler
	s.cron.Start()
	log.Printf("[CRON] Scheduler started with %d jobs (%d enabled)", len(s.jobs), s.countEnabled())

	return nil
}

// Stop stops the scheduler and saves jobs
func (s *Scheduler) Stop() error {
	// Stop the cron scheduler
	ctx := s.cron.Stop()
	<-ctx.Done()

	// Save jobs to disk
	s.mu.RLock()
	jobs := make([]*Job, 0, len(s.jobs))
	for _, job := range s.jobs {
		jobs = append(jobs, job)
	}
	s.mu.RUnlock()

	if err := s.store.Save(jobs); err != nil {
		return fmt.Errorf("failed to save jobs: %w", err)
	}

	log.Printf("[CRON] Scheduler stopped")
	return nil
}

// AddJob adds a new job to the scheduler
func (s *Scheduler) AddJob(name, schedule, tool string, arguments map[string]any) (*Job, error) {
	// Validate cron expression
	if _, err := cron.ParseStandard(schedule); err != nil {
		return nil, fmt.Errorf("invalid cron expression: %w", err)
	}

	// Create new job
	job := &Job{
		ID:        uuid.New().String(),
		Name:      name,
		Schedule:  schedule,
		Tool:      tool,
		Arguments: arguments,
		Enabled:   true,
		CreatedAt: time.Now(),
	}

	// Add to jobs map
	s.mu.Lock()
	s.jobs[job.ID] = job
	s.mu.Unlock()

	// Schedule the job
	if err := s.scheduleJob(job); err != nil {
		s.mu.Lock()
		delete(s.jobs, job.ID)
		s.mu.Unlock()
		return nil, fmt.Errorf("failed to schedule job: %w", err)
	}

	// Save to disk
	if err := s.saveJobs(); err != nil {
		log.Printf("[CRON] Failed to save jobs: %v", err)
	}

	log.Printf("[CRON] Job created: %s (%s) - schedule: %s, tool: %s", job.ID, job.Name, job.Schedule, job.Tool)
	return job, nil
}

// RemoveJob removes a job from the scheduler
func (s *Scheduler) RemoveJob(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	job, exists := s.jobs[id]
	if !exists {
		return fmt.Errorf("job not found: %s", id)
	}

	// Remove from cron scheduler if it has an entry
	if job.EntryID != 0 {
		s.cron.Remove(job.EntryID)
	}

	// Remove from jobs map
	delete(s.jobs, id)

	// Save to disk
	if err := s.saveJobsLocked(); err != nil {
		log.Printf("[CRON] Failed to save jobs: %v", err)
	}

	log.Printf("[CRON] Job removed: %s (%s)", job.ID, job.Name)
	return nil
}

// PauseJob pauses a job
func (s *Scheduler) PauseJob(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	job, exists := s.jobs[id]
	if !exists {
		return fmt.Errorf("job not found: %s", id)
	}

	if !job.Enabled {
		return fmt.Errorf("job is already paused")
	}

	// Remove from cron scheduler
	if job.EntryID != 0 {
		s.cron.Remove(job.EntryID)
		job.EntryID = 0
	}

	job.Enabled = false

	// Save to disk
	if err := s.saveJobsLocked(); err != nil {
		log.Printf("[CRON] Failed to save jobs: %v", err)
	}

	log.Printf("[CRON] Job paused: %s (%s)", job.ID, job.Name)
	return nil
}

// ResumeJob resumes a paused job
func (s *Scheduler) ResumeJob(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	job, exists := s.jobs[id]
	if !exists {
		return fmt.Errorf("job not found: %s", id)
	}

	if job.Enabled {
		return fmt.Errorf("job is already running")
	}

	job.Enabled = true

	// Schedule the job
	s.mu.Unlock()
	err := s.scheduleJob(job)
	s.mu.Lock()

	if err != nil {
		job.Enabled = false
		return fmt.Errorf("failed to schedule job: %w", err)
	}

	// Save to disk
	if err := s.saveJobsLocked(); err != nil {
		log.Printf("[CRON] Failed to save jobs: %v", err)
	}

	log.Printf("[CRON] Job resumed: %s (%s)", job.ID, job.Name)
	return nil
}

// ListJobs returns all jobs
func (s *Scheduler) ListJobs() []*Job {
	s.mu.RLock()
	defer s.mu.RUnlock()

	jobs := make([]*Job, 0, len(s.jobs))
	for _, job := range s.jobs {
		jobs = append(jobs, job.Clone())
	}

	return jobs
}

// scheduleJob schedules a job in the cron scheduler
func (s *Scheduler) scheduleJob(job *Job) error {
	entryID, err := s.cron.AddFunc(job.Schedule, func() {
		s.executeJob(job)
	})
	if err != nil {
		return err
	}

	job.EntryID = entryID
	return nil
}

// executeJob executes a job
func (s *Scheduler) executeJob(job *Job) {
	now := time.Now()
	log.Printf("[CRON] Executing job: %s (%s) - tool: %s", job.ID, job.Name, job.Tool)

	// Execute the tool
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	result, err := s.toolExecutor.ExecuteTool(ctx, job.Tool, job.Arguments)

	// Update job status
	s.mu.Lock()
	job.LastRun = &now
	if err != nil {
		job.LastError = err.Error()
		s.mu.Unlock()

		// Log error to terminal
		log.Printf("[CRON] Job failed: %s (%s) - error: %v", job.ID, job.Name, err)

		// Send error to chat
		if s.chatNotifier != nil {
			errMsg := fmt.Sprintf("⚠️ Scheduled job '%s' failed: %v", job.Name, err)
			if notifyErr := s.chatNotifier.NotifyChat(errMsg); notifyErr != nil {
				log.Printf("[CRON] Failed to send error notification: %v", notifyErr)
			}
		}
	} else {
		job.LastError = ""
		s.mu.Unlock()

		// Log success
		resultStr := ""
		if result != nil {
			if resultJSON, err := json.Marshal(result); err == nil {
				resultStr = fmt.Sprintf(" - result: %s", string(resultJSON))
			}
		}
		log.Printf("[CRON] Job completed: %s (%s)%s", job.ID, job.Name, resultStr)
	}

	// Save updated job status
	if err := s.saveJobs(); err != nil {
		log.Printf("[CRON] Failed to save jobs: %v", err)
	}
}

// saveJobs saves all jobs to disk (with lock)
func (s *Scheduler) saveJobs() error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.saveJobsLocked()
}

// saveJobsLocked saves all jobs to disk (caller must hold lock)
func (s *Scheduler) saveJobsLocked() error {
	jobs := make([]*Job, 0, len(s.jobs))
	for _, job := range s.jobs {
		jobs = append(jobs, job)
	}
	return s.store.Save(jobs)
}

// countEnabled returns the number of enabled jobs
func (s *Scheduler) countEnabled() int {
	count := 0
	for _, job := range s.jobs {
		if job.Enabled {
			count++
		}
	}
	return count
}
