package agent

import (
	"bytes"
	"io"
	"path"
	"strings"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	keaconfig "isc.org/stork/appcfg/kea"
	keactrl "isc.org/stork/appctrl/kea"
	storkutil "isc.org/stork/util"
)

// It holds common and Kea specific runtime information.
type KeaApp struct {
	BaseApp
	HTTPClient        *HTTPClient // to communicate with Kea Control Agent
	ConfiguredDaemons []string
}

// Get base information about Kea app.
func (ka *KeaApp) GetBaseApp() *BaseApp {
	return &ka.BaseApp
}

// Sends a command to Kea and returns a response.
func (ka *KeaApp) sendCommand(command *keactrl.Command, responses interface{}) error {
	ap := &ka.BaseApp.AccessPoints[0]
	caURL := storkutil.HostWithPortURL(ap.Address, ap.Port, ap.UseSecureProtocol)

	// Get the textual representation of the command.
	request := command.Marshal()

	// Send the command to the Kea server.
	response, err := ka.HTTPClient.Call(caURL, bytes.NewBuffer([]byte(request)))
	if err != nil {
		return errors.WithMessagef(err, "failed to send command to Kea: %s", caURL)
	}

	// Read the response.
	body, err := io.ReadAll(response.Body)
	response.Body.Close()
	if err != nil {
		return errors.WithMessagef(err, "failed to read Kea response body received from %s", caURL)
	}

	// Parse the response.
	err = keactrl.UnmarshalResponseList(command, body, responses)
	if err != nil {
		return errors.WithMessagef(err, "failed to parse Kea response body received from %s", caURL)
	}
	return nil
}

// Collect the list of log files which can be viewed by the Stork user
// from the UI. The response variable holds the pointer to the
// response to the config-get command returned by one of the Kea
// daemons. If this response contains loggers' configuration the log
// files are extracted from it and returned. This function is intended
// to be called by the functions which intercept config-get commands
// sent periodically by the server to the agents and by the
// DetectAllowedLogs when the agent is started.
func collectKeaAllowedLogs(response *keactrl.Response) []string {
	if response.Result > 0 {
		log.Warn("Skipped refreshing viewable log files because config-get returned unsuccessful result")
		return nil
	}
	if response.Arguments == nil {
		log.Warn("Skipped refreshing viewable log files because config-get response has no arguments")
		return nil
	}
	cfg := keaconfig.NewConfigFromMap(response.Arguments)
	if cfg == nil {
		log.Warn("Skipped refreshing viewable log files because config-get response contains arguments which could not be parsed")
		return nil
	}

	loggers := cfg.GetLoggers()
	if len(loggers) == 0 {
		log.Info("No loggers found in the returned configuration while trying to refresh the viewable log files")
		return nil
	}

	// Go over returned loggers and collect those found in the returned configuration.
	var paths []string
	for _, l := range loggers {
		for _, o := range l.GetAllOutputOptions() {
			if o.Output != "stdout" && o.Output != "stderr" && !strings.HasPrefix(o.Output, "syslog") {
				paths = append(paths, o.Output)
			}
		}
	}
	return paths
}

// Sends config-get command to all running Kea daemons belonging to the given Kea app
// to fetch logging configuration. The first config-get command is sent to the Kea CA,
// to fetch its logging configuration and to find the daemons running behind it. Next, the
// config-get command is sent to the daemons behind CA and their logging configuration
// is fetched. The log files locations are stored in the logTailer instance of the
// agent as allowed for viewing. This function should be called when the agent has
// been started and the running Kea apps have been detected.
func (ka *KeaApp) DetectAllowedLogs() ([]string, error) {
	// Prepare config-get command to be sent to Kea Control Agent.
	command := keactrl.NewCommand("config-get", nil, nil)
	// Send the command to Kea.
	responses := keactrl.ResponseList{}
	err := ka.sendCommand(command, &responses)
	if err != nil {
		return nil, err
	}

	ap := ka.BaseApp.AccessPoints[0]

	// There should be exactly one response received because we sent the command
	// to only one daemon.
	if len(responses) != 1 {
		return nil, errors.Errorf("invalid response received from Kea CA to config-get command sent to %s:%d", ap.Address, ap.Port)
	}

	// It does not make sense to proceed if the CA returned non-success status
	// because this response neither contains logging configuration nor
	// sockets configurations.
	if responses[0].Result != 0 {
		return nil, errors.Errorf("unsuccessful response %d received from Kea CA to config-get command sent to %s:%d", responses[0].Result, ap.Address, ap.Port)
	}

	// Allow the log files used by the CA.
	paths := collectKeaAllowedLogs(&responses[0])

	// Arguments should be returned in response to the config-get command.
	rawConfig := responses[0].Arguments
	if rawConfig == nil {
		return nil, errors.Errorf("empty arguments received from Kea CA in response to config-get command sent to %s:%d", ap.Address, ap.Port)
	}
	// The returned configuration has unexpected structure.
	config := keaconfig.NewConfigFromMap(rawConfig)
	if config == nil {
		return nil, errors.Errorf("unable to parse the config received from Kea CA in response to config-get command sent to %s:%d", ap.Address, ap.Port)
	}

	// Control Agent should be configured to forward commands to some
	// daemons behind it.
	sockets := config.GetControlSockets()
	daemonNames := sockets.GetConfiguredDaemonNames()

	// Apparently, it isn't configured to forward commands to the daemons behind it.
	if len(daemonNames) == 0 {
		return nil, nil
	}

	// Prepare config-get command to be sent to the daemons behind CA.
	command = keactrl.NewCommand("config-get", daemonNames, nil)

	// Send config-get to the daemons behind CA.
	responses = keactrl.ResponseList{}
	err = ka.sendCommand(command, &responses)
	if err != nil {
		return nil, err
	}

	// Check that we got responses for all daemons.
	if len(responses) != len(daemonNames) {
		return nil, errors.Errorf("invalid number of responses received from daemons to config-get command sent via %s:%d", ap.Address, ap.Port)
	}

	// For each daemon try to extract its logging configuration and allow view
	// the log files it contains.
	for i := range responses {
		paths = append(paths, collectKeaAllowedLogs(&responses[i])...)
	}

	return paths, nil
}

// Returns a list of the configured daemons in a given application.
func (ka *KeaApp) GetConfiguredDaemons() []string {
	return ka.ConfiguredDaemons
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
		err = errors.WithMessage(err, "Cannot parse Kea Control Agent config file")
		return nil, err
	}

	return config, err
}

func detectKeaApp(match []string, cwd string, httpClient *HTTPClient) App {
	if len(match) < 3 {
		log.Warnf("Problem parsing Kea cmdline: %s", match[0])
		return nil
	}
	keaConfPath := match[2]

	// if path to config is not absolute then join it with CWD of kea
	if !strings.HasPrefix(keaConfPath, "/") {
		keaConfPath = path.Join(cwd, keaConfPath)
	}

	config, err := readKeaConfig(keaConfPath)
	if err != nil {
		log.WithError(err).Error("Invalid Kea Control Agent config")
		return nil
	}

	// Port
	port, ok := config.GetHTTPPort()
	if !ok || port == 0 {
		log.Warn("Cannot parse the port")
		return nil
	}

	// Address
	address, _ := config.GetHTTPHost()

	accessPoints := []AccessPoint{
		{
			Type:              AccessPointControl,
			Address:           address,
			Port:              port,
			UseSecureProtocol: config.UseSecureProtocol(),
		},
	}
	keaApp := &KeaApp{
		BaseApp: BaseApp{
			Type:         AppTypeKea,
			AccessPoints: accessPoints,
		},
		HTTPClient:        httpClient,
		ConfiguredDaemons: config.GetControlSockets().GetConfiguredDaemonNames(),
	}

	return keaApp
}
