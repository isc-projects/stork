package storktest

import "github.com/go-pg/pg/v10/types"

// Go-PG reader mock.
type poolReaderMock struct {
	bytes []byte
	err   error
}

func NewPoolReaderMock(bytes []byte, err error) types.Reader {
	return &poolReaderMock{
		bytes, err,
	}
}

func (r *poolReaderMock) Buffered() int {
	panic("not implemented")
}

func (r *poolReaderMock) Bytes() []byte {
	panic("not implemented")
}

func (r *poolReaderMock) Read([]byte) (int, error) {
	panic("not implemented")
}

func (r *poolReaderMock) ReadByte() (byte, error) {
	panic("not implemented")
}

func (r *poolReaderMock) UnreadByte() error {
	panic("not implemented")
}

func (r *poolReaderMock) ReadSlice(byte) ([]byte, error) {
	panic("not implemented")
}

func (r *poolReaderMock) Discard(int) (int, error) {
	panic("not implemented")
}

func (r *poolReaderMock) ReadFull() ([]byte, error) {
	panic("not implemented")
}

func (r *poolReaderMock) ReadFullTemp() ([]byte, error) {
	return r.bytes, r.err
}
