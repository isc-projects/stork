package agent

import (
	"io/ioutil"
	"regexp"
	"strconv"

	log "github.com/sirupsen/logrus"
)

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

func detectKeaApp(match []string) *App {
	keaConfPath := match[1]

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
