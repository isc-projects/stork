package agent

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"isc.org/stork/testutil"
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

// Test that the list of the configured daemon contains only the named daemon.
func TestGetConfiguredDaemons(t *testing.T) {
	// Arrange
	app := &Bind9App{}

	// Act
	daemons := app.GetConfiguredDaemons()

	// Assert
	require.Len(t, daemons, 1)
	require.Contains(t, daemons, "named")
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
			require.Equal(t, c, test.expKey)
		})
	}
}

type catCommandExecutor struct{ file string }

// Pretends to run named-checkconf, but instead does a simple read of the
// specified files contents, similar to "cat" command.
func (e *catCommandExecutor) Output(command string, args ...string) ([]byte, error) {
	if strings.Contains(command, "named-checkconf") {
		// Pretending to run named-checkconf -p <config-file>. The contents of
		// the file are returned as-is.
		text, err := ioutil.ReadFile(args[1])
		if err != nil {
			// Reading failed.
			return nil, err
		}
		return text, nil
	}

	if strings.HasSuffix(command, "named") && len(args) > 0 && args[0] == "-V" {
		// Pretending to run named -V
		text := fmt.Sprintf(`default paths:
		named configuration:  %s
		rndc configuration:   /other/path/rndc.conf`, e.file)

		return []byte(text), nil
	}

	return nil, nil
}

// Looks for a given command in the system PATH and returns absolute path if found.
// (This is the standard behavior that we don't override in tests here.)
func (e *catCommandExecutor) LookPath(command string) (string, error) {
	return e.file + "/" + command, nil
}

// Checks detection STEP 1: if BIND9 detection takes -c parameter into consideration.
func TestDetectBind9Step1ProcessCmdLine(t *testing.T) {
	sb := testutil.NewSandbox()
	defer sb.Close()

	// create alternate config files for each step...
	config1Path, _ := sb.Join("step1.conf")
	config1 := `keys "foo" { algorithm "hmac-sha256"; secret "abcd";};
                controls { inet 1.1.1.1 port 1111 allow { localhost; } keys { "foo"; "bar"; }; };`
	sb.Write("step1.conf", config1)

	// check BIND 9 app detection
	executor := &catCommandExecutor{}

	// Now run the detection as usual
	namedDir, err := sb.JoinDir("usr/sbin")
	require.NoError(t, err)
	_, err = sb.Join("usr/bin/named-checkconf")
	require.NoError(t, err)
	_, err = sb.Join("usr/sbin/rndc")
	require.NoError(t, err)
	app := detectBind9App([]string{"", namedDir, fmt.Sprintf("-c %s", config1Path)}, "", executor)
	require.NotNil(t, app)
	require.Equal(t, app.GetBaseApp().Type, AppTypeBind9)
	require.Len(t, app.GetBaseApp().AccessPoints, 1)
	point := app.GetBaseApp().AccessPoints[0]
	require.Equal(t, AccessPointControl, point.Type)
	require.Equal(t, "1.1.1.1", point.Address)
	require.EqualValues(t, 1111, point.Port)
	require.Empty(t, point.Key)
}

// Checks detection STEP 2: if BIND9 detection takes STORK_BIND9_CONFIG env var into account.
func TestDetectBind9Step2EnvVar(t *testing.T) {
	sb := testutil.NewSandbox()
	defer sb.Close()

	restore := testutil.CreateEnvironmentRestorePoint()
	defer restore()

	// create alternate config file...
	varPath, _ := sb.Join("testing.conf")
	config := `keys "foo" {
		algorithm "hmac-sha256";
		secret "abcd";
   };
controls {
		inet 192.0.2.1 port 1234 allow { localhost; } keys { "foo"; "bar"; };
   };`
	sb.Write("testing.conf", config)

	// ... and point STORK_BIND9_CONFIG to it
	os.Setenv("STORK_BIND9_CONFIG", varPath)

	// check BIND 9 app detection
	executor := &catCommandExecutor{}

	namedDir, err := sb.JoinDir("usr/sbin")
	require.NoError(t, err)
	_, err = sb.Join("usr/bin/named-checkconf")
	require.NoError(t, err)
	_, err = sb.Join("usr/sbin/rndc")
	require.NoError(t, err)
	app := detectBind9App([]string{"", namedDir, "-some -params"}, "", executor)
	require.NotNil(t, app)
	require.Equal(t, app.GetBaseApp().Type, AppTypeBind9)
	require.Len(t, app.GetBaseApp().AccessPoints, 1)
	point := app.GetBaseApp().AccessPoints[0]
	require.Equal(t, AccessPointControl, point.Type)
	require.Equal(t, "192.0.2.1", point.Address)
	require.EqualValues(t, 1234, point.Port)
	require.Empty(t, point.Key)
}

// Checks detection STEP 3: parse output of the named -V command.
func TestDetectBind9Step3BindVOutput(t *testing.T) {
	sb := testutil.NewSandbox()
	defer sb.Close()

	// create alternate config file...
	varPath, _ := sb.Join("testing.conf")
	config := `keys "foo" {
		algorithm "hmac-sha256";
		secret "abcd";
   };
controls {
		inet 192.0.2.1 port 1234 allow { localhost; } keys { "foo"; "bar"; };
   };`
	sb.Write("testing.conf", config)

	// ... and tell the fake executor to return it as the output of named -V
	executor := &catCommandExecutor{file: varPath}

	// Now run the detection as usual
	namedDir, err := sb.JoinDir("usr/sbin")
	require.NoError(t, err)
	_, err = sb.Join("usr/bin/named-checkconf")
	require.NoError(t, err)
	_, err = sb.Join("usr/sbin/rndc")
	require.NoError(t, err)
	app := detectBind9App([]string{"", namedDir, "-some -params"}, "", executor)
	require.NotNil(t, app)
	require.Equal(t, app.GetBaseApp().Type, AppTypeBind9)
	require.Len(t, app.GetBaseApp().AccessPoints, 1)
	point := app.GetBaseApp().AccessPoints[0]
	require.Equal(t, AccessPointControl, point.Type)
	require.Equal(t, "192.0.2.1", point.Address)
	require.EqualValues(t, 1234, point.Port)
	require.Empty(t, point.Key)
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
// - step2.conf (which is passed in STORK_BIND9_CONFIG)
// - step3.conf (which is returned by named -V).
func TestDetectBind9DetectOrder(t *testing.T) {
	sb := testutil.NewSandbox()
	defer sb.Close()

	restore := testutil.CreateEnvironmentRestorePoint()
	defer restore()

	// create alternate config files for each step...
	config1Path, _ := sb.Join("step1.conf")
	config1 := `keys "foo" { algorithm "hmac-sha256"; secret "abcd";};
                controls { inet 1.1.1.1 port 1111 allow { localhost; } keys { "foo"; "bar"; }; };`
	sb.Write("step1.conf", config1)

	config2Path, _ := sb.Join("step2.conf")
	config2 := `keys "foo" { algorithm "hmac-sha256"; secret "abcd";};
                controls { inet 2.2.2.2 port 2222 allow { localhost; } keys { "foo"; "bar"; }; };`
	sb.Write("step2.conf", config2)

	config3Path, _ := sb.Join("step3.conf")
	config3 := `keys "foo" { algorithm "hmac-sha256"; secret "abcd";};
                controls { inet 3.3.3.3 port 3333 allow { localhost; } keys { "foo"; "bar"; }; };`
	sb.Write("step3.conf", config3)

	// ... and tell the fake executor to return it as the output of named -V
	executor := &catCommandExecutor{file: config3Path}

	// ... and point STORK_BIND9_CONFIG to it
	os.Setenv("STORK_BIND9_CONFIG", config2Path)

	// Now run the detection as usual
	namedDir, err := sb.JoinDir("usr/sbin")
	require.NoError(t, err)
	_, err = sb.Join("usr/bin/named-checkconf")
	require.NoError(t, err)
	_, err = sb.Join("usr/sbin/rndc")
	require.NoError(t, err)
	app := detectBind9App([]string{"", namedDir, fmt.Sprintf("-c %s", config1Path)}, "", executor)
	require.NotNil(t, app)
	require.Equal(t, app.GetBaseApp().Type, AppTypeBind9)
	require.Len(t, app.GetBaseApp().AccessPoints, 1)
	point := app.GetBaseApp().AccessPoints[0]
	require.Equal(t, AccessPointControl, point.Type)
	require.Equal(t, "1.1.1.1", point.Address) // we expect the STEP 1 (-c parameter) to take precedence
	require.EqualValues(t, 1111, point.Port)
	require.Empty(t, point.Key)
}

// Test that the empty string is returned if the configuration content is empty.
func TestGetRndcKeyEmptyData(t *testing.T) {
	require.Empty(t, getRndcKey("", "key"))
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
	require.Empty(t, getRndcKey(content, "key"))
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
	require.Empty(t, getRndcKey(content, "key"))
}

// Test that the empty string is returned if a given name is an empty string.
func TestGetRndcKeyBlankName(t *testing.T) {
	// Arrange
	content := `key "foo" {
		algorithm  "bar";
		secret  "baz";
	};`

	// Act & Assert
	require.Empty(t, getRndcKey(content, ""))
}

// Test that the empty string is returned if the algorithm property is missing.
func TestGetRndcKeyMissingAlgorithm(t *testing.T) {
	// Arrange
	content := `key "foo" {
		secret  "baz";
	};`

	// Act & Assert
	require.Empty(t, getRndcKey(content, "foo"))
}

// Test that the empty string is returned if the secret property is missing.
func TestGetRndcKeyMissingSecret(t *testing.T) {
	// Arrange
	content := `key "foo" {
		algorithm  "bar";
	};`

	// Act & Assert
	require.Empty(t, getRndcKey(content, "foo"))
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
	require.EqualValues(t, "bar:baz", key)
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
	require.EqualValues(t, "bar:baz", key)
}
