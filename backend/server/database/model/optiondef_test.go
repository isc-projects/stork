package dbmodel

import (
	"testing"

	"github.com/stretchr/testify/require"
	storkutil "isc.org/stork/util"
)

// Test that the option definition lookup can identify the standard options
// DHCPv4 options for which Kea should know their definitions.
func TestStandardDHCPv4OptionDefinitionExists(t *testing.T) {
	lookup := NewDHCPOptionDefinitionLookup()

	existingCodes := []uint16{99, 108, 175, 210, 220}
	for _, code := range existingCodes {
		option := DHCPOption{
			Code:     code,
			Space:    "dhcp4",
			Universe: storkutil.IPv4,
		}
		require.True(t, lookup.DefinitionExists(1, option))
	}
}

// Test that the option definition lookup indicates that the DHCPv4
// suboption does not have a definition.
func TestDHCPv4SuboptionDefinition(t *testing.T) {
	lookup := NewDHCPOptionDefinitionLookup()

	option := DHCPOption{
		Code:     15,
		Space:    "foo",
		Universe: storkutil.IPv4,
	}
	require.False(t, lookup.DefinitionExists(1, option))
}

// Test that the option definition lookup flags the standard options
// for which the definitions do not exist.
func TestStandardDHCPv4OptionDefinitionNotExists(t *testing.T) {
	lookup := NewDHCPOptionDefinitionLookup()

	nonExistingCodes := []uint16{0, 106, 165, 180, 215, 224}
	for _, code := range nonExistingCodes {
		option := DHCPOption{
			Code:     code,
			Space:    "dhcp4",
			Universe: storkutil.IPv4,
		}
		require.False(t, lookup.DefinitionExists(1, option))
	}
}

// Test that the option definition lookup can identify the standard options
// DHCPv6 options for which Kea should know their definitions.
func TestStandardDHCPv6OptionDefinitionExists(t *testing.T) {
	lookup := NewDHCPOptionDefinitionLookup()

	option := DHCPOption{
		Code:     100,
		Space:    "dhcp6",
		Universe: storkutil.IPv6,
	}
	require.True(t, lookup.DefinitionExists(1, option))
}

// Test that the option definition lookup flags the standard options
// for which the definitions do not exist.
func TestStandardDHCPv6OptionDefinitionNotExists(t *testing.T) {
	lookup := NewDHCPOptionDefinitionLookup()

	nonExistingCodes := []uint16{0, 145}
	for _, code := range nonExistingCodes {
		option := DHCPOption{
			Code:     code,
			Space:    "dhcp6",
			Universe: storkutil.IPv6,
		}
		require.False(t, lookup.DefinitionExists(1, option))
	}
}

// Test that the option definition lookup indicates that the DHCPv6
// suboption does not have a definition.
func TestDHCPv6SuboptionDefinition(t *testing.T) {
	lookup := NewDHCPOptionDefinitionLookup()

	option := DHCPOption{
		Code:     15,
		Space:    "foo",
		Universe: storkutil.IPv6,
	}
	require.False(t, lookup.DefinitionExists(1, option))
}

// Test that standard option definition exists for a non-top level
// option space.
func TestStandardDHCPv6OptionDefinitionInOtherSpace(t *testing.T) {
	lookup := NewDHCPOptionDefinitionLookup()

	option := DHCPOption{
		Code:     89,
		Space:    "s46-cont-mape-options",
		Universe: storkutil.IPv6,
	}
	require.True(t, lookup.DefinitionExists(1, option))
}

// Test that option definition lookup can find a definition for a Kea
// standard option.
func TestFindStdDHCPOptionDefinition(t *testing.T) {
	option := &DHCPOption{
		Code:     89,
		Space:    "s46-cont-mape-options",
		Universe: storkutil.IPv6,
	}
	lookup := NewDHCPOptionDefinitionLookup()
	def := lookup.Find(1, option)
	require.NotNil(t, def)
}

// Test that nil value is returned if an option definition is not found.
func TestFindStdDHCPOptionDefinitionNotFound(t *testing.T) {
	option := &DHCPOption{
		Code:     1,
		Space:    "foo",
		Universe: storkutil.IPv6,
	}
	lookup := NewDHCPOptionDefinitionLookup()
	def := lookup.Find(1, option)
	require.Nil(t, def)
}
