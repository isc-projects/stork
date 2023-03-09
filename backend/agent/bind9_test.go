package agent

import (
	"testing"

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

func TestParseNamedDefaultPaths(t *testing.T) {
	// Define input data
	input := `default paths:
                named configuration:  /some/path/named.conf
                rndc configuration:   /other/path/rndc.conf`

	// Convert input data to []byte
	output := []byte(input)

	// Call parseNamedDefaultPaths with the output
	NamedConf, RndcConf := parseNamedDefaultPaths(output)

	// Assert that the parsed strings are correct
	require.Equal(t, "/some/path/named.conf", NamedConf)
	require.Equal(t, "/other/path/rndc.conf", RndcConf)
}

// Old BIND 9 versions don't print the default paths.
// This test uses actual BIND 9.11.5 output. Makes sure
// that the function doesn't panic.
func TestParseNamedDefaultPathsForOldBind9Versions(t *testing.T) {
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

	// Call parseNamedDefaultPaths with the output
	namedConf, RdncConf := parseNamedDefaultPaths(output)

	// Assert that the returned values are empty
	require.Equal(t, "", namedConf)
	require.Equal(t, "", RdncConf)
}

// Tests if getCtrlAddressFromBind9Config() can handle the right
// cases:
// - CASE 1: no controls block (use defaults)
// - CASE 2: controls block with no options (return nothing)
// - CASE 3: controls block with options (return the address).
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
		"CASE 2: added empty controls section (disabled rndc)": {config: "controls { };", expAddr: "", expPort: 0, expKey: ""},
		"CASE 3: added controls section with options": {config: `
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
