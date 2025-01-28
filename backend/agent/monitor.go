package agent

import (
	"fmt"
	"regexp"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"
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

// Returns an access point of a given type. If the access point is not found,
// it returns nil.
func (ba *BaseApp) GetAccessPoint(accessPointType string) *AccessPoint {
	for _, ap := range ba.AccessPoints {
		if ap.Type == accessPointType {
			return &ap
		}
	}
	return nil
}

// Checks if two applications have the same type.
func (ba *BaseApp) HasEqualType(other *BaseApp) bool {
	return ba.Type == other.Type
}

// Checks if two applications have the same access points. It checks the
// location (address and port) as well as the access point configuration.
func (ba *BaseApp) HasEqualAccessPoints(other *BaseApp) bool {
	if len(ba.AccessPoints) != len(other.AccessPoints) {
		return false
	}

	for _, thisAccessPoint := range ba.AccessPoints {
		otherAccessPoint := other.GetAccessPoint(thisAccessPoint.Type)
		if otherAccessPoint == nil {
			return false
		}
		if thisAccessPoint.Address != otherAccessPoint.Address {
			return false
		}
		if thisAccessPoint.Port != otherAccessPoint.Port {
			return false
		}
		if thisAccessPoint.UseSecureProtocol != otherAccessPoint.UseSecureProtocol {
			return false
		}
		if thisAccessPoint.Key != otherAccessPoint.Key {
			return false
		}
	}
	return true
}

// Checks if two applications are the same. It checks the type and access
// points including their configuration.
func (ba *BaseApp) IsEqual(other *BaseApp) bool {
	return ba.HasEqualType(other) && ba.HasEqualAccessPoints(other)
}

// Specific App like KeaApp or Bind9App have to implement
// this interface. The methods should be implemented
// in a specific way in given concrete App.
type App interface {
	GetBaseApp() *BaseApp
	DetectAllowedLogs() ([]string, error)
	AwaitBackgroundTasks()
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
	requests       chan chan []App // input to app monitor, ie. channel for receiving requests
	quit           chan bool       // channel for stopping app monitor
	running        bool
	wg             *sync.WaitGroup
	commander      storkutil.CommandExecutor
	processManager ProcessManager
	// A flag indicating if the monitor has already detected no apps and reported it.
	isNoAppsReported bool

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
		requests:       make(chan chan []App),
		quit:           make(chan bool),
		wg:             &sync.WaitGroup{},
		commander:      storkutil.NewSystemCommandExecutor(),
		processManager: NewProcessManager(),
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

	// Populate zone inventories for all detected DNS servers.
	sm.populateZoneInventories()

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
			ticker.Stop()
			sm.detectApps(storkAgent)
			sm.detectAllowedLogs(storkAgent)
			sm.populateZoneInventories()
			ticker.Reset(detectionInterval)

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
			if appNew.IsEqual(appOld) {
				found = true
				break
			}
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
				s := fmt.Sprintf("%s: %s", acPt.Type, url)

				// The key attribute is only relevant for BIND 9 control access point.
				if app.GetBaseApp().Type == AppTypeBind9 && acPt.Type == AccessPointControl {
					authKeyFoundStr := "not found"
					if acPt.Key != "" {
						authKeyFoundStr = "found"
					}
					s += fmt.Sprintf(" (auth key: %s)", authKeyFoundStr)
				}

				acPts = append(acPts, s)
			}
			log.Printf("   %s: %s", app.GetBaseApp().Type, strings.Join(acPts, ", "))
		}
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

	processes, _ := sm.processManager.ListProcesses()

	for _, p := range processes {
		procName, _ := p.GetName()
		cmdline := ""
		cwd := ""
		var err error

		if procName == keaProcName || procName == namedProcName {
			cmdline, err = p.GetCmdline()
			if err != nil {
				log.WithError(err).Warn("Cannot get process command line")
				continue
			}
			cwd, err = p.GetCwd()
			if err != nil {
				log.WithError(err).Warn("Cannot get process current working directory")
				cwd = ""
			}
		}

		switch procName {
		case keaProcName:
			// Detect Kea.
			m := keaPattern.FindStringSubmatch(cmdline)
			if m != nil {
				// Detect the app.
				keaApp, err := detectKeaApp(m, cwd, storkAgent.KeaHTTPClientConfig)
				if err != nil {
					log.WithError(err).Warn("Failed to detect Kea app")
					continue
				}

				// Look for the previously detected application.
				var recentlyActiveDaemons []string
				for _, app := range sm.apps {
					if keaApp.GetBaseApp().IsEqual(app.GetBaseApp()) {
						recentlyActiveDaemons = app.(*KeaApp).ActiveDaemons
						break
					}
				}

				// Detect the active daemons.
				keaApp.ActiveDaemons, err = detectKeaActiveDaemons(keaApp, recentlyActiveDaemons)
				if err != nil {
					log.WithError(err).Warn("Failed to detect active Kea daemons")
				}

				keaApp.GetBaseApp().Pid = p.GetPid()
				apps = append(apps, keaApp)
			}
		case namedProcName:
			// detect bind9
			m := bind9Pattern.FindStringSubmatch(cmdline)
			if m != nil {
				bind9App := detectBind9App(
					m,
					cwd,
					sm.commander,
					storkAgent.ExplicitBind9ConfigPath,
				)
				if bind9App != nil {
					// Check if this app already exists. If it does we want to use
					// an existing app to preserve its state.
					if i := slices.IndexFunc(sm.apps, func(app App) bool {
						return app.GetBaseApp().IsEqual(bind9App.GetBaseApp())
					}); i >= 0 {
						bind9App = sm.apps[i].(*Bind9App)
					}
					bind9App.GetBaseApp().Pid = p.GetPid()
					apps = append(apps, bind9App)
				}
			}
		default:
			continue
		}
	}

	// Check changes in apps and print them.
	if len(apps) == 0 {
		if !sm.isNoAppsReported {
			// Agent is starting up but no app to monitor has been detected.
			// Usually, the agent is installed with at least one monitored app.
			// The below message is printed for easier troubleshooting.
			log.Warn("No Kea nor Bind9 app detected for monitoring; please check if they are running, and Stork can communicate with them.")
			// Mark this message as reported to avoid printing it continuously.
			sm.isNoAppsReported = true
		}
	} else {
		printNewOrUpdatedApps(apps, sm.apps)
		sm.isNoAppsReported = false
	}

	// Wait for the zone inventories to complete pending operations.
	for _, app := range sm.apps {
		app.AwaitBackgroundTasks()
	}

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

// Iterates over the detected BIND9 apps and populates their zone inventories.
func (sm *appMonitor) populateZoneInventories() {
	for _, app := range sm.apps {
		if bind9app, ok := app.(*Bind9App); ok {
			if bind9app.zoneInventory == nil || bind9app.zoneInventory.getCurrentState().isReady() {
				continue
			}
			var busyError *zoneInventoryBusyError
			if _, err := bind9app.zoneInventory.populate(false); err != nil {
				switch {
				case errors.As(err, &busyError):
					// Inventory creation is in progress. This is not an error.
					continue
				default:
					log.WithError(err).Error("Failed to populate DNS zones inventory")
				}
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
	for _, app := range sm.GetApps() {
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

// Shut down monitor. Stop background goroutines.
func (sm *appMonitor) Shutdown() {
	for _, app := range sm.GetApps() {
		app.AwaitBackgroundTasks()
	}
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
