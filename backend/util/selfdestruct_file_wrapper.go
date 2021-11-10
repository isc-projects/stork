package storkutil

import (
	"errors"
	"os"
)

// The simple wrapper on the os.File object that removes the file after a successful close.
// It implements only the io.ReadCloser interface.
type SelfDestructFileWrapper struct {
	file *os.File
}

func NewSelfDestructFileWrapper(file *os.File) *SelfDestructFileWrapper {
	return &SelfDestructFileWrapper{file}
}

func (w *SelfDestructFileWrapper) Name() string {
	return w.file.Name()
}

func (w *SelfDestructFileWrapper) Read(p []byte) (n int, err error) {
	return w.file.Read(p)
}

// The inner file should be removed even if the file was directly closed.
func (w *SelfDestructFileWrapper) Close() error {
	closeErr := w.file.Close()
	_, statErr := w.file.Stat()

	if !errors.Is(statErr, os.ErrNotExist) {
		return os.Remove(w.file.Name())
	}

	return closeErr
}
