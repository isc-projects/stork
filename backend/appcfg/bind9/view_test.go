package bind9config

import (
	"testing"

	"github.com/stretchr/testify/require"
	storkutil "isc.org/stork/util"
)

// Test checking if the view contains no-parse directives.
func TestViewHasNoParse(t *testing.T) {
	view := &View{
		Clauses: []*ViewClause{
			{NoParse: &NoParse{}},
		},
	}
	require.True(t, view.HasNoParse())
}

// Test checking if the view contains no-parse directives in the zone.
func TestViewZoneHasNoParseZone(t *testing.T) {
	view := &View{
		Clauses: []*ViewClause{
			{Zone: &Zone{
				Clauses: []*ZoneClause{
					{NoParse: &NoParse{}},
				},
			}},
		},
	}
	require.True(t, view.HasNoParse())
}

// Test checking if the view does not contain no-parse directives.
func TestViewHasNoParseNone(t *testing.T) {
	view := &View{}
	require.False(t, view.HasNoParse())
}

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
					AddressMatchList: &AddressMatchList{
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
	require.Len(t, matchClients.AddressMatchList.Elements, 1)
	require.Equal(t, "example.com", matchClients.AddressMatchList.Elements[0].KeyID)
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

// Test getting the response-policy clause from view.
func TestViewGetResponsePolicy(t *testing.T) {
	view := &View{
		Clauses: []*ViewClause{
			{
				Zone: &Zone{
					Name: "example.com",
				},
			},
			{
				ResponsePolicy: &ResponsePolicy{
					Zones: []*ResponsePolicyZone{
						{
							Zone: "rpz.example.com",
						},
					},
				},
			},
		},
	}
	responsePolicy := view.GetResponsePolicy()
	require.NotNil(t, responsePolicy)
	require.Len(t, responsePolicy.Zones, 1)
}

// Test that the view is formatted correctly.
func TestViewGetFormattedOutput(t *testing.T) {
	view := &View{
		Name:  "trusted",
		Class: "IN",
		Clauses: []*ViewClause{
			{
				Zone: &Zone{
					Name: "example.com",
					Clauses: []*ZoneClause{
						{
							Option: &Option{
								Identifier: "type",
								Switches: []OptionSwitch{
									{
										IdentSwitch: storkutil.Ptr("forward"),
									},
								},
							},
						},
					},
				},
			},
			{
				Option: &Option{
					Identifier: "test-option",
					Switches: []OptionSwitch{
						{
							IdentSwitch: storkutil.Ptr("true"),
						},
					},
				},
			},
		},
	}
	output := view.getFormattedOutput(nil)
	require.NotNil(t, output)
	requireConfigEq(t, `view "trusted" IN {
		zone "example.com" {
			type forward;
		};
		test-option true;
	};`, output)
}

// Test that serializing a view with nil values does not panic.
func TestViewFormatNilValues(t *testing.T) {
	view := &View{}
	require.NotPanics(t, func() { view.getFormattedOutput(nil) })
}

// Test that serializing a view clause with nil values does not panic.
func TestViewClauseFormatNilValues(t *testing.T) {
	viewClause := &ViewClause{}
	require.NotPanics(t, func() { viewClause.getFormattedOutput(nil) })
}
