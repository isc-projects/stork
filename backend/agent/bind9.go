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

	// Patterns for detecting named process.
	bind9Pattern           = regexp.MustCompile(`(.*?)named(\s+.*)?`)
	bind9ChrootPattern     = regexp.MustCompile(`-t\s+(\S+)`)
	bind9ConfigPathPattern = regexp.MustCompile(`-c\s+(\S+)`)
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

// Determine executable using base named directory or system default paths.
func determineBinPath(baseNamedDir, executable string, executor storkutil.CommandExecutor) (string, error) {
	// look for executable in base named directory and sbin or bin subdirectory
	if baseNamedDir != "" {
		for _, binDir := range []string{"sbin", "bin"} {
			fullPath := filepath.Join(baseNamedDir, binDir, executable)
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
// It returns a structure containing the information about the detected files, the
// chroot directory and the base named directory.
func detectBind9ConfigPaths(p supportedProcess, executor storkutil.CommandExecutor, explicitConfigPath string) (*detectedDaemonFiles, error) {
	// We can't proceed without the command line.
	cmdline, err := p.getCmdline()
	if err != nil {
		return nil, err
	}

	// The command line must contain named.
	match := bind9Pattern.FindStringSubmatch(cmdline)
	if len(match) < 3 {
		return nil, errors.Errorf("failed to find named in cmdline: %s", cmdline)
	}

	cwd, err := p.getCwd()
	if err != nil {
		log.WithError(err).Warn("Cannot get named process current working directory")
	}

	var (
		baseNamedDir  string
		bind9ConfPath string
		chrootDir     string
	)

	// Check if the chroot directory is specified in the command line.
	bind9Params := match[2]
	if m := bind9ChrootPattern.FindStringSubmatch(bind9Params); m != nil {
		// Remove extraneous trailing slashes.
		chrootDir = filepath.Clean(m[1])
		// Remove the chroot directory from the current working directory.
		// It leaves us with a path within the chroot directory.
		cwd = strings.TrimPrefix(cwd, chrootDir)
	}

	// STEP 1: Let's try to parse -c parameter passed to named.
	log.Debug("Looking for BIND 9 config file in -c parameter of a running process.")
	if m := bind9ConfigPathPattern.FindStringSubmatch(bind9Params); m != nil {
		bind9ConfPath = filepath.Clean(m[1])
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
		if explicitConfigPath != "" {
			log.Debugf("Looking for BIND 9 config in %s as explicitly specified in settings.", explicitConfigPath)
			switch {
			case !strings.HasPrefix(explicitConfigPath, chrootDir):
				log.Errorf("The explicitly specified config path must be inside the chroot directory: %s, got: %s", chrootDir, explicitConfigPath)
			case executor.IsFileExist(explicitConfigPath):
				// Trim the chroot directory.
				bind9ConfPath = explicitConfigPath[len(chrootDir):]
			default:
				log.Errorf("File explicitly specified in settings (%s) not found or unreadable.", explicitConfigPath)
			}
		}
	}

	// STEP 3: If we still don't have anything, let's try to run named -V and
	// parse its output.

	// determine base named directory
	if namedDir := match[1]; namedDir != "" {
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
			fullPath := filepath.Join(chrootDir, f, "named.conf")
			log.Debugf("Looking for BIND 9 config file in %s", fullPath)
			if executor.IsFileExist(fullPath) {
				bind9ConfPath = f
				break
			}
		}
	}
	if bind9ConfPath == "" {
		return nil, errors.Errorf("BIND 9 config file not found")
	}

	// Create a structure to store the detected files.
	detectedFiles := newDetectedDaemonFiles(chrootDir, baseNamedDir)
	if err := detectedFiles.addFile(detectedFileTypeConfig, bind9ConfPath, executor); err != nil {
		return nil, err
	}

	// The rndc key file is optional.
	rndcKeyPath := filepath.Join(filepath.Dir(bind9ConfPath), RndcKeyFile)
	if executor.IsFileExist(filepath.Join(chrootDir, rndcKeyPath)) {
		if err := detectedFiles.addFile(detectedFileTypeRndcKey, rndcKeyPath, executor); err != nil {
			return nil, err
		}
	}
	return detectedFiles, nil
}

// Parses the BIND 9 config and rndc key files. It extracts the RNDC and statistics
// channel connection parameters.
func configureBind9Daemon(p supportedProcess, files *detectedDaemonFiles, parser bind9FileParser, executor storkutil.CommandExecutor) (*Bind9Daemon, error) {
	configPath := files.getFirstFilePathByType(detectedFileTypeConfig)
	rndcKeyPath := files.getFirstFilePathByType(detectedFileTypeRndcKey)
	chrootDir := files.chrootDir

	// Parse the BIND 9 config file.
	bind9Config, err := parser.ParseFile(configPath, chrootDir)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to parse BIND 9 config file")
	}

	// Resolve include statements.
	bind9Config, err = bind9Config.Expand()
	if err != nil {
		return nil, errors.WithMessage(err, "failed to resolve include statements in BIND 9 config file")
	}

	if bind9Config.HasNoParse() {
		// If some of the configuration parts are elided, it may cause issues with
		// interactions of the Stork agent with BIND 9. The user should be warned.
		log.Warn("BIND 9 config file contains @stork:no-parse directives. Skipping parsing selected config parts improves performance but may cause issues with interactions of the Stork agent with BIND 9. Make sure that you understand the implications of eliding selected config parts, e.g., allow-transfer statements in zones.")
	}

	// rndc.key file typically contains keys to be used for rndc authentication.
	var rndcConfig *bind9config.Config
	if rndcKeyPath != "" {
		rndcConfig, err = parser.ParseFile(rndcKeyPath, chrootDir)
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
	rndcClient := NewRndcClient(executor)
	err = rndcClient.DetermineDetails(
		files.baseDir,
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
func detectBind9Daemon(p supportedProcess, executor storkutil.CommandExecutor, explicitConfigPath string, parser bind9FileParser, existingDaemons ...Daemon) (Daemon, error) {
	detectedFiles, err := detectBind9ConfigPaths(p, executor, explicitConfigPath)
	if err != nil {
		err = errors.WithMessage(err, "failed to detect BIND 9 config path")
		return nil, err
	}
	log.WithFields(log.Fields{
		"path": detectedFiles.getFirstFilePathByType(detectedFileTypeConfig),
	}).Debug("BIND 9 config path detected")

	// Check if the detected files match the files of the existing daemon.
	// If they do, we can use the existing daemon and skip parsing the config files.
	for _, existingDaemon := range existingDaemons {
		bind9Daemon, ok := existingDaemon.(*Bind9Daemon)
		if !ok {
			continue
		}
		if bind9Daemon.getDetectedFiles().isSame(detectedFiles) {
			if !bind9Daemon.getDetectedFiles().isChanged() {
				return existingDaemon, nil
			}
			// Configuration files have changed. We will have to parse the updated config files.
			log.Debug("BIND 9 config files have changed, parsing updated config files")
			break
		}
	}
	// Parse the updated config files.
	daemon, err := configureBind9Daemon(p, detectedFiles, parser, executor)
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
