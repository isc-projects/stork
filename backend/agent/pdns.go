package agent

import (
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
		"/usr/local/etc/pdns/",
		"/usr/local/etc/powerdns/",
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

// Detect the PowerDNS application by parsing the named process command line.
// If the configuration path is not specified in the command line, the function
// will use the explicitly specified path. If the path is not specified in the
// command line and explicitly specified path is not provided, the function will
// try to find the configuration file in the potential locations.
//
// If the path to the configuration file is relative and chroot directory is
// not specified, the path is resolved against the current working directory of
// the process. If the chroot directory is specified, the path is resolved
// against it.
//
// The function reads the configuration file and extracts webserver address,
// port, and API key (if configured).
//
// It returns the PowerDNS app instance or an error if the PowerDNS is not
// recognized or any error occurs.
func detectPowerDNSApp(p supportedProcess, executor storkutil.CommandExecutor, explicitConfigPath string, parser pdnsConfigParser) (App, error) {
	cmdline, err := p.getCmdline()
	if err != nil {
		return nil, err
	}
	cwd, err := p.getCwd()
	if err != nil {
		log.WithError(err).Warnf("Failed to get %s process current working directory", pdnsProcName)
	}
	match := pdnsPattern.FindStringSubmatch(cmdline)
	if match == nil {
		return nil, errors.Errorf("failed to find pdns_server in cmdline: %s", cmdline)
	}

	// STEP 1: Let's try to parse --config-dir and --config-name parameters passed to pdns_server.
	log.Debug("Looking for PowerDNS config file in --config-dir and --config-name parameters of a running process.")

	configDir := ""
	configName := "pdns.conf"
	rootPrefix := ""
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
				// The cwd path is already prefixed with the chroot directory
				// because the /proc/(pid)/cwd is absolute.
				cwd = strings.TrimPrefix(cwd, rootPrefix)
			case "--config-dir":
				configDir = value
				if !filepath.IsAbs(configDir) {
					configDir = filepath.Join(cwd, configDir)
				}
			case "--config-name":
				configName = value
			}
		}
	}

	configPath := ""
	if configDir != "" {
		configPath = filepath.Join(configDir, configName)
	}

	// STEP 2: Check if the config path is explicitly specified in settings. If
	// it is, we'll use whatever value is provided.
	if configPath == "" && explicitConfigPath != "" {
		log.Debugf("Looking for PowerDNS config in %s as explicitly specified in settings.", explicitConfigPath)
		switch {
		case !strings.HasPrefix(explicitConfigPath, rootPrefix):
			log.Errorf("The explicitly specified config path must be inside the chroot directory: %s, got: %s", rootPrefix, explicitConfigPath)
		case executor.IsFileExist(explicitConfigPath):
			// Trim the root prefix.
			configPath = explicitConfigPath[len(rootPrefix):]
		default:
			log.Errorf("Explicitly specified PowerDNS config file (%s) not found or unreadable.", explicitConfigPath)
		}
	}

	// STEP 3: If the config path is not explicitly specified, we'll try to
	// find it in the potential locations.
	if configPath == "" {
		log.Debugf("Looking for PowerDNS config file in typical locations.")
		for _, location := range getPotentialPDNSConfLocations() {
			// Concat with root or chroot.
			fullPath := filepath.Join(rootPrefix, location, configName)
			log.Debugf("Looking for PowerDNS config file in %s", fullPath)
			if executor.IsFileExist(fullPath) {
				configPath = filepath.Join(location, configName)
				break
			}
		}
	}

	if configPath == "" {
		return nil, errors.Errorf("PowerDNS config file not found")
	}

	configPath = filepath.Join(rootPrefix, configPath)

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
