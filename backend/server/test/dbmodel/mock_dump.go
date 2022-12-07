package storktestdbmodel

import "isc.org/stork/server/dumper/dump"

// Mock dump - only for test purposes.
type MockDump struct {
	*dump.BasicDump
	Err       error
	CallCount int
}

// Constructs new mock instance. Accepts the name and fixed error returned
// by the Execute method.
func NewMockDump(name string, err error) *MockDump {
	return &MockDump{
		dump.NewBasicDump(name),
		err,
		0,
	}
}

// Counts the call and returns a fixed error.
func (d *MockDump) Execute() error {
	d.CallCount++
	return d.Err
}
