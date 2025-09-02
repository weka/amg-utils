package scheduler

import (
	"fmt"
	"math/rand"
	"time"
)

// TestRunner defines the interface for running tests
type TestRunner interface {
	RunTest() (passed bool, duration time.Duration, logs string, err error)
}

// PlaceholderTest is a simple test implementation that serves as a placeholder
type PlaceholderTest struct {
	Name string
}

// NewPlaceholderTest creates a new placeholder test
func NewPlaceholderTest() *PlaceholderTest {
	return &PlaceholderTest{
		Name: "AMG Environment Health Check",
	}
}

// RunTest executes the placeholder test
func (t *PlaceholderTest) RunTest() (bool, time.Duration, string, error) {
	start := time.Now()

	// Simulate some test work
	sleepDuration := time.Duration(rand.Intn(3)+1) * time.Second
	time.Sleep(sleepDuration)

	duration := time.Since(start)

	// Simulate occasional failures (10% chance)
	passed := rand.Intn(10) != 0

	var logs string
	if passed {
		logs = fmt.Sprintf("Test '%s' completed successfully\n", t.Name)
		logs += fmt.Sprintf("Simulated work duration: %v\n", sleepDuration)
		logs += "All checks passed\n"
	} else {
		logs = fmt.Sprintf("Test '%s' failed\n", t.Name)
		logs += fmt.Sprintf("Simulated work duration: %v\n", sleepDuration)
		logs += "ERROR: Simulated failure occurred\n"
		logs += "This is a placeholder error for testing purposes\n"
	}

	return passed, duration, logs, nil
}
