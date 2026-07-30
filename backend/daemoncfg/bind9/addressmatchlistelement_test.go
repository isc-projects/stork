package bind9config

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// Test checking that the address match list element is formatted correctly.
func TestAddressMatchListElementFormatAddressMatchList(t *testing.T) {
	amle := &AddressMatchListElement{
		Negation: true,
		AddressMatchList: &AddressMatchList{
			Elements: []*AddressMatchListElement{
				{
					Negation: true,
					AddressMatchList: &AddressMatchList{
						Elements: []*AddressMatchListElement{
							{
								IPAddressOrACLName: "1.1.1.1",
							},
						},
					},
				},
				{
					IPAddressOrACLName: "1.1.1.2",
				},
			},
		},
	}
	output := amle.getFormattedOutput(nil)
	require.NotNil(t, output)
	requireConfigEq(t, `! {
		! {
			"1.1.1.1";
		};
		"1.1.1.2";
	};`, output)
}

// Test that the key match list element is formatted correctly.
func TestAddressMatchListElementFormatKey(t *testing.T) {
	amle := &AddressMatchListElement{
		Negation: false,
		KeyID:    "test-key",
	}
	output := amle.getFormattedOutput(nil)
	require.NotNil(t, output)
	builder := newFormatterStringBuilder()
	err := output.write(0, false, builder)
	require.NoError(t, err)
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
	builder := newFormatterStringBuilder()
	err := output.write(0, false, builder)
	require.NoError(t, err)
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
	builder := newFormatterStringBuilder()
	err := output.write(0, false, builder)
	require.NoError(t, err)
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
	builder := newFormatterStringBuilder()
	err := output.write(0, false, builder)
	require.NoError(t, err)
	require.Equal(t, `! "1.1.1.1";`, builder.getString())
}

// Test that serializing an address match list element with nil values does not panic.
func TestAddressMatchListElementFormatNilValues(t *testing.T) {
	amle := &AddressMatchListElement{}
	require.NotPanics(t, func() { amle.getFormattedOutput(nil) })
}
