package storkutil

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"io"
	"os"
	"time"

	"github.com/pkg/errors"
)

// Helper object to creating the tarball (TAR archive).
// It allows to append the file or binary content.
// It doesn't support subfolders (not implemented).
// The owner is responsible for call the close method.
type TarballWriter struct {
	gzipWriter *gzip.Writer
	tarWriter  *tar.Writer
}

// Constructs a new tarball wrapper instance. It accepts a writer where it will
// pass the tarball bytes. The output data will be compressed using gzip.
func NewTarballWriter(target io.Writer) *TarballWriter {
	if target == nil {
		return nil
	}
	gzipWriter := gzip.NewWriter(target)
	tarWriter := tar.NewWriter(gzipWriter)
	return &TarballWriter{
		gzipWriter: gzipWriter,
		tarWriter:  tarWriter,
	}
}

// Add a file to the tarball. Path is a path to the physical file,
// info is the file info object that describes this file.
func (t *TarballWriter) AddFile(path string, info os.FileInfo) error {
	header, err := tar.FileInfoHeader(info, "")
	if err != nil {
		return errors.Wrap(err, "could not create a TAR header")
	}
	header.Name = path

	file, err := os.Open(path)
	if err != nil {
		return errors.Wrap(err, "could not open the file")
	}
	defer file.Close()

	return t.addRaw(header, file)
}

// Add a binary content to the tarball. The path is a location inside the tarball,
// content is binary data, modTime is modification time related to the content.
func (t *TarballWriter) AddContent(path string, content []byte, modTime time.Time) error {
	header := &tar.Header{
		Name:    path,
		Size:    int64(len(content)),
		ModTime: modTime,
		Mode:    0o444,
	}

	return t.addRaw(header, bytes.NewReader(content))
}

// Add a header and binary content to the tarball.
func (t *TarballWriter) addRaw(header *tar.Header, content io.Reader) error {
	err := t.tarWriter.WriteHeader(header)
	if err != nil {
		return errors.Wrap(err, "could not write header to TAR archive")
	}

	_, err = io.Copy(t.tarWriter, content)
	return errors.Wrap(err, "could not add the file to TAR archive")
}

// Close the internal writers.
func (t *TarballWriter) Close() {
	t.tarWriter.Close()
	t.gzipWriter.Close()
}
