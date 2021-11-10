package storkutil

import (
	"archive/tar"
	"compress/gzip"
	"io"

	"github.com/pkg/errors"
)

// Callback that accepts the TAR header of a file/directory/link,
// and a read content function - it retruns a binary content or error.
// Callback must return the flag indicating to need to continue walking
// (true - continue, false - stop).
type WalkCallback = func(header *tar.Header, read func() ([]byte, error)) bool

// General purpose walk function. It unpacks the Tarball and call the
// callback with each entry one-by-one.
// Current implementation doesn't support the subdirectory walking
// and read the content from the non-regular files.
func WalkFilesInTarball(tarball io.Reader, callback WalkCallback) error {
	gzipReader, err := gzip.NewReader(tarball)
	if err != nil {
		return err
	}
	defer gzipReader.Close()

	tarReader := tar.NewReader(gzipReader)

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}

		if err != nil {
			return errors.Wrap(err, "problem with read next header")
		}

		switch header.Typeflag {
		case tar.TypeReg:
			if !callback(header, func() ([]byte, error) {
				data := make([]byte, header.Size)
				_, err := tarReader.Read(data)
				if err == io.EOF {
					// The full content is read.
					err = nil
				}

				return data,
					errors.Wrapf(
						err,
						"cannot read content for the tarball file (%s)",
						header.Name,
					)
			}) {
				break
			}
		default:
			if !callback(header, func() ([]byte, error) {
				return nil, errors.New("reading unsupported")
			}) {
				break
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
		})

	if err != nil {
		return nil, err
	}

	return result, readErr
}
