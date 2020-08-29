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
	err = keactrl.UnmarshalResponseList(command, string(body), responses)
	if err != nil {
		return errors.WithMessagef(err, "failed to parse Kea response body received from %s", caURL)
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
