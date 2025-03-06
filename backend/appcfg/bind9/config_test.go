package bind9config

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// Tests that GetView returns expected view.
func TestGetView(t *testing.T) {
	cfg, err := ParseFile("testdata/named.conf")
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
	cfg, err := ParseFile("testdata/named.conf")
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

// Tests that GetACL returns expected ACL.
func TestGetACL(t *testing.T) {
	cfg, err := ParseFile("testdata/named.conf")
	require.NoError(t, err)
	require.NotNil(t, cfg)

	cfg, err = cfg.Expand("testdata")
	require.NoError(t, err)
	require.NotNil(t, cfg)

	acl := cfg.GetACL("trusted-networks")
	require.NotNil(t, acl)
	require.Equal(t, "trusted-networks", acl.Name)
	require.Len(t, acl.AdressMatchList.Elements, 8)

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
	cfg, err := Parse("", strings.NewReader(config))
	require.NoError(t, err)
	require.NotNil(t, cfg)

	cfg, err = cfg.Expand("testdata")
	require.NoError(t, err)
	require.NotNil(t, cfg)

	key, err := cfg.GetViewKey("trusted")
	require.ErrorContains(t, err, "too much recursion in address-match-list")
	require.Nil(t, key)
}

// Tests that GetViewKey returns associated keys.
func TestGetViewKey(t *testing.T) {
	cfg, err := ParseFile("testdata/named.conf")
	require.NoError(t, err)
	require.NotNil(t, cfg)

	cfg, err = cfg.Expand("testdata")
	require.NoError(t, err)
	require.NotNil(t, cfg)

	key, err := cfg.GetViewKey("trusted")
	require.NoError(t, err)
	require.NotNil(t, key)
	require.Equal(t, "trusted-key", key.Name)
	algorithm, secret, err := key.GetAlgorithmSecret()
	require.NoError(t, err)
	require.Equal(t, "hmac-sha256", algorithm)
	require.Equal(t, "VO6xA4Tc1PWYaqMuPaf6wfkITb+c9/mkzlEaWJavejU=", secret)

	key, err = cfg.GetViewKey("guest")
	require.NoError(t, err)
	require.NotNil(t, key)
	require.Equal(t, "guest-key", key.Name)
	algorithm, secret, err = key.GetAlgorithmSecret()
	require.NoError(t, err)
	require.Equal(t, "hmac-sha256", algorithm)
	require.Equal(t, "6L8DwXFboA7FDQJQP051hjFV/n9B3IR/SwDLX7y5czE=", secret)

	key, err = cfg.GetViewKey("non-existent")
	require.NoError(t, err)
	require.Nil(t, key)
}

// Tests that IsMatchExpected returns true when the element does not
// contain negation, false otherwise.
func TestIsMatchExpected(t *testing.T) {
	element := AddressMatchListElement{
		Negation:  false,
		IPAddress: "192.168.1.1",
	}
	require.True(t, element.IsMatchExpected())

	element.Negation = true
	require.False(t, element.IsMatchExpected())
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
