package testutil

import (
	"errors"
	"io"
	"os"
	"sync"
	"time"

	pkgerrors "github.com/pkg/errors"
)

// Capture the stdout and stderr content produced by a given function.
// During the function execution it calls a chunk function with a partial
// stdout. The maximum size of the chunk is limited by the size parameter.
// The chunk callback is called with the already caputured stdout
// and the number of read bytes from the last chunk.
// If the chunk callback is nil or size equals to zero then the callback is not
// used.
func CaptureOutput(f func(), chunk func(stdout []byte, n int), size int) (stdout []byte, stderr []byte, err error) {
	rescueStdout := os.Stdout
	rescueStderr := os.Stderr
	rOut, wOut, _ := os.Pipe()
	rErr, wErr, _ := os.Pipe()
	os.Stdout = wOut
	os.Stderr = wErr

	// Restore the standard pipelines at the end.
	defer func() {
		os.Stdout = rescueStdout
		os.Stderr = rescueStderr
	}()

	// The stdout pipeline doesn't support the seek operation.
	// This helper reads some bytes from the pipe and stores them in a buffer.
	// Additionally, it sets the deadline for reading (100 miliseconds).
	var stdoutBuffer SafeBuffer
	readStdout := func(b []byte) (int, error) {
		err = rOut.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
		if err != nil {
			return 0, pkgerrors.Wrap(err, "cannot set the deadline")
		}
		n, readErr := rOut.Read(b)
		if n > 0 {
			_, err = stdoutBuffer.Write(b[:n])
			if err != nil {
				return 0, pkgerrors.Wrap(err, "cannot write to stdout buffer")
			}
		}
		return n, readErr
	}

	// Guard waiting for the end of the function execution.
	var wgExecute sync.WaitGroup
	wgExecute.Add(1)

	// The function executor.
	go func() {
		f()
		wgExecute.Done()
	}()

	// Guard waiting for end the chunk read.
	var wgChunk sync.WaitGroup

	// Reads the chunks of the stdout.
	if chunk != nil && size > 0 {
		wgChunk.Add(1)
		go func() {
			// Chunk buffer.
			stdoutData := make([]byte, size)
			for {
				// Reads bytes with the reading deadline.
				n, err := readStdout(stdoutData)
				// The number of bytes equals 0 if the reading is interrupted
				// by the deadline.
				chunk(stdoutBuffer.Bytes(), n)

				// Deadline exceed.
				if errors.Is(pkgerrors.Cause(err), os.ErrDeadlineExceeded) {
					continue
				}

				// Stops at the end of the file or any error.
				if err != nil {
					break
				}
			}
			wgChunk.Done()
		}()
	}

	// Block until the function runs.
	wgExecute.Wait()

	// Close the internal pipelines.
	wOut.Close()
	wErr.Close()

	// Wait for end chunk read.
	wgChunk.Wait()

	// Reads the bytes if the chunking is not used.
	stdoutTail, err := io.ReadAll(rOut)
	if err != nil {
		err = pkgerrors.Wrap(err, "cannot read stdout")
		return
	}
	_, err = stdoutBuffer.Write(stdoutTail)
	if err != nil {
		err = pkgerrors.Wrap(err, "cannot write to stdout buffer")
		return stdout, stderr, err
	}

	// Prepare outputs.
	stdout = stdoutBuffer.Bytes()
	stderr, err = io.ReadAll(rErr)
	err = pkgerrors.Wrap(err, "cannot read stderr")
	return stdout, stderr, err
}
