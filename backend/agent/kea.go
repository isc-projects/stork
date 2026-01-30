package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path"
	"regexp"
	"strings"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	keaconfig "isc.org/stork/daemoncfg/kea"
	keactrl "isc.org/stork/daemonctrl/kea"
	keadata "isc.org/stork/daemondata/kea"
	"isc.org/stork/datamodel/daemonname"
	"isc.org/stork/datamodel/protocoltype"
	storkutil "isc.org/stork/util"
)

var _ Daemon = (*keaDaemon)(nil)

// It holds common and Kea specific runtime information.
type keaDaemon struct {
	daemon
	connector keaConnector // to communicate with Kea daemon
	snooper   MemfileSnooper
}

// Interface to a Kea command that allows overriding the daemon list.
type command interface {
	keactrl.SerializableCommand
	SetDaemonsList(daemons []daemonname.Name)
}

// Sends a command to Kea and returns a response.
func (d *keaDaemon) sendCommand(ctx context.Context, command command, response any) error {
	if d.connector == nil {
		return errors.New("cannot send command to Kea because no control access point is configured")
	}

	// Stork requires that command has exactly one target daemon.
	// However, Kea CA expects that if the command is targeted to itself,
	// the Daemons field must be empty. The daemon must be temporarily removed
	// from the list before sending the command.
	daemons := command.GetDaemonsList()
	if len(daemons) == 1 && daemons[0] == daemonname.CA {
		command.SetDaemonsList(nil)
		defer func() {
			command.SetDaemonsList(daemons)
		}()
	}

	commandBytes, err := command.Marshal()
	if err != nil {
		return errors.WithMessagef(err, "failed to marshal command to JSON")
	}

	// Send the command to the Kea server.
	responseBytes, err := d.connector.sendPayload(ctx, commandBytes)
	if err != nil {
		return errors.WithMessagef(err, "failed to send command to Kea")
	}

	err = json.Unmarshal(responseBytes, response)
	if err != nil {
		return errors.Wrap(err, "failed to parse Kea response")
	}

	return nil
}

// Collect the list of log files which can be viewed by the Stork user
// from the UI. The config variable holds the Kea config fetched by the
// config-get command returned by one of the Kea daemons. If the config
// contains loggers' configuration the log files are extracted from it
// and returned.
func collectKeaAllowedLogs(config *keaconfig.Config) []string {
	loggers := config.GetLoggers()
	if len(loggers) == 0 {
		log.Info("No loggers found in the returned configuration while trying to refresh the viewable log files")
		return nil
	}

	// Go over returned loggers and collect those found in the returned configuration.
	var paths []string
	for _, l := range loggers {
		for _, o := range l.GetAllOutputOptions() {
			// TODO: We could read the stdout and stderr too by reading
			// "/proc/<pid>/fd/1" and "/proc/<pid>/fd/2" symlinks.
			// It is also possible to read syslog.
			if o.Output != "stdout" && o.Output != "stderr" && !strings.HasPrefix(o.Output, "syslog") {
				paths = append(paths, o.Output)
			}
		}
	}
	return paths
}

// Fetches the Kea configuration from the daemon by sending config-get command.
func (d *keaDaemon) fetchConfig(ctx context.Context) (*keaconfig.Config, error) {
	// Prepare config-get command to be sent to Kea Control Agent.
	command := keactrl.NewCommandBase(keactrl.ConfigGet, d.GetName())
	// Send the command to Kea.
	response := keactrl.Response{}
	err := d.sendCommand(ctx, command, &response)
	if err != nil {
		return nil, err
	}

	// It does not make sense to proceed if the CA returned non-success status
	// because this response neither contains logging configuration nor
	// sockets configurations.
	if err := response.GetError(); err != nil {
		return nil, errors.WithMessagef(
			err, "unsuccessful response received from Kea CA to config-get command sent to %s", d,
		)
	}

	if response.Arguments == nil {
		return nil, errors.New("config-get response has no arguments")
	}
	config, err := keaconfig.NewConfig(response.Arguments)
	if err != nil {
		return nil, errors.WithMessage(err, "config-get response contains arguments which could not be parsed")
	}

	return config, nil
}

// Fetches the status of Kea from the daemon by sending the status-get command.
func (d *keaDaemon) fetchStatus(ctx context.Context) (*keaconfig.Status, error) {
	command := keactrl.NewCommandBase(keactrl.StatusGet, d.GetName())
	response := keactrl.Response{}
	err := d.sendCommand(ctx, command, &response)
	if err != nil {
		return nil, err
	}

	if err := response.GetError(); err != nil {
		return nil, errors.WithMessagef(
			err, "unsuccessful response received from Kea to status-get command sent to %s", d,
		)
	}

	if response.Arguments == nil {
		return nil, errors.New("status-get response has no arguments")
	}
	status, err := keaconfig.NewStatus(response.Arguments)
	if err != nil {
		return nil, errors.WithMessage(err, "status-get response contains arguments which could not be parsed")
	}

	return status, nil
}

// Reads the Kea configuration file, resolves the includes, and parses the content.
func readKeaConfig(path string) (*keaconfig.Config, error) {
	text, err := storkutil.ReadFileWithIncludes(path)
	if err != nil {
		err = errors.WithMessage(err, "Cannot read Kea config file")
		return nil, err
	}

	config, err := keaconfig.NewConfig(text)
	if err != nil {
		err = errors.WithMessage(err, "Cannot parse Kea config file")
		return nil, err
	}

	return config, err
}

// Detect the Kea daemon(s).
//
// The communication model with Kea changed significantly with the release of
// Kea 3.0. The Kea Control Agent is no longer required to establish connection
// with the Kea daemons (DHCP, DDNS, etc.). Instead, the daemons provide its
// own control channels. The Kea CA still exists and can be used to manage the
// daemons but it is deprecated and may be removed in future releases.
// The Kea daemons support two modes of control channel: HTTP-based (same as
// the Kea CA) and socket-based. In both cases, the expected data format is
// JSON.
//
// This function supports all Kea versions (prior and post 3.0) and all modes
// of control channel (HTTP- and socket-based).
//
// For Kea prior to 3.0, the function detects multiple daemons if CA daemon is
// passed, and no daemons if any other daemon is passed. It is because only Kea
// CA can contact other daemons in this Kea version. All daemons detected this
// way have the same control access point because they are connected via the
// CA.
// For Kea 3.0 and later, the function detects only the passed daemon because
// it expects the connection will be established directly with the daemon. Each
// daemon has its own control channel.
//
// The access points of the daemons are detected by reading the daemon
// configuration file. The function parses command line of the specified
// process. It looks for the configuration file path in the command line. If
// the path is relative, it is resolved against the current working directory
// of the process.
//
// It reads the configuration file and extracts its HTTP host, port,
// TLS configuration, basic authentication credentials. For Kea prior to 3.0,
// the function also reads the list of configured daemons and then sends the
// version-get command to each daemon to check if it is running.
//
// The version of the Kea daemon is recognized by calling its executable with
// the --version flag.
//
// The monitor's keaHTTPClientConfig is used to create a new HTTP client instance
// for the detected Kea app. The client inherits the the general HTTP client
// configuration from the Stork agent configuration and additionally sets the
// basic authentication credentials if they are provided in the Kea CA
// configuration. It picks the first credentials with the user name "stork" or
// starting with "stork." If there are no such credentials, it picks the first
// one. See @readClientCredentials for details.
//
// It returns the Kea daemon instance or an error if the Kea is not recognized or
// any error occurs.
func (sm *monitor) detectKeaDaemons(ctx context.Context, p supportedProcess) ([]Daemon, error) {
	// Extract the daemon name from the process.
	processName, err := p.getName()
	if err != nil {
		return nil, errors.Wrap(err, "cannot get process name")
	}

	daemonName := p.getDaemonName()
	if daemonName == "" {
		return nil, errors.Errorf("unsupported Kea process: %s", processName)
	}

	// Extract the config path and the executable path from the command line.
	cmdline, err := p.getCmdline()
	if err != nil {
		return nil, err
	}
	cwd, err := p.getCwd()
	if err != nil {
		log.WithError(err).Warn("Cannot get Kea process current working directory")
	}

	pattern := regexp.MustCompile(fmt.Sprintf(`(.*?)%s\s+.*-c\s+(\S+)`, processName))

	match := pattern.FindStringSubmatch(cmdline)
	if match == nil {
		return nil, errors.Errorf("problem parsing Kea command line: %s", cmdline)
	}

	if len(match) < 3 {
		return nil, errors.Errorf("problem parsing Kea command line: %s", match[0])
	}

	// Check the version of the Kea binary. We need to differentiate between
	// Kea prior to 3.0 and Kea post 3.0.
	executablePath := match[1] + processName
	if !path.IsAbs(executablePath) {
		if cwd == "" {
			return nil, errors.New("cannot resolve Kea executable path because the current working directory is unknown")
		}
		executablePath = path.Join(cwd, executablePath)
	}

	versionRaw, err := sm.commander.Output(executablePath, "-v")
	if err != nil {
		return nil, errors.WithMessagef(err, "cannot get Kea version by executing %s -v", executablePath)
	}
	version, err := storkutil.ParseSemanticVersion(string(versionRaw))
	if err != nil {
		return nil, errors.WithMessagef(err, "cannot parse Kea version: %s", string(versionRaw))
	}
	shouldTunnelViaCA := version.LessThan(storkutil.SemanticVersion{Major: 3, Minor: 0, Patch: 0})
	if shouldTunnelViaCA && daemonName != daemonname.CA {
		// For Kea prior to 3.0, only the CA daemon can connect to other daemons.
		// If the process is not CA, we cannot detect any daemons.
		return nil, nil
	}

	// Read the configuration file.
	configPath := match[2]

	if !strings.HasPrefix(configPath, "/") {
		// If path to config is not absolute then join it with CWD of Kea.
		configPath = path.Join(cwd, configPath)
	}

	config, err := readKeaConfig(configPath)
	if err != nil {
		return nil, errors.WithMessagef(err, "invalid Kea %s config: %s", daemonName, configPath)
	}

	controlSockets := config.GetListeningControlSockets()
	var accessPoints []AccessPoint
	var httpClientConfigs []HTTPClientConfig

	for _, controlSocket := range controlSockets {
		// Credentials
		// Key is a user name that Stork uses to authenticate with Kea.
		var key string
		httpClientConfig := sm.keaHTTPClientConfig
		if controlSocket.Authentication != nil {
			allCredentials, err := readClientCredentials(controlSocket.Authentication)
			if err != nil {
				return nil, errors.WithMessage(err, "cannot read client credentials")
			}

			if len(allCredentials) > 0 {
				// Fall back to the first set of credentials.
				credentials := allCredentials[0]

				// Look for the credentials prefixed with "stork".
				for _, c := range allCredentials {
					if strings.HasPrefix(c.User, "stork") {
						credentials = c
						break
					}
				}

				httpClientConfig.BasicAuth = basicAuthCredentials(credentials)
				key = credentials.User
			}
		}

		accessPoint := AccessPoint{
			Type:     AccessPointControl,
			Address:  controlSocket.GetAddress(),
			Port:     controlSocket.GetPort(),
			Protocol: controlSocket.GetProtocol(),
			Key:      key,
		}
		accessPoints = append(accessPoints, accessPoint)
		httpClientConfigs = append(httpClientConfigs, httpClientConfig)
	}

	thisDaemon := &keaDaemon{
		daemon: daemon{
			Name:         daemonName,
			AccessPoints: accessPoints,
		},
		connector: newMultiConnector(accessPoints, httpClientConfigs),
	}

	detectedDaemons := []Daemon{thisDaemon}
	if shouldTunnelViaCA && len(accessPoints) != 0 {
		// For Kea prior to 3.0, get the list of configured daemons.
		managementControlSockets := config.GetManagementControlSockets()
		managedDaemonNames := managementControlSockets.GetManagedDaemonNames()
		for _, managedDaemonName := range managedDaemonNames {
			command := keactrl.NewCommandBase(keactrl.VersionGet, managedDaemonName)
			response := keactrl.Response{}
			err = thisDaemon.sendCommand(ctx, command, &response)
			if err == nil {
				err = response.GetError()
			}
			if err != nil {
				log.WithError(err).WithField("daemon", managedDaemonName).
					Error("Cannot send version-get command to Kea daemon")
				continue
			}

			// Add the detected daemon.
			managedDaemon := &keaDaemon{
				daemon: daemon{
					Name:         managedDaemonName,
					AccessPoints: accessPoints,
				},
				connector: thisDaemon.connector,
			}
			detectedDaemons = append(detectedDaemons, managedDaemon)
		}
	}

	return detectedDaemons, nil
}

type ClientCredentials struct {
	User     string
	Password string
}

// Reads the client credentials.
// Kea supports multiple ways of providing client credentials.
//
// 1. Username and password can be provided directly in the configuration file.
// 2. Username and password can be provided in separate files.
// 3. Username and password can be provided in a separate file delimited by a colon.
// 4. Username can be provided directly in the configuration file and the password in a separate file.
// 5. Username can be provided in the separate file and the password directly in the configuration file.
func readClientCredentials(authentication *keaconfig.Authentication) ([]ClientCredentials, error) {
	allCredentials := []ClientCredentials{}

	directory := "/"
	if authentication.Directory != nil {
		directory = *authentication.Directory
	}

	for _, client := range authentication.Clients {
		var credentials ClientCredentials

		// Read the user.
		switch {
		case client.User != nil:
			// The user provided as a string.
			credentials.User = *client.User
		case client.UserFile != nil:
			// The user is provided in a file.
			userPath := path.Join(directory, *client.UserFile)
			userRaw, err := os.ReadFile(userPath)
			if err != nil {
				return nil, errors.WithMessagef(err,
					"could not read the user file '%s'",
					userPath,
				)
			}
			credentials.User = strings.TrimSpace(string(userRaw))
		case client.PasswordFile != nil:
			// The user and password are provided in a single file.
			passwordPath := path.Join(directory, *client.PasswordFile)
			passwordRaw, err := os.ReadFile(passwordPath)
			if err != nil {
				return nil, errors.WithMessagef(err,
					"could not read the password file '%s'",
					passwordPath,
				)
			}
			parts := strings.Split(strings.TrimSpace(string(passwordRaw)), ":")
			if len(parts) != 2 {
				return nil, errors.Errorf(
					"invalid format of the password file '%s'",
					passwordPath,
				)
			}
			credentials.User = parts[0]
			credentials.Password = parts[1]
		default:
			// Missing user.
			return nil, errors.New(
				"invalid client credentials: neither user nor user-file provided",
			)
		}

		// Read the password.
		switch {
		case credentials.Password != "":
			// The password has been provided together with the user in
			// the password file.
		case client.Password != nil:
			// The password provided as a string.
			credentials.Password = *client.Password
		case client.PasswordFile != nil:
			// The password is provided in a file.
			passwordPath := path.Join(directory, *client.PasswordFile)
			passwordRaw, err := os.ReadFile(passwordPath)
			if err != nil {
				return nil, errors.WithMessagef(err,
					"could not read the password file '%s'",
					passwordPath,
				)
			}
			credentials.Password = strings.TrimSpace(string(passwordRaw))
		default:
			// Missing password.
			return nil, errors.New(
				"invalid client credentials - password or password-file is not provided",
			)
		}

		allCredentials = append(allCredentials, credentials)
	}
	return allCredentials, nil
}

// Lifecycle of the daemon.
// Called once when the daemon is newly detected.
func (d *keaDaemon) Bootstrap() error {
	return nil
}

// Called periodically to update the daemon state.
// Gathers the configured log files for detected apps and enables them
// for viewing from the UI.
func (d *keaDaemon) RefreshState(ctx context.Context, agent agentManager) error {
	config, err := d.fetchConfig(ctx)
	if err != nil {
		return errors.WithMessage(err, "cannot fetch Kea configuration")
	}
	paths := collectKeaAllowedLogs(config)
	err = d.ensureWatchingLeasefile(ctx, config)
	if err != nil {
		return err
	}

	for _, p := range paths {
		agent.allowLog(p)
	}
	return nil
}

// Ensure that this keaDaemon is watching the lease file it's supposed to be
// watching.  This function will use the get-status API to ask Kea for the
// current lease file path (so that we don't have to guess based on defaults and
// whatever's in the config).
func (d *keaDaemon) ensureWatchingLeasefile(ctx context.Context, config *keaconfig.Config) error {
	var leaseDBType string
	var persist bool

	switch d.Name {
	case daemonname.DHCPv4:
		if config != nil &&
			config.DHCPv4Config != nil &&
			config.DHCPv4Config.LeaseDatabase != nil {
			leaseDBType = config.DHCPv4Config.LeaseDatabase.Type
			if config.DHCPv4Config.LeaseDatabase.Persist != nil {
				persist = *config.DHCPv4Config.LeaseDatabase.Persist
			} else {
				// The default when persist is unspecified is true, per https://kea.readthedocs.io/en/stable/arm/dhcp4-srv.html#memfile-basic-storage-for-leases.
				persist = true
			}
		}
	case daemonname.DHCPv6:
		if config != nil &&
			config.DHCPv6Config != nil &&
			config.DHCPv6Config.LeaseDatabase != nil {
			leaseDBType = config.DHCPv6Config.LeaseDatabase.Type
			if config.DHCPv6Config.LeaseDatabase.Persist != nil {
				persist = *config.DHCPv6Config.LeaseDatabase.Persist
			} else {
				// The default when persist is unspecified is true, per https://kea.readthedocs.io/en/stable/arm/dhcp6-srv.html#memfile-basic-storage-for-leases.
				persist = true
			}
		}
	default:
		// do nothing, the variables should stay nil so that it doesn't try to look at a leasefile from D2 or something.
	}

	if leaseDBType == "memfile" && persist {
		// I should likely be watching a leasefile...
		status, err := d.fetchStatus(ctx)
		if err != nil {
			return err
		}
		if status.CSVLeaseFile == nil {
			return errors.New("Kea's configuration says that it is in memfile mode with persistence on, but its status API did not return the path to the lease memfile.")
		}
		if d.snooper == nil {
			// ...but I am currently not.
			rs, err := NewRowSource(*status.CSVLeaseFile)
			if err != nil {
				return err
			}
			d.snooper, err = NewMemfileSnooper(d.Name, rs)
			if err != nil {
				return err
			}
			d.snooper.Start()
		} else {
			// ...and I am, but I should make sure I'm looking at the right file.
			return d.snooper.EnsureWatching(*status.CSVLeaseFile)
		}
	} else if d.snooper != nil {
		// I shouldn't be watching a leasefile, but I currently am, so I should stop.
		// e.g. Kea was reconfigured to use a lease PostgreSQL lease database.
		d.snooper.Stop()
		d.snooper = nil
	}
	return nil
}

// Get a snapshot of all current leases known for this daemon.
func (d *keaDaemon) GetLeaseSnapshot() []*keadata.Lease {
	return d.snooper.GetSnapshot()
}

// Called once before the daemon is removed.
func (d *keaDaemon) Cleanup() error {
	if d.snooper != nil {
		d.snooper.Stop()
	}
	return nil
}

// Interface for sending bytes to Kea and receiving bytes back.
// All kinds of API supported by Kea (HTTP, socket) expect JSON data.
// This abstraction encapsulates the way how the data is sent and received.
type keaConnector interface {
	sendPayload(ctx context.Context, command []byte) ([]byte, error)
}

// Factory function to create a keaConnector based on the access point
// configuration.
func newKeaConnector(accessPoint AccessPoint, httpClientConfig HTTPClientConfig) keaConnector {
	if accessPoint.Protocol == protocoltype.Socket {
		socketPath := accessPoint.Address
		return &keaSocketConnector{socketPath: socketPath}
	}

	// HTTP or HTTPS
	url := storkutil.HostWithPortURL(
		accessPoint.Address,
		accessPoint.Port,
		string(accessPoint.Protocol),
	)
	return &keaHTTPConnector{
		url:        url,
		httpClient: NewHTTPClient(httpClientConfig),
	}
}

// Implements keaConnector interface for connecting to Kea via a Unix socket.
type keaSocketConnector struct {
	socketPath string
}

// Sends the command to Kea via a Unix socket and returns the response.
func (c *keaSocketConnector) sendPayload(ctx context.Context, command []byte) ([]byte, error) {
	var dialer net.Dialer
	conn, err := dialer.DialContext(ctx, "unix", c.socketPath)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to connect to unix socket: %s", c.socketPath)
	}
	defer conn.Close()

	_, err = conn.Write(command)
	if err != nil {
		return nil, errors.Wrap(err, "failed to write command to unix socket")
	}

	response, err := io.ReadAll(conn)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read response from unix socket")
	}

	return response, nil
}

// Implements keaConnector interface for connecting to Kea via HTTP.
type keaHTTPConnector struct {
	url        string
	httpClient *httpClient
}

// Sends the command to Kea via HTTP and returns the response.
func (c *keaHTTPConnector) sendPayload(ctx context.Context, command []byte) ([]byte, error) {
	response, err := c.httpClient.Call(ctx, c.url, bytes.NewBuffer(command))
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to send command to Kea: %s", c.url)
	}

	// Kea returned a non-success status code.
	if response.StatusCode != http.StatusOK {
		return nil, errors.Errorf("received non-success status code %d from Kea, with status text: %s; url: %s", response.StatusCode, response.Status, c.url)
	}

	// Read the response.
	body, err := io.ReadAll(response.Body)
	response.Body.Close()
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to read Kea response body received from %s", c.url)
	}

	// The responses from the Kea send over HTTP are wrapped in a JSON array.
	// Responses from the socket channel are always single JSON objects.
	body, err = keactrl.UnwrapKeaResponseArray(body)
	if err != nil {
		return nil, err
	}

	return body, nil
}

// Handles connections to multiple control sockets of a Kea daemon. It tries
// each connector in order until one succeeds.
type multiConnector struct {
	connectors []keaConnector
}

// Creates a new multiConnector from the list of access points.
// It instantiates a keaConnector for each access point.
// It is expected that all access points belong to the same Kea daemon.
// The httpClientConfigs should match the accessPoints by index.
func newMultiConnector(accessPoints []AccessPoint, httpClientConfigs []HTTPClientConfig) *multiConnector {
	var connectors []keaConnector
	for i, ap := range accessPoints {
		httpClientConfig := HTTPClientConfig{}
		if i < len(httpClientConfigs) {
			httpClientConfig = httpClientConfigs[i]
		}
		connector := newKeaConnector(ap, httpClientConfig)
		connectors = append(connectors, connector)
	}
	return &multiConnector{connectors: connectors}
}

// Sends the command to Kea via one of the connectors and returns the response.
func (c *multiConnector) sendPayload(ctx context.Context, command []byte) ([]byte, error) {
	if len(c.connectors) == 0 {
		return nil, errors.New("no connectors available to send command to Kea")
	}

	var lastErr error
	for i, connector := range c.connectors {
		response, err := connector.sendPayload(ctx, command)
		if err == nil {
			return response, nil
		}
		log.WithError(err).WithField("index", i).Debug("Connection to Kea failed, trying another control socket")
		lastErr = err
	}
	return nil, errors.WithMessagef(lastErr, "all connectors failed to send command to Kea")
}
