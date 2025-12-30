package agent

import (
	"bytes"
	"fmt"
	"path"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	gomock "go.uber.org/mock/gomock"

	bind9config "isc.org/stork/daemoncfg/bind9"
	pdnsconfig "isc.org/stork/daemoncfg/pdns"
	"isc.org/stork/datamodel/daemonname"
	"isc.org/stork/datamodel/protocoltype"
	"isc.org/stork/testutil"
)

//go:generate mockgen -source process.go -package=agent -destination=processmock_test.go -mock_names=processLister=MockProcessLister,supportedProcess=MockSupportedProcess isc.org/agent supportedProcess processLister
//go:generate mockgen -source monitor.go -package=agent -destination=monitormock_test.go -mock_names=agentManager=MockAgentManager isc.org/agent agentManager

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

func TestGetDaemons(t *testing.T) {
	monitor := NewMonitor("", "", HTTPClientConfig{})
	hm := NewHookManager()
	bind9StatsClient := NewBind9StatsClient()
	sa := NewStorkAgent("foo", 42, monitor, bind9StatsClient, hm)
	monitor.Start(t.Context(), sa)
	daemons := monitor.GetDaemons()
	require.Len(t, daemons, 0)
	monitor.Shutdown()
}

// Check if detected daemons are returned by GetDaemonByAccessPoint.
func TestGetDaemonByAccessPoint(t *testing.T) {
	m := NewMonitor("", "", HTTPClientConfig{})

	daemons := []Daemon{
		&keaDaemon{
			daemon: daemon{
				Name: daemonname.CA,
				AccessPoints: []AccessPoint{
					{
						Type:     AccessPointControl,
						Address:  "1.2.3.1",
						Port:     1234,
						Protocol: protocoltype.HTTP,
					},
				},
			},
		},
		&Bind9Daemon{
			dnsDaemonImpl: dnsDaemonImpl{
				daemon: daemon{
					Name: daemonname.Bind9,
					AccessPoints: []AccessPoint{
						{
							Type:     AccessPointControl,
							Address:  "2.3.4.4",
							Port:     2345,
							Protocol: protocoltype.HTTP,
							Key:      "abcd",
						},
						{
							Type:     AccessPointStatistics,
							Address:  "2.3.4.5",
							Port:     2346,
							Protocol: protocoltype.HTTP,
						},
					},
				},
			},
		},
	}

	// Monitor holds daemons in background goroutine. So to get daemons we need
	// to send a request over a channel to this goroutine and wait for
	// a response with detected daemons. We do not want to spawn monitor background
	// goroutine so we are calling GetDaemonByAccessPoint in our background goroutine
	// and are serving this request in the main thread.
	// To make it in sync the wait group is used here.
	var wg sync.WaitGroup

	// find kea daemon
	wg.Add(1)
	go func() {
		defer wg.Done()
		daemon := m.GetDaemonByAccessPoint(AccessPointControl, "1.2.3.1", 1234)
		require.NotNil(t, daemon)
		require.EqualValues(t, daemonname.CA, daemon.GetName())
	}()
	ret := <-m.(*monitor).requests
	ret <- daemons
	wg.Wait()

	// find bind daemon
	wg.Add(1) // expect 1 Done in the wait group
	go func() {
		defer wg.Done()
		daemon := m.GetDaemonByAccessPoint(AccessPointControl, "2.3.4.4", 2345)
		require.NotNil(t, daemon)
		require.EqualValues(t, daemonname.Bind9, daemon.GetName())
	}()
	ret = <-m.(*monitor).requests
	ret <- daemons
	wg.Wait()

	// find not existing daemon - should return nil
	wg.Add(1) // expect 1 Done in the wait group
	go func() {
		defer wg.Done()
		daemon := m.GetDaemonByAccessPoint(AccessPointControl, "0.0.0.0", 1)
		require.Nil(t, daemon)
	}()
	ret = <-m.(*monitor).requests
	ret <- daemons
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

	controlSockets := config.GetListeningControlSockets()
	require.Len(t, controlSockets, 1)
	controlSocket := controlSockets[0]

	port := controlSocket.GetPort()
	require.EqualValues(t, 1234, port)

	address := controlSocket.GetAddress()
	require.Equal(t, "host.example.org", address)

	protocol := controlSocket.GetProtocol()
	require.Equal(t, protocoltype.HTTP, protocol)
}

// Test that the Kea, BIND 9 and PowerDNS daemons are detected properly.
func TestDetectDaemons(t *testing.T) {
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
	keaProcess.EXPECT().getDaemonName().AnyTimes().Return(daemonname.CA)
	keaProcess.EXPECT().getCmdline().AnyTimes().Return(fmt.Sprintf(
		"kea-ctrl-agent -c %s", keaConfPath,
	), nil)
	keaProcess.EXPECT().getCwd().AnyTimes().Return("/etc/kea", nil)
	keaProcess.EXPECT().getPid().AnyTimes().Return(int32(1234))
	keaProcess.EXPECT().getParentPid().AnyTimes().Return(int32(2345), nil)

	bind9Process := NewMockSupportedProcess(ctrl)
	bind9Process.EXPECT().getName().AnyTimes().Return("named", nil)
	bind9Process.EXPECT().getDaemonName().AnyTimes().Return(daemonname.Bind9)
	bind9Process.EXPECT().getCmdline().AnyTimes().Return("named -c /etc/named.conf", nil)
	bind9Process.EXPECT().getCwd().AnyTimes().Return("/etc", nil)
	bind9Process.EXPECT().getPid().AnyTimes().Return(int32(5678))
	bind9Process.EXPECT().getParentPid().AnyTimes().Return(int32(6789), nil)

	pdnsProcess := NewMockSupportedProcess(ctrl)
	pdnsProcess.EXPECT().getName().AnyTimes().Return("pdns_server", nil)
	pdnsProcess.EXPECT().getDaemonName().AnyTimes().Return(daemonname.PDNS)
	pdnsProcess.EXPECT().getCmdline().AnyTimes().Return("pdns_server --config-dir=/etc/powerdns", nil)
	pdnsProcess.EXPECT().getCwd().AnyTimes().Return("/etc", nil)
	pdnsProcess.EXPECT().getPid().AnyTimes().Return(int32(7890))
	pdnsProcess.EXPECT().getParentPid().AnyTimes().Return(int32(8901), nil)

	unknownProcess := NewMockSupportedProcess(ctrl)
	unknownProcess.EXPECT().getName().AnyTimes().Return("unknown", nil)
	unknownProcess.EXPECT().getDaemonName().AnyTimes().Return(daemonname.Name(""))
	unknownProcess.EXPECT().getPid().AnyTimes().Return(int32(3456))
	unknownProcess.EXPECT().getParentPid().AnyTimes().Return(int32(4567), nil)

	processManager := NewProcessManager()
	lister := NewMockProcessLister(ctrl)
	lister.EXPECT().listProcesses().AnyTimes().Return([]supportedProcess{
		keaProcess, bind9Process, pdnsProcess, unknownProcess,
	}, nil)
	processManager.lister = lister

	bind9ConfigParser := NewMockBind9FileParser(ctrl)
	bind9ConfigParser.EXPECT().ParseFile("/etc/named.conf", "").AnyTimes().DoAndReturn(func(configPath, rootPath string) (*bind9config.Config, error) {
		return bind9config.NewParser().Parse(configPath, rootPath, strings.NewReader(defaultBind9Config))
	})

	pdnsConfigParser := NewMockPDNSConfigParser(ctrl)
	pdnsConfigParser.EXPECT().ParseFile("/etc/powerdns/pdns.conf").AnyTimes().DoAndReturn(func(configPath string) (*pdnsconfig.Config, error) {
		return pdnsconfig.NewParser().Parse(configPath, strings.NewReader(defaultPDNSConfig))
	})

	monitor := &monitor{
		processManager:   processManager,
		commander:        commander,
		bind9FileParser:  bind9ConfigParser,
		pdnsConfigParser: pdnsConfigParser,
	}

	// Create fake daemon for which the zone inventory should be stopped
	// when new apps are detected.
	fakeDaemon := NewMockDaemon(ctrl)
	fakeDaemon.EXPECT().Cleanup().Times(1)
	fakeDaemon.EXPECT().IsSame(gomock.Any()).AnyTimes().Return(false)
	fakeDaemon.EXPECT().String().AnyTimes().Return("fake-daemon")

	monitor.daemons = append(monitor.daemons, fakeDaemon)

	// Act
	monitor.detectDaemons(t.Context())
	daemons := monitor.daemons
	sort.Slice(daemons, func(i, j int) bool {
		return daemons[i].GetName() < daemons[j].GetName()
	})

	// Assert
	require.Len(t, daemons, 3)
	require.Equal(t, daemonname.CA, daemons[0].GetName())
	require.Equal(t, daemonname.Bind9, daemons[1].GetName())
	require.Equal(t, daemonname.PDNS, daemons[2].GetName())

	// Detect tha apps again. The zone inventory should be preserved.
	monitor.detectDaemons(t.Context())
	daemons2 := monitor.daemons
	sort.Slice(daemons2, func(i, j int) bool {
		return daemons2[i].GetName() < daemons2[j].GetName()
	})

	require.Len(t, daemons2, 3)
	require.Equal(t, daemons[1].(*Bind9Daemon).zoneInventory, daemons2[1].(*Bind9Daemon).zoneInventory)
	require.True(t, daemons[1].(*Bind9Daemon).zoneInventory.(*zoneInventoryImpl).isAXFRWorkersActive())
	require.True(t, daemons2[1].(*Bind9Daemon).zoneInventory.(*zoneInventoryImpl).isAXFRWorkersActive())
	require.Equal(t, daemons[2].(*pdnsDaemon).zoneInventory, daemons2[2].(*pdnsDaemon).zoneInventory)
	require.True(t, daemons[2].(*pdnsDaemon).zoneInventory.(*zoneInventoryImpl).isAXFRWorkersActive())
	require.True(t, daemons2[2].(*pdnsDaemon).zoneInventory.(*zoneInventoryImpl).isAXFRWorkersActive())

	// If the daemon access point changes, the inventory should be recreated.
	for index, accessPoint := range monitor.daemons[1].(*Bind9Daemon).AccessPoints {
		if accessPoint.Type == AccessPointControl {
			// Change the access point port.
			monitor.daemons[1].(*Bind9Daemon).AccessPoints[index].Port = 5453
		}
	}
	for index, accessPoint := range monitor.daemons[2].(*pdnsDaemon).AccessPoints {
		if accessPoint.Type == AccessPointControl {
			// Change the access point port.
			monitor.daemons[2].(*pdnsDaemon).AccessPoints[index].Port = 8082
		}
	}

	// Also, emulate the size change of the configuration file to ensure
	// that the new daemon instance is used.
	commander.addFileInfo("/etc/named.conf", &testFileInfo{size: 200})

	// Redetect apps. It should result in recreating the zone inventory.
	monitor.detectDaemons(t.Context())
	daemons3 := monitor.daemons
	sort.Slice(daemons3, func(i, j int) bool {
		return daemons3[i].GetName() < daemons3[j].GetName()
	})

	require.Len(t, daemons3, 3)
	require.NotEqual(t, daemons[1].(*Bind9Daemon).zoneInventory, daemons3[1].(*Bind9Daemon).zoneInventory)
	require.False(t, daemons[1].(*Bind9Daemon).zoneInventory.(*zoneInventoryImpl).isAXFRWorkersActive())
	require.True(t, daemons3[1].(*Bind9Daemon).zoneInventory.(*zoneInventoryImpl).isAXFRWorkersActive())
	require.NotEqual(t, daemons[2].(*pdnsDaemon).zoneInventory, daemons3[2].(*pdnsDaemon).zoneInventory)
	require.False(t, daemons[2].(*pdnsDaemon).zoneInventory.(*zoneInventoryImpl).isAXFRWorkersActive())
	require.True(t, daemons3[2].(*pdnsDaemon).zoneInventory.(*zoneInventoryImpl).isAXFRWorkersActive())
}

// Test that verifies that when the zone inventory is not initialized
// re-detecting the daemons does not cause an error.
func TestDetectDaemonsConfigNoStatistics(t *testing.T) {
	// Arrange
	sb := testutil.NewSandbox()
	defer sb.Close()

	// Prepare the command executor.
	executor := newTestCommandExecutor().
		addCheckConfOutput("/etc/named.conf", bind9ConfigWithoutStatistics).
		setConfigPathInNamedOutput("/etc/named.conf").
		addFileInfo("/etc/named.conf", &testFileInfo{})

	// Prepare process mocks.
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	bind9Process := NewMockSupportedProcess(ctrl)
	bind9Process.EXPECT().getName().AnyTimes().Return("named", nil)
	bind9Process.EXPECT().getDaemonName().AnyTimes().Return(daemonname.Bind9)
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
	parser.EXPECT().ParseFile("/etc/named.conf", "").AnyTimes().DoAndReturn(func(configPath, rootPath string) (*bind9config.Config, error) {
		return bind9config.NewParser().Parse(configPath, "", strings.NewReader(bind9ConfigWithoutStatistics))
	})
	monitor := &monitor{processManager: processManager, commander: executor, bind9FileParser: parser}

	// Create fake daemon to test that the monitor stops zone inventory
	// when new daemons are detected.
	fakeDaemon := NewMockDaemon(ctrl)
	fakeDaemon.EXPECT().RefreshState(gomock.Any(), gomock.Any()).AnyTimes()
	fakeDaemon.EXPECT().Cleanup().Times(1)
	fakeDaemon.EXPECT().IsSame(gomock.Any()).AnyTimes().Return(false)
	fakeDaemon.EXPECT().String().AnyTimes().Return("fake-daemon")

	monitor.daemons = append(monitor.daemons, fakeDaemon)

	// Detect daemons for the first time.
	monitor.detectDaemons(t.Context())
	daemons := monitor.daemons

	// Zone inventory should not be initialized.
	require.Len(t, daemons, 1)
	require.Nil(t, daemons[0].(*Bind9Daemon).zoneInventory)

	// Detect daemons again. It should not panic even though the zone
	// inventory is not initialized.
	monitor.detectDaemons(t.Context())
	daemons2 := monitor.daemons
	require.Len(t, daemons2, 1)
	require.Nil(t, daemons2[0].(*Bind9Daemon).zoneInventory)
}

// Test that the processes for which the command line cannot be read are
// not skipped.
func TestDetectDaemonsContinueOnNotAvailableCommandLine(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	bind9Process := NewMockSupportedProcess(ctrl)
	bind9Process.EXPECT().getName().AnyTimes().Return("named", nil)
	bind9Process.EXPECT().getDaemonName().AnyTimes().Return(daemonname.Bind9)
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
	parser.EXPECT().ParseFile("/etc/named.conf", "").AnyTimes().DoAndReturn(func(configPath string, _ string) (*bind9config.Config, error) {
		return bind9config.NewParser().Parse(configPath, "", strings.NewReader(defaultBind9Config))
	})
	executor := newTestCommandExecutorDefault()
	monitor := &monitor{processManager: processManager, commander: executor, bind9FileParser: parser}

	// Act
	monitor.detectDaemons(t.Context())

	// Assert
	require.Len(t, monitor.daemons, 1)
	require.Equal(t, daemonname.Bind9, monitor.daemons[0].GetName())
}

// Test that the processes for which the current working directory cannot be
// read are skipped.
func TestDetectDaemonsSkipOnNotAvailableCwd(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	noCwdProcess := NewMockSupportedProcess(ctrl)
	noCwdProcess.EXPECT().getName().AnyTimes().Return("kea-ctrl-agent", nil)
	noCwdProcess.EXPECT().getDaemonName().AnyTimes().Return(daemonname.CA)
	noCwdProcess.EXPECT().getCmdline().AnyTimes().Return("kea-ctrl-agent -c /etc/kea/kea.conf", nil)
	noCwdProcess.EXPECT().getCwd().AnyTimes().Return("", errors.New("no current working directory"))
	noCwdProcess.EXPECT().getPid().AnyTimes().Return(int32(1234))
	noCwdProcess.EXPECT().getParentPid().AnyTimes().Return(int32(2345), nil)

	bind9Process := NewMockSupportedProcess(ctrl)
	bind9Process.EXPECT().getName().AnyTimes().Return("named", nil)
	bind9Process.EXPECT().getDaemonName().AnyTimes().Return(daemonname.Bind9)
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
	parser.EXPECT().ParseFile("/etc/named.conf", "").AnyTimes().DoAndReturn(func(configPath string, _ string) (*bind9config.Config, error) {
		return bind9config.NewParser().Parse(configPath, "", strings.NewReader(defaultBind9Config))
	})

	monitor := &monitor{processManager: processManager, commander: executor, bind9FileParser: parser}

	// Act
	monitor.detectDaemons(t.Context())

	// Assert
	require.Len(t, monitor.daemons, 1)
	require.Equal(t, daemonname.Bind9, monitor.daemons[0].GetName())
}

// The monitor periodically searches for the Kea/Bind9 instances. Usually, at
// least one application should be available. If no monitored daemon is found,
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
	monitor := &monitor{processManager: processManager, commander: executor}

	// Act
	monitor.detectDaemons(t.Context())

	// Assert
	require.Contains(t, buffer.String(), "No daemon detected for monitoring")
}

// Test that detectAllowedLogs does not panic when Kea server is unreachable.
func TestDetectAllowedLogsKeaUnreachable(t *testing.T) {
	monitor := &monitor{}
	bind9StatsClient := NewBind9StatsClient()
	monitor.daemons = append(monitor.daemons, &keaDaemon{
		daemon: daemon{
			Name: daemonname.CA,
			AccessPoints: []AccessPoint{
				{
					Type:    AccessPointControl,
					Address: "localhost",
					Port:    45678,
				},
			},
		},
	})

	hm := NewHookManager()
	sa := NewStorkAgent("foo", 42, monitor, bind9StatsClient, hm)

	require.NotPanics(t, func() { monitor.refreshDaemons(t.Context(), sa) })
}

// Returns a fixed output and no error for any data. The output contains the
// Bind 9 response with statistic channel details.
func newTestCommandExecutorDefault() *testCommandExecutor {
	return newTestCommandExecutor().
		addCheckConfOutput("/etc/named.conf", defaultBind9Config).
		setConfigPathInNamedOutput("/etc/named.conf").
		addFileInfo("/etc/named.conf", &testFileInfo{}).
		addFileInfo("/etc/powerdns/pdns.conf", &testFileInfo{})
}

// Check BIND 9 daemon detection when its conf file is absolute path.
func TestDetectBind9DaemonAbsPath(t *testing.T) {
	// check BIND 9 daemon detection
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	monitor := newMonitor("", "", HTTPClientConfig{})

	parser := NewMockBind9FileParser(ctrl)
	parser.EXPECT().ParseFile("/etc/named.conf", "").AnyTimes().DoAndReturn(func(configPath string, _ string) (*bind9config.Config, error) {
		return bind9config.NewParser().Parse(configPath, "", strings.NewReader(defaultBind9Config))
	})
	monitor.bind9FileParser = parser

	process := NewMockSupportedProcess(ctrl)
	process.EXPECT().getCmdline().Return("/dir/named -c /etc/named.conf", nil)
	process.EXPECT().getCwd().Return("", nil)
	process.EXPECT().getPid().Return(int32(1234))

	monitor.commander = newTestCommandExecutorDefault()

	daemon, err := monitor.detectBind9Daemon(process)
	require.NoError(t, err)
	require.NotNil(t, daemon)
	require.Equal(t, daemonname.Bind9, daemon.GetName())
	require.Len(t, daemon.GetAccessPoints(), 2)
	point := daemon.GetAccessPoint(AccessPointControl)
	require.Equal(t, AccessPointControl, point.Type)
	require.Equal(t, "127.0.0.53", point.Address)
	require.EqualValues(t, 5353, point.Port)
	require.NotEmpty(t, point.Key)
	point = daemon.GetAccessPoint(AccessPointStatistics)
	require.Equal(t, AccessPointStatistics, point.Type)
	require.Equal(t, "127.0.0.80", point.Address)
	require.EqualValues(t, 80, point.Port)
	require.Empty(t, point.Key)
}

// Check BIND 9 daemon detection when its conf file is relative to CWD of its process.
func TestDetectBind9DaemonRelativePath(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	monitor := newMonitor("", "", HTTPClientConfig{})

	parser := NewMockBind9FileParser(ctrl)
	parser.EXPECT().ParseFile("/etc/named.conf", "").AnyTimes().DoAndReturn(func(configPath string, _ string) (*bind9config.Config, error) {
		return bind9config.NewParser().Parse(configPath, "", strings.NewReader(defaultBind9Config))
	})
	monitor.bind9FileParser = parser

	process := NewMockSupportedProcess(ctrl)
	process.EXPECT().getCmdline().Return("/dir/named -c named.conf", nil)
	process.EXPECT().getCwd().Return("/etc", nil)
	process.EXPECT().getPid().Return(int32(1234))
	monitor.commander = newTestCommandExecutorDefault()
	daemon, err := monitor.detectBind9Daemon(process)
	require.NoError(t, err)
	require.NotNil(t, daemon)
	require.Equal(t, daemonname.Bind9, daemon.GetName())
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

func TestDetectKeaDaemon(t *testing.T) {
	checkDaemon := func(daemons []Daemon) {
		require.Len(t, daemons, 1)
		keaDaemon, ok := daemons[0].(*keaDaemon)
		require.True(t, ok)
		require.NotNil(t, keaDaemon)
		require.Equal(t, daemonname.CA, keaDaemon.GetName())
		require.Len(t, keaDaemon.GetAccessPoints(), 1)
		ctrlPoint := keaDaemon.GetAccessPoint(AccessPointControl)
		require.Equal(t, AccessPointControl, ctrlPoint.Type)
		require.Equal(t, "localhost", ctrlPoint.Address)
		require.EqualValues(t, 45634, ctrlPoint.Port)
		require.Empty(t, ctrlPoint.Key)
	}

	httpClientConfig := HTTPClientConfig{}
	commander := newTestCommandExecutorDefault()

	t.Run("config file without include statement", func(t *testing.T) {
		tmpFilePath, clean := makeKeaConfFile()
		defer clean()

		// check kea daemon detection
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		process := NewMockSupportedProcess(ctrl)
		process.EXPECT().getName().Return("kea-ctrl-agent", nil)
		process.EXPECT().getDaemonName().Return(daemonname.CA)
		process.EXPECT().getCmdline().Return(fmt.Sprintf("/usr/bin/kea-ctrl-agent -c %s", tmpFilePath), nil)
		process.EXPECT().getCwd().Return("", nil)
		daemon, err := detectKeaDaemons(t.Context(), process, httpClientConfig, commander)
		require.NoError(t, err)
		checkDaemon(daemon)

		// check kea daemon detection when kea conf file is relative to CWD of kea process
		cwd, file := path.Split(tmpFilePath)
		process.EXPECT().getName().Return("kea-ctrl-agent", nil)
		process.EXPECT().getDaemonName().Return(daemonname.CA)
		process.EXPECT().getCmdline().Return(fmt.Sprintf("kea-ctrl-agent -c %s", file), nil)
		process.EXPECT().getCwd().Return(cwd, nil)
		daemon, err = detectKeaDaemons(t.Context(), process, httpClientConfig, commander)
		require.NoError(t, err)
		checkDaemon(daemon)
	})

	t.Run("config file with include statement", func(t *testing.T) {
		// Check configuration with an include statement
		tmpFilePath, clean := makeKeaConfFileWithInclude()
		defer clean()

		// check kea daemon detection
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		process := NewMockSupportedProcess(ctrl)
		process.EXPECT().getName().Return("kea-ctrl-agent", nil)
		process.EXPECT().getDaemonName().Return(daemonname.CA)
		process.EXPECT().getCmdline().Return(fmt.Sprintf("/usr/bin/kea-ctrl-agent -c %s", tmpFilePath), nil)
		process.EXPECT().getCwd().Return("", nil)

		daemon, err := detectKeaDaemons(t.Context(), process, httpClientConfig, commander)
		require.NoError(t, err)
		checkDaemon(daemon)

		// check kea daemon detection when kea conf file is relative to CWD of kea process
		cwd, file := path.Split(tmpFilePath)
		process.EXPECT().getName().Return("kea-ctrl-agent", nil)
		process.EXPECT().getDaemonName().Return(daemonname.CA)
		process.EXPECT().getCmdline().Return(fmt.Sprintf("kea-ctrl-agent -c %s", file), nil)
		process.EXPECT().getCwd().Return(cwd, nil)
		daemon, err = detectKeaDaemons(t.Context(), process, httpClientConfig, commander)
		require.NoError(t, err)
		checkDaemon(daemon)
	})
}

func TestGetAccessPoint(t *testing.T) {
	bind9Daemon := &Bind9Daemon{
		dnsDaemonImpl: dnsDaemonImpl{
			daemon: daemon{
				Name: daemonname.Bind9,
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
		},
	}

	keaDaemon := &keaDaemon{
		daemon: daemon{
			Name: daemonname.CA,
			AccessPoints: []AccessPoint{
				{
					Type:    AccessPointControl,
					Address: "localhost",
					Port:    int64(45634),
					Key:     "",
				},
			},
		},
	}

	// test get bind 9 access points
	point := bind9Daemon.GetAccessPoint(AccessPointControl)
	require.NotNil(t, point)
	require.Equal(t, AccessPointControl, point.Type)
	require.Equal(t, "127.0.0.53", point.Address)
	require.EqualValues(t, 5353, point.Port)
	require.Equal(t, "hmac-sha256:abcd", point.Key)

	point = bind9Daemon.GetAccessPoint(AccessPointStatistics)
	require.NotNil(t, point)
	require.Equal(t, AccessPointStatistics, point.Type)
	require.Equal(t, "127.0.0.80", point.Address)
	require.EqualValues(t, 80, point.Port)
	require.Empty(t, point.Key)

	// test get kea access points
	point = keaDaemon.GetAccessPoint(AccessPointControl)
	require.NotNil(t, point)
	require.Equal(t, AccessPointControl, point.Type)
	require.Equal(t, "localhost", point.Address)
	require.EqualValues(t, 45634, point.Port)
	require.Empty(t, point.Key)

	point = keaDaemon.GetAccessPoint(AccessPointStatistics)
	require.Nil(t, point)
}

// Test that the access point can be retrieved from the daemon.
func TestDaemonGetAccessPoint(t *testing.T) {
	// Arrange
	daemon := daemon{
		AccessPoints: []AccessPoint{
			{Type: AccessPointControl, Address: "localhost", Port: 1234},
		},
	}

	// Act & Assert
	// Known access point.
	require.NotNil(t, daemon.GetAccessPoint(AccessPointControl))
	// Unknown access point.
	require.Nil(t, daemon.GetAccessPoint(AccessPointStatistics))
}

// Test that the daemon can be compared by their overall content.
func TestDaemonIsSame(t *testing.T) {
	// Arrange
	daemon1 := &keaDaemon{
		daemon: daemon{
			Name: daemonname.CA,
			AccessPoints: []AccessPoint{
				{Type: AccessPointControl, Address: "localhost", Port: 1234},
			},
		},
	}
	daemon2 := &keaDaemon{
		daemon: daemon{
			Name: daemonname.CA,
			AccessPoints: []AccessPoint{
				{Type: AccessPointControl, Address: "localhost", Port: 1234},
			},
		},
	}
	daemon3 := &keaDaemon{
		daemon: daemon{
			Name: daemonname.DHCPv4,
			AccessPoints: []AccessPoint{
				{Type: AccessPointControl, Address: "localhost", Port: 1234},
			},
		},
	}
	daemon4 := &keaDaemon{
		daemon: daemon{
			Name: daemonname.CA,
			AccessPoints: []AccessPoint{
				{
					Type: AccessPointControl, Address: "localhost", Port: 1235,
					Protocol: protocoltype.HTTPS, Key: "key",
				},
			},
		},
	}
	daemon5 := &keaDaemon{
		daemon: daemon{
			Name: daemonname.CA,
			AccessPoints: []AccessPoint{
				{Type: AccessPointControl, Address: "localhost", Port: 1234},
				{Type: AccessPointStatistics, Address: "localhost", Port: 1235},
			},
		},
	}

	// Act & Assert
	// Same daemon names and access points.
	require.True(t, daemon1.IsSame(daemon2))
	// Different daemon names but the same access points.
	require.False(t, daemon1.IsSame(daemon3))
	// Same daemon names, and the same access point location but different
	// configuration.
	require.False(t, daemon1.IsSame(daemon4))
	// The second daemon has the same daemon name and includes the access
	// points from the first daemon but it has an additional access point.
	require.False(t, daemon1.IsSame(daemon5))
}

// Test that the DNS zone inventories are successfully populated.
func TestPopulateZoneInventories(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	agentManager := NewMockAgentManager(ctrl)

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

	m := NewMonitor("", "", HTTPClientConfig{})
	daemonMonitor, ok := m.(*monitor)
	require.True(t, ok)

	daemon0 := &Bind9Daemon{
		dnsDaemonImpl: dnsDaemonImpl{
			daemon: daemon{
				Name: daemonname.Bind9,
			},
			zoneInventory: nil,
		},
	}
	zi1 := newZoneInventory(newZoneInventoryStorageMemory(), config, bind9StatsClient, "localhost", 5380)
	zi1.start()
	defer zi1.stop()

	daemon1 := &Bind9Daemon{
		dnsDaemonImpl: dnsDaemonImpl{
			daemon: daemon{
				Name: daemonname.Bind9,
			},
			zoneInventory: zi1,
		},
	}

	zi2 := newZoneInventory(newZoneInventoryStorageMemory(), config, bind9StatsClient, "localhost", 5380)
	zi2.start()
	defer zi2.stop()

	daemon2 := &Bind9Daemon{
		dnsDaemonImpl: dnsDaemonImpl{
			daemon: daemon{
				Name: daemonname.Bind9,
			},
			zoneInventory: zi2,
		},
	}
	zi3 := newZoneInventory(newZoneInventoryStorageMemory(), config, bind9StatsClient, "localhost", 5380)
	zi3.start()
	defer zi3.stop()

	daemon3 := &pdnsDaemon{
		dnsDaemonImpl: dnsDaemonImpl{
			daemon: daemon{
				Name: daemonname.PDNS,
			},
			zoneInventory: zi3,
		},
	}
	daemon4 := &keaDaemon{
		daemon: daemon{
			Name: daemonname.CA,
		},
		connector: newKeaConnector(AccessPoint{Type: AccessPointControl, Address: "localhost", Port: 45634}, HTTPClientConfig{}),
	}
	daemonMonitor.daemons = append(daemonMonitor.daemons, daemon0, daemon1, daemon2, daemon3, daemon4)
	daemonMonitor.refreshDaemons(t.Context(), agentManager)

	require.Eventually(t, func() bool {
		for _, daemon := range daemonMonitor.daemons {
			var zoneInventory zoneInventory
			switch concreteDaemon := daemon.(type) {
			case *Bind9Daemon:
				zoneInventory = concreteDaemon.zoneInventory
			case *pdnsDaemon:
				zoneInventory = concreteDaemon.zoneInventory
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
