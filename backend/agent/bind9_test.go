package agent

import (
	"fmt"
	"path"
	"strings"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	gomock "go.uber.org/mock/gomock"
	bind9config "isc.org/stork/appcfg/bind9"
	"isc.org/stork/testutil"
	storkutil "isc.org/stork/util"
)

//go:generate mockgen -package=agent -destination=bind9mock_test.go -mock_names=bind9FileParser=MockBind9FileParser isc.org/stork/agent bind9FileParser

// Test the function which extracts the list of log files from the Bind9
// application by sending the request to the Kea Control Agent and the
// daemons behind it.
func TestBind9AllowedLogs(t *testing.T) {
	ba := &Bind9App{}
	paths, err := ba.DetectAllowedLogs()
	require.NoError(t, err)
	require.Len(t, paths, 0)
}

// Test that the zone inventory can be accessed.
func TestBind9GetZoneInventory(t *testing.T) {
	app := &Bind9App{
		zoneInventory: &zoneInventory{},
	}
	inventory := app.GetZoneInventory()
	require.Equal(t, app.zoneInventory, inventory)
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

// The command executor implementation for the testing purposes.
// It implements the builder pattern for the configuration methods.
type testCommandExecutor struct {
	configPathInNamedOutput *string
	checkConfOutputs        map[string]string
	rndcStatusError         error
	rndcStatus              string
}

// Constructs a new instance of the test command executor.
func newTestCommandExecutor() *testCommandExecutor {
	return &testCommandExecutor{
		checkConfOutputs: map[string]string{},
		rndcStatus:       "Server is up and running",
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

	return nil, nil
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

	// Check BIND 9 app detection.
	executor := newTestCommandExecutor().
		addCheckConfOutput(config1Path, config1)

	// Now run the detection as usual.
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	process := NewMockSupportedProcess(ctrl)
	absolutePath := path.Join(sandbox.BasePath, "named")
	process.EXPECT().getCmdline().Return(fmt.Sprintf("%s -c %s", absolutePath, config1Path), nil)
	process.EXPECT().getCwd().Return("", nil)
	app, err := detectBind9App(process, executor, "", bind9config.NewParser())
	require.NoError(t, err)
	require.NotNil(t, app)
	require.Equal(t, app.GetBaseApp().Type, AppTypeBind9)
	require.Len(t, app.GetBaseApp().AccessPoints, 1)
	point := app.GetBaseApp().AccessPoints[0]
	require.Equal(t, AccessPointControl, point.Type)
	require.Equal(t, "1.1.1.1", point.Address)
	require.EqualValues(t, 1111, point.Port)
	require.EqualValues(t, "foo:hmac-sha256:abcd", point.Key)
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

	// Check BIND 9 app detection.
	executor := newTestCommandExecutor().
		addCheckConfOutput(path.Join(chrootPath, config1Path), config1)

	// Now run the detection as usual.
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	process := NewMockSupportedProcess(ctrl)
	absolutePath := path.Join(sandbox.BasePath, "named")
	process.EXPECT().getCmdline().Return(fmt.Sprintf("%s -t %s -c %s", absolutePath, chrootPath, config1Path), nil)
	process.EXPECT().getCwd().Return("", nil)
	app, err := detectBind9App(process, executor, "", bind9config.NewParser())
	require.NoError(t, err)
	require.NotNil(t, app)
	require.Equal(t, app.GetBaseApp().Type, AppTypeBind9)
	require.Len(t, app.GetBaseApp().AccessPoints, 1)
	point := app.GetBaseApp().AccessPoints[0]
	require.Equal(t, AccessPointControl, point.Type)
	require.Equal(t, "1.1.1.1", point.Address)
	require.EqualValues(t, 1111, point.Port)
	require.EqualValues(t, "foo:hmac-sha256:abcd", point.Key)
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

	// Check BIND 9 app detection.
	executor := newTestCommandExecutor().
		addCheckConfOutput(confPath, config)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	process := NewMockSupportedProcess(ctrl)
	absolutePath := path.Join(sandbox.BasePath, "usr", "sbin", "named")
	process.EXPECT().getCmdline().Return(fmt.Sprintf("%s -some -params", absolutePath), nil)
	process.EXPECT().getCwd().Return("", nil)
	app, err := detectBind9App(process, executor, confPath, bind9config.NewParser())
	require.NoError(t, err)
	require.NotNil(t, app)
	require.Equal(t, app.GetBaseApp().Type, AppTypeBind9)
	require.Len(t, app.GetBaseApp().AccessPoints, 1)
	point := app.GetBaseApp().AccessPoints[0]
	require.Equal(t, AccessPointControl, point.Type)
	require.Equal(t, "192.0.2.1", point.Address)
	require.EqualValues(t, 1234, point.Port)
	require.EqualValues(t, "foo:hmac-sha256:abcd", point.Key)
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

	// Check BIND 9 app detection.
	executor := newTestCommandExecutor().
		addCheckConfOutput(fullConfPath, config)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	process := NewMockSupportedProcess(ctrl)
	absolutePath := path.Join(sandbox.BasePath, "named")
	process.EXPECT().getCmdline().Return(fmt.Sprintf("%s -t %s -some -params", absolutePath, chrootPath), nil)
	process.EXPECT().getCwd().Return("", nil)
	app, err := detectBind9App(process, executor, fullConfPath, bind9config.NewParser())
	require.NoError(t, err)
	require.NotNil(t, app)
	require.Equal(t, app.GetBaseApp().Type, AppTypeBind9)
	require.Len(t, app.GetBaseApp().AccessPoints, 1)
	point := app.GetBaseApp().AccessPoints[0]
	require.Equal(t, AccessPointControl, point.Type)
	require.Equal(t, "192.0.2.1", point.Address)
	require.EqualValues(t, 1234, point.Port)
	require.EqualValues(t, "foo:hmac-sha256:abcd", point.Key)
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

	// Check BIND 9 app detection.
	executor := newTestCommandExecutor().
		addCheckConfOutput(fullConfPath, config)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	process := NewMockSupportedProcess(ctrl)
	absolutePath := path.Join(sandbox.BasePath, "named")
	process.EXPECT().getCmdline().Return(fmt.Sprintf("%s -t %s -some -params", absolutePath, chrootPath), nil)
	process.EXPECT().getCwd().Return("", nil)
	app, err := detectBind9App(process, executor, confPath, bind9config.NewParser())
	require.ErrorContains(t, err, "cannot find config file for BIND 9")
	require.Nil(t, app)
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
	executor := newTestCommandExecutor().
		addCheckConfOutput(varPath, config).
		setConfigPathInNamedOutput(varPath)

	// Now run the detection as usual.
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	process := NewMockSupportedProcess(ctrl)
	absolutePath := path.Join(sandbox.BasePath, "named")
	process.EXPECT().getCmdline().Return(fmt.Sprintf("%s -some -params", absolutePath), nil)
	process.EXPECT().getCwd().Return("", nil)
	app, err := detectBind9App(process, executor, "", bind9config.NewParser())
	require.NoError(t, err)
	require.NotNil(t, app)
	require.Equal(t, app.GetBaseApp().Type, AppTypeBind9)
	require.Len(t, app.GetBaseApp().AccessPoints, 1)
	point := app.GetBaseApp().AccessPoints[0]
	require.Equal(t, AccessPointControl, point.Type)
	require.Equal(t, "192.0.2.1", point.Address)
	require.EqualValues(t, 1234, point.Port)
	require.EqualValues(t, "foo:hmac-sha256:abcd", point.Key)
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
	executor := newTestCommandExecutor().
		addCheckConfOutput(path.Join(chrootPath, varPath), config).
		// The named -V returns the path relative to the chroot directory.
		setConfigPathInNamedOutput(varPath)

	// Now run the detection as usual.
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	process := NewMockSupportedProcess(ctrl)
	absolutePath := path.Join(sandbox.BasePath, "named")
	process.EXPECT().getCmdline().Return(fmt.Sprintf("%s -t %s -some -params", absolutePath, chrootPath), nil)
	process.EXPECT().getCwd().Return("", nil)
	app, err := detectBind9App(process, executor, "", bind9config.NewParser())
	require.NoError(t, err)
	require.NotNil(t, app)
	require.Equal(t, app.GetBaseApp().Type, AppTypeBind9)
	require.Len(t, app.GetBaseApp().AccessPoints, 1)
	point := app.GetBaseApp().AccessPoints[0]
	require.Equal(t, AccessPointControl, point.Type)
	require.Equal(t, "192.0.2.1", point.Address)
	require.EqualValues(t, 1234, point.Port)
	require.EqualValues(t, "foo:hmac-sha256:abcd", point.Key)
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

	executor := newTestCommandExecutor()

	for _, expectedPath := range getPotentialNamedConfLocations() {
		// getPotentialNamedConfLocations now returns dirs, need to append
		// filename.
		expectedConfigPath := path.Join(expectedPath, "named.conf")

		executor.
			clear().
			addCheckConfOutput(expectedConfigPath, config).
			setConfigPathInNamedOutput(expectedConfigPath)

		t.Run(expectedConfigPath, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			parser := NewMockBind9FileParser(ctrl)
			parser.EXPECT().ParseFile(expectedConfigPath).DoAndReturn(func(configPath string) (*bind9config.Config, error) {
				return bind9config.NewParser().ParseFile(path.Join(sandbox.BasePath, "testing.conf"))
			})

			// Act
			process := NewMockSupportedProcess(ctrl)
			absolutePath := path.Join(sandbox.BasePath, "named")
			process.EXPECT().getCmdline().Return(fmt.Sprintf("%s -some -params", absolutePath), nil)
			process.EXPECT().getCwd().Return("", nil)
			app, err := detectBind9App(process, executor, "", parser)

			// Assert
			require.NoError(t, err)
			require.NotNil(t, app)
			require.Equal(t, app.GetBaseApp().Type, AppTypeBind9)
			require.Len(t, app.GetBaseApp().AccessPoints, 1)
			point := app.GetBaseApp().AccessPoints[0]
			require.Equal(t, AccessPointControl, point.Type)
			require.Equal(t, "192.0.2.1", point.Address)
			require.EqualValues(t, 1234, point.Port)
			require.EqualValues(t, "foo:hmac-sha256:abcd", point.Key)
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
	_, err := sandbox.Write("testing.conf", config)
	require.NoError(t, err)

	executor := newTestCommandExecutor()
	chrootPath := path.Join(sandbox.BasePath, "chroot")
	for _, expectedPath := range getPotentialNamedConfLocations() {
		expectedConfigPath := path.Join(expectedPath, "named.conf")
		executor.
			clear().
			addCheckConfOutput(path.Join(chrootPath, expectedConfigPath), config).
			setConfigPathInNamedOutput(expectedConfigPath)

		t.Run(expectedPath, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			parser := NewMockBind9FileParser(ctrl)
			parser.EXPECT().ParseFile(path.Join(chrootPath, expectedConfigPath)).DoAndReturn(func(configPath string) (*bind9config.Config, error) {
				return bind9config.NewParser().ParseFile(path.Join(sandbox.BasePath, "testing.conf"))
			})

			// Act
			process := NewMockSupportedProcess(ctrl)
			absolutePath := path.Join(sandbox.BasePath, "named")
			process.EXPECT().getCmdline().Return(fmt.Sprintf("%s -t %s -some -params", absolutePath, chrootPath), nil)
			process.EXPECT().getCwd().Return("", nil)
			app, err := detectBind9App(process, executor, "", parser)

			// Assert
			require.NoError(t, err)
			require.NotNil(t, app)
			require.Equal(t, app.GetBaseApp().Type, AppTypeBind9)
			require.Len(t, app.GetBaseApp().AccessPoints, 1)
			point := app.GetBaseApp().AccessPoints[0]
			require.Equal(t, AccessPointControl, point.Type)
			require.Equal(t, "192.0.2.1", point.Address)
			require.EqualValues(t, 1234, point.Port)
			require.EqualValues(t, "foo:hmac-sha256:abcd", point.Key)
		})
	}
}

// There is no reliable way to test step 4 (checking typical locations). The
// code is not mockable. We could check if there's BIND config in any of the
// typical locations, but what exactly are we supposed to do if we find one?
// The actual Ubuntu 22.04 system is a good example. I have BIND 9 installed
// and the detection actually detects the BIND 9 config file. However, it fails
// to read rndc.key file, because it's set to be read by bind user only.
// Without rndc the BIND detection fails and returns no apps.
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
	executor := newTestCommandExecutor().
		addCheckConfOutput(config1Path, config1).
		addCheckConfOutput(config2Path, config2).
		addCheckConfOutput(config3Path, config3).
		setConfigPathInNamedOutput(config3Path)

	// Now run the detection as usual
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	process := NewMockSupportedProcess(ctrl)
	absolutePath := path.Join(sandbox.BasePath, "named")
	process.EXPECT().getCmdline().Return(fmt.Sprintf("%s -c %s", absolutePath, config1Path), nil)
	process.EXPECT().getCwd().Return("", nil)
	app, err := detectBind9App(process, executor, config2Path, bind9config.NewParser())
	require.NoError(t, err)
	require.NotNil(t, app)
	require.Equal(t, app.GetBaseApp().Type, AppTypeBind9)
	require.Len(t, app.GetBaseApp().AccessPoints, 1)
	point := app.GetBaseApp().AccessPoints[0]
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

// Test that awaiting background tasks doesn't panic when zone inventory is nil.
func TestBind9AppAwaitBackgroundTasksNilZoneInventory(t *testing.T) {
	app := &Bind9App{}
	require.NotPanics(t, app.StopZoneInventory)
}
