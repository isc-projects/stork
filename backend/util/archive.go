package storkutil

import (
	"archive/tar"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
)

// The function traverses over the files in the directory and saves them
// into a target file. Returns error is any problem occur.
func SaveDirectoryContentToTarbal(sourceDirPath string, target *os.File) error {
	gzipWriter := gzip.NewWriter(target)
	defer gzipWriter.Close()
	tarWriter := tar.NewWriter(gzipWriter)
	defer tarWriter.Close()

	return filepath.Walk(sourceDirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return errors.Wrap(err, "IO error occurs")
		}

		// Prepare the TAR entry content
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

		// Write the TAR entry content
		err = tarWriter.WriteHeader(header)
		if err != nil {
			return errors.Wrap(err, "could not write header to TAR archive")
		}

		_, err = io.Copy(tarWriter, file)
		return errors.Wrap(err, "could not add the file to TAR archive")
	})
}
