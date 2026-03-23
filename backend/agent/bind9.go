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
	bind9config "isc.org/stork/daemoncfg/bind9"
	"isc.org/stork/datamodel/daemonname"
	"isc.org/stork/datamodel/protocoltype"
	storkutil "isc.org/stork/util"
)

var (
	_ Daemon          = (*Bind9Daemon)(nil)
	_ dnsDaemon       = (*Bind9Daemon)(nil)
	_ bind9FileParser = (*bind9config.Parser)(nil)
)

// An interface for parsing BIND 9 configuration files.
// It is mocked in the tests.
type bind9FileParser interface {
	ParseFile(path string, chrootDir string) (*bind9config.Config, error)
}

// It holds common and BIND 9 specific runtime information.
type Bind9Daemon struct {
	dnsDaemonImpl
	rndcClient    *RndcClient // to communicate with BIND 9 via rndc
	pid           int32       // PID of the named process
	bind9Config   *bind9config.Config
	rndcKeyConfig *bind9config.Config
}

// Checks if the current daemon instance is the same as the other daemon instance.
// Besides checking the name and the access points, it also checks if the detected
// files are the same.
func (b *Bind9Daemon) IsSame(other Daemon) bool {
	switch other := other.(type) {
	case *Bind9Daemon:
		return b.isSame(other)
	default:
		return false
	}
}

// List of BIND 9 executables used during daemon detection.
const (
	rndcExec  = "rndc"
	namedExec = "named"
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
// It prepares base command with all necessary parameters including rndc secret
// key.
func (rc *RndcClient) DetermineDetails(binaryNamedDir, bind9ConfDir string, ctrlAddress string, ctrlPort int64, ctrlKey *bind9config.Key) error {
	rndcPath := filepath.Join(binaryNamedDir, rndcExec)
	rndcKeyPath := filepath.Join(bind9ConfDir, RndcKeyFile)
	rndcConfPath := filepath.Join(bind9ConfDir, RndcConfigurationFile)

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

// Holds the parsed components of a named process command line.
type namedCommandLine struct {
	binaryPath string
	chrootDir  string
	configPath string
}

// Parses the command line arguments of a named process to extract the binary
// path, chroot directory (-t flag), and config path (-c flag).
//
// It scans the arguments for the named binary by comparing
// filepath.Base(arg) == "named". Only arguments before the first
// dash-prefixed argument are considered as the binary path.
//
// Returns nil if no named binary is found.
func parseNamedCommandLine(args []string) *namedCommandLine {
	result := &namedCommandLine{}

	// Phase 1: Find the named binary path. Only look at arguments before the
	// first dash-prefixed argument.
	found := false
	flagsStart := len(args)
	for i, arg := range args {
		if strings.HasPrefix(arg, "-") {
			flagsStart = i
			break
		}
		if filepath.Base(arg) == namedExec {
			result.binaryPath = filepath.Clean(arg)
			found = true
			flagsStart = i + 1
			break
		}
	}
	if !found {
		return nil
	}

	// Phase 2: Parse flags from the remaining arguments.
	flags := args[flagsStart:]
	for i := 0; i < len(flags); i++ {
		flag := flags[i]
		key, value, ok := strings.Cut(flag, "=")

		switch key {
		case "-t":
			if !ok {
				if i+1 >= len(flags) {
					continue
				}
				i++
				value = flags[i]
			}
			result.chrootDir = filepath.Clean(value)
		case "-c":
			if !ok {
				if i+1 >= len(flags) {
					continue
				}
				i++
				value = flags[i]
			}
			result.configPath = filepath.Clean(value)
		}
	}

	return result
}

// Detect the BIND 9 config and rndc key files
//
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
//
// It returns a path to the directory with named binaries and structure
// containing the information about the detected files, the chroot directory.
func (sm *monitor) detectBind9ConfigPaths(p supportedProcess) (string, *detectedDaemonFiles, error) {
	// We can't proceed without the command line.
	args, err := p.getCmdlineSlice()
	if err != nil {
		return "", nil, err
	}

	// The command line must contain named.
	parsedCommandLine := parseNamedCommandLine(args)
	if parsedCommandLine == nil {
		return "", nil, errors.Errorf("failed to find named in cmdline: %s", strings.Join(args, " "))
	}

	cwd, err := p.getCwd()
	if err != nil {
		log.WithError(err).Warn("Cannot get named process current working directory")
	}

	var (
		bind9ConfPath string
		chrootDir     = parsedCommandLine.chrootDir
	)

	// Remove the chroot directory from the current working directory.
	if chrootDir != "" {
		cwd = strings.TrimPrefix(cwd, chrootDir)
	}

	// STEP 1: Let's try to parse -c parameter passed to named.
	log.Debug("Looking for BIND 9 config file in -c parameter of a running process.")
	if parsedCommandLine.configPath != "" {
		bind9ConfPath = parsedCommandLine.configPath
		// If path to config is not absolute then join it with CWD of named.
		if !filepath.IsAbs(bind9ConfPath) {
			bind9ConfPath = filepath.Join(cwd, bind9ConfPath)
		}
	}

	// STEP 2: Check if the config path is explicitly specified in settings. If
	// it is, we'll use whatever value is provided. User knows best *cough*.
	// We assume it is an absolute path and it includes the chroot directory if
	// any.
	if bind9ConfPath == "" {
		if sm.settings.ExplicitBind9ConfigPath != "" {
			log.Debugf("Looking for BIND 9 config in %s as explicitly specified in settings.", sm.settings.ExplicitBind9ConfigPath)
			switch {
			case !strings.HasPrefix(sm.settings.ExplicitBind9ConfigPath, chrootDir):
				log.Errorf("The explicitly specified config path must be inside the chroot directory: %s, got: %s", chrootDir, sm.settings.ExplicitBind9ConfigPath)
			case sm.commander.IsFileExist(sm.settings.ExplicitBind9ConfigPath):
				// Trim the chroot directory.
				bind9ConfPath = sm.settings.ExplicitBind9ConfigPath[len(chrootDir):]
			default:
				log.Errorf("File explicitly specified in settings (%s) not found or unreadable.", sm.settings.ExplicitBind9ConfigPath)
			}
		}
	}

	// STEP 3: If we still don't have anything, let's try to run named -V and
	// parse its output.
	binaryPath := parsedCommandLine.binaryPath
	if !filepath.IsAbs(binaryPath) {
		// The binary path is read from the process info, so it should never
		// be relative but just in case, let's resolve it against the CWD of
		// the process.
		binaryPath = filepath.Join(cwd, binaryPath)
	}
	binaryDir := filepath.Dir(binaryPath)

	if bind9ConfPath == "" {
		log.Debugf("Looking for BIND 9 config file in output of `named -V`.")
		out, err := sm.commander.Output(binaryPath, "-V")
		if err != nil {
			return "", nil, errors.Wrapf(err, "failed to run '%s -V'", binaryPath)
		}
		bind9ConfPath = parseNamedDefaultPath(out)
	}

	// STEP 4: If we still don't have anything, let's look at typical locations.
	if bind9ConfPath == "" {
		log.Debugf("Looking for BIND 9 config file in typical locations.")
		// config path not found in cmdline params so try to guess its location
		for _, f := range getPotentialNamedConfLocations() {
			// Concat with root or chroot.
			fullPath := filepath.Join(chrootDir, f, "named.conf")
			log.Debugf("Looking for BIND 9 config file in %s", fullPath)
			if sm.commander.IsFileExist(fullPath) {
				bind9ConfPath = f
				break
			}
		}
	}
	if bind9ConfPath == "" {
		return "", nil, errors.Errorf("BIND 9 config file not found")
	}

	// Create a structure to store the detected files.
	detectedFiles := newDetectedDaemonFiles(chrootDir)
	if err := detectedFiles.addFile(detectedFileTypeConfig, bind9ConfPath, sm.commander); err != nil {
		return "", nil, err
	}

	// The rndc key file is optional.
	rndcKeyPath := filepath.Join(filepath.Dir(bind9ConfPath), RndcKeyFile)
	if sm.commander.IsFileExist(filepath.Join(chrootDir, rndcKeyPath)) {
		if err := detectedFiles.addFile(detectedFileTypeRndcKey, rndcKeyPath, sm.commander); err != nil {
			return "", nil, err
		}
	}
	return binaryDir, detectedFiles, nil
}

// Parses the BIND 9 config and rndc key files. It extracts the RNDC and statistics
// channel connection parameters.
func (sm *monitor) configureBind9Daemon(p supportedProcess, binaryNamedDir string, files *detectedDaemonFiles) (*Bind9Daemon, error) {
	configPath := files.getFirstFilePathByType(detectedFileTypeConfig)
	rndcKeyPath := files.getFirstFilePathByType(detectedFileTypeRndcKey)
	chrootDir := files.chrootDir

	// Parse the BIND 9 config file.
	bind9Config, err := sm.bind9FileParser.ParseFile(configPath, chrootDir)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to parse BIND 9 config file")
	}

	// Resolve include statements.
	var includes []string
	bind9Config, includes, err = bind9Config.Expand()
	if err != nil {
		return nil, errors.WithMessage(err, "failed to resolve include statements in BIND 9 config file")
	}

	for _, includeFilePath := range includes {
		// Record the included configuration files because later modification of those
		// files should trigger configuration parsing again. It will only be the case if
		// the file belongs to the set because it will permit for detecting file changes.
		if err := files.addFile(detectedFileTypeInclude, includeFilePath, sm.commander); err != nil {
			return nil, err
		}
	}

	if bind9Config.HasNoParse() {
		// If some of the configuration parts are elided, it may cause issues with
		// interactions of the Stork agent with BIND 9. The user should be warned.
		log.Warn("BIND 9 config file contains @stork:no-parse directives. Skipping parsing selected config parts improves performance but may cause issues with interactions of the Stork agent with BIND 9. Make sure that you understand the implications of eliding selected config parts, e.g., allow-transfer statements in zones.")
	}

	// rndc.key file typically contains keys to be used for rndc authentication.
	var rndcConfig *bind9config.Config
	if rndcKeyPath != "" {
		rndcConfig, err = sm.bind9FileParser.ParseFile(rndcKeyPath, chrootDir)
		if err != nil {
			return nil, errors.WithMessage(err, "failed to parse BIND 9 rndc key file")
		}
	}

	// look for control address in config
	ctrlAddress, ctrlPort, ctrlKey, enabled, err := bind9Config.GetRndcConnParams(rndcConfig)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to get BIND 9 rndc credentials")
	}
	if !enabled {
		return nil, errors.Errorf("found BIND 9 config file (%s) but rndc support was disabled (empty `controls` clause)", path.Join(chrootDir, configPath))
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
			Type:     AccessPointControl,
			Address:  *ctrlAddress,
			Port:     *ctrlPort,
			Key:      rndcKey,
			Protocol: protocoltype.RNDC,
		},
	}

	// look for statistics channel address in config
	var inventory zoneInventory
	address, port, enabled := bind9Config.GetStatisticsChannelConnParams()
	if enabled {
		accessPoints = append(accessPoints, AccessPoint{
			Type:     AccessPointStatistics,
			Address:  *address,
			Port:     *port,
			Protocol: protocoltype.HTTP,
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
	rndcClient := NewRndcClient(sm.commander)
	err = rndcClient.DetermineDetails(
		binaryNamedDir,
		// rndc client doesn't support chroot.
		filepath.Join(chrootDir, filepath.Dir(configPath)),
		*ctrlAddress,
		*ctrlPort,
		ctrlKey,
	)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to determine BIND 9 rndc details")
	}

	// prepare final BIND 9 daemon
	daemon := &Bind9Daemon{
		dnsDaemonImpl: dnsDaemonImpl{
			daemon: daemon{
				Name:         daemonname.Bind9,
				AccessPoints: accessPoints,
			},
			zoneInventory: inventory,
			detectedFiles: files,
		},
		rndcClient:    rndcClient,
		pid:           p.getPid(),
		bind9Config:   bind9Config,
		rndcKeyConfig: rndcConfig,
	}

	return daemon, nil
}

// Detects the BIND 9 process, parses its configuration and returns its
// instance with all its access points.
func (sm *monitor) detectBind9Daemon(p supportedProcess) (Daemon, error) {
	namedBinaryDir, detectedFiles, err := sm.detectBind9ConfigPaths(p)
	if err != nil {
		err = errors.WithMessage(err, "failed to detect BIND 9 config path")
		return nil, err
	}
	log.WithFields(log.Fields{
		"path": detectedFiles.getFirstFilePathByType(detectedFileTypeConfig),
	}).Debug("BIND 9 config path detected")

	// Check if the detected files match the files of the existing daemon.
	// If they do, we can use the existing daemon and skip parsing the config files.
	for _, existingDaemon := range sm.daemons {
		bind9Daemon, ok := existingDaemon.(*Bind9Daemon)
		if !ok {
			continue
		}
		if bind9Daemon.getDetectedFiles().isSame(detectedFiles) {
			if !bind9Daemon.getDetectedFiles().isChanged() {
				return existingDaemon, nil
			}
		}
	}

	// Configuration files have changed. We will have to parse the updated config files.
	log.Debug("BIND 9 config files have changed, parsing updated config files")

	// Parse the updated config files.
	daemon, err := sm.configureBind9Daemon(p, namedBinaryDir, detectedFiles)
	if err != nil {
		err = errors.WithMessage(err, "failed to configure BIND 9 daemon")
		return nil, err
	}
	return daemon, nil
}

// Send a command to named using rndc client.
func (b *Bind9Daemon) sendRNDCCommand(command []string) (output []byte, err error) {
	return b.rndcClient.SendCommand(command)
}
