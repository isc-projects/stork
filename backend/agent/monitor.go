package agent

import (
	"time"
	"bytes"
	"regexp"
	"strconv"
	"io/ioutil"
	"encoding/json"
	"os/exec"

	log "github.com/sirupsen/logrus"
	"github.com/pkg/errors"
	"github.com/shirou/gopsutil/process"
	"isc.org/stork/util"
)

type KeaDaemon struct {
	Pid int32
	Name string
	Active bool
	Version string
	ExtendedVersion string
}

type Bind9Daemon struct {
	Pid int32
	Name string
	Active bool
	Version string
}

type AppCommon struct {
	Version     string
	CtrlAddress string
	CtrlPort    int64
	Active      bool
}

type AppKea struct {
	AppCommon
	ExtendedVersion string
	Daemons []KeaDaemon
}

type AppBind9 struct {
	AppCommon
	Daemon Bind9Daemon
}

type AppMonitor interface {
	GetApps() []interface{}
	Shutdown()
}

type appMonitor struct {
	requests chan chan []interface{}     // input to app monitor, ie. channel for receiving requests
	quit chan bool       // channel for stopping app monitor

	apps []interface{} // list of detected apps on the host
}

func NewAppMonitor() *appMonitor {
	sm := &appMonitor{
		requests: make(chan chan []interface{}),
		quit: make(chan bool),
	}
	go sm.run()
	return sm
}

func (sm *appMonitor) run() {
	const DETECTION_INTERVAL = 10 * time.Second

	for {
		select {
		case ret := <- sm.requests:
			// process user request
			ret <- sm.apps

		case <- time.After(DETECTION_INTERVAL):
			// periodic detection
			sm.detectApps()

		case <- sm.quit:
			// exit run
			return
		}
	}
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
		}
	}

	return address, int64(port)
}


func keaDaemonVersionGet(caUrl string, daemon string) (map[string]interface{}, error) {
	var jsonCmd = []byte(`{"command": "version-get"}`)
	if daemon != "" {
		jsonCmd = []byte(`{"command": "version-get", "service": ["` + daemon + `"]}`)
	}

	resp, err := httpClient11.Post(caUrl, "application/json", bytes.NewBuffer(jsonCmd))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)

	var data interface{}
	err = json.Unmarshal(body, &data)
	if err != nil {
		return nil, err
	}

	data1, ok := data.([]interface{})
	if !ok || len(data1) == 0 {
		return nil, errors.New("bad data")
	}
	data2, ok := data1[0].(map[string]interface{})
	if !ok {
		return nil, errors.New("bad data")
	}
	return data2, nil
}

func detectBind9App() (bind9App *AppBind9) {
	bind9App = &AppBind9{
		AppCommon: AppCommon{
			Active: false,
		},
	}

	// version
	cmd := exec.Command("rndc", "-k", "/etc/bind/rndc.key", "status")
	out, err := cmd.Output()
	if err != nil {
		log.Warnf("cannot get BIND 9 status: %+v", err)
	} else {
		versionPtrn := regexp.MustCompile(`version:\s(.+)\n`)
		match := versionPtrn.FindStringSubmatch(string(out[:]))
		if match != nil {
			bind9App.Version = match[1]
		} else {
			log.Warnf("cannot get BIND 9 version: unable to find version in rndc output")
		}

		bind9App.Active = true
	}

	// TODO: control port, pid

	namedDaemon := Bind9Daemon{
		Name: "named",
		Active: bind9App.Active,
		Version: bind9App.Version,
	}

	bind9App.Daemon = namedDaemon

	return bind9App
}

func detectKeaApp(match []string) *AppKea {
	var keaApp *AppKea

	keaConfPath := match[1]

	ctrlAddress, ctrlPort := getCtrlAddressFromKeaConfig(keaConfPath)
	keaApp = &AppKea{
		AppCommon: AppCommon{
			Active: false,
			CtrlAddress: ctrlAddress,
			CtrlPort: ctrlPort,
		},
		Daemons: []KeaDaemon{},
	}
	if ctrlPort == 0 || len(ctrlAddress) == 0 {
		return nil
	}

	caUrl := storkutil.HostWithPortUrl(ctrlAddress, ctrlPort)

	// retrieve ctrl-agent information, it is also used as a general app information
	info, err := keaDaemonVersionGet(caUrl, "")
	if err == nil {
		if int(info["result"].(float64)) == 0 {
			keaApp.Active = true
			keaApp.Version = info["text"].(string)
			info2 := info["arguments"].(map[string]interface{})
			keaApp.ExtendedVersion = info2["extended"].(string)
		} else {
			log.Warnf("ctrl-agent returned negative response: %+v", info)
		}
	} else {
		log.Warnf("cannot get daemon version: %+v", err)
	}

	// add info about ctrl-agent daemon
	caDaemon := KeaDaemon{
		Name: "ca",
		Active: keaApp.Active,
		Version: keaApp.Version,
		ExtendedVersion: keaApp.ExtendedVersion,
	}
	keaApp.Daemons = append(keaApp.Daemons, caDaemon)

	// get list of daemons configured in ctrl-agent
	var jsonCmd = []byte(`{"command": "config-get"}`)
	resp, err := httpClient11.Post(caUrl, "application/json", bytes.NewBuffer(jsonCmd))
	if err != nil {
		log.Warnf("problem with request to kea-ctrl-agent: %+v", err)
		return nil
	}
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)
	var data interface{}
	err = json.Unmarshal(body, &data)
	if err != nil {
		log.Warnf("cannot parse response from kea-ctrl-agent: %+v", err)
		return nil
	}

	// unpack the data in the JSON structure until we reach the daemons list.
	m, ok := data.([]interface{})
	if !ok || len(m) == 0 {
		return nil
	}
	m2, ok := m[0].(map[string]interface{})
	if !ok {
		return nil
	}
	m3, ok := m2["arguments"].(map[string]interface{})
	if !ok {
		return nil
	}
	m4, ok := m3["Control-agent"].(map[string]interface{})
	if !ok {
		return nil
	}
	daemonsListInCA, ok := m4["control-sockets"].(map[string]interface{})
	if !ok {
		return nil
	}
	for daemonName := range daemonsListInCA {
		daemon := KeaDaemon{
			Name: daemonName,
			Active: false,
		}

		// retrieve info about daemon
		info, err := keaDaemonVersionGet(caUrl, daemonName)
		if err == nil {
			if int(info["result"].(float64)) == 0 {
				daemon.Active = true
				daemon.Version = info["text"].(string)
				info2 := info["arguments"].(map[string]interface{})
				daemon.ExtendedVersion = info2["extended"].(string)
			} else {
				log.Warnf("ctrl-agent returned negative response: %+v", info)
			}
		} else {
			log.Warnf("cannot get daemon version: %+v", err)
		}
		// if any daemon is inactive, then whole kea app is treated as inactive
		if !daemon.Active {
			keaApp.Active = false
		}

		// if any daemon is inactive, then whole kea app is treated as inactive
		if !daemon.Active {
			keaApp.Active = false
		}

		keaApp.Daemons = append(keaApp.Daemons, daemon)
	}

	return keaApp
}

func (sm *appMonitor) detectApps() {
	// Kea app is being detected by browsing list of processes in the systam
	// where cmdline of the process contains given pattern with kea-ctrl-agent
	// substring. Such found processes are being processed further and all other
	// Kea daemons are discovered and queried for their versions, etc.
	keaPtrn := regexp.MustCompile(`kea-ctrl-agent.*-c\s+(\S+)`)
	// Bind9 app is being detecting by browsing list of processes in the system
	// where cmdline of the process contains given pattern with named substring.
	bind9Ptrn := regexp.MustCompile(`named.*-c\s+(\S+)`)

	var apps []interface{}

	procs, _ := process.Processes()
	for _, p := range procs {
		procName, _ := p.Name()
		if procName == "kea-ctrl-agent" {
			cmdline, err := p.Cmdline()
			if err != nil {
				log.Warnf("cannot get process command line %+v", err)
			}

			// detect kea
			m := keaPtrn.FindStringSubmatch(cmdline)
			if m != nil {
				keaApp := detectKeaApp(m)
				if keaApp != nil {
					apps = append(apps, *keaApp)
				}
			}
			continue
		}

		if procName == "named" {
			cmdline, err := p.Cmdline()
			if err != nil {
				log.Warnf("cannot get process command line %+v", err)
			}

			// detect bind9
			m := bind9Ptrn.FindStringSubmatch(cmdline)
			if m != nil {
				bind9App := detectBind9App()
				if bind9App != nil {
					apps = append(apps, *bind9App)
				}
			}
			continue
		}
	}

	sm.apps = apps
}

func (sm *appMonitor) GetApps() []interface{} {
	ret := make(chan []interface{})
	sm.requests <- ret
	srvs := <- ret
	return srvs
}

func (sm *appMonitor) Shutdown() {
	sm.quit <- true
}
