package test

// Package test provides integration and end-to-end tests for RangeDB.
// This file ensures there are statements available for coverage measurement.

// TestHelper provides utility functions for test setup and teardown.
type TestHelper struct {
	initialized bool
}

// NewTestHelper creates a new test helper instance.
func NewTestHelper() *TestHelper {
	return &TestHelper{
		initialized: true,
	}
}

// IsInitialized returns whether the test helper is initialized.
func (h *TestHelper) IsInitialized() bool {
	return h.initialized
}