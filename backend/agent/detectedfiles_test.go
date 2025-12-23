package agent

import (
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	gomock "go.uber.org/mock/gomock"
)

// Test instantiation of the detected daemon file with chroot directory.
func TestNewDetectedDaemonFileChroot(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	executor := NewMockCommandExecutor(ctrl)
	executor.EXPECT().GetFileInfo("/chroot/etc/bind/config/config.conf").Return(&testFileInfo{}, nil)

	detectedFile, err := newDetectedDaemonFile(detectedFileTypeConfig, "/etc/bind/config/config.conf", "/chroot", executor)
	require.NoError(t, err)
	require.Equal(t, detectedFileTypeConfig, detectedFile.fileType)
	require.Equal(t, "/etc/bind/config/config.conf", detectedFile.path)
	require.Equal(t, &testFileInfo{}, detectedFile.info)
}

// Test instantiation of the detected daemon file without chroot directory.
func TestNewDetectedDaemonFileNoChroot(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	executor := NewMockCommandExecutor(ctrl)
	executor.EXPECT().GetFileInfo("/etc/bind/config/config.conf").Return(&testFileInfo{}, nil)

	detectedFile, err := newDetectedDaemonFile(detectedFileTypeConfig, "/etc/bind/config/config.conf", "", executor)
	require.NoError(t, err)
	require.Equal(t, detectedFileTypeConfig, detectedFile.fileType)
	require.Equal(t, "/etc/bind/config/config.conf", detectedFile.path)
	require.Equal(t, &testFileInfo{}, detectedFile.info)
}

// Test that an error is returned when instantiating the detected daemon file
// and errors occurs.
func TestNewDetectedDaemonFileError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	executor := NewMockCommandExecutor(ctrl)
	executor.EXPECT().GetFileInfo("/etc/bind/config/config.conf").Return(nil, errors.New("test error"))

	detectedFile, err := newDetectedDaemonFile(detectedFileTypeConfig, "/etc/bind/config/config.conf", "", executor)
	require.Error(t, err)
	require.ErrorContains(t, err, "test error")
	require.Nil(t, detectedFile)
}

// Test instantiation of the detected daemon files.
func TestNewDetectedDaemonFiles(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	fileInfo := &testFileInfo{}
	executor := NewMockCommandExecutor(ctrl)
	executor.EXPECT().GetFileInfo("/chroot/etc/bind/config/config.conf").Return(fileInfo, nil)
	executor.EXPECT().GetFileInfo("/chroot/etc/bind/rndc.key").Return(fileInfo, nil)

	files := newDetectedDaemonFiles("/chroot/.", "/base/../base")
	err := files.addFile(detectedFileTypeConfig, "/etc/bind/config/config.conf", executor)
	require.NoError(t, err)
	err = files.addFile(detectedFileTypeRndcKey, "/etc/bind/rndc.key", executor)
	require.NoError(t, err)
	require.Equal(t, "/chroot", files.chrootDir)
	require.Equal(t, "/base", files.baseDir)
	require.Len(t, files.files, 2)
	require.Equal(t, detectedFileTypeConfig, files.files[0].fileType)
	require.Equal(t, "/etc/bind/config/config.conf", files.files[0].path)
	require.Equal(t, fileInfo, files.files[0].info)
	require.Equal(t, detectedFileTypeRndcKey, files.files[1].fileType)
	require.Equal(t, "/etc/bind/rndc.key", files.files[1].path)
	require.Equal(t, fileInfo, files.files[1].info)
}

// Test getting the first file path by type.
func TestDetectedDaemonFilesGetFirstFilePathByType(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	fileInfo := &testFileInfo{}
	executor := NewMockCommandExecutor(ctrl)
	executor.EXPECT().GetFileInfo("/etc/bind/config/config.conf").Return(fileInfo, nil)
	executor.EXPECT().GetFileInfo("/etc/bind/rndc.key").Return(fileInfo, nil)

	files := newDetectedDaemonFiles("", "")
	err := files.addFile(detectedFileTypeConfig, "/etc/bind/config/config.conf", executor)
	require.NoError(t, err)
	err = files.addFile(detectedFileTypeRndcKey, "/etc/bind/rndc.key", executor)
	require.NoError(t, err)
	path := files.getFirstFilePathByType(detectedFileTypeConfig)
	require.Equal(t, "/etc/bind/config/config.conf", path)
	require.Equal(t, fileInfo, files.files[0].info)
	path = files.getFirstFilePathByType(detectedFileTypeRndcKey)
	require.Equal(t, "/etc/bind/rndc.key", path)
	require.Equal(t, fileInfo, files.files[1].info)
}

// Test that an error is returned when adding a new file to the set of
// detected files fails due to an IO error while getting the file information.
func TestDetectDaemonFilesAddFileError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	executor := NewMockCommandExecutor(ctrl)
	executor.EXPECT().GetFileInfo("/etc/bind/config/config.conf").Return(nil, errors.New("test error"))

	files := newDetectedDaemonFiles("", "")
	err := files.addFile(detectedFileTypeConfig, "/etc/bind/config/config.conf", executor)
	require.ErrorContains(t, err, "test error")
}

// Test that it is correctly verified that two sets of detected files are equal.
func TestDetectedDaemonFilesIsEqual(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	executor := NewMockCommandExecutor(ctrl)
	executor.EXPECT().GetFileInfo("/etc/bind/config/config.conf").AnyTimes().Return(&testFileInfo{}, nil)
	executor.EXPECT().GetFileInfo("/etc/bind/rndc.key").AnyTimes().Return(&testFileInfo{}, nil)

	files1 := newDetectedDaemonFiles("", "")
	files2 := newDetectedDaemonFiles("", "")
	require.True(t, files1.isEqual(files2))

	err := files1.addFile(detectedFileTypeConfig, "/etc/bind/config/config.conf", executor)
	require.NoError(t, err)
	err = files2.addFile(detectedFileTypeConfig, "/etc/bind/config/config.conf", executor)
	require.NoError(t, err)
	require.True(t, files1.isEqual(files2))

	files1.addFile(detectedFileTypeRndcKey, "/etc/bind/rndc.key", executor)
	files2.addFile(detectedFileTypeRndcKey, "/etc/bind/rndc.key", executor)
	require.True(t, files1.isEqual(files2))
}

// Test that it is correctly verified that two sets of detected files are equal
// even if the files are in different order.
func TestDetectedDaemonFilesIsEqualOutOfOrder(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	executor := NewMockCommandExecutor(ctrl)
	executor.EXPECT().GetFileInfo("/etc/bind/config/config.conf").AnyTimes().Return(&testFileInfo{}, nil)
	executor.EXPECT().GetFileInfo("/etc/bind/rndc.key").AnyTimes().Return(&testFileInfo{}, nil)

	files1 := newDetectedDaemonFiles("", "")
	files2 := newDetectedDaemonFiles("", "")
	err := files1.addFile(detectedFileTypeConfig, "/etc/bind/config/config.conf", executor)
	require.NoError(t, err)
	err = files1.addFile(detectedFileTypeRndcKey, "/etc/bind/rndc.key", executor)
	require.NoError(t, err)
	err = files2.addFile(detectedFileTypeRndcKey, "/etc/bind/rndc.key", executor)
	require.NoError(t, err)
	err = files2.addFile(detectedFileTypeConfig, "/etc/bind/config/config.conf", executor)
	require.NoError(t, err)
	require.True(t, files1.isEqual(files2))
}

// Test that it is correctly verified that two sets of detected files are not equal
// if the chroot directories are different.
func TestDetectedDaemonFilesIsEqualDifferentChrootDir(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	executor := NewMockCommandExecutor(ctrl)
	executor.EXPECT().GetFileInfo("/etc/bind/config/config.conf").AnyTimes().Return(&testFileInfo{}, nil)

	files1 := newDetectedDaemonFiles("/chroot1", "")
	files2 := newDetectedDaemonFiles("/chroot2", "")
	require.False(t, files1.isEqual(files2))
}

// Test that it is correctly verified that two sets of detected files are not equal
// if the base directories are different.
func TestDetectedDaemonFilesIsEqualDifferentBaseDir(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	executor := NewMockCommandExecutor(ctrl)
	executor.EXPECT().GetFileInfo("/etc/bind/config/config.conf").AnyTimes().Return(&testFileInfo{}, nil)

	files1 := newDetectedDaemonFiles("", "/base1")
	files2 := newDetectedDaemonFiles("", "/base2")
	require.False(t, files1.isEqual(files2))
}

// Test that it is correctly verified that two sets of detected files are not equal
// if the file paths are different.
func TestDetectedDaemonFilesIsEqualDifferentFilePaths(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	executor := NewMockCommandExecutor(ctrl)
	executor.EXPECT().GetFileInfo("/etc/bind/config/config.conf").AnyTimes().Return(&testFileInfo{}, nil)
	executor.EXPECT().GetFileInfo("/etc/bind/config/rndc.key").AnyTimes().Return(&testFileInfo{}, nil)

	files1 := newDetectedDaemonFiles("", "")
	files2 := newDetectedDaemonFiles("", "")

	err := files1.addFile(detectedFileTypeConfig, "/etc/bind/config/config.conf", executor)
	require.NoError(t, err)
	err = files2.addFile(detectedFileTypeConfig, "/etc/bind/config/rndc.key", executor)
	require.NoError(t, err)
	require.False(t, files1.isEqual(files2))
}

// Test that it is correctly verified that two sets of detected files are not equal
// if the file types are different.
func TestDetectedDaemonFilesIsEqualDifferentFileTypes(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	executor := NewMockCommandExecutor(ctrl)
	executor.EXPECT().GetFileInfo("/etc/bind/config/config.conf").AnyTimes().Return(&testFileInfo{}, nil)

	files1 := newDetectedDaemonFiles("", "")
	files2 := newDetectedDaemonFiles("", "")
	err := files1.addFile(detectedFileTypeConfig, "/etc/bind/config/config.conf", executor)
	require.NoError(t, err)
	err = files2.addFile(detectedFileTypeRndcKey, "/etc/bind/config/config.conf", executor)
	require.NoError(t, err)
	require.False(t, files1.isEqual(files2))
}

// Test that it is correctly verified that two sets of detected files are not equal
// if the file sizes are different.
func TestDetectedDaemonFilesIsEqualDifferentFileSizes(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	executor1 := NewMockCommandExecutor(ctrl)
	executor1.EXPECT().GetFileInfo("/etc/bind/config/config.conf").AnyTimes().Return(&testFileInfo{size: 100}, nil)
	executor2 := NewMockCommandExecutor(ctrl)
	executor2.EXPECT().GetFileInfo("/etc/bind/config/config.conf").AnyTimes().Return(&testFileInfo{size: 200}, nil)

	files1 := newDetectedDaemonFiles("", "")
	files2 := newDetectedDaemonFiles("", "")
	err := files1.addFile(detectedFileTypeConfig, "/etc/bind/config/config.conf", executor1)
	require.NoError(t, err)
	err = files2.addFile(detectedFileTypeConfig, "/etc/bind/config/config.conf", executor2)
	require.NoError(t, err)
	require.False(t, files1.isEqual(files2))
}

// Test that it is correctly verified that two sets of detected files are not equal
// if the file modification times are different.
func TestDetectedDaemonFilesIsEqualDifferentFileModificationTimes(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	executor1 := NewMockCommandExecutor(ctrl)
	executor1.EXPECT().GetFileInfo("/etc/bind/config/config.conf").AnyTimes().Return(&testFileInfo{modTime: time.Unix(0, 0)}, nil)
	executor2 := NewMockCommandExecutor(ctrl)
	executor2.EXPECT().GetFileInfo("/etc/bind/config/config.conf").AnyTimes().Return(&testFileInfo{modTime: time.Unix(0, 1)}, nil)

	files1 := newDetectedDaemonFiles("", "")
	files2 := newDetectedDaemonFiles("", "")
	err := files1.addFile(detectedFileTypeConfig, "/etc/bind/config/config.conf", executor1)
	require.NoError(t, err)
	err = files2.addFile(detectedFileTypeConfig, "/etc/bind/config/config.conf", executor2)
	require.NoError(t, err)
	require.False(t, files1.isEqual(files2))
}
