package utils

import "time"

// Clock is the interface for getting the current time.
type Clock interface {
	// Now returns the current time in UTC.
	Now() time.Time
}

// RealClock is the real implementation of the Clock interface.
type RealClock struct{}

func (RealClock) Now() time.Time {
	return time.Now().UTC()
}

// NewRealClock creates a new RealClock.
func NewRealClock() *RealClock {
	return &RealClock{}
}

// MockClock is a mock implementation of the Clock interface.
type MockClock struct {
	Time time.Time
}

// SetMockClockTime sets the time of the mock clock.
func (m *MockClock) SetMockClockTime(t time.Time) {
	m.Time = t
}

// Now returns the current time in UTC.
func (m *MockClock) Now() time.Time {
	return m.Time
}

// NewMockClock creates a new MockClock.
func NewMockClock() *MockClock {
	return &MockClock{}
}
