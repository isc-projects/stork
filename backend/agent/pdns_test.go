package agent

import (
	"strings"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	gomock "go.uber.org/mock/gomock"
	pdnsconfig "isc.org/stork/appcfg/pdns"
)

//go:generate mockgen -package=agent -destination=pdnsmock_test.go -mock_names=pdnsConfigParser=MockPDNSConfigParser isc.org/stork/agent pdnsConfigParser

// Test that the BaseApp structure can be accessed.
func TestPowerDNSAppGetBaseApp(t *testing.T) {
	app := &pdnsApp{
		BaseApp: BaseApp{
			Type: AppTypePowerDNS,
		},
	}
	require.Equal(t, &app.BaseApp, app.GetBaseApp())
}

// Test that no allowed logs are returned.
func TestPowerDNSAppDetectAllowedLogs(t *testing.T) {
	app := &pdnsApp{}
	logs, err := app.DetectAllowedLogs()
	require.NoError(t, err)
	require.Empty(t, logs)
}

// Test that awaiting background tasks doesn't panic when zone inventory is nil.
func TestPowerDNSAppAwaitBackgroundTasksNilZoneInventory(t *testing.T) {
	app := &pdnsApp{}
	require.NotPanics(t, app.AwaitBackgroundTasks)
}

// Test that the zone inventory can be accessed.
func TestPowerDNSAppGetZoneInventory(t *testing.T) {
	app := &pdnsApp{
		zoneInventory: &zoneInventory{},
	}
	require.Equal(t, app.zoneInventory, app.GetZoneInventory())
}

// Test successfully detecting PowerDNS app.
func TestDetectPowerDNSApp(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	process := NewMockSupportedProcess(ctrl)
	process.EXPECT().getCmdline().Return("/dir/pdns_server --config-dir=/etc", nil)
	process.EXPECT().getCwd().Return("/etc", nil)

	parser := NewMockPDNSConfigParser(ctrl)
	parser.EXPECT().ParseFile("/etc/pdns.conf").DoAndReturn(func(path string) (*pdnsconfig.Config, error) {
		return pdnsconfig.NewParser().Parse(strings.NewReader(defaultPDNSConfig))
	})

	app, err := detectPowerDNSApp(process, parser)
	require.NoError(t, err)
	require.NotNil(t, app)

	require.IsType(t, &pdnsApp{}, app)
	require.Equal(t, AppTypePowerDNS, app.GetBaseApp().Type)
	require.Zero(t, app.GetBaseApp().Pid)
	require.Len(t, app.GetBaseApp().AccessPoints, 1)
	require.Equal(t, AccessPointControl, app.GetBaseApp().AccessPoints[0].Type)
	require.EqualValues(t, 8081, app.GetBaseApp().AccessPoints[0].Port)
	require.Equal(t, "127.0.0.1", app.GetBaseApp().AccessPoints[0].Address)
	require.Equal(t, "stork", app.GetBaseApp().AccessPoints[0].Key)
	require.NotNil(t, app.GetZoneInventory())
}

// Test that an error is returned when getting a process command line fails.
func TestDetectPowerDNSAppCmdLineError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	process := NewMockSupportedProcess(ctrl)
	process.EXPECT().getCmdline().Return("", errors.New("test error"))

	app, err := detectPowerDNSApp(process, nil)
	require.Error(t, err)
	require.ErrorContains(t, err, "test error")
	require.Nil(t, app)
}

// Test that an error is returned when getting a process current working directory fails.
func TestDetectPowerDNSAppCwdError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	process := NewMockSupportedProcess(ctrl)
	process.EXPECT().getCmdline().Return("/dir/pdns_server --config-name=pdns.conf", nil)
	process.EXPECT().getCwd().Return("", errors.New("test error"))

	parser := NewMockPDNSConfigParser(ctrl)
	parser.EXPECT().ParseFile("pdns.conf").DoAndReturn(func(path string) (*pdnsconfig.Config, error) {
		return pdnsconfig.NewParser().Parse(strings.NewReader(defaultPDNSConfig))
	})

	app, err := detectPowerDNSApp(process, parser)
	require.NoError(t, err)
	require.NotNil(t, app)

	require.IsType(t, &pdnsApp{}, app)
	require.Equal(t, AppTypePowerDNS, app.GetBaseApp().Type)
	require.Zero(t, app.GetBaseApp().Pid)
	require.Len(t, app.GetBaseApp().AccessPoints, 1)
	require.Equal(t, AccessPointControl, app.GetBaseApp().AccessPoints[0].Type)
	require.EqualValues(t, 8081, app.GetBaseApp().AccessPoints[0].Port)
	require.Equal(t, "127.0.0.1", app.GetBaseApp().AccessPoints[0].Address)
	require.Equal(t, "stork", app.GetBaseApp().AccessPoints[0].Key)
	require.NotNil(t, app.GetZoneInventory())
}

// Test that the app can be detected when the chroot directory is used.
func TestDetectPowerDNSAppChroot(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	process := NewMockSupportedProcess(ctrl)
	process.EXPECT().getCmdline().Return("/dir/pdns_server --chroot=/chroot --config-dir=/etc --config-name=pdns.conf", nil)
	process.EXPECT().getCwd().Return("/chroot", nil)

	parser := NewMockPDNSConfigParser(ctrl)
	parser.EXPECT().ParseFile("/chroot/etc/pdns.conf").DoAndReturn(func(path string) (*pdnsconfig.Config, error) {
		return pdnsconfig.NewParser().Parse(strings.NewReader(defaultPDNSConfig))
	})

	app, err := detectPowerDNSApp(process, parser)
	require.NoError(t, err)
	require.NotNil(t, app)

	require.IsType(t, &pdnsApp{}, app)
	require.Equal(t, AppTypePowerDNS, app.GetBaseApp().Type)
	require.Zero(t, app.GetBaseApp().Pid)
	require.Len(t, app.GetBaseApp().AccessPoints, 1)
	require.Equal(t, AccessPointControl, app.GetBaseApp().AccessPoints[0].Type)
	require.EqualValues(t, 8081, app.GetBaseApp().AccessPoints[0].Port)
	require.Equal(t, "127.0.0.1", app.GetBaseApp().AccessPoints[0].Address)
	require.Equal(t, "stork", app.GetBaseApp().AccessPoints[0].Key)
	require.NotNil(t, app.GetZoneInventory())
}

// Test that custom config directory and name can be specified while detecting
// PowerDNS app.
func TestDetectPowerDNSAppConfigDir(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	process := NewMockSupportedProcess(ctrl)
	process.EXPECT().getCmdline().Return("/dir/pdns_server --config-dir=/opt/etc --config-name=server.conf", nil)
	process.EXPECT().getCwd().Return("/chroot", nil)

	parser := NewMockPDNSConfigParser(ctrl)
	parser.EXPECT().ParseFile("/opt/etc/server.conf").DoAndReturn(func(path string) (*pdnsconfig.Config, error) {
		return pdnsconfig.NewParser().Parse(strings.NewReader(defaultPDNSConfig))
	})

	app, err := detectPowerDNSApp(process, parser)
	require.NoError(t, err)
	require.NotNil(t, app)

	require.IsType(t, &pdnsApp{}, app)
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
func TestDetectPowerDNSAppParseError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	process := NewMockSupportedProcess(ctrl)
	process.EXPECT().getCmdline().Return("/dir/pdns_server --config-dir=/etc --config-name=pdns.conf", nil)
	process.EXPECT().getCwd().Return("/etc", nil)

	parser := NewMockPDNSConfigParser(ctrl)
	parser.EXPECT().ParseFile("/etc/pdns.conf").Return(nil, errors.New("test error"))

	app, err := detectPowerDNSApp(process, parser)
	require.Error(t, err)
	require.ErrorContains(t, err, "test error")
	require.Nil(t, app)
}

// Test that default webserver address and port are used when not specified
// in the configuration file.
func TestDetectPowerDNSAppDefaultWebserver(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	process := NewMockSupportedProcess(ctrl)
	process.EXPECT().getCmdline().Return("/dir/pdns_server --config-dir=/etc", nil)
	process.EXPECT().getCwd().Return("", nil)

	parser := NewMockPDNSConfigParser(ctrl)
	parser.EXPECT().ParseFile("/etc/pdns.conf").DoAndReturn(func(path string) (*pdnsconfig.Config, error) {
		return pdnsconfig.NewParser().Parse(strings.NewReader(`
			api=yes
			webserver=yes
			api-key=stork
		`))
	})

	app, err := detectPowerDNSApp(process, parser)
	require.NoError(t, err)
	require.NotNil(t, app)

	require.IsType(t, &pdnsApp{}, app)
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
func TestDetectPowerDNSAppNoAPIKey(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	process := NewMockSupportedProcess(ctrl)
	process.EXPECT().getCmdline().Return("/dir/pdns_server --config-dir=/etc", nil)
	process.EXPECT().getCwd().Return("", nil)

	parser := NewMockPDNSConfigParser(ctrl)
	parser.EXPECT().ParseFile("/etc/pdns.conf").DoAndReturn(func(path string) (*pdnsconfig.Config, error) {
		return pdnsconfig.NewParser().Parse(strings.NewReader(`
			api
			webserver=yes
		`))
	})

	app, err := detectPowerDNSApp(process, parser)
	require.Error(t, err)
	require.ErrorContains(t, err, "api-key not found in /etc/pdns.conf")
	require.Nil(t, app)
}

// Test that an error is returned when the webserver is disabled in the
// configuration file.
func TestDetectPowerDNSAppNoWebserver(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	process := NewMockSupportedProcess(ctrl)
	process.EXPECT().getCmdline().Return("/dir/pdns_server --config-dir=/etc", nil)
	process.EXPECT().getCwd().Return("", nil)

	parser := NewMockPDNSConfigParser(ctrl)
	parser.EXPECT().ParseFile("/etc/pdns.conf").DoAndReturn(func(path string) (*pdnsconfig.Config, error) {
		return pdnsconfig.NewParser().Parse(strings.NewReader(`
			api
			webserver=no
		`))
	})

	app, err := detectPowerDNSApp(process, parser)
	require.Error(t, err)
	require.ErrorContains(t, err, "webserver disabled in /etc/pdns.conf")
	require.Nil(t, app)
}

// Test that an error is returned when the API is disabled in the
// configuration file.
func TestDetectPowerDNSAppNoAPI(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	process := NewMockSupportedProcess(ctrl)
	process.EXPECT().getCmdline().Return("/dir/pdns_server --config-dir=/etc", nil)
	process.EXPECT().getCwd().Return("", nil)

	parser := NewMockPDNSConfigParser(ctrl)
	parser.EXPECT().ParseFile("/etc/pdns.conf").DoAndReturn(func(path string) (*pdnsconfig.Config, error) {
		return pdnsconfig.NewParser().Parse(strings.NewReader(`
			webserver=yes
		`))
	})

	app, err := detectPowerDNSApp(process, parser)
	require.Error(t, err)
	require.ErrorContains(t, err, "API disabled in /etc/pdns.conf")
	require.Nil(t, app)
}
