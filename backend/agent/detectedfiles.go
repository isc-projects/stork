package agent

import (
	"os"
	"path/filepath"
	"slices"

	storkutil "isc.org/stork/util"
)

// A type of a detected daemon file, e.g., configuration file.
type detectedFileType int

const (
	detectedFileTypeConfig detectedFileType = iota
	detectedFileTypeRndcKey
)

// A structure representing a single file (typically a configuration file)
// used by a detected daemon. For example, a BIND 9 config file, a
// PowerDNS config file, or an rndc.key file.
//
// It also holds supplementary information about the file, including file size
// and the modification time. It allows for detecting whether the file has
// changed since the last detection, and skip parsing the file if it has not
// changed.
type detectedDaemonFile struct {
	fileType detectedFileType
	path     string
	info     os.FileInfo
}

// Creates a new detected daemon file instance. It returns an error if gathering
// file information fails.
func newDetectedDaemonFile(fileType detectedFileType, path, chrootDir string, executor storkutil.CommandExecutor) (*detectedDaemonFile, error) {
	info, err := executor.GetFileInfo(filepath.Join(chrootDir, path))
	if err != nil {
		return nil, err
	}
	return &detectedDaemonFile{
		fileType: fileType,
		path:     filepath.Clean(path),
		info:     info,
	}, nil
}

// Checks if the specified file is equal to the current file. It compares
// the file type, path, size, and modification time.
func (df *detectedDaemonFile) isEqual(other *detectedDaemonFile) bool {
	return df.fileType == other.fileType &&
		df.path == other.path &&
		df.info.Size() == other.info.Size() &&
		df.info.ModTime().Equal(other.info.ModTime())
}

// A structure representing a collection of the files used by the detected
// daemon. In addition, it optionally holds the chroot directory and the base
// directory of the detected daemon.
type detectedDaemonFiles struct {
	files     []*detectedDaemonFile
	chrootDir string
	baseDir   string
}

// Creates a new detected daemon files instance.
func newDetectedDaemonFiles(chrootDir, baseDir string) *detectedDaemonFiles {
	if chrootDir != "" {
		chrootDir = filepath.Clean(chrootDir)
	}
	if baseDir != "" {
		baseDir = filepath.Clean(baseDir)
	}
	return &detectedDaemonFiles{
		chrootDir: chrootDir,
		baseDir:   baseDir,
	}
}

// Adds a file to the collection.
func (df *detectedDaemonFiles) addFile(fileType detectedFileType, path string, executor storkutil.CommandExecutor) error {
	detectedFile, err := newDetectedDaemonFile(fileType, path, df.chrootDir, executor)
	if err != nil {
		return err
	}
	df.files = append(df.files, detectedFile)
	return nil
}

// Returns the path to the first file having the specified type.
func (df *detectedDaemonFiles) getFirstFilePathByType(fileType detectedFileType) string {
	for _, file := range df.files {
		if file.fileType == fileType {
			return file.path
		}
	}
	return ""
}

// Check if the specified file sets contain the same set of files neglecting their
// order.
func (df *detectedDaemonFiles) isEqual(other *detectedDaemonFiles) bool {
	if df.chrootDir != other.chrootDir || df.baseDir != other.baseDir || len(df.files) != len(other.files) {
		return false
	}
	for _, file := range df.files {
		if !slices.ContainsFunc(other.files, file.isEqual) {
			return false
		}
	}
	return true
}
