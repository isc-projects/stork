package agent

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	gomock "go.uber.org/mock/gomock"
	bind9config "isc.org/stork/daemoncfg/bind9"
	"isc.org/stork/datamodel/daemonname"
	"isc.org/stork/datamodel/protocoltype"
	"isc.org/stork/testutil"
	storkutil "isc.org/stork/util"
)

var _ = (os.FileInfo)((*testFileInfo)(nil))

//go:generate mockgen -package=agent -destination=bind9mock_test.go -mock_names=bind9FileParser=MockBind9FileParser,zoneInventory=MockZoneInventory isc.org/stork/agent bind9FileParser,zoneInventory

// Test that the function correctly checks if two BIND 9 daemons are the same.
// Note that it is not checking them for equality. It merely checks if they
// represent the same daemon in terms of their name, access points, and detected
// files.
func TestBind9DaemonIsSame(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	executor := NewMockCommandExecutor(ctrl)
	executor.EXPECT().GetFileInfo("/etc/bind/named.conf").AnyTimes().Return(&testFileInfo{}, nil)

	detectedFiles := newDetectedDaemonFiles("", "")
	err := detectedFiles.addFile(detectedFileTypeConfig, "/etc/bind/named.conf", executor)
	require.NoError(t, err)

	comparedDaemon := &Bind9Daemon{
		dnsDaemonImpl: dnsDaemonImpl{
			daemon: daemon{
				Name: daemonname.Bind9,
				AccessPoints: []AccessPoint{
					{
						Type:     AccessPointControl,
						Address:  "127.0.0.1",
						Port:     8081,
						Protocol: protocoltype.HTTP,
					},
				},
			},
			detectedFiles: detectedFiles,
		},
	}

	t.Run("same daemon", func(t *testing.T) {
		otherDaemon := &Bind9Daemon{
			dnsDaemonImpl: dnsDaemonImpl{
				daemon: daemon{
					Name: daemonname.Bind9,
					AccessPoints: []AccessPoint{
						{
							Type:     AccessPointControl,
							Address:  "127.0.0.1",
							Port:     8081,
							Protocol: protocoltype.HTTP,
						},
					},
				},
				detectedFiles: detectedFiles,
			},
		}
		require.True(t, comparedDaemon.IsSame(otherDaemon))
	})

	t.Run("different daemon name", func(t *testing.T) {
		otherDaemon := &Bind9Daemon{
			dnsDaemonImpl: dnsDaemonImpl{
				daemon: daemon{
					Name: daemonname.PDNS,
					AccessPoints: []AccessPoint{
						{
							Type:     AccessPointControl,
							Address:  "127.0.0.1",
							Port:     8081,
							Protocol: protocoltype.HTTP,
						},
					},
				},
				detectedFiles: detectedFiles,
			},
		}
		require.False(t, comparedDaemon.IsSame(otherDaemon))
	})

	t.Run("different access points", func(t *testing.T) {
		otherDaemon := &Bind9Daemon{
			dnsDaemonImpl: dnsDaemonImpl{
				daemon: daemon{
					Name: daemonname.Bind9,
					AccessPoints: []AccessPoint{
						{
							Type:     AccessPointControl,
							Address:  "127.0.0.1",
							Port:     8082,
							Protocol: protocoltype.HTTP,
						},
					},
				},
				detectedFiles: detectedFiles,
			},
		}
		require.False(t, comparedDaemon.IsSame(otherDaemon))
	})

	t.Run("different detected files", func(t *testing.T) {
		otherDaemon := &Bind9Daemon{
			dnsDaemonImpl: dnsDaemonImpl{
				daemon: daemon{
					Name: daemonname.Bind9,
					AccessPoints: []AccessPoint{
						{
							Type:     AccessPointControl,
							Address:  "127.0.0.1",
							Port:     8081,
							Protocol: protocoltype.HTTP,
						},
					},
				},
				detectedFiles: nil,
			},
		}
		require.False(t, comparedDaemon.IsSame(otherDaemon))
	})

	t.Run("not a BIND 9 daemon", func(t *testing.T) {
		otherDaemon := &pdnsDaemon{
			dnsDaemonImpl: dnsDaemonImpl{
				daemon: daemon{
					Name: daemonname.Bind9,
					AccessPoints: []AccessPoint{
						{
							Type:     AccessPointControl,
							Address:  "127.0.0.1",
							Port:     8081,
							Protocol: protocoltype.HTTP,
						},
					},
				},
				detectedFiles: detectedFiles,
			},
		}
		require.False(t, comparedDaemon.IsSame(otherDaemon))
	})
}

// Test the state is refreshed properly. It should fetch the zone inventory
// data.
func TestBind9RefreshState(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	agentManager := NewMockAgentManager(ctrl)

	zoneInventory := NewMockZoneInventory(ctrl)
	zoneInventory.EXPECT().populate(gomock.Any()).Return(nil, nil)
	zoneInventory.EXPECT().getCurrentState().Return(&zoneInventoryState{})

	daemon := &Bind9Daemon{
		dnsDaemonImpl: dnsDaemonImpl{
			zoneInventory: zoneInventory,
		},
	}

	// Act
	err := daemon.RefreshState(t.Context(), agentManager)

	// Assert
	require.NoError(t, err)
}

// Test that the zone inventory can be accessed.
func TestBind9GetZoneInventory(t *testing.T) {
	daemon := &Bind9Daemon{
		dnsDaemonImpl: dnsDaemonImpl{
			zoneInventory: &zoneInventoryImpl{},
		},
	}
	inventory := daemon.getZoneInventory()
	require.Equal(t, daemon.zoneInventory, inventory)
}

// Check if getPotentialNamedConfLocations returns paths.
func TestGetPotentialNamedConfLocations(t *testing.T) {
	paths := getPotentialNamedConfLocations()
	require.Greater(t, len(paths), 1)
}

// Test that the system command executor returns a proper output.
func TestSystemCommandExecutorOutput(t *testing.T) {
	// Arrange
	executor := storkutil.NewSystemCommandExecutor()

	// Act
	output, err := executor.Output("echo", "-n", "foo")

	// Assert
	require.NoError(t, err)
	require.EqualValues(t, "foo", string(output))
}

// Test that the system command executor returns an error for invalid command.
func TestSystemCommandExecutorOnFail(t *testing.T) {
	// Arrange
	executor := storkutil.NewSystemCommandExecutor()

	// Act
	output, err := executor.Output("non-exist-command")

	// Assert
	require.Error(t, err)
	require.Nil(t, output)
}

func TestParseNamedDefaultPath(t *testing.T) {
	// Define input data
	input := `default paths:
                named configuration:  /some/path/named.conf
                rndc configuration:   /other/path/rndc.conf`

	// Convert input data to []byte
	output := []byte(input)

	// Call parseNamedDefaultPath with the output
	NamedConf := parseNamedDefaultPath(output)

	// Assert that the parsed strings are correct
	require.Equal(t, "/some/path/named.conf", NamedConf)
}

// Old BIND 9 versions don't print the default paths.
// This test uses actual BIND 9.11.5 output. Makes sure
// that the function doesn't panic.
func TestParseNamedDefaultPathForOldBind9Versions(t *testing.T) {
	// Define input data (actual output from BIND 9.11.5)
	input := `BIND 9.11.5-P4-5.1+deb10u8-Debian (Extended Support Version) <id:998753c>
running on Linux x86_64 4.19.0-22-amd64 #1 SMP Debian 4.19.260-1 (2022-09-29)
built by make with '--build=x86_64-linux-gnu' '--prefix=/usr' '--includedir=/usr/include'
'--mandir=/usr/share/man' '--infodir=/usr/share/info' '--sysconfdir=/etc' '--localstatedir=/var'
'--disable-silent-rules' '--libdir=/usr/lib/x86_64-linux-gnu' '--libexecdir=/usr/lib/x86_64-linux-gnu'
'--disable-maintainer-mode' '--disable-dependency-tracking' '--libdir=/usr/lib/x86_64-linux-gnu'
'--sysconfdir=/etc/bind' '--with-python=python3' '--localstatedir=/' '--enable-threads'
'--enable-largefile' '--with-libtool' '--enable-shared' '--enable-static' '--with-gost=no'
'--with-openssl=/usr' '--with-gssapi=/usr' '--disable-isc-spnego' '--with-libidn2'
'--with-libjson=/usr' '--with-lmdb=/usr' '--with-gnu-ld' '--with-geoip=/usr' '--with-atf=no'
'--enable-ipv6' '--enable-rrl' '--enable-filter-aaaa' '--enable-native-pkcs11'
'--with-pkcs11=/usr/lib/softhsm/libsofthsm2.so' '--with-randomdev=/dev/urandom'
'--enable-dnstap' 'build_alias=x86_64-linux-gnu' 'CFLAGS=-g -O2
-fdebug-prefix-map=/build/bind9-S4LHfc/bind9-9.11.5.P4+dfsg=. -fstack-protector-strong
-Wformat -Werror=format-security -fno-strict-aliasing -fno-delete-null-pointer-checks
-DNO_VERSION_DATE -DDIG_SIGCHASE' 'LDFLAGS=-Wl,-z,relro -Wl,-z,now' 'CPPFLAGS=-Wdate-time
-D_FORTIFY_SOURCE=2'
compiled by GCC 8.3.0
compiled with OpenSSL version: OpenSSL 1.1.1n  15 Mar 2022
linked to OpenSSL version: OpenSSL 1.1.1n  15 Mar 2022
compiled with libxml2 version: 2.9.4
linked to libxml2 version: 20904
compiled with libjson-c version: 0.12.1
linked to libjson-c version: 0.12.1
threads support is enabled`

	// Convert input data to []byte
	output := []byte(input)

	// Call parseNamedDefaultPath with the output
	namedConf := parseNamedDefaultPath(output)

	// Assert that the returned values are empty
	require.Equal(t, "", namedConf)
}

// A mock implementation of the os.FileInfo interface for the testing purposes.
type testFileInfo struct {
	size    int64
	modTime time.Time
}

// Returns empty file name.
func (t *testFileInfo) Name() string {
	return ""
}

// Returns false indicating that the file is not a directory.
func (t *testFileInfo) IsDir() bool {
	return false
}

// Returns 0 as a file mode.
func (t *testFileInfo) Mode() os.FileMode {
	return 0
}

// Returns configured modification time of the file.
func (t *testFileInfo) ModTime() time.Time {
	return t.modTime
}

// Returns configured size of the file.
func (t *testFileInfo) Size() int64 {
	return t.size
}

// Returns nil as the system interface is not implemented.
func (t *testFileInfo) Sys() any {
	return nil
}

// The command executor implementation for the testing purposes.
// It implements the builder pattern for the configuration methods.
type testCommandExecutor struct {
	configPathInNamedOutput *string
	checkConfOutputs        map[string]string
	rndcStatusError         error
	rndcStatus              string
	fileInfos               map[string]os.FileInfo
}

// Constructs a new instance of the test command executor.
func newTestCommandExecutor() *testCommandExecutor {
	return &testCommandExecutor{
		checkConfOutputs: map[string]string{},
		rndcStatus:       "Server is up and running",
		fileInfos:        map[string]os.FileInfo{},
	}
}

// Clears all added outputs and the config path used in the named -V output.
func (e *testCommandExecutor) clear() *testCommandExecutor {
	e.checkConfOutputs = map[string]string{}
	e.configPathInNamedOutput = nil
	return e
}

// Add a output mock of the named-checkconf call. It accepts the configuration
// path program and expected output text.
func (e *testCommandExecutor) addCheckConfOutput(path, content string) *testCommandExecutor {
	e.checkConfOutputs[path] = content
	return e
}

// Set the named configuration path used in the output of the named -V call.
// If the path is not set, the output doesn't contain the configuration path.
func (e *testCommandExecutor) setConfigPathInNamedOutput(path string) *testCommandExecutor {
	e.configPathInNamedOutput = &path
	return e
}

// Set the output from the RNDC status command.
func (e *testCommandExecutor) setRndcStatus(status string, err error) *testCommandExecutor {
	e.rndcStatus = status
	e.rndcStatusError = err
	return e
}

// Pretends to run named-checkconf, but instead does a simple read of the
// specified files contents, similar to "cat" command.
func (e *testCommandExecutor) Output(command string, args ...string) ([]byte, error) {
	if strings.Contains(command, "named-checkconf") {
		root := "/"
		for i := 0; i < len(args)-2; i++ {
			if args[i] == "-t" {
				root = args[i+1]
				break
			}
		}

		config := args[len(args)-1]

		fullPath := path.Join(root, config)
		content, ok := e.checkConfOutputs[fullPath]

		if !ok {
			// Reading failed.
			return nil, errors.New("missing configuration")
		}
		return []byte(content), nil
	}

	if strings.HasSuffix(command, "named") && len(args) > 0 && args[0] == "-V" {
		// Pretending to run named -V
		namedPathEntry := ""
		if e.configPathInNamedOutput != nil {
			namedPathEntry = fmt.Sprintf("named configuration:  %s", *e.configPathInNamedOutput)
		}

		text := fmt.Sprintf(`default paths:
		%s
		rndc configuration:   /other/path/rndc.conf`, namedPathEntry)

		return []byte(text), nil
	}

	if strings.HasSuffix(command, "rndc") {
		if len(args) > 0 && args[len(args)-1] == "status" {
			return []byte(e.rndcStatus), e.rndcStatusError
		}
		return []byte("unknown command"), nil
	}

	if strings.HasSuffix(command, "kea-ctrl-agent") && len(args) == 1 && args[0] == "-v" {
		return []byte("3.0.1"), nil
	}

	return nil, errors.Errorf("unknown command: %s %s", command, strings.Join(args, " "))
}

// Looks for a given command in the system PATH and returns absolute path if found.
// (This is the standard behavior that we don't override in tests here.)
func (e *testCommandExecutor) LookPath(command string) (string, error) {
	allowedCommands := []string{"named-checkconf", "named", "rndc"}
	for _, allowedCommand := range allowedCommands {
		if command == allowedCommand {
			return "/usr/sbin/" + command, nil
		}
	}
	return "", errors.New("command not found")
}

// Check if there is an output for a given configuration path.
func (e *testCommandExecutor) IsFileExist(path string) bool {
	for configuredPath := range e.checkConfOutputs {
		if configuredPath == path {
			return true
		}
	}
	return false
}

// Adds file information for a given path.
func (e *testCommandExecutor) addFileInfo(path string, info os.FileInfo) *testCommandExecutor {
	e.fileInfos[path] = info
	return e
}

// Returns file information for a given path.
func (e *testCommandExecutor) GetFileInfo(path string) (os.FileInfo, error) {
	info, ok := e.fileInfos[path]
	if !ok {
		return nil, errors.New("file not found")
	}
	return info, nil
}

// Checks detection STEP 1: if BIND9 detection takes -c parameter into consideration.
func TestDetectBind9Step1ProcessCmdLine(t *testing.T) {
	// Create alternate config files for each step.
	sandbox := testutil.NewSandbox()
	defer sandbox.Close()
	config1Path := path.Join(sandbox.BasePath, "step1.conf")
	config1 := `key "foo" { algorithm "hmac-sha256"; secret "abcd";};
                controls { inet 1.1.1.1 port 1111 allow { localhost; } keys { "foo"; "bar"; }; };`
	_, err := sandbox.Write("step1.conf", config1)
	require.NoError(t, err)

	// Check BIND 9 daemon detection.
	monitor := newMonitor("", "", HTTPClientConfig{})
	monitor.commander = newTestCommandExecutor().
		addCheckConfOutput(config1Path, config1).
		addFileInfo(config1Path, &testFileInfo{})

	// Now run the detection as usual.
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	process := NewMockSupportedProcess(ctrl)
	absolutePath := path.Join(sandbox.BasePath, "named")
	process.EXPECT().getCmdline().Return(fmt.Sprintf("%s -c %s", absolutePath, config1Path), nil)
	process.EXPECT().getCwd().Return("", nil)

	detectedFiles, err := monitor.detectBind9ConfigPaths(process)
	require.NoError(t, err)
	require.NotNil(t, detectedFiles)
	require.Len(t, detectedFiles.files, 1)

	require.Equal(t, config1Path, detectedFiles.getFirstFilePathByType(detectedFileTypeConfig))
	require.Empty(t, detectedFiles.getFirstFilePathByType(detectedFileTypeRndcKey))
	require.Empty(t, detectedFiles.chrootDir)
	expectedBaseDir, _ := filepath.Split(sandbox.BasePath)
	require.Equal(t, filepath.Clean(expectedBaseDir), detectedFiles.baseDir)
}

// Checks detection with chroot STEP 1: if BIND9 detection takes -c parameter
// into consideration.
func TestDetectBind9ChrootStep1ProcessCmdLine(t *testing.T) {
	// Create alternate config files for each step.
	sandbox := testutil.NewSandbox()
	defer sandbox.Close()
	config1Path := path.Join(sandbox.BasePath, "step1.conf")
	chrootPath := path.Join(sandbox.BasePath, "chroot")
	config1 := `key "foo" { algorithm "hmac-sha256"; secret "abcd";};
                controls { inet 1.1.1.1 port 1111 allow { localhost; } keys { "foo"; "bar"; }; };`
	_, err := sandbox.Write(path.Join("chroot", config1Path), config1)
	require.NoError(t, err)

	// Check BIND 9 daemon detection.
	monitor := newMonitor("", "", HTTPClientConfig{})
	monitor.commander = newTestCommandExecutor().
		addCheckConfOutput(path.Join(chrootPath, config1Path), config1).
		addFileInfo(path.Join(chrootPath, config1Path), &testFileInfo{})

	// Now run the detection as usual.
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	process := NewMockSupportedProcess(ctrl)
	absolutePath := path.Join(sandbox.BasePath, "named")
	process.EXPECT().getCmdline().Return(fmt.Sprintf("%s -t %s -c %s", absolutePath, chrootPath, config1Path), nil)
	process.EXPECT().getCwd().Return("", nil)
	detectedFiles, err := monitor.detectBind9ConfigPaths(process)
	require.NoError(t, err)
	require.NotNil(t, detectedFiles)
	require.Len(t, detectedFiles.files, 1)
	path := detectedFiles.getFirstFilePathByType(detectedFileTypeConfig)
	require.Equal(t, config1Path, path)
	rndcKeyPath := detectedFiles.getFirstFilePathByType(detectedFileTypeRndcKey)
	require.Empty(t, rndcKeyPath)
	require.Equal(t, chrootPath, detectedFiles.chrootDir)
	expectedBaseDir, _ := filepath.Split(sandbox.BasePath)
	require.Equal(t, filepath.Clean(expectedBaseDir), detectedFiles.baseDir)
}

// Checks detection STEP 2: if BIND9 detection takes the explicit config path
// into account.
func TestDetectBind9Step2ExplicitPath(t *testing.T) {
	// Create alternate config file...
	sandbox := testutil.NewSandbox()
	defer sandbox.Close()
	confPath := path.Join(sandbox.BasePath, "testing.conf")
	config := `key "foo" {
		algorithm "hmac-sha256";
		secret "abcd";
   };
   controls {
		inet 192.0.2.1 port 1234 allow { localhost; } keys { "foo"; "bar"; };
   };`
	_, err := sandbox.Write("testing.conf", config)
	require.NoError(t, err)

	// Check BIND 9 daemon detection.
	monitor := newMonitor(confPath, "", HTTPClientConfig{})
	monitor.commander = newTestCommandExecutor().
		addCheckConfOutput(confPath, config).
		addFileInfo(confPath, &testFileInfo{})

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	process := NewMockSupportedProcess(ctrl)
	absolutePath := path.Join(sandbox.BasePath, "usr", "sbin", "named")
	process.EXPECT().getCmdline().Return(fmt.Sprintf("%s -some -params", absolutePath), nil)
	process.EXPECT().getCwd().Return("", nil)

	detectedFiles, err := monitor.detectBind9ConfigPaths(process)
	require.NoError(t, err)
	require.NotNil(t, detectedFiles)
	require.Len(t, detectedFiles.files, 1)
	path := detectedFiles.getFirstFilePathByType(detectedFileTypeConfig)
	require.Equal(t, confPath, path)
	rndcKeyPath := detectedFiles.getFirstFilePathByType(detectedFileTypeRndcKey)
	require.Empty(t, rndcKeyPath)
	require.Empty(t, detectedFiles.chrootDir)
	require.Equal(t, sandbox.BasePath+"/usr", detectedFiles.baseDir)
}

// Checks detection with chroot STEP 2: if BIND9 detection takes
// the explicit config path into account.
func TestDetectBind9ChrootStep2ExplicitPath(t *testing.T) {
	// Create alternate config file...
	sandbox := testutil.NewSandbox()
	defer sandbox.Close()
	confPath := path.Join(sandbox.BasePath, "testing.conf")
	chrootPath := path.Join(sandbox.BasePath, "chroot")
	fullConfPath := path.Join(chrootPath, confPath)
	config := `key "foo" {
		algorithm "hmac-sha256";
		secret "abcd";
	};
	controls {
		inet 192.0.2.1 port 1234 allow { localhost; } keys { "foo"; "bar"; };
	};`
	_, err := sandbox.Write(path.Join("chroot", confPath), config)
	require.NoError(t, err)

	// Check BIND 9 daemon detection.
	monitor := newMonitor(fullConfPath, "", HTTPClientConfig{})
	monitor.commander = newTestCommandExecutor().
		addCheckConfOutput(fullConfPath, config).
		addFileInfo(fullConfPath, &testFileInfo{})

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	process := NewMockSupportedProcess(ctrl)
	absolutePath := path.Join(sandbox.BasePath, "named")
	process.EXPECT().getCmdline().Return(fmt.Sprintf("%s -t %s -some -params", absolutePath, chrootPath), nil)
	process.EXPECT().getCwd().Return("", nil)

	detectedFiles, err := monitor.detectBind9ConfigPaths(process)
	require.NoError(t, err)
	require.NotNil(t, detectedFiles)
	require.Len(t, detectedFiles.files, 1)
	path := detectedFiles.getFirstFilePathByType(detectedFileTypeConfig)
	require.Equal(t, confPath, path)
	rndcKeyPath := detectedFiles.getFirstFilePathByType(detectedFileTypeRndcKey)
	require.Empty(t, rndcKeyPath)
	require.Equal(t, chrootPath, detectedFiles.chrootDir)
	expectedBaseDir, _ := filepath.Split(sandbox.BasePath)
	require.Equal(t, filepath.Clean(expectedBaseDir), detectedFiles.baseDir)
}

// Checks detection with chroot STEP 2: the explicit config path must be
// prefixed with the chroot directory.
func TestDetectBind9ChrootStep2ExplicitPathNotPrefixed(t *testing.T) {
	// Create alternate config file...
	sandbox := testutil.NewSandbox()
	defer sandbox.Close()
	confPath := path.Join(sandbox.BasePath, "testing.conf")
	chrootPath := path.Join(sandbox.BasePath, "chroot")
	fullConfPath := path.Join(chrootPath, confPath)
	config := `key "foo" {
		algorithm "hmac-sha256";
		secret "abcd";
	};
	controls {
		inet 192.0.2.1 port 1234 allow { localhost; } keys { "foo"; "bar"; };
	};`
	_, err := sandbox.Write(path.Join("chroot", confPath), config)
	require.NoError(t, err)

	// Check BIND 9 daemon detection.
	monitor := newMonitor("", "", HTTPClientConfig{})
	monitor.commander = newTestCommandExecutor().
		addCheckConfOutput(fullConfPath, config).
		addFileInfo(fullConfPath, &testFileInfo{})

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	process := NewMockSupportedProcess(ctrl)
	absolutePath := path.Join(sandbox.BasePath, "named")
	process.EXPECT().getCmdline().Return(fmt.Sprintf("%s -t %s -some -params", absolutePath, chrootPath), nil)
	process.EXPECT().getCwd().Return("", nil)
	detectedFiles, err := monitor.detectBind9ConfigPaths(process)
	require.ErrorContains(t, err, "BIND 9 config file not found")
	require.Nil(t, detectedFiles)
}

// Checks detection STEP 3: parse output of the named -V command.
func TestDetectBind9Step3BindVOutput(t *testing.T) {
	// Create alternate config file...
	sandbox := testutil.NewSandbox()
	defer sandbox.Close()
	varPath := path.Join(sandbox.BasePath, "testing.conf")
	config := `key "foo" {
		algorithm "hmac-sha256";
		secret "abcd";
    };
	controls {
		inet 192.0.2.1 port 1234 allow { localhost; } keys { "foo"; "bar"; };
    };`
	_, err := sandbox.Write("testing.conf", config)
	require.NoError(t, err)

	// ... and tell the fake executor to return it as the output of named -V.
	monitor := newMonitor("", "", HTTPClientConfig{})
	monitor.commander = newTestCommandExecutor().
		addCheckConfOutput(varPath, config).
		setConfigPathInNamedOutput(varPath).
		addFileInfo(varPath, &testFileInfo{})

	// Now run the detection as usual.
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	process := NewMockSupportedProcess(ctrl)
	absolutePath := path.Join(sandbox.BasePath, "named")
	process.EXPECT().getCmdline().Return(fmt.Sprintf("%s -some -params", absolutePath), nil)
	process.EXPECT().getCwd().Return("", nil)
	detectedFiles, err := monitor.detectBind9ConfigPaths(process)
	require.NoError(t, err)
	require.NotNil(t, detectedFiles)
	require.Len(t, detectedFiles.files, 1)
	path := detectedFiles.getFirstFilePathByType(detectedFileTypeConfig)
	require.Equal(t, varPath, path)
	rndcKeyPath := detectedFiles.getFirstFilePathByType(detectedFileTypeRndcKey)
	require.Empty(t, rndcKeyPath)
	require.Empty(t, detectedFiles.chrootDir)
	expectedBaseDir, _ := filepath.Split(sandbox.BasePath)
	require.Equal(t, filepath.Clean(expectedBaseDir), detectedFiles.baseDir)
}

// Checks detection with chroot STEP 3: parse output of the named -V command.
func TestDetectBind9ChrootStep3BindVOutput(t *testing.T) {
	// Create alternate config file...
	sandbox := testutil.NewSandbox()
	defer sandbox.Close()
	varPath := path.Join(sandbox.BasePath, "testing.conf")
	chrootPath := path.Join(sandbox.BasePath, "chroot")
	config := `key "foo" {
		algorithm "hmac-sha256";
		secret "abcd";
    };
	controls {
		inet 192.0.2.1 port 1234 allow { localhost; } keys { "foo"; "bar"; };
    };`
	_, err := sandbox.Write(path.Join("chroot", varPath), config)
	require.NoError(t, err)

	// ... and tell the fake executor to return it as the output of named -V.
	monitor := newMonitor("", "", HTTPClientConfig{})
	monitor.commander = newTestCommandExecutor().
		addCheckConfOutput(path.Join(chrootPath, varPath), config).
		// The named -V returns the path relative to the chroot directory.
		setConfigPathInNamedOutput(varPath).
		addFileInfo(path.Join(chrootPath, varPath), &testFileInfo{})

	// Now run the detection as usual.
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	process := NewMockSupportedProcess(ctrl)
	absolutePath := path.Join(sandbox.BasePath, "named")
	process.EXPECT().getCmdline().Return(fmt.Sprintf("%s -t %s -some -params", absolutePath, chrootPath), nil)
	process.EXPECT().getCwd().Return("", nil)

	detectedFiles, err := monitor.detectBind9ConfigPaths(process)
	require.NoError(t, err)
	require.NotNil(t, detectedFiles)
	require.Len(t, detectedFiles.files, 1)
	require.Equal(t, varPath, detectedFiles.getFirstFilePathByType(detectedFileTypeConfig))
	require.Empty(t, detectedFiles.getFirstFilePathByType(detectedFileTypeRndcKey))
	require.Equal(t, chrootPath, detectedFiles.chrootDir)
	expectedBaseDir, _ := filepath.Split(sandbox.BasePath)
	require.Equal(t, filepath.Clean(expectedBaseDir), detectedFiles.baseDir)
}

// Checks detection STEP 4: look at the typical locations.
func TestDetectBind9Step4TypicalLocations(t *testing.T) {
	// Arrange
	sandbox := testutil.NewSandbox()
	defer sandbox.Close()
	config := `key "foo" {
		algorithm "hmac-sha256";
		secret "abcd";
    };
	controls {
		inet 192.0.2.1 port 1234 allow { localhost; } keys { "foo"; "bar"; };
    };`
	_, err := sandbox.Write("testing.conf", config)
	require.NoError(t, err)

	for _, expectedPath := range getPotentialNamedConfLocations() {
		// getPotentialNamedConfLocations now returns dirs, need to append
		// filename.
		expectedConfigPath := path.Join(expectedPath, "named.conf")

		monitor := newMonitor("", "", HTTPClientConfig{})
		monitor.commander = newTestCommandExecutor().
			clear().
			addCheckConfOutput(expectedConfigPath, config).
			setConfigPathInNamedOutput(expectedConfigPath).
			addFileInfo(expectedConfigPath, &testFileInfo{})

		t.Run(expectedConfigPath, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			// Act
			process := NewMockSupportedProcess(ctrl)
			absolutePath := path.Join(sandbox.BasePath, "named")
			process.EXPECT().getCmdline().Return(fmt.Sprintf("%s -some -params", absolutePath), nil)
			process.EXPECT().getCwd().Return("", nil)
			detectedFiles, err := monitor.detectBind9ConfigPaths(process)

			// Assert
			require.NoError(t, err)
			require.NotNil(t, detectedFiles)
			require.Len(t, detectedFiles.files, 1)
			path := detectedFiles.getFirstFilePathByType(detectedFileTypeConfig)
			require.Equal(t, expectedConfigPath, path)
			rndcKeyPath := detectedFiles.getFirstFilePathByType(detectedFileTypeRndcKey)
			require.Empty(t, rndcKeyPath)
			require.Empty(t, detectedFiles.chrootDir)
			expectedBaseDir, _ := filepath.Split(sandbox.BasePath)
			require.Equal(t, filepath.Clean(expectedBaseDir), detectedFiles.baseDir)
		})
	}
}

// Checks detection with chroot STEP 4: look at the typical locations.
func TestDetectBind9ChrootStep4TypicalLocations(t *testing.T) {
	// Arrange
	sandbox := testutil.NewSandbox()
	defer sandbox.Close()
	config := `key "foo" {
		algorithm "hmac-sha256";
		secret "abcd";
    };
	controls {
		inet 192.0.2.1 port 1234 allow { localhost; } keys { "foo"; "bar"; };
    };`
	_, err := sandbox.Write("chroot/testing.conf", config)
	require.NoError(t, err)

	chrootPath := path.Join(sandbox.BasePath, "chroot")
	for _, expectedPath := range getPotentialNamedConfLocations() {
		expectedConfigPath := path.Join(expectedPath, "named.conf")
		monitor := newMonitor("", "", HTTPClientConfig{})
		monitor.commander = newTestCommandExecutor().
			clear().
			addCheckConfOutput(path.Join(chrootPath, expectedConfigPath), config).
			setConfigPathInNamedOutput(expectedConfigPath).
			addFileInfo(path.Join(chrootPath, expectedConfigPath), &testFileInfo{})

		t.Run(expectedPath, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			// Act
			process := NewMockSupportedProcess(ctrl)
			absolutePath := path.Join(sandbox.BasePath, "named")
			process.EXPECT().getCmdline().Return(fmt.Sprintf("%s -t %s -some -params", absolutePath, chrootPath), nil)
			process.EXPECT().getCwd().Return("", nil)
			detectedFiles, err := monitor.detectBind9ConfigPaths(process)

			// Assert
			require.NoError(t, err)
			require.NotNil(t, detectedFiles)
			require.Len(t, detectedFiles.files, 1)
			path := detectedFiles.getFirstFilePathByType(detectedFileTypeConfig)
			require.Equal(t, expectedConfigPath, path)
			rndcKeyPath := detectedFiles.getFirstFilePathByType(detectedFileTypeRndcKey)
			require.Empty(t, rndcKeyPath)
			require.Equal(t, chrootPath, detectedFiles.chrootDir)
			expectedBaseDir, _ := filepath.Split(sandbox.BasePath)
			require.Equal(t, filepath.Clean(expectedBaseDir), detectedFiles.baseDir)
		})
	}
}

// Check that an error is returned when detecting the BIND 9 configuration files
// but an attempt to get the file information fails.
func TestDetectBind9DaemonGetFileInfoError(t *testing.T) {
	sandbox := testutil.NewSandbox()
	defer sandbox.Close()
	config := `
		controls {
			inet 1.1.1.1 port 1111 allow { localhost; } keys { "foo"; "bar"; };
		};
	`
	_, err := sandbox.Write("named.conf", config)
	require.NoError(t, err)

	configPath := path.Join(sandbox.BasePath, "named.conf")
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	process := NewMockSupportedProcess(ctrl)
	absolutePath := path.Join(sandbox.BasePath, "named")
	process.EXPECT().getCmdline().Return(fmt.Sprintf("%s -c %s", absolutePath, configPath), nil)
	process.EXPECT().getCwd().Return("", nil)
	process.EXPECT().getPid().Times(0)

	// Create the command executor without adding an expectation for
	// GetFileInfo call. It should return an error when this call is
	// made. We want to make sure that this error is propagated to the caller.
	monitor := newMonitor("", "", HTTPClientConfig{})
	monitor.commander = newTestCommandExecutor().
		addCheckConfOutput(configPath, config)

	detectedFiles, err := monitor.detectBind9ConfigPaths(process)
	require.ErrorContains(t, err, "file not found")
	require.Nil(t, detectedFiles)
}

// Check that the detected BIND 9 daemon is instantiated and configured
// when both config and rndc key files are present.
func TestConfigureBind9DaemonBothConfigRndcKey(t *testing.T) {
	sandbox := testutil.NewSandbox()
	defer sandbox.Close()

	// Create config file.
	configPath := path.Join(sandbox.BasePath, "named.conf")
	config := `
		controls {
			inet 1.1.1.1 port 1111 allow { localhost; } keys { "foo"; "bar"; };
		};
		statistics-channels {
			inet 1.1.1.1 port 1112 allow { localhost; } keys { "foo"; "bar"; };
		};
	`
	_, err := sandbox.Write("named.conf", config)
	require.NoError(t, err)

	// Create rndc.key file.
	rndcKeyPath := path.Join(sandbox.BasePath, "rndc.key")
	rndcKeyConfig := `
		key "foo" {
			algorithm "hmac-sha256";
			secret "abcd";
		};
	`
	_, err = sandbox.Write("rndc.key", rndcKeyConfig)
	require.NoError(t, err)

	// Check BIND 9 daemon detection.
	monitor := newMonitor("", "", HTTPClientConfig{})
	monitor.commander = newTestCommandExecutor().
		addCheckConfOutput(configPath, config).
		addCheckConfOutput(rndcKeyPath, rndcKeyConfig).
		addFileInfo(configPath, &testFileInfo{}).
		addFileInfo(rndcKeyPath, &testFileInfo{})

	// Now run the detection as usual.
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	process := NewMockSupportedProcess(ctrl)
	process.EXPECT().getPid().Return(int32(1234))

	files := newDetectedDaemonFiles("", sandbox.BasePath)
	err = files.addFile(detectedFileTypeConfig, configPath, monitor.commander)
	require.NoError(t, err)

	daemon, err := monitor.configureBind9Daemon(process, files)
	require.NoError(t, err)
	require.NotNil(t, daemon)
	require.Equal(t, daemonname.Bind9, daemon.GetName())
	require.Len(t, daemon.GetAccessPoints(), 2)

	// Check control access point.
	controlPoint := daemon.GetAccessPoints()[0]
	require.Equal(t, AccessPointControl, controlPoint.Type)
	require.Equal(t, "1.1.1.1", controlPoint.Address)
	require.EqualValues(t, 1111, controlPoint.Port)

	// Check statistics access point.
	statisticsPoint := daemon.GetAccessPoints()[1]
	require.Equal(t, AccessPointStatistics, statisticsPoint.Type)
	require.Equal(t, "1.1.1.1", statisticsPoint.Address)
	require.EqualValues(t, 1112, statisticsPoint.Port)
	require.Empty(t, statisticsPoint.Key)
}

// Check that the detected BIND 9 daemon is instantiated and configured
// when rndc key file is absent.
func TestConfigureBind9DaemonConfigOnly(t *testing.T) {
	sandbox := testutil.NewSandbox()
	defer sandbox.Close()

	// Create config file.
	configPath := path.Join(sandbox.BasePath, "named.conf")
	config := `
		key "foo" {
			algorithm "hmac-sha256";
			secret "abcd";
		};
		controls {
			inet 1.1.1.1 port 1111 allow { localhost; } keys { "foo"; "bar"; };
		};
		statistics-channels {
			inet 1.1.1.1 port 1112 allow { localhost; } keys { "foo"; "bar"; };
		};
	`
	_, err := sandbox.Write("named.conf", config)
	require.NoError(t, err)

	// Check BIND 9 daemon detection.
	monitor := newMonitor("", "", HTTPClientConfig{})
	monitor.commander = newTestCommandExecutor().
		addCheckConfOutput(configPath, config).
		addFileInfo(configPath, &testFileInfo{})

	// Now run the detection as usual.
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	process := NewMockSupportedProcess(ctrl)
	process.EXPECT().getPid().Return(int32(1234))

	files := newDetectedDaemonFiles("", sandbox.BasePath)
	err = files.addFile(detectedFileTypeConfig, configPath, monitor.commander)
	require.NoError(t, err)

	daemon, err := monitor.configureBind9Daemon(process, files)
	require.NoError(t, err)
	require.NotNil(t, daemon)
	require.Equal(t, daemonname.Bind9, daemon.GetName())
	require.Len(t, daemon.GetAccessPoints(), 2)

	// Check control access point.
	controlPoint := daemon.GetAccessPoints()[0]
	require.Equal(t, AccessPointControl, controlPoint.Type)
	require.Equal(t, "1.1.1.1", controlPoint.Address)
	require.EqualValues(t, 1111, controlPoint.Port)

	// Check statistics access point.
	statisticsPoint := daemon.GetAccessPoints()[1]
	require.Equal(t, AccessPointStatistics, statisticsPoint.Type)
	require.Equal(t, "1.1.1.1", statisticsPoint.Address)
	require.EqualValues(t, 1112, statisticsPoint.Port)
	require.Empty(t, statisticsPoint.Key)
}

// Test that the included BIND 9 configuration files are recorded in the set
// of detected files.
func TestConfigureBind9DaemonIncludedFiles(t *testing.T) {
	sandbox := testutil.NewSandbox()
	defer sandbox.Close()

	// Create config file with two included files.
	configPath := path.Join(sandbox.BasePath, "named.conf")
	config := `
		include "include1.conf";
		include "include2.conf";

		key "foo" {
			algorithm "hmac-sha256";
			secret "abcd";
		};
		controls {
			inet 1.1.1.1 port 1111 allow { localhost; } keys { "foo"; "bar"; };
		};
		statistics-channels {
			inet 1.1.1.1 port 1112 allow { localhost; } keys { "foo"; "bar"; };
		};
	`
	_, err := sandbox.Write("named.conf", config)
	require.NoError(t, err)

	// First included file.
	include1Path := path.Join(sandbox.BasePath, "include1.conf")
	include1Config := `
		acl first { 1.1.1.1; };
	`
	_, err = sandbox.Write("include1.conf", include1Config)
	require.NoError(t, err)

	// Second included file.
	include2Path := path.Join(sandbox.BasePath, "include2.conf")
	include2Config := `
		acl second { 2.2.2.2; };
	`
	_, err = sandbox.Write("include2.conf", include2Config)
	require.NoError(t, err)

	// Create the monitor with mock command executor.
	monitor := newMonitor("", "", HTTPClientConfig{})
	monitor.commander = newTestCommandExecutor().
		addCheckConfOutput(configPath, config).
		addCheckConfOutput(include1Path, include1Config).
		addCheckConfOutput(include2Path, include2Config).
		addFileInfo(configPath, &testFileInfo{}).
		addFileInfo(include1Path, &testFileInfo{}).
		addFileInfo(include2Path, &testFileInfo{})

	// Create the mock process.
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	process := NewMockSupportedProcess(ctrl)
	process.EXPECT().getPid().Return(int32(1234))

	// Create the set of detected files with one config file
	// and without the includes.
	files := newDetectedDaemonFiles("", sandbox.BasePath)
	err = files.addFile(detectedFileTypeConfig, configPath, monitor.commander)
	require.NoError(t, err)

	// Parse and expand the configuration files.
	daemon, err := monitor.configureBind9Daemon(process, files)
	require.NoError(t, err)
	require.NotNil(t, daemon)
	require.Equal(t, daemonname.Bind9, daemon.GetName())
	require.Len(t, daemon.GetAccessPoints(), 2)

	// We should now have three files recorded. One main config file
	// and two included files.
	require.NotNil(t, daemon.detectedFiles)
	require.Len(t, daemon.detectedFiles.files, 3)

	// Make sure their paths are correct.
	require.Equal(t, configPath, daemon.detectedFiles.files[0].path)
	require.Equal(t, include1Path, daemon.detectedFiles.files[1].path)
	require.Equal(t, include2Path, daemon.detectedFiles.files[2].path)
}

// Check that the detected BIND 9 daemon is instantiated and configured
// when the statistics channels are not configured.
func TestConfigureBind9DaemonNoStatistics(t *testing.T) {
	sandbox := testutil.NewSandbox()
	defer sandbox.Close()

	// Create config file.
	configPath := path.Join(sandbox.BasePath, "named.conf")
	config := `
		key "foo" {
			algorithm "hmac-sha256";
			secret "abcd";
		};
		controls {
			inet 1.1.1.1 port 1111 allow { localhost; } keys { "foo"; "bar"; };
		};
	`
	_, err := sandbox.Write("named.conf", config)
	require.NoError(t, err)

	// Check BIND 9 daemon detection.
	monitor := newMonitor("", "", HTTPClientConfig{})
	monitor.commander = newTestCommandExecutor().
		addCheckConfOutput(configPath, config).
		addFileInfo(configPath, &testFileInfo{})

	// Now run the detection as usual.
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	process := NewMockSupportedProcess(ctrl)
	process.EXPECT().getPid().Return(int32(1234))

	files := newDetectedDaemonFiles("", sandbox.BasePath)
	err = files.addFile(detectedFileTypeConfig, configPath, monitor.commander)
	require.NoError(t, err)

	daemon, err := monitor.configureBind9Daemon(process, files)
	require.NoError(t, err)
	require.NotNil(t, daemon)
	require.Equal(t, daemonname.Bind9, daemon.GetName())
	require.Len(t, daemon.GetAccessPoints(), 1)

	// Check control access point.
	controlPoint := daemon.GetAccessPoints()[0]
	require.Equal(t, AccessPointControl, controlPoint.Type)
	require.Equal(t, "1.1.1.1", controlPoint.Address)
	require.EqualValues(t, 1111, controlPoint.Port)
	require.EqualValues(t, "foo:hmac-sha256:abcd", controlPoint.Key)

	// Check other daemon fields.
	require.Equal(t, daemon.Name, daemonname.Bind9)
	require.Nil(t, daemon.zoneInventory)
	require.NotNil(t, daemon.rndcClient)
	require.EqualValues(t, 1234, daemon.pid)
	require.NotNil(t, daemon.bind9Config)
	require.Nil(t, daemon.rndcKeyConfig)
}

// Check that an error is returned when parsing the BIND 9 config file fails.
func TestConfigureBind9DaemonParseError(t *testing.T) {
	// Now run the detection as usual.
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	process := NewMockSupportedProcess(ctrl)

	monitor := newMonitor("", "", HTTPClientConfig{})
	monitor.commander = newTestCommandExecutor().
		addFileInfo("/chroot/etc/bind/named.conf", &testFileInfo{})

	parser := NewMockBind9FileParser(ctrl)
	parser.EXPECT().ParseFile("/etc/bind/named.conf", "/chroot").Return(nil, errors.New("test error"))

	monitor.bind9FileParser = parser

	files := newDetectedDaemonFiles("/chroot", "")
	err := files.addFile(detectedFileTypeConfig, "/etc/bind/named.conf", monitor.commander)
	require.NoError(t, err)

	daemon, err := monitor.configureBind9Daemon(process, files)
	require.Error(t, err)
	require.Nil(t, daemon)
	require.ErrorContains(t, err, "failed to parse BIND 9 config file")
}

// There is no reliable way to test step 4 (checking typical locations). The
// code is not mockable. We could check if there's BIND config in any of the
// typical locations, but what exactly are we supposed to do if we find one?
// The actual Ubuntu 22.04 system is a good example. I have BIND 9 installed
// and the detection actually detects the BIND 9 config file. However, it fails
// to read rndc.key file, because it's set to be read by bind user only.
// Without rndc the BIND detection fails and returns no daemons.
//
// An alternative approach would be to enable debug logging, then capture the
// stdout and parse if the messages mention default locations. We _could_ do
// that, but it's not worth the effort, especially given the detection code
// is really a simple for loop.

// Checks detection order. Several steps are configured. It checks if the
// steps order is as expected. There are 3 configs on disk:
// - step1.conf (which is passed in named -c step1.conf)
// - step2.conf (which is passed in settings)
// - step3.conf (which is returned by named -V).
func TestDetectBind9DetectOrder(t *testing.T) {
	// Create alternate config files for each step...
	sandbox := testutil.NewSandbox()
	defer sandbox.Close()
	config1 := `key "foo" { algorithm "hmac-sha256"; secret "abcd";};
                controls { inet 1.1.1.1 port 1111 allow { localhost; } keys { "foo"; "bar"; }; };`
	config1Path := path.Join(sandbox.BasePath, "step1.conf")
	_, err := sandbox.Write("step1.conf", config1)
	require.NoError(t, err)
	config2 := `key "foo" { algorithm "hmac-sha256"; secret "abcd";};
                controls { inet 2.2.2.2 port 2222 allow { localhost; } keys { "foo"; "bar"; }; };`
	config2Path := path.Join(sandbox.BasePath, "step2.conf")
	config3 := `key "foo" { algorithm "hmac-sha256"; secret "abcd";};
                controls { inet 3.3.3.3 port 3333 allow { localhost; } keys { "foo"; "bar"; }; };`
	config3Path := path.Join(sandbox.BasePath, "step3.conf")

	// ... and tell the fake executor to return it as the output of named -V
	monitor := newMonitor("", "", HTTPClientConfig{})
	monitor.commander = newTestCommandExecutor().
		addCheckConfOutput(config1Path, config1).
		addCheckConfOutput(config2Path, config2).
		addCheckConfOutput(config3Path, config3).
		setConfigPathInNamedOutput(config3Path).
		addFileInfo(config1Path, &testFileInfo{}).
		addFileInfo(config2Path, &testFileInfo{}).
		addFileInfo(config3Path, &testFileInfo{})

	// Now run the detection as usual
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	process := NewMockSupportedProcess(ctrl)
	absolutePath := path.Join(sandbox.BasePath, "named")
	process.EXPECT().getCmdline().Return(fmt.Sprintf("%s -c %s", absolutePath, config1Path), nil)
	process.EXPECT().getCwd().Return("", nil)
	process.EXPECT().getPid().Return(int32(1234))
	daemon, err := monitor.detectBind9Daemon(process)
	require.NoError(t, err)
	require.NotNil(t, daemon)
	require.Equal(t, daemonname.Bind9, daemon.GetName())
	require.Len(t, daemon.GetAccessPoints(), 1)
	point := daemon.GetAccessPoints()[0]
	require.Equal(t, AccessPointControl, point.Type)
	require.Equal(t, "1.1.1.1", point.Address) // we expect the STEP 1 (-c parameter) to take precedence
	require.EqualValues(t, 1111, point.Port)
	require.EqualValues(t, "foo:hmac-sha256:abcd", point.Key)
}

// Test that the RNDC parameters are determined correctly if the RNDC key name
// is not set and the default RNDC key exists.
func TestDetermineDetailsUseDefaultKey(t *testing.T) {
	// Arrange
	executor := newTestCommandExecutor()
	executor.addCheckConfOutput("/conf_dir/rndc.key", "")
	client := NewRndcClient(executor)
	expectedBaseCommand := []string{
		"/usr/sbin/rndc",
		"-s", "address",
		"-p", "42",
		"-k", "/conf_dir/rndc.key",
	}

	// Act
	err := client.DetermineDetails("/exe_dir", "/conf_dir", "address", 42, nil)

	// Assert
	require.NoError(t, err)
	require.Equal(t, expectedBaseCommand, client.BaseCommand)
}

// Test that the determination of the RNDC parameters throws an error if the
// RNDC key name is not set but the default RNDC key does not exist.
func TestDetermineDetailsUseMissingDefaultKey(t *testing.T) {
	// Arrange
	executor := newTestCommandExecutor()
	client := NewRndcClient(executor)

	// Act
	err := client.DetermineDetails("/exe_dir", "/conf_dir", "address", 42, nil)

	// Assert
	require.Error(t, err)
}

// Test that the RNDC parameters are determined correctly if the RNDC key name
// is set and the default RNDC config exists.
func TestDetermineDetailsCustomKeyExistingConfig(t *testing.T) {
	// Arrange
	executor := newTestCommandExecutor()
	executor.addCheckConfOutput("/conf_dir/rndc.conf", "")
	client := NewRndcClient(executor)
	expectedBaseCommand := []string{
		"/usr/sbin/rndc",
		"-s", "address",
		"-p", "42",
		"-y", "name",
		"-c", "/conf_dir/rndc.conf",
	}
	key := &bind9config.Key{
		Name: "name",
		Clauses: []*bind9config.KeyClause{
			{
				Algorithm: "alg",
				Secret:    "sec",
			},
		},
	}

	// Act
	err := client.DetermineDetails("/exe_dir", "/conf_dir", "address", 42, key)

	// Assert
	require.NoError(t, err)
	require.Equal(t, expectedBaseCommand, client.BaseCommand)
}

// Test that the RNDC parameters are determined correctly if the RNDC key name
// is set, the default RNDC config does not exists but the default RNDC key
// file exists.
func TestDetermineDetailsCustomKeyMissingConfigExistingKey(t *testing.T) {
	// Arrange
	executor := newTestCommandExecutor()
	executor.addCheckConfOutput("/conf_dir/rndc.key", "")
	client := NewRndcClient(executor)
	expectedBaseCommand := []string{
		"/usr/sbin/rndc",
		"-s", "address",
		"-p", "42",
		"-y", "name",
		// Use the -c flag instead of -k.
		"-c", "/conf_dir/rndc.key",
	}
	key := &bind9config.Key{
		Name: "name",
		Clauses: []*bind9config.KeyClause{
			{
				Algorithm: "alg",
				Secret:    "sec",
			},
		},
	}

	// Act
	err := client.DetermineDetails("/exe_dir", "/conf_dir", "address", 42, key)

	// Assert
	require.NoError(t, err)
	require.Equal(t, expectedBaseCommand, client.BaseCommand)
}

// Test that the RNDC parameters are determined correctly if the RNDC key name
// is set, the default RNDC config does not exists, and the default RNDC key
// does not exist too.
func TestDetermineDetailsCustomKeyMissingConfigMissingKey(t *testing.T) {
	// Arrange
	executor := newTestCommandExecutor()
	client := NewRndcClient(executor)
	expectedBaseCommand := []string{
		"/usr/sbin/rndc",
		"-s", "address",
		"-p", "42",
		"-y", "name",
	}
	key := &bind9config.Key{
		Name: "name",
		Clauses: []*bind9config.KeyClause{
			{
				Algorithm: "alg",
				Secret:    "sec",
			},
		},
	}

	// Act
	err := client.DetermineDetails("/exe_dir", "/conf_dir", "address", 42, key)

	// Assert
	require.NoError(t, err)
	require.Equal(t, expectedBaseCommand, client.BaseCommand)
}

// Test that stopping zone inventory doesn't panic when zone inventory is nil.
func TestBind9DaemonStopZoneInventoryNil(t *testing.T) {
	daemon := &Bind9Daemon{}
	require.NotPanics(t, func() {
		err := daemon.Cleanup()
		require.NoError(t, err)
	})
}
