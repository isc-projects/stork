package storkutil

import (
	"archive/tar"
	"compress/gzip"
	"errors"
	"io"

	pkgerrors "github.com/pkg/errors"
)

// Callback that accepts the TAR header of a file/directory/link,
// and a read content function - it returns a binary content or error.
// Callback must return the flag indicating to need to continue walking
// (true - continue, false - stop).
type WalkCallback = func(header *tar.Header, read func() ([]byte, error)) bool

// General purpose walk function. It unpacks the Tarball and calls the
// callback with each entry one-by-one.
// Current implementation doesn't support the subdirectory walking
// and reading the content from the non-regular files.
func WalkFilesInTarball(tarball io.Reader, callback WalkCallback) error {
	gzipReader, err := gzip.NewReader(tarball)
	if err != nil {
		return pkgerrors.Wrap(err, "invalid tarball")
	}
	defer gzipReader.Close()

	tarReader := tar.NewReader(gzipReader)

	for {
		header, err := tarReader.Next()
		if errors.Is(err, io.EOF) {
			break
		}

		if err != nil {
			return pkgerrors.Wrap(err, "problem reading next header")
		}

		switch header.Typeflag {
		case tar.TypeReg:
			if !callback(header, func() ([]byte, error) {
				data := make([]byte, header.Size)
				_, err := tarReader.Read(data)
				if errors.Is(err, io.EOF) {
					// The full content is read.
					err = nil
				}

				return data,
					pkgerrors.Wrapf(
						err,
						"cannot read content of the tarball file (%s)",
						header.Name,
					)
			}) {
				return nil
			}
		default:
			if !callback(header, func() ([]byte, error) {
				return nil, pkgerrors.New("reading unsupported")
			}) {
				return nil
			}
		}
	}

	return nil
}

// List the files inside the tarball.
func ListFilesInTarball(tarball io.Reader) ([]string, error) {
	result := make([]string, 0)

	err := WalkFilesInTarball(tarball,
		func(header *tar.Header, read func() ([]byte, error)) bool {
			if header.Typeflag == tar.TypeReg {
				result = append(result, header.Name)
			}
			return true
		})
	return result, err
}

// Search for a specific file in the tarball.
// If the file is found returns its binary content.
// If file doesn't exist in the tarball returns nil content and no error.
// Returns error if the tarball is unavailable or any reading problem occurs.
func SearchFileInTarball(tarball io.Reader, filename string) ([]byte, error) {
	var result []byte
	var readErr error

	err := WalkFilesInTarball(tarball,
		func(header *tar.Header, read func() ([]byte, error)) bool {
			if header.Typeflag != tar.TypeReg {
				return true
			}
			if header.Name == filename {
				result, readErr = read()
				return false
			}
			return true
		},
	)
	if err != nil {
		return nil, err
	}

	return result, readErr
}
