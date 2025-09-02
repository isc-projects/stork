package agent

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	pdnsconfig "isc.org/stork/appcfg/pdns"
	storkutil "isc.org/stork/util"
)

var (
	_ App              = (*PDNSApp)(nil)
	_ pdnsConfigParser = (*pdnsconfig.Parser)(nil)

	// Pattern for detecting PowerDNS process.
	pdnsPattern = regexp.MustCompile(`(.*?)pdns_server(\s+.*)?`)
)

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

// PDNSApp implements the App interface for PowerDNS.
type PDNSApp struct {
	BaseApp
	zoneInventory *zoneInventory
}

// Returns the base app.
func (pa *PDNSApp) GetBaseApp() *BaseApp {
	return &pa.BaseApp
}

// Returns the allowed logs. Always returns nil.
func (pa *PDNSApp) DetectAllowedLogs() ([]string, error) {
	return nil, nil
}

// Waits for the zone inventory to complete background tasks.
func (pa *PDNSApp) AwaitBackgroundTasks() {
	if pa.zoneInventory != nil {
		pa.zoneInventory.awaitBackgroundTasks()
	}
}

// Returns the zone inventory.
func (pa *PDNSApp) GetZoneInventory() *zoneInventory {
	return pa.zoneInventory
}

// Stops the zone inventory.
func (pa *PDNSApp) StopZoneInventory() {
	if pa.zoneInventory != nil {
		pa.zoneInventory.stop()
	}
}

// Detects the PowerDNS application config path using the following algorithm:
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
func detectPowerDNSAppConfigPath(p supportedProcess, executor storkutil.CommandExecutor, explicitConfigPath string) (*string, error) {
	// We can't proceed without the command line.
	cmdline, err := p.getCmdline()
	if err != nil {
		return nil, err
	}

	// The command line must contain pdns_server.
	match := pdnsPattern.FindStringSubmatch(cmdline)
	if match == nil {
		return nil, errors.Errorf("failed to find pdns_server in cmdline: %s", cmdline)
	}

	// STEP 1: Let's try to parse --chroot, --config-dir and --config-name parameters passed to pdns_server.
	log.Debug("Looking for PowerDNS config file in --config-dir and --config-name parameters of a running process.")

	var configDir, configName, configPath, rootPrefix string
	if len(match) >= 3 {
		// The command line contains parameters. Check if they specify config
		// directory or config name.
		pdnsParams := match[2]
		paramsSlice := strings.Fields(pdnsParams)
		for _, param := range paramsSlice {
			key, value, found := strings.Cut(param, "=")
			if !found {
				continue
			}
			switch key {
			case "--chroot":
				rootPrefix = strings.TrimRight(value, "/")
			case "--config-dir":
				configDir = strings.TrimRight(value, "/")
			case "--config-name":
				configName = value
			}
		}
	}
	log.WithFields(log.Fields{
		"config-dir":  configDir,
		"config-name": configName,
		"chroot":      rootPrefix,
	}).Debug("PowerDNS was started with the following command line arguments")

	if rootPrefix != "" && !filepath.IsAbs(rootPrefix) {
		// If the chroot directory is relative, we can use cwd as chroot because
		// cwd is absolute and points to the chroot directory.
		cwd, err := p.getCwd()
		if err != nil {
			// That's unexpected. We're unable to determine the chroot directory.
			// We can't reliably proceed.
			return nil, errors.Wrapf(err, "failed to get PowerDNS current working directory to determine absolute chroot path")
		}
		log.Debugf("The PowerDNS chroot directory (%s) is relative. Using current working directory (%s) as chroot", rootPrefix, cwd)
		rootPrefix = cwd
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
			if rootPrefix != "" {
				// Using a relative config directory with chroot is not supported. Suppose
				// we start the server like this:
				// pdns_server --chroot=/home/frank/chroot --config-dir=frank/chroot/etc.
				// It is likely that the config file is under /home/frank/chroot/etc. However,
				// there is no guarantee. It depends on the current working directory from which
				// the pdns_server process was started. We don't know this directory. cwd will
				// rather point me to the chroot directory, not the directory from which the
				// process was started. We still can use other methods to detect the config file
				// but let's log the issue.
				log.Warnf("Config directory (%s) is relative while chroot is set (%s)", configDir, rootPrefix)
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
	if configPath != "" {
		// Found the config file location. There is nothing more to do.
		return &configPath, nil
	}

	// STEP 2: Check if the config path is explicitly specified in settings. If
	// it is, we'll use whatever value is provided.
	if explicitConfigPath != "" {
		var candidatePath string
		log.Debugf("Looking for PowerDNS config in the location explicitly specified in settings: %s", explicitConfigPath)
		if rootPrefix != "" {
			rel, err := filepath.Rel(rootPrefix, explicitConfigPath)
			if err != nil || strings.HasPrefix(rel, "..") {
				// The explicit config path does not belong to the chroot directory when
				// it is impossible to build a relative path between the two (error case).
				// If the explicit path is a parent of the chroot directory, it is also
				// wrong (the double dot case).
				log.Errorf("The explicitly specified config path must be inside the chroot directory: %s, got: %s", rootPrefix, explicitConfigPath)
			} else {
				candidatePath = explicitConfigPath
			}
		} else {
			candidatePath = explicitConfigPath
		}
		if candidatePath != "" {
			if executor.IsFileExist(candidatePath) {
				configPath = candidatePath
			} else {
				log.Errorf("Explicitly specified PowerDNS config file (%s) not found or unreadable", candidatePath)
			}
		}
	}

	if configPath != "" {
		// Found the config file location. There is nothing more to do.
		return &configPath, nil
	}

	// STEP 3: If the config path is not explicitly specified, we'll try to
	// find it in the potential locations.
	log.Debugf("Looking for PowerDNS config file in typical locations")
	for _, location := range getPotentialPDNSConfLocations() {
		// Concat with root or chroot.
		path := filepath.Join(rootPrefix, location, configFileName)
		log.Debugf("Checking if config file exists: %s", path)
		if executor.IsFileExist(path) {
			return &path, nil
		}
	}
	return nil, errors.Errorf("PowerDNS config file not found")
}

// Parses the PowerDNS configuration file specified in the first argument. It extracts
// the webserver configuration and the API key. If the webserver is disabled or the
// API key does not exist it returns an error. Otherwise it instantiates the
// PowerDNS app and the zone inventory.
func configurePowerDNSApp(configPath string, parser pdnsConfigParser) (App, error) {
	// Parse the configuration file.
	parsedConfig, err := parser.ParseFile(configPath)
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

	// Create the PowerDNS app.
	pdnsApp := &PDNSApp{
		BaseApp: BaseApp{
			Type: AppTypePowerDNS,
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
	}
	return pdnsApp, nil
}
