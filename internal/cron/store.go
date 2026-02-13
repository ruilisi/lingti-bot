package cron

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

// Store handles persistence of scheduled jobs using SQLite
type Store struct {
	db *sql.DB
	mu sync.RWMutex
}

// NewStore creates a new SQLite-backed job store at the given path
func NewStore(path string) (*Store, error) {
	// Ensure parent directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory: %w", err)
	}

	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Enable WAL mode for better concurrent read performance
	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to set WAL mode: %w", err)
	}

	s := &Store{db: db}

	if err := s.init(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	// Auto-migrate from legacy JSON file
	s.migrateFromJSON()

	return s, nil
}

// init creates the jobs table if it doesn't exist
func (s *Store) init() error {
	_, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS jobs (
			id         TEXT PRIMARY KEY,
			name       TEXT NOT NULL,
			schedule   TEXT NOT NULL,
			tool       TEXT,
			arguments  TEXT,
			message    TEXT,
			prompt     TEXT,
			platform   TEXT,
			channel_id TEXT,
			user_id    TEXT,
			enabled    INTEGER NOT NULL DEFAULT 1,
			created_at TEXT NOT NULL,
			last_run   TEXT,
			last_error TEXT
		)
	`)
	return err
}

// migrateFromJSON imports jobs from the legacy crons.json if it exists
func (s *Store) migrateFromJSON() {
	// The old JSON path was ~/.lingti/crons.json
	// Derive it: dbPath is ~/.lingti.db, so old path is ~/.lingti/crons.json
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return
	}
	jsonPath := filepath.Join(homeDir, ".lingti", "crons.json")

	data, err := os.ReadFile(jsonPath)
	if err != nil {
		return // file doesn't exist or unreadable
	}

	var jobs []*Job
	if err := json.Unmarshal(data, &jobs); err != nil {
		log.Printf("[CRON] Warning: failed to parse legacy %s for migration: %v", jsonPath, err)
		return
	}

	if len(jobs) == 0 {
		return
	}

	for _, job := range jobs {
		if err := s.SaveJob(job); err != nil {
			log.Printf("[CRON] Warning: failed to migrate job %s: %v", job.ID, err)
		}
	}

	// Rename old file to .bak
	bakPath := jsonPath + ".bak"
	if err := os.Rename(jsonPath, bakPath); err != nil {
		log.Printf("[CRON] Warning: failed to rename %s to %s: %v", jsonPath, bakPath, err)
	} else {
		log.Printf("[CRON] Migrated %d jobs from %s to SQLite", len(jobs), jsonPath)
	}
}

// Load reads all jobs from the database
func (s *Store) Load() ([]*Job, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	rows, err := s.db.Query(`
		SELECT id, name, schedule, tool, arguments, message, prompt,
		       platform, channel_id, user_id, enabled, created_at, last_run, last_error
		FROM jobs
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to query jobs: %w", err)
	}
	defer rows.Close()

	var jobs []*Job
	for rows.Next() {
		job, err := scanJob(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan job: %w", err)
		}
		jobs = append(jobs, job)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate jobs: %w", err)
	}

	if jobs == nil {
		jobs = []*Job{}
	}
	return jobs, nil
}

// SaveJob upserts a single job into the database
func (s *Store) SaveJob(job *Job) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	argsJSON, err := json.Marshal(job.Arguments)
	if err != nil {
		return fmt.Errorf("failed to marshal arguments: %w", err)
	}

	var lastRun *string
	if job.LastRun != nil {
		t := job.LastRun.Format(time.RFC3339)
		lastRun = &t
	}

	var lastError *string
	if job.LastError != "" {
		lastError = &job.LastError
	}

	enabled := 0
	if job.Enabled {
		enabled = 1
	}

	_, err = s.db.Exec(`
		INSERT INTO jobs (id, name, schedule, tool, arguments, message, prompt,
		                  platform, channel_id, user_id, enabled, created_at, last_run, last_error)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			name=excluded.name, schedule=excluded.schedule, tool=excluded.tool,
			arguments=excluded.arguments, message=excluded.message, prompt=excluded.prompt,
			platform=excluded.platform, channel_id=excluded.channel_id, user_id=excluded.user_id,
			enabled=excluded.enabled, created_at=excluded.created_at,
			last_run=excluded.last_run, last_error=excluded.last_error
	`,
		job.ID, job.Name, job.Schedule, job.Tool, string(argsJSON), job.Message, job.Prompt,
		job.Platform, job.ChannelID, job.UserID, enabled, job.CreatedAt.Format(time.RFC3339),
		lastRun, lastError,
	)
	return err
}

// DeleteJob removes a job from the database
func (s *Store) DeleteJob(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, err := s.db.Exec("DELETE FROM jobs WHERE id = ?", id)
	return err
}

// Save writes all jobs to the database (bulk upsert, used by Stop)
func (s *Store) Save(jobs []*Job) error {
	for _, job := range jobs {
		if err := s.SaveJob(job); err != nil {
			return err
		}
	}
	return nil
}

// Close closes the database connection
func (s *Store) Close() error {
	return s.db.Close()
}

// scanner interface for both *sql.Row and *sql.Rows
type scanner interface {
	Scan(dest ...any) error
}

func scanJob(s scanner) (*Job, error) {
	var (
		job       Job
		argsJSON  sql.NullString
		tool      sql.NullString
		message   sql.NullString
		prompt    sql.NullString
		platform  sql.NullString
		channelID sql.NullString
		userID    sql.NullString
		enabled   int
		createdAt string
		lastRun   sql.NullString
		lastError sql.NullString
	)

	err := s.Scan(
		&job.ID, &job.Name, &job.Schedule, &tool, &argsJSON, &message, &prompt,
		&platform, &channelID, &userID, &enabled, &createdAt, &lastRun, &lastError,
	)
	if err != nil {
		return nil, err
	}

	job.Tool = tool.String
	job.Message = message.String
	job.Prompt = prompt.String
	job.Platform = platform.String
	job.ChannelID = channelID.String
	job.UserID = userID.String
	job.Enabled = enabled != 0
	job.LastError = lastError.String

	if t, err := time.Parse(time.RFC3339, createdAt); err == nil {
		job.CreatedAt = t
	}
	if lastRun.Valid {
		if t, err := time.Parse(time.RFC3339, lastRun.String); err == nil {
			job.LastRun = &t
		}
	}

	if argsJSON.Valid && argsJSON.String != "" && argsJSON.String != "null" {
		if err := json.Unmarshal([]byte(argsJSON.String), &job.Arguments); err != nil {
			return nil, fmt.Errorf("failed to unmarshal arguments: %w", err)
		}
	}

	return &job, nil
}
