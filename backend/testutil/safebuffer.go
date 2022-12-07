package testutil

import (
	"bytes"
	"sync"
)

// The thread-safe wrapper for bytes.Buffer.
// Source: https://stackoverflow.com/a/36226525
type SafeBuffer struct {
	b bytes.Buffer
	m sync.Mutex
}

// Read the bytes from the buffer. It's a blocking operation.
func (b *SafeBuffer) Read(p []byte) (n int, err error) {
	b.m.Lock()
	defer b.m.Unlock()
	return b.b.Read(p)
}

// Write the bytes to the buffer. It's a blocking operation.
func (b *SafeBuffer) Write(p []byte) (n int, err error) {
	b.m.Lock()
	defer b.m.Unlock()
	return b.b.Write(p)
}

// Read all bytes from the buffer. It's a blocking operation.
func (b *SafeBuffer) Bytes() []byte {
	b.m.Lock()
	defer b.m.Unlock()
	return b.b.Bytes()
}
