package testutil

import (
	"io"
	"os"

	pkgerrors "github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// Capture the stdout (including log output) and stderr content produced by
// a given function.
func CaptureOutput(f func()) (stdout []byte, stderr []byte, err error) {
	rescueStdout := os.Stdout
	rescueStderr := os.Stderr
	rOut, wOut, _ := os.Pipe()
	rErr, wErr, _ := os.Pipe()
	os.Stdout = wOut
	os.Stderr = wErr
	rescueLogOutput := logrus.StandardLogger().Out
	logrus.StandardLogger().SetOutput(wOut)
	// Restore the standard pipelines at the end.
	defer func() {
		os.Stdout = rescueStdout
		os.Stderr = rescueStderr
		logrus.StandardLogger().SetOutput(rescueLogOutput)
	}()

	// Execute function
	f()

	// Close the internal pipelines.
	wOut.Close()
	wErr.Close()

	// Reads the stdout
	stdout, err = io.ReadAll(rOut)
	if err != nil {
		err = pkgerrors.Wrap(err, "cannot read stdout")
		return
	}

	stderr, err = io.ReadAll(rErr)
	err = pkgerrors.Wrap(err, "cannot read stderr")
	return stdout, stderr, err
}
