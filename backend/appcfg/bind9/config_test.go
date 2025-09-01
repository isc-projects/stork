package bind9config

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// Test checking if the configuration contains a no-parse directives.
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

// Test checking if the configuration does not contain a no-parse directives.
func TestConfigHasNoParseNone(t *testing.T) {
	cfg := &Config{}
	require.False(t, cfg.HasNoParse())
}

// Tests that GetView returns expected view.
func TestGetView(t *testing.T) {
	cfg, err := NewParser().ParseFile("testdata/named.conf")
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
	cfg, err := NewParser().ParseFile("testdata/named.conf")
	require.NoError(t, err)
	require.NotNil(t, cfg)

	key := cfg.GetKey("trusted-key")
	require.NotNil(t, key)
	require.Equal(t, "trusted-key", key.Name)
	algorithm, secret, err := key.GetAlgorithmSecret()
	require.NoError(t, err)
	require.NotNil(t, algorithm)
	require.NotNil(t, secret)
	require.Equal(t, "hmac-sha256", *algorithm)
	require.Equal(t, "VO6xA4Tc1PWYaqMuPaf6wfkITb+c9/mkzlEaWJavejU=", *secret)

	key = cfg.GetKey("non-existent")
	require.Nil(t, key)
}

// Tests that GetACL returns expected ACL.
func TestGetACL(t *testing.T) {
	cfg, err := NewParser().ParseFile("testdata/named.conf")
	require.NoError(t, err)
	require.NotNil(t, cfg)

	cfg, err = cfg.Expand("testdata")
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
	cfg, err := NewParser().Parse("", strings.NewReader(config))
	require.NoError(t, err)
	require.NotNil(t, cfg)

	cfg, err = cfg.Expand("testdata")
	require.NoError(t, err)
	require.NotNil(t, cfg)

	key, err := cfg.GetZoneKey("trusted", "example.com")
	require.ErrorContains(t, err, "too much recursion in address-match-list")
	require.Nil(t, key)
}

// Tests that GetViewKey returns associated keys.
func TestGetViewKey(t *testing.T) {
	cfg, err := NewParser().ParseFile("testdata/named.conf")
	require.NoError(t, err)
	require.NotNil(t, cfg)

	cfg, err = cfg.Expand("testdata")
	require.NoError(t, err)
	require.NotNil(t, cfg)

	key, err := cfg.GetZoneKey("trusted", "example.com")
	require.NoError(t, err)
	require.NotNil(t, key)
	require.Equal(t, "trusted-key", key.Name)
	algorithm, secret, err := key.GetAlgorithmSecret()
	require.NoError(t, err)
	require.NotNil(t, algorithm)
	require.NotNil(t, secret)
	require.Equal(t, "hmac-sha256", *algorithm)
	require.Equal(t, "VO6xA4Tc1PWYaqMuPaf6wfkITb+c9/mkzlEaWJavejU=", *secret)

	key, err = cfg.GetZoneKey("guest", "example.com")
	require.NoError(t, err)
	require.NotNil(t, key)
	require.Equal(t, "guest-key", key.Name)
	algorithm, secret, err = key.GetAlgorithmSecret()
	require.NoError(t, err)
	require.NotNil(t, algorithm)
	require.NotNil(t, secret)
	require.Equal(t, "hmac-sha256", *algorithm)
	require.Equal(t, "6L8DwXFboA7FDQJQP051hjFV/n9B3IR/SwDLX7y5czE=", *secret)

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
	cfg, err := NewParser().Parse("", strings.NewReader(config))
	require.NoError(t, err)
	require.NotNil(t, cfg)

	address, keyName, algorithm, secret, err := cfg.GetAXFRCredentials(DefaultViewName, "example.com")
	require.NoError(t, err)
	// Localhost is preferred. The port should be 853 as it matches both
	// the listener port and the allow-transfer port.
	require.Equal(t, "127.0.0.1:853", *address)
	require.NotNil(t, keyName)
	require.Equal(t, "trusted-key", *keyName)
	require.NotNil(t, algorithm)
	require.Equal(t, "hmac-sha256", *algorithm)
	require.NotNil(t, secret)
	require.Equal(t, "VO6xA4Tc1PWYaqMuPaf6wfkITb+c9/mkzlEaWJavejU=", *secret)
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
	cfg, err := NewParser().Parse("", strings.NewReader(config))
	require.NoError(t, err)
	require.NotNil(t, cfg)

	address, keyName, algorithm, secret, err := cfg.GetAXFRCredentials("trusted", "example.com")
	require.NoError(t, err)
	// The local loopback addresss from the second clause is preferred.
	require.Equal(t, "127.0.0.1:53", *address)
	require.NotNil(t, keyName)
	require.Equal(t, "trusted-key", *keyName)
	require.NotNil(t, algorithm)
	require.Equal(t, "hmac-sha256", *algorithm)
	require.NotNil(t, secret)
	require.Equal(t, "VO6xA4Tc1PWYaqMuPaf6wfkITb+c9/mkzlEaWJavejU=", *secret)
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
	cfg, err := NewParser().Parse("", strings.NewReader(config))
	require.NoError(t, err)
	require.NotNil(t, cfg)

	address, keyName, algorithm, secret, err := cfg.getAXFRCredentialsForView("trusted", "example.com")
	require.NoError(t, err)
	require.Equal(t, "[::1]:54", *address)
	require.NotNil(t, keyName)
	require.Equal(t, "trusted-key", *keyName)
	require.NotNil(t, algorithm)
	require.Equal(t, "hmac-sha256", *algorithm)
	require.NotNil(t, secret)
	require.Equal(t, "VO6xA4Tc1PWYaqMuPaf6wfkITb+c9/mkzlEaWJavejU=", *secret)
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
	cfg, err := NewParser().Parse("", strings.NewReader(config))
	require.NoError(t, err)
	require.NotNil(t, cfg)

	address, keyName, algorithm, secret, err := cfg.getAXFRCredentialsForView("trusted", "example.com")
	require.NoError(t, err)
	// If the port was not specified in the allow-transfer clause, the IPv4
	// loopback address would have been selected. However, since the port is
	// specified, we try to match it with the address listening on that port.
	require.Equal(t, "[::1]:853", *address)
	require.NotNil(t, keyName)
	require.Equal(t, "trusted-key", *keyName)
	require.NotNil(t, algorithm)
	require.Equal(t, "hmac-sha256", *algorithm)
	require.NotNil(t, secret)
	require.Equal(t, "VO6xA4Tc1PWYaqMuPaf6wfkITb+c9/mkzlEaWJavejU=", *secret)
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
	cfg, err := NewParser().Parse("", strings.NewReader(config))
	require.NoError(t, err)
	require.NotNil(t, cfg)

	address, keyName, algorithm, secret, err := cfg.getAXFRCredentialsForView("trusted", "example.com")
	require.NoError(t, err)
	require.Equal(t, "127.0.0.1:53", *address)
	require.NotNil(t, keyName)
	require.Equal(t, "trusted-key", *keyName)
	require.NotNil(t, algorithm)
	require.Equal(t, "hmac-sha256", *algorithm)
	require.NotNil(t, secret)
	require.Equal(t, "VO6xA4Tc1PWYaqMuPaf6wfkITb+c9/mkzlEaWJavejU=", *secret)
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
	cfg, err := NewParser().Parse("", strings.NewReader(config))
	require.NoError(t, err)
	require.NotNil(t, cfg)

	address, keyName, algorithm, secret, err := cfg.getAXFRCredentialsForView("trusted", "example.com")
	require.NoError(t, err)
	require.Equal(t, "[::1]:853", *address)
	require.NotNil(t, keyName)
	require.Equal(t, "trusted-key", *keyName)
	require.NotNil(t, algorithm)
	require.Equal(t, "hmac-sha256", *algorithm)
	require.NotNil(t, secret)
	require.Equal(t, "VO6xA4Tc1PWYaqMuPaf6wfkITb+c9/mkzlEaWJavejU=", *secret)
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
	cfg, err := NewParser().Parse("", strings.NewReader(config))
	require.NoError(t, err)
	require.NotNil(t, cfg)

	address, keyName, algorithm, secret, err := cfg.getAXFRCredentialsForView("trusted", "example.com")
	require.ErrorContains(t, err, "allow-transfer port 853 does not match any listen-on setting")
	require.Nil(t, address)
	require.Nil(t, keyName)
	require.Nil(t, algorithm)
	require.Nil(t, secret)
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
	cfg, err := NewParser().Parse("", strings.NewReader(config))
	require.NoError(t, err)
	require.NotNil(t, cfg)

	address, keyName, algorithm, secret, err := cfg.getAXFRCredentialsForView("trusted", "example.com")
	require.ErrorContains(t, err, "allow-transfer port 853 does not match any listen-on setting")
	require.Nil(t, address)
	require.Nil(t, keyName)
	require.Nil(t, algorithm)
	require.Nil(t, secret)
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
	cfg, err := NewParser().Parse("", strings.NewReader(config))
	require.NoError(t, err)
	require.NotNil(t, cfg)

	address, keyName, algorithm, secret, err := cfg.getAXFRCredentialsForView("trusted", "example.com")
	require.ErrorContains(t, err, "allow-transfer is disabled")
	require.Nil(t, address)
	require.Nil(t, keyName)
	require.Nil(t, algorithm)
	require.Nil(t, secret)
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
	cfg, err := NewParser().Parse("", strings.NewReader(config))
	require.NoError(t, err)
	require.NotNil(t, cfg)

	address, keyName, algorithm, secret, err := cfg.GetAXFRCredentials("trusted", "example.com")
	require.NoError(t, err)
	require.NotNil(t, address)
	require.Equal(t, "[::1]:54", *address)
	require.NotNil(t, keyName)
	require.Equal(t, "trusted-key", *keyName)
	require.NotNil(t, algorithm)
	require.Equal(t, "hmac-sha256", *algorithm)
	require.NotNil(t, secret)
	require.Equal(t, "VO6xA4Tc1PWYaqMuPaf6wfkITb+c9/mkzlEaWJavejU=", *secret)
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
	cfg, err := NewParser().Parse("", strings.NewReader(config))
	require.NoError(t, err)
	require.NotNil(t, cfg)

	address, keyName, algorithm, secret, err := cfg.GetAXFRCredentials(DefaultViewName, "example.com")
	require.NoError(t, err)
	// Localhost is preferred. The port should be 853 as it matches both
	// the listener port and the allow-transfer port.
	require.Equal(t, "127.0.0.1:853", *address)
	require.NotNil(t, keyName)
	require.Equal(t, "trusted-key", *keyName)
	require.NotNil(t, algorithm)
	require.Equal(t, "hmac-sha256", *algorithm)
	require.NotNil(t, secret)
	require.Equal(t, "VO6xA4Tc1PWYaqMuPaf6wfkITb+c9/mkzlEaWJavejU=", *secret)
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
	cfg, err := NewParser().Parse("", strings.NewReader(config))
	require.NoError(t, err)
	require.NotNil(t, cfg)

	address, keyName, algorithm, secret, err := cfg.GetAXFRCredentials(DefaultViewName, "example.com")
	require.NoError(t, err)
	// The local loopback addresss from the second clause is preferred.
	require.Equal(t, "127.0.0.1:53", *address)
	require.NotNil(t, keyName)
	require.Equal(t, "trusted-key", *keyName)
	require.NotNil(t, algorithm)
	require.Equal(t, "hmac-sha256", *algorithm)
	require.NotNil(t, secret)
	require.Equal(t, "VO6xA4Tc1PWYaqMuPaf6wfkITb+c9/mkzlEaWJavejU=", *secret)
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
	cfg, err := NewParser().Parse("", strings.NewReader(config))
	require.NoError(t, err)
	require.NotNil(t, cfg)

	address, keyName, algorithm, secret, err := cfg.GetAXFRCredentials(DefaultViewName, "example.com")
	require.NoError(t, err)
	require.Equal(t, "[::1]:54", *address)
	require.NotNil(t, keyName)
	require.Equal(t, "trusted-key", *keyName)
	require.NotNil(t, algorithm)
	require.Equal(t, "hmac-sha256", *algorithm)
	require.NotNil(t, secret)
	require.Equal(t, "VO6xA4Tc1PWYaqMuPaf6wfkITb+c9/mkzlEaWJavejU=", *secret)
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
	cfg, err := NewParser().Parse("", strings.NewReader(config))
	require.NoError(t, err)
	require.NotNil(t, cfg)

	address, keyName, algorithm, secret, err := cfg.GetAXFRCredentials(DefaultViewName, "example.com")
	require.NoError(t, err)
	// If the port was not specified in the allow-transfer clause, the IPv4
	// loopback address would have been selected. However, since the port is
	// specified, we try to match it with the address listening on that port.
	require.Equal(t, "[::1]:853", *address)
	require.NotNil(t, keyName)
	require.Equal(t, "trusted-key", *keyName)
	require.NotNil(t, algorithm)
	require.Equal(t, "hmac-sha256", *algorithm)
	require.NotNil(t, secret)
	require.Equal(t, "VO6xA4Tc1PWYaqMuPaf6wfkITb+c9/mkzlEaWJavejU=", *secret)
}

func TestGetAXFRCredentialsForDefaultViewListenOnAllowTransferAny(t *testing.T) {
	config := `
		options {
			allow-transfer { any; };
		};
	`
	cfg, err := NewParser().Parse("", strings.NewReader(config))
	require.NoError(t, err)
	require.NotNil(t, cfg)

	address, keyName, algorithm, secret, err := cfg.GetAXFRCredentials(DefaultViewName, "example.com")
	require.NoError(t, err)
	require.Equal(t, "127.0.0.1:53", *address)
	require.Nil(t, keyName)
	require.Nil(t, algorithm)
	require.Nil(t, secret)
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
	cfg, err := NewParser().Parse("", strings.NewReader(config))
	require.NoError(t, err)
	require.NotNil(t, cfg)

	address, keyName, algorithm, secret, err := cfg.GetAXFRCredentials(DefaultViewName, "example.com")
	require.NoError(t, err)
	require.Equal(t, "[::1]:853", *address)
	require.Nil(t, keyName)
	require.Nil(t, algorithm)
	require.Nil(t, secret)
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
	cfg, err := NewParser().Parse("", strings.NewReader(config))
	require.NoError(t, err)
	require.NotNil(t, cfg)

	address, keyName, algorithm, secret, err := cfg.GetAXFRCredentials(DefaultViewName, "example.com")
	require.ErrorContains(t, err, "allow-transfer port 853 does not match any listen-on setting")
	require.Nil(t, address)
	require.Nil(t, keyName)
	require.Nil(t, algorithm)
	require.Nil(t, secret)
}

func TestGetAXFRCredentialsForDefaultViewListenOnAllowTransferZoneOverrideListenerMismatch(t *testing.T) {
	config := `
		zone "example.com" {
			allow-transfer port 854 { any; };
		};
	`
	cfg, err := NewParser().Parse("", strings.NewReader(config))
	require.NoError(t, err)
	require.NotNil(t, cfg)

	address, keyName, algorithm, secret, err := cfg.GetAXFRCredentials(DefaultViewName, "example.com")
	require.ErrorContains(t, err, "allow-transfer port 854 does not match any listen-on setting")
	require.Nil(t, address)
	require.Nil(t, keyName)
	require.Nil(t, algorithm)
	require.Nil(t, secret)
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
	cfg, err := NewParser().Parse("", strings.NewReader(config))
	require.NoError(t, err)
	require.NotNil(t, cfg)

	address, keyName, algorithm, secret, err := cfg.GetAXFRCredentials(DefaultViewName, "example.com")
	require.NoError(t, err)
	require.NotNil(t, address)
	require.Equal(t, "192.0.2.1:53", *address)
	require.Nil(t, keyName)
	require.Nil(t, algorithm)
	require.Nil(t, secret)
}

func TestGetAXFRCredentialsForDefaultViewNoAllowTransfer(t *testing.T) {
	config := `
		options {
			listen-on port 54 { 192.0.2.1; 127.0.0.1; };
			listen-on-v6 port 54 { 2001:db8:1::1; };
		};
	`
	cfg, err := NewParser().Parse("", strings.NewReader(config))
	require.NoError(t, err)
	require.NotNil(t, cfg)

	address, keyName, algorithm, secret, err := cfg.GetAXFRCredentials(DefaultViewName, "example.com")
	require.ErrorContains(t, err, "allow-transfer is disabled")
	require.Nil(t, address)
	require.Nil(t, keyName)
	require.Nil(t, algorithm)
	require.Nil(t, secret)
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
	cfg, err := NewParser().Parse("", strings.NewReader(config))
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
	require.NotNil(t, algorithm)
	require.NotNil(t, secret)
	require.Equal(t, "hmac-sha256", *algorithm)
	require.Equal(t, "VO6xA4Tc1PWYaqMuPaf6wfkITb+c9/mkzlEaWJavejU=", *secret)
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
	cfg, err := NewParser().ParseFile("testdata/named.conf")
	require.NoError(t, err)
	require.NotNil(t, cfg)
	require.Empty(t, cfg.GetAPIKey())
}
