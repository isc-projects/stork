package eventcenter

import (
	"testing"
	"testing/synctest"

	pkgerrors "github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	dbmodel "isc.org/stork/server/database/model"
	dbtest "isc.org/stork/server/database/test"
)

// Test that the event with a machine entry is created property.
func TestCreateEventMachine(t *testing.T) {
	// Arrange
	machine := &dbmodel.Machine{
		ID:      456,
		Address: "my-address",
		State: dbmodel.MachineState{
			Hostname: "my-hostname",
		},
	}

	// Act
	ev := CreateEvent(dbmodel.EvInfo, "foo {machine} bar", machine)

	// Assert
	require.EqualValues(t, "foo <machine id=\"456\" address=\"my-address\" hostname=\"my-hostname\"> bar", ev.Text)
	require.EqualValues(t, dbmodel.EvInfo, ev.Level)
	require.EqualValues(t, 456, ev.Relations.MachineID)
	require.Zero(t, ev.Relations.DaemonID)
	require.Zero(t, ev.Relations.SubnetID)
	require.Zero(t, ev.Relations.UserID)
	require.Empty(t, ev.Details)
	require.Zero(t, ev.CreatedAt)
}

// Test that the missing machine state doesn't cause problems.
func TestCreateEventMachineWithoutState(t *testing.T) {
	// Arrange
	machine := &dbmodel.Machine{
		ID: 456,
	}

	// Act
	ev := CreateEvent(dbmodel.EvInfo, "foo {machine} bar", machine)

	// Assert
	require.EqualValues(t, "foo <machine id=\"456\" address=\"\" hostname=\"\"> bar", ev.Text)
	require.EqualValues(t, dbmodel.EvInfo, ev.Level)
	require.EqualValues(t, 456, ev.Relations.MachineID)
	require.Zero(t, ev.Relations.DaemonID)
	require.Zero(t, ev.Relations.SubnetID)
	require.Zero(t, ev.Relations.UserID)
	require.Empty(t, ev.Details)
	require.Zero(t, ev.CreatedAt)
}

// Test that the event with a daemon entry is created properly.
func TestCreateEventDaemon(t *testing.T) {
	// Arrange
	daemon := &dbmodel.Daemon{
		ID:        234,
		Name:      "dhcp4",
		MachineID: 456,
		Machine: &dbmodel.Machine{
			ID: 456,
		},
		AccessPoints: []*dbmodel.AccessPoint{{
			Type:    dbmodel.AccessPointControl,
			Address: "localhost",
			Port:    8080,
		}},
	}

	// Act
	ev := CreateEvent(dbmodel.EvError, "foo {daemon} bar", daemon)

	// Assert
	require.EqualValues(t, "foo <daemon id=\"234\" name=\"dhcp4\" machineId=\"456\"> bar", ev.Text)
	require.EqualValues(t, dbmodel.EvError, ev.Level)
	require.NotNil(t, ev.Relations)
	require.EqualValues(t, 456, ev.Relations.MachineID)
	require.EqualValues(t, 234, ev.Relations.DaemonID)
	require.Zero(t, ev.Relations.SubnetID)
	require.Zero(t, ev.Relations.UserID)
	require.Empty(t, ev.Details)
	require.Zero(t, ev.CreatedAt)
}

// Test that the event with a daemon entry is created properly even if the
// daemon misses the machine reference.
func TestCreateEventDaemonWithoutMachine(t *testing.T) {
	// Arrange
	daemon := &dbmodel.Daemon{
		ID:        234,
		Name:      "dhcp4",
		Machine:   nil,
		MachineID: 456,
		AccessPoints: []*dbmodel.AccessPoint{{
			Type:    dbmodel.AccessPointControl,
			Address: "localhost",
			Port:    8080,
		}},
	}

	// Act
	ev := CreateEvent(dbmodel.EvError, "foo {daemon} bar", daemon)

	// Assert
	require.EqualValues(t, "foo <daemon id=\"234\" name=\"dhcp4\" machineId=\"456\"> bar", ev.Text)
	require.EqualValues(t, dbmodel.EvError, ev.Level)
	require.NotNil(t, ev.Relations)
	require.EqualValues(t, 456, ev.Relations.MachineID)
	require.EqualValues(t, 234, ev.Relations.DaemonID)
	require.Zero(t, ev.Relations.SubnetID)
	require.Zero(t, ev.Relations.UserID)
	require.Empty(t, ev.Details)
	require.Zero(t, ev.CreatedAt)
}

// Test that the event with a subnet entry is created properly.
func TestCreateEventSubnet(t *testing.T) {
	// Arrange
	subnet := &dbmodel.Subnet{
		ID:     345,
		Prefix: "192.0.0.0/8",
	}

	// Act
	ev := CreateEvent(dbmodel.EvInfo, "foo {subnet} bar", subnet)

	// Assert
	require.EqualValues(t, "foo <subnet id=\"345\" prefix=\"192.0.0.0/8\"> bar", ev.Text)
	require.EqualValues(t, dbmodel.EvInfo, ev.Level)
	require.NotNil(t, ev.Relations)
	require.Zero(t, ev.Relations.MachineID)
	require.Zero(t, ev.Relations.DaemonID)
	require.EqualValues(t, 345, ev.Relations.SubnetID)
	require.Zero(t, ev.Relations.UserID)
	require.Empty(t, ev.Details)
	require.Zero(t, ev.CreatedAt)
}

// Test that the error with a user entry is created properly.
func TestCreateEventUser(t *testing.T) {
	// Arrange
	user := &dbmodel.SystemUser{
		ID:    567,
		Login: "login",
		Email: "email",
	}

	// Act
	ev := CreateEvent(dbmodel.EvWarning, "foo {user} bar", user)

	// Assert
	require.EqualValues(t, "foo <user id=\"567\" login=\"login\" email=\"email\"> bar", ev.Text)
	require.EqualValues(t, dbmodel.EvWarning, ev.Level)
	require.NotNil(t, ev.Relations)
	require.Zero(t, ev.Relations.MachineID)
	require.Zero(t, ev.Relations.DaemonID)
	require.Zero(t, ev.Relations.SubnetID)
	require.EqualValues(t, 567, ev.Relations.UserID)
	require.Empty(t, ev.Details)
	require.Zero(t, ev.CreatedAt)
}

// Test that the error with the machine and daemon entries is created properly.
func TestCreateEventMachineAndDaemon(t *testing.T) {
	// Arrange
	machine := &dbmodel.Machine{
		ID:      123,
		Address: "foo",
		State: dbmodel.MachineState{
			Hostname: "bar",
		},
	}

	daemon := &dbmodel.Daemon{
		ID:        234,
		Name:      "dhcp4",
		Machine:   machine,
		MachineID: machine.ID,
	}

	// Act
	ev := CreateEvent(dbmodel.EvInfo, "foo {daemon} bar {machine} baz", daemon, machine)

	// Assert
	require.EqualValues(t, "foo <daemon id=\"234\" name=\"dhcp4\" machineId=\"123\"> bar <machine id=\"123\" address=\"foo\" hostname=\"bar\"> baz", ev.Text)
	require.EqualValues(t, dbmodel.EvInfo, ev.Level)
	require.NotNil(t, ev.Relations)

	require.EqualValues(t, 123, ev.Relations.MachineID)
	require.EqualValues(t, 234, ev.Relations.DaemonID)
	require.Zero(t, ev.Relations.SubnetID)
	require.Zero(t, ev.Relations.UserID)
	require.Empty(t, ev.Details)
	require.Zero(t, ev.CreatedAt)
}

// Test that the event with the machine and subnet entries is created properly.
func TestCreateEventMachineAndSubnet(t *testing.T) {
	// Arrange
	machine := &dbmodel.Machine{
		ID:      456,
		Address: "my-address",
		State: dbmodel.MachineState{
			Hostname: "my-hostname",
		},
	}

	subnet := &dbmodel.Subnet{
		ID:     345,
		Prefix: "192.0.0.0/8",
	}

	// Act
	ev := CreateEvent(dbmodel.EvError, "foo {subnet} bar {machine} baz", subnet, machine)

	// Assert
	require.EqualValues(t, "foo <subnet id=\"345\" prefix=\"192.0.0.0/8\"> bar <machine id=\"456\" address=\"my-address\" hostname=\"my-hostname\"> baz", ev.Text)
	require.EqualValues(t, dbmodel.EvError, ev.Level)
	require.NotNil(t, ev.Relations)
	require.Zero(t, ev.Relations.DaemonID)
	require.EqualValues(t, 345, ev.Relations.SubnetID)
	require.EqualValues(t, 456, ev.Relations.MachineID)
}

// Test that the event with the machine and user entries is created properly.
func TestCreateEventMachineAndUser(t *testing.T) {
	// Arrange
	machine := &dbmodel.Machine{
		ID:      123,
		Address: "my-address",
		State: dbmodel.MachineState{
			Hostname: "my-hostname",
		},
	}

	user := &dbmodel.SystemUser{
		ID:    567,
		Login: "login",
		Email: "email",
	}

	// Act
	ev := CreateEvent(dbmodel.EvWarning, "foo {machine} bar {user} baz", machine, user)

	// Assert
	require.EqualValues(t, "foo <machine id=\"123\" address=\"my-address\" hostname=\"my-hostname\"> bar <user id=\"567\" login=\"login\" email=\"email\"> baz", ev.Text)
	require.EqualValues(t, dbmodel.EvWarning, ev.Level)
	require.NotNil(t, ev.Relations)
	require.EqualValues(t, 123, ev.Relations.MachineID)
	require.Zero(t, ev.Relations.DaemonID)
	require.Zero(t, ev.Relations.SubnetID)
	require.EqualValues(t, 123, ev.Relations.MachineID)
	require.EqualValues(t, 567, ev.Relations.UserID)
}

// Test that the event with the relationships with the SSE message types.
func TestCreateEventSSEMessages(t *testing.T) {
	ev := CreateEvent(dbmodel.EvInfo, "foo bar baz", dbmodel.SSEStream("prime"), dbmodel.SSEStream("second"))

	// Assert
	require.EqualValues(t, "foo bar baz", ev.Text)
	require.EqualValues(t, dbmodel.EvInfo, ev.Level)
	require.NotNil(t, ev.Relations)
	require.Zero(t, ev.Relations.DaemonID)
	require.Zero(t, ev.Relations.SubnetID)
	require.Zero(t, ev.Relations.MachineID)
	require.Zero(t, ev.Relations.UserID)
	require.Len(t, ev.SSEStreams, 2)
	require.EqualValues(t, "prime", ev.SSEStreams[0])
	require.EqualValues(t, "second", ev.SSEStreams[1])
}

// Test that event details are set from a string.
func TestCreateEventStringDetails(t *testing.T) {
	details := "A string error"
	ev := CreateEvent(dbmodel.EvInfo, "foo bar baz", details)

	require.EqualValues(t, "foo bar baz", ev.Text)
	require.EqualValues(t, details, ev.Details)
}

// Test that event details are set from an error.
func TestCreateEventErrorDetails(t *testing.T) {
	ev := CreateEvent(dbmodel.EvInfo, "foo bar baz", pkgerrors.New("an error"))

	require.EqualValues(t, "foo bar baz", ev.Text)
	require.EqualValues(t, "an error", ev.Details)
}

// Test that event details are set from an array of errors.
func TestCreateEventErrorArrayDetails(t *testing.T) {
	errs := []error{pkgerrors.New("first error"), pkgerrors.New("second error")}
	ev := CreateEvent(dbmodel.EvInfo, "foo bar baz", errs)

	require.EqualValues(t, "foo bar baz", ev.Text)
	require.EqualValues(t, "first error; second error", ev.Details)
}

// Test that event details are set from an array of errors that contains nil.
func TestCreateEventErrorArrayWithNilDetails(t *testing.T) {
	// Arrange
	errs := []error{
		pkgerrors.New("first error"),
		nil,
		pkgerrors.New("second error"),
	}

	// Act
	ev := CreateEvent(dbmodel.EvInfo, "foo bar baz", errs)

	// Assert
	require.EqualValues(t, "foo bar baz", ev.Text)
	require.EqualValues(t, "first error; second error", ev.Details)
}

// Test that the event without tags is created properly.
func TestCreateEventNoTags(t *testing.T) {
	// Act
	ev := CreateEvent(dbmodel.EvInfo, "foo bar")

	// Assert
	require.EqualValues(t, "foo bar", ev.Text)
	require.EqualValues(t, dbmodel.EvInfo, ev.Level)
	require.Zero(t, ev.Relations.MachineID)
	require.Zero(t, ev.Relations.DaemonID)
	require.Zero(t, ev.Relations.SubnetID)
	require.Zero(t, ev.Relations.UserID)
	require.Empty(t, ev.Details)
	require.Zero(t, ev.CreatedAt)
}

// Test that the missing related object causes the tag is not resolved.
func TestCreateEventMissingObject(t *testing.T) {
	// Act
	ev := CreateEvent(dbmodel.EvInfo, "foo {machine} bar")

	// Assert
	require.EqualValues(t, "foo {machine} bar", ev.Text)
	require.EqualValues(t, dbmodel.EvInfo, ev.Level)
	require.Zero(t, ev.Relations.MachineID)
	require.Zero(t, ev.Relations.DaemonID)
	require.Zero(t, ev.Relations.SubnetID)
	require.Zero(t, ev.Relations.UserID)
	require.Empty(t, ev.Details)
	require.Zero(t, ev.CreatedAt)
}

// Test that the unknown tag is not resolved.
func TestCreateEventUnknownTag(t *testing.T) {
	// Arrange
	machine := &dbmodel.Machine{
		ID:      456,
		Address: "my-address",
		State: dbmodel.MachineState{
			Hostname: "my-hostname",
		},
	}

	// Act
	ev := CreateEvent(dbmodel.EvInfo, "foo {unknown} {machine} bar", machine)

	// Assert
	require.EqualValues(t, "foo {unknown} <machine id=\"456\" address=\"my-address\" hostname=\"my-hostname\"> bar", ev.Text)
	require.EqualValues(t, dbmodel.EvInfo, ev.Level)
	require.EqualValues(t, 456, ev.Relations.MachineID)
	require.Zero(t, ev.Relations.DaemonID)
	require.Zero(t, ev.Relations.SubnetID)
	require.Zero(t, ev.Relations.UserID)
	require.Empty(t, ev.Details)
	require.Zero(t, ev.CreatedAt)
}

// Check adding event.
func TestAddEvent(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	synctest.Test(t, func(t *testing.T) {
		ec := NewEventCenter(db)
		defer ec.Shutdown()

		subnet := &dbmodel.Subnet{
			ID: 345,
		}
		machine := &dbmodel.Machine{
			ID: 456,
		}
		daemon := &dbmodel.Daemon{
			Name:      "dhcp4",
			MachineID: machine.ID,
			Machine:   machine,
		}

		ec.AddInfoEvent("some text", daemon, machine)
		ec.AddWarningEvent("some text", subnet, machine)
		ec.AddErrorEvent("some text", daemon, machine)

		// events are stored in db in separate goroutine so it may be delay
		// so wait for it a little bit
		var events []dbmodel.Event
		var total int64
		var err error

		synctest.Wait()
		events, total, err = dbmodel.GetEventsByPage(db, 0, 10, 0, nil, nil, nil, nil, "", dbmodel.SortDirAny)

		require.NoError(t, err)
		require.EqualValues(t, 3, total)
		require.Len(t, events, 3)
		require.EqualValues(t, "some text", events[0].Text)
	})
}
