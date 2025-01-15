package agent

import (
	"fmt"
	"path"
	"strings"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	storkutil "isc.org/stork/util"
)

// Test the function which extracts the list of log files from the Bind9
// application by sending the request to the Kea Control Agent and the
// daemons behind it.
func TestBind9AllowedLogs(t *testing.T) {
	ba := &Bind9App{}
	paths, err := ba.DetectAllowedLogs()
	require.NoError(t, err)
	require.Len(t, paths, 0)
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

// Tests if getCtrlAddressFromBind9Config() can handle the right
// cases:
// - CASE 1: no controls block (use defaults)
// - CASE 2: empty controls block (returns nothing)
// - CASE 3: empty multi-line controls block with no options (returns nothing)
// - CASE 4: controls block with options (return the address).
func TestGetCtrlAddressFromBind9Config(t *testing.T) {
	// Define test cases
	type testCase struct {
		config  string
		expAddr string
		expPort int64
		expKey  string
	}

	testCases := map[string]testCase{
		"CASE 1: default config from Ubuntu 22.04": {config: `
		options {
			directory "/var/cache/bind";
			listen-on-v6  {
				"any";
			};
			dnssec-validation auto;
		};
		zone "." {
			type hint;
			file "/usr/share/dns/root.hints";
		};
		zone "localhost" {
			type master;
			file "/etc/bind/db.local";
		};
		zone "127.in-addr.arpa" {
			type master;
			file "/etc/bind/db.127";
		};`, expAddr: "127.0.0.1", expPort: 953, expKey: ""},
		"CASE 2: empty controls section (disabled rndc)": {config: "controls { };", expAddr: "", expPort: 0, expKey: ""},
		"CASE 3: empty multi-line controls section (disabled rndc)": {config: `controls
	{

};`, expAddr: "", expPort: 0, expKey: ""},
		"CASE 4: added controls section with options": {config: `
		controls {
			inet 192.0.2.1 allow { localhost; };
		};`, expAddr: "192.0.2.1", expPort: 953, expKey: ""},
	}

	for name, test := range testCases {
		t.Run(name, func(t *testing.T) {
			a, b, c := getCtrlAddressFromBind9Config(test.config)
			require.Equal(t, a, test.expAddr)
			require.Equal(t, b, test.expPort)
			require.Nil(t, c, test.expKey)
		})
	}
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
	config1Path := "/dir/step1.conf"
	config1 := `key "foo" { algorithm "hmac-sha256"; secret "abcd";};
                controls { inet 1.1.1.1 port 1111 allow { localhost; } keys { "foo"; "bar"; }; };`

	// Check BIND 9 app detection.
	executor := newTestCommandExecutor().
		addCheckConfOutput(config1Path, config1)

	// Now run the detection as usual.
	app := detectBind9App([]string{"", "/dir", fmt.Sprintf("-c %s", config1Path)}, "", executor, "")
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
	config1Path := "/dir/step1.conf"
	chrootPath := "/chroot"
	config1 := `key "foo" { algorithm "hmac-sha256"; secret "abcd";};
                controls { inet 1.1.1.1 port 1111 allow { localhost; } keys { "foo"; "bar"; }; };`

	// Check BIND 9 app detection.
	executor := newTestCommandExecutor().
		addCheckConfOutput(path.Join(chrootPath, config1Path), config1)

	// Now run the detection as usual.
	app := detectBind9App([]string{
		"",
		"/dir",
		fmt.Sprintf("-t %s -c %s", chrootPath, config1Path),
	}, "", executor, "")
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
	confPath := "/dir/testing.conf"
	config := `key "foo" {
		algorithm "hmac-sha256";
		secret "abcd";
   };
   controls {
		inet 192.0.2.1 port 1234 allow { localhost; } keys { "foo"; "bar"; };
   };`

	// Check BIND 9 app detection.
	executor := newTestCommandExecutor().
		addCheckConfOutput(confPath, config)

	namedDir := "/dir/usr/sbin"
	app := detectBind9App(
		[]string{"", namedDir, "-some -params"},
		"",
		executor,
		confPath,
	)
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
	confPath := "/dir/testing.conf"
	chrootPath := "/chroot"
	fullConfPath := path.Join(chrootPath, confPath)
	config := `key "foo" {
		algorithm "hmac-sha256";
		secret "abcd";
	};
	controls {
		inet 192.0.2.1 port 1234 allow { localhost; } keys { "foo"; "bar"; };
	};`

	// Check BIND 9 app detection.
	executor := newTestCommandExecutor().
		addCheckConfOutput(fullConfPath, config)

	namedDir := "/dir/usr/sbin"
	app := detectBind9App([]string{
		"",
		namedDir,
		fmt.Sprintf("-t %s -some -params", chrootPath),
	}, "", executor, fullConfPath)
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
	confPath := "/dir/testing.conf"
	chrootPath := "/chroot"
	fullConfPath := path.Join(chrootPath, confPath)
	config := `key "foo" {
		algorithm "hmac-sha256";
		secret "abcd";
	};
	controls {
		inet 192.0.2.1 port 1234 allow { localhost; } keys { "foo"; "bar"; };
	};`

	// Check BIND 9 app detection.
	executor := newTestCommandExecutor().
		addCheckConfOutput(fullConfPath, config)

	namedDir := "/dir/usr/sbin"
	app := detectBind9App([]string{
		"",
		namedDir,
		fmt.Sprintf("-t %s -some -params", chrootPath),
	}, "", executor, confPath)
	require.Nil(t, app)
}

// Checks detection STEP 3: parse output of the named -V command.
func TestDetectBind9Step3BindVOutput(t *testing.T) {
	// Create alternate config file...
	varPath := "/dir/testing.conf"
	config := `key "foo" {
		algorithm "hmac-sha256";
		secret "abcd";
    };
	controls {
		inet 192.0.2.1 port 1234 allow { localhost; } keys { "foo"; "bar"; };
    };`

	// ... and tell the fake executor to return it as the output of named -V.
	executor := newTestCommandExecutor().
		addCheckConfOutput(varPath, config).
		setConfigPathInNamedOutput(varPath)

	// Now run the detection as usual.
	namedDir := "/dir/usr/sbin"
	app := detectBind9App([]string{"", namedDir, "-some -params"}, "", executor, "")
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
	varPath := "/dir/testing.conf"
	chrootPath := "/chroot"
	config := `key "foo" {
		algorithm "hmac-sha256";
		secret "abcd";
    };
	controls {
		inet 192.0.2.1 port 1234 allow { localhost; } keys { "foo"; "bar"; };
    };`

	// ... and tell the fake executor to return it as the output of named -V.
	executor := newTestCommandExecutor().
		addCheckConfOutput(path.Join(chrootPath, varPath), config).
		// The named -V returns the path relative to the chroot directory.
		setConfigPathInNamedOutput(varPath)

	// Now run the detection as usual.
	namedDir := "/dir/usr/sbin"
	app := detectBind9App([]string{
		"",
		namedDir,
		fmt.Sprintf("-t %s -some -params", chrootPath),
	}, "", executor, "")
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
	config := `key "foo" {
		algorithm "hmac-sha256";
		secret "abcd";
    };
	controls {
		inet 192.0.2.1 port 1234 allow { localhost; } keys { "foo"; "bar"; };
    };`

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
			// Act
			app := detectBind9App([]string{"", "/dir", "-some -params"}, "", executor, "")

			// Assert
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

// Checks detection wit chroot STEP 4: look at the typical locations.
func TestDetectBind9ChrootStep4TypicalLocations(t *testing.T) {
	// Arrange
	config := `key "foo" {
		algorithm "hmac-sha256";
		secret "abcd";
    };
	controls {
		inet 192.0.2.1 port 1234 allow { localhost; } keys { "foo"; "bar"; };
    };`

	executor := newTestCommandExecutor()

	for _, expectedPath := range getPotentialNamedConfLocations() {
		expectedConfigPath := path.Join(expectedPath, "named.conf")
		executor.
			clear().
			addCheckConfOutput(path.Join("/chroot", expectedConfigPath), config).
			setConfigPathInNamedOutput(expectedConfigPath)

		t.Run(expectedPath, func(t *testing.T) {
			// Act
			app := detectBind9App(
				[]string{"", "/dir", "-t /chroot -some -params"},
				"", executor, "",
			)

			// Assert
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
	config1 := `key "foo" { algorithm "hmac-sha256"; secret "abcd";};
                controls { inet 1.1.1.1 port 1111 allow { localhost; } keys { "foo"; "bar"; }; };`
	config1Path := "/dir/step1.conf"
	config2 := `key "foo" { algorithm "hmac-sha256"; secret "abcd";};
                controls { inet 2.2.2.2 port 2222 allow { localhost; } keys { "foo"; "bar"; }; };`
	config2Path := "/dir/step2.conf"
	config3 := `key "foo" { algorithm "hmac-sha256"; secret "abcd";};
                controls { inet 3.3.3.3 port 3333 allow { localhost; } keys { "foo"; "bar"; }; };`
	config3Path := "/dir/step3.conf"

	// ... and tell the fake executor to return it as the output of named -V
	executor := newTestCommandExecutor().
		addCheckConfOutput(config1Path, config1).
		addCheckConfOutput(config2Path, config2).
		addCheckConfOutput(config3Path, config3).
		setConfigPathInNamedOutput(config3Path)

	// Now run the detection as usual
	app := detectBind9App([]string{"", "/dir", fmt.Sprintf("-c %s", config1Path)}, "", executor, config2Path)
	require.NotNil(t, app)
	require.Equal(t, app.GetBaseApp().Type, AppTypeBind9)
	require.Len(t, app.GetBaseApp().AccessPoints, 1)
	point := app.GetBaseApp().AccessPoints[0]
	require.Equal(t, AccessPointControl, point.Type)
	require.Equal(t, "1.1.1.1", point.Address) // we expect the STEP 1 (-c parameter) to take precedence
	require.EqualValues(t, 1111, point.Port)
	require.EqualValues(t, "foo:hmac-sha256:abcd", point.Key)
}

// Test that the empty string is returned if the configuration content is empty.
func TestGetRndcKeyEmptyData(t *testing.T) {
	require.Nil(t, getRndcKey("", "key"))
}

// Test that the empty string is returned if the configuration content contains
// no 'key' clause.
func TestGetRndcKeyInvalidData(t *testing.T) {
	// Arrange
	content := `
		algorithm  "bar";
		secret  "baz"
	`

	// Act & Assert
	require.Nil(t, getRndcKey(content, "key"))
}

// Test that the empty string is returned if the key with a given name doesn't
// exist.
func TestGetRndcKeyUnknownKey(t *testing.T) {
	// Arrange
	content := `key "foo" {
		algorithm  "bar";
		secret  "baz";
	};`

	// Act & Assert
	require.Nil(t, getRndcKey(content, "key"))
}

// Test that the first key is returned if a given key name is an empty string.
func TestGetRndcKeyBlankName(t *testing.T) {
	// Arrange
	content := `key "foo" {
		algorithm  "bar";
		secret  "baz";
	};`

	// Act & Assert
	require.NotNil(t, getRndcKey(content, ""))
}

// Test that the empty string is returned if the algorithm property is missing.
func TestGetRndcKeyMissingAlgorithm(t *testing.T) {
	// Arrange
	content := `key "foo" {
		secret  "baz";
	};`

	// Act & Assert
	require.Nil(t, getRndcKey(content, "foo"))
}

// Test that the empty string is returned if the secret property is missing.
func TestGetRndcKeyMissingSecret(t *testing.T) {
	// Arrange
	content := `key "foo" {
		algorithm  "bar";
	};`

	// Act & Assert
	require.Nil(t, getRndcKey(content, "foo"))
}

// Test that the combination of algorithm and secret is returned if the key
// configuration entry is valid.
func TestGetRndcKeyValidData(t *testing.T) {
	// Arrange
	content := `key "foo" {
		algorithm  "bar";
		secret  "baz";
	};`

	// Act
	key := getRndcKey(content, "foo")

	// Assert
	require.NotNil(t, key)
	require.EqualValues(t, "foo", key.Name)
	require.EqualValues(t, "bar", key.Algorithm)
	require.EqualValues(t, "baz", key.Secret)
}

// Test that the combination of algorithm and secret is returned if the key
// configuration entry is valid.
func TestGetRndcKeyValidDataMultipleKeys(t *testing.T) {
	// Arrange
	content := `key "oof" {
		algorithm  "bar";
		secret  "baz";
	};

	key "foo" {
		algorithm  "bar";
		secret  "baz";
	};`

	// Act
	key := getRndcKey(content, "foo")

	// Assert
	require.NotNil(t, key)
	require.EqualValues(t, "foo", key.Name)
	require.EqualValues(t, "bar", key.Algorithm)
	require.EqualValues(t, "baz", key.Secret)
}

// Test that the key name is recognized for various formatting styles of the
// inet clause.
func TestParseInetSpecForDifferentKeysFormatting(t *testing.T) {
	// Arrange
	keysClauses := map[string]string{
		"single-name-single-space-around":    `keys { "rndc-key"; }`,
		"single-name-redundant-space-around": `keys   {   "rndc-key"   ;   }  `,
		"single-name-no-space-around":        `keys{"rndc-key";}`,
		"multi-name-single-space-around":     `keys { "rndc-key"; "foo"; "bar"; }`,
		"multi-name-redundant-space-around":  `keys  {   "rndc-key";   "foo";   "bar";   }  `,
		"multi-name-no-space-around":         `keys{"rndc-key";"foo";"bar";}`,
	}

	allowClauses := map[string]string{
		"localhost-hostname": "allow { localhost; }",
		"localhost-ip":       "allow { 127.0.0.1; }",
		"ip":                 "allow { 10.0.0.1; }",
		"empty":              "allow { }",
	}

	for keysClauseName, keysClause := range keysClauses {
		for allowClauseName, allowClause := range allowClauses {
			testCaseName := fmt.Sprintf("%s_%s", keysClauseName, allowClauseName)
			t.Run(testCaseName, func(t *testing.T) {
				inetClause := fmt.Sprintf("inet 127.0.0.1 %s %s;", allowClause, keysClause)

				config := fmt.Sprintf(`
					controls {
						%s
					};

					key "rndc-key" {
						algorithm "algorithm";
						secret "secret";
					};
				`, inetClause)

				// Act
				_, _, key := parseInetSpec(config, inetClause)

				// Assert
				require.NotNil(t, key)
				require.EqualValues(t, "rndc-key", key.Name)
				require.EqualValues(t, "algorithm", key.Algorithm)
				require.EqualValues(t, "secret", key.Secret)
			})
		}
	}
}

// Test that unquoted algorithm is parsed correctly. All other tests
// use quoted input data.
func TestGetRndcKeyUnquoted(t *testing.T) {
	keyConfig := `key "rndc-key" {
						algorithm hmac-sha256;
						secret "secret";
					};`
	key := getRndcKey(keyConfig, "rndc-key")

	require.NotNil(t, key)
	require.EqualValues(t, "hmac-sha256", key.Algorithm)
}

// Test that the wildcard addresses are resolved to the localhost address.
func TestParseInetSpecForDifferentWildcardAddresses(t *testing.T) {
	// Arrange
	addresses := []string{"*", "0.0.0.0", "::", "localhost"}

	for _, address := range addresses {
		inetClause := fmt.Sprintf("inet %s allow { };", address)

		config := fmt.Sprintf(`
			controls {
				%s
			};
		`, inetClause)

		// Act
		address, _, _ := parseInetSpec(config, inetClause)

		// Assert
		require.Equal(t, "localhost", address)
	}
}

// Test that the RNDC key is converted to string properly.
func TestBind9RndcKeyString(t *testing.T) {
	// Arrange
	key := &Bind9RndcKey{
		Name:      "foo",
		Algorithm: "bar",
		Secret:    "baz",
	}

	// Act & Assert
	require.EqualValues(t, "foo:bar:baz", key.String())
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
	key := &Bind9RndcKey{Name: "name", Algorithm: "alg", Secret: "sec"}

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
	key := &Bind9RndcKey{Name: "name", Algorithm: "alg", Secret: "sec"}

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
	key := &Bind9RndcKey{Name: "name", Algorithm: "alg", Secret: "sec"}

	// Act
	err := client.DetermineDetails("/exe_dir", "/conf_dir", "address", 42, key)

	// Assert
	require.NoError(t, err)
	require.Equal(t, expectedBaseCommand, client.BaseCommand)
}

// Test that awaiting background tasks doesn't panic when zone inventory is nil.
func TestBind9AppAwaitBackgroundTasksNilZoneInventory(t *testing.T) {
	app := &Bind9App{}
	require.NotPanics(t, app.AwaitBackgroundTasks)
}
