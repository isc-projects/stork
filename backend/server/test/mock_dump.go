package storktest

import "isc.org/stork/server/dumper/dump"

// Mock dump - only for test purposes.
type MockDump struct {
	*dump.BasicDump
	Err       error
	CallCount int
}

func NewMockDump(name string, err error) *MockDump {
	return &MockDump{
		dump.NewBasicDump(name),
		err,
		0,
	}
}

func (d *MockDump) Execute() error {
	d.CallCount++
	return d.Err
}
