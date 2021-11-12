package dumps_test

import (
	"testing"

	"github.com/go-pg/pg/v9"
	"github.com/stretchr/testify/require"
	dbmodel "isc.org/stork/server/database/model"
	dbtest "isc.org/stork/server/database/test"
	"isc.org/stork/server/dumper/dumps"
)

// Helper function that fill the database
// and returns the created machine.
func initDatabase(db *pg.DB) *dbmodel.Machine {
	m := &dbmodel.Machine{
		ID:         42,
		Address:    "localhost",
		AgentPort:  8080,
		Authorized: true,
	}
	_ = dbmodel.AddMachine(db, m)

	a := &dbmodel.App{
		ID:        0,
		MachineID: m.ID,
		Type:      "bind9",
		AccessPoints: []*dbmodel.AccessPoint{
			{
				MachineID: m.ID,
				Type:      "control",
				Address:   "dns.example.",
				Port:      953,
				Key:       "abcd",
			},
		},
		Daemons: []*dbmodel.Daemon{
			dbmodel.NewKeaDaemon(dbmodel.DaemonNameDHCPv4, true),
			dbmodel.NewBind9Daemon(true),
		},
	}
	ds, _ := dbmodel.AddApp(db, a)

	d := ds[0]
	_ = d.SetConfigFromJSON(`{
        "Dhcp4": {
            "valid-lifetime": 1234,
			"secret": "hidden"
        }
    }`)
	_ = dbmodel.UpdateDaemon(db, d)

	m, _ = dbmodel.GetMachineByID(db, m.ID)
	return m
}

// Helper function that extract the machine from the dump.
func extractMachineFromDump(dump dumps.Dump) (*dbmodel.Machine, bool) {
	if dump.NumberOfArtifacts() != 1 {
		return nil, false
	}
	artifact := dump.GetArtifact(0).(dumps.StructArtifact)
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

	dump := dumps.NewMachineDump(m)

	// Act
	err := dump.Execute()
	machine, ok := extractMachineFromDump(dump)

	// Assert
	require.NoError(t, err)
	require.EqualValues(t, 1, dump.NumberOfArtifacts())
	require.True(t, ok)

	require.EqualValues(t, "localhost", machine.Address)
	require.Len(t, machine.Apps, 1)
	require.Len(t, machine.Apps[0].AccessPoints, 1)
	require.Len(t, machine.Apps[0].Daemons, 2)
	require.NotNil(t, machine.Apps[0].Daemons[0].KeaDaemon.Config)
}

// Test that the dump doesn't contain the secrets.
func TestMachineDumpExecuteHideSecrets(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()
	m := initDatabase(db)
	dump := dumps.NewMachineDump(m)

	// Act
	_ = dump.Execute()
	machine, _ := extractMachineFromDump(dump)

	// Assert
	app := machine.Apps[0]
	daemon := app.Daemons[0]
	config := *(*map[string]interface{})(daemon.KeaDaemon.Config)
	secret := (config["Dhcp4"]).(map[string]interface{})["secret"]
	require.Nil(t, secret)
}
