package eventcenter

import (
	"testing"
	"time"

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
	require.Zero(t, ev.Relations.AppID)
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
	require.Zero(t, ev.Relations.AppID)
	require.Zero(t, ev.Relations.DaemonID)
	require.Zero(t, ev.Relations.SubnetID)
	require.Zero(t, ev.Relations.UserID)
	require.Empty(t, ev.Details)
	require.Zero(t, ev.CreatedAt)
}

// Test that the event with an app entry is created properly.
func TestCreateEventApp(t *testing.T) {
	// Arrange
	app := &dbmodel.App{
		ID:   123,
		Type: dbmodel.AppTypeKea,
		Name: "dhcp-server",
		Meta: dbmodel.AppMeta{
			Version: "1.2.3",
		},
		MachineID: 456,
	}

	// Act
	ev := CreateEvent(dbmodel.EvWarning, "foo {app} bar", app)

	// Assert
	require.EqualValues(t, ev.Text, "foo <app id=\"123\" name=\"dhcp-server\" type=\"kea\" version=\"1.2.3\"> bar")
	require.EqualValues(t, dbmodel.EvWarning, ev.Level)
	require.NotNil(t, ev.Relations)
	require.EqualValues(t, 456, ev.Relations.MachineID)
	require.EqualValues(t, 123, ev.Relations.AppID)
	require.Zero(t, ev.Relations.DaemonID)
	require.Zero(t, ev.Relations.SubnetID)
	require.Zero(t, ev.Relations.UserID)
	require.Empty(t, ev.Details)
	require.Zero(t, ev.CreatedAt)
}

// Test that missing app meta doesn't cause problems.
func TestCreateEventAppWithoutMeta(t *testing.T) {
	// Arrange
	app := &dbmodel.App{
		ID:        123,
		Type:      dbmodel.AppTypeKea,
		Name:      "dhcp-server",
		MachineID: 456,
	}

	// Act
	ev := CreateEvent(dbmodel.EvWarning, "foo {app} bar", app)

	// Assert
	require.EqualValues(t, ev.Text, "foo <app id=\"123\" name=\"dhcp-server\" type=\"kea\" version=\"\"> bar")
	require.EqualValues(t, dbmodel.EvWarning, ev.Level)
	require.NotNil(t, ev.Relations)
	require.EqualValues(t, 456, ev.Relations.MachineID)
	require.EqualValues(t, 123, ev.Relations.AppID)
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
		ID:   234,
		Name: "dhcp4",
		App: &dbmodel.App{
			ID:        123,
			MachineID: 456,
			Type:      dbmodel.AppTypeKea,
		},
		AppID: 123,
	}

	// Act
	ev := CreateEvent(dbmodel.EvError, "foo {daemon} bar", daemon)

	// Assert
	require.EqualValues(t, "foo <daemon id=\"234\" name=\"dhcp4\" appId=\"123\" appType=\"kea\"> bar", ev.Text)
	require.EqualValues(t, dbmodel.EvError, ev.Level)
	require.NotNil(t, ev.Relations)
	require.EqualValues(t, 456, ev.Relations.MachineID)
	require.EqualValues(t, 123, ev.Relations.AppID)
	require.EqualValues(t, 234, ev.Relations.DaemonID)
	require.Zero(t, ev.Relations.SubnetID)
	require.Zero(t, ev.Relations.UserID)
	require.Empty(t, ev.Details)
	require.Zero(t, ev.CreatedAt)
}

// Test that the event with a daemon entry is created properly even if the
// daemon misses the app reference.
func TestCreateEventDaemonWithoutApp(t *testing.T) {
	// Arrange
	daemon := &dbmodel.Daemon{
		ID:    234,
		Name:  "dhcp4",
		App:   nil,
		AppID: 123,
	}

	// Act
	ev := CreateEvent(dbmodel.EvError, "foo {daemon} bar", daemon)

	// Assert
	require.EqualValues(t, "foo <daemon id=\"234\" name=\"dhcp4\" appId=\"123\" appType=\"\"> bar", ev.Text)
	require.EqualValues(t, dbmodel.EvError, ev.Level)
	require.NotNil(t, ev.Relations)
	require.Zero(t, ev.Relations.MachineID)
	require.EqualValues(t, 123, ev.Relations.AppID)
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
	require.Zero(t, ev.Relations.AppID)
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
	require.Zero(t, ev.Relations.AppID)
	require.Zero(t, ev.Relations.DaemonID)
	require.Zero(t, ev.Relations.SubnetID)
	require.EqualValues(t, 567, ev.Relations.UserID)
	require.Empty(t, ev.Details)
	require.Zero(t, ev.CreatedAt)
}

// Test that the error with the app and daemon entries is created properly.
func TestCreateEventAppAndDaemon(t *testing.T) {
	// Arrange
	app := &dbmodel.App{
		ID:   123,
		Type: dbmodel.AppTypeKea,
		Name: "dhcp-server",
		Meta: dbmodel.AppMeta{
			Version: "1.2.3",
		},
		MachineID: 456,
	}

	daemon := &dbmodel.Daemon{
		ID:    234,
		Name:  "dhcp4",
		App:   app,
		AppID: app.ID,
	}

	// Act
	ev := CreateEvent(dbmodel.EvInfo, "foo {daemon} bar {app} baz", daemon, app)

	// Assert
	require.EqualValues(t, "foo <daemon id=\"234\" name=\"dhcp4\" appId=\"123\" appType=\"kea\"> bar <app id=\"123\" name=\"dhcp-server\" type=\"kea\" version=\"1.2.3\"> baz", ev.Text)
	require.EqualValues(t, dbmodel.EvInfo, ev.Level)
	require.NotNil(t, ev.Relations)

	require.EqualValues(t, 456, ev.Relations.MachineID)
	require.EqualValues(t, 123, ev.Relations.AppID)
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
	require.Zero(t, ev.Relations.AppID)
	require.Zero(t, ev.Relations.DaemonID)
	require.EqualValues(t, 345, ev.Relations.SubnetID)
	require.EqualValues(t, 456, ev.Relations.MachineID)
}

// Test that the event with the app and user entries is created properly.
func TestCreateEventAppAndUser(t *testing.T) {
	// Arrange
	app := &dbmodel.App{
		ID:        123,
		Type:      dbmodel.AppTypeKea,
		Name:      "dhcp-server",
		MachineID: 456,
	}

	user := &dbmodel.SystemUser{
		ID:    567,
		Login: "login",
		Email: "email",
	}

	// Act
	ev := CreateEvent(dbmodel.EvWarning, "foo {app} bar {user} baz", app, user)

	// Assert
	require.EqualValues(t, "foo <app id=\"123\" name=\"dhcp-server\" type=\"kea\" version=\"\"> bar <user id=\"567\" login=\"login\" email=\"email\"> baz", ev.Text)
	require.EqualValues(t, dbmodel.EvWarning, ev.Level)
	require.NotNil(t, ev.Relations)
	require.EqualValues(t, 123, ev.Relations.AppID)
	require.Zero(t, ev.Relations.DaemonID)
	require.Zero(t, ev.Relations.SubnetID)
	require.EqualValues(t, 456, ev.Relations.MachineID)
	require.EqualValues(t, 567, ev.Relations.UserID)
}

// Test that the event with the relationships with the SSE message types.
func TestCreateEventSSEMessages(t *testing.T) {
	ev := CreateEvent(dbmodel.EvInfo, "foo bar baz", dbmodel.SSEStream("prime"), dbmodel.SSEStream("second"))

	// Assert
	require.EqualValues(t, "foo bar baz", ev.Text)
	require.EqualValues(t, dbmodel.EvInfo, ev.Level)
	require.NotNil(t, ev.Relations)
	require.Zero(t, ev.Relations.AppID)
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
	require.Zero(t, ev.Relations.AppID)
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
	require.Zero(t, ev.Relations.AppID)
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
	require.Zero(t, ev.Relations.AppID)
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

	ec := NewEventCenter(db)

	app := &dbmodel.App{
		ID:   123,
		Type: dbmodel.AppTypeKea,
		Meta: dbmodel.AppMeta{
			Version: "1.2.3",
		},
	}
	daemon := &dbmodel.Daemon{
		Name:  "dhcp4",
		App:   app,
		AppID: app.ID,
	}
	subnet := &dbmodel.Subnet{
		ID: 345,
	}
	machine := &dbmodel.Machine{
		ID: 456,
	}

	ec.AddInfoEvent("some text", daemon, app)
	ec.AddWarningEvent("some text", subnet, app)
	ec.AddErrorEvent("some text", daemon, machine)

	// events are stored in db in separate goroutine so it may be delay
	// so wait for it a little bit
	var events []dbmodel.Event
	var total int64
	var err error

	require.Eventually(t, func() bool {
		events, total, err = dbmodel.GetEventsByPage(db, 0, 10, 0, nil, nil, nil, nil, "", dbmodel.SortDirAny)
		return total >= 3
	}, time.Second, 10*time.Millisecond)

	require.NoError(t, err)
	require.EqualValues(t, 3, total)
	require.Len(t, events, 3)
	require.EqualValues(t, "some text", events[0].Text)
}
