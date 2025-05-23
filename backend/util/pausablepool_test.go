package storkutil

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// Test the error returned by the PausablePoolPausedError.
func TestPausablePoolPausedError(t *testing.T) {
	err := &PausablePoolPausedError{}
	require.Equal(t, err.Error(), "pausable pool is paused")
}

// Test the error returned by the PausablePoolStoppedError.
func TestPausablePoolStoppedError(t *testing.T) {
	err := &PausablePoolStoppedError{}
	require.Equal(t, err.Error(), "pausable pool is stopped")
}

// Test that the pool can be started, paused, resumed and stopped.
func TestPausablePool(t *testing.T) {
	// Create the pool.
	pool := NewPausablePool(10)

	// Create communication channels for each task.
	var (
		channels []chan int
		wg       sync.WaitGroup
	)
	wg.Add(10)
	for i := 0; i < 10; i++ {
		ch := make(chan int)
		channels = append(channels, ch)
		value := i
		pool.Submit(func() {
			wg.Done()
			// This is the blocking write to the channel.
			// The task is guaranteed to be blocked until we read
			// from the channel.
			ch <- value
		})
	}
	wg.Wait()

	// Read from several channels to unblock selected tasks.
	for i := 0; i < 5; i++ {
		v := <-channels[i]
		require.Equal(t, v, i)
	}

	// Schedule waiting for the pool to complete running tasks.
	var paused atomic.Bool
	go func() {
		// This call should block until all tasks are finished.
		pool.Pause()
		// Indicate that we finished waiting for the pool.
		paused.Store(true)
	}()

	// Verify that waiting didn't finish.
	require.Never(t, paused.Load, time.Second*1, time.Millisecond*10)

	// Read from the remaining channels to unblock the remaining tasks.
	for i := 5; i < 10; i++ {
		<-channels[i]
	}

	// This time waiting should finish successfully.
	require.Eventually(t, paused.Load, time.Second*1, time.Millisecond*10)

	// When the pool is paused, submitting new tasks should fail.
	err := pool.Submit(func() {})
	require.Error(t, err)
	pausablePoolPausedError := &PausablePoolPausedError{}
	require.ErrorAs(t, err, &pausablePoolPausedError)

	// Resume the pool.
	pool.Resume()

	// Submit new tasks.
	for i := 0; i < 10; i++ {
		value := i
		err = pool.Submit(func() {
			channels[i] <- value
		})
		require.NoError(t, err)
	}

	// Read from the channels to unblock the tasks.
	for i := 0; i < 10; i++ {
		v := <-channels[i]
		require.Equal(t, v, i)
	}

	// Stop the pool.
	pool.Stop()

	// Submitting new tasks should fail with different error.
	err = pool.Submit(func() {})
	require.Error(t, err)
	pausablePoolStoppedError := &PausablePoolStoppedError{}
	require.ErrorAs(t, err, &pausablePoolStoppedError)
}
