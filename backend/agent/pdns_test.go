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

// Test that the PowerDNS config detection function returns an error when
// getting the process command line fails.
func TestPowerDNSAppCmdLineError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	process := NewMockSupportedProcess(ctrl)
	process.EXPECT().getCmdline().Return("", errors.New("test error"))

	executor := NewMockCommandExecutor(ctrl)
	configPath, err := detectPowerDNSAppConfigPath(process, executor, "")
	require.Error(t, err)
	require.ErrorContains(t, err, "test error")
	require.Nil(t, configPath)
}

// Test that the PowerDNS config path is correctly detected when no command
// line arguments are specified. It should use the default config directory.
func TestDetectPowerDNSAppNoArguments(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	process := NewMockSupportedProcess(ctrl)
	process.EXPECT().getCmdline().Return("/dir/pdns_server", nil)

	executor := NewMockCommandExecutor(ctrl)
	executor.EXPECT().IsFileExist(gomock.Any()).DoAndReturn(func(path string) bool {
		return path == "/etc/powerdns/pdns.conf"
	})

	configPath, err := detectPowerDNSAppConfigPath(process, executor, "")
	require.NoError(t, err)
	require.NotNil(t, configPath)
	require.Equal(t, "/etc/powerdns/pdns.conf", *configPath)
}

// Test that the PowerDNS config path is correctly detected when the config-dir
// is specified and points to a non-standard location.
func TestDetectPowerDNSAppConfigDir(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	executor := NewMockCommandExecutor(ctrl)
	process := NewMockSupportedProcess(ctrl)
	process.EXPECT().getCmdline().Return("/dir/pdns_server --config-dir=/etc", nil)

	configPath, err := detectPowerDNSAppConfigPath(process, executor, "")
	require.NoError(t, err)
	require.NotNil(t, configPath)
	require.Equal(t, "/etc/pdns.conf", *configPath)
}

// Test that the config directory is correctly detected when it is relative
// and the chroot is not set.
func TestDetectPowerDNSAppRelConfigDir(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	executor := NewMockCommandExecutor(ctrl)
	process := NewMockSupportedProcess(ctrl)
	process.EXPECT().getCmdline().Return("/dir/pdns_server --config-dir=etc", nil)
	process.EXPECT().getCwd().Return("/opt", nil)

	configPath, err := detectPowerDNSAppConfigPath(process, executor, "")
	require.NoError(t, err)
	require.NotNil(t, configPath)
	require.Equal(t, "/opt/etc/pdns.conf", *configPath)
}

// Test that an error is returned when getting a process current working directory fails
// when the config directory is relative and the chroot is not set.
func TestDetectPowerDNSAppRelConfigDirCwdError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	executor := NewMockCommandExecutor(ctrl)
	process := NewMockSupportedProcess(ctrl)
	process.EXPECT().getCmdline().Return("/dir/pdns_server --config-dir=etc", nil)
	process.EXPECT().getCwd().Return("", errors.New("test error"))

	configPath, err := detectPowerDNSAppConfigPath(process, executor, "")
	require.Error(t, err)
	require.ErrorContains(t, err, "test error")
	require.Nil(t, configPath)
}

// Test that the config-name parameter is correctly interpreted when detected
// in the PowerDNS process command line.
func TestDetectPowerDNSAppConfigName(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	executor := NewMockCommandExecutor(ctrl)
	executor.EXPECT().IsFileExist(gomock.Any()).DoAndReturn(func(path string) bool {
		return path == "/etc/powerdns/pdns-foo.conf"
	})

	process := NewMockSupportedProcess(ctrl)
	process.EXPECT().getCmdline().Return("/dir/pdns_server --config-name=foo", nil)

	configPath, err := detectPowerDNSAppConfigPath(process, executor, "")
	require.NoError(t, err)
	require.NotNil(t, configPath)
	require.Equal(t, "/etc/powerdns/pdns-foo.conf", *configPath)
}

// Test that the PowerDNS config path is correctly detected when both
// the chroot and config-dir are absolute and the config-dir belongs to
// the chroot directory.
func TestDetectPowerDNSAppChrootAbsConfigDir(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	process := NewMockSupportedProcess(ctrl)
	process.EXPECT().getCmdline().Return("/dir/pdns_server --chroot=/chroot --config-dir=/chroot/etc --config-name=foo", nil)

	executor := NewMockCommandExecutor(ctrl)
	configPath, err := detectPowerDNSAppConfigPath(process, executor, "")
	require.NoError(t, err)
	require.NotNil(t, configPath)
	require.Equal(t, "/chroot/etc/pdns-foo.conf", *configPath)
}

// Test that using chroot and relative config-dir falls back to alternative
// locations.
func TestDetectPowerDNSAppChrootRelConfigDir(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	process := NewMockSupportedProcess(ctrl)
	process.EXPECT().getCmdline().Return("/dir/pdns_server --chroot=/var/chroot --config-dir=chroot/etc", nil)

	executor := NewMockCommandExecutor(ctrl)
	executor.EXPECT().IsFileExist(gomock.Any()).DoAndReturn(func(path string) bool {
		return path == "/var/chroot/etc/powerdns/pdns.conf"
	})

	configPath, err := detectPowerDNSAppConfigPath(process, executor, "")
	require.NoError(t, err)
	require.NotNil(t, configPath)
	require.Equal(t, "/var/chroot/etc/powerdns/pdns.conf", *configPath)
}

// Test that the PowerDNS config path is correctly detected even for a relative
// chroot directory if cwd is correctly set.
func TestDetectPowerDNSAppRelChroot(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Specify relative chroot directory. Since cwd for the chroot case is
	// always set to the absolute path of the chroot directory, we should
	// get correct absolute path by prepending cwd to the config path.
	process := NewMockSupportedProcess(ctrl)
	process.EXPECT().getCmdline().Return("/dir/pdns_server --chroot=chroot", nil)
	process.EXPECT().getCwd().Return("/var/chroot", nil)

	executor := NewMockCommandExecutor(ctrl)
	executor.EXPECT().IsFileExist(gomock.Any()).DoAndReturn(func(path string) bool {
		// Since there is no config-dir, the agent should look for the config
		// in the default locations. The first tried default location is the
		// one below.
		return path == "/var/chroot/etc/powerdns/pdns.conf"
	})

	configPath, err := detectPowerDNSAppConfigPath(process, executor, "")
	require.NoError(t, err)
	require.NotNil(t, configPath)
	require.Equal(t, "/var/chroot/etc/powerdns/pdns.conf", *configPath)
}

// Test that an error is returned when getting a process current working directory fails.
// This is a corner case scenario when the chroot directory is relative.
func TestDetectPowerDNSAppRelChrootCwdError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	process := NewMockSupportedProcess(ctrl)
	process.EXPECT().getCmdline().Return("/dir/pdns_server --chroot=chroot", nil)
	process.EXPECT().getCwd().Return("", errors.New("test error"))

	executor := NewMockCommandExecutor(ctrl)
	configPath, err := detectPowerDNSAppConfigPath(process, executor, "")
	require.Error(t, err)
	require.ErrorContains(t, err, "test error")
	require.Nil(t, configPath)
}

// Test that the PowerDNS config path is correctly detected when the explicit
// config path is specified and the config-dir is not specified in the process
// command line.
func TestDetectPowerDNSAppExplicitConfigPath(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	process := NewMockSupportedProcess(ctrl)
	process.EXPECT().getCmdline().Return("/dir/pdns_server", nil)

	executor := NewMockCommandExecutor(ctrl)
	executor.EXPECT().IsFileExist(gomock.Any()).DoAndReturn(func(path string) bool {
		return path == "/etc/custom/powerdns/pdns.conf"
	})

	configPath, err := detectPowerDNSAppConfigPath(process, executor, "/etc/custom/powerdns/pdns.conf")
	require.NoError(t, err)
	require.NotNil(t, configPath)
	require.Equal(t, "/etc/custom/powerdns/pdns.conf", *configPath)
}

// Test that the explicit PowerDNS config path is respected when this path
// belongs to the chroot directory.
func TestDetectPowerDNSAppExplicitConfigPathChroot(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	process := NewMockSupportedProcess(ctrl)
	process.EXPECT().getCmdline().Return("/dir/pdns_server --chroot=/chroot/", nil)

	executor := NewMockCommandExecutor(ctrl)
	executor.EXPECT().IsFileExist(gomock.Any()).DoAndReturn(func(path string) bool {
		return path == "/chroot/etc/custom/powerdns/pdns.conf"
	})

	configPath, err := detectPowerDNSAppConfigPath(process, executor, "/chroot/etc/custom/powerdns/pdns.conf")
	require.NoError(t, err)
	require.NotNil(t, configPath)
	require.Equal(t, "/chroot/etc/custom/powerdns/pdns.conf", *configPath)
}

// Test that the explicit PowerDNS config path is ignored when it is not
// inside the chroot directory.
func TestDetectPowerDNSAppExplicitConfigPathChrootMismatch(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	process := NewMockSupportedProcess(ctrl)
	process.EXPECT().getCmdline().Return("/dir/pdns_server --chroot=/var/chroot/", nil)

	executor := NewMockCommandExecutor(ctrl)
	executor.EXPECT().IsFileExist(gomock.Any()).DoAndReturn(func(path string) bool {
		// We expect the agent to use a default location rather than the
		// explicitly specified one. That's because the explicit path does
		// not belong to the chroot directory.
		return path == "/var/chroot/etc/powerdns/pdns.conf"
	})

	// Explicit path does not belong to the chroot directory.
	configPath, err := detectPowerDNSAppConfigPath(process, executor, "/chroot/etc/custom/powerdns/pdns.conf")
	require.NoError(t, err)
	require.NotNil(t, configPath)
	require.Equal(t, "/var/chroot/etc/powerdns/pdns.conf", *configPath)
}

// Test that the explicit PowerDNS config path is ignored when it contains
// a path to a file which is in a parent of the chroot directory.
func TestDetectPowerDNSAppExplicitConfigPathInChrootParent(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	process := NewMockSupportedProcess(ctrl)
	process.EXPECT().getCmdline().Return("/dir/pdns_server --chroot=/var/chroot", nil)

	executor := NewMockCommandExecutor(ctrl)
	executor.EXPECT().IsFileExist(gomock.Any()).DoAndReturn(func(path string) bool {
		// We expect the agent to use a default location rather than the
		// explicitly specified one. That's because the explicit path does
		// not belong to the chroot directory.
		return path == "/var/chroot/etc/powerdns/pdns.conf"
	})

	// Explicit path does not belong to the chroot directory. The
	// explicit path should be ignored and one of the default locations
	// should be used.
	configPath, err := detectPowerDNSAppConfigPath(process, executor, "/var/pdns.conf")
	require.NoError(t, err)
	require.NotNil(t, configPath)
	require.Equal(t, "/var/chroot/etc/powerdns/pdns.conf", *configPath)
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
				return path == filepath.Join(location, "pdns-custom.conf")
			})

			process := NewMockSupportedProcess(ctrl)
			process.EXPECT().getCmdline().Return("/dir/pdns_server --config-name=custom", nil)

			configPath, err := detectPowerDNSAppConfigPath(process, executor, "")
			require.NoError(t, err)
			require.NotNil(t, configPath)
			require.Equal(t, filepath.Join(location, "pdns-custom.conf"), *configPath)
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
