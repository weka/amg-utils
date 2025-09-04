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

// New creates a new scheduler with all test runners
func New(testTime string, store *storage.Storage, version string) *Scheduler {
	return &Scheduler{
		testTime: testTime,
		storage:  store,
		testRunners: []TestRunner{
			NewAmgctlFetchLatestTest(version),     // Real integration test
			NewAmgctlUpgradeToLatestTest(version), // Upgrade functionality test
			NewAmgctlSetupTest(),                  // Host setup functionality test
			NewAmgctlOnDiagnosticsTest(),          // Diagnostic test depending on setup test
			NewAmgctlConfigCufileTest(),           // Cufile configuration test depending on setup test
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

// executeTest runs all tests using dependency engine and stores the results
func (s *Scheduler) executeTest() error {
	log.Printf("Executing %d scheduled tests with dependency resolution...", len(s.testRunners))

	overallStart := time.Now()

	// Create dependency engine and execute tests
	engine := NewTestDependencyEngine(s.testRunners)
	results, err := engine.ExecuteTests()
	if err != nil {
		return fmt.Errorf("test execution failed: %w", err)
	}

	allPassed := true
	var combinedLogs strings.Builder
	var individualTests []storage.IndividualTest

	// Process results
	for i, result := range results {
		testName := result.Name

		if result.Skipped {
			log.Printf("Running test %d/%d: %s - SKIPPED (%s)", i+1, len(results), testName, result.Reason)
		} else {
			log.Printf("Running test %d/%d: %s", i+1, len(results), testName)
		}

		passed := result.Passed && !result.Skipped
		logs := result.Logs

		if result.Error != nil && !result.Skipped {
			passed = false
			logs = fmt.Sprintf("Test execution error: %v\n%s", result.Error, logs)
		}

		// Create individual test record for embedding in test suite
		individualTest := storage.IndividualTest{
			Name:     testName,
			Passed:   passed,
			Duration: result.Duration.String(),
			Logs:     logs,
		}
		individualTests = append(individualTests, individualTest)

		// Track overall status
		if !passed {
			allPassed = false
		}

		status := "PASSED"
		if result.Skipped {
			status = "SKIPPED"
		} else if !passed {
			status = "FAILED"
		}

		log.Printf("Test %s completed: %s (duration: %v)", testName, status, result.Duration)

		// Print detailed logs to console if test failed or was skipped with important info
		if !passed || result.Skipped {
			statusEmoji := "❌"
			if result.Skipped {
				statusEmoji = "⏭️"
			}
			fmt.Printf("\n%s %s LOGS for %s:\n", statusEmoji, status, testName)
			fmt.Printf("---\n%s---\n\n", logs)
		}

		// Add to combined logs for summary
		fmt.Fprintf(&combinedLogs, "=== %s ===\n%s\n\n", testName, logs)
	}

	overallDuration := time.Since(overallStart)

	// Create test suite result with embedded individual tests
	suiteResult := storage.TestResult{
		Timestamp:  time.Now(),
		Passed:     allPassed,
		Duration:   overallDuration.String(),
		Parameters: fmt.Sprintf("test_suite_%d_tests", len(results)),
		Logs:       fmt.Sprintf("Test Suite Summary:\n%d tests executed in %v\nOverall result: %t\n\n%s", len(results), overallDuration, allPassed, combinedLogs.String()),
		Tests:      individualTests, // Embed individual test results
	}

	if err := s.storage.SaveResult(suiteResult); err != nil {
		log.Printf("Failed to save test suite result: %v", err)
		return fmt.Errorf("failed to save test suite result: %w", err)
	}

	status := "PASSED"
	if !allPassed {
		status = "FAILED"
	}

	log.Printf("All tests completed: %s (total duration: %v)", status, overallDuration)
	return nil
}
