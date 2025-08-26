package agent

import (
	"bytes"
	"fmt"
	"path"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	gomock "go.uber.org/mock/gomock"
	"gopkg.in/h2non/gock.v1"

	bind9config "isc.org/stork/appcfg/bind9"
	pdnsconfig "isc.org/stork/appcfg/pdns"
	"isc.org/stork/testutil"
)

//go:generate mockgen -source process.go -package=agent -destination=processmock_test.go -mock_names=processLister=MockProcessLister,supportedProcess=MockSupportedProcess isc.org/agent supportedProcess processLister
//go:generate mockgen -source monitor.go -package=agent -destination=monitormock_test.go isc.org/agent App

const defaultBind9Config = `
	key "foo" {
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

const bind9ConfigWithoutStatistics = `
	key "foo" {
		algorithm "hmac-sha256";
		secret "abcd";
	};
	controls {
		inet 127.0.0.53 port 5353 allow { localhost; } keys { "foo"; "bar"; };
		inet * port 5454 allow { localhost; 1.2.3.4; };
	};`

const defaultPDNSConfig = `
	api
	webserver
	webserver-port=8081
	webserver-address=127.0.0.1
	api-key=stork
`

func TestGetApps(t *testing.T) {
	am := NewAppMonitor()
	hm := NewHookManager()
	bind9StatsClient := NewBind9StatsClient()
	httpClientConfig := HTTPClientConfig{}
	sa := NewStorkAgent("foo", 42, am, bind9StatsClient, httpClientConfig, hm, "", "")
	am.Start(sa)
	apps := am.GetApps()
	require.Len(t, apps, 0)
	am.Shutdown()
}

// Check if detected apps are returned by GetApp.
func TestGetApp(t *testing.T) {
	am := NewAppMonitor()

	apps := []App{
		&KeaApp{
			BaseApp: BaseApp{
				Type:         AppTypeKea,
				AccessPoints: makeAccessPoint(AccessPointControl, "1.2.3.1", "", 1234, true),
			},
			HTTPClient: nil,
		},
		&Bind9App{
			BaseApp: BaseApp{
				Type: AppTypeBind9,
				AccessPoints: []AccessPoint{
					{
						Type:              AccessPointControl,
						Address:           "2.3.4.4",
						Port:              2345,
						UseSecureProtocol: false,
						Key:               "abcd",
					},
					{
						Type:              AccessPointStatistics,
						Address:           "2.3.4.5",
						Port:              2346,
						UseSecureProtocol: false,
						Key:               "",
					},
				},
			},
		},
	}

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
		app := am.GetApp(AccessPointControl, "1.2.3.1", 1234)
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
		app := am.GetApp(AccessPointControl, "2.3.4.4", 2345)
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
		app := am.GetApp(AccessPointControl, "0.0.0.0", 1)
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

// Test that the Kea, BIND 9 and PowerDNS apps are detected properly.
func TestDetectApps(t *testing.T) {
	// Arrange
	sb := testutil.NewSandbox()
	defer sb.Close()

	// Prepare Kea config file.
	keaConfPath, _ := sb.Write("kea-control-agent.conf", `{ "Control-agent": {
		"http-host": "localhost",
		"http-port": 45634
	} }`)

	// Prepare the command commander.
	commander := newTestCommandExecutorDefault()

	// Prepare process mocks.
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	keaProcess := NewMockSupportedProcess(ctrl)
	keaProcess.EXPECT().getName().AnyTimes().Return("kea-ctrl-agent", nil)
	keaProcess.EXPECT().getCmdline().AnyTimes().Return(fmt.Sprintf(
		"kea-ctrl-agent -c %s", keaConfPath,
	), nil)
	keaProcess.EXPECT().getCwd().AnyTimes().Return("/etc/kea", nil)
	keaProcess.EXPECT().getPid().AnyTimes().Return(int32(1234))
	keaProcess.EXPECT().getParentPid().AnyTimes().Return(int32(2345), nil)

	bind9Process := NewMockSupportedProcess(ctrl)
	bind9Process.EXPECT().getName().AnyTimes().Return("named", nil)
	bind9Process.EXPECT().getCmdline().AnyTimes().Return("named -c /etc/named.conf", nil)
	bind9Process.EXPECT().getCwd().AnyTimes().Return("/etc", nil)
	bind9Process.EXPECT().getPid().AnyTimes().Return(int32(5678))
	bind9Process.EXPECT().getParentPid().AnyTimes().Return(int32(6789), nil)

	pdnsProcess := NewMockSupportedProcess(ctrl)
	pdnsProcess.EXPECT().getName().AnyTimes().Return("pdns_server", nil)
	pdnsProcess.EXPECT().getCmdline().AnyTimes().Return("pdns_server --config-dir=/etc/powerdns", nil)
	pdnsProcess.EXPECT().getCwd().AnyTimes().Return("/etc", nil)
	pdnsProcess.EXPECT().getPid().AnyTimes().Return(int32(7890))
	pdnsProcess.EXPECT().getParentPid().AnyTimes().Return(int32(8901), nil)

	unknownProcess := NewMockSupportedProcess(ctrl)
	unknownProcess.EXPECT().getName().AnyTimes().Return("unknown", nil)
	unknownProcess.EXPECT().getPid().AnyTimes().Return(int32(3456))
	unknownProcess.EXPECT().getParentPid().AnyTimes().Return(int32(4567), nil)

	processManager := NewProcessManager()
	lister := NewMockProcessLister(ctrl)
	lister.EXPECT().listProcesses().AnyTimes().Return([]supportedProcess{
		keaProcess, bind9Process, pdnsProcess, unknownProcess,
	}, nil)
	processManager.lister = lister

	bind9ConfigParser := NewMockBind9FileParser(ctrl)
	bind9ConfigParser.EXPECT().ParseFile("/etc/named.conf").AnyTimes().DoAndReturn(func(configPath string) (*bind9config.Config, error) {
		return bind9config.NewParser().Parse(configPath, strings.NewReader(defaultBind9Config))
	})

	pdnsConfigParser := NewMockPDNSConfigParser(ctrl)
	pdnsConfigParser.EXPECT().ParseFile("/etc/powerdns/pdns.conf").AnyTimes().DoAndReturn(func(configPath string) (*pdnsconfig.Config, error) {
		return pdnsconfig.NewParser().Parse(strings.NewReader(defaultPDNSConfig))
	})

	am := &appMonitor{
		processManager:   processManager,
		commander:        commander,
		bind9FileParser:  bind9ConfigParser,
		pdnsConfigParser: pdnsConfigParser,
	}
	hm := NewHookManager()
	bind9StatsClient := NewBind9StatsClient()
	httpConfig := HTTPClientConfig{}
	sa := NewStorkAgent("foo", 42, am, bind9StatsClient, httpConfig, hm, "", "")

	// Create fake app for which the zone inventory should be stopped
	// when new apps are detected.
	fakeApp := NewMockApp(ctrl)
	fakeApp.EXPECT().GetBaseApp().AnyTimes().Return(&BaseApp{})
	fakeApp.EXPECT().StopZoneInventory().Times(1)

	am.apps = append(am.apps, fakeApp)

	// Act
	am.detectApps(sa)
	apps := am.apps

	// Assert
	require.Len(t, apps, 3)
	require.Equal(t, AppTypeKea, apps[0].GetBaseApp().Type)
	require.EqualValues(t, 1234, apps[0].GetBaseApp().Pid)
	require.Equal(t, AppTypeBind9, apps[1].GetBaseApp().Type)
	require.EqualValues(t, 5678, apps[1].GetBaseApp().Pid)
	require.Equal(t, AppTypePowerDNS, apps[2].GetBaseApp().Type)
	require.EqualValues(t, 7890, apps[2].GetBaseApp().Pid)

	// Detect tha apps again. The zone inventory should be preserved.
	am.detectApps(sa)
	apps2 := am.apps
	require.Len(t, apps2, 3)
	require.Equal(t, apps[1].(*Bind9App).zoneInventory, apps2[1].(*Bind9App).zoneInventory)
	require.True(t, apps[1].(*Bind9App).zoneInventory.isAXFRWorkersActive())
	require.True(t, apps2[1].(*Bind9App).zoneInventory.isAXFRWorkersActive())
	require.Equal(t, apps[2].(*PDNSApp).zoneInventory, apps2[2].(*PDNSApp).zoneInventory)
	require.True(t, apps[2].(*PDNSApp).zoneInventory.isAXFRWorkersActive())
	require.True(t, apps2[2].(*PDNSApp).zoneInventory.isAXFRWorkersActive())

	// If the app access point changes, the inventory should be recreated.
	for index, accessPoint := range am.apps[1].(*Bind9App).AccessPoints {
		if accessPoint.Type == AccessPointControl {
			// Change the access point port.
			am.apps[1].(*Bind9App).AccessPoints[index].Port = 5453
		}
	}
	for index, accessPoint := range am.apps[2].(*PDNSApp).AccessPoints {
		if accessPoint.Type == AccessPointControl {
			// Change the access point port.
			am.apps[2].(*PDNSApp).AccessPoints[index].Port = 8082
		}
	}

	// Redetect apps. It should result in recreating the zone inventory.
	am.detectApps(sa)
	apps3 := am.apps
	require.Len(t, apps3, 3)
	require.NotEqual(t, apps[1].(*Bind9App).zoneInventory, apps3[1].(*Bind9App).zoneInventory)
	require.False(t, apps[1].(*Bind9App).zoneInventory.isAXFRWorkersActive())
	require.True(t, apps3[1].(*Bind9App).zoneInventory.isAXFRWorkersActive())
	require.NotEqual(t, apps[2].(*PDNSApp).zoneInventory, apps3[2].(*PDNSApp).zoneInventory)
	require.False(t, apps[2].(*PDNSApp).zoneInventory.isAXFRWorkersActive())
	require.True(t, apps3[2].(*PDNSApp).zoneInventory.isAXFRWorkersActive())
}

// Test that verifies that when the zone inventory is not initialized
// re-detecting the app does not cause an error.
func TestDetectAppsConfigNoStatistics(t *testing.T) {
	// Arrange
	sb := testutil.NewSandbox()
	defer sb.Close()

	// Prepare the command executor.
	executor := newTestCommandExecutor().
		addCheckConfOutput("/etc/named.conf", bind9ConfigWithoutStatistics).
		setConfigPathInNamedOutput("/etc/named.conf")

	// Prepare process mocks.
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	bind9Process := NewMockSupportedProcess(ctrl)
	bind9Process.EXPECT().getName().AnyTimes().Return("named", nil)
	bind9Process.EXPECT().getCmdline().AnyTimes().Return("named -c /etc/named.conf", nil)
	bind9Process.EXPECT().getCwd().AnyTimes().Return("/etc", nil)
	bind9Process.EXPECT().getPid().AnyTimes().Return(int32(5678))
	bind9Process.EXPECT().getParentPid().AnyTimes().Return(int32(6789), nil)

	processManager := NewProcessManager()
	lister := NewMockProcessLister(ctrl)
	lister.EXPECT().listProcesses().AnyTimes().Return([]supportedProcess{
		bind9Process,
	}, nil)
	processManager.lister = lister

	parser := NewMockBind9FileParser(ctrl)
	parser.EXPECT().ParseFile("/etc/named.conf").AnyTimes().DoAndReturn(func(configPath string) (*bind9config.Config, error) {
		return bind9config.NewParser().Parse(configPath, strings.NewReader(bind9ConfigWithoutStatistics))
	})
	am := &appMonitor{processManager: processManager, commander: executor, bind9FileParser: parser}
	hm := NewHookManager()
	bind9StatsClient := NewBind9StatsClient()
	httpConfig := HTTPClientConfig{}
	sa := NewStorkAgent("foo", 42, am, bind9StatsClient, httpConfig, hm, "", "")

	// Create fake app to test that the monitor stops zone inventory
	// when new apps are detected.
	fakeApp := NewMockApp(ctrl)
	fakeApp.EXPECT().GetBaseApp().AnyTimes().Return(&BaseApp{})
	fakeApp.EXPECT().StopZoneInventory().Times(1)

	am.apps = append(am.apps, fakeApp)

	// Detect apps for the first time.
	am.detectApps(sa)
	apps := am.apps

	// Zone inventory should not be initialized.
	require.Len(t, apps, 1)
	require.Nil(t, apps[0].(*Bind9App).zoneInventory)

	// Detect apps again. It should not panic even though the zone
	// inventory is not initialized.
	am.detectApps(sa)
	apps2 := am.apps
	require.Len(t, apps2, 1)
	require.Nil(t, apps2[0].(*Bind9App).zoneInventory)
}

// Test that the processes for which the command line cannot be read are
// not skipped.
func TestDetectAppsContinueOnNotAvailableCommandLine(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	bind9Process := NewMockSupportedProcess(ctrl)
	bind9Process.EXPECT().getName().AnyTimes().Return("named", nil)
	bind9Process.EXPECT().getCmdline().AnyTimes().Return("named -c /etc/named.conf", nil)
	bind9Process.EXPECT().getCwd().Return("", errors.New("no current working directory"))
	bind9Process.EXPECT().getPid().AnyTimes().Return(int32(5678))
	bind9Process.EXPECT().getParentPid().AnyTimes().Return(int32(6789), nil)

	processManager := NewProcessManager()
	lister := NewMockProcessLister(ctrl)
	lister.EXPECT().listProcesses().Return([]supportedProcess{
		bind9Process,
	}, nil)
	processManager.lister = lister

	parser := NewMockBind9FileParser(ctrl)
	parser.EXPECT().ParseFile("/etc/named.conf").AnyTimes().DoAndReturn(func(configPath string) (*bind9config.Config, error) {
		return bind9config.NewParser().Parse(configPath, strings.NewReader(defaultBind9Config))
	})
	executor := newTestCommandExecutorDefault()
	am := &appMonitor{processManager: processManager, commander: executor, bind9FileParser: parser}
	hm := NewHookManager()
	bind9StatsClient := NewBind9StatsClient()
	httpConfig := HTTPClientConfig{}
	sa := NewStorkAgent("foo", 42, am, bind9StatsClient, httpConfig, hm, "", "")

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

	noCwdProcess := NewMockSupportedProcess(ctrl)
	noCwdProcess.EXPECT().getName().AnyTimes().Return("kea-ctrl-agent", nil)
	noCwdProcess.EXPECT().getCmdline().AnyTimes().Return("kea-ctrl-agent -c /etc/kea/kea.conf", nil)
	noCwdProcess.EXPECT().getCwd().AnyTimes().Return("", errors.New("no current working directory"))
	noCwdProcess.EXPECT().getPid().AnyTimes().Return(int32(1234))
	noCwdProcess.EXPECT().getParentPid().AnyTimes().Return(int32(2345), nil)

	bind9Process := NewMockSupportedProcess(ctrl)
	bind9Process.EXPECT().getName().AnyTimes().Return("named", nil)
	bind9Process.EXPECT().getCmdline().AnyTimes().Return("named -c /etc/named.conf", nil)
	bind9Process.EXPECT().getCwd().AnyTimes().Return("/etc", nil)
	bind9Process.EXPECT().getPid().AnyTimes().Return(int32(5678))
	bind9Process.EXPECT().getParentPid().AnyTimes().Return(int32(6789), nil)

	processManager := NewProcessManager()
	lister := NewMockProcessLister(ctrl)
	lister.EXPECT().listProcesses().Return([]supportedProcess{
		noCwdProcess, bind9Process,
	}, nil)
	processManager.lister = lister

	executor := newTestCommandExecutorDefault()

	parser := NewMockBind9FileParser(ctrl)
	parser.EXPECT().ParseFile("/etc/named.conf").AnyTimes().DoAndReturn(func(configPath string) (*bind9config.Config, error) {
		return bind9config.NewParser().Parse(configPath, strings.NewReader(defaultBind9Config))
	})

	am := &appMonitor{processManager: processManager, commander: executor, bind9FileParser: parser}
	hm := NewHookManager()
	bind9StatsClient := NewBind9StatsClient()
	httpConfig := HTTPClientConfig{}
	sa := NewStorkAgent("foo", 42, am, bind9StatsClient, httpConfig, hm, "", "")

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

	processManager := NewProcessManager()
	lister := NewMockProcessLister(ctrl)
	lister.EXPECT().listProcesses().Return([]supportedProcess{}, nil)
	processManager.lister = lister

	executor := newTestCommandExecutorDefault()
	am := &appMonitor{processManager: processManager, commander: executor}
	hm := NewHookManager()
	bind9StatsClient := NewBind9StatsClient()
	httpConfig := HTTPClientConfig{}
	sa := NewStorkAgent("foo", 42, am, bind9StatsClient, httpConfig, hm, "", "")

	// Act
	am.detectApps(sa)

	// Assert
	require.Contains(t, buffer.String(), "No app detected for monitoring")
}

// Test that detectAllowedLogs does not panic when Kea server is unreachable.
func TestDetectAllowedLogsKeaUnreachable(t *testing.T) {
	am := &appMonitor{}
	bind9StatsClient := NewBind9StatsClient()
	httpConfig := HTTPClientConfig{}
	httpClient := NewHTTPClient(httpConfig)
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
	sa := NewStorkAgent("foo", 42, am, bind9StatsClient, httpConfig, hm, "", "")

	require.NotPanics(t, func() { am.detectAllowedLogs(sa) })
}

// Returns a fixed output and no error for any data. The output contains the
// Bind 9 response with statistic channel details.
func newTestCommandExecutorDefault() *testCommandExecutor {
	return newTestCommandExecutor().
		addCheckConfOutput("/etc/named.conf", defaultBind9Config).
		setConfigPathInNamedOutput("/etc/named.conf")
}

// Check BIND 9 app detection when its conf file is absolute path.
func TestDetectBind9AppAbsPath(t *testing.T) {
	// check BIND 9 app detection
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	parser := NewMockBind9FileParser(ctrl)
	parser.EXPECT().ParseFile("/etc/named.conf").AnyTimes().DoAndReturn(func(configPath string) (*bind9config.Config, error) {
		return bind9config.NewParser().Parse(configPath, strings.NewReader(defaultBind9Config))
	})
	process := NewMockSupportedProcess(ctrl)
	process.EXPECT().getCmdline().Return("/dir/named -c /etc/named.conf", nil)
	process.EXPECT().getCwd().Return("", nil)
	executor := newTestCommandExecutorDefault()
	app, err := detectBind9App(process, executor, "", parser)
	require.NoError(t, err)
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
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	parser := NewMockBind9FileParser(ctrl)
	parser.EXPECT().ParseFile("/etc/named.conf").AnyTimes().DoAndReturn(func(configPath string) (*bind9config.Config, error) {
		return bind9config.NewParser().Parse(configPath, strings.NewReader(defaultBind9Config))
	})
	executor := newTestCommandExecutorDefault()
	process := NewMockSupportedProcess(ctrl)
	process.EXPECT().getCmdline().Return("/dir/named -c named.conf", nil)
	process.EXPECT().getCwd().Return("/etc", nil)
	app, err := detectBind9App(process, executor, "", parser)
	require.NoError(t, err)
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
	nestedChildPath, _ := sb.Write("nested-child.json", `
		"control-sockets": {
			"dhcp4": { },
			"dhcp6": { },
			"d2": { }
		}`,
	)

	childPath, _ := sb.Write("child.json", fmt.Sprintf(`{
		"http-host": "localhost",
		"http-port": 45634,
		<?include "%s"?>
	}`, nestedChildPath))

	text := fmt.Sprintf(`{ "Control-agent": <?include "%s"?> }`, childPath)
	parentPath, _ := sb.Write("parent.json", text)

	return parentPath, func() { sb.Close() }
}

func TestDetectKeaApp(t *testing.T) {
	checkApp := func(app App) {
		keaApp, ok := app.(*KeaApp)
		require.True(t, ok)
		require.NotNil(t, app)
		require.Equal(t, AppTypeKea, app.GetBaseApp().Type)
		require.Len(t, app.GetBaseApp().AccessPoints, 1)
		ctrlPoint := app.GetBaseApp().AccessPoints[0]
		require.Equal(t, AccessPointControl, ctrlPoint.Type)
		require.Equal(t, "localhost", ctrlPoint.Address)
		require.EqualValues(t, 45634, ctrlPoint.Port)
		require.Empty(t, ctrlPoint.Key)
		require.Len(t, keaApp.ConfiguredDaemons, 3)
		require.Nil(t, keaApp.ActiveDaemons)
	}

	httpClientConfig := HTTPClientConfig{}

	t.Run("config file without include statement", func(t *testing.T) {
		tmpFilePath, clean := makeKeaConfFile()
		defer clean()

		// check kea app detection
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		process := NewMockSupportedProcess(ctrl)
		process.EXPECT().getCmdline().Return(fmt.Sprintf("kea-ctrl-agent -c %s", tmpFilePath), nil)
		process.EXPECT().getCwd().Return("", nil)
		app, err := detectKeaApp(process, httpClientConfig)
		require.NoError(t, err)
		checkApp(app)

		// check kea app detection when kea conf file is relative to CWD of kea process
		cwd, file := path.Split(tmpFilePath)
		process.EXPECT().getCmdline().Return(fmt.Sprintf("kea-ctrl-agent -c %s", file), nil)
		process.EXPECT().getCwd().Return(cwd, nil)
		app, err = detectKeaApp(process, httpClientConfig)
		require.NoError(t, err)
		checkApp(app)
	})

	t.Run("config file with include statement", func(t *testing.T) {
		// Check configuration with an include statement
		tmpFilePath, clean := makeKeaConfFileWithInclude()
		defer clean()

		// check kea app detection
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		process := NewMockSupportedProcess(ctrl)
		process.EXPECT().getCmdline().Return(fmt.Sprintf("kea-ctrl-agent -c %s", tmpFilePath), nil)
		process.EXPECT().getCwd().Return("", nil)

		app, err := detectKeaApp(process, httpClientConfig)
		require.NoError(t, err)
		checkApp(app)

		// check kea app detection when kea conf file is relative to CWD of kea process
		cwd, file := path.Split(tmpFilePath)
		process.EXPECT().getCmdline().Return(fmt.Sprintf("kea-ctrl-agent -c %s", file), nil)
		process.EXPECT().getCwd().Return(cwd, nil)
		app, err = detectKeaApp(process, httpClientConfig)
		require.NoError(t, err)
		checkApp(app)
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
	output := logrus.StandardLogger().Out
	defer func() {
		logrus.SetOutput(output)
	}()
	var buffer bytes.Buffer
	logrus.SetOutput(&buffer)

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

	require.Contains(t, buffer.String(), "New or updated apps detected:")
	require.Contains(t, buffer.String(),
		"bind9: control: http://127.0.0.53:5353/ (auth key: found), statistics: http://127.0.0.80:80/",
	)
	require.Contains(t, buffer.String(), "kea: control: http://localhost:45634/")
}

// Test that the active Kea daemons are recognized properly.
func TestDetectActiveDaemons(t *testing.T) {
	// Arrange
	output := logrus.StandardLogger().Out
	defer func() {
		logrus.SetOutput(output)
	}()
	var buffer bytes.Buffer
	logrus.SetOutput(&buffer)

	httpClient := newHTTPClientWithDefaults()
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
			"text": "Detection error occurred",
			"arguments": {}
		}, {
			"result": 0,
			"arguments": {}
		}, {
			"result": 0,
			"arguments": {}
		}]`)

	app := &KeaApp{
		ConfiguredDaemons: []string{"d2", "dhcp4", "dhcp6"},
		BaseApp: BaseApp{
			Type:         AppTypeKea,
			AccessPoints: makeAccessPoint(AccessPointControl, "localhost", "", 45634, false),
		},
		HTTPClient: httpClient,
	}

	t.Run("first detection", func(t *testing.T) {
		buffer.Reset()

		// Act
		activeDaemons, err := detectKeaActiveDaemons(app, nil)

		// Assert
		require.NoError(t, err)
		require.Len(t, activeDaemons, 2)
		require.Contains(t, activeDaemons, "dhcp4")
		require.Contains(t, activeDaemons, "dhcp6")
		require.Contains(t, buffer.String(), "Detection error occurred")
	})

	t.Run("previous detection doesn't find any active daemons", func(t *testing.T) {
		buffer.Reset()

		// Act
		activeDaemons, err := detectKeaActiveDaemons(app, []string{})

		// Assert
		require.NoError(t, err)
		require.Len(t, activeDaemons, 2)
		require.Contains(t, activeDaemons, "dhcp4")
		require.Contains(t, activeDaemons, "dhcp6")
		require.NotContains(t, buffer.String(), "Detection error occurred")
	})

	t.Run("following detection, D2 is still inactive", func(t *testing.T) {
		buffer.Reset()

		// Act
		activeDaemons, err := detectKeaActiveDaemons(app, []string{"dhcp4"})

		// Assert
		require.NoError(t, err)
		require.Len(t, activeDaemons, 2)
		require.Contains(t, activeDaemons, "dhcp4")
		require.Contains(t, activeDaemons, "dhcp6")
		require.NotContains(t, buffer.String(), "Detection error occurred")
	})

	t.Run("following detection, D2 switch to the inactive state", func(t *testing.T) {
		buffer.Reset()

		// Act
		activeDaemons, err := detectKeaActiveDaemons(app, []string{"d2", "dhcp4"})

		// Assert
		require.NoError(t, err)
		require.Len(t, activeDaemons, 2)
		require.Contains(t, activeDaemons, "dhcp4")
		require.Contains(t, activeDaemons, "dhcp6")
		require.Contains(t, buffer.String(), "Detection error occurred")
	})

	t.Run("previous detection finds all active daemons", func(t *testing.T) {
		buffer.Reset()

		//	Act
		activeDaemons, err := detectKeaActiveDaemons(app, []string{"dhcp4", "dhcp6"})

		//	Assert
		require.NoError(t, err)
		require.Len(t, activeDaemons, 2)
		require.Contains(t, activeDaemons, "dhcp4")
		require.Contains(t, activeDaemons, "dhcp6")
		require.NotContains(t, buffer.String(), "Detection error occurred")
	})
}

// Test that the access point can be retrieved from the type.
func TestBaseAppGetAccessPoint(t *testing.T) {
	// Arrange
	app := BaseApp{
		AccessPoints: makeAccessPoint(AccessPointControl, "", "", 0, false),
	}

	// Act & Assert
	// Known access point.
	require.NotNil(t, app.GetAccessPoint(AccessPointControl))
	// Unknown access point.
	require.Nil(t, app.GetAccessPoint(AccessPointStatistics))
}

// Test that the applications can be compared by their types.
func TestBaseAppHasEqualType(t *testing.T) {
	// Arrange
	appKea1 := &BaseApp{Type: AppTypeKea}
	appKea2 := &BaseApp{Type: AppTypeKea}
	appBind := &BaseApp{Type: AppTypeBind9}

	// Act & Assert
	require.True(t, appKea1.HasEqualType(appKea2))
	require.True(t, appKea2.HasEqualType(appKea1))
	require.False(t, appKea1.HasEqualType(appBind))
	require.False(t, appBind.HasEqualType(appKea2))
}

// Test that the applications can be compared by their access points.
func TestBaseAppHasEqualAccessPoints(t *testing.T) {
	// Arrange
	app1 := &BaseApp{
		Type: AppTypeKea,
		AccessPoints: []AccessPoint{
			{Type: AccessPointControl, Address: "localhost", Port: 1234},
		},
	}
	app2 := &BaseApp{
		Type: AppTypeKea,
		AccessPoints: []AccessPoint{
			{Type: AccessPointControl, Address: "localhost", Port: 1234},
		},
	}
	app3 := &BaseApp{
		Type: AppTypeBind9,
		AccessPoints: []AccessPoint{
			{Type: AccessPointControl, Address: "localhost", Port: 1234},
		},
	}
	app4 := &BaseApp{
		Type: AppTypeKea,
		AccessPoints: []AccessPoint{
			{
				Type: AccessPointControl, Address: "localhost", Port: 1235,
				UseSecureProtocol: true, Key: "key",
			},
		},
	}
	app5 := &BaseApp{
		Type: AppTypeKea,
		AccessPoints: []AccessPoint{
			{Type: AccessPointControl, Address: "localhost", Port: 1234},
			{Type: AccessPointStatistics, Address: "localhost", Port: 1235},
		},
	}

	// Act & Assert
	// Same app types and access points.
	require.True(t, app1.HasEqualAccessPoints(app2))
	// Different app types but the same access points.
	require.True(t, app1.HasEqualAccessPoints(app3))
	// Same app types, and the same access point location but different
	// configuration.
	require.False(t, app1.HasEqualAccessPoints(app4))
	// The second app has the same app type and includes the access points from
	// the first app but it has an additional access point.
	require.False(t, app1.HasEqualAccessPoints(app5))
}

// Test that the applications can be compared by their overall content.
func TestBaseAppEqual(t *testing.T) {
	// Arrange
	app1 := &BaseApp{
		Type: AppTypeKea,
		AccessPoints: []AccessPoint{
			{Type: AccessPointControl, Address: "localhost", Port: 1234},
		},
	}
	app2 := &BaseApp{
		Type: AppTypeKea,
		AccessPoints: []AccessPoint{
			{Type: AccessPointControl, Address: "localhost", Port: 1234},
		},
	}
	app3 := &BaseApp{
		Type: AppTypeBind9,
		AccessPoints: []AccessPoint{
			{Type: AccessPointControl, Address: "localhost", Port: 1234},
		},
	}
	app4 := &BaseApp{
		Type: AppTypeKea,
		AccessPoints: []AccessPoint{
			{
				Type: AccessPointControl, Address: "localhost", Port: 1235,
				UseSecureProtocol: true, Key: "key",
			},
		},
	}
	app5 := &BaseApp{
		Type: AppTypeKea,
		AccessPoints: []AccessPoint{
			{Type: AccessPointControl, Address: "localhost", Port: 1234},
			{Type: AccessPointStatistics, Address: "localhost", Port: 1235},
		},
	}

	// Act & Assert
	// Same app types and access points.
	require.True(t, app1.IsEqual(app2))
	// Different app types but the same access points.
	require.False(t, app1.IsEqual(app3))
	// Same app types, and the same access point location but different
	// configuration.
	require.False(t, app1.IsEqual(app4))
	// The second app has the same app type and includes the access points from
	// the first app but it has an additional access point.
	require.False(t, app1.IsEqual(app5))
}

// Test that the DNS zone inventories are successfully populated.
func TestPopulateZoneInventories(t *testing.T) {
	response := map[string]any{
		"views": map[string]any{
			"_default": map[string]any{
				"zones": generateRandomZones(10),
			},
		},
	}
	bind9StatsClient, off := setGetViewsResponseOK(t, response)
	defer off()

	config := parseDefaultBind9Config(t)
	require.NotNil(t, config)

	monitor := NewAppMonitor()
	appMonitor, ok := monitor.(*appMonitor)
	require.True(t, ok)

	app0 := &Bind9App{
		BaseApp: BaseApp{
			Type: AppTypeBind9,
		},
		zoneInventory: nil,
	}
	zi1 := newZoneInventory(newZoneInventoryStorageMemory(), config, bind9StatsClient, "localhost", 5380)
	app1 := &Bind9App{
		BaseApp: BaseApp{
			Type: AppTypeBind9,
		},
		zoneInventory: zi1,
	}
	zi2 := newZoneInventory(newZoneInventoryStorageMemory(), config, bind9StatsClient, "localhost", 5380)
	app2 := &Bind9App{
		BaseApp: BaseApp{
			Type: AppTypeBind9,
		},
		zoneInventory: zi2,
	}
	zi3 := newZoneInventory(newZoneInventoryStorageMemory(), config, bind9StatsClient, "localhost", 5380)
	app3 := &PDNSApp{
		BaseApp: BaseApp{
			Type: AppTypePowerDNS,
		},
		zoneInventory: zi3,
	}
	app4 := &KeaApp{
		BaseApp: BaseApp{
			Type: AppTypeKea,
		},
	}
	appMonitor.apps = append(appMonitor.apps, app0, app1, app2, app3, app4)
	appMonitor.populateZoneInventories()

	require.Eventually(t, func() bool {
		for _, app := range appMonitor.apps {
			var zoneInventory *zoneInventory
			switch concreteApp := app.(type) {
			case *Bind9App:
				zoneInventory = concreteApp.zoneInventory
			case *PDNSApp:
				zoneInventory = concreteApp.zoneInventory
			default:
				continue
			}
			if zoneInventory == nil {
				continue
			}
			if !zoneInventory.getCurrentState().isReady() {
				return false
			}
		}
		return true
	}, 5*time.Second, time.Millisecond)
}
