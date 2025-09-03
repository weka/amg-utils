package scheduler

import (
	"time"
)

// TestRunner defines the interface for running tests
type TestRunner interface {
	RunTest() (passed bool, duration time.Duration, logs string, err error)
	GetName() string
}
