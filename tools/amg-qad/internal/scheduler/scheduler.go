package scheduler

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/weka/amg-utils/tools/amg-qad/internal/storage"
)

// Scheduler manages test scheduling and execution
type Scheduler struct {
	testTime    string
	storage     *storage.Storage
	testRunners []TestRunner
	stopChan    chan bool
	isRunning   bool
}

// New creates a new scheduler with both test runners
func New(testTime string, store *storage.Storage) *Scheduler {
	return &Scheduler{
		testTime: testTime,
		storage:  store,
		testRunners: []TestRunner{
			NewPlaceholderTest(), // Fast, reliable test
			NewAmgctlTest(),      // Real integration test
		},
		stopChan: make(chan bool),
	}
}

// SetTestRunners allows changing the test runners (for testing or configuration)
func (s *Scheduler) SetTestRunners(runners []TestRunner) {
	s.testRunners = runners
}

// AddTestRunner adds a test runner to the existing list
func (s *Scheduler) AddTestRunner(runner TestRunner) {
	s.testRunners = append(s.testRunners, runner)
}

// Start starts the scheduler
func (s *Scheduler) Start() error {
	if s.isRunning {
		return fmt.Errorf("scheduler is already running")
	}

	log.Printf("Starting scheduler with test time: %s", s.testTime)
	s.isRunning = true

	go s.run()
	return nil
}

// Stop stops the scheduler
func (s *Scheduler) Stop() {
	if !s.isRunning {
		return
	}

	log.Println("Stopping scheduler...")
	s.stopChan <- true
	s.isRunning = false
}

// run is the main scheduler loop
func (s *Scheduler) run() {
	for {
		nextRun := s.getNextRunTime()
		waitDuration := time.Until(nextRun)

		log.Printf("Next test scheduled for: %s (in %v)",
			nextRun.Format("2006-01-02 15:04:05"), waitDuration)

		select {
		case <-time.After(waitDuration):
			if err := s.executeTest(); err != nil {
				log.Printf("Test execution failed: %v", err)
			}
		case <-s.stopChan:
			log.Println("Scheduler stopped")
			return
		}
	}
}

// getNextRunTime calculates the next time the test should run
func (s *Scheduler) getNextRunTime() time.Time {
	now := time.Now()

	// Parse the test time (HH:MM)
	parts := strings.Split(s.testTime, ":")
	hour, _ := strconv.Atoi(parts[0])
	minute, _ := strconv.Atoi(parts[1])

	// Create today's test time
	today := time.Date(now.Year(), now.Month(), now.Day(), hour, minute, 0, 0, now.Location())

	// If today's time has passed, schedule for tomorrow
	if now.After(today) {
		return today.Add(24 * time.Hour)
	}

	return today
}

// ExecuteTest runs the test immediately (public method for run-once mode)
func (s *Scheduler) ExecuteTest() error {
	return s.executeTest()
}

// executeTest runs all tests and stores the results
func (s *Scheduler) executeTest() error {
	log.Printf("Executing %d scheduled tests...", len(s.testRunners))

	overallStart := time.Now()
	allPassed := true
	var combinedLogs strings.Builder

	for i, testRunner := range s.testRunners {
		testName := fmt.Sprintf("test-%d", i+1)
		if _, ok := testRunner.(*PlaceholderTest); ok {
			testName = "placeholder_test"
		}
		if _, ok := testRunner.(*AmgctlTest); ok {
			testName = "amgctl_integration_test"
		}

		log.Printf("Running test %d/%d: %s", i+1, len(s.testRunners), testName)

		passed, duration, logs, err := testRunner.RunTest()

		// Create individual result for this test
		result := storage.TestResult{
			Timestamp:  time.Now(),
			Passed:     passed,
			Duration:   duration.String(),
			Parameters: testName,
			Logs:       logs,
		}

		if err != nil {
			result.Passed = false
			result.Logs = fmt.Sprintf("Test execution error: %v\n%s", err, logs)
		}

		// Save individual test result
		if err := s.storage.SaveResult(result); err != nil {
			log.Printf("Failed to save test result for %s: %v", testName, err)
			// Continue with other tests even if one fails to save
		}

		// Track overall status
		if !passed {
			allPassed = false
		}

		status := "PASSED"
		if !passed {
			status = "FAILED"
		}

		log.Printf("Test %s completed: %s (duration: %v)", testName, status, duration)

		// Add to combined logs for summary
		fmt.Fprintf(&combinedLogs, "=== %s ===\n%s\n\n", testName, logs)
	}

	overallDuration := time.Since(overallStart)

	// Create summary result
	summaryResult := storage.TestResult{
		Timestamp:  time.Now(),
		Passed:     allPassed,
		Duration:   overallDuration.String(),
		Parameters: fmt.Sprintf("test_suite_%d_tests", len(s.testRunners)),
		Logs:       fmt.Sprintf("Test Suite Summary:\n%d tests executed in %v\nOverall result: %t\n\n%s", len(s.testRunners), overallDuration, allPassed, combinedLogs.String()),
	}

	if err := s.storage.SaveResult(summaryResult); err != nil {
		log.Printf("Failed to save summary result: %v", err)
		return fmt.Errorf("failed to save summary result: %w", err)
	}

	status := "PASSED"
	if !allPassed {
		status = "FAILED"
	}

	log.Printf("All tests completed: %s (total duration: %v)", status, overallDuration)
	return nil
}
