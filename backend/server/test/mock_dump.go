package storktest

import "isc.org/stork/server/dumper/dumps"

// Mock dump - only for test purposes.
type MockDump struct {
	*dumps.BasicDump
	Err       error
	CallCount int
}

func NewMockDump(name string, err error) *MockDump {
	return &MockDump{
		dumps.NewBasicDump(name),
		err,
		0,
	}
}

func (d *MockDump) Execute() error {
	d.CallCount++
	return d.Err
}
