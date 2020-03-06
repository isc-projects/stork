package dbmodel

import (
	"context"
	"testing"

	require "github.com/stretchr/testify/require"
	//log "github.com/sirupsen/logrus"

	dbtest "isc.org/stork/server/database/test"
)

func TestAddApp(t *testing.T) {
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

	// add app but without machine, error should be raised
	s := &App{
		ID:   0,
		Type: AppTypeKea,
	}
	err = AddApp(db, s)
	require.NotNil(t, err)

	// add app but without type, error should be raised
	s = &App{
		ID:        0,
		MachineID: m.ID,
	}
	err = AddApp(db, s)
	require.NotNil(t, err)

	// add app, no error expected
	var accessPoints []*AccessPoint
	accessPoints = AppendAccessPoint(accessPoints, AccessPointControl, "cool.example.org", "", 1234)

	s = &App{
		ID:           0,
		MachineID:    m.ID,
		Type:         AppTypeKea,
		Active:       true,
		AccessPoints: accessPoints,
	}
	err = AddApp(db, s)
	require.NoError(t, err)
	require.NotZero(t, s.ID)

	// add app for the same machine and ctrl port - error should be raised
	accessPoints = []*AccessPoint{}
	accessPoints = AppendAccessPoint(accessPoints, AccessPointControl, "", "", 1234)
	s = &App{
		ID:           0,
		MachineID:    m.ID,
		Type:         AppTypeBind9,
		Active:       true,
		AccessPoints: accessPoints,
	}
	err = AddApp(db, s)
	require.NotNil(t, err)
	require.Contains(t, err.Error(), "duplicate")

	// add app with empty control address, no error expected.
	accessPoints = []*AccessPoint{}
	accessPoints = AppendAccessPoint(accessPoints, AccessPointControl, "", "abcd", 4321)
	s = &App{
		ID:           0,
		MachineID:    m.ID,
		Type:         AppTypeBind9,
		Active:       true,
		AccessPoints: accessPoints,
	}
	err = AddApp(db, s)
	require.Nil(t, err)

	// add app with two control points - error should be raised.
	accessPoints = []*AccessPoint{}
	accessPoints = AppendAccessPoint(accessPoints, AccessPointControl, "dns1.example.org", "", 5555)
	accessPoints = AppendAccessPoint(accessPoints, AccessPointControl, "dns2.example.org", "", 5656)
	s = &App{
		ID:           0,
		MachineID:    m.ID,
		Type:         AppTypeBind9,
		Active:       true,
		AccessPoints: accessPoints,
	}
	err = AddApp(db, s)
	require.NotNil(t, err)
	require.Contains(t, err.Error(), "duplicate")

	// add app with explicit access point, bad type - error should be raised.
	accessPoints = []*AccessPoint{}
	accessPoints = AppendAccessPoint(accessPoints, "foobar", "dns1.example.org", "", 6666)
	s = &App{
		ID:           0,
		MachineID:    m.ID,
		Type:         AppTypeBind9,
		Active:       true,
		AccessPoints: accessPoints,
	}
	err = AddApp(db, s)
	require.NotNil(t, err)
}

// Test that the app can be updated in the database.
func TestUpdateApp(t *testing.T) {
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

	accessPoints := []*AccessPoint{}
	accessPoints = AppendAccessPoint(accessPoints, AccessPointControl, "cool.example.org", "", 1234)
	a := &App{
		ID:           0,
		MachineID:    m.ID,
		Type:         AppTypeKea,
		Active:       true,
		AccessPoints: accessPoints,
	}

	err = UpdateApp(db, a)
	require.Error(t, err)

	err = AddApp(db, a)
	require.NoError(t, err)
	require.NotZero(t, a.ID)

	accessPoints = []*AccessPoint{}
	accessPoints = AppendAccessPoint(accessPoints, AccessPointControl, "cool.example.org", "", 2345)
	a.AccessPoints = accessPoints
	err = UpdateApp(db, a)
	require.NoError(t, err)
}

func TestDeleteApp(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// delete non-existing app
	s0 := &App{
		ID: 123,
	}
	err := DeleteApp(db, s0)
	require.Contains(t, err.Error(), "no rows in result")

	// add first machine, should be no error
	m := &Machine{
		ID:        0,
		Address:   "localhost",
		AgentPort: 8080,
	}
	err = AddMachine(db, m)
	require.NoError(t, err)
	require.NotZero(t, m.ID)

	// add app, no error expected
	var accessPoints []*AccessPoint
	accessPoints = AppendAccessPoint(accessPoints, AccessPointControl, "10.0.0.1", "", 4321)

	s := &App{
		ID:           0,
		MachineID:    m.ID,
		Type:         AppTypeKea,
		Active:       true,
		AccessPoints: accessPoints,
	}
	err = AddApp(db, s)
	require.NoError(t, err)
	require.NotZero(t, s.ID)

	// delete added app
	err = DeleteApp(db, s)
	require.NoError(t, err)
}

func TestGetAppsByMachine(t *testing.T) {
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

	// there should be no apps yet
	apps, err := GetAppsByMachine(db, m.ID)
	require.Len(t, apps, 0)
	require.NoError(t, err)

	// add app, no error expected
	var accessPoints []*AccessPoint
	accessPoints = AppendAccessPoint(accessPoints, AccessPointControl, "", "", 1234)

	s := &App{
		ID:           0,
		MachineID:    m.ID,
		Type:         AppTypeKea,
		Active:       true,
		AccessPoints: accessPoints,
	}
	err = AddApp(db, s)
	require.NoError(t, err)
	require.NotZero(t, s.ID)

	// get apps of given machine
	apps, err = GetAppsByMachine(db, m.ID)
	require.Len(t, apps, 1)
	require.NoError(t, err)
	app := apps[0]
	require.Equal(t, m.ID, app.MachineID)
	require.Equal(t, AppTypeKea, app.Type)
	// check access point
	require.Equal(t, 1, len(app.AccessPoints))
	pt := app.AccessPoints[0]
	require.Equal(t, AccessPointControl, pt.Type)
	require.Equal(t, "localhost", pt.Address)
	require.Equal(t, int64(1234), pt.Port)
	require.Empty(t, pt.Key)

	// test GetAccessPoint
	pt, err = app.GetAccessPoint(AccessPointControl)
	require.NotNil(t, pt)
	require.NoError(t, err)
	require.Equal(t, AccessPointControl, pt.Type)
	require.Equal(t, "localhost", pt.Address)
	require.Equal(t, int64(1234), pt.Port)
	require.Empty(t, pt.Key)
	// bad access point type
	pt, err = app.GetAccessPoint("foobar")
	require.Nil(t, pt)
	require.Error(t, err)
}

// Check getting apps by type only.
func TestGetAppsByType(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// add a machine
	m := &Machine{
		ID:        0,
		Address:   "localhost",
		AgentPort: 8080,
	}
	err := AddMachine(db, m)
	require.NoError(t, err)
	require.NotZero(t, m.ID)

	// add kea app
	var keaPoints []*AccessPoint
	keaPoints = AppendAccessPoint(keaPoints, AccessPointControl, "", "", 1234)
	aKea := &App{
		ID:           0,
		MachineID:    m.ID,
		Type:         AppTypeKea,
		Active:       true,
		AccessPoints: keaPoints,
	}
	err = AddApp(db, aKea)
	require.NoError(t, err)
	require.NotZero(t, aKea.ID)

	// add bind9 app
	var bind9Points []*AccessPoint
	bind9Points = AppendAccessPoint(bind9Points, AccessPointControl, "", "", 2234)
	aBind9 := &App{
		ID:           0,
		MachineID:    m.ID,
		Type:         AppTypeBind9,
		Active:       true,
		AccessPoints: bind9Points,
	}
	err = AddApp(db, aBind9)
	require.NoError(t, err)
	require.NotZero(t, aBind9.ID)

	// check getting kea apps
	apps, err := GetAppsByType(db, AppTypeKea)
	require.NoError(t, err)
	require.Len(t, apps, 1)
	require.Equal(t, aKea.ID, apps[0].ID)
	require.NotNil(t, apps[0].Machine)

	// check getting bind9 apps
	apps, err = GetAppsByType(db, AppTypeBind9)
	require.NoError(t, err)
	require.Len(t, apps, 1)
	require.Equal(t, aBind9.ID, apps[0].ID)
	require.NotNil(t, apps[0].Machine)
}

// Check getting app by its ID.
func TestGetAppByID(t *testing.T) {
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	// get non-existing app
	app, err := GetAppByID(db, 321)
	require.NoError(t, err)
	require.Nil(t, app)

	// add first machine, should be no error
	m := &Machine{
		ID:        0,
		Address:   "localhost",
		AgentPort: 8080,
	}
	err = AddMachine(db, m)
	require.NoError(t, err)
	require.NotZero(t, m.ID)

	// add app, no error expected
	var accessPoints []*AccessPoint
	accessPoints = AppendAccessPoint(accessPoints, AccessPointControl, "", "", 4444)
	accessPoints = AppendAccessPoint(accessPoints, AccessPointStatistics, "10.0.0.2", "abcd", 5555)

	s := &App{
		ID:           0,
		MachineID:    m.ID,
		Type:         AppTypeBind9,
		Active:       true,
		AccessPoints: accessPoints,
	}

	err = AddApp(db, s)
	require.NoError(t, err)
	require.NotZero(t, s.ID)

	// Get app by ID.
	app, err = GetAppByID(db, s.ID)
	require.NoError(t, err)
	require.NotNil(t, app)
	require.Equal(t, s.ID, app.ID)
	// Machine is set.
	require.Equal(t, m.ID, app.Machine.ID)
	require.Equal(t, "localhost", app.Machine.Address)
	require.Equal(t, int64(8080), app.Machine.AgentPort)
	// Check access points.
	require.Equal(t, 2, len(app.AccessPoints))

	pt := app.AccessPoints[0]
	require.Equal(t, AccessPointControl, pt.Type)
	// The control address is a special case.
	// If it is not specified it should be localhost.
	require.Equal(t, "localhost", pt.Address)
	require.Equal(t, int64(4444), pt.Port)
	require.Empty(t, pt.Key)

	pt = app.AccessPoints[1]
	require.Equal(t, AccessPointStatistics, pt.Type)
	require.Equal(t, "10.0.0.2", pt.Address)
	require.Equal(t, int64(5555), pt.Port)
	require.Equal(t, "abcd", pt.Key)

	// test GetAccessPoint
	pt, err = app.GetAccessPoint(AccessPointControl)
	require.NotNil(t, pt)
	require.NoError(t, err)
	require.Equal(t, AccessPointControl, pt.Type)
	require.Equal(t, "localhost", pt.Address)
	require.Equal(t, int64(4444), pt.Port)
	require.Empty(t, pt.Key)

	pt, err = app.GetAccessPoint(AccessPointStatistics)
	require.NotNil(t, pt)
	require.NoError(t, err)
	require.Equal(t, AccessPointStatistics, pt.Type)
	require.Equal(t, "10.0.0.2", pt.Address)
	require.Equal(t, int64(5555), pt.Port)
	require.Equal(t, "abcd", pt.Key)
}

func TestGetAppsByPage(t *testing.T) {
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

	// add kea app, no error expected
	var keaPoints []*AccessPoint
	keaPoints = AppendAccessPoint(keaPoints, AccessPointControl, "", "", 1234)

	sKea := &App{
		ID:           0,
		MachineID:    m.ID,
		Type:         AppTypeKea,
		Active:       true,
		AccessPoints: keaPoints,
	}
	err = AddApp(db, sKea)
	require.NoError(t, err)
	require.NotZero(t, sKea.ID)

	// add bind app, no error expected
	var bind9Points []*AccessPoint
	bind9Points = AppendAccessPoint(bind9Points, AccessPointControl, "", "abcd", 4321)

	sBind := &App{
		ID:           0,
		MachineID:    m.ID,
		Type:         AppTypeBind9,
		Active:       true,
		AccessPoints: bind9Points,
	}
	err = AddApp(db, sBind)
	require.NoError(t, err)
	require.NotZero(t, sBind.ID)

	// get all apps
	apps, total, err := GetAppsByPage(db, 0, 10, "", "")
	require.NoError(t, err)
	require.Len(t, apps, 2)
	require.Equal(t, int64(2), total)

	// get kea apps
	apps, total, err = GetAppsByPage(db, 0, 10, "", AppTypeKea)
	require.NoError(t, err)
	require.Len(t, apps, 1)
	require.Equal(t, int64(1), total)
	require.Equal(t, AppTypeKea, apps[0].Type)
	require.Equal(t, 1, len(apps[0].AccessPoints))
	pt := apps[0].AccessPoints[0]
	require.Equal(t, AccessPointControl, pt.Type)
	require.Equal(t, "localhost", pt.Address)
	require.Equal(t, int64(1234), pt.Port)
	require.Empty(t, pt.Key)

	// get bind apps
	apps, total, err = GetAppsByPage(db, 0, 10, "", AppTypeBind9)
	require.NoError(t, err)
	require.Len(t, apps, 1)
	require.Equal(t, int64(1), total)
	require.Equal(t, AppTypeBind9, apps[0].Type)
	require.Equal(t, 1, len(apps[0].AccessPoints))
	pt = apps[0].AccessPoints[0]
	require.Equal(t, AccessPointControl, pt.Type)
	require.Equal(t, "localhost", pt.Address)
	require.Equal(t, int64(4321), pt.Port)
	require.Equal(t, "abcd", pt.Key)
}

// Test that two names of the active DHCP daemons are returned.
func TestGetActiveDHCPMultiple(t *testing.T) {
	a := &App{
		Type: AppTypeKea,
		Details: AppKea{
			Daemons: []*KeaDaemon{
				{
					Active: true,
					Name:   "dhcp4",
				},
				{
					Active: true,
					Name:   "dhcp6",
				},
			},
		},
	}

	daemons := a.GetActiveDHCPDeamonNames()
	require.Equal(t, 2, len(daemons))
	require.Contains(t, daemons, "dhcp4")
	require.Contains(t, daemons, "dhcp6")
}

// Test that a single name of the active DHCP deamon is returned.
func TestGetActiveDHCPSingle(t *testing.T) {
	a := &App{
		Type: AppTypeKea,
		Details: AppKea{
			Daemons: []*KeaDaemon{
				{
					Active: false,
					Name:   "dhcp4",
				},
				{
					Active: true,
					Name:   "dhcp6",
				},
			},
		},
	}
	daemons := a.GetActiveDHCPDeamonNames()
	require.Equal(t, 1, len(daemons))
	require.NotContains(t, daemons, "dhcp4")
	require.Contains(t, daemons, "dhcp6")
}

// Test that empty list of deamons is returned if the application type
// is not Kea.
func TestGetActiveDHCPAppMismatch(t *testing.T) {
	a := &App{
		Type:    AppTypeKea,
		Details: AppBind9{},
	}
	daemons := a.GetActiveDHCPDeamonNames()
	require.Empty(t, daemons)
}

func TestGetAllApps(t *testing.T) {
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

	// add kea app, no error expected
	var keaPoints []*AccessPoint
	keaPoints = AppendAccessPoint(keaPoints, AccessPointControl, "", "", 1234)

	aKea := &App{
		ID:           0,
		MachineID:    m.ID,
		Type:         AppTypeKea,
		Active:       true,
		AccessPoints: keaPoints,
	}
	err = AddApp(db, aKea)
	require.NoError(t, err)
	require.NotZero(t, aKea.ID)

	// add bind app, no error expected
	var bind9Points []*AccessPoint
	bind9Points = AppendAccessPoint(bind9Points, AccessPointControl, "", "abcd", 4321)

	aBind := &App{
		ID:           0,
		MachineID:    m.ID,
		Type:         AppTypeBind9,
		Active:       true,
		AccessPoints: bind9Points,
	}
	err = AddApp(db, aBind)
	require.NoError(t, err)
	require.NotZero(t, aBind.ID)

	// get all apps
	apps, err := GetAllApps(db)
	require.NoError(t, err)
	require.Len(t, apps, 2)
	require.True(t, apps[0].Type == AppTypeKea || apps[1].Type == AppTypeKea)
	require.True(t, apps[0].Type == AppTypeBind9 || apps[1].Type == AppTypeBind9)
}

func TestAfterScanKea(t *testing.T) {
	ctx := context.Background()

	// for now details are nil
	var accessPoints []*AccessPoint
	accessPoints = AppendAccessPoint(accessPoints, AccessPointControl, "", "", 1234)

	aKea := &App{
		ID:           0,
		MachineID:    0,
		Type:         AppTypeKea,
		Active:       true,
		AccessPoints: accessPoints,
	}
	err := aKea.AfterScan(ctx)
	require.Nil(t, err)
	require.Nil(t, aKea.Details)

	// add some details
	aKea.Details = map[string]interface{}{
		"ExtendedVersion": "1.2.3",
		"Daemons": []map[string]interface{}{
			{
				"Pid":  123,
				"Name": "dhcp4",
			},
		},
	}
	err = aKea.AfterScan(ctx)
	require.Nil(t, err)
	require.NotNil(t, aKea.Details)
	require.Equal(t, "1.2.3", aKea.Details.(AppKea).ExtendedVersion)
	require.Equal(t, "dhcp4", aKea.Details.(AppKea).Daemons[0].Name)
}

func TestAfterScanBind(t *testing.T) {
	ctx := context.Background()

	// for now details are nil
	var accessPoints []*AccessPoint
	accessPoints = AppendAccessPoint(accessPoints, AccessPointControl, "", "abcd", 4321)

	aBind := &App{
		ID:           0,
		MachineID:    0,
		Type:         AppTypeBind9,
		Active:       true,
		AccessPoints: accessPoints,
	}
	err := aBind.AfterScan(ctx)
	require.Nil(t, err)
	require.Nil(t, aBind.Details)

	// add some details
	aBind.Details = map[string]interface{}{
		"Daemon": map[string]interface{}{
			"Pid":  123,
			"Name": "named",
		},
	}
	err = aBind.AfterScan(ctx)
	require.Nil(t, err)
	require.NotNil(t, aBind.Details)
	require.Equal(t, "named", aBind.Details.(AppBind9).Daemon.Name)
	require.Equal(t, int32(123), aBind.Details.(AppBind9).Daemon.Pid)
}

// Test that local subnet id of the Kea subnet can be extracted.
func TestGetLocalSubnetID(t *testing.T) {
	ctx := context.Background()

	accessPoints := []*AccessPoint{}
	accessPoints = AppendAccessPoint(accessPoints, AccessPointControl, "", "", 1234)
	aKea := &App{
		ID:           0,
		MachineID:    0,
		Type:         AppTypeKea,
		Active:       true,
		AccessPoints: accessPoints,
	}

	// Add a DHCPv4 daemon with a simple configuration comprising a single subnet.
	aKea.Details = map[string]interface{}{
		"Daemons": []map[string]interface{}{
			{
				"Name": "dhcp4",
				"Config": &map[string]interface{}{
					"Dhcp4": map[string]interface{}{
						"subnet4": []map[string]interface{}{
							{
								"id":     1,
								"subnet": "192.0.2.0/24",
							},
						},
					},
				},
			},
		},
	}

	err := aKea.AfterScan(ctx)
	require.Nil(t, err)
	require.NotNil(t, aKea.Details)

	// Try to find a non-existing subnet.
	require.Zero(t, aKea.GetLocalSubnetID("192.0.3.0/24"))
	// Next, try to find the existing subnet.
	require.EqualValues(t, 1, aKea.GetLocalSubnetID("192.0.2.0/24"))
}
