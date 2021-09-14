//go:build !race

// Tests that intentionally use race conditions.
package dbmodel

import (
	"encoding/json"
	"sync"
	"testing"

	require "github.com/stretchr/testify/require"
	dbtest "isc.org/stork/server/database/test"
	storktestutil "isc.org/stork/server/testutil"
)

// When multiple parallel queries try to add the same application
// then only one of them may success.
func TestAddAppRace(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// add first machine, should be no error
	m := &Machine{
		ID:        0,
		Address:   "localhost",
		AgentPort: 8080,
	}
	err := AddMachine(db, m)
	require.NoError(t, err)
	require.NotZero(t, m.ID)

	app := &App{
		ID:        0,
		Type:      AppTypeKea,
		MachineID: m.ID,
		Name:      "single",
	}

	// Chain of errors, only one of them should be nil
	tasks := 50
	out := make(chan error, tasks)
	var wg sync.WaitGroup

	wg.Add(tasks)
	// Add function
	for i := 0; i < tasks; i++ {
		go func() {
			_, err = AddApp(db, app)
			out <- err
			wg.Done()
		}()
	}

	wg.Wait()
	close(out)

	nilCount := 0
	for err := range out {
		if err == nil {
			nilCount++
		}
	}

	require.EqualValues(t, nilCount, 1)
}

// When multiple parallel queries try to add or update the same application
// then only one of them may create new entry, rest should only update.
func TestAddOrUpdateAppRace(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// add first machine, should be no error
	m := &Machine{
		ID:        0,
		Address:   "localhost",
		AgentPort: 8080,
	}
	err := AddMachine(db, m)
	require.NoError(t, err)
	require.NotZero(t, m.ID)

	// Chain of errors, only one of them should be nil
	tasks := 50

	type Result struct {
		Error error // should be nil for all tasks
		AppID int64 // should be the same for all tasks
		IsNew bool  // should be true only for the one task
	}
	out := make(chan Result, tasks)
	var wg sync.WaitGroup

	// DHCPv4 configuration.
	var kea4Config *KeaConfig
	v4Config, err := json.Marshal(storktestutil.GenerateKeaConfig(5000))
	require.NoError(t, err)
	kea4Config, err = NewKeaConfigFromJSON(string(v4Config))
	require.NoError(t, err)

	protoApp := App{
		ID:        0,
		Type:      AppTypeKea,
		MachineID: m.ID,
		Name:      "single",
		Daemons: []*Daemon{
			{
				Name:   "dhcp4",
				Active: true,
				KeaDaemon: &KeaDaemon{
					Config:        kea4Config,
					KeaDHCPDaemon: &KeaDHCPDaemon{},
				},
			},
		},
	}

	wg.Add(tasks)
	// Add function
	for i := 0; i < tasks; i++ {
		go func() {
			// Copy app object
			app := protoApp
			_, _, newApp, err := AddOrUpdateApp(db, &app)
			id := app.ID
			res := Result{err, id, newApp}
			out <- res
			wg.Done()
		}()
	}

	wg.Wait()
	close(out)

	counts := make(map[int64]int)
	errorCount := 0
	newCount := 0
	for res := range out {
		require.NoError(t, res.Error)
		if res.Error != nil {
			errorCount++
		}
		if res.IsNew {
			newCount++
		}
		if count, ok := counts[res.AppID]; ok {
			counts[res.AppID] = count + 1
		} else {
			counts[res.AppID] = 1
		}
	}

	require.EqualValues(t, errorCount, 0)
	require.EqualValues(t, newCount, 1)
	require.EqualValues(t, len(counts), 1)
}
