package agent

import (
	"fmt"
	"time"
	"bytes"
	"net/http"
	"regexp"
	"strconv"
	"io/ioutil"
	"encoding/json"

	log "github.com/sirupsen/logrus"
	"github.com/pkg/errors"
	"github.com/shirou/gopsutil/process"
)

type KeaDaemon struct {
	Pid int32
	Name string
	Active bool
	Version string
	ExtendedVersion string
}

type ServiceCommon struct {
	Version string
	CtrlPort int64
	Active bool
}

type ServiceKea struct {
	ServiceCommon
	ExtendedVersion string
	Daemons []KeaDaemon
}

type ServiceBind struct {
	ServiceCommon
}

type ServiceMonitor interface {
	GetServices() []interface{}
	Shutdown()
}

type serviceMonitor struct {
	requests chan chan []interface{}     // input to service monitor, ie. channel for receiving requests
	quit chan bool       // channel for stopping service monitor

	services []interface{} // list of detected services on the host
}

func NewServiceMonitor() *serviceMonitor {
	sm := &serviceMonitor{
		requests: make(chan chan []interface{}),
		quit: make(chan bool),
	}
	go sm.run()
	return sm
}

func (sm *serviceMonitor) run() {
	const DETECTION_INTERVAL = 10 * time.Second

	for {
		select {
		case ret := <- sm.requests:
			// process user request
			ret <- sm.services

		case <- time.After(DETECTION_INTERVAL):
			// periodic detection
			sm.detectServices()

		case <- sm.quit:
			// exit run
			return
		}
	}
}

func getCtrlPortFromKeaConfig(path string) int {
	text, err := ioutil.ReadFile(path)
	if err != nil {
		log.Warnf("cannot read kea config file: %+v", err)
		return 0
	}

	ptrn := regexp.MustCompile(`"http-port"\s*:\s*([0-9]+)`)
	m := ptrn.FindStringSubmatch(string(text))
	if len(m) == 0 {
		log.Warnf("cannot parse port: %+v", err)
		return 0
	}

	port, err := strconv.Atoi(m[1])
	if err != nil {
		log.Warnf("cannot parse port: %+v", err)
		return 0
	}
	return port
}


func keaDaemonVersionGet(caUrl string, daemon string) (map[string]interface{}, error) {
	var jsonCmd = []byte(`{"command": "version-get"}`)
	if daemon != "" {
		jsonCmd = []byte(`{"command": "version-get", "service": ["` + daemon + `"]}`)
	}

	resp, err := http.Post(caUrl, "application/json", bytes.NewBuffer(jsonCmd))
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

func detectKeaService(match []string) *ServiceKea {
	var keaService *ServiceKea

	keaConfPath := match[1]

	ctrlPort := int64(getCtrlPortFromKeaConfig(keaConfPath))
	keaService = &ServiceKea{
		ServiceCommon: ServiceCommon{
			CtrlPort: ctrlPort,
			Active: false,
		},
		Daemons: []KeaDaemon{},
	}
	if ctrlPort == 0 {
		return nil
	}

	caUrl := fmt.Sprintf("http://localhost:%d", ctrlPort)

	// retrieve ctrl-agent information, it is also used as a general service information
	info, err := keaDaemonVersionGet(caUrl, "")
	if err == nil {
		if int(info["result"].(float64)) == 0 {
			keaService.Active = true
			keaService.Version = info["text"].(string)
			info2 := info["arguments"].(map[string]interface{})
			keaService.ExtendedVersion = info2["extended"].(string)
		} else {
			log.Warnf("ctrl-agent returned negative response: %+v", info)
		}
	} else {
		log.Warnf("cannot get daemon version: %+v", err)
	}

	// add info about ctrl-agent daemon
	caDaemon := KeaDaemon{
		Name: "ca",
		Active: keaService.Active,
		Version: keaService.Version,
		ExtendedVersion: keaService.ExtendedVersion,
	}
	keaService.Daemons = append(keaService.Daemons, caDaemon)

	// get list of daemons configured in ctrl-agent
	var jsonCmd = []byte(`{"command": "config-get"}`)
	resp, err := http.Post(caUrl, "application/json", bytes.NewBuffer(jsonCmd))
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
		// if any daemon is inactive, then whole kea service is treated as inactive
		if !daemon.Active {
			keaService.Active = false
		}

		// if any daemon is inactive, then whole kea service is treated as inactive
		if !daemon.Active {
			keaService.Active = false
		}

		keaService.Daemons = append(keaService.Daemons, daemon)
	}

	return keaService
}

func (sm *serviceMonitor) detectServices() {
	// Kea service is being detected by browsing list of processes in the systam
	// where cmdline of the process contains given pattern with kea-ctr-agent
	// substring. Such found processes are being processed further and all other
	// Kea daemons are discovered and queried for their versions, etc.
	keaPtrn := regexp.MustCompile(`kea-ctrl-agent.*-c\s+(\S+)`)

	// TODO: BIND service is not yet being detect. It should happen here as well.

	var services []interface{}

	procs, _ := process.Processes()
	for _, p := range procs {
		cmdline, err := p.Cmdline()
		if err != nil {
			log.Warnf("cannot get process command line %+v", err)
		}

		// detect kea
		m := keaPtrn.FindStringSubmatch(cmdline)
		if m != nil {
			keaService := detectKeaService(m)
			if keaService != nil {
				services = append(services, *keaService)
			}
		}
	}

	sm.services = services
}

func (sm *serviceMonitor) GetServices() []interface{} {
	ret := make(chan []interface{})
	sm.requests <- ret
	srvs := <- ret
	return srvs
}

func (sm *serviceMonitor) Shutdown() {
	sm.quit <- true
}
