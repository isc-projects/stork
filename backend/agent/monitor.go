package agent

import (
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/shirou/gopsutil/process"
	log "github.com/sirupsen/logrus"

	storkutil "isc.org/stork/util"
)

// An access point for an application to retrieve information such
// as status or metrics.
type AccessPoint struct {
	Type    string
	Address string
	Port    int64
	Key     string
}

// Currently supported types are: "control" and "statistics".
const (
	AccessPointControl    = "control"
	AccessPointStatistics = "statistics"
)

type App struct {
	Pid          int32
	Type         string
	AccessPoints []AccessPoint
}

// Currently supported types are: "kea" and "bind9".
const (
	AppTypeKea   = "kea"
	AppTypeBind9 = "bind9"
)

type AppMonitor interface {
	GetApps() []*App
	Start(agent *StorkAgent)
	Shutdown()
}

type appMonitor struct {
	requests chan chan []*App // input to app monitor, ie. channel for receiving requests
	quit     chan bool        // channel for stopping app monitor
	running  bool
	wg       *sync.WaitGroup

	apps []*App // list of detected apps on the host
}

// Names of apps that are being detected.
const (
	keaProcName   = "kea-ctrl-agent"
	namedProcName = "named"
)

// Creates an AppMonitor instance. It used to start it as well, but this is now done
// by a dedicated method Start(). Make sure you call Start() before using app monitor.
func NewAppMonitor() AppMonitor {
	sm := &appMonitor{
		requests: make(chan chan []*App),
		quit:     make(chan bool),
		wg:       &sync.WaitGroup{},
	}
	return sm
}

// This function starts the actual monitor. This start is delayed in case we want to only
// do command line parameters parsing, e.g. to print version or help and quit.
func (sm *appMonitor) Start(storkAgent *StorkAgent) {
	sm.wg.Add(1)
	go sm.run(storkAgent)
}

func (sm *appMonitor) run(storkAgent *StorkAgent) {
	log.Printf("Started app monitor")

	sm.running = true
	defer sm.wg.Done()

	// run app detection one time immediately at startup
	sm.detectApps()

	// For each detected Kea app, let's gather the logs which can be viewed
	// from the UI.
	sm.detectAllowedLogs(storkAgent)

	// prepare ticker
	const detectionInterval = 10 * time.Second
	ticker := time.NewTicker(detectionInterval)
	defer ticker.Stop()

	for {
		select {
		case ret := <-sm.requests:
			// process user request
			ret <- sm.apps

		case <-ticker.C:
			// periodic detection
			sm.detectApps()

		case <-sm.quit:
			// exit run
			log.Printf("Stopped app monitor")
			sm.running = false
			return
		}
	}
}

func printNewOrUpdatedApps(newApps []*App, oldApps []*App) {
	// look for new or updated apps
	var newUpdatedApps []*App
	for _, appNew := range newApps {
		found := false
		for _, appOld := range oldApps {
			if appOld.Type != appNew.Type {
				continue
			}
			if len(appNew.AccessPoints) != len(appOld.AccessPoints) {
				continue
			}
			for idx, acPtNew := range appNew.AccessPoints {
				acPtOld := appOld.AccessPoints[idx]
				if acPtNew.Type != acPtOld.Type {
					continue
				}
				if acPtNew.Address != acPtOld.Address {
					continue
				}
				if acPtNew.Port != acPtOld.Port {
					continue
				}
			}
			found = true
		}
		if !found {
			newUpdatedApps = append(newUpdatedApps, appNew)
		}
	}
	// if found print new or updated apps
	if len(newUpdatedApps) > 0 {
		log.Printf("new or updated apps detected:")
		for _, app := range newUpdatedApps {
			var acPts []string
			for _, acPt := range app.AccessPoints {
				s := fmt.Sprintf("%s: %s:%d", acPt.Type, acPt.Address, acPt.Port)
				acPts = append(acPts, s)
			}
			log.Printf("   %s: %s", app.Type, strings.Join(acPts, ", "))
		}
	}
}

func (sm *appMonitor) detectApps() {
	// Kea app is being detected by browsing list of processes in the systam
	// where cmdline of the process contains given pattern with kea-ctrl-agent
	// substring. Such found processes are being processed further and all other
	// Kea daemons are discovered and queried for their versions, etc.
	keaPtrn := regexp.MustCompile(`(.*?)kea-ctrl-agent\s+.*-c\s+(\S+)`)
	// BIND 9 app is being detecting by browsing list of processes in the system
	// where cmdline of the process contains given pattern with named substring.
	bind9Ptrn := regexp.MustCompile(`(.*?)named\s+(.*)`)

	var apps []*App

	procs, _ := process.Processes()
	for _, p := range procs {
		procName, _ := p.Name()
		cmdline := ""
		cwd := ""
		var err error
		if procName == keaProcName || procName == namedProcName {
			cmdline, err = p.Cmdline()
			if err != nil {
				log.Warnf("cannot get process command line: %+v", err)
				continue
			}
			cwd, err = p.Cwd()
			if err != nil {
				log.Warnf("cannot get process current working directory: %+v", err)
				cwd = ""
			}
		}

		if procName == keaProcName {
			// detect kea
			m := keaPtrn.FindStringSubmatch(cmdline)
			if m != nil {
				keaApp := detectKeaApp(m, cwd)
				if keaApp != nil {
					keaApp.Pid = p.Pid
					apps = append(apps, keaApp)
				}
			}
			continue
		}

		if procName == namedProcName {
			// detect bind9
			m := bind9Ptrn.FindStringSubmatch(cmdline)
			if m != nil {
				cmdr := &storkutil.RealCommander{}
				bind9App := detectBind9App(m, cwd, cmdr)
				if bind9App != nil {
					bind9App.Pid = p.Pid
					apps = append(apps, bind9App)
				}
			}
			continue
		}
	}

	// check changes in apps and print them
	printNewOrUpdatedApps(apps, sm.apps)

	// remember detected apps
	sm.apps = apps
}

// Gathers the configured log files for detected apps and enables them
// for viewing from the UI.
func (sm *appMonitor) detectAllowedLogs(storkAgent *StorkAgent) {
	// Nothing to do if the agent is not set. It may be nil when running some
	// tests.
	if storkAgent == nil {
		return
	}
	for _, app := range sm.apps {
		// Only Kea apps are currently supported.
		if app.Type == AppTypeKea {
			for _, ac := range app.AccessPoints {
				if ac.Type == AccessPointControl {
					err := detectKeaAllowedLogs(storkAgent, ac.Address, ac.Port)
					if err != nil {
						err = errors.WithMessagef(err, "failed to detect log files for Kea")
						log.WithFields(
							log.Fields{
								"address": ac.Address,
								"port":    ac.Port,
							},
						).Warn(err)
					}
					break
				}
			}
		}
	}
}

func (sm *appMonitor) GetApps() []*App {
	ret := make(chan []*App)
	sm.requests <- ret
	srvs := <-ret
	return srvs
}

func (sm *appMonitor) Shutdown() {
	sm.quit <- true
	sm.wg.Wait()
}

// getAccessPoint retrieves the requested type of access point from the app.
func getAccessPoint(app *App, accessType string) (*AccessPoint, error) {
	for _, point := range app.AccessPoints {
		if point.Type != accessType {
			continue
		}

		if point.Port == 0 {
			return nil, errors.Errorf("%s access point does not have port number", accessType)
		} else if len(point.Address) == 0 {
			return nil, errors.Errorf("%s access point does not have address", accessType)
		}

		// found a good access point
		return &point, nil
	}

	return nil, errors.Errorf("%s access point not found", accessType)
}
