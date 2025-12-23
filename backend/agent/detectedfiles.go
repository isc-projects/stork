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
	fileType  detectedFileType
	path      string
	info      os.FileInfo
	chrootDir string
	executor  storkutil.CommandExecutor
}

// Creates a new detected daemon file instance. It returns an error if gathering
// file information fails.
func newDetectedDaemonFile(fileType detectedFileType, path, chrootDir string, executor storkutil.CommandExecutor) (*detectedDaemonFile, error) {
	info, err := executor.GetFileInfo(filepath.Join(chrootDir, path))
	if err != nil {
		return nil, err
	}
	return &detectedDaemonFile{
		fileType:  fileType,
		path:      filepath.Clean(path),
		info:      info,
		chrootDir: chrootDir,
		executor:  executor,
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

// Checks if the file has changed on disk since the information about it
// was gathered.
func (df *detectedDaemonFile) isChanged() bool {
	info, err := df.executor.GetFileInfo(filepath.Join(df.chrootDir, df.path))
	if err != nil {
		return true
	}
	return df.info.Size() != info.Size() || df.info.ModTime().Before(info.ModTime())
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

// Checks if the other set of detected files is a subset of the current set, and if
// the files are equal in terms of their type, path, size, and modification time. The
// current set can contain more files, as they could have been added after parsing the
// detected daemon's configuration (included files). If the function receiver or the argument
// are nil, the function returns false indicating that the sets are not the same. It is
// a special case for when the detected files are not set and should be always
// re-detected.
func (df *detectedDaemonFiles) isSame(other *detectedDaemonFiles) bool {
	if df == nil || other == nil || df.chrootDir != other.chrootDir || df.baseDir != other.baseDir || len(df.files) < len(other.files) {
		return false
	}
	for _, file := range df.files {
		if !slices.ContainsFunc(other.files, file.isEqual) {
			return false
		}
	}
	return true
}

// Checks if any of the files in the collection have changed on disk since the
// information about them was gathered. If the receiver is nil, the function always
// returns true to force re-detection and re-parsing of the detected daemon's
// configuration.
func (df *detectedDaemonFiles) isChanged() bool {
	if df == nil {
		return true
	}
	for _, file := range df.files {
		if file.isChanged() {
			return true
		}
	}
	return false
}
