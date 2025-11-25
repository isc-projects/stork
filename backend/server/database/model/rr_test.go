package dbmodel

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"isc.org/stork/daemoncfg/dnsconfig"
	"isc.org/stork/datamodel/daemonname"
	dbtest "isc.org/stork/server/database/test"
)

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
		parsedRR, err := dnsconfig.NewRR(rr)
		require.NoError(t, err)
		parsedRRs = append(parsedRRs, &LocalZoneRR{
			RR:          *parsedRR,
			LocalZoneID: zone.LocalZones[0].ID,
		})
	}
	err = AddLocalZoneRRs(db, parsedRRs...)
	require.NoError(t, err)

	// Get RRs from the database.
	returnedRRs, err := GetDNSConfigRRs(db, zone.LocalZones[0].ID)
	require.NoError(t, err)
	require.Len(t, returnedRRs, 2)

	// Make sure they are correct.
	for i, returnedRR := range returnedRRs {
		require.Equal(t, rrs[i], returnedRR.GetString())
	}

	// Delete RRs from the database using non-existing local zone ID.
	err = DeleteLocalZoneRRs(db, zone.LocalZones[0].ID+1)
	require.NoError(t, err)

	// Make sure they are not deleted.
	returnedRRs, err = GetDNSConfigRRs(db, zone.LocalZones[0].ID)
	require.NoError(t, err)
	require.Len(t, returnedRRs, 2)

	// Delete RRs from the database using existing local zone ID.
	err = DeleteLocalZoneRRs(db, zone.LocalZones[0].ID)
	require.NoError(t, err)

	// Make sure they are deleted.
	returnedRRs, err = GetDNSConfigRRs(db, zone.LocalZones[0].ID)
	require.NoError(t, err)
	require.Len(t, returnedRRs, 0)
}
