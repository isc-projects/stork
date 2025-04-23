package bind9config

import (
	"testing"

	"github.com/stretchr/testify/require"
	storkutil "isc.org/stork/util"
)

// Tests that allow-transfer is returned when specified.
func TestViewGetAllowTransfer(t *testing.T) {
	view := &View{
		Clauses: []*ViewClause{
			{
				AllowTransfer: &AllowTransfer{
					Port: storkutil.Ptr(int64(53)),
				},
			},
			{
				Zone: &Zone{
					Name: "example.com",
				},
			},
		},
	}
	allowTransfer := view.GetAllowTransfer()
	require.NotNil(t, allowTransfer)
	require.Equal(t, int64(53), *allowTransfer.Port)
}

// Tests that allow-transfer is not returned when not specified.
func TestViewGetAllowTransferLacking(t *testing.T) {
	view := &View{
		Clauses: []*ViewClause{
			{
				Zone: &Zone{
					Name: "example.com",
				},
			},
		},
	}
	allowTransfer := view.GetAllowTransfer()
	require.Nil(t, allowTransfer)
}

// Tests that match-clients is returned when specified.
func TestViewGetMatchClients(t *testing.T) {
	view := &View{
		Clauses: []*ViewClause{
			{
				MatchClients: &MatchClients{
					AdressMatchList: &AddressMatchList{
						Elements: []*AddressMatchListElement{
							{
								KeyID: "example.com",
							},
						},
					},
				},
			},
			{
				Zone: &Zone{
					Name: "example.com",
				},
			},
		},
	}
	matchClients := view.GetMatchClients()
	require.NotNil(t, matchClients)
	require.Len(t, matchClients.AdressMatchList.Elements, 1)
	require.Equal(t, "example.com", matchClients.AdressMatchList.Elements[0].KeyID)
}

// Tests that match-clients is not returned when not specified.
func TestViewGetMatchClientsLacking(t *testing.T) {
	view := &View{
		Clauses: []*ViewClause{
			{
				Zone: &Zone{
					Name: "example.com",
				},
			},
		},
	}
	require.Nil(t, view.GetMatchClients())
}

// Tests that zone is returned when specified.
func TestViewGetZone(t *testing.T) {
	view := &View{
		Clauses: []*ViewClause{
			{
				Zone: &Zone{
					Name: "example.com",
				},
			},
			{
				Zone: &Zone{
					Name: "example.org",
				},
			},
		},
	}
	zone := view.GetZone("example.com")
	require.NotNil(t, zone)
	require.Equal(t, "example.com", zone.Name)
}

// Tests that zone is not returned when not specified.
func TestViewGetZoneLacking(t *testing.T) {
	view := &View{
		Clauses: []*ViewClause{
			{
				Zone: &Zone{
					Name: "example.com",
				},
			},
			{
				Zone: &Zone{
					Name: "example.org",
				},
			},
		},
	}
	require.Nil(t, view.GetZone("test.example.com"))
}
