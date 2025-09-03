package scheduler

import (
	"fmt"
	"time"
)

// TestRunner defines the interface for running tests
type TestRunner interface {
	RunTest() (passed bool, duration time.Duration, logs string, err error)
	GetName() string
}

// DependentTestRunner defines the interface for tests with dependencies
type DependentTestRunner interface {
	TestRunner
	GetDependencies() []string // Returns list of test names this test depends on
}

// TestResult represents the result of a single test execution
type TestExecutionResult struct {
	Name     string
	Passed   bool
	Duration time.Duration
	Logs     string
	Error    error
	Skipped  bool
	Reason   string
}

// TestDependencyEngine handles test execution with dependency resolution
type TestDependencyEngine struct {
	tests   []TestRunner
	results map[string]*TestExecutionResult
}

// NewTestDependencyEngine creates a new dependency engine
func NewTestDependencyEngine(tests []TestRunner) *TestDependencyEngine {
	return &TestDependencyEngine{
		tests:   tests,
		results: make(map[string]*TestExecutionResult),
	}
}

// ExecuteTests runs all tests respecting dependencies
func (e *TestDependencyEngine) ExecuteTests() ([]*TestExecutionResult, error) {
	// Build dependency graph and execution order
	executionOrder, err := e.buildExecutionOrder()
	if err != nil {
		return nil, fmt.Errorf("failed to build execution order: %w", err)
	}

	var results []*TestExecutionResult

	// Execute tests in dependency order
	for _, testName := range executionOrder {
		test := e.findTestByName(testName)
		if test == nil {
			return nil, fmt.Errorf("test not found: %s", testName)
		}

		result := e.executeTest(test)
		e.results[testName] = result
		results = append(results, result)
	}

	return results, nil
}

// buildExecutionOrder creates a topologically sorted execution order
func (e *TestDependencyEngine) buildExecutionOrder() ([]string, error) {
	// Simple dependency resolution using Kahn's algorithm
	dependencies := make(map[string][]string)
	inDegree := make(map[string]int)

	// Initialize all tests
	for _, test := range e.tests {
		name := test.GetName()
		dependencies[name] = []string{}
		inDegree[name] = 0
	}

	// Build dependency graph
	for _, test := range e.tests {
		name := test.GetName()
		if depTest, ok := test.(DependentTestRunner); ok {
			deps := depTest.GetDependencies()
			for _, dep := range deps {
				if _, exists := inDegree[dep]; !exists {
					return nil, fmt.Errorf("dependency %s not found for test %s", dep, name)
				}
				dependencies[dep] = append(dependencies[dep], name)
				inDegree[name]++
			}
		}
	}

	// Topological sort
	var queue []string
	var result []string

	// Start with tests that have no dependencies
	for name, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, name)
		}
	}

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		result = append(result, current)

		// Reduce in-degree for dependent tests
		for _, dependent := range dependencies[current] {
			inDegree[dependent]--
			if inDegree[dependent] == 0 {
				queue = append(queue, dependent)
			}
		}
	}

	// Check for circular dependencies
	if len(result) != len(e.tests) {
		return nil, fmt.Errorf("circular dependency detected")
	}

	return result, nil
}

// findTestByName finds a test by its name
func (e *TestDependencyEngine) findTestByName(name string) TestRunner {
	for _, test := range e.tests {
		if test.GetName() == name {
			return test
		}
	}
	return nil
}

// executeTest runs a single test, checking dependencies first
func (e *TestDependencyEngine) executeTest(test TestRunner) *TestExecutionResult {
	name := test.GetName()

	// Check if this test has dependencies
	if depTest, ok := test.(DependentTestRunner); ok {
		dependencies := depTest.GetDependencies()
		for _, depName := range dependencies {
			depResult, exists := e.results[depName]
			if !exists || !depResult.Passed {
				// Skip this test because dependency failed
				reason := fmt.Sprintf("Dependency %s failed or was skipped", depName)
				if exists && depResult.Skipped {
					reason = fmt.Sprintf("Dependency %s was skipped: %s", depName, depResult.Reason)
				}
				return &TestExecutionResult{
					Name:     name,
					Passed:   false,
					Duration: 0,
					Logs:     fmt.Sprintf("Test skipped due to failed dependency: %s", reason),
					Error:    nil,
					Skipped:  true,
					Reason:   reason,
				}
			}
		}
	}

	// Execute the test
	passed, duration, logs, err := test.RunTest()

	return &TestExecutionResult{
		Name:     name,
		Passed:   passed,
		Duration: duration,
		Logs:     logs,
		Error:    err,
		Skipped:  false,
		Reason:   "",
	}
}
