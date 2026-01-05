package dbmodel

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"isc.org/stork/datamodel/daemonname"
	dnsmodel "isc.org/stork/datamodel/dns"
	dbtest "isc.org/stork/server/database/test"
	storkutil "isc.org/stork/util"
)

// Test creating a filter with nil parameters.
func TestNewGetZoneRRsFilterWithNilParams(t *testing.T) {
	filter := NewGetZoneRRsFilterWithParams(nil, nil, nil, nil)
	require.NotNil(t, filter)
	require.Empty(t, filter.GetTypes())
	require.Empty(t, filter.GetText())
	require.Equal(t, 0, filter.GetOffset())
	require.Equal(t, 0, filter.GetLimit())
}

// Test creating a filter with actual parameters.
func TestNewGetZoneRRsFilterWithActualParams(t *testing.T) {
	filter := NewGetZoneRRsFilterWithParams(storkutil.Ptr(int64(1)), storkutil.Ptr(int64(10)), []string{"A", "AAAA"}, storkutil.Ptr("example.com"))
	require.NotNil(t, filter)
	require.ElementsMatch(t, filter.GetTypes(), []string{"A", "AAAA"})
	require.Equal(t, "example.com", filter.GetText())
	require.Equal(t, 1, filter.GetOffset())
	require.Equal(t, 10, filter.GetLimit())
}

// Test that the IsTextMatches method returns true if the text is contained in the RR name.
func TestGetZoneRRsFilterIsTextMatchesName(t *testing.T) {
	filter := NewGetZoneRRsFilter()
	filter.SetText("EXAMPLE.co")
	require.True(t, filter.IsTextMatches(&dnsmodel.RR{Name: "example.COM", Type: "A", Rdata: "192.0.2.1"}))
	require.False(t, filter.IsTextMatches(&dnsmodel.RR{Name: "example.org", Type: "A", Rdata: "192.0.2.1"}))
}

// Test that the IsTextMatches method returns true if the text is contained in the RR rdata.
func TestGetZoneRRsFilterIsTextMatchesRdata(t *testing.T) {
	filter := NewGetZoneRRsFilter()
	filter.SetText("0.2.2")
	require.False(t, filter.IsTextMatches(&dnsmodel.RR{Name: "example.com", Type: "A", Rdata: "192.0.2.1"}))
	require.True(t, filter.IsTextMatches(&dnsmodel.RR{Name: "example.org", Type: "A", Rdata: "192.0.2.2"}))
}

// Test that the IsTextMatches method returns true if the text is empty.
func TestGetZoneRRsFilterIsTextMatchesEmpty(t *testing.T) {
	filter := NewGetZoneRRsFilter()
	filter.SetText("")
	require.True(t, filter.IsTextMatches(&dnsmodel.RR{Name: "example.com", Type: "A", Rdata: "192.0.2.1"}))
	require.True(t, filter.IsTextMatches(&dnsmodel.RR{Name: "example.org", Type: "A", Rdata: "192.0.2.2"}))
}

// Test setting the offset on the filter.
func TestZoneRRFilterSetOffset(t *testing.T) {
	filter := NewGetZoneRRsFilter()
	require.Equal(t, 0, filter.GetOffset())

	filter.SetOffset(1)
	require.Equal(t, 1, filter.GetOffset())
}

// Test setting the limit on the filter.
func TestZoneRRFilterSetLimit(t *testing.T) {
	filter := NewGetZoneRRsFilter()
	require.Equal(t, 0, filter.GetLimit())

	filter.SetLimit(10)
	require.Equal(t, 10, filter.GetLimit())
}

// Test that 0 offset is returned from a nil filter.
func TestZoneRRFilterGetOffsetFromNilFilter(t *testing.T) {
	var filter *GetZoneRRsFilter
	require.Equal(t, 0, filter.GetOffset())
}

// Test that 0 limit is returned from a nil filter.
func TestZoneRRFilterGetLimitFromNilFilter(t *testing.T) {
	var filter *GetZoneRRsFilter
	require.Equal(t, 0, filter.GetLimit())
}

// Test enabling filtering by RR using case-insensitive type names.
func TestZoneRRFilterEnableType(t *testing.T) {
	filter := NewGetZoneRRsFilter()
	require.Empty(t, filter.GetTypes())

	filter.EnableType("a")
	require.ElementsMatch(t, filter.GetTypes(), []string{"A"})

	filter.EnableType("AAAA")
	require.ElementsMatch(t, filter.GetTypes(), []string{"A", "AAAA"})

	require.True(t, filter.IsTypeEnabled("A"))
	require.True(t, filter.IsTypeEnabled("AAAA"))
	require.False(t, filter.IsTypeEnabled("SOA"))
}

// Test setting filtering by text.
func TestZoneRRFilterSetText(t *testing.T) {
	filter := NewGetZoneRRsFilter()
	require.Empty(t, filter.GetText())

	filter.SetText("example.com")
	require.Equal(t, "example.com", filter.GetText())

	filter.SetText("")
	require.Empty(t, filter.GetText())
}

// Test adding a set of RRs to the database.
func TestAddGetDeleteLocalZoneRRs(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// Add a machine.
	machine := &Machine{
		ID:        0,
		Address:   "localhost",
		AgentPort: int64(8080),
	}
	err := AddMachine(db, machine)
	require.NoError(t, err)

	// Add a daemon.
	daemon := NewDaemon(machine, daemonname.Bind9, true, []*AccessPoint{})
	err = AddDaemon(db, daemon)
	require.NoError(t, err)

	// Add a zone.
	zone := &Zone{
		Name: "example.com.",
		LocalZones: []*LocalZone{
			{
				DaemonID: daemon.ID,
				View:     "_default",
				Class:    "IN",
				Serial:   123456,
				Type:     string(ZoneTypePrimary),
				LoadedAt: time.Now().UTC(),
			},
		},
	}
	err = AddZones(db, zone)
	require.NoError(t, err)

	// Add RRs to the database.
	rrs := []string{
		"example.com. 3600 IN SOA ns1.example.com. hostmaster.example.com. 123456 7200 3600 1209600 3600",
		"example.com. 3600 IN A 192.0.2.1",
	}
	var parsedRRs []*LocalZoneRR
	for _, rr := range rrs {
		parsedRR, err := dnsmodel.NewRR(rr)
		require.NoError(t, err)
		parsedRRs = append(parsedRRs, &LocalZoneRR{
			RR:          *parsedRR,
			LocalZoneID: zone.LocalZones[0].ID,
		})
	}
	err = AddLocalZoneRRs(db, parsedRRs...)
	require.NoError(t, err)

	// Get RRs from the database.
	returnedRRs, total, err := GetDNSConfigRRs(db, zone.LocalZones[0].ID, nil)
	require.NoError(t, err)
	require.Len(t, returnedRRs, 2)
	require.Equal(t, 2, total)

	// Make sure they are correct.
	for i, returnedRR := range returnedRRs {
		require.Equal(t, rrs[i], returnedRR.GetString())
	}

	// Delete RRs from the database using non-existing local zone ID.
	err = DeleteLocalZoneRRs(db, zone.LocalZones[0].ID+1)
	require.NoError(t, err)

	// Make sure they are not deleted.
	returnedRRs, total, err = GetDNSConfigRRs(db, zone.LocalZones[0].ID, nil)
	require.NoError(t, err)
	require.Equal(t, 2, total)
	require.Len(t, returnedRRs, 2)

	// Delete RRs from the database using existing local zone ID.
	err = DeleteLocalZoneRRs(db, zone.LocalZones[0].ID)
	require.NoError(t, err)

	// Make sure they are deleted.
	returnedRRs, total, err = GetDNSConfigRRs(db, zone.LocalZones[0].ID, nil)
	require.NoError(t, err)
	require.Equal(t, 0, total)
	require.Len(t, returnedRRs, 0)
}

// Test filtering RRs from the database.
func TestFilterLocalZoneRRs(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// Add a machine.
	machine := &Machine{
		ID:        0,
		Address:   "localhost",
		AgentPort: int64(8080),
	}
	err := AddMachine(db, machine)
	require.NoError(t, err)

	// Add a daemon.
	daemon := NewDaemon(machine, daemonname.Bind9, true, []*AccessPoint{})
	err = AddDaemon(db, daemon)
	require.NoError(t, err)

	// Add a zone.
	zone := &Zone{
		Name: "example.com.",
		LocalZones: []*LocalZone{
			{
				DaemonID: daemon.ID,
				View:     "_default",
				Class:    "IN",
				Serial:   123456,
				Type:     string(ZoneTypePrimary),
				LoadedAt: time.Now().UTC(),
			},
		},
	}
	err = AddZones(db, zone)
	require.NoError(t, err)

	// Add RRs to the database.
	rrs := []string{
		"example.com. 3600 IN SOA ns1.example.com. hostmaster.example.com. 123456 7200 3600 1209600 3600",
		"a.example.com. 3600 IN A 192.0.2.1",
		"aaaa.example.com. 3600 IN AAAA 2001:db8::1",
		"foo.example.com. 3600 IN A 192.0.2.2",
		"bar.example.com. 3600 IN AAAA 2001:db8::2",
	}
	var parsedRRs []*LocalZoneRR
	for _, rr := range rrs {
		parsedRR, err := dnsmodel.NewRR(rr)
		require.NoError(t, err)
		parsedRRs = append(parsedRRs, &LocalZoneRR{
			RR:          *parsedRR,
			LocalZoneID: zone.LocalZones[0].ID,
		})
	}
	err = AddLocalZoneRRs(db, parsedRRs...)
	require.NoError(t, err)

	t.Run("filter by SOA type", func(t *testing.T) {
		filter := NewGetZoneRRsFilter()
		filter.EnableType("SOA")
		returnedRRs, total, err := GetDNSConfigRRs(db, zone.LocalZones[0].ID, filter)
		require.NoError(t, err)
		require.Len(t, returnedRRs, 1)
		require.Equal(t, 1, total)
		require.Equal(t, rrs[0], returnedRRs[0].GetString())
	})

	t.Run("filter by A type", func(t *testing.T) {
		filter := NewGetZoneRRsFilter()
		filter.EnableType("A")
		returnedRRs, total, err := GetDNSConfigRRs(db, zone.LocalZones[0].ID, filter)
		require.NoError(t, err)
		require.Len(t, returnedRRs, 2)
		require.Equal(t, 2, total)
		require.Equal(t, rrs[1], returnedRRs[0].GetString())
		require.Equal(t, rrs[3], returnedRRs[1].GetString())
	})

	t.Run("filter by AAAA type", func(t *testing.T) {
		filter := NewGetZoneRRsFilter()
		filter.EnableType("AAAA")
		returnedRRs, total, err := GetDNSConfigRRs(db, zone.LocalZones[0].ID, filter)
		require.NoError(t, err)
		require.Len(t, returnedRRs, 2)
		require.Equal(t, 2, total)
		require.Equal(t, rrs[2], returnedRRs[0].GetString())
		require.Equal(t, rrs[4], returnedRRs[1].GetString())
	})

	t.Run("filter SOA by radata", func(t *testing.T) {
		filter := NewGetZoneRRsFilter()
		filter.SetText("HOSTMASTER")
		returnedRRs, total, err := GetDNSConfigRRs(db, zone.LocalZones[0].ID, filter)
		require.NoError(t, err)
		require.Equal(t, 1, total)
		require.Len(t, returnedRRs, 1)
		require.Equal(t, rrs[0], returnedRRs[0].GetString())
	})

	t.Run("filter A by rdata", func(t *testing.T) {
		filter := NewGetZoneRRsFilter()
		filter.SetText("192.0.2.1")
		returnedRRs, total, err := GetDNSConfigRRs(db, zone.LocalZones[0].ID, filter)
		require.NoError(t, err)
		require.Equal(t, 1, total)
		require.Len(t, returnedRRs, 1)
		require.Equal(t, rrs[1], returnedRRs[0].GetString())
	})

	t.Run("filter AAAA by rdata", func(t *testing.T) {
		filter := NewGetZoneRRsFilter()
		filter.SetText("2001:db8::1")
		returnedRRs, total, err := GetDNSConfigRRs(db, zone.LocalZones[0].ID, filter)
		require.NoError(t, err)
		require.Equal(t, 1, total)
		require.Len(t, returnedRRs, 1)
		require.Equal(t, rrs[2], returnedRRs[0].GetString())
	})

	t.Run("filter by type and text", func(t *testing.T) {
		filter := NewGetZoneRRsFilter()
		filter.SetText("FOO.example")
		filter.EnableType("A")
		returnedRRs, total, err := GetDNSConfigRRs(db, zone.LocalZones[0].ID, filter)
		require.NoError(t, err)
		require.Equal(t, 1, total)
		require.Len(t, returnedRRs, 1)
		require.Equal(t, rrs[3], returnedRRs[0].GetString())
	})

	t.Run("filter with no matching results", func(t *testing.T) {
		filter := NewGetZoneRRsFilter()
		filter.SetText("192.0.2.3")
		filter.EnableType("A")
		returnedRRs, total, err := GetDNSConfigRRs(db, zone.LocalZones[0].ID, filter)
		require.NoError(t, err)
		require.Equal(t, 0, total)
		require.Empty(t, returnedRRs)
	})

	t.Run("filter with limit and offset", func(t *testing.T) {
		filter := NewGetZoneRRsFilter()
		filter.SetLimit(1)
		filter.SetOffset(1)
		returnedRRs, total, err := GetDNSConfigRRs(db, zone.LocalZones[0].ID, filter)
		require.NoError(t, err)
		require.Equal(t, 5, total)
		require.Len(t, returnedRRs, 1)
		require.Equal(t, rrs[1], returnedRRs[0].GetString())
	})

	t.Run("filter with offset and no limit", func(t *testing.T) {
		filter := NewGetZoneRRsFilter()
		filter.SetOffset(2)
		returnedRRs, total, err := GetDNSConfigRRs(db, zone.LocalZones[0].ID, filter)
		require.NoError(t, err)
		require.Equal(t, 5, total)
		require.Len(t, returnedRRs, 3)
		require.Equal(t, rrs[2], returnedRRs[0].GetString())
		require.Equal(t, rrs[3], returnedRRs[1].GetString())
		require.Equal(t, rrs[4], returnedRRs[2].GetString())
	})
}
