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
	testTime   string
	storage    *storage.Storage
	testRunner TestRunner
	stopChan   chan bool
	isRunning  bool
}

// New creates a new scheduler
func New(testTime string, store *storage.Storage) *Scheduler {
	return &Scheduler{
		testTime:   testTime,
		storage:    store,
		testRunner: NewPlaceholderTest(), // Use placeholder test by default for reliability
		stopChan:   make(chan bool),
	}
}

// SetTestRunner allows changing the test runner (for testing or configuration)
func (s *Scheduler) SetTestRunner(runner TestRunner) {
	s.testRunner = runner
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

// executeTest runs the test and stores the result
func (s *Scheduler) executeTest() error {
	log.Println("Executing scheduled test...")

	passed, duration, logs, err := s.testRunner.RunTest()

	result := storage.TestResult{
		Timestamp:  time.Now(),
		Passed:     passed,
		Duration:   duration.String(),
		Parameters: "placeholder_test",
		Logs:       logs,
	}

	if err != nil {
		result.Passed = false
		result.Logs = fmt.Sprintf("Test execution error: %v\n%s", err, logs)
	}

	if err := s.storage.SaveResult(result); err != nil {
		log.Printf("Failed to save test result: %v", err)
		return fmt.Errorf("failed to save test result: %w", err)
	}

	status := "PASSED"
	if !passed {
		status = "FAILED"
	}

	log.Printf("Test completed: %s (duration: %v)", status, duration)
	return nil
}
