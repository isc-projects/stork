package agent

import (
	"regexp"
	"time"

	"github.com/shirou/gopsutil/process"
	log "github.com/sirupsen/logrus"
)

type App struct {
	Type        string // currently supported types are: "kea" and "bind9"
	CtrlAddress string
	CtrlPort    int64
	CtrlKey     string
}

type AppMonitor interface {
	GetApps() []*App
	Shutdown()
}

type appMonitor struct {
	requests chan chan []*App // input to app monitor, ie. channel for receiving requests
	quit     chan bool        // channel for stopping app monitor

	apps []*App // list of detected apps on the host
}

func NewAppMonitor() AppMonitor {
	sm := &appMonitor{
		requests: make(chan chan []*App),
		quit:     make(chan bool),
	}
	go sm.run()
	return sm
}

func (sm *appMonitor) run() {
	log.Printf("Started app monitor")
	const initialDetectionInterval = 10 * time.Second
	detectionInterval := initialDetectionInterval

	for {
		t0 := time.Now()
		select {
		case ret := <-sm.requests:
			// process user request
			ret <- sm.apps

			// If sleeping in time.After below has been interrupted by user request served here,
			// then reduce next sleep time by the amout of time that we already slept for.
			detectionInterval = detectionInterval - time.Now().Sub(t0)
			if detectionInterval < 0 {
				detectionInterval = 0
			}

		case <-time.After(detectionInterval):
			// periodic detection
			t1 := time.Now()
			elapsed := t1.Sub(t0)
			sm.detectApps()

			// restore sleep time to initial one
			detectionInterval = initialDetectionInterval

		case <-sm.quit:
			// exit run
			log.Printf("Stopped app monitor")
			return
		}
	}
}

func (sm *appMonitor) detectApps() {
	// Kea app is being detected by browsing list of processes in the systam
	// where cmdline of the process contains given pattern with kea-ctrl-agent
	// substring. Such found processes are being processed further and all other
	// Kea daemons are discovered and queried for their versions, etc.
	keaPtrn := regexp.MustCompile(`kea-ctrl-agent.*-c\s+(\S+)`)
	// BIND 9 app is being detecting by browsing list of processes in the system
	// where cmdline of the process contains given pattern with named substring.
	bind9Ptrn := regexp.MustCompile(`named.*-c\s+(\S+)`)

	var apps []*App

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
					apps = append(apps, keaApp)
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
				bind9App := detectBind9App(m)
				if bind9App != nil {
					apps = append(apps, bind9App)
				}
			}
			continue
		}
	}

	sm.apps = apps
}

func (sm *appMonitor) GetApps() []*App {
	ret := make(chan []*App)
	sm.requests <- ret
	srvs := <-ret
	return srvs
}

func (sm *appMonitor) Shutdown() {
	sm.quit <- true
}
