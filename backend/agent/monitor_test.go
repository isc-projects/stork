package agent

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"path"
	"sync"
	"testing"

	pkgerrors "github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v2"

	"isc.org/stork/testutil"
)

func TestGetApps(t *testing.T) {
	am := NewAppMonitor()
	settings := cli.NewContext(nil, flag.NewFlagSet("", 0), nil)
	sa := NewStorkAgent(settings, am)
	am.Start(sa)
	apps := am.GetApps()
	require.Len(t, apps, 0)
	am.Shutdown()
}

// Check if detected apps are returned by GetApp.
func TestGetApp(t *testing.T) {
	am := NewAppMonitor()

	var apps []App
	apps = append(apps, &KeaApp{
		BaseApp: BaseApp{
			Type:         AppTypeKea,
			AccessPoints: makeAccessPoint(AccessPointControl, "1.2.3.1", "", 1234, true),
		},
		HTTPClient: nil,
	})

	accessPoints := makeAccessPoint(AccessPointControl, "2.3.4.4", "abcd", 2345, false)
	accessPoints = append(accessPoints, AccessPoint{
		Type:    AccessPointStatistics,
		Address: "2.3.4.5",
		Port:    2346,
		Key:     "",
	})

	apps = append(apps, &Bind9App{
		BaseApp: BaseApp{
			Type:         AppTypeBind9,
			AccessPoints: accessPoints,
		},
	})

	// Monitor holds apps in background goroutine. So to get apps we need
	// to send a request over a channel to this goroutine and wait for
	// a response with detected apps. We do not want to spawn monitor background
	// goroutine so we are calling GetApp in our background goroutine
	// and are serving this request in the main thread.
	// To make it in sync the wait group is used here.
	var wg sync.WaitGroup

	// find kea app
	wg.Add(1)
	go func() {
		defer wg.Done()
		app := am.GetApp(AppTypeKea, AccessPointControl, "1.2.3.1", 1234)
		require.NotNil(t, app)
		require.EqualValues(t, AppTypeKea, app.GetBaseApp().Type)
	}()
	ret := <-am.(*appMonitor).requests
	ret <- apps
	wg.Wait()

	// find bind app
	wg.Add(1) // expect 1 Done in the wait group
	go func() {
		defer wg.Done()
		app := am.GetApp(AppTypeBind9, AccessPointControl, "2.3.4.4", 2345)
		require.NotNil(t, app)
		require.EqualValues(t, AppTypeBind9, app.GetBaseApp().Type)
	}()
	ret = <-am.(*appMonitor).requests
	ret <- apps
	wg.Wait()

	// find not existing app - should return nil
	wg.Add(1) // expect 1 Done in the wait group
	go func() {
		defer wg.Done()
		app := am.GetApp(AppTypeKea, AccessPointControl, "0.0.0.0", 1)
		require.Nil(t, app)
	}()
	ret = <-am.(*appMonitor).requests
	ret <- apps
	wg.Wait()
}

func TestGetCtrlAddressFromKeaConfigNonExisting(t *testing.T) {
	// check reading from non existing file
	path := "/tmp/non-existing-path"
	address, port, useSecureProtocol := getCtrlTargetFromKeaConfig(path)
	require.Zero(t, port)
	require.Empty(t, address)
	require.False(t, useSecureProtocol)
}

func TestGetCtrlFromKeaConfigBadContent(t *testing.T) {
	// prepare kea conf file
	tmpFile, err := os.CreateTemp(os.TempDir(), "prefix-")
	require.NoError(t, err)

	defer os.Remove(tmpFile.Name())

	text := []byte("random content")
	_, err = tmpFile.Write(text)
	require.NoError(t, err)

	err = tmpFile.Close()
	require.NoError(t, err)

	// check reading from prepared file with bad content
	// so 0 should be returned as port
	address, port, useSecureProtocol := getCtrlTargetFromKeaConfig(tmpFile.Name())
	require.Zero(t, port)
	require.Empty(t, address)
	require.False(t, useSecureProtocol)
}

func TestGetCtrlAddressFromKeaConfigOk(t *testing.T) {
	// prepare kea conf file
	tmpFile, err := os.CreateTemp(os.TempDir(), "prefix-")
	require.NoError(t, err)

	defer os.Remove(tmpFile.Name())

	text := []byte(`{ "Control-agent": {
		"http-host": "host.example.org",
		"http-port": 1234
	} }`)
	_, err = tmpFile.Write(text)
	require.NoError(t, err)

	err = tmpFile.Close()
	require.NoError(t, err)

	// check reading from proper file
	address, port, useSecureProtocol := getCtrlTargetFromKeaConfig(tmpFile.Name())
	require.EqualValues(t, 1234, port)
	require.Equal(t, "host.example.org", address)
	require.False(t, useSecureProtocol)
}

func TestGetCtrlAddressFromKeaConfigAddress0000(t *testing.T) {
	// prepare kea conf file
	tmpFile, err := os.CreateTemp(os.TempDir(), "prefix-")
	require.NoError(t, err)

	defer os.Remove(tmpFile.Name())

	text := []byte(`{ "Control-agent": {
		"http-host": "0.0.0.0",
		"http-port": 1234
	} }`)
	_, err = tmpFile.Write(text)
	require.NoError(t, err)

	err = tmpFile.Close()
	require.NoError(t, err)

	// check reading from proper file;
	// if CA is listening on 0.0.0.0 then 127.0.0.1 should be returned
	// as it is not possible to connect to 0.0.0.0
	address, port, useSecureProtocol := getCtrlTargetFromKeaConfig(tmpFile.Name())
	require.EqualValues(t, 1234, port)
	require.Equal(t, "127.0.0.1", address)
	require.False(t, useSecureProtocol)
}

func TestGetCtrlAddressFromKeaConfigAddressColons(t *testing.T) {
	// prepare kea conf file
	tmpFile, err := os.CreateTemp(os.TempDir(), "prefix-")
	require.NoError(t, err)

	defer os.Remove(tmpFile.Name())

	text := []byte(`{ "Control-agent": {
		"http-host": "::",
		"http-port": 1234
	} }`)
	_, err = tmpFile.Write(text)
	require.NoError(t, err)

	err = tmpFile.Close()
	require.NoError(t, err)

	// check reading from proper file;
	// if CA is listening on :: then ::1 should be returned
	// as it is not possible to connect to ::
	address, port, useSecureProtocol := getCtrlTargetFromKeaConfig(tmpFile.Name())
	require.EqualValues(t, 1234, port)
	require.Equal(t, "::1", address)
	require.False(t, useSecureProtocol)
}

func TestDetectApps(t *testing.T) {
	am := &appMonitor{}
	settings := cli.NewContext(nil, flag.NewFlagSet("", 0), nil)
	sa := NewStorkAgent(settings, am)
	am.detectApps(sa)
}

// Test that detectAllowedLogs does not panic when Kea server is unreachable.
func TestDetectAllowedLogsKeaUnreachable(t *testing.T) {
	am := &appMonitor{}
	am.apps = append(am.apps, &KeaApp{
		BaseApp: BaseApp{
			Type: AppTypeKea,
			AccessPoints: []AccessPoint{
				{
					Type:    AccessPointControl,
					Address: "localhost",
					Port:    45678,
				},
			},
		},
		HTTPClient: NewHTTPClient(false),
	})

	settings := cli.NewContext(nil, flag.NewFlagSet("", 0), nil)
	sa := NewStorkAgent(settings, am)

	require.NotPanics(t, func() { am.detectAllowedLogs(sa) })
}

type testCommandExecutor struct{}

// Returns a fixed output and no error for any data. The output contains the
// Bind 9 response with statistic channel details.
func (e *testCommandExecutor) Output(command string, args ...string) ([]byte, error) {
	text := `keys "foo" {
                      algorithm "hmac-sha256";
                      secret "abcd";
                 };
	         controls {
                      inet 127.0.0.53 port 5353 allow { localhost; } keys { "foo"; "bar"; };
                      inet * port 5454 allow { localhost; 1.2.3.4; };
                 };
                 statistics-channels {
                      inet 127.0.0.80 port 80 allow { localhost; 1.2.3.4; };
                      inet 127.0.0.88 port 88 allow { localhost; 1.2.3.4; };
                 };`

	return []byte(text), nil
}

// Check BIND 9 app detection when its conf file is absolute path.
func TestDetectBind9AppAbsPath(t *testing.T) {
	sb := testutil.NewSandbox()
	defer sb.Close()

	// check BIND 9 app detection
	executor := &testCommandExecutor{}
	cfgPath, err := sb.Join("etc/path.cfg")
	require.NoError(t, err)
	namedDir, err := sb.JoinDir("usr/sbin")
	require.NoError(t, err)
	_, err = sb.Join("usr/bin/named-checkconf")
	require.NoError(t, err)
	_, err = sb.Join("usr/sbin/rndc")
	require.NoError(t, err)
	app := detectBind9App([]string{"", namedDir, fmt.Sprintf("-c %s", cfgPath)}, "", executor)
	require.NotNil(t, app)
	require.Equal(t, app.GetBaseApp().Type, AppTypeBind9)
	require.Len(t, app.GetBaseApp().AccessPoints, 2)
	point := app.GetBaseApp().AccessPoints[0]
	require.Equal(t, AccessPointControl, point.Type)
	require.Equal(t, "127.0.0.53", point.Address)
	require.EqualValues(t, 5353, point.Port)
	point = app.GetBaseApp().AccessPoints[1]
	require.Equal(t, AccessPointStatistics, point.Type)
	require.Equal(t, "127.0.0.80", point.Address)
	require.EqualValues(t, 80, point.Port)
	require.Empty(t, point.Key)
}

// Check BIND 9 app detection when its conf file is relative to CWD of its process.
func TestDetectBind9AppRelativePath(t *testing.T) {
	sb := testutil.NewSandbox()
	defer sb.Close()

	cmdr := &testCommandExecutor{}
	sb.Join("etc/path.cfg")
	cfgDir, err := sb.JoinDir("etc")
	require.NoError(t, err)
	namedDir, err := sb.JoinDir("usr/sbin")
	require.NoError(t, err)
	_, err = sb.Join("usr/sbin/named-checkconf")
	require.NoError(t, err)
	_, err = sb.Join("usr/bin/rndc")
	require.NoError(t, err)
	app := detectBind9App([]string{"", namedDir, "-c path.cfg"}, cfgDir, cmdr)
	require.NotNil(t, app)
	require.Equal(t, app.GetBaseApp().Type, AppTypeBind9)
}

// Creates a basic Kea configuration file.
// Caller is responsible for remove the file.
func makeKeaConfFile() (*os.File, error) {
	// prepare kea conf file
	file, err := os.CreateTemp(os.TempDir(), "prefix-")
	if err != nil {
		return nil, pkgerrors.Wrap(err, "cannot create temporary file")
	}

	text := []byte(`{ "Control-agent": {
		"http-host": "localhost",
		"http-port": 45634
	} }`)
	if _, err = file.Write(text); err != nil {
		return nil, pkgerrors.Wrap(err, "failed to write to temporary file")
	}
	if err := file.Close(); err != nil {
		return nil, pkgerrors.Wrap(err, "failed to close a temporary file")
	}

	return file, nil
}

// Creates a basic Kea configuration file with include statement.
// It returns both inner and outer files.
// Caller is responsible for removing the files.
func makeKeaConfFileWithInclude() (parentConfig *os.File, childConfig *os.File, err error) {
	// prepare kea conf file
	parentConfig, err = os.CreateTemp(os.TempDir(), "prefix-*.json")

	if err != nil {
		return nil, nil, pkgerrors.Wrap(err, "cannot create temporary file for parent config")
	}

	childConfig, err = os.CreateTemp(os.TempDir(), "prefix-*.json")
	if err != nil {
		return nil, nil, pkgerrors.Wrap(err, "cannot create temporary file for child config")
	}

	text := []byte(`{
		"http-host": "localhost",
		"http-port": 45634
	}`)

	if _, err = childConfig.Write(text); err != nil {
		return nil, nil, pkgerrors.Wrap(err, "failed to write to temporary file")
	}
	if err := childConfig.Close(); err != nil {
		return nil, nil, pkgerrors.Wrap(err, "failed to close to temporary file")
	}

	text = []byte(fmt.Sprintf("{ \"Control-agent\": <?include \"%s\"?> }", childConfig.Name()))
	if _, err = parentConfig.Write(text); err != nil {
		return nil, nil, pkgerrors.Wrap(err, "failed to write to temporary file")
	}
	if err := parentConfig.Close(); err != nil {
		return nil, nil, pkgerrors.Wrap(err, "failed to close to temporary file")
	}

	return parentConfig, childConfig, nil
}

func TestDetectKeaApp(t *testing.T) {
	tmpFile, err := makeKeaConfFile()
	require.NoError(t, err)
	tmpFilePath := tmpFile.Name()
	defer os.Remove(tmpFilePath)

	checkApp := func(app App) {
		require.NotNil(t, app)
		require.Equal(t, AppTypeKea, app.GetBaseApp().Type)
		require.Len(t, app.GetBaseApp().AccessPoints, 1)
		ctrlPoint := app.GetBaseApp().AccessPoints[0]
		require.Equal(t, AccessPointControl, ctrlPoint.Type)
		require.Equal(t, "localhost", ctrlPoint.Address)
		require.EqualValues(t, 45634, ctrlPoint.Port)
		require.Empty(t, ctrlPoint.Key)
	}

	httpClient := NewHTTPClient(false)

	// check kea app detection
	app := detectKeaApp([]string{"", "", tmpFilePath}, "", httpClient)
	checkApp(app)

	// check kea app detection when kea conf file is relative to CWD of kea process
	cwd, file := path.Split(tmpFilePath)
	app = detectKeaApp([]string{"", "", file}, cwd, httpClient)
	checkApp(app)

	// Check configuration with an include statement
	tmpFile, nestedFile, err := makeKeaConfFileWithInclude()
	require.NoError(t, err)
	tmpFilePath = tmpFile.Name()
	defer os.Remove(tmpFilePath)
	defer os.Remove(nestedFile.Name())

	// check kea app detection
	app = detectKeaApp([]string{"", "", tmpFilePath}, "", httpClient)
	checkApp(app)

	// check kea app detection when kea conf file is relative to CWD of kea process
	cwd, file = path.Split(tmpFilePath)
	app = detectKeaApp([]string{"", "", file}, cwd, httpClient)
	checkApp(app)
}

func TestGetAccessPoint(t *testing.T) {
	bind9App := &Bind9App{
		BaseApp: BaseApp{
			Type: AppTypeBind9,
			AccessPoints: []AccessPoint{
				{
					Type:    AccessPointControl,
					Address: "127.0.0.53",
					Port:    int64(5353),
					Key:     "hmac-sha256:abcd",
				},
				{
					Type:    AccessPointStatistics,
					Address: "127.0.0.80",
					Port:    int64(80),
					Key:     "",
				},
			},
		},
		RndcClient: nil,
	}

	keaApp := &KeaApp{
		BaseApp: BaseApp{
			Type: AppTypeKea,
			AccessPoints: []AccessPoint{
				{
					Type:    AccessPointControl,
					Address: "localhost",
					Port:    int64(45634),
					Key:     "",
				},
			},
		},
		HTTPClient: nil,
	}

	// test get bind 9 access points
	point, err := getAccessPoint(bind9App, AccessPointControl)
	require.NotNil(t, point)
	require.NoError(t, err)
	require.Equal(t, AccessPointControl, point.Type)
	require.Equal(t, "127.0.0.53", point.Address)
	require.EqualValues(t, 5353, point.Port)
	require.Equal(t, "hmac-sha256:abcd", point.Key)

	point, err = getAccessPoint(bind9App, AccessPointStatistics)
	require.NotNil(t, point)
	require.NoError(t, err)
	require.Equal(t, AccessPointStatistics, point.Type)
	require.Equal(t, "127.0.0.80", point.Address)
	require.EqualValues(t, 80, point.Port)
	require.Empty(t, point.Key)

	// test get kea access points
	point, err = getAccessPoint(keaApp, AccessPointControl)
	require.NotNil(t, point)
	require.NoError(t, err)
	require.Equal(t, AccessPointControl, point.Type)
	require.Equal(t, "localhost", point.Address)
	require.EqualValues(t, 45634, point.Port)
	require.Empty(t, point.Key)

	point, err = getAccessPoint(keaApp, AccessPointStatistics)
	require.Error(t, err)
	require.Nil(t, point)
}

func TestPrintNewOrUpdatedApps(t *testing.T) {
	bind9App := &Bind9App{
		BaseApp: BaseApp{
			Type: AppTypeBind9,
			AccessPoints: []AccessPoint{
				{
					Type:    AccessPointControl,
					Address: "127.0.0.53",
					Port:    int64(5353),
					Key:     "hmac-sha256:abcd",
				},
				{
					Type:    AccessPointStatistics,
					Address: "127.0.0.80",
					Port:    int64(80),
					Key:     "",
				},
			},
		},
		RndcClient: nil,
	}

	keaApp := &KeaApp{
		BaseApp: BaseApp{
			Type: AppTypeKea,
			AccessPoints: []AccessPoint{
				{
					Type:    AccessPointControl,
					Address: "localhost",
					Port:    int64(45634),
					Key:     "",
				},
			},
		},
		HTTPClient: nil,
	}

	newApps := []App{bind9App, keaApp}
	var oldApps []App

	printNewOrUpdatedApps(newApps, oldApps)
}

// The monitor periodically searches for the Kea/Bind9 instances. Usually, at
// least one application should be available. If no monitored app is found,
// the Stork prints the warning message to indicate that something unexpected
// happened.
func TestPrintNewOrUpdatedAppsNoAppDetectedWarning(t *testing.T) {
	// Arrange
	output := logrus.StandardLogger().Out
	defer func() {
		logrus.SetOutput(output)
	}()
	var buffer bytes.Buffer
	logrus.SetOutput(&buffer)

	// Act
	printNewOrUpdatedApps([]App{}, []App{})

	// Assert
	require.Contains(t, buffer.String(), "No Kea nor Bind9 app detected for monitoring")
}
