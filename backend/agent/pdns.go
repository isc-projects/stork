package agent

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	pdnsconfig "isc.org/stork/daemoncfg/pdns"
	"isc.org/stork/datamodel/daemonname"
)

var (
	_ Daemon           = (*pdnsDaemon)(nil)
	_ dnsDaemon        = (*pdnsDaemon)(nil)
	_ pdnsConfigParser = (*pdnsconfig.Parser)(nil)
)

// The name of the PowerDNS Authoritative Server binary.
const pdnsServerExec = "pdns_server"

// Holds the parsed components of a pdns_server process command line.
type pdnsServerCommandLine struct {
	binaryPath string
	chrootDir  string
	configDir  string
	configName string
}

// Parses the command line arguments of a pdns_server process to extract the
// binary path and the --chroot, --config-dir, --config-name flags.
//
// It scans the arguments for the pdns_server binary by comparing
// filepath.Base(arg) == "pdns_server". Only arguments before the first
// dash-prefixed argument are considered as the binary path.
//
// Returns nil if no pdns_server binary is found.
func parsePDNSServerCommandLine(args []string) *pdnsServerCommandLine {
	result := &pdnsServerCommandLine{}

	// Phase 1: Find the pdns_server binary path. Only look at arguments
	// before the first dash-prefixed argument.
	found := false
	flagsStart := len(args)
	for i, arg := range args {
		if strings.HasPrefix(arg, "-") {
			flagsStart = i
			break
		}
		if filepath.Base(arg) == pdnsServerExec {
			result.binaryPath = filepath.Clean(arg)
			found = true
			flagsStart = i + 1
			break
		}
	}
	if !found {
		return nil
	}

	// Phase 2: Parse --key=value flags from the remaining arguments.
	flags := args[flagsStart:]
	for i := 0; i < len(flags); i++ {
		flag := flags[i]
		key, value, ok := strings.Cut(flag, "=")
		switch key {
		case "--chroot":
			if !ok {
				if i+1 >= len(flags) {
					continue
				}
				i++
				value = flags[i]
			}
			result.chrootDir = filepath.Clean(strings.TrimRight(value, "/"))
		case "--config-dir":
			if !ok {
				if i+1 >= len(flags) {
					continue
				}
				i++
				value = flags[i]
			}
			result.configDir = filepath.Clean(strings.TrimRight(value, "/"))
		case "--config-name":
			if !ok {
				if i+1 >= len(flags) {
					continue
				}
				i++
				value = flags[i]
			}
			result.configName = value
		}
	}

	return result
}

// Returns potential locations of PowerDNS configs.
func getPotentialPDNSConfLocations() []string {
	return []string{
		"/etc/powerdns/",
		"/etc/pdns/",
		"/usr/local/etc/",
		"/opt/homebrew/etc/powerdns/",
	}
}

// An interface for parsing PowerDNS configuration files.
// It is mocked in the tests.
type pdnsConfigParser interface {
	ParseFile(path string) (*pdnsconfig.Config, error)
}

// Implements the Daemon interface for PowerDNS.
type pdnsDaemon struct {
	dnsDaemonImpl
}

// Checks if the current daemon instance is the same as the other daemon instance.
// Besides checking the name and the access points, it also checks if the detected
// files are the same.
func (p *pdnsDaemon) IsSame(other Daemon) bool {
	switch other := other.(type) {
	case *pdnsDaemon:
		return p.isSame(other)
	default:
		return false
	}
}

// It returns the PowerDNS daemon instance or an error if the PowerDNS is not
// recognized or any error occurs.
func (sm *monitor) detectPowerDNSDaemon(p supportedProcess) (Daemon, error) {
	// PowerDNS server configuration location detection.
	detectedFiles, err := sm.detectPowerDNSConfigPath(p)
	if err != nil {
		err = errors.WithMessage(err, "failed to detect PowerDNS server config path")
		return nil, err
	}
	log.WithFields(log.Fields{
		"path": detectedFiles.getFirstFilePathByType(detectedFileTypeConfig),
	}).Debug("PowerDNS server config path detected")

	// Check if the detected files match the files of the existing daemon.
	// If they do, we can use the existing daemon and skip parsing the config files.
	for _, existingDaemon := range sm.daemons {
		pdnsDaemon, ok := existingDaemon.(*pdnsDaemon)
		if !ok {
			continue
		}
		if pdnsDaemon.getDetectedFiles().isSame(detectedFiles) {
			if !pdnsDaemon.getDetectedFiles().isChanged() {
				return existingDaemon, nil
			}
		}
	}

	// Configuration file has changed. We will have to parse the updated config files.
	log.Debug("PowerDNS config file has changed, parsing the updated config file")

	// Parse and interpret the PowerDNS server configuration.
	daemon, err := sm.configurePowerDNSDaemon(detectedFiles)
	if err != nil {
		err = errors.WithMessage(err, "PowerDNS server configuration is invalid")
		return nil, err
	}
	return daemon, nil
}

// Detects the PowerDNS daemon config path using the following algorithm:
//
// STEP 1: Parse the command line arguments. If the config-dir is specified
// in the process command line it points to the directory with the config file
// actually used by the server.
// - If the config-dir is absolute, use this directory.
// - If the config-dir is relative, use it as relative to the current working
// directory.
// - If the config-dir is relative and chroot is set, return an error given that
// it is impossible to determine the absolute path to the config directory.
//
// STEP 2: If the config path is explicitly specified in settings, use it.
// - Make sure that the path is inside the chroot directory if chroot is set.
//
// STEP 3: Try to find the config file in the common locations:
// - Use the locations returned by getPotentialPDNSConfLocations() function.
// - Prepend the chroot directory if it is set.
func (sm *monitor) detectPowerDNSConfigPath(p supportedProcess) (*detectedDaemonFiles, error) {
	// We can't proceed without the command line.
	args, err := p.getCmdlineSlice()
	if err != nil {
		return nil, err
	}

	// The command line must contain pdns_server.
	parsedCommandLine := parsePDNSServerCommandLine(args)
	if parsedCommandLine == nil {
		return nil, errors.Errorf("failed to find pdns_server in cmdline: %s", strings.Join(args, " "))
	}

	// STEP 1: Let's try to parse --chroot, --config-dir and --config-name parameters passed to pdns_server.
	log.Debug("Looking for PowerDNS config file in --config-dir and --config-name parameters of a running process.")

	var configPath string
	chrootDir := parsedCommandLine.chrootDir
	configDir := parsedCommandLine.configDir
	configName := parsedCommandLine.configName
	log.WithFields(log.Fields{
		"config-dir":  configDir,
		"config-name": configName,
		"chroot":      chrootDir,
	}).Debug("PowerDNS was started with the following command line arguments")

	if chrootDir != "" && !filepath.IsAbs(chrootDir) {
		// If the chroot directory is relative, we can use cwd as chroot because
		// cwd is absolute and points to the chroot directory.
		cwd, err := p.getCwd()
		if err != nil {
			// That's unexpected. We're unable to determine the chroot directory.
			// We can't reliably proceed.
			return nil, errors.Wrapf(err, "failed to get PowerDNS current working directory to determine absolute chroot path")
		}
		log.Debugf("The PowerDNS chroot directory (%s) is relative. Using current working directory (%s) as chroot", chrootDir, cwd)
		chrootDir = cwd
	}

	// The default config file name is pdns.conf. The name can be overridden
	// by passing --config-name parameter. For example, setting --config-name=custom
	// yields a config file named pdns-custom.conf.
	configFileName := "pdns.conf"
	if configName != "" {
		configFileName = fmt.Sprintf("pdns-%s.conf", configName)
		log.Debugf("Using custom config file name: %s", configFileName)
	}

	// If the config directory is specified in the process command line, it points
	// to the directory with the config file actually used by the server.
	if configDir != "" {
		if !filepath.IsAbs(configDir) {
			// The config directory location is relative.
			if chrootDir != "" {
				// Using a relative config directory with chroot is not supported. Suppose
				// we start the server like this:
				// pdns_server --chroot=/home/frank/chroot --config-dir=frank/chroot/etc.
				// It is likely that the config file is under /home/frank/chroot/etc. However,
				// there is no guarantee. It depends on the current working directory from which
				// the pdns_server process was started. We don't know this directory. cwd will
				// rather point me to the chroot directory, not the directory from which the
				// process was started. We still can use other methods to detect the config file
				// but let's log the issue.
				log.Warnf("Config directory (%s) is relative while chroot is set (%s)", configDir, chrootDir)
				log.Warn("Unable to match relative config directory against chroot directory. Falling back to other possible locations")
			} else {
				// The config directory location is relative but the chroot is not set.
				// We can assume that cwd points to the directory from which the process
				// was started.
				cwd, err := p.getCwd()
				if err != nil {
					// That's unexpected. We're unable to determine the chroot directory.
					// We can't reliably proceed.
					return nil, errors.Wrapf(err, "failed to get PowerDNS current working directory to determine absolute config directory path")
				}
				configPath = filepath.Join(cwd, configDir, configFileName)
			}
		} else {
			// The config directory location is absolute. We can simply join it
			// with the config file name and that's where our config resides.
			configPath = filepath.Join(configDir, configFileName)
		}
	}

	// STEP 2: Check if the config path is explicitly specified in settings. If
	// it is, we'll use whatever value is provided.
	if configPath == "" && sm.settings.ExplicitPowerDNSConfigPath != "" {
		var candidatePath string
		log.Debugf("Looking for PowerDNS config in the location explicitly specified in settings: %s", sm.settings.ExplicitPowerDNSConfigPath)
		if chrootDir != "" {
			rel, err := filepath.Rel(chrootDir, sm.settings.ExplicitPowerDNSConfigPath)
			if err != nil || strings.HasPrefix(rel, "..") {
				// The explicit config path does not belong to the chroot directory when
				// it is impossible to build a relative path between the two (error case).
				// If the explicit path is a parent of the chroot directory, it is also
				// wrong (the double dot case).
				log.Errorf("The explicitly specified config path must be inside the chroot directory: %s, got: %s", chrootDir, sm.settings.ExplicitPowerDNSConfigPath)
			} else {
				candidatePath = sm.settings.ExplicitPowerDNSConfigPath
			}
		} else {
			candidatePath = sm.settings.ExplicitPowerDNSConfigPath
		}
		if candidatePath != "" {
			if sm.commander.IsFileExist(candidatePath) {
				configPath = candidatePath
			} else {
				log.Errorf("Explicitly specified PowerDNS config file (%s) not found or unreadable", candidatePath)
			}
		}
	}

	// STEP 3: If the config path is not explicitly specified, we'll try to
	// find it in the potential locations.
	if configPath == "" {
		log.Debugf("Looking for PowerDNS config file in typical locations")
		for _, location := range getPotentialPDNSConfLocations() {
			// Concat with root or chroot.
			path := filepath.Join(chrootDir, location, configFileName)
			log.Debugf("Checking if config file exists: %s", path)
			if sm.commander.IsFileExist(path) {
				configPath = path
				break
			}
		}
	}

	if configPath == "" {
		return nil, errors.Errorf("PowerDNS config file not found")
	}

	detectedFiles := newDetectedDaemonFiles(chrootDir)
	if err := detectedFiles.addFileFromChroot(detectedFileTypeConfig, configPath, sm.commander); err != nil {
		return nil, err
	}

	return detectedFiles, nil
}

// Parses the PowerDNS configuration file specified in the first argument. It extracts
// the webserver configuration and the API key. If the webserver is disabled or the
// API key does not exist it returns an error. Otherwise it instantiates the
// PowerDNS daemon and the zone inventory.
func (sm *monitor) configurePowerDNSDaemon(detectedFiles *detectedDaemonFiles) (*pdnsDaemon, error) {
	// Parse the configuration file.
	configPath := detectedFiles.getFirstFilePathByType(detectedFileTypeConfig)
	chrootDir := detectedFiles.chrootDir
	parsedConfig, err := sm.pdnsConfigParser.ParseFile(filepath.Join(chrootDir, configPath))
	if err != nil {
		return nil, err
	}
	// Get the webserver address and port.
	webserverAddress, webserverPort, enabled := parsedConfig.GetWebserverConfig()
	if !enabled {
		return nil, errors.Errorf("API or webserver disabled in %s", configPath)
	}
	// Get the API key. It is mandatory.
	key := parsedConfig.GetString("api-key")
	if key == nil {
		return nil, errors.Errorf("api-key not found in %s", configPath)
	}
	// Create webserver client.
	client := newPDNSClient()
	// For larger deployments, it may take several minutes to retrieve the
	// zones from the DNS server.
	client.SetRequestTimeout(time.Minute * 3)

	// Create the zone inventory.
	inventory := newZoneInventory(newZoneInventoryStorageMemory(), parsedConfig, client, *webserverAddress, *webserverPort)

	// Create the PowerDNS daemon.
	daemon := &pdnsDaemon{
		dnsDaemonImpl: dnsDaemonImpl{
			daemon: daemon{
				Name: daemonname.PDNS,
				AccessPoints: []AccessPoint{
					{
						Type:    AccessPointControl,
						Address: *webserverAddress,
						Port:    *webserverPort,
						Key:     *key,
					},
				},
			},
			zoneInventory: inventory,
			detectedFiles: detectedFiles,
		},
	}
	return daemon, nil
}
