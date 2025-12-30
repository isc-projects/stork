package agent

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	gomock "go.uber.org/mock/gomock"
	pdnsconfig "isc.org/stork/daemoncfg/pdns"
	"isc.org/stork/datamodel/daemonname"
	"isc.org/stork/datamodel/protocoltype"
)

//go:generate mockgen -package=agent -destination=pdnsconfigparsermock_test.go -mock_names=pdnsConfigParser=MockPDNSConfigParser isc.org/stork/agent pdnsConfigParser
//go:generate mockgen -package=agent -destination=commandexecutormock_test.go -mock_names=commandExecutor=MockCommandExecutor isc.org/stork/util CommandExecutor

// Test that the function correctly checks if two PowerDNS daemons are the same.
// Note that it is not checking them for equality. It merely checks if they
// represent the same daemon in terms of their name, access points, and detected
// files.
func TestPowerDNSDaemonIsSame(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	executor := NewMockCommandExecutor(ctrl)
	executor.EXPECT().GetFileInfo("/etc/pdns.conf").AnyTimes().Return(&testFileInfo{}, nil)

	detectedFiles := newDetectedDaemonFiles("", "")
	err := detectedFiles.addFile(detectedFileTypeConfig, "/etc/pdns.conf", executor)
	require.NoError(t, err)

	comparedDaemon := &pdnsDaemon{
		dnsDaemonImpl: dnsDaemonImpl{
			daemon: daemon{
				Name: daemonname.PDNS,
				AccessPoints: []AccessPoint{
					{
						Type:     AccessPointControl,
						Address:  "127.0.0.1",
						Port:     8081,
						Protocol: protocoltype.HTTP,
					},
				},
			},
			detectedFiles: detectedFiles,
		},
	}

	t.Run("same daemon", func(t *testing.T) {
		otherDaemon := &pdnsDaemon{
			dnsDaemonImpl: dnsDaemonImpl{
				daemon: daemon{
					Name: daemonname.PDNS,
					AccessPoints: []AccessPoint{
						{
							Type:     AccessPointControl,
							Address:  "127.0.0.1",
							Port:     8081,
							Protocol: protocoltype.HTTP,
						},
					},
				},
				detectedFiles: detectedFiles,
			},
		}
		require.True(t, comparedDaemon.IsSame(otherDaemon))
	})

	t.Run("different daemon name", func(t *testing.T) {
		otherDaemon := &pdnsDaemon{
			dnsDaemonImpl: dnsDaemonImpl{
				daemon: daemon{
					Name: daemonname.Bind9,
					AccessPoints: []AccessPoint{
						{
							Type:     AccessPointControl,
							Address:  "127.0.0.1",
							Port:     8081,
							Protocol: protocoltype.HTTP,
						},
					},
				},
				detectedFiles: detectedFiles,
			},
		}
		require.False(t, comparedDaemon.IsSame(otherDaemon))
	})

	t.Run("different access points", func(t *testing.T) {
		otherDaemon := &pdnsDaemon{
			dnsDaemonImpl: dnsDaemonImpl{
				daemon: daemon{
					Name: daemonname.PDNS,
					AccessPoints: []AccessPoint{
						{
							Type:     AccessPointControl,
							Address:  "127.0.0.1",
							Port:     8082,
							Protocol: protocoltype.HTTP,
						},
					},
				},
				detectedFiles: detectedFiles,
			},
		}
		require.False(t, comparedDaemon.IsSame(otherDaemon))
	})

	t.Run("different detected files", func(t *testing.T) {
		otherDaemon := &pdnsDaemon{
			dnsDaemonImpl: dnsDaemonImpl{
				daemon: daemon{
					Name: daemonname.PDNS,
					AccessPoints: []AccessPoint{
						{
							Type:     AccessPointControl,
							Address:  "127.0.0.1",
							Port:     8081,
							Protocol: protocoltype.HTTP,
						},
					},
				},
				detectedFiles: nil,
			},
		}
		require.False(t, comparedDaemon.IsSame(otherDaemon))
	})

	t.Run("not a PowerDNS daemon", func(t *testing.T) {
		otherDaemon := &Bind9Daemon{
			dnsDaemonImpl: dnsDaemonImpl{
				daemon: daemon{
					Name: daemonname.Bind9,
					AccessPoints: []AccessPoint{
						{
							Type:     AccessPointControl,
							Address:  "127.0.0.1",
							Port:     8081,
							Protocol: protocoltype.HTTP,
						},
					},
				},
				detectedFiles: detectedFiles,
			},
		}
		require.False(t, comparedDaemon.IsSame(otherDaemon))
	})
}

// Test that the daemon structure can be accessed.
func TestPowerDNSDaemonGetBaseDaemon(t *testing.T) {
	daemon := &pdnsDaemon{
		dnsDaemonImpl: dnsDaemonImpl{
			daemon: daemon{
				Name: daemonname.PDNS,
			},
		},
	}
	require.Equal(t, daemonname.PDNS, daemon.GetName())
}

// Test that the refreshing state of the PowerDNS daemon doesn't return any errors.
func TestPowerDNSDaemonRefreshState(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	agentManager := NewMockAgentManager(ctrl)

	zoneInventory := NewMockZoneInventory(ctrl)
	zoneInventory.EXPECT().populate(gomock.Any()).Return(nil, nil)
	zoneInventory.EXPECT().getCurrentState().Return(&zoneInventoryState{})

	daemon := &pdnsDaemon{dnsDaemonImpl: dnsDaemonImpl{zoneInventory: zoneInventory}}

	// Act
	err := daemon.RefreshState(t.Context(), agentManager)

	// Assert
	require.NoError(t, err)
}

// Test that cleanup doesn't panic when zone inventory is nil.
func TestPowerDNSDaemonCleanupNilZoneInventory(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	zoneInventory := NewMockZoneInventory(ctrl)
	zoneInventory.EXPECT().stop()

	daemon := &pdnsDaemon{dnsDaemonImpl: dnsDaemonImpl{zoneInventory: zoneInventory}}

	// Act & Assert
	require.NotPanics(t, func() {
		err := daemon.Cleanup()
		require.NoError(t, err)
	})
}

// Test that the zone inventory can be accessed.
func TestPowerDNSDaemonGetZoneInventory(t *testing.T) {
	daemon := &pdnsDaemon{dnsDaemonImpl: dnsDaemonImpl{
		zoneInventory: &zoneInventoryImpl{},
	}}
	require.Equal(t, daemon.zoneInventory, daemon.getZoneInventory())
}

// Test that the PowerDNS config detection function returns an error when
// getting the process command line fails.
func TestPowerDNSDaemonCmdLineError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	process := NewMockSupportedProcess(ctrl)
	process.EXPECT().getCmdline().Return("", errors.New("test error"))

	executor := NewMockCommandExecutor(ctrl)

	monitor := newMonitor("", "", HTTPClientConfig{})
	monitor.commander = executor

	configPath, err := monitor.detectPowerDNSConfigPath(process)
	require.Error(t, err)
	require.ErrorContains(t, err, "test error")
	require.Nil(t, configPath)
}

// Test successfully detecting PowerDNS daemon.
func TestDetectPowerDNSDaemon(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	process := NewMockSupportedProcess(ctrl)
	process.EXPECT().getCmdline().Return("/dir/pdns_server --config-dir=/etc", nil)

	parser := NewMockPDNSConfigParser(ctrl)
	parser.EXPECT().ParseFile("/etc/pdns.conf").DoAndReturn(func(path string) (*pdnsconfig.Config, error) {
		return pdnsconfig.NewParser().Parse("pdns.conf", strings.NewReader(defaultPDNSConfig))
	})

	executor := NewMockCommandExecutor(ctrl)
	executor.EXPECT().GetFileInfo("/etc/pdns.conf").Return(&testFileInfo{}, nil)

	monitor := newMonitor("", "", HTTPClientConfig{})
	monitor.pdnsConfigParser = parser
	monitor.commander = executor

	daemon, err := monitor.detectPowerDNSDaemon(process)
	require.NoError(t, err)
	require.NotNil(t, daemon)

	require.IsType(t, &pdnsDaemon{}, daemon)
	require.Equal(t, daemonname.PDNS, daemon.GetName())
	require.Len(t, daemon.GetAccessPoints(), 1)
	require.Equal(t, AccessPointControl, daemon.GetAccessPoints()[0].Type)
	require.EqualValues(t, 8081, daemon.GetAccessPoints()[0].Port)
	require.Equal(t, "127.0.0.1", daemon.GetAccessPoints()[0].Address)
	require.Equal(t, "stork", daemon.GetAccessPoints()[0].Key)

	pdnsDaemon := daemon.(*pdnsDaemon)
	require.NotNil(t, pdnsDaemon.getZoneInventory())
}

// Test that the PowerDNS is correctly detected when no parameters are
// specified. It should use the default config directory.
func TestDetectPowerDNSDaemonNoConfigDir(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	process := NewMockSupportedProcess(ctrl)
	process.EXPECT().getCmdline().Return("/dir/pdns_server", nil)

	executor := NewMockCommandExecutor(ctrl)
	executor.EXPECT().IsFileExist(gomock.Any()).DoAndReturn(func(path string) bool {
		return path == "/etc/powerdns/pdns.conf"
	})
	executor.EXPECT().GetFileInfo("/etc/powerdns/pdns.conf").Return(&testFileInfo{}, nil)

	monitor := newMonitor("", "", HTTPClientConfig{})
	monitor.commander = executor

	detectedFiles, err := monitor.detectPowerDNSConfigPath(process)
	require.NoError(t, err)
	require.NotNil(t, detectedFiles)
	require.Equal(t, "/etc/powerdns/pdns.conf", detectedFiles.getFirstFilePathByType(detectedFileTypeConfig))
}

// Test that the PowerDNS config path is correctly detected when the config-dir
// is specified and points to a non-standard location.
func TestDetectPowerDNSDaemonConfigDir(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	executor := NewMockCommandExecutor(ctrl)
	executor.EXPECT().GetFileInfo("/etc/pdns.conf").Return(&testFileInfo{}, nil)

	process := NewMockSupportedProcess(ctrl)
	process.EXPECT().getCmdline().Return("/dir/pdns_server --config-dir=/etc", nil)

	monitor := newMonitor("", "", HTTPClientConfig{})
	monitor.commander = executor

	detectedFiles, err := monitor.detectPowerDNSConfigPath(process)
	require.NoError(t, err)
	require.NotNil(t, detectedFiles)
	require.Equal(t, "/etc/pdns.conf", detectedFiles.getFirstFilePathByType(detectedFileTypeConfig))
}

// Test that the config directory is correctly detected when it is relative
// and the chroot is not set.
func TestDetectPowerDNSDaemonRelConfigDir(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	executor := NewMockCommandExecutor(ctrl)
	executor.EXPECT().GetFileInfo("/opt/etc/pdns.conf").Return(&testFileInfo{}, nil)

	process := NewMockSupportedProcess(ctrl)
	process.EXPECT().getCmdline().Return("/dir/pdns_server --config-dir=etc", nil)
	process.EXPECT().getCwd().Return("/opt", nil)

	monitor := newMonitor("", "", HTTPClientConfig{})
	monitor.commander = executor

	detectedFiles, err := monitor.detectPowerDNSConfigPath(process)
	require.NoError(t, err)
	require.NotNil(t, detectedFiles)
	require.Equal(t, "/opt/etc/pdns.conf", detectedFiles.getFirstFilePathByType(detectedFileTypeConfig))
}

// Test that an error is returned when getting a process current working directory fails
// when the config directory is relative and the chroot is not set.
func TestDetectPowerDNSDaemonRelConfigDirCwdError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	executor := NewMockCommandExecutor(ctrl)
	process := NewMockSupportedProcess(ctrl)
	process.EXPECT().getCmdline().Return("/dir/pdns_server --config-dir=etc", nil)
	process.EXPECT().getCwd().Return("", errors.New("test error"))

	monitor := newMonitor("", "", HTTPClientConfig{})
	monitor.commander = executor

	detectedFiles, err := monitor.detectPowerDNSConfigPath(process)
	require.Error(t, err)
	require.ErrorContains(t, err, "test error")
	require.Nil(t, detectedFiles)
}

// Test that the config-name parameter is correctly interpreted when detected
// in the PowerDNS process command line.
func TestDetectPowerDNSDaemonConfigName(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	executor := NewMockCommandExecutor(ctrl)
	executor.EXPECT().IsFileExist(gomock.Any()).DoAndReturn(func(path string) bool {
		return path == "/etc/powerdns/pdns-foo.conf"
	})
	executor.EXPECT().GetFileInfo("/etc/powerdns/pdns-foo.conf").Return(&testFileInfo{}, nil)

	process := NewMockSupportedProcess(ctrl)
	process.EXPECT().getCmdline().Return("/dir/pdns_server --config-name=foo", nil)

	monitor := newMonitor("", "", HTTPClientConfig{})
	monitor.commander = executor

	detectedFiles, err := monitor.detectPowerDNSConfigPath(process)
	require.NoError(t, err)
	require.NotNil(t, detectedFiles)
	require.Equal(t, "/etc/powerdns/pdns-foo.conf", detectedFiles.getFirstFilePathByType(detectedFileTypeConfig))
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
	executor.EXPECT().GetFileInfo("/chroot/etc/pdns-foo.conf").Return(&testFileInfo{}, nil)

	monitor := newMonitor("", "", HTTPClientConfig{})
	monitor.commander = executor

	detectedFiles, err := monitor.detectPowerDNSConfigPath(process)
	require.NoError(t, err)
	require.NotNil(t, detectedFiles)
	require.Equal(t, "/etc/pdns-foo.conf", detectedFiles.getFirstFilePathByType(detectedFileTypeConfig))
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
	executor.EXPECT().GetFileInfo("/var/chroot/etc/powerdns/pdns.conf").Return(&testFileInfo{}, nil)

	monitor := newMonitor("", "", HTTPClientConfig{})
	monitor.commander = executor

	detectedFiles, err := monitor.detectPowerDNSConfigPath(process)
	require.NoError(t, err)
	require.NotNil(t, detectedFiles)
	require.Equal(t, "/etc/powerdns/pdns.conf", detectedFiles.getFirstFilePathByType(detectedFileTypeConfig))
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
	executor.EXPECT().GetFileInfo("/var/chroot/etc/powerdns/pdns.conf").Return(&testFileInfo{}, nil)

	monitor := newMonitor("", "", HTTPClientConfig{})
	monitor.commander = executor

	detectedFiles, err := monitor.detectPowerDNSConfigPath(process)
	require.NoError(t, err)
	require.NotNil(t, detectedFiles)
	require.Equal(t, "/etc/powerdns/pdns.conf", detectedFiles.getFirstFilePathByType(detectedFileTypeConfig))
}

// Test that an error is returned when getting a process current working directory fails.
// This is a corner case scenario when the chroot directory is relative.
func TestDetectPowerDNSDaemonRelChrootCwdError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	process := NewMockSupportedProcess(ctrl)
	process.EXPECT().getCmdline().Return("/dir/pdns_server --chroot=chroot", nil)
	process.EXPECT().getCwd().Return("", errors.New("test error"))

	executor := NewMockCommandExecutor(ctrl)

	monitor := newMonitor("", "", HTTPClientConfig{})
	monitor.commander = executor

	detectedFiles, err := monitor.detectPowerDNSConfigPath(process)
	require.Error(t, err)
	require.ErrorContains(t, err, "test error")
	require.Nil(t, detectedFiles)
}

// Test that the PowerDNS config path is correctly detected when the explicit
// config path is specified and the config-dir is not specified in the process
// command line.
func TestDetectPowerDNSDaemonExplicitConfigPath(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	process := NewMockSupportedProcess(ctrl)
	process.EXPECT().getCmdline().Return("/dir/pdns_server", nil)

	executor := NewMockCommandExecutor(ctrl)
	executor.EXPECT().IsFileExist(gomock.Any()).DoAndReturn(func(path string) bool {
		return path == "/etc/custom/powerdns/pdns.conf"
	})
	executor.EXPECT().GetFileInfo("/etc/custom/powerdns/pdns.conf").Return(&testFileInfo{}, nil)

	monitor := newMonitor("", "", HTTPClientConfig{})
	monitor.commander = executor
	monitor.explicitPowerDNSConfigPath = "/etc/custom/powerdns/pdns.conf"

	detectedFiles, err := monitor.detectPowerDNSConfigPath(process)
	require.NoError(t, err)
	require.NotNil(t, detectedFiles)
	require.Equal(t, "/etc/custom/powerdns/pdns.conf", detectedFiles.getFirstFilePathByType(detectedFileTypeConfig))
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
	executor.EXPECT().GetFileInfo("/chroot/etc/custom/powerdns/pdns.conf").Return(&testFileInfo{}, nil)

	monitor := newMonitor("", "", HTTPClientConfig{})
	monitor.commander = executor
	monitor.explicitPowerDNSConfigPath = "/chroot/etc/custom/powerdns/pdns.conf"

	detectedFiles, err := monitor.detectPowerDNSConfigPath(process)
	require.NoError(t, err)
	require.NotNil(t, detectedFiles)
	require.Equal(t, "/etc/custom/powerdns/pdns.conf", detectedFiles.getFirstFilePathByType(detectedFileTypeConfig))
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
	executor.EXPECT().GetFileInfo("/var/chroot/etc/powerdns/pdns.conf").Return(&testFileInfo{}, nil)

	monitor := newMonitor("", "", HTTPClientConfig{})
	monitor.commander = executor

	// Explicit path does not belong to the chroot directory.
	monitor.explicitPowerDNSConfigPath = "/chroot/etc/custom/powerdns/pdns.conf"

	detectedFiles, err := monitor.detectPowerDNSConfigPath(process)
	require.NoError(t, err)
	require.NotNil(t, detectedFiles)
	require.Equal(t, "/etc/powerdns/pdns.conf", detectedFiles.getFirstFilePathByType(detectedFileTypeConfig))
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
	executor.EXPECT().GetFileInfo("/var/chroot/etc/powerdns/pdns.conf").Return(&testFileInfo{}, nil)

	monitor := newMonitor("", "", HTTPClientConfig{})
	monitor.commander = executor

	// Explicit path does not belong to the chroot directory. The
	// explicit path should be ignored and one of the default locations
	// should be used.
	monitor.explicitPowerDNSConfigPath = "/var/pdns.conf"

	detectedFiles, err := monitor.detectPowerDNSConfigPath(process)
	require.NoError(t, err)
	require.NotNil(t, detectedFiles)
	require.Equal(t, "/etc/powerdns/pdns.conf", detectedFiles.getFirstFilePathByType(detectedFileTypeConfig))
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
			executor.EXPECT().GetFileInfo(gomock.Any()).AnyTimes().Return(&testFileInfo{}, nil)

			process := NewMockSupportedProcess(ctrl)
			process.EXPECT().getCmdline().Return("/dir/pdns_server --config-name=custom", nil)

			monitor := newMonitor("", "", HTTPClientConfig{})
			monitor.commander = executor

			configPath, err := monitor.detectPowerDNSConfigPath(process)
			require.NoError(t, err)
			require.NotNil(t, configPath)
			require.Equal(t, filepath.Join(location, "pdns-custom.conf"), configPath.getFirstFilePathByType(detectedFileTypeConfig))
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

	monitor := newMonitor("", "", HTTPClientConfig{})
	monitor.commander = executor

	detectedFiles, err := monitor.detectPowerDNSConfigPath(process)
	require.Error(t, err)
	require.ErrorContains(t, err, "test error")
	require.Nil(t, detectedFiles)
}

// Test instantiating and configuring the PowerDNS app using specified config path.
func TestConfigurePowerDNSApp(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	monitor := newMonitor("", "", HTTPClientConfig{})
	monitor.commander = newTestCommandExecutor().
		addFileInfo("/etc/pdns.conf", &testFileInfo{})

	parser := NewMockPDNSConfigParser(ctrl)
	parser.EXPECT().ParseFile("/etc/pdns.conf").DoAndReturn(func(path string) (*pdnsconfig.Config, error) {
		return pdnsconfig.NewParser().Parse(path, strings.NewReader(defaultPDNSConfig))
	})
	monitor.pdnsConfigParser = parser

	detectedFiles := newDetectedDaemonFiles("", "")
	err := detectedFiles.addFile(detectedFileTypeConfig, "/etc/pdns.conf", monitor.commander)
	require.NoError(t, err)

	daemon, err := monitor.configurePowerDNSDaemon(detectedFiles)
	require.NoError(t, err)
	require.NotNil(t, daemon)

	require.IsType(t, &pdnsDaemon{}, daemon)
	require.Equal(t, daemonname.PDNS, daemon.GetName())
	require.Len(t, daemon.GetAccessPoints(), 1)
	require.Equal(t, AccessPointControl, daemon.GetAccessPoints()[0].Type)
	require.EqualValues(t, 8081, daemon.GetAccessPoints()[0].Port)
	require.Equal(t, "127.0.0.1", daemon.GetAccessPoints()[0].Address)
	require.Equal(t, "stork", daemon.GetAccessPoints()[0].Key)
	require.NotNil(t, daemon.getZoneInventory())
}

// Test that an error is returned when parsing the configuration file fails.
func TestConfigurePowerDNSAppParseError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	monitor := newMonitor("", "", HTTPClientConfig{})
	monitor.commander = newTestCommandExecutor().
		addFileInfo("/etc/pdns.conf", &testFileInfo{})

	parser := NewMockPDNSConfigParser(ctrl)
	parser.EXPECT().ParseFile("/etc/pdns.conf").Return(nil, errors.New("test error"))
	monitor.pdnsConfigParser = parser

	detectedFiles := newDetectedDaemonFiles("", "")
	err := detectedFiles.addFile(detectedFileTypeConfig, "/etc/pdns.conf", monitor.commander)
	require.NoError(t, err)

	daemon, err := monitor.configurePowerDNSDaemon(detectedFiles)
	require.Error(t, err)
	require.ErrorContains(t, err, "test error")
	require.Nil(t, daemon)
}

// Test that default webserver address and port are used when not specified
// in the configuration file.
func TestConfigurePowerDNSAppDefaultWebserver(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	monitor := newMonitor("", "", HTTPClientConfig{})
	monitor.commander = newTestCommandExecutor().
		addFileInfo("/etc/pdns.conf", &testFileInfo{})

	parser := NewMockPDNSConfigParser(ctrl)
	parser.EXPECT().ParseFile("/etc/pdns.conf").DoAndReturn(func(path string) (*pdnsconfig.Config, error) {
		return pdnsconfig.NewParser().Parse(path, strings.NewReader(`
			api=yes
			webserver=yes
			api-key=stork
		`))
	})
	monitor.pdnsConfigParser = parser

	detectedFiles := newDetectedDaemonFiles("", "")
	err := detectedFiles.addFile(detectedFileTypeConfig, "/etc/pdns.conf", monitor.commander)
	require.NoError(t, err)

	daemon, err := monitor.configurePowerDNSDaemon(detectedFiles)
	require.NoError(t, err)
	require.NotNil(t, daemon)

	require.IsType(t, &pdnsDaemon{}, daemon)
	require.Equal(t, daemonname.PDNS, daemon.GetName())
	require.Len(t, daemon.GetAccessPoints(), 1)
	require.Equal(t, AccessPointControl, daemon.GetAccessPoints()[0].Type)
	require.EqualValues(t, 8081, daemon.GetAccessPoints()[0].Port)
	require.Equal(t, "127.0.0.1", daemon.GetAccessPoints()[0].Address)
	require.Equal(t, "stork", daemon.GetAccessPoints()[0].Key)
	require.NotNil(t, daemon.getZoneInventory())
}

// Test that an error is returned when the API key is not specified in the
// configuration file.
func TestConfigurePowerDNSAppNoAPIKey(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	monitor := newMonitor("", "", HTTPClientConfig{})
	monitor.commander = newTestCommandExecutor().
		addFileInfo("/etc/pdns.conf", &testFileInfo{})

	parser := NewMockPDNSConfigParser(ctrl)
	parser.EXPECT().ParseFile("/etc/pdns.conf").DoAndReturn(func(path string) (*pdnsconfig.Config, error) {
		return pdnsconfig.NewParser().Parse(path, strings.NewReader(`
			api
			webserver=yes
		`))
	})
	monitor.pdnsConfigParser = parser

	detectedFiles := newDetectedDaemonFiles("", "")
	err := detectedFiles.addFile(detectedFileTypeConfig, "/etc/pdns.conf", monitor.commander)
	require.NoError(t, err)

	daemon, err := monitor.configurePowerDNSDaemon(detectedFiles)
	require.ErrorContains(t, err, "api-key not found in /etc/pdns.conf")
	require.Nil(t, daemon)
}

// Test that an error is returned when the webserver is disabled in the
// configuration file.
func TestConfigurePowerDNSAppNoWebserver(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	monitor := newMonitor("", "", HTTPClientConfig{})
	monitor.commander = newTestCommandExecutor().
		addFileInfo("/etc/pdns.conf", &testFileInfo{})

	parser := NewMockPDNSConfigParser(ctrl)
	parser.EXPECT().ParseFile("/etc/pdns.conf").DoAndReturn(func(path string) (*pdnsconfig.Config, error) {
		return pdnsconfig.NewParser().Parse(path, strings.NewReader(`
			api
			webserver=no
		`))
	})
	monitor.pdnsConfigParser = parser

	detectedFiles := newDetectedDaemonFiles("", "")
	err := detectedFiles.addFile(detectedFileTypeConfig, "/etc/pdns.conf", monitor.commander)
	require.NoError(t, err)

	daemon, err := monitor.configurePowerDNSDaemon(detectedFiles)
	require.ErrorContains(t, err, "webserver disabled in /etc/pdns.conf")
	require.Nil(t, daemon)
}

// Test that an error is returned when the API is disabled in the
// configuration file.
func TestConfigurePowerDNSAppNoAPI(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	monitor := newMonitor("", "", HTTPClientConfig{})
	monitor.commander = newTestCommandExecutor().
		addFileInfo("/etc/pdns.conf", &testFileInfo{})

	parser := NewMockPDNSConfigParser(ctrl)
	parser.EXPECT().ParseFile("/etc/pdns.conf").DoAndReturn(func(path string) (*pdnsconfig.Config, error) {
		return pdnsconfig.NewParser().Parse(path, strings.NewReader(`
			webserver=yes
		`))
	})
	monitor.pdnsConfigParser = parser

	detectedFiles := newDetectedDaemonFiles("", "")
	err := detectedFiles.addFile(detectedFileTypeConfig, "/etc/pdns.conf", monitor.commander)
	require.NoError(t, err)

	daemon, err := monitor.configurePowerDNSDaemon(detectedFiles)
	require.ErrorContains(t, err, "API or webserver disabled in /etc/pdns.conf")
	require.Nil(t, daemon)
}
