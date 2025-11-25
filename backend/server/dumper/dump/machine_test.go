package dump_test

import (
	"testing"
	"time"

	"github.com/go-pg/pg/v10"
	"github.com/stretchr/testify/require"
	"isc.org/stork/datamodel/daemonname"
	dbmodel "isc.org/stork/server/database/model"
	dbtest "isc.org/stork/server/database/test"
	dumppkg "isc.org/stork/server/dumper/dump"
)

// Helper function that fills the database
// and returns the created machine.
func initDatabase(db *pg.DB) *dbmodel.Machine {
	m := &dbmodel.Machine{
		ID:         42,
		Address:    "localhost",
		AgentPort:  8080,
		Authorized: true,
		AgentToken: "secret",
	}
	_ = dbmodel.AddMachine(db, m)

	// Create access points
	accessPoints1 := []*dbmodel.AccessPoint{
		{
			Type:    "control",
			Address: "localhost",
			Port:    8080,
			Key:     "secret",
		},
	}

	accessPoints2 := []*dbmodel.AccessPoint{
		{
			Type:    "control",
			Address: "dns.example.",
			Port:    953,
			Key:     "abcd",
		},
	}

	// Create daemons
	daemon1 := dbmodel.NewDaemon(m, daemonname.DHCPv4, true, accessPoints1)
	daemon2 := dbmodel.NewDaemon(m, daemonname.Bind9, true, accessPoints2)
	daemon2.Version = "1.0.0"
	daemon2.LogTargets = []*dbmodel.LogTarget{
		{
			Output: "stdout",
		},
		{
			Output: "/tmp/filename.log",
		},
	}

	_ = dbmodel.AddDaemon(db, daemon1)
	_ = dbmodel.AddDaemon(db, daemon2)

	_ = daemon1.SetKeaConfigFromJSON([]byte(`{
        "Dhcp4": {
            "valid-lifetime": 1234,
			"secret": "hidden"
        }
    }`))
	daemon1.LogTargets = []*dbmodel.LogTarget{
		{
			Name:      "foo",
			Severity:  "bar",
			Output:    "/var/log/foo",
			CreatedAt: time.Time{},
			DaemonID:  daemon1.ID,
		},
	}
	_ = dbmodel.UpdateDaemon(db, daemon1)

	m, _ = dbmodel.GetMachineByIDWithRelations(db, m.ID,
		dbmodel.MachineRelationDaemons,
		dbmodel.MachineRelationKeaDaemons,
		dbmodel.MachineRelationBind9Daemons,
		dbmodel.MachineRelationDaemonLogTargets,
		dbmodel.MachineRelationDaemonAccessPoints,
		dbmodel.MachineRelationKeaDHCPConfigs,
	)
	return m
}

// Helper function that extract the machine from the dump.
func extractMachineFromDump(dump dumppkg.Dump) (*dbmodel.Machine, bool) {
	if dump.GetArtifactsNumber() != 1 {
		return nil, false
	}
	artifact := dump.GetArtifact(0).(dumppkg.StructArtifact)
	artifactContent := artifact.GetStruct()
	machine, ok := artifactContent.(*dbmodel.Machine)
	return machine, ok
}

// Test that the dump is executed properly.
func TestMachineDumpExecute(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()
	m := initDatabase(db)

	dump := dumppkg.NewMachineDump(m)

	// Act
	err := dump.Execute()
	machine, ok := extractMachineFromDump(dump)
	require.True(t, ok)

	// Assert
	require.NoError(t, err)
	require.EqualValues(t, 1, dump.GetArtifactsNumber())
	require.True(t, ok)

	require.EqualValues(t, "localhost", machine.Address)
	require.Len(t, machine.Daemons, 2)
	// Daemons can be returned out of order from the database, so we
	// have to iterate over them.
	for _, daemon := range machine.Daemons {
		switch daemon.Name {
		case daemonname.DHCPv4:
			require.NotNil(t, daemon.KeaDaemon.Config)
		case daemonname.Bind9:
			require.Len(t, daemon.LogTargets, 2)
		default:
			require.FailNow(t, "unknown daemon name", daemon.Name)
		}
	}
}

// Test that the dump doesn't contain the secrets.
func TestMachineDumpExecuteHideSecrets(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()
	m := initDatabase(db)
	dump := dumppkg.NewMachineDump(m)

	// Act
	_ = dump.Execute()
	machine, ok := extractMachineFromDump(dump)
	require.True(t, ok)

	// Assert
	for _, daemon := range machine.Daemons {
		// Daemons can be returned out of order from the database, so we
		// have to iterate over them.
		if daemon.Name == daemonname.DHCPv4 {
			config, _ := daemon.KeaDaemon.Config.GetRawConfig()
			secret := (config["Dhcp4"]).(map[string]interface{})["secret"]
			require.Nil(t, secret)
			require.Empty(t, machine.AgentToken)
		}
	}
}
