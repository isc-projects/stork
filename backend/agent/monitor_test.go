package agent

import (
	"bytes"
	"fmt"
	"path"
	"sync"
	"testing"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	gomock "go.uber.org/mock/gomock"
	"gopkg.in/h2non/gock.v1"

	"isc.org/stork/testutil"
)

//go:generate mockgen -source process.go -package=agent -destination=processmock_test.go isc.org/agent Process ProcessManager

func TestGetApps(t *testing.T) {
	am := NewAppMonitor()
	hm := NewHookManager()
	httpClient := NewHTTPClient()
	sa := NewStorkAgent("foo", 42, am, httpClient, httpClient, hm, "")
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

// Test that the reading from non existing file causes an error.
func TestReadKeaConfigNonExisting(t *testing.T) {
	// Arrange
	path := "/tmp/non-existing-path"

	// Act
	config, err := readKeaConfig(path)

	// Assert
	require.Error(t, err)
	require.Nil(t, config)
}

// Test that reading from a file with bad content causes an error.
func TestReadKeaConfigBadContent(t *testing.T) {
	// Arrange
	sb := testutil.NewSandbox()
	defer sb.Close()
	path, _ := sb.Write("config", "random content")

	// Act
	config, err := readKeaConfig(path)

	// Assert
	require.Nil(t, config)
	require.Error(t, err)
}

// Test that reading from proper file causes no error.
func TestReadKeaConfigOk(t *testing.T) {
	// Arrange
	sb := testutil.NewSandbox()
	defer sb.Close()
	configRaw := `{ "Control-agent": {
		"http-host": "host.example.org",
		"http-port": 1234
	} }`
	path, _ := sb.Write("config", configRaw)

	// Act
	config, err := readKeaConfig(path)

	// Assert
	require.NoError(t, err)
	require.NotNil(t, config)

	port, _ := config.GetHTTPPort()
	require.EqualValues(t, 1234, port)

	address, _ := config.GetHTTPHost()
	require.Equal(t, "host.example.org", address)

	require.False(t, config.UseSecureProtocol())
}

// Test that the Kea and BIND 9 apps are detected properly.
func TestDetectApps(t *testing.T) {
	// Arrange
	sb := testutil.NewSandbox()
	defer sb.Close()

	// Prepare Kea config file.
	keaConfPath, _ := sb.Write("kea-control-agent.conf", `{ "Control-agent": {
		"http-host": "localhost",
		"http-port": 45634
	} }`)

	// Prepare the command executor.
	executor := newTestCommandExecutorDefault()

	// Prepare process mocks.
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	keaProcess := NewMockProcess(ctrl)
	keaProcess.EXPECT().GetName().Return("kea-ctrl-agent", nil)
	keaProcess.EXPECT().GetCmdline().Return(fmt.Sprintf(
		"kea-ctrl-agent -c %s", keaConfPath,
	), nil)
	keaProcess.EXPECT().GetCwd().Return("/etc/kea", nil)
	keaProcess.EXPECT().GetPid().Return(int32(1234))

	bind9Process := NewMockProcess(ctrl)
	bind9Process.EXPECT().GetName().Return("named", nil)
	bind9Process.EXPECT().GetCmdline().Return("named -c /etc/named.conf", nil)
	bind9Process.EXPECT().GetCwd().Return("/etc", nil)
	bind9Process.EXPECT().GetPid().Return(int32(5678))

	unknownProcess := NewMockProcess(ctrl)
	unknownProcess.EXPECT().GetName().Return("unknown", nil)

	processManager := NewMockProcessManager(ctrl)
	processManager.EXPECT().ListProcesses().Return([]Process{
		keaProcess, bind9Process, unknownProcess,
	}, nil)

	am := &appMonitor{processManager: processManager, commander: executor}
	hm := NewHookManager()
	httpClient := NewHTTPClient()
	sa := NewStorkAgent("foo", 42, am, httpClient, httpClient, hm, "")

	// Act
	am.detectApps(sa)

	// Assert
	require.Len(t, am.apps, 2)
	require.Equal(t, AppTypeKea, am.apps[0].GetBaseApp().Type)
	require.EqualValues(t, 1234, am.apps[0].GetBaseApp().Pid)
	require.Equal(t, AppTypeBind9, am.apps[1].GetBaseApp().Type)
	require.EqualValues(t, 5678, am.apps[1].GetBaseApp().Pid)
}

// Test that the processes for which the command line cannot be read are
// not skipped.
func TestDetectAppsContinueOnNotAvailableCommandLine(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	bind9Process := NewMockProcess(ctrl)
	bind9Process.EXPECT().GetName().Return("named", nil)
	bind9Process.EXPECT().GetCmdline().Return("named -c /etc/named.conf", nil)
	bind9Process.EXPECT().GetCwd().Return("", errors.New("no current working directory"))
	bind9Process.EXPECT().GetPid().Return(int32(5678))

	processManager := NewMockProcessManager(ctrl)
	processManager.EXPECT().ListProcesses().Return([]Process{
		bind9Process,
	}, nil)

	executor := newTestCommandExecutorDefault()
	am := &appMonitor{processManager: processManager, commander: executor}
	hm := NewHookManager()
	httpClient := NewHTTPClient()
	sa := NewStorkAgent("foo", 42, am, httpClient, httpClient, hm, "")

	// Act
	am.detectApps(sa)

	// Assert
	require.Len(t, am.apps, 1)
	require.Equal(t, AppTypeBind9, am.apps[0].GetBaseApp().Type)
}

// Test that the processes for which the current working directory cannot be
// read are skipped.
func TestDetectAppsSkipOnNotAvailableCwd(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	noCwdProcess := NewMockProcess(ctrl)
	noCwdProcess.EXPECT().GetName().Return("kea-ctrl-agent", nil)
	noCwdProcess.EXPECT().GetCmdline().Return("kea-ctrl-agent -c /etc/kea/kea.conf", nil)
	noCwdProcess.EXPECT().GetCwd().Return("", errors.New("no current working directory"))

	bind9Process := NewMockProcess(ctrl)
	bind9Process.EXPECT().GetName().Return("named", nil)
	bind9Process.EXPECT().GetCmdline().Return("named -c /etc/named.conf", nil)
	bind9Process.EXPECT().GetCwd().Return("/etc", nil)
	bind9Process.EXPECT().GetPid().Return(int32(5678))

	processManager := NewMockProcessManager(ctrl)
	processManager.EXPECT().ListProcesses().Return([]Process{
		noCwdProcess, bind9Process,
	}, nil)

	executor := newTestCommandExecutorDefault()
	am := &appMonitor{processManager: processManager, commander: executor}
	hm := NewHookManager()
	httpClient := NewHTTPClient()
	sa := NewStorkAgent("foo", 42, am, httpClient, httpClient, hm, "")

	// Act
	am.detectApps(sa)

	// Assert
	require.Len(t, am.apps, 1)
	require.Equal(t, AppTypeBind9, am.apps[0].GetBaseApp().Type)
}

// The monitor periodically searches for the Kea/Bind9 instances. Usually, at
// least one application should be available. If no monitored app is found,
// the Stork prints the warning message to indicate that something unexpected
// happened.
func TestDetectAppsNoAppDetectedWarning(t *testing.T) {
	// Arrange
	// Prepare logger.
	output := logrus.StandardLogger().Out
	defer func() {
		logrus.SetOutput(output)
	}()
	var buffer bytes.Buffer
	logrus.SetOutput(&buffer)

	// Mock Stork agent.
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	processManager := NewMockProcessManager(ctrl)
	processManager.EXPECT().ListProcesses().Return([]Process{}, nil)

	executor := newTestCommandExecutorDefault()
	am := &appMonitor{processManager: processManager, commander: executor}
	hm := NewHookManager()
	httpClient := NewHTTPClient()
	sa := NewStorkAgent("foo", 42, am, httpClient, httpClient, hm, "")

	// Act
	am.detectApps(sa)

	// Assert
	require.Contains(t, buffer.String(), "No Kea nor Bind9 app detected for monitoring")
}

// Test that detectAllowedLogs does not panic when Kea server is unreachable.
func TestDetectAllowedLogsKeaUnreachable(t *testing.T) {
	am := &appMonitor{}
	httpClient := NewHTTPClient()
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
		HTTPClient: httpClient,
	})

	hm := NewHookManager()
	sa := NewStorkAgent("foo", 42, am, httpClient, httpClient, hm, "")

	require.NotPanics(t, func() { am.detectAllowedLogs(sa) })
}

// Returns a fixed output and no error for any data. The output contains the
// Bind 9 response with statistic channel details.
func newTestCommandExecutorDefault() *testCommandExecutor {
	return newTestCommandExecutor().
		addCheckConfOutput("/etc/named.conf", `key "foo" {
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
		};`).
		setConfigPathInNamedOutput("/etc/named.conf")
}

// Check BIND 9 app detection when its conf file is absolute path.
func TestDetectBind9AppAbsPath(t *testing.T) {
	// check BIND 9 app detection
	executor := newTestCommandExecutorDefault()
	app := detectBind9App([]string{"", "/dir", "-c /etc/named.conf"}, "", executor, "")
	require.NotNil(t, app)
	require.Equal(t, app.GetBaseApp().Type, AppTypeBind9)
	require.Len(t, app.GetBaseApp().AccessPoints, 2)
	point := app.GetBaseApp().AccessPoints[0]
	require.Equal(t, AccessPointControl, point.Type)
	require.Equal(t, "127.0.0.53", point.Address)
	require.EqualValues(t, 5353, point.Port)
	require.NotEmpty(t, point.Key)
	point = app.GetBaseApp().AccessPoints[1]
	require.Equal(t, AccessPointStatistics, point.Type)
	require.Equal(t, "127.0.0.80", point.Address)
	require.EqualValues(t, 80, point.Port)
	require.Empty(t, point.Key)
}

// Check BIND 9 app detection when its conf file is relative to CWD of its process.
func TestDetectBind9AppRelativePath(t *testing.T) {
	executor := newTestCommandExecutorDefault()
	app := detectBind9App([]string{"", "/dir", "-c named.conf"}, "/etc", executor, "")
	require.NotNil(t, app)
	require.Equal(t, app.GetBaseApp().Type, AppTypeBind9)
}

// Creates a basic Kea configuration file.
// Caller is responsible for remove the file.
func makeKeaConfFile() (string, func()) {
	sb := testutil.NewSandbox()
	clean := func() { sb.Close() }

	path, _ := sb.Write("config", `{ "Control-agent": {
		"http-host": "localhost",
		"http-port": 45634,
		"control-sockets": {
			"dhcp4": { },
			"dhcp6": { },
			"d2": { }
		}
	} }`)

	return path, clean
}

// Creates a basic Kea configuration file with include statement.
// It returns both inner and outer files.
// Caller is responsible for removing the files.
func makeKeaConfFileWithInclude() (string, func()) {
	sb := testutil.NewSandbox()
	// prepare kea conf file
	childPath, _ := sb.Write("child.json", `{
		"http-host": "localhost",
		"http-port": 45634
	}`)

	text := fmt.Sprintf("{ \"Control-agent\": <?include \"%s\"?> }", childPath)
	parentPath, _ := sb.Write("parent.json", text)

	return parentPath, func() { sb.Close() }
}

func TestDetectKeaApp(t *testing.T) {
	checkApp := func(app *KeaApp) {
		require.NotNil(t, app)
		require.Equal(t, AppTypeKea, app.GetBaseApp().Type)
		require.Len(t, app.GetBaseApp().AccessPoints, 1)
		ctrlPoint := app.GetBaseApp().AccessPoints[0]
		require.Equal(t, AccessPointControl, ctrlPoint.Type)
		require.Equal(t, "localhost", ctrlPoint.Address)
		require.EqualValues(t, 45634, ctrlPoint.Port)
		require.Empty(t, ctrlPoint.Key)
		require.Len(t, app.ConfiguredDaemons, 3)
	}

	httpClient := NewHTTPClient()

	t.Run("config file without include statement", func(t *testing.T) {
		tmpFilePath, clean := makeKeaConfFile()
		defer clean()

		// check kea app detection
		app := detectKeaApp([]string{"", "", tmpFilePath}, "", httpClient)
		checkApp(app)

		// check kea app detection when kea conf file is relative to CWD of kea process
		cwd, file := path.Split(tmpFilePath)
		app = detectKeaApp([]string{"", "", file}, cwd, httpClient)
		checkApp(app)
		require.Empty(t, app.ActiveDaemons)
	})

	t.Run("config file with include statement", func(t *testing.T) {
		// Check configuration with an include statement
		tmpFilePath, clean := makeKeaConfFileWithInclude()
		defer clean()

		// check kea app detection
		app := detectKeaApp([]string{"", "", tmpFilePath}, "", httpClient)
		checkApp(app)

		// check kea app detection when kea conf file is relative to CWD of kea process
		cwd, file := path.Split(tmpFilePath)
		app = detectKeaApp([]string{"", "", file}, cwd, httpClient)
		checkApp(app)
	})

	t.Run("active daemons", func(t *testing.T) {
		httpClient := NewHTTPClient()
		gock.InterceptClient(httpClient.client)

		defer gock.Off()
		gock.New("http://localhost:45634/").
			JSON(map[string]interface{}{
				"command": "version-get",
				"service": []string{"d2", "dhcp4", "dhcp6"},
			}).
			Post("/").
			Persist().
			Reply(200).
			BodyString(`[
				{ "result": 0 }, 
				{ "result": 0 },
				{ "result": 0 }
			]`)

		tmpFilePath, clean := makeKeaConfFile()
		defer clean()

		// check kea app detection
		app := detectKeaApp([]string{"", "", tmpFilePath}, "", httpClient)
		checkApp(app)
		require.Len(t, app.ActiveDaemons, 3)
	})

	t.Run("active and inactive daemons", func(t *testing.T) {
		httpClient := NewHTTPClient()
		gock.InterceptClient(httpClient.client)

		defer gock.Off()
		gock.New("http://localhost:45634/").
			JSON(map[string]interface{}{
				"command": "version-get",
				"service": []string{"d2", "dhcp4", "dhcp6"},
			}).
			Post("/").
			Persist().
			Reply(200).
			BodyString(`[
				{ "result": 1 }, 
				{ "result": 0 },
				{ "result": 1 }
			]`)

		tmpFilePath, clean := makeKeaConfFile()
		defer clean()

		// check kea app detection
		app := detectKeaApp([]string{"", "", tmpFilePath}, "", httpClient)
		checkApp(app)
		require.Len(t, app.ActiveDaemons, 1)
	})
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

// Test that the active Kea daemons are recognized properly.
func TestDetectActiveDaemons(t *testing.T) {
	// Arrange
	httpClient := NewHTTPClient()
	gock.InterceptClient(httpClient.client)

	defer gock.Off()
	gock.New("http://localhost:45634/").
		JSON(map[string]interface{}{
			"command": "version-get",
			"service": []string{"d2", "dhcp4", "dhcp6"},
		}).
		Post("/").
		Persist().
		Reply(200).
		BodyString(`[{
			"result": 1,
			"arguments": {}
		}, {
			"result": 0,
			"arguments": {}
		}, {
			"result": 0,
			"arguments": {}
		}]`)

	configPath, clean := makeKeaConfFile()
	defer clean()

	// Act
	app := detectKeaApp([]string{"", "", configPath}, "", httpClient)
	configuredDaemons := app.GetConfiguredDaemons()
	activeDaemons := app.GetActiveDaemons()

	// Assert
	require.Len(t, configuredDaemons, 3)
	require.Contains(t, configuredDaemons, "dhcp4")
	require.Contains(t, configuredDaemons, "dhcp6")
	require.Contains(t, configuredDaemons, "d2")

	require.Len(t, activeDaemons, 2)
	require.Contains(t, activeDaemons, "dhcp4")
	require.Contains(t, activeDaemons, "dhcp6")
}

// Test that the active Kea daemons are recognized properly even if a single
// daemon is provided.
func TestDetectActiveSingleDaemon(t *testing.T) {
	// Arrange
	httpClient := NewHTTPClient()
	gock.InterceptClient(httpClient.client)

	defer gock.Off()
	gock.New("http://localhost:45634/").
		JSON(map[string]interface{}{
			"command": "version-get",
			"service": []string{"dhcp4"},
		}).
		Post("/").
		Persist().
		Reply(200).
		BodyString(`[{
			"result": 0,
			"arguments": {}
		}]`)

	sb := testutil.NewSandbox()
	defer sb.Close()

	configPath, _ := sb.Write("config", `{ "Control-agent": {
		"http-host": "localhost",
		"http-port": 45634,
		"control-sockets": {
			"dhcp4": { }
		}
	} }`)

	// Act
	app := detectKeaApp([]string{"", "", configPath}, "", httpClient)
	configuredDaemons := app.GetConfiguredDaemons()
	activeDaemons := app.GetActiveDaemons()

	// Assert
	require.Len(t, configuredDaemons, 1)
	require.Contains(t, configuredDaemons, "dhcp4")
	require.Len(t, activeDaemons, 1)
	require.Contains(t, activeDaemons, "dhcp4")
}
