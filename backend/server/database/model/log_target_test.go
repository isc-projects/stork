package dbmodel

import (
	"testing"

	"github.com/stretchr/testify/require"
	"isc.org/stork/datamodel/daemonname"
	dbtest "isc.org/stork/server/database/test"
)

// Test that the log target can be fetched from the database by ID.
func TestGetLogTargetByID(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	m := &Machine{
		ID:        0,
		Address:   "localhost",
		AgentPort: 8080,
	}
	err := AddMachine(db, m)
	require.NoError(t, err)
	require.NotZero(t, m.ID)

	daemon := NewDaemon(m, "kea-dhcp4", true, []*AccessPoint{
		{
			Type:    AccessPointControl,
			Address: "",
			Port:    8000,
			Key:     "",
		},
	})
	daemon.Version = "1.7.5"
	daemon.LogTargets = []*LogTarget{
		{
			Output: "stdout",
		},
		{
			Output: "/tmp/filename.log",
		},
	}
	err = AddDaemon(db, daemon)
	require.NoError(t, err)
	require.NotZero(t, daemon.ID)

	require.Len(t, daemon.LogTargets, 2)

	// Make sure that the log targets have been assigned IDs.
	require.NotZero(t, daemon.LogTargets[0].ID)
	require.NotZero(t, daemon.LogTargets[1].ID)

	// Get the first log target from the database by id.
	logTarget, err := GetLogTargetByID(db, daemon.LogTargets[0].ID)
	require.NoError(t, err)
	require.NotNil(t, logTarget)
	require.Equal(t, "stdout", logTarget.Output)
	require.NotNil(t, logTarget.Daemon)
	require.NotNil(t, logTarget.Daemon.Machine)

	// Get the second log target by id.
	logTarget, err = GetLogTargetByID(db, daemon.LogTargets[1].ID)
	require.NoError(t, err)
	require.NotNil(t, logTarget)
	require.Equal(t, "/tmp/filename.log", logTarget.Output)
	require.NotNil(t, logTarget.Daemon)
	require.NotNil(t, logTarget.Daemon.Machine)

	// Use the non existing id. This should return nil.
	logTarget, err = GetLogTargetByID(db, daemon.LogTargets[1].ID+1000)
	require.NoError(t, err)
	require.Nil(t, logTarget)
}

// Test that log targets can be deleted by daemon ID except for specified IDs.
func TestDeleteLogTargetsByDaemonIDExcept(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	m := &Machine{
		ID:        0,
		Address:   "localhost",
		AgentPort: 8080,
	}
	err := AddMachine(db, m)
	require.NoError(t, err)
	require.NotZero(t, m.ID)

	daemon1 := NewDaemon(m, daemonname.DHCPv4, true, []*AccessPoint{
		{
			Type:    AccessPointControl,
			Address: "",
			Port:    8000,
			Key:     "",
		},
	})
	daemon1.LogTargets = []*LogTarget{
		{
			Output: "stdout",
		},
		{
			Output: "/tmp/filename.log",
		},
	}
	_ = AddDaemon(db, daemon1)

	daemon2 := NewDaemon(m, daemonname.DHCPv6, true, []*AccessPoint{
		{
			Type:    AccessPointControl,
			Address: "",
			Port:    8000,
			Key:     "",
		},
	})
	daemon2.LogTargets = []*LogTarget{
		{
			Output: "stdout",
		},
		{
			Output: "/tmp/filename.log",
		},
	}
	_ = AddDaemon(db, daemon2)

	// Act
	err = deleteLogTargetsByDaemonIDExcept(db, daemon1.ID, []int64{daemon1.LogTargets[1].ID})

	// Assert
	require.NoError(t, err)

	daemons, _ := GetAllDaemons(db)
	daemon1 = &daemons[0]
	daemon2 = &daemons[1]

	require.Equal(t, daemonname.DHCPv4, daemon1.Name)
	require.Len(t, daemon1.LogTargets, 1)
	require.Equal(t, "/tmp/filename.log", daemon1.LogTargets[0].Output)

	require.Equal(t, daemonname.DHCPv6, daemon2.Name)
	require.Len(t, daemon2.LogTargets, 2)
}

// Test that all log targets of a daemon can be deleted.
func TestDeleteLogTargetsByDaemonIDAll(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	m := &Machine{
		ID:        0,
		Address:   "localhost",
		AgentPort: 8080,
	}
	err := AddMachine(db, m)
	require.NoError(t, err)
	require.NotZero(t, m.ID)

	daemon1 := NewDaemon(m, daemonname.DHCPv4, true, []*AccessPoint{
		{
			Type:    AccessPointControl,
			Address: "",
			Port:    8000,
			Key:     "",
		},
	})
	daemon1.LogTargets = []*LogTarget{
		{
			Output: "stdout",
		},
		{
			Output: "/tmp/filename.log",
		},
	}
	_ = AddDaemon(db, daemon1)

	daemon2 := NewDaemon(m, daemonname.DHCPv6, true, []*AccessPoint{
		{
			Type:    AccessPointControl,
			Address: "",
			Port:    8000,
			Key:     "",
		},
	})
	daemon2.LogTargets = []*LogTarget{
		{
			Output: "stdout",
		},
		{
			Output: "/tmp/filename.log",
		},
	}
	_ = AddDaemon(db, daemon2)

	// Act
	err = deleteLogTargetsByDaemonIDExcept(db, daemon1.ID, []int64{})

	// Assert
	require.NoError(t, err)

	daemons, _ := GetAllDaemons(db)
	daemon1 = &daemons[0]
	daemon2 = &daemons[1]

	require.Equal(t, daemonname.DHCPv4, daemon1.Name)
	require.Len(t, daemon1.LogTargets, 0)

	require.Equal(t, daemonname.DHCPv6, daemon2.Name)
	require.Len(t, daemon2.LogTargets, 2)
}

// Test that log targets can be added to a daemon.
func TestAddLogTarget(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	m := &Machine{
		ID:        0,
		Address:   "localhost",
		AgentPort: 8080,
	}
	err := AddMachine(db, m)
	require.NoError(t, err)
	require.NotZero(t, m.ID)

	daemon := NewDaemon(m, daemonname.DHCPv4, true, []*AccessPoint{
		{
			Type:    AccessPointControl,
			Address: "",
			Port:    8000,
			Key:     "",
		},
	})
	_ = AddDaemon(db, daemon)

	// Act
	logTarget := &LogTarget{
		Output:   "/var/log/stork.log",
		Severity: "info",
		DaemonID: daemon.ID,
	}
	err = addLogTarget(db, logTarget)

	// Assert
	require.NoError(t, err)
	require.NotZero(t, logTarget.ID)

	fetchedLogTarget, err := GetLogTargetByID(db, logTarget.ID)
	require.NoError(t, err)
	require.NotNil(t, fetchedLogTarget)
	require.Equal(t, "/var/log/stork.log", fetchedLogTarget.Output)
	require.Equal(t, "info", fetchedLogTarget.Severity)
}

// Test that log targets can be updated.
func TestUpdateLogTarget(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	m := &Machine{
		ID:        0,
		Address:   "localhost",
		AgentPort: 8080,
	}
	err := AddMachine(db, m)
	require.NoError(t, err)
	require.NotZero(t, m.ID)

	daemon := NewDaemon(m, daemonname.DHCPv4, true, []*AccessPoint{
		{
			Type:    AccessPointControl,
			Address: "",
			Port:    8000,
			Key:     "",
		},
	})
	daemon.LogTargets = []*LogTarget{
		{
			Output:   "/var/log/stork.log",
			Severity: "info",
		},
	}

	_ = AddDaemon(db, daemon)

	// Act
	logTarget := daemon.LogTargets[0]
	logTarget.Output = "/var/log/updated_stork.log"
	logTarget.Severity = "debug"
	err = updateLogTarget(db, logTarget)

	// Assert
	require.NoError(t, err)

	fetchedLogTarget, err := GetLogTargetByID(db, logTarget.ID)
	require.NoError(t, err)
	require.NotNil(t, fetchedLogTarget)
	require.Equal(t, "/var/log/updated_stork.log", fetchedLogTarget.Output)
	require.Equal(t, "debug", fetchedLogTarget.Severity)
}

// Test that log targets can be added or updated.
func TestAddOrUpdateLogTarget(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	m := &Machine{
		ID:        0,
		Address:   "localhost",
		AgentPort: 8080,
	}
	err := AddMachine(db, m)
	require.NoError(t, err)
	require.NotZero(t, m.ID)

	daemon := NewDaemon(m, daemonname.DHCPv4, true, []*AccessPoint{
		{
			Type:    AccessPointControl,
			Address: "",
			Port:    8000,
			Key:     "",
		},
	})
	_ = AddDaemon(db, daemon)

	// Act
	err1 := addOrUpdateLogTarget(db, &LogTarget{
		Output:   "/var/log/new_stork.log",
		Severity: "info",
		DaemonID: daemon.ID,
	})

	err2 := addOrUpdateLogTarget(db, &LogTarget{
		ID:       1,
		Output:   "/var/log/updated_stork.log",
		Severity: "debug",
		DaemonID: daemon.ID,
	})

	// Assert
	require.NoError(t, err1)
	require.NoError(t, err2)

	fetchedLogTarget1, err := GetLogTargetByID(db, 1)
	require.NoError(t, err)
	require.NotNil(t, fetchedLogTarget1)
	require.Equal(t, "/var/log/updated_stork.log", fetchedLogTarget1.Output)
	require.Equal(t, "debug", fetchedLogTarget1.Severity)
}
