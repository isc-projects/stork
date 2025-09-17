package bind9config

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// Test checking that the ACL match list element is formatted correctly.
func TestAddressMatchListElementFormatACL(t *testing.T) {
	amle := &AddressMatchListElement{
		Negation: false,
		ACL: &ACL{
			Name: "test-acl",
		},
		KeyID:              "test-key",
		IPAddressOrACLName: "test-ip",
	}
	output := amle.getFormattedOutput(nil)
	require.NotNil(t, output)
	builder := newFormatterBuilder()
	output.write(0, false, builder)
	require.Equal(t, `"test-acl";`, builder.getString())
}

// Test checking that the negated ACL match list element is formatted correctly.
func TestAddressMatchListElementFormatACLNegation(t *testing.T) {
	amle := &AddressMatchListElement{
		Negation: true,
		ACL: &ACL{
			Name: "test-acl",
		},
	}
	output := amle.getFormattedOutput(nil)
	require.NotNil(t, output)
	builder := newFormatterBuilder()
	output.write(0, false, builder)
	require.Equal(t, `! "test-acl";`, builder.getString())
}

// Test that the key match list element is formatted correctly.
func TestAddressMatchListElementFormatKey(t *testing.T) {
	amle := &AddressMatchListElement{
		Negation: false,
		KeyID:    "test-key",
	}
	output := amle.getFormattedOutput(nil)
	require.NotNil(t, output)
	builder := newFormatterBuilder()
	output.write(0, false, builder)
	require.Equal(t, `key "test-key";`, builder.getString())
}

// Test that the negated key match list element is formatted correctly.
func TestAddressMatchListElementFormatKeyNegation(t *testing.T) {
	amle := &AddressMatchListElement{
		Negation: true,
		KeyID:    "test-key",
	}
	output := amle.getFormattedOutput(nil)
	require.NotNil(t, output)
	builder := newFormatterBuilder()
	output.write(0, false, builder)
	require.Equal(t, `! key "test-key";`, builder.getString())
}

// Test that the IP address match list element is formatted correctly.
func TestAddressMatchListElementFormatIPAddress(t *testing.T) {
	amle := &AddressMatchListElement{
		Negation:           false,
		IPAddressOrACLName: "1.1.1.1",
	}
	output := amle.getFormattedOutput(nil)
	require.NotNil(t, output)
	builder := newFormatterBuilder()
	output.write(0, false, builder)
	require.Equal(t, `"1.1.1.1";`, builder.getString())
}

// Test that the negated IP address match list element is formatted correctly.
func TestAddressMatchListElementFormatIPAddressNegation(t *testing.T) {
	amle := &AddressMatchListElement{
		Negation:           true,
		IPAddressOrACLName: "1.1.1.1",
	}
	output := amle.getFormattedOutput(nil)
	require.NotNil(t, output)
	builder := newFormatterBuilder()
	output.write(0, false, builder)
	require.Equal(t, `! "1.1.1.1";`, builder.getString())
}

// Test that serializing an address match list element with nil values does not panic.
func TestAddressMatchListElementFormatNilValues(t *testing.T) {
	amle := &AddressMatchListElement{}
	require.NotPanics(t, func() { amle.getFormattedOutput(nil) })
}
