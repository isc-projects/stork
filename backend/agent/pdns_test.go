package agent

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	gomock "go.uber.org/mock/gomock"
	pdnsconfig "isc.org/stork/appcfg/pdns"
)

//go:generate mockgen -package=agent -destination=pdnsconfigparsermock_test.go -mock_names=pdnsConfigParser=MockPDNSConfigParser isc.org/stork/agent pdnsConfigParser
//go:generate mockgen -package=agent -destination=commandexecutormock_test.go -mock_names=commandExecutor=MockCommandExecutor isc.org/stork/util CommandExecutor

// Test that the BaseApp structure can be accessed.
func TestPowerDNSAppGetBaseApp(t *testing.T) {
	app := &PDNSApp{
		BaseApp: BaseApp{
			Type: AppTypePowerDNS,
		},
	}
	require.Equal(t, &app.BaseApp, app.GetBaseApp())
}

// Test that no allowed logs are returned.
func TestPowerDNSAppDetectAllowedLogs(t *testing.T) {
	app := &PDNSApp{}
	logs, err := app.DetectAllowedLogs()
	require.NoError(t, err)
	require.Empty(t, logs)
}

// Test that awaiting background tasks doesn't panic when zone inventory is nil.
func TestPowerDNSAppAwaitBackgroundTasksNilZoneInventory(t *testing.T) {
	app := &PDNSApp{}
	require.NotPanics(t, app.AwaitBackgroundTasks)
}

// Test that the zone inventory can be accessed.
func TestPowerDNSAppGetZoneInventory(t *testing.T) {
	app := &PDNSApp{
		zoneInventory: &zoneInventory{},
	}
	require.Equal(t, app.zoneInventory, app.GetZoneInventory())
}

// Test successfully detecting PowerDNS app config path.
func TestDetectPowerDNSAppConfigPath(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	executor := NewMockCommandExecutor(ctrl)
	process := NewMockSupportedProcess(ctrl)
	process.EXPECT().getCmdline().Return("/dir/pdns_server --config-dir=/etc", nil)
	process.EXPECT().getCwd().Return("/etc", nil)

	configPath, err := detectPowerDNSAppConfigPath(process, executor, "")
	require.NoError(t, err)
	require.NotNil(t, configPath)
	require.Equal(t, "/etc/pdns.conf", *configPath)
}

// Test that the PowerDNS config path is correctly detected when no parameters are
// specified. It should use the default config directory.
func TestDetectPowerDNSAppNoConfigDir(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	process := NewMockSupportedProcess(ctrl)
	process.EXPECT().getCmdline().Return("/dir/pdns_server", nil)
	process.EXPECT().getCwd().Return("/etc", nil)

	executor := NewMockCommandExecutor(ctrl)
	executor.EXPECT().IsFileExist(filepath.Join("/etc", "powerdns", "pdns.conf")).DoAndReturn(func(path string) bool {
		return path == filepath.Join("/etc", "powerdns", "pdns.conf")
	})

	configPath, err := detectPowerDNSAppConfigPath(process, executor, "")
	require.NoError(t, err)
	require.NotNil(t, configPath)
	require.Equal(t, "/etc/powerdns/pdns.conf", *configPath)
}

// Test that the PowerDNS config path is correctly detected when the explicit
// config path is specified.
func TestDetectPowerDNSAppExplicitConfigPath(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	process := NewMockSupportedProcess(ctrl)
	process.EXPECT().getCmdline().Return("/dir/pdns_server", nil)
	process.EXPECT().getCwd().Return("/etc", nil)

	executor := NewMockCommandExecutor(ctrl)
	executor.EXPECT().IsFileExist("/etc/custom/powerdns/pdns.conf").Return(true)

	configPath, err := detectPowerDNSAppConfigPath(process, executor, "/etc/custom/powerdns/pdns.conf")
	require.NoError(t, err)
	require.NotNil(t, configPath)
	require.Equal(t, "/etc/custom/powerdns/pdns.conf", *configPath)
}

// Test that the PowerDNS config path is correctly detected when the config
// directory is not specified. It should try to find the config file in
// typical locations.
func TestDetectPowerDNSAppConfigPathPotentialConfLocations(t *testing.T) {
	for _, location := range getPotentialPDNSConfLocations() {
		t.Run(location, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			executor := NewMockCommandExecutor(ctrl)
			executor.EXPECT().IsFileExist(gomock.Any()).AnyTimes().DoAndReturn(func(path string) bool {
				return path == filepath.Join(location, "powerdns.conf")
			})

			process := NewMockSupportedProcess(ctrl)
			process.EXPECT().getCmdline().Return("/dir/pdns_server --config-name=powerdns.conf", nil)
			process.EXPECT().getCwd().Return("/etc", nil)

			configPath, err := detectPowerDNSAppConfigPath(process, executor, "")
			require.NoError(t, err)
			require.NotNil(t, configPath)
			require.Equal(t, filepath.Join(location, "powerdns.conf"), *configPath)
		})
	}
}

// Test that an error is returned when getting a process command line fails.
func TestDetectPowerDNSAppConfigPathCmdLineError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	executor := NewMockCommandExecutor(ctrl)
	process := NewMockSupportedProcess(ctrl)
	process.EXPECT().getCmdline().Return("", errors.New("test error"))

	configPath, err := detectPowerDNSAppConfigPath(process, executor, "")
	require.Error(t, err)
	require.ErrorContains(t, err, "test error")
	require.Nil(t, configPath)
}

// Test that an error is returned when getting a process current working directory fails.
func TestDetectPowerDNSAppConfigPathCwdError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	process := NewMockSupportedProcess(ctrl)
	process.EXPECT().getCmdline().Return("/dir/pdns_server --config-name=pdns.conf", nil)
	process.EXPECT().getCwd().Return("", errors.New("test error"))

	executor := NewMockCommandExecutor(ctrl)
	executor.EXPECT().IsFileExist(filepath.Join("/etc", "powerdns", "pdns.conf")).DoAndReturn(func(path string) bool {
		return path == filepath.Join("/etc", "powerdns", "pdns.conf")
	})

	configPath, err := detectPowerDNSAppConfigPath(process, executor, "")
	require.NoError(t, err)
	require.NotNil(t, configPath)
	require.Equal(t, "/etc/powerdns/pdns.conf", *configPath)
}

// Test that the app can be detected when the chroot directory is used.
func TestDetectPowerDNSAppConfigPathChroot(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	process := NewMockSupportedProcess(ctrl)
	process.EXPECT().getCmdline().Return("/dir/pdns_server --chroot=/chroot --config-dir=/etc --config-name=pdns.conf", nil)
	process.EXPECT().getCwd().Return("/chroot", nil)

	executor := NewMockCommandExecutor(ctrl)
	configPath, err := detectPowerDNSAppConfigPath(process, executor, "")
	require.NoError(t, err)
	require.NotNil(t, configPath)
	require.Equal(t, "/chroot/etc/pdns.conf", *configPath)
}

// Test that custom config directory and name can be specified while detecting
// PowerDNS app.
func TestDetectPowerDNSAppConfigPathConfigDirSpecified(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	process := NewMockSupportedProcess(ctrl)
	process.EXPECT().getCmdline().Return("/dir/pdns_server --config-dir=/opt/etc --config-name=server.conf", nil)
	process.EXPECT().getCwd().Return("/chroot", nil)

	executor := NewMockCommandExecutor(ctrl)
	configPath, err := detectPowerDNSAppConfigPath(process, executor, "")
	require.NoError(t, err)
	require.NotNil(t, configPath)
	require.Equal(t, "/opt/etc/server.conf", *configPath)
}

// Test instantiating and configuring the PowerDNS app using specified config path.
func TestConfigurePowerDNSApp(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	parser := NewMockPDNSConfigParser(ctrl)
	parser.EXPECT().ParseFile("/etc/pdns.conf").DoAndReturn(func(path string) (*pdnsconfig.Config, error) {
		return pdnsconfig.NewParser().Parse(strings.NewReader(defaultPDNSConfig))
	})

	app, err := configurePowerDNSApp("/etc/pdns.conf", parser)
	require.NoError(t, err)
	require.NotNil(t, app)

	require.IsType(t, &PDNSApp{}, app)
	require.Equal(t, AppTypePowerDNS, app.GetBaseApp().Type)
	require.Zero(t, app.GetBaseApp().Pid)
	require.Len(t, app.GetBaseApp().AccessPoints, 1)
	require.Equal(t, AccessPointControl, app.GetBaseApp().AccessPoints[0].Type)
	require.EqualValues(t, 8081, app.GetBaseApp().AccessPoints[0].Port)
	require.Equal(t, "127.0.0.1", app.GetBaseApp().AccessPoints[0].Address)
	require.Equal(t, "stork", app.GetBaseApp().AccessPoints[0].Key)
	require.NotNil(t, app.GetZoneInventory())
}

// Test that an error is returned when parsing the configuration file fails.
func TestConfigurePowerDNSAppParseError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	parser := NewMockPDNSConfigParser(ctrl)
	parser.EXPECT().ParseFile("/etc/pdns.conf").Return(nil, errors.New("test error"))

	app, err := configurePowerDNSApp("/etc/pdns.conf", parser)
	require.Error(t, err)
	require.ErrorContains(t, err, "test error")
	require.Nil(t, app)
}

// Test that default webserver address and port are used when not specified
// in the configuration file.
func TestConfigurePowerDNSAppDefaultWebserver(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	parser := NewMockPDNSConfigParser(ctrl)
	parser.EXPECT().ParseFile("/etc/pdns.conf").DoAndReturn(func(path string) (*pdnsconfig.Config, error) {
		return pdnsconfig.NewParser().Parse(strings.NewReader(`
			api=yes
			webserver=yes
			api-key=stork
		`))
	})

	app, err := configurePowerDNSApp("/etc/pdns.conf", parser)
	require.NoError(t, err)
	require.NotNil(t, app)

	require.IsType(t, &PDNSApp{}, app)
	require.Equal(t, AppTypePowerDNS, app.GetBaseApp().Type)
	require.Zero(t, app.GetBaseApp().Pid)
	require.Len(t, app.GetBaseApp().AccessPoints, 1)
	require.Equal(t, AccessPointControl, app.GetBaseApp().AccessPoints[0].Type)
	require.EqualValues(t, 8081, app.GetBaseApp().AccessPoints[0].Port)
	require.Equal(t, "127.0.0.1", app.GetBaseApp().AccessPoints[0].Address)
	require.Equal(t, "stork", app.GetBaseApp().AccessPoints[0].Key)
	require.NotNil(t, app.GetZoneInventory())
}

// Test that an error is returned when the API key is not specified in the
// configuration file.
func TestConfigurePowerDNSAppNoAPIKey(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	parser := NewMockPDNSConfigParser(ctrl)
	parser.EXPECT().ParseFile("/etc/pdns.conf").DoAndReturn(func(path string) (*pdnsconfig.Config, error) {
		return pdnsconfig.NewParser().Parse(strings.NewReader(`
			api
			webserver=yes
		`))
	})

	app, err := configurePowerDNSApp("/etc/pdns.conf", parser)
	require.Error(t, err)
	require.ErrorContains(t, err, "api-key not found in /etc/pdns.conf")
	require.Nil(t, app)
}

// Test that an error is returned when the webserver is disabled in the
// configuration file.
func TestConfigurePowerDNSAppNoWebserver(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	parser := NewMockPDNSConfigParser(ctrl)
	parser.EXPECT().ParseFile("/etc/pdns.conf").DoAndReturn(func(path string) (*pdnsconfig.Config, error) {
		return pdnsconfig.NewParser().Parse(strings.NewReader(`
			api
			webserver=no
		`))
	})

	app, err := configurePowerDNSApp("/etc/pdns.conf", parser)
	require.Error(t, err)
	require.ErrorContains(t, err, "webserver disabled in /etc/pdns.conf")
	require.Nil(t, app)
}

// Test that an error is returned when the API is disabled in the
// configuration file.
func TestConfigurePowerDNSAppNoAPI(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	parser := NewMockPDNSConfigParser(ctrl)
	parser.EXPECT().ParseFile("/etc/pdns.conf").DoAndReturn(func(path string) (*pdnsconfig.Config, error) {
		return pdnsconfig.NewParser().Parse(strings.NewReader(`
			webserver=yes
		`))
	})

	app, err := configurePowerDNSApp("/etc/pdns.conf", parser)
	require.Error(t, err)
	require.ErrorContains(t, err, "API or webserver disabled in /etc/pdns.conf")
	require.Nil(t, app)
}
