package testutil

import (
	"io"
	"os"
)

// Capture the stdout and stderr content produced by a given function.
func CaptureOutput(f func()) (stdout []byte, stderr []byte, err error) {
	rescueStdout := os.Stdout
	rescueStderr := os.Stderr
	rOut, wOut, _ := os.Pipe()
	rErr, wErr, _ := os.Pipe()
	os.Stdout = wOut
	os.Stderr = wErr

	defer func() {
		os.Stdout = rescueStdout
		os.Stderr = rescueStderr
	}()

	f()

	wOut.Close()
	wErr.Close()

	stdout, err = io.ReadAll(rOut)
	if err != nil {
		return
	}

	stderr, err = io.ReadAll(rErr)
	return
}
