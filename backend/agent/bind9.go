package agent

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"regexp"
	"strconv"

	log "github.com/sirupsen/logrus"
)

type Bind9Daemon struct {
	Pid     int32
	Name    string
	Active  bool
	Version string
}

type Bind9State struct {
	Version string
	Active  bool
	Daemon  Bind9Daemon
}

// Helpful constants for rndc (the remote control command for the named daemon).
const RndcDefaultPort = 953
const RndcKeyFile = "/etc/bind/rndc.key"

// getRndcKey looks for the key with a given `name` in `contents`.
//
// Example key clause:
//
//    key "name" {
//        algorithm "hmac-md5";
//        secret "OmItW1lOyLVUEuvv+Fme+Q==";
//    };
//
func getRndcKey(contents, name string) (controlKey string) {
	ptrn := regexp.MustCompile(`(?s)keys\s+\"(\S+)\"\s+\{(.*)\}\s*;`)
	keys := ptrn.FindAllStringSubmatch(contents, -1)
	if len(keys) == 0 {
		return ""
	}

	for _, key := range keys {
		if key[1] != name {
			continue
		}
		ptrn = regexp.MustCompile(`(?s)algorithm\s+\"(\S+)\";`)
		algorithm := ptrn.FindStringSubmatch(key[2])
		if len(algorithm) < 2 {
			log.Warnf("no key algorithm found for name %s", name)
			return ""
		}

		ptrn = regexp.MustCompile(`(?s)secret\s+\"(\S+)\";`)
		secret := ptrn.FindStringSubmatch(key[2])
		if len(secret) < 2 {
			log.Warnf("no key secret found for name %s", name)
			return ""
		}

		// this key clause matches the name we are looking for
		controlKey = fmt.Sprintf("%s:%s", algorithm[1], secret[1])
		break
	}

	return controlKey
}

// getCtrlAddressFromBind9Config retrieves the rndc control access address,
// port, and secret key (if configured) from the configuration `path`.
//
// The controls clause can also be in an include file, but currently this
// function is not following include paths.
//
// Multiple controls clauses may be configured but currently this function
// only matches the first one.  Multiple access points may be listed inside
// a single controls clause, but this function currently only matches the
// first in the list.  A controls clause may look like this:
//
//    controls {
//        inet 127.0.0.1 allow {localhost;};
//        inet * port 7766 allow {"rndc-users";} keys {"rndc-remote";};
//    };
//
// In this example, "rndc-users" and "rndc-remote" refer to an acl and key
// clauses.
//
// Finding the key is done by looking if the control access point has a
// keys parameter and if so, it looks in `path` for a key clause with the
// same name.  A key clause may look like this:
//
//    key "rndc-remote" {
//        algorithm hmac-md5;
//        secret "OmItW1lOyLVUEuvv+Fme+Q==";
//    };
func getCtrlAddressFromBind9Config(path string) (controlAddress string, controlPort int64, controlKey string) {
	text, err := ioutil.ReadFile(path)
	if err != nil {
		log.Warnf("cannot read BIND 9 config file (%s): %+v", path, err)
		return "", 0, ""
	}

	// Match the following clause:
	//     controls {
	//         inet inet_spec [inet_spec] ;
	//     };
	ptrn := regexp.MustCompile(`(?s)controls\s*\{\s*(.*)\s*\}\s*;`)
	controls := ptrn.FindStringSubmatch(string(text))
	if len(controls) == 0 {
		log.Warnf("cannot parse BIND 9 controls clause: %+v, %+v", string(text), err)
		return "", 0, ""
	}

	// We only pick the first match, but the controls clause
	// can list multiple control access points.
	// inet_spec = ( ip_addr | * ) [ port ip_port ]
	//             allow { address_match_list }
	//             keys { key_list };
	ptrn = regexp.MustCompile(`(?s)inet\s+(\S+\s*\S*\s*\d*)\s+allow\s*\{\s*\S+\s*;\s*\}(.*);`)
	match := ptrn.FindStringSubmatch(controls[1])
	if len(match) == 0 {
		log.Warnf("cannot parse BIND 9 inet configuration: %+v, %+v", controls[1], err)
		return "", 0, ""
	}

	inetSpec := regexp.MustCompile(`\s+`).Split(match[1], 3)
	port := RndcDefaultPort
	var address string
	switch len(inetSpec) {
	case 1:
		address = inetSpec[0]
	case 3:
		address = inetSpec[0]
		if inetSpec[1] != "port" {
			log.Warnf("cannot parse BIND 9 control port: %+v, %+v", inetSpec, err)
			return "", 0, ""
		}

		var err error
		port, err = strconv.Atoi(inetSpec[2])
		if err != nil {
			log.Warnf("cannot parse BIND 9 control port: %+v, %+v", inetSpec, err)
			return "", 0, ""
		}
	case 2:
	default:
		log.Warnf("cannot parse BIND 9 inet_spec configuration: %+v, %+v", inetSpec, err)
		return "", 0, ""
	}

	if len(match) == 3 {
		// Find a key clause
		ptrn = regexp.MustCompile(`(?s)keys\s*\{\s*\"(\S+)\"\s*;\s*\}\s*`)
		keyName := ptrn.FindStringSubmatch(match[2])
		if len(keyName) > 1 {
			controlKey = getRndcKey(string(text), keyName[1])
		}
	}

	if address == "*" {
		controlAddress = "localhost"
	} else {
		controlAddress = address
	}
	controlPort = int64(port)

	return controlAddress, controlPort, controlKey
}

func detectBind9App(match []string) (bind9App *App) {
	bind9ConfPath := match[1]

	ctrlAddress, ctrlPort, ctrlKey := getCtrlAddressFromBind9Config(bind9ConfPath)
	if ctrlPort == 0 || len(ctrlAddress) == 0 {
		return nil
	}

	bind9App = &App{
		Type:        "bind9",
		CtrlAddress: ctrlAddress,
		CtrlPort:    ctrlPort,
		CtrlKey:     ctrlKey,
	}
	return bind9App
}

// rndc is an utility function to send a command to the named daemon.
func (app *App) rndc(command []string) ([]byte, error) {
	if app.CtrlPort == 0 {
		return nil, fmt.Errorf("rndc requires control port")
	}
	if len(app.CtrlAddress) == 0 {
		return nil, fmt.Errorf("rndc requires control address")
	}

	rndcCommand := []string{"rndc", "-s", app.CtrlAddress, "-p", fmt.Sprintf("%d", app.CtrlPort)}
	if len(app.CtrlKey) > 0 {
		rndcCommand = append(rndcCommand, "-y")
		rndcCommand = append(rndcCommand, app.CtrlKey)
	} else if _, err := os.Stat(RndcKeyFile); err == nil {
		rndcCommand = append(rndcCommand, "-k")
		rndcCommand = append(rndcCommand, RndcKeyFile)
	}
	rndcCommand = append(rndcCommand, command...)
	log.Warnf("rndc: %+v", rndcCommand)

	cmd := exec.Command(rndcCommand[0], rndcCommand[1:]...) //nolint:gosec
	return cmd.Output()
}

func getBind9State(app *App) (*Bind9State, error) { //nolint:unparam
	state := &Bind9State{
		Active: false,
	}

	// version
	out, err := app.rndc([]string{"status"})
	if err != nil {
		log.Warnf("cannot get BIND 9 status: %+v", err)
	} else {
		versionPtrn := regexp.MustCompile(`version:\s(.+)\n`)
		match := versionPtrn.FindStringSubmatch(string(out))
		if match != nil {
			state.Version = match[1]
		} else {
			log.Warnf("cannot get BIND 9 version: unable to find version in rndc output")
		}

		upPtrn := regexp.MustCompile(`server is up and running`)
		up := upPtrn.FindString(string(out))
		if up != "" {
			state.Active = true
		}
	}

	namedDaemon := Bind9Daemon{
		Name:    "named",
		Active:  state.Active,
		Version: state.Version,
	}

	state.Daemon = namedDaemon

	return state, err
}
