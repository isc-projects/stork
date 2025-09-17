package bind9config

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// Test that the match-clients clause is formatted correctly.
func TestMatchClientsFormat(t *testing.T) {
	matchClients := &MatchClients{
		AddressMatchList: &AddressMatchList{
			Elements: []*AddressMatchListElement{
				{
					IPAddressOrACLName: "127.0.0.1",
				},
				{
					KeyID: "foo",
				},
			},
		},
	}
	output := matchClients.getFormattedOutput(nil)
	require.NotNil(t, output)
	cfgEq(t, `match-clients { "127.0.0.1"; key "foo"; };`, output)
}

// Test that serializing a match-clients clause with nil values does not panic.
func TestMatchClientsFormatNilValues(t *testing.T) {
	matchClients := &MatchClients{}
	require.NotPanics(t, func() { matchClients.getFormattedOutput(nil) })
}
