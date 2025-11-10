package agent

import (
	"fmt"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	bind9config "isc.org/stork/appcfg/bind9"
	storkutil "isc.org/stork/util"
)

var (
	_ App             = (*Bind9App)(nil)
	_ bind9FileParser = (*bind9config.Parser)(nil)

	// Patterns for detecting named process.
	bind9Pattern       = regexp.MustCompile(`(.*?)named\s+(.*)`)
	bind9ChrootPattern = regexp.MustCompile(`-t\s+(\S+)`)
)

// An interface for parsing BIND 9 configuration files.
// It is mocked in the tests.
type bind9FileParser interface {
	ParseFile(path string) (*bind9config.Config, error)
}

// Represents the BIND 9 process metadata.
type Bind9Daemon struct {
	Pid     int32
	Name    string
	Version string
	Active  bool
}

// Represents the state of BIND 9.
type Bind9State struct {
	Version string
	Active  bool
	Daemon  Bind9Daemon
}

// It holds common and BIND 9 specific runtime information.
type Bind9App struct {
	BaseApp
	RndcClient    *RndcClient // to communicate with BIND 9 via rndc
	zoneInventory *zoneInventory
}

// Get base information about BIND 9 app.
func (ba *Bind9App) GetBaseApp() *BaseApp {
	return &ba.BaseApp
}

// Detect allowed logs provided by BIND 9.
// TODO: currently it is not implemented and not used,
// it returns always empty list and no error.
func (ba *Bind9App) DetectAllowedLogs() ([]string, error) {
	return nil, nil
}

// Stops the zone inventory.
func (ba *Bind9App) StopZoneInventory() {
	if ba.zoneInventory != nil {
		ba.zoneInventory.stop()
	}
}

// Returns the zone inventory instance associated with the BIND 9 app.
func (ba *Bind9App) GetZoneInventory() *zoneInventory {
	return ba.zoneInventory
}

// List of BIND 9 executables used during app detection.
const (
	namedCheckconfExec = "named-checkconf"
	rndcExec           = "rndc"
	namedExec          = "named"
)

// rndc-related file names.
const (
	RndcKeyFile           = "rndc.key"
	RndcConfigurationFile = "rndc.conf"
)

// Object for interacting with named using rndc.
type RndcClient struct {
	executor    storkutil.CommandExecutor
	BaseCommand []string
}

// Create an rndc client to communicate with BIND 9 named daemon.
func NewRndcClient(ce storkutil.CommandExecutor) *RndcClient {
	rndcClient := &RndcClient{
		executor: ce,
	}
	return rndcClient
}

// Determine rndc details in the system.
// It find rndc executable and prepare base command with all necessary
// parameters including rndc secret key.
func (rc *RndcClient) DetermineDetails(baseNamedDir, bind9ConfDir string, ctrlAddress string, ctrlPort int64, ctrlKey *bind9config.Key) error {
	rndcPath, err := determineBinPath(baseNamedDir, rndcExec, rc.executor)
	if err != nil {
		return err
	}
	rndcKeyPath := path.Join(bind9ConfDir, RndcKeyFile)
	rndcConfPath := path.Join(bind9ConfDir, RndcConfigurationFile)

	cmd := []string{rndcPath, "-s", ctrlAddress, "-p", fmt.Sprintf("%d", ctrlPort)}

	if ctrlKey != nil {
		cmd = append(cmd, "-y")
		cmd = append(cmd, ctrlKey.Name)
		switch {
		case rc.executor.IsFileExist(rndcConfPath):
			cmd = append(cmd, "-c")
			cmd = append(cmd, rndcConfPath)
		case rc.executor.IsFileExist(rndcKeyPath):
			cmd = append(cmd, "-c")
			cmd = append(cmd, rndcKeyPath)
		default:
			log.Warnf("Could not determine rndc key file in the %s directory. It may be wrong detected by BIND 9.", bind9ConfDir)
		}
	} else {
		keyPath := path.Join(bind9ConfDir, RndcKeyFile)
		if !rc.executor.IsFileExist(keyPath) {
			return errors.Errorf("the rndc key file %s does not exist", keyPath)
		}
		cmd = append(cmd, "-k")
		cmd = append(cmd, keyPath)
	}
	rc.BaseCommand = cmd
	return nil
}

// Send command to named using rndc executable.
func (rc *RndcClient) SendCommand(command []string) (output []byte, err error) {
	var rndcCommand []string
	rndcCommand = append(rndcCommand, rc.BaseCommand...)
	rndcCommand = append(rndcCommand, command...)
	log.Debugf("Rndc: %+v", rndcCommand)

	if len(rndcCommand) == 0 {
		return nil, errors.New("no rndc command specified")
	}

	return rc.executor.Output(rndcCommand[0], rndcCommand[1:]...)
}

// Determine executable using base named directory or system default paths.
func determineBinPath(baseNamedDir, executable string, executor storkutil.CommandExecutor) (string, error) {
	// look for executable in base named directory and sbin or bin subdirectory
	if baseNamedDir != "" {
		for _, binDir := range []string{"sbin", "bin"} {
			fullPath := path.Join(baseNamedDir, binDir, executable)
			if executor.IsFileExist(fullPath) {
				return fullPath, nil
			}
		}
	}

	// not found so try to find generally in the system
	fullPath, err := executor.LookPath(executable)
	if err != nil {
		return "", errors.Errorf("cannot determine location of %s", executable)
	}
	return fullPath, nil
}

// Get potential locations of BIND configs: named.conf, rndc.conf and others.
func getPotentialNamedConfLocations() []string {
	return []string{
		"/etc/bind/",                  // default for many Linux distros
		"/etc/opt/isc/isc-bind/",      // default for make install?
		"/etc/opt/isc/scls/isc-bind/", // default for some RHEL installs
		"/usr/local/etc/namedb/",      // default for FreeBSD
	}
}

// Parses output of the named -V, which contains default paths for named, rndc
// configurations among many other things. Returns one string:
// default path for named configuration (may be empty if parsing fails).

// The output of named -V contains the following info we're looking for:
//
// default paths:
//
//	named configuration:  /etc/bind/named.conf
//	rndc configuration:   /etc/bind/rndc.conf
//	DNSSEC root key:      /etc/bind/bind.keys
//	nsupdate session key: //run/named/session.key
//	named PID file:       //run/named/named.pid
//	named lock file:      //run/named/named.lock
//	geoip-directory:      /usr/share/GeoIP
func parseNamedDefaultPath(output []byte) string {
	// Using []byte is inconvenient to use, let's convert to plain string first.
	text := string(output)

	// Let's use regexp to find interesting string.
	namedConfPattern := regexp.MustCompile(`named configuration: *(\/.*)`)
	namedConfMatch := namedConfPattern.FindStringSubmatch(text)

	// If found, Match is an array of strings. match[0] is always full string,
	// match[1] is the first (and only in this case) group.
	if len(namedConfMatch) < 2 {
		log.Warnf("Unable to find 'named configuration:' line in 'named -V' output.")
		return ""
	}

	return namedConfMatch[1]
}

// Detect the BIND 9 application by parsing the named process command line.
// If the path to the configuration file is relative and chroot directory is
// not specified, the path is resolved against the current working directory of
// the process. If the chroot directory is specified, the path is resolved
// against it.
//
// When the configuration file path cannot be determined from the command line,
// the function tries to parse the output of the named -V command. As a last
// resort, it tries to find the configuration file in the default locations.
// Optionally, an explicit path to the configuration file can be provided.
//
// Here is the summary of the configuration file detection process:
//
// Step 1: Try to parse -c parameter of the running process.
// Step 2: Checks if the explicit config path is defined and exists. If it is, uses that.
// Step 3: Try to parse output of the named -V command.
// Step 4: Try to find named.conf in the default locations.

// The function reads the configuration file and extracts its control address,
// port, and secret key (if configured). The returned instance lacks information
// about the active daemons. It must be detected separately.
//
// It returns the BIND 9 app instance or an error if the BIND 9 is not
// recognized or any error occurs.
//
// ToDo: Enable the linter check after splitting this function in #1991.
//
//nolint:gocyclo
func detectBind9App(p supportedProcess, executor storkutil.CommandExecutor, explicitConfigPath string, parser bind9FileParser) (App, error) {
	cmdline, err := p.getCmdline()
	if err != nil {
		return nil, err
	}
	cwd, err := p.getCwd()
	if err != nil {
		log.WithError(err).Warn("Cannot get named process current working directory")
	}
	match := bind9Pattern.FindStringSubmatch(cmdline)
	if match == nil {
		return nil, errors.Wrapf(err, "failed to find named cmdline: %s", cmdline)
	}
	if len(match) < 3 {
		return nil, errors.Errorf("failed to parse named cmdline: %s", cmdline)
	}

	// Try to find bind9 config file(s).
	namedDir := match[1]
	bind9Params := match[2]
	// Path to actual chroot (empty if not used).
	rootPrefix := ""
	// Absolute path from the actual root or chroot.
	bind9ConfPath := ""

	// Look for the chroot directory.
	m := bind9ChrootPattern.FindStringSubmatch(bind9Params)
	if m != nil {
		rootPrefix = strings.TrimRight(m[1], "/")

		// The cwd path is already prefixed with the chroot directory
		// because the /proc/(pid)/cwd is absolute.
		cwd = strings.TrimPrefix(cwd, rootPrefix)
	}

	// Look for config file in cmd params.
	configPathPattern := regexp.MustCompile(`-c\s+(\S+)`)
	m = configPathPattern.FindStringSubmatch(bind9Params)

	// STEP 1: Let's try to parse -c parameter passed to named.
	log.Debug("Looking for BIND 9 config file in -c parameter of a running process.")
	if m != nil {
		bind9ConfPath = m[1]
		// If path to config is not absolute then join it with CWD of named.
		if !path.IsAbs(bind9ConfPath) {
			bind9ConfPath = path.Join(cwd, bind9ConfPath)
		}
	}

	// STEP 2: Check if the config path is explicitly specified in settings. If
	// it is, we'll use whatever value is provided. User knows best *cough*.
	// We assume it is an absolute path and it includes the chroot directory if
	// any.
	if bind9ConfPath == "" {
		if explicitConfigPath != "" {
			log.Debugf("Looking for BIND 9 config in %s as explicitly specified in settings.", explicitConfigPath)
			switch {
			case !strings.HasPrefix(explicitConfigPath, rootPrefix):
				log.Errorf("The explicitly specified config path must be inside the chroot directory: %s, got: %s", rootPrefix, explicitConfigPath)
			case executor.IsFileExist(explicitConfigPath):
				// Trim the root prefix.
				bind9ConfPath = explicitConfigPath[len(rootPrefix):]
			default:
				log.Errorf("File explicitly specified in settings (%s) not found or unreadable.", explicitConfigPath)
			}
		}
	}

	// STEP 3: If we still don't have anything, let's try to run named -V and
	// parse its output.

	// determine base named directory
	baseNamedDir := ""
	if namedDir != "" {
		// remove sbin or bin at the end
		baseNamedDir, _ = filepath.Split(strings.TrimRight(namedDir, "/"))
	}

	if bind9ConfPath == "" {
		log.Debugf("Looking for BIND 9 config file in output of `named -V`.")
		namedPath, err := determineBinPath(baseNamedDir, namedExec, executor)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to determine BIND 9 executable %s", namedExec)
		}
		out, err := executor.Output(namedPath, "-V")
		if err != nil {
			return nil, errors.Wrapf(err, "failed to run '%s -V'", namedPath)
		}
		bind9ConfPath = parseNamedDefaultPath(out)
	}

	// STEP 4: If we still don't have anything, let's look at typical locations.
	if bind9ConfPath == "" {
		log.Debugf("Looking for BIND 9 config file in typical locations.")
		// config path not found in cmdline params so try to guess its location
		for _, f := range getPotentialNamedConfLocations() {
			// Concat with root or chroot.
			fullPath := path.Join(rootPrefix, f, "named.conf")
			log.Debugf("Looking for BIND 9 config file in %s", fullPath)
			if executor.IsFileExist(fullPath) {
				bind9ConfPath = f
				break
			}
		}
	}

	// no config file so nothing to do
	if bind9ConfPath == "" {
		return nil, errors.Errorf("cannot find config file for BIND 9")
	}
	prefixedBind9ConfPath := path.Join(rootPrefix, bind9ConfPath)

	// Parse the BIND 9 config file.
	bind9Config, err := parser.ParseFile(prefixedBind9ConfPath)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse BIND 9 config file %s", prefixedBind9ConfPath)
	}

	// Resolve include statements.
	bind9Config, err = bind9Config.Expand(filepath.Dir(prefixedBind9ConfPath))
	if err != nil {
		return nil, errors.Wrapf(err, "failed to resolve include statements in BIND 9 config file %s", prefixedBind9ConfPath)
	}

	if bind9Config.HasNoParse() {
		// If some of the configuration parts are elided, it may cause issues with
		// interactions of the Stork agent with BIND 9. The user should be warned.
		log.Warn("BIND 9 config file contains @stork:no-parse directives. Skipping parsing selected config parts improves performance but may cause issues with interactions of the Stork agent with BIND 9. Make sure that you understand the implications of eliding selected config parts, e.g., allow-transfer statements in zones.")
	}

	// rndc.key file typically contains keys to be used for rndc authentication.
	var rndcConfig *bind9config.Config
	prefixedRndcKeyPath := filepath.Join(filepath.Dir(prefixedBind9ConfPath), RndcKeyFile)
	if executor.IsFileExist(prefixedRndcKeyPath) {
		rndcConfig, err = parser.ParseFile(prefixedRndcKeyPath)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to parse BIND 9 rndc key file %s", prefixedRndcKeyPath)
		}
	}

	// look for control address in config
	ctrlAddress, ctrlPort, ctrlKey, enabled, err := bind9Config.GetRndcConnParams(rndcConfig)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get BIND 9 rndc credentials")
	}
	if !enabled {
		return nil, errors.Errorf("found BIND 9 config file (%s) but rndc support was disabled (empty `controls` clause)", prefixedBind9ConfPath)
	}

	rndcKey := ""
	if ctrlKey != nil {
		algorithm, secret, err := ctrlKey.GetAlgorithmSecret()
		if err != nil {
			return nil, err
		}
		rndcKey = fmt.Sprintf("%s:%s:%s", ctrlKey.Name, algorithm, secret)
	}

	accessPoints := []AccessPoint{
		{
			Type:    AccessPointControl,
			Address: *ctrlAddress,
			Port:    *ctrlPort,
			Key:     rndcKey,
		},
	}

	// look for statistics channel address in config
	var inventory *zoneInventory
	address, port, enabled := bind9Config.GetStatisticsChannelConnParams()
	if enabled {
		accessPoints = append(accessPoints, AccessPoint{
			Type:    AccessPointStatistics,
			Address: *address,
			Port:    *port,
		})
		client := NewBind9StatsClient()
		// For larger deployments, it may take several minutes to retrieve the
		// zones from the BIND9 server.
		client.SetRequestTimeout(time.Minute * 3)
		inventory = newZoneInventory(newZoneInventoryStorageMemory(), bind9Config, client, *address, *port)
	} else {
		log.Warn("BIND 9 `statistics-channels` clause unparsable or not found. Neither statistics export nor zone viewer will work.")
		log.Warn("To fix this problem, please configure `statistics-channels` in named.conf and ensure Stork-agent is able to access it.")
		log.Warn("The `statistics-channels` clause must contain explicit `allow` statement.")
	}

	// determine rndc details
	rndcClient := NewRndcClient(executor)
	err = rndcClient.DetermineDetails(
		baseNamedDir,
		// rndc client doesn't support chroot.
		path.Dir(prefixedBind9ConfPath),
		*ctrlAddress,
		*ctrlPort,
		ctrlKey,
	)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to determine BIND 9 rndc details")
	}

	// prepare final BIND 9 app
	bind9App := &Bind9App{
		BaseApp: BaseApp{
			Type:         AppTypeBind9,
			AccessPoints: accessPoints,
		},
		RndcClient:    rndcClient,
		zoneInventory: inventory,
	}

	return bind9App, nil
}

// Send a command to named using rndc client.
func (ba *Bind9App) sendCommand(command []string) (output []byte, err error) {
	return ba.RndcClient.SendCommand(command)
}
