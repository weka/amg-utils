package storage

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// TestResult represents a single test run result
type TestResult struct {
	Timestamp  time.Time        `json:"timestamp"`
	Passed     bool             `json:"passed"`
	Duration   string           `json:"duration"`
	Parameters string           `json:"parameters"`
	Logs       string           `json:"logs,omitempty"`
	Tests      []IndividualTest `json:"tests,omitempty"` // For test suites
}

// IndividualTest represents a single test within a test suite
type IndividualTest struct {
	Name     string `json:"name"`
	Passed   bool   `json:"passed"`
	Duration string `json:"duration"`
	Logs     string `json:"logs"`
}

// Storage handles persistent storage of test results
type Storage struct {
	resultsFile string
}

// New creates a new storage instance
func New(resultsPath string) (*Storage, error) {
	if err := os.MkdirAll(resultsPath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create results directory: %w", err)
	}

	resultsFile := filepath.Join(resultsPath, "results.jsonl")

	return &Storage{
		resultsFile: resultsFile,
	}, nil
}

// SaveResult saves a test result to storage
func (s *Storage) SaveResult(result TestResult) error {
	file, err := os.OpenFile(s.resultsFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("failed to open results file: %w", err)
	}
	defer func() {
		_ = file.Close()
	}()

	data, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("failed to marshal result: %w", err)
	}

	if _, err := file.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("failed to write result: %w", err)
	}

	return nil
}

// GetLastResults returns the last N test results (only test suites for dashboard)
func (s *Storage) GetLastResults(limit int) ([]TestResult, error) {
	file, err := os.Open(s.resultsFile)
	if err != nil {
		if os.IsNotExist(err) {
			return []TestResult{}, nil
		}
		return nil, fmt.Errorf("failed to open results file: %w", err)
	}
	defer func() {
		_ = file.Close()
	}()

	var results []TestResult
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		var result TestResult
		if err := json.Unmarshal(scanner.Bytes(), &result); err != nil {
			continue // Skip invalid lines
		}

		// Only include test suite results for dashboard (not individual test results)
		if strings.HasPrefix(result.Parameters, "test_suite_") || !strings.Contains(result.Parameters, "test") {
			results = append(results, result)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read results file: %w", err)
	}

	// Sort by timestamp (newest first)
	sort.Slice(results, func(i, j int) bool {
		return results[i].Timestamp.After(results[j].Timestamp)
	})

	// Limit results
	if len(results) > limit {
		results = results[:limit]
	}

	return results, nil
}
