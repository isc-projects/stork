package agent

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	storkutil "isc.org/stork/util"
)

var _ App = (*Bind9App)(nil)

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

// Represents the RNDC key entry.
type Bind9RndcKey struct {
	Name      string
	Algorithm string
	Secret    string
}

// Returns the string representation of the key.
func (k *Bind9RndcKey) String() string {
	return fmt.Sprintf("%s:%s:%s", k.Name, k.Algorithm, k.Secret)
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

// Waits for the zone inventory to complete background tasks.
func (ba *Bind9App) AwaitBackgroundTasks() {
	if ba.zoneInventory != nil {
		ba.zoneInventory.awaitBackgroundTasks()
	}
}

// List of BIND 9 executables used during app detection.
const (
	namedCheckconfExec = "named-checkconf"
	rndcExec           = "rndc"
	namedExec          = "named"
)

// RNDC-related file names.
const (
	RndcKeyFile           = "rndc.key"
	RndcConfigurationFile = "rndc.conf"
)

// Default ports for rndc and stats channel.
const (
	RndcDefaultPort         = 953
	StatsChannelDefaultPort = 80
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
func (rc *RndcClient) DetermineDetails(baseNamedDir, bind9ConfDir string, ctrlAddress string, ctrlPort int64, ctrlKey *Bind9RndcKey) error {
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
			log.Warnf("Could not determine RNDC key file in the %s directory. It may be wrong detected by BIND 9.", bind9ConfDir)
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

// getRndcKey looks for the key with a given `name` in `contents`.
// If `name` is empty, first key is returned. This is useful, if parsing
// the default /etc/bind/rndc.key file. No matter the name, we want to
// take it (or first in case there are multiple)
//
// Example key clause:
//
//	key "name" {
//		algorithm "hmac-sha256";
//		secret "OmItW1lOyLVUEuvv+Fme+Q==";
//	};
func getRndcKey(contents, name string) (controlKey *Bind9RndcKey) {
	pattern := regexp.MustCompile(`(?s)key\s+\"(\S+)\"\s+\{(.*?)\}\s*;`)
	keys := pattern.FindAllStringSubmatch(contents, -1)
	if len(keys) == 0 {
		return nil
	}

	for _, key := range keys {
		// skip this key if the name doesn't match. If user didn't specify
		// a name, use the first key.
		if len(name) > 0 && key[1] != name {
			continue
		}

		// This regex matches both quoted (algorithm "hmac-sha256") and
		// unquoted (algorithm hmac-sha256).
		pattern = regexp.MustCompile(`algorithm\s+"?(\S+?)"?;`)
		algorithm := pattern.FindStringSubmatch(key[2])
		if len(algorithm) < 2 {
			log.Warnf("No key algorithm found for name %s", name)
			return nil
		}

		pattern = regexp.MustCompile(`(?s)secret\s+\"(\S+)\";`)
		secret := pattern.FindStringSubmatch(key[2])
		if len(secret) < 2 {
			log.Warnf("No key secret found for name %s", name)
			return nil
		}

		// this key clause matches the name we are looking for
		controlKey = &Bind9RndcKey{
			Name:      key[1],
			Algorithm: algorithm[1],
			Secret:    secret[1],
		}
		break
	}

	return controlKey
}

// parseInetSpec parses an inet statement from a named configuration excerpt.
// The inet statement is defined by inet_spec:
//
//	inet_spec = ( ip_addr | * ) [ port ip_port ]
//				allow { address_match_list }
//				keys { key_list };
//
// This function returns the ip_addr, port and the first key that is
// referenced in the key_list.  If instead of an ip_addr, the asterisk (*) is
// specified, this function will return 'localhost' as an address.
func parseInetSpec(config, excerpt string) (address string, port int64, key *Bind9RndcKey) {
	pattern := regexp.MustCompile(
		`(?s)` + // Enable multiline mode (\s matches a new line).
			`inet` + // Fixed prefix statement.
			`\s+` + // Mandatory spacing.
			// First matching group.
			`(\S+\s*\S*\s*\d*)` + // ( ip_addr | * ) [ port ip_port ]
			`\s+` + // Mandatory spacing.
			`allow` + // Fixed statement.
			`\s*\{\s*` + // Opening clause surrounded by optional spacing.
			// Non-matching group repeated zero or more times.
			`(?:\s*\S+\s*;\s*)*` + // address_match_list
			`\s*\}` + // Closing clause prepended by an optional spacing.
			// Second matching group.
			`(.*)` + // keys { key_list }; (pattern matched below)
			`;`, // Trailing semicolon.
	)
	match := pattern.FindStringSubmatch(excerpt)
	if len(match) == 0 {
		log.Warnf("Cannot parse BIND 9 inet configuration: no match (%+v)", config)
		return "", 0, nil
	}

	inetSpec := regexp.MustCompile(`\s+`).Split(match[1], 3)
	switch len(inetSpec) {
	case 1:
		address = inetSpec[0]
	case 3:
		address = inetSpec[0]
		if inetSpec[1] != "port" {
			log.Warnf("Cannot parse BIND 9 control port: bad port statement (%+v)", inetSpec)
			return "", 0, nil
		}

		iPort, err := strconv.Atoi(inetSpec[2])
		if err != nil {
			log.Warnf("Cannot parse BIND 9 control port: %+v (%+v)", inetSpec, err)
			return "", 0, nil
		}
		port = int64(iPort)
	case 2:
	default:
		log.Warnf("Cannot parse BIND 9 inet_spec configuration: no match (%+v)", inetSpec)
		return "", 0, nil
	}

	if len(match) == 3 {
		// Find a key clause. This pattern is build up like this:
		// keys\s*                - keys
		// \{\s*                  - {
		// \"(.*?)\"\s*;\s*       - key_list (first)
		// (?:\s*\".*?\"\s*;\s*)* - key_list (remainder)
		// \}                     - }
		pattern = regexp.MustCompile(`(?s)keys\s*\{\s*\"(.*?)\"\s*;\s*(?:\s*\".*?\"\s*;\s*)*\}`)

		keyName := pattern.FindStringSubmatch(match[2])
		if len(keyName) > 1 {
			key = getRndcKey(config, keyName[1])
			if key == nil {
				log.WithField("key", keyName[1]).Warn("Cannot find key details")
			}
		}
	}

	// The named-checkconf tool converts the asterisk (*) to the IPv4 wildcard
	// address (0.0.0.0). I'm not sure if this worked the same way in previous
	// versions of BIND9, so I keep the old behavior for now. I cannot find an
	// exact place where the wildcard is converted in the BIND9 source code
	// then I added the check for the zero IPv6 address too to be sure.
	if address == "*" || address == "0.0.0.0" || address == "::" {
		address = "localhost"
	}

	return address, port, key
}

// getCtrlAddressFromBind9Config retrieves the rndc control access address,
// port, and secret key (if configured) from the configuration `text`.
//
// We need to cover the following cases:
// - no controls clause - BIND9 will open a control socket on localhost
// - empty controls clause - no control socket will be opened
// - controls clause with keys - BIND9 will open a control socket
//
// Multiple controls clauses may be configured but currently this function
// only matches the first one.  Multiple access points may be listed inside
// a single controls clause, but this function currently only matches the
// first in the list.  A controls clause may look like this:
//
//		controls {
//			inet 127.0.0.1 allow {localhost;};
//			inet * port 7766 allow {"rndc-users";};
//	        keys {"rndc-remote";};
//		};
//
// In this example, "rndc-users" and "rndc-remote" refer to an acl and key
// clauses.
//
// Finding the key is done by looking if the control access point has a
// keys parameter and if so, it looks in `path` for a key clause with the
// same name.
func getCtrlAddressFromBind9Config(text string) (controlAddress string, controlPort int64, controlKey *Bind9RndcKey) {
	// Match the following clause:
	//     controls {
	//         inet inet_spec [inet_spec] ;
	//     };
	// or
	//     controls { /maybe some whitespace chars here/ };
	pattern := regexp.MustCompile(`(?s)controls\s*\{\s*(.*)\s*\}\s*;`)
	controls := pattern.FindStringSubmatch(text)
	if len(controls) == 0 {
		// Try to load rndc key from the default locations.
		for _, f := range getPotentialNamedConfLocations() {
			rndcPath := path.Join(f, RndcKeyFile)
			txt, err := os.ReadFile(rndcPath)
			if err != nil {
				log.Debugf("Tried to load %s, but failed: %v", rndcPath, err)
			}

			controlKey = getRndcKey(string(txt), "")
			if controlKey != nil {
				log.Debugf("Loaded rdnc key %s from the default location (%s)", controlKey.Name, rndcPath)
				break
			}
		}

		// We need to consider two cases here. First, there's rndc key file and we found it
		// (the loop above found it). The rndc channel is configured, we detected the key
		// and all is good. The alternative, however, is that there is no rndc key file on disk
		// or we haven't found it. BIND will start without it, but will have rndc channel
		// disabled. In this case we technically found BIND, but can't communicate with it.
		// Our code doesn't have a good way to represent "BIND found, but can't communicate"
		// scenario. In such case we return the defaults (127.0.0.0, port 953), but the key
		// is nil and BIND doesn't listen.
		log.Debugf("BIND9 has no `controls` clause, assuming defaults (127.0.0.1, port 953)")
		return "127.0.0.1", 953, controlKey
	}

	// See if there's any non-whitespace characters in the controls clause.
	// If not, there's `controls {};`, which means: disable control socket.
	txt := strings.TrimSpace(controls[1])
	if len(txt) == 0 {
		log.Debugf("BIND9 has rndc support disabled (empty 'controls' found)")
		return "", 0, nil
	}

	// We only pick the first match, but the controls clause
	// can list multiple control access points.
	controlAddress, controlPort, controlKey = parseInetSpec(text, controls[1])
	if controlAddress != "" {
		// If no port was provided, use the default rndc port.
		if controlPort == 0 {
			controlPort = RndcDefaultPort
		}
	}

	return controlAddress, controlPort, controlKey
}

// getStatisticsChannelFromBind9Config retrieves the statistics channel access
// address, port, and secret key (if configured) from the configuration `text`.
//
// Multiple statistics-channels clauses may be configured but currently this
// function only matches the first one.  Multiple access points may be listed
// inside a single controls clause, but this function currently only matches
// the first in the list.  A statistics-channels clause may look like this:
//
//	statistics-channels {
//		inet 10.1.10.10 port 8080 allow { 192.168.2.10; 10.1.10.2; };
//		inet 127.0.0.1  port 8080 allow { "stats-clients" };
//	};
//
// In this example, "stats-clients" refers to an acl clause.
func getStatisticsChannelFromBind9Config(text string) (statsAddress string, statsPort int64) {
	// Match the following clause:
	//     statistics-channels {
	//         inet inet_spec [inet_spec] ;
	//     };
	pattern := regexp.MustCompile(`(?s)statistics-channels\s*\{\s*(.*)\s*\}\s*;`)
	channels := pattern.FindStringSubmatch(text)
	if len(channels) == 0 {
		return "", 0
	}

	// We only pick the first match, but the statistics-channels clause
	// can list multiple control access points.
	statsAddress, statsPort, _ = parseInetSpec(text, channels[1])
	if statsAddress != "" {
		// If no port was provided, use the default statistics channel port.
		if statsPort == 0 {
			statsPort = StatsChannelDefaultPort
		}
	}
	return statsAddress, statsPort
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

// Detects the running Bind 9 application.
// It accepts the components of the Bind 9 process name (the "match" argument),
// the current working directory of the process (the "cwd" argument; it may be
// empty), a command executor instance, and an optional, explicit path to the
// configuration that can be checked. It uses multiple steps to attempt
// detection:
//
// Step 1: Try to parse -c parameter of the running process.
// Step 2: Checks if the explicit config path is defined and exists. If it is, uses that.
// Step 3: Try to parse output of the named -V command.
// Step 4: Try to find named.conf in the default locations.
//
// Returns the collected data or nil if the Bind 9 is not recognized or any
// error occurs.
func detectBind9App(match []string, cwd string, executor storkutil.CommandExecutor, explicitConfigPath string) *Bind9App {
	if len(match) < 3 {
		log.Warnf("Problem with parsing BIND 9 cmdline: %s", match[0])
		return nil
	}

	// Try to find bind9 config file(s).
	namedDir := match[1]
	bind9Params := match[2]
	// Path to actual chroot (empty if not used).
	rootPrefix := ""
	// Absolute path from the actual root or chroot.
	bind9ConfPath := ""

	// Look for the chroot directory.
	chrootPathPattern := regexp.MustCompile(`-t\s+(\S+)`)
	m := chrootPathPattern.FindStringSubmatch(bind9Params)
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
			log.Warnf("Could not determine BIND 9 executable %s: %s", namedExec, err)
			return nil
		}
		out, err := executor.Output(namedPath, "-V")
		if err != nil {
			log.Warnf("Attempt to run '%s -V' failed. I give up. Error: %s", namedPath, err)
			return nil
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
		log.Warnf("Cannot find config file for BIND 9")
		return nil
	}
	prefixedBind9ConfPath := path.Join(rootPrefix, bind9ConfPath)

	// run named-checkconf on main config file and get preprocessed content of whole config
	namedCheckconfPath, err := determineBinPath(baseNamedDir, namedCheckconfExec, executor)
	if err != nil {
		log.Warnf("Cannot find BIND 9 %s: %s", namedCheckconfExec, err)
		return nil
	}

	// Prepare named-checkconf arguments.
	args := []string{}
	if rootPrefix != "" {
		args = append(args, "-t", rootPrefix)
	}
	// The config path must be last.
	args = append(args, "-p", bind9ConfPath)

	out, err := executor.Output(namedCheckconfPath, args...)
	if err != nil {
		log.Warnf("Cannot parse BIND 9 config file %s: %+v; %s", prefixedBind9ConfPath, err, out)
		return nil
	}
	cfgText := string(out)

	// look for control address in config
	ctrlAddress, ctrlPort, ctrlKey := getCtrlAddressFromBind9Config(cfgText)
	if ctrlPort == 0 || len(ctrlAddress) == 0 {
		log.Warnf("Found BIND 9 config file (%s) but rndc support was disabled (empty `controls` clause)", prefixedBind9ConfPath)
		return nil
	}

	rndcKey := ""
	if ctrlKey != nil {
		rndcKey = ctrlKey.String()
	}

	accessPoints := []AccessPoint{
		{
			Type:    AccessPointControl,
			Address: ctrlAddress,
			Port:    ctrlPort,
			Key:     rndcKey,
		},
	}

	// look for statistics channel address in config
	var inventory *zoneInventory
	address, port := getStatisticsChannelFromBind9Config(cfgText)
	if port > 0 && len(address) != 0 {
		accessPoints = append(accessPoints, AccessPoint{
			Type:    AccessPointStatistics,
			Address: address,
			Port:    port,
		})
		inventory = newZoneInventory(newZoneInventoryStorageMemory(), NewBind9StatsClient(), address, port)
	} else {
		log.Warn("BIND 9 `statistics-channels` clause unparsable or not found. Neither statistics export nor zone viewer will work.")
		log.Warn("To fix this problem, please configure `statistics-channels` in named.conf and ensure Stork-agent is able to access it.")
	}

	// determine rndc details
	rndcClient := NewRndcClient(executor)
	err = rndcClient.DetermineDetails(
		baseNamedDir,
		// RNDC client doesn't support chroot.
		path.Dir(prefixedBind9ConfPath),
		ctrlAddress,
		ctrlPort,
		ctrlKey,
	)
	if err != nil {
		log.Warnf("Cannot determine BIND 9 rndc details: %s", err)
		return nil
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

	return bind9App
}

// Send a command to named using rndc client.
func (ba *Bind9App) sendCommand(command []string) (output []byte, err error) {
	return ba.RndcClient.SendCommand(command)
}
