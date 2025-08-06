package agent

import (
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	bind9config "isc.org/stork/appcfg/bind9"
	pdnsconfig "isc.org/stork/appcfg/pdns"
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

// An interface to be implemented by all apps detected and monitored
// by the agent. Not all functions are relevant to all apps. For example,
// getting zone inventory is only relevant for DNS servers. For other apps
// it should return nil.
type App interface {
	GetBaseApp() *BaseApp
	DetectAllowedLogs() ([]string, error)
	GetZoneInventory() *zoneInventory
	StopZoneInventory()
}

// Supported app types: "kea", "bind9" and "pdns".
const (
	AppTypeKea      = "kea"
	AppTypeBind9    = "bind9"
	AppTypePowerDNS = "pdns"
)

// The application monitor is responsible for detecting the applications
// running in the operating system and periodically refreshing their states.
// They are available through assessors.
type AppMonitor interface {
	GetApps() []App
	GetApp(apType, address string, port int64) App
	Start(agent *StorkAgent)
	Shutdown()
}

type appMonitor struct {
	requests         chan chan []App // input to app monitor, ie. channel for receiving requests
	quit             chan bool       // channel for stopping app monitor
	running          bool
	wg               *sync.WaitGroup
	commander        storkutil.CommandExecutor
	processManager   *ProcessManager
	bind9FileParser  bind9FileParser
	pdnsConfigParser pdnsConfigParser
	// A flag indicating if the monitor has already detected no apps and reported it.
	isNoAppsReported bool

	apps []App // list of detected apps on the host
}

// Names of apps that are being detected.
const (
	keaProcName   = "kea-ctrl-agent"
	namedProcName = "named"
	pdnsProcName  = "pdns_server"
)

// Creates an AppMonitor instance. It used to start it as well, but this is now done
// by a dedicated method Start(). Make sure you call Start() before using app monitor.
func NewAppMonitor() AppMonitor {
	sm := &appMonitor{
		requests:         make(chan chan []App),
		quit:             make(chan bool),
		wg:               &sync.WaitGroup{},
		commander:        storkutil.NewSystemCommandExecutor(),
		processManager:   NewProcessManager(),
		bind9FileParser:  bind9config.NewParser(),
		pdnsConfigParser: pdnsconfig.NewParser(),
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

func printNewOrUpdatedApps(detectedApps []App, existingApps []App) {
	// Check if the detected apps are new or updated.
	var newOrUpdatedApps []App
	for _, detectedApp := range detectedApps {
		if !slices.ContainsFunc(existingApps, func(existingApp App) bool {
			return detectedApp.GetBaseApp().IsEqual(existingApp.GetBaseApp())
		}) {
			newOrUpdatedApps = append(newOrUpdatedApps, detectedApp)
		}
	}
	if len(newOrUpdatedApps) == 0 {
		// No new or updated apps detected.
		return
	}
	log.Infof("New or updated apps detected:")
	for _, app := range newOrUpdatedApps {
		var accessPoints []string
		for _, accessPoint := range app.GetBaseApp().AccessPoints {
			url := storkutil.HostWithPortURL(accessPoint.Address, accessPoint.Port, accessPoint.UseSecureProtocol)
			var b strings.Builder
			b.WriteString(accessPoint.Type)
			b.WriteString(": ")
			b.WriteString(url)

			// The key attribute is currently only relevant for BIND 9 and PowerDNS servers.
			if (app.GetBaseApp().Type == AppTypeBind9 || app.GetBaseApp().Type == AppTypePowerDNS) && accessPoint.Type == AccessPointControl {
				b.WriteString(" (auth key: ")
				if accessPoint.Key != "" {
					b.WriteString("found")
				} else {
					b.WriteString("not found")
				}
				b.WriteString(")")
			}

			accessPoints = append(accessPoints, b.String())
		}
		log.Printf("   %s: %s", app.GetBaseApp().Type, strings.Join(accessPoints, ", "))
	}
}

func (sm *appMonitor) detectApps(storkAgent *StorkAgent) {
	var apps []App

	// Lists processes running on the host and detectable by the monitor.
	processes, _ := sm.processManager.ListProcesses()

	for _, p := range processes {
		procName, _ := p.getName()
		var (
			detectedApp App
			err         error
		)
		switch procName {
		case keaProcName:
			// Kea DHCP server.
			detectedApp, err = detectKeaApp(p, storkAgent.KeaHTTPClientConfig)
			if err != nil {
				log.WithError(err).Warn("Failed to detect Kea app")
				continue
			}

			// Look for the previously detected application.
			var recentlyActiveDaemons []string
			if i := slices.IndexFunc(sm.apps, func(app App) bool {
				return app.GetBaseApp().IsEqual(detectedApp.GetBaseApp())
			}); i >= 0 {
				recentlyActiveDaemons = sm.apps[i].(*KeaApp).ActiveDaemons
			}

			// Detect the active daemons.
			keaApp := detectedApp.(*KeaApp)
			keaApp.ActiveDaemons, err = detectKeaActiveDaemons(keaApp, recentlyActiveDaemons)
			if err != nil {
				log.WithError(err).Warn("Failed to detect active Kea daemons")
			}
		case namedProcName:
			// BIND 9 DNS server.
			if detectedApp, err = detectBind9App(
				p,
				sm.commander,
				storkAgent.ExplicitBind9ConfigPath,
				sm.bind9FileParser,
			); err != nil {
				log.WithError(err).Warnf("Failed to detect BIND 9 DNS server app")
				continue
			}
			for _, app := range sm.apps {
				if app.GetBaseApp().IsEqual(detectedApp.GetBaseApp()) {
					existingApp := app.(*Bind9App)
					if existingApp.zoneInventory != nil {
						// Stop the zone inventory of the detected app because we're going
						// inherit the zone inventory from the existing app. This is a
						// temporary solution to be removed with:
						// https://gitlab.isc.org/isc-projects/stork/-/issues/1934
						detectedApp.StopZoneInventory()
						detectedApp = existingApp
					}
					break
				}
			}
		case pdnsProcName:
			// PowerDNS server.
			if detectedApp, err = detectPowerDNSApp(p, sm.commander, storkAgent.ExplicitPowerDNSConfigPath, sm.pdnsConfigParser); err != nil {
				log.WithError(err).Warn("Failed to detect PowerDNS server app")
				continue
			}
			for _, app := range sm.apps {
				if app.GetBaseApp().IsEqual(detectedApp.GetBaseApp()) {
					existingApp := app.(*PDNSApp)
					if existingApp.zoneInventory != nil {
						// Stop the zone inventory of the detected app because we're going
						// inherit the zone inventory from the existing app. This is a
						// temporary solution to be removed with:
						// https://gitlab.isc.org/isc-projects/stork/-/issues/1934
						detectedApp.StopZoneInventory()
						detectedApp = existingApp
					}
					break
				}
			}
		default:
			// This should never be the case given that we list only supported processes.
			continue
		}
		detectedApp.GetBaseApp().Pid = p.getPid()
		apps = append(apps, detectedApp)
	}

	// Check changes in apps and print them.
	if len(apps) == 0 {
		if !sm.isNoAppsReported {
			// Agent is starting up but no app to monitor has been detected.
			// Usually, the agent is installed with at least one monitored app.
			// The below message is printed for easier troubleshooting.
			log.Warn("No app detected for monitoring; please check if they are running, and Stork can communicate with them.")
			// Mark this message as reported to avoid printing it continuously.
			sm.isNoAppsReported = true
		}
	} else {
		printNewOrUpdatedApps(apps, sm.apps)
		sm.isNoAppsReported = false
	}

	// Stop no longer used zone inventories and wait for the completion of
	// the pending operations.
	for _, app := range sm.apps {
		if !slices.Contains(apps, app) {
			app.StopZoneInventory()
		}
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

// Iterates over the detected DNS apps and populates their zone inventories.
func (sm *appMonitor) populateZoneInventories() {
	for _, app := range sm.apps {
		zoneInventory := app.GetZoneInventory()
		if zoneInventory == nil || zoneInventory.getCurrentState().isReady() {
			continue
		}
		var busyError *zoneInventoryBusyError
		if _, err := zoneInventory.populate(false); err != nil {
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

// Get a list of detected apps by a monitor.
func (sm *appMonitor) GetApps() []App {
	ret := make(chan []App)
	sm.requests <- ret
	applications := <-ret
	return applications
}

// Get an app from a monitor that matches provided params.
func (sm *appMonitor) GetApp(apType, address string, port int64) App {
	for _, app := range sm.GetApps() {
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
		app.StopZoneInventory()
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
