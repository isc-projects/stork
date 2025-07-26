package agent

import (
	"path"
	"regexp"
	"strings"
	"time"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	pdnsconfig "isc.org/stork/appcfg/pdns"
)

var (
	_ App              = (*PDNSApp)(nil)
	_ pdnsConfigParser = (*pdnsconfig.Parser)(nil)

	// Pattern for detecting PowerDNS process.
	pdnsPattern = regexp.MustCompile(`(.*?)pdns_server(\s+.*)?`)
)

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
func detectPowerDNSApp(p supportedProcess, parser pdnsConfigParser) (App, error) {
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
			case "--config-dir":
				configDir = value
			case "--config-name":
				configName = value
			}
		}
	}
	if !path.IsAbs(configDir) {
		// PowerDNS configuration is typically stored in /etc/powerdns.
		configDir = path.Join("/etc/powerdns", configDir)
	}
	configPath := path.Join(configDir, configName)
	if rootPrefix != "" {
		configPath = path.Join(rootPrefix, configPath)
	}
	if !path.IsAbs(configPath) {
		// If path to config is not absolute then join it with current working directory.
		configPath = path.Join(cwd, configPath)
	}
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
