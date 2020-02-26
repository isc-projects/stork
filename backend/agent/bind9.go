package agent

import (
	"fmt"
	"io/ioutil"
	"regexp"
	"strconv"

	log "github.com/sirupsen/logrus"
)

type Bind9Daemon struct {
	Pid     int32
	Name    string
	Version string
	Active  bool
}

type Bind9State struct {
	Version string
	Active  bool
	Daemon  Bind9Daemon
}

const RndcDefaultPort = 953
const StatsChannelDefaultPort = 80

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

// parseInetSpec parses an inet statement from a named configuration excerpt.
// The inet statement is defined by inet_spec:
//
//    inet_spec = ( ip_addr | * ) [ port ip_port ]
//                allow { address_match_list }
//                keys { key_list };
//
// This function returns the ip_addr, port and the first key that is
// referenced in the key_list.  If instead of an ip_addr, the asterisk (*) is
// specified, this function will return 'localhost' as an address.
func parseInetSpec(config, excerpt string) (address string, port int64, key string) {
	ptrn := regexp.MustCompile(`(?s)inet\s+(\S+\s*\S*\s*\d*)\s+allow\s*\{\s*\S+\s*;\s*\}(.*);`)
	match := ptrn.FindStringSubmatch(excerpt)
	if len(match) == 0 {
		log.Warnf("cannot parse BIND 9 inet configuration: no match (%+v)", config)
		return "", 0, ""
	}

	inetSpec := regexp.MustCompile(`\s+`).Split(match[1], 3)
	switch len(inetSpec) {
	case 1:
		address = inetSpec[0]
	case 3:
		address = inetSpec[0]
		if inetSpec[1] != "port" {
			log.Warnf("cannot parse BIND 9 control port: bad port statement (%+v)", inetSpec)
			return "", 0, ""
		}

		iPort, err := strconv.Atoi(inetSpec[2])
		if err != nil {
			log.Warnf("cannot parse BIND 9 control port: %+v (%+v)", inetSpec, err)
			return "", 0, ""
		}
		port = int64(iPort)
	case 2:
	default:
		log.Warnf("cannot parse BIND 9 inet_spec configuration: no match (%+v)", inetSpec)
		return "", 0, ""
	}

	if len(match) == 3 {
		// Find a key clause
		ptrn = regexp.MustCompile(`(?s)keys\s*\{\s*\"(\S+)\"\s*;\s*\}\s*`)
		keyName := ptrn.FindStringSubmatch(match[2])
		if len(keyName) > 1 {
			key = getRndcKey(config, keyName[1])
		}
	}

	if address == "*" {
		address = "localhost"
	}

	return address, port, key
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
// same name.
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
	controlAddress, controlPort, controlKey = parseInetSpec(string(text), controls[1])
	if controlAddress != "" {
		// If no port was provided, use the default rndc port.
		if controlPort == 0 {
			controlPort = RndcDefaultPort
		}
	}
	return controlAddress, controlPort, controlKey
}

// getStatisticsChannelFromBind9Config retrieves the statistics channel access
// address, port, and secret key (if configured) from the configuration `path`.
//
// The statistics-channels clause can also be in an include file, but
// currently this function is not following include paths.
//
// Multiple statistics-channels clauses may be configured but currently this
// function only matches the first one.  Multiple access points may be listed
// inside a single controls clause, but this function currently only matches
// the first in the list.  A statistics-channels clause may look like this:
//
//    statistics-channels {
//        inet 10.1.10.10 port 8080 allow { 192.168.2.10; 10.1.10.2; };
//        inet 127.0.0.1  port 8080 allow { "stats-clients" };
//    };
//
// In this example, "stats-clients" refers to an acl clause.
//
// Finding the key is done by looking if the control access point has a
// keys parameter and if so, it looks in `path` for a key clause with the
// same name.
func getStatisticsChannelFromBind9Config(path string) (statsAddress string, statsPort int64, statsKey string) {
	text, err := ioutil.ReadFile(path)
	if err != nil {
		log.Warnf("cannot read BIND 9 config file (%s): %+v", path, err)
		return "", 0, ""
	}

	// Match the following clause:
	//     statistics-channels {
	//         inet inet_spec [inet_spec] ;
	//     };
	ptrn := regexp.MustCompile(`(?s)statistics-channels\s*\{\s*(.*)\s*\}\s*;`)
	channels := ptrn.FindStringSubmatch(string(text))
	if len(channels) == 0 {
		log.Warnf("cannot parse BIND 9 statistics-channels clause: %+v, %+v", string(text), err)
		return "", 0, ""
	}

	// We only pick the first match, but the statistics-channels clause
	// can list multiple control access points.
	statsAddress, statsPort, statsKey = parseInetSpec(string(text), channels[1])
	if statsAddress != "" {
		// If no port was provided, use the default statschannel port.
		if statsPort == 0 {
			statsPort = StatsChannelDefaultPort
		}
	}
	return statsAddress, statsPort, statsKey
}

func detectBind9App(match []string) (bind9App *App) {
	bind9ConfPath := match[1]

	address, port, key := getCtrlAddressFromBind9Config(bind9ConfPath)
	if port == 0 || len(address) == 0 {
		return nil
	}
	accessPoints := []AccessPoint{
		{
			Type:    "control",
			Address: address,
			Port:    port,
			Key:     key,
		},
	}

	address, port, key = getStatisticsChannelFromBind9Config(bind9ConfPath)
	if port > 0 && len(address) != 0 {
		accessPoints = append(accessPoints, AccessPoint{
			Type:    "statistics",
			Address: address,
			Port:    port,
			Key:     key,
		})
	}

	return &App{
		Type:         "bind9",
		AccessPoints: accessPoints,
	}
}
