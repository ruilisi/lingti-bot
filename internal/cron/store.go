package cron

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// Store handles persistence of scheduled jobs
type Store struct {
	path string
	mu   sync.RWMutex
}

// NewStore creates a new job store
func NewStore(path string) *Store {
	return &Store{
		path: path,
	}
}

// Load reads jobs from disk
func (s *Store) Load() ([]*Job, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Create directory if it doesn't exist
	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory: %w", err)
	}

	// Check if file exists
	if _, err := os.Stat(s.path); os.IsNotExist(err) {
		// File doesn't exist, return empty slice
		return []*Job{}, nil
	}

	// Read file
	data, err := os.ReadFile(s.path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// Parse JSON
	var jobs []*Job
	if err := json.Unmarshal(data, &jobs); err != nil {
		return nil, fmt.Errorf("failed to unmarshal jobs: %w", err)
	}

	return jobs, nil
}

// Save writes jobs to disk using atomic write
func (s *Store) Save(jobs []*Job) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Create directory if it doesn't exist
	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Marshal to JSON with indentation for readability
	data, err := json.MarshalIndent(jobs, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal jobs: %w", err)
	}

	// Write to temporary file first
	tmpPath := s.path + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write temp file: %w", err)
	}

	// Atomic rename
	if err := os.Rename(tmpPath, s.path); err != nil {
		// Clean up temp file on error
		os.Remove(tmpPath)
		return fmt.Errorf("failed to rename temp file: %w", err)
	}

	return nil
}
