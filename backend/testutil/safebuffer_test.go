package testutil

import (
	"errors"
	"fmt"
	"io"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

// Test that reading bytes from safe buffer works properly.
func TestSafeBufferRead(t *testing.T) {
	// Arrange
	var buffer SafeBuffer
	_, _ = buffer.Write([]byte("foobar"))
	prefix := make([]byte, 3)
	suffix := make([]byte, 10)

	// Act
	nPrefix, errPrefix := buffer.Read(prefix)
	nSuffix, errSuffix := buffer.Read(suffix)

	// Assert
	require.NoError(t, errPrefix)
	require.EqualValues(t, 3, nPrefix)
	require.EqualValues(t, []byte("foo"), prefix)

	require.NoError(t, errSuffix)
	require.EqualValues(t, 3, nSuffix)
	require.EqualValues(t, []byte("bar"), suffix[:3])
}

// Test that writing bytes to safe buffer works properly.
func TestSafeBufferWrite(t *testing.T) {
	// Arrange
	var buffer SafeBuffer

	// Act
	n, err := buffer.Write([]byte("foobar"))

	// Assert
	require.NoError(t, err)
	require.EqualValues(t, 6, n)
	require.EqualValues(t, []byte("foobar"), buffer.Bytes())
}

// Test that exporting bytes from safe buffer works properly.
func TestSafeBufferBytes(t *testing.T) {
	// Arrange
	var buffer SafeBuffer
	_, _ = buffer.Write([]byte("foobar"))

	// Act
	data := buffer.Bytes()

	// Assert
	require.EqualValues(t, []byte("foobar"), data)
}

// Test that the safe buffer produces no race.
func TestSafeBufferNoRace(t *testing.T) {
	// Arrange
	var buffer SafeBuffer
	var wg sync.WaitGroup

	writer := func(i int) {
		_, err := buffer.Write([]byte(fmt.Sprint(i)))
		wg.Done()
		require.NoError(t, err)
	}

	reader := func(i int) {
		_, err := buffer.Read(make([]byte, i))
		wg.Done()
		if !errors.Is(err, io.EOF) {
			require.NoError(t, err)
		}
	}

	exporter := func() {
		_ = buffer.Bytes()
		wg.Done()
	}

	// Act
	// There is a limit on 8128 simultaneously alive goroutines.
	count := 1000
	wg.Add(3 * count)
	for i := 1; i <= count; i++ {
		go writer(i)
		go reader(i)
		go exporter()
	}
	wg.Wait()

	// Assert
	// No race
}
