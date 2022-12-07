package storktest

import "github.com/go-pg/pg/v10/types"

// Mock for an internal Go-PG Pool Reader structure.
type poolReaderMock struct {
	bytes []byte
	err   error
}

// Constructs new mock instance. Accepts the fixed byte slice and error
// returned by the methods.
func NewPoolReaderMock(bytes []byte, err error) types.Reader {
	return &poolReaderMock{
		bytes, err,
	}
}

// Not implemented.
func (r *poolReaderMock) Buffered() int {
	panic("not implemented")
}

// Not implemented.
func (r *poolReaderMock) Bytes() []byte {
	panic("not implemented")
}

// Not implemented.
func (r *poolReaderMock) Read([]byte) (int, error) {
	panic("not implemented")
}

// Not implemented.
func (r *poolReaderMock) ReadByte() (byte, error) {
	panic("not implemented")
}

// Not implemented.
func (r *poolReaderMock) UnreadByte() error {
	panic("not implemented")
}

// Not implemented.
func (r *poolReaderMock) ReadSlice(byte) ([]byte, error) {
	panic("not implemented")
}

// Not implemented.
func (r *poolReaderMock) Discard(int) (int, error) {
	panic("not implemented")
}

// Not implemented.
func (r *poolReaderMock) ReadFull() ([]byte, error) {
	panic("not implemented")
}

// Returns a fixed bytes slice and an error.
func (r *poolReaderMock) ReadFullTemp() ([]byte, error) {
	return r.bytes, r.err
}
