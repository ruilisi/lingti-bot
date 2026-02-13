package cron

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/robfig/cron/v3"
)

// ToolExecutor interface for executing MCP tools
type ToolExecutor interface {
	ExecuteTool(ctx context.Context, toolName string, arguments map[string]any) (any, error)
}

// PromptExecutor interface for running full AI conversations
type PromptExecutor interface {
	ExecutePrompt(ctx context.Context, platform, channelID, userID, prompt string) (string, error)
}

// ChatNotifier interface for sending messages to chat
type ChatNotifier interface {
	NotifyChat(message string) error
	NotifyChatUser(platform, channelID, userID, message string) error
}

// Scheduler manages scheduled jobs
type Scheduler struct {
	cron           *cron.Cron
	store          *Store
	toolExecutor   ToolExecutor
	promptExecutor PromptExecutor
	chatNotifier   ChatNotifier
	jobs           map[string]*Job
	mu             sync.RWMutex
}

// NewScheduler creates a new scheduler
func NewScheduler(store *Store, toolExecutor ToolExecutor, promptExecutor PromptExecutor, chatNotifier ChatNotifier) *Scheduler {
	return &Scheduler{
		cron:           cron.New(cron.WithSeconds()), // Support second-level precision
		store:          store,
		toolExecutor:   toolExecutor,
		promptExecutor: promptExecutor,
		chatNotifier:   chatNotifier,
		jobs:           make(map[string]*Job),
	}
}

// normalizeCron prepends "0 " to standard 5-field cron expressions
// so they work with the 6-field (with seconds) parser.
func normalizeCron(schedule string) string {
	if len(strings.Fields(schedule)) == 5 {
		return "0 " + schedule
	}
	return schedule
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

// Stop stops the scheduler and closes the store
func (s *Scheduler) Stop() error {
	// Stop the cron scheduler
	ctx := s.cron.Stop()
	<-ctx.Done()

	// Close the database
	if err := s.store.Close(); err != nil {
		return fmt.Errorf("failed to close store: %w", err)
	}

	log.Printf("[CRON] Scheduler stopped")
	return nil
}

// AddJob adds a new tool-based job to the scheduler
func (s *Scheduler) AddJob(name, schedule, tool string, arguments map[string]any) (*Job, error) {
	return s.addJob(&Job{
		Name:      name,
		Schedule:  schedule,
		Tool:      tool,
		Arguments: arguments,
	})
}

// AddJobWithMessage adds a new message-based job that sends text to a chat user
func (s *Scheduler) AddJobWithMessage(name, schedule, message, platform, channelID, userID string) (*Job, error) {
	return s.addJob(&Job{
		Name:      name,
		Schedule:  schedule,
		Message:   message,
		Platform:  platform,
		ChannelID: channelID,
		UserID:    userID,
	})
}

// AddJobWithPrompt adds a new prompt-based job that runs a full AI conversation
func (s *Scheduler) AddJobWithPrompt(name, schedule, prompt, platform, channelID, userID string) (*Job, error) {
	return s.addJob(&Job{
		Name:      name,
		Schedule:  schedule,
		Prompt:    prompt,
		Platform:  platform,
		ChannelID: channelID,
		UserID:    userID,
	})
}

// addJob validates and schedules a job
func (s *Scheduler) addJob(job *Job) (*Job, error) {
	// Normalize 5-field cron to 6-field (our cron instance uses WithSeconds)
	job.Schedule = normalizeCron(job.Schedule)

	// Validate cron expression using the 6-field (with seconds) parser
	parser := cron.NewParser(cron.Second | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)
	if _, err := parser.Parse(job.Schedule); err != nil {
		return nil, fmt.Errorf("invalid cron expression: %w", err)
	}

	job.ID = uuid.New().String()
	job.Enabled = true
	job.CreatedAt = time.Now()

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

	// Save to database
	if err := s.store.SaveJob(job); err != nil {
		log.Printf("[CRON] Failed to save job: %v", err)
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

	// Delete from database
	if err := s.store.DeleteJob(id); err != nil {
		log.Printf("[CRON] Failed to delete job: %v", err)
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

	// Save to database
	if err := s.store.SaveJob(job); err != nil {
		log.Printf("[CRON] Failed to save job: %v", err)
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

	// Save to database
	if err := s.store.SaveJob(job); err != nil {
		log.Printf("[CRON] Failed to save job: %v", err)
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

	// Message-based job: send message directly to user
	if job.Message != "" {
		log.Printf("[CRON] Sending message for job: %s (%s)", job.ID, job.Name)

		s.mu.Lock()
		job.LastRun = &now
		s.mu.Unlock()

		if s.chatNotifier != nil && job.Platform != "" && job.ChannelID != "" {
			if err := s.chatNotifier.NotifyChatUser(job.Platform, job.ChannelID, job.UserID, job.Message); err != nil {
				s.mu.Lock()
				job.LastError = err.Error()
				s.mu.Unlock()
				log.Printf("[CRON] Job failed to send message: %s (%s) - error: %v", job.ID, job.Name, err)
			} else {
				s.mu.Lock()
				job.LastError = ""
				s.mu.Unlock()
				log.Printf("[CRON] Job message sent: %s (%s)", job.ID, job.Name)
			}
		} else {
			log.Printf("[CRON] Job %s has no chat target, logging message: %s", job.ID, job.Message)
			if s.chatNotifier != nil {
				s.chatNotifier.NotifyChat(fmt.Sprintf("[%s] %s", job.Name, job.Message))
			}
		}

		if err := s.store.SaveJob(job); err != nil {
			log.Printf("[CRON] Failed to save job: %v", err)
		}
		return
	}

	// Prompt-based job: run full AI conversation
	if job.Prompt != "" {
		log.Printf("[CRON] Running AI prompt for job: %s (%s)", job.ID, job.Name)

		s.mu.Lock()
		job.LastRun = &now
		s.mu.Unlock()

		if s.promptExecutor == nil {
			s.mu.Lock()
			job.LastError = "prompt executor not available"
			s.mu.Unlock()
			log.Printf("[CRON] Job failed: %s (%s) - prompt executor not available", job.ID, job.Name)
			if err := s.store.SaveJob(job); err != nil {
				log.Printf("[CRON] Failed to save job: %v", err)
			}
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()

		result, err := s.promptExecutor.ExecutePrompt(ctx, job.Platform, job.ChannelID, job.UserID, job.Prompt)
		if err != nil {
			s.mu.Lock()
			job.LastError = err.Error()
			s.mu.Unlock()
			log.Printf("[CRON] Job prompt failed: %s (%s) - error: %v", job.ID, job.Name, err)

			if s.chatNotifier != nil && job.Platform != "" && job.ChannelID != "" {
				s.chatNotifier.NotifyChatUser(job.Platform, job.ChannelID, job.UserID,
					fmt.Sprintf("⚠️ Scheduled AI task '%s' failed: %v", job.Name, err))
			}
		} else {
			s.mu.Lock()
			job.LastError = ""
			s.mu.Unlock()
			log.Printf("[CRON] Job prompt completed: %s (%s)", job.ID, job.Name)

			if s.chatNotifier != nil && job.Platform != "" && job.ChannelID != "" {
				s.chatNotifier.NotifyChatUser(job.Platform, job.ChannelID, job.UserID, result)
			}
		}

		if err := s.store.SaveJob(job); err != nil {
			log.Printf("[CRON] Failed to save job: %v", err)
		}
		return
	}

	// Tool-based job: execute MCP tool
	log.Printf("[CRON] Executing job: %s (%s) - tool: %s", job.ID, job.Name, job.Tool)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	result, err := s.toolExecutor.ExecuteTool(ctx, job.Tool, job.Arguments)

	// Update job status
	s.mu.Lock()
	job.LastRun = &now
	if err != nil {
		job.LastError = err.Error()
		s.mu.Unlock()

		log.Printf("[CRON] Job failed: %s (%s) - error: %v", job.ID, job.Name, err)

		if s.chatNotifier != nil {
			errMsg := fmt.Sprintf("⚠️ Scheduled job '%s' failed: %v", job.Name, err)
			if job.Platform != "" && job.ChannelID != "" {
				s.chatNotifier.NotifyChatUser(job.Platform, job.ChannelID, job.UserID, errMsg)
			} else {
				s.chatNotifier.NotifyChat(errMsg)
			}
		}
	} else {
		job.LastError = ""
		s.mu.Unlock()

		resultStr := ""
		if result != nil {
			if resultJSON, err := json.Marshal(result); err == nil {
				resultStr = fmt.Sprintf(" - result: %s", string(resultJSON))
			}
		}
		log.Printf("[CRON] Job completed: %s (%s)%s", job.ID, job.Name, resultStr)
	}

	if err := s.store.SaveJob(job); err != nil {
		log.Printf("[CRON] Failed to save job: %v", err)
	}
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
