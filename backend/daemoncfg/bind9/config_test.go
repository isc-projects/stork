package bind9config

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"isc.org/stork/testutil"
)

// Test getting the source path of the configuration file.
func TestConfigGetSourcePath(t *testing.T) {
	cfg := &Config{
		sourcePath: "testdata/dir/named.conf",
	}
	require.Equal(t, "testdata/dir/named.conf", cfg.GetSourcePath())
}

// Test checking if the configuration contains no-parse directives.
func TestConfigHasNoParse(t *testing.T) {
	cfg := &Config{
		Statements: []*Statement{
			{Options: &Options{
				Clauses: []*OptionClause{},
			}},
			{NoParse: &NoParse{}},
		},
	}
	require.True(t, cfg.HasNoParse())
}

// Test checking if the configuration does not contain no-parse directives.
func TestConfigHasNoParseNone(t *testing.T) {
	cfg := &Config{}
	require.False(t, cfg.HasNoParse())
}

// Test getting the first controls statement.
func TestGetControls(t *testing.T) {
	cfg := &Config{
		Statements: []*Statement{
			{StatisticsChannels: &StatisticsChannels{}},
			{Controls: &Controls{}},
			{Controls: &Controls{}},
		},
	}
	controls := cfg.GetControls()
	require.NotNil(t, controls)
	require.Equal(t, cfg.Statements[1].Controls, controls)
}

// Test that nil is returned when getting the controls statement
// when it does not exist.
func TestGetControlsNone(t *testing.T) {
	cfg := &Config{}
	require.Nil(t, cfg.GetControls())
}

// Test getting the first statistics channels statement.
func TestGetStatisticsChannels(t *testing.T) {
	cfg := &Config{
		Statements: []*Statement{
			{Controls: &Controls{}},
			{StatisticsChannels: &StatisticsChannels{}},
			{StatisticsChannels: &StatisticsChannels{}},
		},
	}
	statisticsChannels := cfg.GetStatisticsChannels()
	require.NotNil(t, statisticsChannels)
	require.Equal(t, cfg.Statements[1].StatisticsChannels, cfg.GetStatisticsChannels())
}

// Test that nil is returned when getting the statistics channels statement
// when it does not exist.
func TestGetStatisticsChannelsNone(t *testing.T) {
	cfg := &Config{}
	require.Nil(t, cfg.GetStatisticsChannels())
}

// Tests that GetView returns expected view.
func TestGetView(t *testing.T) {
	cfg, err := NewParser().ParseFile("testdata/dir/named.conf", "")
	require.NoError(t, err)
	require.NotNil(t, cfg)

	view := cfg.GetView("trusted")
	require.NotNil(t, view)
	require.Equal(t, "trusted", view.Name)

	view = cfg.GetView("non-existent")
	require.Nil(t, view)
}

// Tests that GetKey returns expected key.
func TestGetKey(t *testing.T) {
	cfg, err := NewParser().ParseFile("testdata/dir/named.conf", "")
	require.NoError(t, err)
	require.NotNil(t, cfg)

	key := cfg.GetKey("trusted-key")
	require.NotNil(t, key)
	require.Equal(t, "trusted-key", key.Name)
	algorithm, secret, err := key.GetAlgorithmSecret()
	require.NoError(t, err)
	require.Equal(t, "hmac-sha256", algorithm)
	require.Equal(t, "VO6xA4Tc1PWYaqMuPaf6wfkITb+c9/mkzlEaWJavejU=", secret)

	key = cfg.GetKey("non-existent")
	require.Nil(t, key)
}

// Test getting the first key in the configuration file.
func TestGetFirstKey(t *testing.T) {
	cfg, err := NewParser().ParseFile("testdata/dir/named.conf", "")
	require.NoError(t, err)
	require.NotNil(t, cfg)

	key := cfg.GetFirstKey()
	require.NotNil(t, key)
	require.Equal(t, "trusted-key", key.Name)
}

// Test that nil is returned while getting the first key when no key is found.
func TestGetFirstKeyNone(t *testing.T) {
	cfg := &Config{}
	require.Nil(t, cfg.GetFirstKey())
}

// Tests that GetACL returns expected ACL.
func TestGetACL(t *testing.T) {
	cfg, err := NewParser().ParseFile("testdata/dir/named.conf", "")
	require.NoError(t, err)
	require.NotNil(t, cfg)

	cfg, err = cfg.Expand()
	require.NoError(t, err)
	require.NotNil(t, cfg)

	acl := cfg.GetACL("trusted-networks")
	require.NotNil(t, acl)
	require.Equal(t, "trusted-networks", acl.Name)
	require.Len(t, acl.AddressMatchList.Elements, 8)

	acl = cfg.GetACL("non-existent")
	require.Nil(t, acl)
}

// Tests that GetViewKey emits an error indicating too much recursion for cyclic
// dependencies between ACLs.
func TestViewKeysTooMuchRecursion(t *testing.T) {
	config := `
		acl acl1 { !key negated-key; acl2; };
		acl acl2 { acl3; };
		acl acl3 { acl1; };
		view trusted {
			match-clients { acl1; };
		};
	`
	cfg, err := NewParser().Parse("", "", strings.NewReader(config))
	require.NoError(t, err)
	require.NotNil(t, cfg)

	cfg, err = cfg.Expand()
	require.NoError(t, err)
	require.NotNil(t, cfg)

	key, err := cfg.GetZoneKey("trusted", "example.com")
	require.ErrorContains(t, err, "too much recursion in address-match-list")
	require.Nil(t, key)
}

// Tests that GetViewKey returns associated keys.
func TestGetViewKey(t *testing.T) {
	cfg, err := NewParser().ParseFile("testdata/dir/named.conf", "")
	require.NoError(t, err)
	require.NotNil(t, cfg)

	cfg, err = cfg.Expand()
	require.NoError(t, err)
	require.NotNil(t, cfg)

	key, err := cfg.GetZoneKey("trusted", "example.com")
	require.NoError(t, err)
	require.NotNil(t, key)
	require.Equal(t, "trusted-key", key.Name)
	algorithm, secret, err := key.GetAlgorithmSecret()
	require.NoError(t, err)
	require.Equal(t, "hmac-sha256", algorithm)
	require.Equal(t, "VO6xA4Tc1PWYaqMuPaf6wfkITb+c9/mkzlEaWJavejU=", secret)

	key, err = cfg.GetZoneKey("guest", "example.com")
	require.NoError(t, err)
	require.NotNil(t, key)
	require.Equal(t, "guest-key", key.Name)
	algorithm, secret, err = key.GetAlgorithmSecret()
	require.NoError(t, err)
	require.Equal(t, "hmac-sha256", algorithm)
	require.Equal(t, "6L8DwXFboA7FDQJQP051hjFV/n9B3IR/SwDLX7y5czE=", secret)

	key, err = cfg.GetZoneKey("non-existent", "example.com")
	require.NoError(t, err)
	require.Nil(t, key)
}

// Test that that IPv4 listener address and port is used when the
// allow-transfer clause matches.
func TestGetAXFRCredentialsForViewListenOnIPv4(t *testing.T) {
	config := `
		options {
			allow-transfer port 853 { key trusted-key; };
			listen-on port 853 { 192.0.2.1; 127.0.0.1; };
			listen-on port 54 { 192.0.2.1; 127.0.0.1; };
			listen-on-v6 port 54 { 2001:db8:1::1; };
		};
		key "trusted-key" {
			algorithm hmac-sha256;
			secret "VO6xA4Tc1PWYaqMuPaf6wfkITb+c9/mkzlEaWJavejU=";
		};
	`
	cfg, err := NewParser().Parse("", "", strings.NewReader(config))
	require.NoError(t, err)
	require.NotNil(t, cfg)

	address, keyName, algorithm, secret, err := cfg.GetAXFRCredentials(DefaultViewName, "example.com")
	require.NoError(t, err)
	// Localhost is preferred. The port should be 853 as it matches both
	// the listener port and the allow-transfer port.
	require.Equal(t, "127.0.0.1:853", address)
	require.Equal(t, "trusted-key", keyName)
	require.Equal(t, "hmac-sha256", algorithm)
	require.Equal(t, "VO6xA4Tc1PWYaqMuPaf6wfkITb+c9/mkzlEaWJavejU=", secret)
}

// Test that the local loopback listener address is preferred even if the first
// listen-on clause contains a different address.
func TestGetAXFRCredentialsForViewListenOnMultipleClauses(t *testing.T) {
	config := `
		options {
			allow-transfer { key trusted-key; };
			listen-on { 192.0.2.1; };
			listen-on-v6 { 2001:db8:1::1; };
			listen-on port 53 { 127.0.0.1; };
		};
		key "trusted-key" {
			algorithm hmac-sha256;
			secret "VO6xA4Tc1PWYaqMuPaf6wfkITb+c9/mkzlEaWJavejU=";
		};
		view "trusted" {
			match-clients { key trusted-key; };
		};
	`
	cfg, err := NewParser().Parse("", "", strings.NewReader(config))
	require.NoError(t, err)
	require.NotNil(t, cfg)

	address, keyName, algorithm, secret, err := cfg.GetAXFRCredentials("trusted", "example.com")
	require.NoError(t, err)
	// The local loopback address from the second clause is preferred.
	require.Equal(t, "127.0.0.1:53", address)
	require.NotNil(t, keyName)
	require.Equal(t, "trusted-key", keyName)
	require.Equal(t, "hmac-sha256", algorithm)
	require.Equal(t, "VO6xA4Tc1PWYaqMuPaf6wfkITb+c9/mkzlEaWJavejU=", secret)
}

// Test that the IPv6 local loopback address is preferred over non-loopback
// IPv4 addresses.
func TestGetAXFRCredentialsForViewListenOnIPv6(t *testing.T) {
	config := `
		options {
			allow-transfer port 54 { key trusted-key; };
			listen-on port 853 { 192.0.2.1; };
			listen-on port 54 { 192.0.2.1; };
			listen-on-v6 port 54 { 2001:db8:1::1; ::1; };
		};
		key "trusted-key" {
			algorithm hmac-sha256;
			secret "VO6xA4Tc1PWYaqMuPaf6wfkITb+c9/mkzlEaWJavejU=";
		};
		view "trusted" {
			match-clients { key trusted-key; };
		};
	`
	cfg, err := NewParser().Parse("", "", strings.NewReader(config))
	require.NoError(t, err)
	require.NotNil(t, cfg)

	address, keyName, algorithm, secret, err := cfg.getAXFRCredentialsForView("trusted", "example.com")
	require.NoError(t, err)
	require.Equal(t, "[::1]:54", address)
	require.NotNil(t, keyName)
	require.Equal(t, "trusted-key", keyName)
	require.Equal(t, "hmac-sha256", algorithm)
	require.Equal(t, "VO6xA4Tc1PWYaqMuPaf6wfkITb+c9/mkzlEaWJavejU=", secret)
}

// Test that the address is picked that matches the port specified in the
// allow-transfer clause.
func TestGetAXFRCredentialsForViewListenOnAllowTransferPreferredPort(t *testing.T) {
	config := `
		options {
			allow-transfer port 853 { key trusted-key; };
			listen-on port 54 { 127.0.0.1; };
			listen-on-v6 port 853 { 2001:db8:1::1; ::1; };
		};
		key "trusted-key" {
			algorithm hmac-sha256;
			secret "VO6xA4Tc1PWYaqMuPaf6wfkITb+c9/mkzlEaWJavejU=";
		};
		view "trusted" {
			match-clients { key trusted-key; };
		};
	`
	cfg, err := NewParser().Parse("", "", strings.NewReader(config))
	require.NoError(t, err)
	require.NotNil(t, cfg)

	address, keyName, algorithm, secret, err := cfg.getAXFRCredentialsForView("trusted", "example.com")
	require.NoError(t, err)
	// If the port was not specified in the allow-transfer clause, the IPv4
	// loopback address would have been selected. However, since the port is
	// specified, we try to match it with the address listening on that port.
	require.Equal(t, "[::1]:853", address)
	require.Equal(t, "trusted-key", keyName)
	require.Equal(t, "hmac-sha256", algorithm)
	require.Equal(t, "VO6xA4Tc1PWYaqMuPaf6wfkITb+c9/mkzlEaWJavejU=", secret)
}

// Test that the local loopback address is picked when the allow-transfer clause
// is set to any and the listen-on clause is not specified.
func TestGetAXFRCredentialsForViewListenOnAllowTransferAny(t *testing.T) {
	config := `
		options {
			allow-transfer { any; };
		};
		key "trusted-key" {
			algorithm hmac-sha256;
			secret "VO6xA4Tc1PWYaqMuPaf6wfkITb+c9/mkzlEaWJavejU=";
		};
		view "trusted" {
			match-clients { key trusted-key; };
		};
	`
	cfg, err := NewParser().Parse("", "", strings.NewReader(config))
	require.NoError(t, err)
	require.NotNil(t, cfg)

	address, keyName, algorithm, secret, err := cfg.getAXFRCredentialsForView("trusted", "example.com")
	require.NoError(t, err)
	require.Equal(t, "127.0.0.1:53", address)
	require.Equal(t, "trusted-key", keyName)
	require.Equal(t, "hmac-sha256", algorithm)
	require.Equal(t, "VO6xA4Tc1PWYaqMuPaf6wfkITb+c9/mkzlEaWJavejU=", secret)
}

// Test that IPv6 local loopback address is picked when the allow-transfer clause
// is set to any and the listen-on-v6 clause contains the local loopback address.
func TestGetAXFRCredentialsForViewListenOnAllowTransferZoneOverride(t *testing.T) {
	config := `
		options {
			allow-transfer port 854 { any; };
			listen-on port 54 { 127.0.0.1; };
			listen-on-v6 port 853 { 2001:db8:1::1; ::1; };
		};
		key "trusted-key" {
			algorithm hmac-sha256;
			secret "VO6xA4Tc1PWYaqMuPaf6wfkITb+c9/mkzlEaWJavejU=";
		};
		view "trusted" {
			match-clients { key trusted-key; };
			zone "example.com" {
				allow-transfer port 853 { any; };
			};
		};
	`
	cfg, err := NewParser().Parse("", "", strings.NewReader(config))
	require.NoError(t, err)
	require.NotNil(t, cfg)

	address, keyName, algorithm, secret, err := cfg.getAXFRCredentialsForView("trusted", "example.com")
	require.NoError(t, err)
	require.Equal(t, "[::1]:853", address)
	require.Equal(t, "trusted-key", keyName)
	require.Equal(t, "hmac-sha256", algorithm)
	require.Equal(t, "VO6xA4Tc1PWYaqMuPaf6wfkITb+c9/mkzlEaWJavejU=", secret)
}

// Test that the errors is returned when the allow-transfer port does not match the
// default listen-on port.
func TestGetAXFRCredentialsForViewListenOnAllowTransferZoneOverrideNoListener(t *testing.T) {
	config := `
		options {
			allow-transfer port 854 { any; };
		};
		key "trusted-key" {
			algorithm hmac-sha256;
			secret "VO6xA4Tc1PWYaqMuPaf6wfkITb+c9/mkzlEaWJavejU=";
		};
		view "trusted" {
			match-clients { key trusted-key; };
			zone "example.com" {
				allow-transfer port 853 { any; };
			};
		};
	`
	cfg, err := NewParser().Parse("", "", strings.NewReader(config))
	require.NoError(t, err)
	require.NotNil(t, cfg)

	address, keyName, algorithm, secret, err := cfg.getAXFRCredentialsForView("trusted", "example.com")
	require.ErrorContains(t, err, "allow-transfer port 853 does not match any listen-on setting")
	require.Empty(t, address)
	require.Empty(t, keyName)
	require.Empty(t, algorithm)
	require.Empty(t, secret)
}

// Test that the error is returned when the allow-transfer port does not match the
// listen-on port.
func TestGetAXFRCredentialsForViewListenOnAllowTransferZoneOverrideListenerMismatch(t *testing.T) {
	config := `
		options {
			allow-transfer port 854 { any; };
			listen-on port 54 { 127.0.0.1; };
		};
		key "trusted-key" {
			algorithm hmac-sha256;
			secret "VO6xA4Tc1PWYaqMuPaf6wfkITb+c9/mkzlEaWJavejU=";
		};
		view "trusted" {
			match-clients { key trusted-key; };
			zone "example.com" {
				allow-transfer port 853 { any; };
			};
		};
	`
	cfg, err := NewParser().Parse("", "", strings.NewReader(config))
	require.NoError(t, err)
	require.NotNil(t, cfg)

	address, keyName, algorithm, secret, err := cfg.getAXFRCredentialsForView("trusted", "example.com")
	require.ErrorContains(t, err, "allow-transfer port 853 does not match any listen-on setting")
	require.Empty(t, address)
	require.Empty(t, keyName)
	require.Empty(t, algorithm)
	require.Empty(t, secret)
}

// Test that the error is returned when the allow-transfer is disabled.
func TestGetAXFRCredentialsForViewNoAllowTransfer(t *testing.T) {
	config := `
		options {
			listen-on port 54 { 192.0.2.1; 127.0.0.1; };
			listen-on-v6 port 54 { 2001:db8:1::1; };
		};
		key "trusted-key" {
			algorithm hmac-sha256;
			secret "VO6xA4Tc1PWYaqMuPaf6wfkITb+c9/mkzlEaWJavejU=";
		};
		view "trusted" {
			match-clients { key trusted-key; };
		};
	`
	cfg, err := NewParser().Parse("", "", strings.NewReader(config))
	require.NoError(t, err)
	require.NotNil(t, cfg)

	address, keyName, algorithm, secret, err := cfg.getAXFRCredentialsForView("trusted", "example.com")
	require.ErrorContains(t, err, "allow-transfer is disabled")
	require.Empty(t, address)
	require.Empty(t, keyName)
	require.Empty(t, algorithm)
	require.Empty(t, secret)
}

// Test that the allow-transfer option is used to identify the correct key
// for the zone transfer when match-clients is not specified.
func TestGetAXFRCredentialsForView(t *testing.T) {
	config := `
		options {
			listen-on port 54 { 192.0.2.2; };
			listen-on-v6 port 54 { ::1; };
		};
		key "trusted-key" {
			algorithm hmac-sha256;
			secret "VO6xA4Tc1PWYaqMuPaf6wfkITb+c9/mkzlEaWJavejU=";
		};

		key "guest-key" {
			algorithm hmac-sha256;
			secret "6L8DwXFboA7FDQJQP051hjFV/n9B3IR/SwDLX7y5czE=";
		};

		view "trusted" {
			allow-transfer port 54 transport tls { key trusted-key; };
		};

		view "guest" {
			allow-transfer transport "tls" { key guest-key; };
		};
	`
	cfg, err := NewParser().Parse("", "", strings.NewReader(config))
	require.NoError(t, err)
	require.NotNil(t, cfg)

	address, keyName, algorithm, secret, err := cfg.GetAXFRCredentials("trusted", "example.com")
	require.NoError(t, err)
	require.Equal(t, "[::1]:54", address)
	require.Equal(t, "trusted-key", keyName)
	require.Equal(t, "hmac-sha256", algorithm)
	require.Equal(t, "VO6xA4Tc1PWYaqMuPaf6wfkITb+c9/mkzlEaWJavejU=", secret)
}

// Test that that IPv4 listener address and port is used when the
// allow-transfer clause matches.
func TestGetAXFRCredentialsForDefaultViewListenOnIPv4(t *testing.T) {
	config := `
		options {
			allow-transfer port 853 { key trusted-key; };
			listen-on port 853 { 192.0.2.1; 127.0.0.1; };
			listen-on port 54 { 192.0.2.1; 127.0.0.1; };
			listen-on-v6 port 54 { 2001:db8:1::1; };
		};
		key "trusted-key" {
			algorithm hmac-sha256;
			secret "VO6xA4Tc1PWYaqMuPaf6wfkITb+c9/mkzlEaWJavejU=";
		};
	`
	cfg, err := NewParser().Parse("", "", strings.NewReader(config))
	require.NoError(t, err)
	require.NotNil(t, cfg)

	address, keyName, algorithm, secret, err := cfg.GetAXFRCredentials(DefaultViewName, "example.com")
	require.NoError(t, err)
	// Localhost is preferred. The port should be 853 as it matches both
	// the listener port and the allow-transfer port.
	require.Equal(t, "127.0.0.1:853", address)
	require.Equal(t, "trusted-key", keyName)
	require.Equal(t, "hmac-sha256", algorithm)
	require.Equal(t, "VO6xA4Tc1PWYaqMuPaf6wfkITb+c9/mkzlEaWJavejU=", secret)
}

// Test that the local loopback listener address is preferred even if the first
// listen-on clause contains a different address.
func TestGetAXFRCredentialsForDefaultViewListenOnMultipleClauses(t *testing.T) {
	config := `
		options {
			allow-transfer { key trusted-key; };
			listen-on { 192.0.2.1; };
			listen-on-v6 { 2001:db8:1::1; };
			listen-on port 53 { 127.0.0.1; };
		};
		key "trusted-key" {
			algorithm hmac-sha256;
			secret "VO6xA4Tc1PWYaqMuPaf6wfkITb+c9/mkzlEaWJavejU=";
		};
	`
	cfg, err := NewParser().Parse("", "", strings.NewReader(config))
	require.NoError(t, err)
	require.NotNil(t, cfg)

	address, keyName, algorithm, secret, err := cfg.GetAXFRCredentials(DefaultViewName, "example.com")
	require.NoError(t, err)
	// The local loopback address from the second clause is preferred.
	require.Equal(t, "127.0.0.1:53", address)
	require.Equal(t, "trusted-key", keyName)
	require.Equal(t, "hmac-sha256", algorithm)
	require.Equal(t, "VO6xA4Tc1PWYaqMuPaf6wfkITb+c9/mkzlEaWJavejU=", secret)
}

// Test that the IPv6 local loopback address is preferred over non-loopback
// IPv4 addresses.
func TestGetAXFRCredentialsForDefaultViewListenOnIPv6(t *testing.T) {
	config := `
		options {
			allow-transfer port 54 { key trusted-key; };
			listen-on port 853 { 192.0.2.1; };
			listen-on port 54 { 192.0.2.1; };
			listen-on-v6 port 54 { 2001:db8:1::1; ::1; };
		};
		key "trusted-key" {
			algorithm hmac-sha256;
			secret "VO6xA4Tc1PWYaqMuPaf6wfkITb+c9/mkzlEaWJavejU=";
		};
	`
	cfg, err := NewParser().Parse("", "", strings.NewReader(config))
	require.NoError(t, err)
	require.NotNil(t, cfg)

	address, keyName, algorithm, secret, err := cfg.GetAXFRCredentials(DefaultViewName, "example.com")
	require.NoError(t, err)
	require.Equal(t, "[::1]:54", address)
	require.Equal(t, "trusted-key", keyName)
	require.Equal(t, "hmac-sha256", algorithm)
	require.Equal(t, "VO6xA4Tc1PWYaqMuPaf6wfkITb+c9/mkzlEaWJavejU=", secret)
}

// Test that the address is picked that matches the port specified in the
// allow-transfer clause.
func TestGetAXFRCredentialsForDefaultViewListenOnAllowTransferPreferredPort(t *testing.T) {
	config := `
		options {
			allow-transfer port 853 { key trusted-key; };
			listen-on port 54 { 127.0.0.1; };
			listen-on-v6 port 853 { 2001:db8:1::1; ::1; };
		};
		key "trusted-key" {
			algorithm hmac-sha256;
			secret "VO6xA4Tc1PWYaqMuPaf6wfkITb+c9/mkzlEaWJavejU=";
		};
	`
	cfg, err := NewParser().Parse("", "", strings.NewReader(config))
	require.NoError(t, err)
	require.NotNil(t, cfg)

	address, keyName, algorithm, secret, err := cfg.GetAXFRCredentials(DefaultViewName, "example.com")
	require.NoError(t, err)
	// If the port was not specified in the allow-transfer clause, the IPv4
	// loopback address would have been selected. However, since the port is
	// specified, we try to match it with the address listening on that port.
	require.Equal(t, "[::1]:853", address)
	require.Equal(t, "trusted-key", keyName)
	require.Equal(t, "hmac-sha256", algorithm)
	require.Equal(t, "VO6xA4Tc1PWYaqMuPaf6wfkITb+c9/mkzlEaWJavejU=", secret)
}

func TestGetAXFRCredentialsForDefaultViewListenOnAllowTransferAny(t *testing.T) {
	config := `
		options {
			allow-transfer { any; };
		};
	`
	cfg, err := NewParser().Parse("", "", strings.NewReader(config))
	require.NoError(t, err)
	require.NotNil(t, cfg)

	address, keyName, algorithm, secret, err := cfg.GetAXFRCredentials(DefaultViewName, "example.com")
	require.NoError(t, err)
	require.Equal(t, "127.0.0.1:53", address)
	require.Empty(t, keyName)
	require.Empty(t, algorithm)
	require.Empty(t, secret)
}

func TestGetAXFRCredentialsForDefaultViewListenOnAllowTransferZoneOverride(t *testing.T) {
	config := `
		options {
			allow-transfer port 854 { any; };
			listen-on port 54 { 127.0.0.1; };
			listen-on-v6 port 853 { 2001:db8:1::1; ::1; };
		};
		zone "example.com" {
			allow-transfer port 853 { any; };
		};
	`
	cfg, err := NewParser().Parse("", "", strings.NewReader(config))
	require.NoError(t, err)
	require.NotNil(t, cfg)

	address, keyName, algorithm, secret, err := cfg.GetAXFRCredentials(DefaultViewName, "example.com")
	require.NoError(t, err)
	require.Equal(t, "[::1]:853", address)
	require.Empty(t, keyName)
	require.Empty(t, algorithm)
	require.Empty(t, secret)
}

func TestGetAXFRCredentialsForDefaultViewListenOnAllowTransferZoneOverrideNoListener(t *testing.T) {
	config := `
		options {
			allow-transfer port 854 { any; };
		};
		zone "example.com" {
			allow-transfer port 853 { any; };
		};
	`
	cfg, err := NewParser().Parse("", "", strings.NewReader(config))
	require.NoError(t, err)
	require.NotNil(t, cfg)

	address, keyName, algorithm, secret, err := cfg.GetAXFRCredentials(DefaultViewName, "example.com")
	require.ErrorContains(t, err, "allow-transfer port 853 does not match any listen-on setting")
	require.Empty(t, address)
	require.Empty(t, keyName)
	require.Empty(t, algorithm)
	require.Empty(t, secret)
}

func TestGetAXFRCredentialsForDefaultViewListenOnAllowTransferZoneOverrideListenerMismatch(t *testing.T) {
	config := `
		zone "example.com" {
			allow-transfer port 854 { any; };
		};
	`
	cfg, err := NewParser().Parse("", "", strings.NewReader(config))
	require.NoError(t, err)
	require.NotNil(t, cfg)

	address, keyName, algorithm, secret, err := cfg.GetAXFRCredentials(DefaultViewName, "example.com")
	require.ErrorContains(t, err, "allow-transfer port 854 does not match any listen-on setting")
	require.Empty(t, address)
	require.Empty(t, keyName)
	require.Empty(t, algorithm)
	require.Empty(t, secret)
}

func TestGetAXFRCredentialsForDefaultViewListenOnPermissiveAllowTransferNegatedListener(t *testing.T) {
	config := `
		options {
			listen-on port 53 { !127.0.0.1; 192.0.2.1; };
		};
		zone "example.com" {
			allow-transfer { 192.0.2.1; };
		};
	`
	cfg, err := NewParser().Parse("", "", strings.NewReader(config))
	require.NoError(t, err)
	require.NotNil(t, cfg)

	address, keyName, algorithm, secret, err := cfg.GetAXFRCredentials(DefaultViewName, "example.com")
	require.NoError(t, err)
	require.Equal(t, "192.0.2.1:53", address)
	require.Empty(t, keyName)
	require.Empty(t, algorithm)
	require.Empty(t, secret)
}

func TestGetAXFRCredentialsForDefaultViewNoAllowTransfer(t *testing.T) {
	config := `
		options {
			listen-on port 54 { 192.0.2.1; 127.0.0.1; };
			listen-on-v6 port 54 { 2001:db8:1::1; };
		};
	`
	cfg, err := NewParser().Parse("", "", strings.NewReader(config))
	require.NoError(t, err)
	require.NotNil(t, cfg)

	address, keyName, algorithm, secret, err := cfg.GetAXFRCredentials(DefaultViewName, "example.com")
	require.ErrorContains(t, err, "allow-transfer is disabled")
	require.Empty(t, address)
	require.Empty(t, keyName)
	require.Empty(t, algorithm)
	require.Empty(t, secret)
}

// Test checking whether or not a zone is a RPZ.
func TestIsRPZDefaultView(t *testing.T) {
	config := `options {
		response-policy {
			zone "rpz.example.com";
			zone "db.local";
		};
	};
	view "trusted" {
		response-policy {
			zone "rpz.example.org";
		};
	};`
	cfg, err := NewParser().Parse("", "", strings.NewReader(config))
	require.NoError(t, err)
	require.NotNil(t, cfg)

	t.Run("default view", func(t *testing.T) {
		require.True(t, cfg.IsRPZ(DefaultViewName, "RPZ.EXAMPLE.COM"))
		require.True(t, cfg.IsRPZ(DefaultViewName, "db.local"))
		require.False(t, cfg.IsRPZ(DefaultViewName, "example.com"))
		require.False(t, cfg.IsRPZ(DefaultViewName, "rpz.example.org"))
	})

	t.Run("trusted view", func(t *testing.T) {
		require.False(t, cfg.IsRPZ("trusted", "rpz.example.com"))
		require.False(t, cfg.IsRPZ("trusted", "db.local"))
		require.False(t, cfg.IsRPZ("trusted", "example.com"))
		require.True(t, cfg.IsRPZ("trusted", "rpz.example.org"))
	})
}

// Tests that GetAlgorithmSecret returns parsed algorithm and secret.
func TestGetAlgorithmSecret(t *testing.T) {
	key := Key{
		Name: "test-key",
		Clauses: []*KeyClause{
			{
				Algorithm: "hmac-sha256",
				Secret:    "VO6xA4Tc1PWYaqMuPaf6wfkITb+c9/mkzlEaWJavejU=",
			},
		},
	}
	algorithm, secret, err := key.GetAlgorithmSecret()
	require.NoError(t, err)
	require.Equal(t, "hmac-sha256", algorithm)
	require.Equal(t, "VO6xA4Tc1PWYaqMuPaf6wfkITb+c9/mkzlEaWJavejU=", secret)
}

// Tests that GetAlgorithmSecret emits an error when no algorithm is found in the key.
func TestGetAlgorithmSecretNoAlgorithm(t *testing.T) {
	key := Key{
		Name: "test-key",
		Clauses: []*KeyClause{
			{
				Secret: "VO6xA4Tc1PWYaqMuPaf6wfkITb+c9/mkzlEaWJavejU=",
			},
		},
	}
	_, _, err := key.GetAlgorithmSecret()
	require.ErrorContains(t, err, "no algorithm or secret found in key test-key")
}

// Tests that GetAlgorithmSecret emits an error when no secret is found in the key.
func TestGetAlgorithmSecretNoSecret(t *testing.T) {
	key := Key{
		Name: "test-key",
		Clauses: []*KeyClause{
			{
				Algorithm: "hmac-sha256",
			},
		},
	}
	_, _, err := key.GetAlgorithmSecret()
	require.ErrorContains(t, err, "no algorithm or secret found in key test-key")
}

// Tests that the API key is empty.
func TestGetAPIKey(t *testing.T) {
	cfg, err := NewParser().ParseFile("testdata/dir/named.conf", "")
	require.NoError(t, err)
	require.NotNil(t, cfg)
	require.Empty(t, cfg.GetAPIKey())
}

// Test getting the rndc credentials when the controls statement is present.
func TestGetRndcCredentials(t *testing.T) {
	config := `
		key "rndc-key" {
			algorithm hmac-sha256;
			secret "iCQvHPqq43AvFK/xRHaKrUiq4GPaFyBpvt/GwKSvKwM=";
		};
		controls {
			inet 192.0.2.1 port 953 allow { 192.0.2.0/24; } keys { "rndc-key"; };
		};
	`
	cfg, err := NewParser().Parse("", "", strings.NewReader(config))
	require.NoError(t, err)
	require.NotNil(t, cfg)

	address, port, key, enabled, err := cfg.GetRndcConnParams(nil)
	require.True(t, enabled)
	require.NoError(t, err)
	require.NotNil(t, address)
	require.Equal(t, "192.0.2.1", *address)
	require.NotNil(t, port)
	require.Equal(t, int64(953), *port)
	require.NotNil(t, key)
	require.Equal(t, "rndc-key", key.Name)
	algorithm, secret, err := key.GetAlgorithmSecret()
	require.NoError(t, err)
	require.Equal(t, "hmac-sha256", algorithm)
	require.Equal(t, "iCQvHPqq43AvFK/xRHaKrUiq4GPaFyBpvt/GwKSvKwM=", secret)
}

// Test getting the rndc credentials when the key is not specified in the configuration
// file.
func TestGetRndcCredentialsNoKey(t *testing.T) {
	config := `
		controls {
			inet 192.0.2.1 port 953 allow { 192.0.2.0/24; } keys { "rndc-key"; };
		};
	`
	cfg, err := NewParser().Parse("", "", strings.NewReader(config))
	require.NoError(t, err)
	require.NotNil(t, cfg)

	address, port, key, enabled, err := cfg.GetRndcConnParams(nil)
	require.True(t, enabled)
	require.NoError(t, err)
	require.Equal(t, "192.0.2.1", *address)
	require.Equal(t, int64(953), *port)
	require.Nil(t, key)
}

// Test that rndc control channel is assumed to be disabled when the
// controls statement does not contain an inet clause.
func TestGetRndcCredentialsNoInetClause(t *testing.T) {
	config := `
		controls {
			unix "/var/run/rndc.sock" perm 0666 owner 0 group 0 keys { "rndc-key"; };
		};
	`
	cfg, err := NewParser().Parse("", "", strings.NewReader(config))
	require.NoError(t, err)
	require.NotNil(t, cfg)

	address, port, key, enabled, err := cfg.GetRndcConnParams(nil)
	require.False(t, enabled)
	require.Nil(t, address)
	require.Nil(t, port)
	require.Nil(t, key)
	require.NoError(t, err)
}

// Test that rndc control channel is assumed to be disabled when the
// controls statement is empty.
func TestGetRndcCredentialsEmptyControls(t *testing.T) {
	config := `
		controls { };
	`
	cfg, err := NewParser().Parse("", "", strings.NewReader(config))
	require.NoError(t, err)
	require.NotNil(t, cfg)
	address, port, key, enabled, err := cfg.GetRndcConnParams(nil)
	require.False(t, enabled)
	require.Nil(t, address)
	require.Nil(t, port)
	require.Nil(t, key)
	require.NoError(t, err)
}

// Test that default credentials are assumed for the rndc control channel
// when the controls statement is not present.
func TestGetRndcCredentialsNoControls(t *testing.T) {
	config := `
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
		};`
	cfg, err := NewParser().Parse("", "", strings.NewReader(config))
	require.NoError(t, err)
	require.NotNil(t, cfg)
	address, port, key, enabled, err := cfg.GetRndcConnParams(nil)
	require.NoError(t, err)
	require.True(t, enabled)
	require.Equal(t, "127.0.0.1", *address)
	require.EqualValues(t, 953, *port)
	require.Nil(t, key)
}

// Test various cases of getting the rndc credentials when the keys
// are specified in the config file or rndc.key file.
func TestGetRndcCredentialsNoControlsRndcConfig(t *testing.T) {
	testCases := []struct {
		name                  string
		config                string
		expectedRndcKey       string
		expectedRndcKeySecret string
	}{
		// The params should include the default controls configuration and use the
		// rndc-key key defined in the external file (typically rndc.key).
		{
			name:                  "no controls",
			config:                ``,
			expectedRndcKey:       "rndc-key",
			expectedRndcKeySecret: "iCQvHPqq43AvFK/xRHaKrUiq4GPaFyBpvt/GwKSvKwM=",
		},
		// The params should include the configured controls and the rndc-key key
		// defined in the external file (typically rndc.key).
		{
			name: "controls with rndc key",
			config: `
				controls {
					inet 127.0.0.1 port 953 allow { 127.0.0.1; } keys { "rndc-key"; };
				};
			`,
			expectedRndcKey:       "rndc-key",
			expectedRndcKeySecret: "iCQvHPqq43AvFK/xRHaKrUiq4GPaFyBpvt/GwKSvKwM=",
		},
		// The params should include the configured controls and the rndc-key-in-config
		// key defined in the config file, as this key is directly referenced in the
		// inet clause.
		{
			name: "controls with rndc key in config file",
			config: `
				key "rndc-key-in-config" {
					algorithm hmac-sha256;
					secret "VO6xA4Tc1PWYaqMuPaf6wfkITb+c9/mkzlEaWJavejU=";
				};
				controls {
					inet 127.0.0.1 port 953 allow { 127.0.0.1; } keys { "rndc-key-in-config"; };
				};
			`,
			expectedRndcKey:       "rndc-key-in-config",
			expectedRndcKeySecret: "VO6xA4Tc1PWYaqMuPaf6wfkITb+c9/mkzlEaWJavejU=",
		},
		// The params should include the rndc-key key defined in the external file
		// (typically rndc.key) even though the key under the same name is defined
		// in the config file.
		{
			name: "controls with rndc-key in rndc.key and config file",
			config: `
				key "rndc-key" {
					algorithm hmac-sha256;
					secret "6L8DwXFboA7FDQJQP051hjFV/n9B3IR/SwDLX7y5czE=";
				};
				controls {
					inet 127.0.0.1 port 953 allow { 127.0.0.1; } keys { "rndc-key"; };
				};
			`,
			expectedRndcKey:       "rndc-key",
			expectedRndcKeySecret: "iCQvHPqq43AvFK/xRHaKrUiq4GPaFyBpvt/GwKSvKwM=",
		},
	}
	rndcConfig := `
		key "rndc-key" {
			algorithm hmac-sha256;
			secret "iCQvHPqq43AvFK/xRHaKrUiq4GPaFyBpvt/GwKSvKwM=";
		};
	`

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			rndcCfg, err := NewParser().Parse("", "", strings.NewReader(rndcConfig))
			require.NoError(t, err)
			require.NotNil(t, rndcCfg)

			cfg, err := NewParser().Parse("", "", strings.NewReader(testCase.config))
			require.NoError(t, err)
			require.NotNil(t, cfg)

			address, port, key, enabled, err := cfg.GetRndcConnParams(rndcCfg)
			require.True(t, enabled)
			require.NoError(t, err)
			require.Equal(t, "127.0.0.1", *address)
			require.EqualValues(t, 953, *port)
			require.NotNil(t, key)
			require.Equal(t, testCase.expectedRndcKey, key.Name)
			algorithm, secret, err := key.GetAlgorithmSecret()
			require.NoError(t, err)
			require.Equal(t, "hmac-sha256", algorithm)
			require.Equal(t, testCase.expectedRndcKeySecret, secret)
		})
	}
}

// Test parsing statistics-channels statement and returning the address and port.
func TestGetStatisticsChannelCredentials(t *testing.T) {
	config := `
		statistics-channels {
			inet 192.0.2.1 port 80 allow { 192.0.2.0/24; };
		};
	`
	cfg, err := NewParser().Parse("", "", strings.NewReader(config))
	require.NoError(t, err)
	require.NotNil(t, cfg)

	address, port, enabled := cfg.GetStatisticsChannelConnParams()
	require.True(t, enabled)
	require.NoError(t, err)
	require.NotNil(t, address)
	require.Equal(t, "192.0.2.1", *address)
	require.NotNil(t, port)
	require.Equal(t, int64(80), *port)
}

// Test that statistics-channels are not enabled when there is no inet clause.
func TestGetStatisticsChannelCredentialsNoInetClause(t *testing.T) {
	config := `
		statistics-channels {};
	`
	cfg, err := NewParser().Parse("", "", strings.NewReader(config))
	require.NoError(t, err)
	require.NotNil(t, cfg)

	address, port, enabled := cfg.GetStatisticsChannelConnParams()
	require.False(t, enabled)
	require.Nil(t, address)
	require.Nil(t, port)
	require.NoError(t, err)
}

// Test that statistics-channels are not enabled when there is no statistics-channels statement.
func TestGetStatisticsChannelCredentialsNoStatisticsChannels(t *testing.T) {
	config := ``
	cfg, err := NewParser().Parse("", "", strings.NewReader(config))
	require.NoError(t, err)
	require.NotNil(t, cfg)

	address, port, enabled := cfg.GetStatisticsChannelConnParams()
	require.False(t, enabled)
	require.Nil(t, address)
	require.Nil(t, port)
	require.NoError(t, err)
}

// Test various combinations of absolute and relative paths for source config file,
// included file, and cyclic include file, both in chroot and non-chroot
// environments. The config should be expanded correctly in all cases without
// errors.
func TestConfigExpand(t *testing.T) {
	type testCase struct {
		isSourcePathAbs  bool
		isIncludePathAbs bool
		isCyclePathAbs   bool
		isChroot         bool
	}

	// String representation of the test case for easier identification in case
	// of failure.
	stringifyTestCase := func(tc testCase) string {
		return fmt.Sprintf("SourcePathAbs=%t, IncludePathAbs=%t, CyclePathAbs=%t, IsChroot=%t",
			tc.isSourcePathAbs, tc.isIncludePathAbs, tc.isCyclePathAbs, tc.isChroot)
	}

	// Generate test case based on the integer. There are 4 boolean parameters,
	// so we can represent each combination as a number from 0 to 15, where each
	// bit represents one of the boolean parameters.
	newTestCase := func(seed int) testCase {
		return testCase{
			isSourcePathAbs:  seed&1 == 0,
			isIncludePathAbs: seed&2 == 0,
			isCyclePathAbs:   seed&4 == 0,
			isChroot:         seed&8 == 0,
		}
	}

	for i := range 16 {
		tc := newTestCase(i)

		t.Run(stringifyTestCase(tc), func(t *testing.T) {
			// Arrange
			sb := testutil.NewSandbox()
			defer sb.Close()

			sourcePathInSandbox, _ := sb.Join("dir/source.file")
			sourcePathAbs := sourcePathInSandbox
			sourcePathRel := "dir/source.file"
			sourcePathRelToItsDir := "source.file"

			includePathAbs, _ := sb.Write("dir/include.file", "controls { };")
			includePathRel := "include.file"

			config := Config{Statements: []*Statement{
				// Standard include statement to be expanded.
				{Include: &Include{}},
				// Cyclic include to test that cycles are detected.
				{Include: &Include{}},
			}}

			if tc.isChroot {
				config.rootPrefix = sb.BasePath
				sourcePathAbs = "/dir/source.file"
				includePathAbs = "/dir/include.file"
			}

			if tc.isSourcePathAbs {
				config.sourcePath = sourcePathAbs
			} else {
				config.sourcePath = sourcePathRel
				// We were able to read the config file by a relative path, it
				// means that the parser is running in the directory where
				// the source file is located. Change the current working
				// directory to the sandbox base path to simulate this.
				cwd, _ := os.Getwd()
				defer os.Chdir(cwd)
				os.Chdir(sb.BasePath)
			}

			if tc.isIncludePathAbs {
				config.Statements[0].Include = &Include{Path: includePathAbs}
			} else {
				config.Statements[0].Include = &Include{Path: includePathRel}
			}

			if tc.isCyclePathAbs {
				config.Statements[1].Include = &Include{Path: sourcePathAbs}
			} else {
				config.Statements[1].Include = &Include{Path: sourcePathRelToItsDir}
			}
			os.WriteFile(
				sourcePathInSandbox,
				[]byte(fmt.Sprintf(`include "%s";`, config.Statements[1].Include.Path)),
				0644,
			)

			// Act
			expanded, err := config.Expand()

			// Assert
			require.NoError(t, err)
			require.Len(t, expanded.Statements, 2)
			require.NotNil(t, expanded.Statements[0].Controls)
			require.NotNil(t, expanded.Statements[1].Include)
			// The include path should remain unchanged.
			require.Equal(t, config.Statements[1].Include, expanded.Statements[1].Include)
		})
	}
}
