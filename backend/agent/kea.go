package agent

import (
	"bytes"
	"io/ioutil"
	"path"
	"regexp"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	keaconfig "isc.org/stork/appcfg/kea"
	keactrl "isc.org/stork/appctrl/kea"
	storkutil "isc.org/stork/util"
)

// Sends a command to Kea and returns a response.
func sendToKeaOverHTTP(storkAgent *StorkAgent, caAddress string, caPort int64, command *keactrl.Command, responses interface{}) error {
	caURL := storkutil.HostWithPortURL(caAddress, caPort)

	// Get the textual representation of the command.
	request := command.Marshal()

	// Send the command to the Kea server.
	response, err := storkAgent.HTTPClient.Call(caURL, bytes.NewBuffer([]byte(request)))
	if err != nil {
		return errors.WithMessagef(err, "failed to send command to Kea: %s", caURL)
	}

	// Read the response.
	body, err := ioutil.ReadAll(response.Body)
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

// Updates the list of log files which can be viewed by the Stork user from the
// UI. The response variable holds the pointer to the response to the config-get
// command returned by one of the Kea daemons. If this response contains loggers'
// configuration the log files are extracted from it and stored in the logTailer
// instance of the agent. This function is intended to be called by the functions
// which intercept config-get commands sent periodically by the server to the
// agents and by the detectKeaAllowedLogs when the agent is started.
func updateKeaAllowedLogs(agent *StorkAgent, response *keactrl.Response) {
	if response.Result > 0 {
		log.Warn("skipped refreshing viewable log files because config-get returned non success result")
		return
	}
	if response.Arguments == nil {
		log.Warn("skipped refreshing viewable log files because config-get response has no arguments")
		return
	}
	cfg := keaconfig.New(response.Arguments)
	if cfg == nil {
		log.Warn("skipped refreshing viewable log files because config-get response contains arguments which could not be parsed")
		return
	}

	loggers := cfg.GetLoggers()
	if len(loggers) == 0 {
		log.Info("no loggers found in the returned configuration while trying to refresh the viewable log files")
		return
	}

	// Go over returned loggers and allow those found in the returned configuration.
	for _, l := range loggers {
		for _, o := range l.OutputOptions {
			if o.Output != "stdout" && o.Output != "stderr" && !strings.HasPrefix(o.Output, "syslog") {
				agent.logTailer.allow(o.Output)
			}
		}
	}
}

// Sends config-get command to all running Kea daemons belonging to the given Kea app
// to fetch logging configuration. The first config-get command is sent to the Kea CA,
// to fetch its logging configuration and to find the daemons running behind it. Next, the
// config-get command is sent to the daemons behind CA and their logging configuration
// is fetched. The log files locations are stored in the logTailer instance of the
// agent as allowed for viewing. This function should be called when the agent has
// been started and the running Kea apps have been detected.
func detectKeaAllowedLogs(storkAgent *StorkAgent, caAddress string, caPort int64) error {
	// Prepare config-get command to be sent to Kea Control Agent.
	command, err := keactrl.NewCommand("config-get", nil, nil)
	if err != nil {
		return err
	}

	// Send the command to Kea.
	responses := keactrl.ResponseList{}
	err = sendToKeaOverHTTP(storkAgent, caAddress, caPort, command, &responses)
	if err != nil {
		return err
	}

	// There should be exactly one response received because we sent the command
	// to only one daemon.
	if len(responses) != 1 {
		return errors.Errorf("invalid response received from Kea CA to config-get command sent to %s:%d", caAddress, caPort)
	}

	// It does not make sense to proceed if the CA returned non-success status
	// because this response neither contains logging configuration nor
	// sockets configurations.
	if responses[0].Result != 0 {
		return errors.Errorf("non success response %d received from Kea CA to config-get command sent to %s:%d", responses[0].Result, caAddress, caPort)
	}

	// Allow the log files used by the CA.
	updateKeaAllowedLogs(storkAgent, &responses[0])

	// Arguments should be returned in response to the config-get command.
	rawConfig := responses[0].Arguments
	if rawConfig == nil {
		return errors.Errorf("empty arguments received from Kea CA in response to config-get command sent to %s:%d", caAddress, caPort)
	}
	// The returned configuration has unexpected structure.
	config := keaconfig.New(rawConfig)
	if config == nil {
		return errors.Errorf("unable to parse the config received from Kea CA in response to config-get command sent to %s:%d", caAddress, caPort)
	}

	// Control Agent should be configured to forward commands to some
	// daemons behind it.
	sockets := config.GetControlSockets()
	daemonNames := sockets.ConfiguredDaemonNames()

	// Apparently, it isn't configured to forward commands to the daemons behind it.
	if len(daemonNames) == 0 {
		return nil
	}

	// Prepare config-get command to be sent to the daemons behind CA.
	daemons, err := keactrl.NewDaemons(daemonNames...)
	if err != nil {
		return err
	}
	command, err = keactrl.NewCommand("config-get", daemons, nil)
	if err != nil {
		return err
	}

	// Send config-get to the daemons behind CA.
	responses = keactrl.ResponseList{}
	err = sendToKeaOverHTTP(storkAgent, caAddress, caPort, command, &responses)
	if err != nil {
		return err
	}

	// Check that we got responses for all daemons.
	if len(responses) != len(daemonNames) {
		return errors.Errorf("invalid number of responses received from daemons to config-get command sent via %s:%d", caAddress, caPort)
	}

	// For each daemon try to extract its logging configuration and allow view
	// the log files it contains.
	for i := range responses {
		updateKeaAllowedLogs(storkAgent, &responses[i])
	}

	return nil
}

func getCtrlAddressFromKeaConfig(path string) (string, int64) {
	text, err := ioutil.ReadFile(path)
	if err != nil {
		log.Warnf("cannot read kea config file: %+v", err)
		return "", 0
	}

	ptrn := regexp.MustCompile(`"http-port"\s*:\s*([0-9]+)`)
	m := ptrn.FindStringSubmatch(string(text))
	if len(m) == 0 {
		log.Warnf("cannot parse http-port: %+v", err)
		return "", 0
	}

	port, err := strconv.Atoi(m[1])
	if err != nil {
		log.Warnf("cannot parse http-port: %+v", err)
		return "", 0
	}

	ptrn = regexp.MustCompile(`"http-host"\s*:\s*\"(\S+)\"\s*,`)
	m = ptrn.FindStringSubmatch(string(text))
	address := "localhost"
	if len(m) == 0 {
		log.Warnf("cannot parse http-host: %+v", err)
	} else {
		address = m[1]
		if address == "0.0.0.0" {
			address = "127.0.0.1"
		} else if address == "::" {
			address = "::1"
		}
	}

	return address, int64(port)
}

func detectKeaApp(match []string, cwd string) *App {
	if len(match) < 3 {
		log.Warnf("problem with parsing Kea cmdline: %s", match[0])
		return nil
	}
	keaConfPath := match[2]

	// if path to config is not absolute then join it with CWD of kea
	if !strings.HasPrefix(keaConfPath, "/") {
		keaConfPath = path.Join(cwd, keaConfPath)
	}

	address, port := getCtrlAddressFromKeaConfig(keaConfPath)
	if port == 0 || len(address) == 0 {
		return nil
	}
	accessPoints := []AccessPoint{
		{
			Type:    AccessPointControl,
			Address: address,
			Port:    port,
		},
	}
	keaApp := &App{
		Type:         AppTypeKea,
		AccessPoints: accessPoints,
	}

	return keaApp
}
