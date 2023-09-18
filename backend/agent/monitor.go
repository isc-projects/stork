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
	Type              string
	Address           string
	Port              int64
	UseSecureProtocol bool
	Key               string
}

// Currently supported types are: "control" and "statistics".
const (
	AccessPointControl    = "control"
	AccessPointStatistics = "statistics"
)

// Base application information. This structure is embedded
// in other app specific structures like KeaApp and Bind9App.
type BaseApp struct {
	Pid          int32
	Type         string
	AccessPoints []AccessPoint
}

// Specific App like KeaApp or Bind9App have to implement
// this interface. The methods should be implemented
// in a specific way in given concrete App.
type App interface {
	GetBaseApp() *BaseApp
	DetectAllowedLogs() ([]string, error)
	GetConfiguredDaemons() []string
}

// Currently supported types are: "kea" and "bind9".
const (
	AppTypeKea   = "kea"
	AppTypeBind9 = "bind9"
)

// The application monitor is responsible for detecting the applications
// running in the operating system and periodically refreshing their states.
// They are available through assessors.
type AppMonitor interface {
	GetApps() []App
	GetApp(appType, apType, address string, port int64) App
	Start(agent *StorkAgent)
	Shutdown()
}

type appMonitor struct {
	requests chan chan []App // input to app monitor, ie. channel for receiving requests
	quit     chan bool       // channel for stopping app monitor
	running  bool
	wg       *sync.WaitGroup

	apps []App // list of detected apps on the host
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
		requests: make(chan chan []App),
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
	sm.detectApps(storkAgent)

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
			sm.detectApps(storkAgent)

		case <-sm.quit:
			// exit run
			log.Printf("Stopped app monitor")
			sm.running = false
			return
		}
	}
}

func printNewOrUpdatedApps(newApps []App, oldApps []App) {
	// look for new or updated apps
	var newUpdatedApps []App
	for _, an := range newApps {
		appNew := an.GetBaseApp()
		found := false
		for _, ao := range oldApps {
			appOld := ao.GetBaseApp()
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
				if acPtNew.UseSecureProtocol != acPtOld.UseSecureProtocol {
					continue
				}
				if acPtNew.Key != acPtOld.Key {
					continue
				}
			}
			found = true
		}
		if !found {
			newUpdatedApps = append(newUpdatedApps, an)
		}
	}
	// if found print new or updated apps
	if len(newUpdatedApps) > 0 {
		log.Printf("New or updated apps detected:")
		for _, app := range newUpdatedApps {
			var acPts []string
			for _, acPt := range app.GetBaseApp().AccessPoints {
				url := storkutil.HostWithPortURL(acPt.Address, acPt.Port, acPt.UseSecureProtocol)
				authKeyFoundStr := "not found"
				if acPt.Key != "" {
					authKeyFoundStr = "found"
				}
				s := fmt.Sprintf("%s: %s (auth key: %s)", acPt.Type, url, authKeyFoundStr)
				acPts = append(acPts, s)
			}
			log.Printf("   %s: %s", app.GetBaseApp().Type, strings.Join(acPts, ", "))
		}
	} else if len(oldApps) == 0 {
		// Agent is starting up but no app to monitor has been detected.
		// Usually, the agent is installed with at least one monitored app.
		// The below message is printed for easier troubleshooting.
		log.Warnf("No Kea nor Bind9 app detected for monitoring; please check if they are running, and Stork can communicate with them.")
	}
}

func (sm *appMonitor) detectApps(storkAgent *StorkAgent) {
	// Kea app is being detected by browsing list of processes in the system
	// where cmdline of the process contains given pattern with kea-ctrl-agent
	// substring. Such found processes are being processed further and all other
	// Kea daemons are discovered and queried for their versions, etc.
	keaPattern := regexp.MustCompile(`(.*?)kea-ctrl-agent\s+.*-c\s+(\S+)`)
	// BIND 9 app is being detecting by browsing list of processes in the system
	// where cmdline of the process contains given pattern with named substring.
	bind9Pattern := regexp.MustCompile(`(.*?)named\s+(.*)`)

	var apps []App

	processes, _ := process.Processes()
	for _, p := range processes {
		procName, _ := p.Name()
		cmdline := ""
		cwd := ""
		var err error

		if procName == keaProcName || procName == namedProcName {
			cmdline, err = p.Cmdline()
			if err != nil {
				log.WithError(err).Warnf("Cannot get process command line")
				continue
			}
			cwd, err = p.Cwd()
			if err != nil {
				log.WithError(err).Warn("Cannot get process current working directory")
				cwd = ""
			}
		}

		if procName == keaProcName {
			// detect kea
			m := keaPattern.FindStringSubmatch(cmdline)
			if m != nil {
				keaApp := detectKeaApp(m, cwd, storkAgent.KeaHTTPClient)
				if keaApp != nil {
					keaApp.GetBaseApp().Pid = p.Pid
					apps = append(apps, keaApp)
				}
			}
			continue
		}

		if procName == namedProcName {
			// detect bind9
			m := bind9Pattern.FindStringSubmatch(cmdline)
			if m != nil {
				cmdr := storkutil.NewSystemCommandExecutor()
				bind9App := detectBind9App(m, cwd, cmdr)
				if bind9App != nil {
					bind9App.GetBaseApp().Pid = p.Pid
					apps = append(apps, bind9App)
				}
			}
			continue
		}
	}

	// Check changes in apps and print them.
	printNewOrUpdatedApps(apps, sm.apps)

	// Remember detected apps.
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
		paths, err := app.DetectAllowedLogs()
		if err != nil {
			ap := app.GetBaseApp().AccessPoints[0]
			err = errors.WithMessagef(err, "Failed to detect log files for Kea")
			log.WithFields(
				log.Fields{
					"address": ap.Address,
					"port":    ap.Port,
				},
			).Warn(err)
		} else {
			for _, p := range paths {
				storkAgent.logTailer.allow(p)
			}
		}
	}
}

// Get a list of detected apps by a monitor.
func (sm *appMonitor) GetApps() []App {
	ret := make(chan []App)
	sm.requests <- ret
	applications := <-ret
	return applications
}

// Get an app from a monitor that matches provided params.
func (sm *appMonitor) GetApp(appType, apType, address string, port int64) App {
	apps := sm.GetApps()
	for _, app := range apps {
		if app.GetBaseApp().Type != appType {
			continue
		}
		for _, ap := range app.GetBaseApp().AccessPoints {
			if ap.Type == apType && ap.Address == address && ap.Port == port {
				return app
			}
		}
	}
	return nil
}

// Shut down monitor. Stop its background goroutine.
func (sm *appMonitor) Shutdown() {
	sm.quit <- true
	sm.wg.Wait()
}

// getAccessPoint retrieves the requested type of access point from the app.
func getAccessPoint(app App, accessType string) (*AccessPoint, error) {
	for _, point := range app.GetBaseApp().AccessPoints {
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
