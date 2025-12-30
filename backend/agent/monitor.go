package agent

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	bind9config "isc.org/stork/daemoncfg/bind9"
	pdnsconfig "isc.org/stork/daemoncfg/pdns"
	"isc.org/stork/datamodel/daemonname"
	"isc.org/stork/datamodel/protocoltype"
	storkutil "isc.org/stork/util"
)

// Operations provided by the Stork agent to set up daemon-related configuration.
type agentManager interface {
	allowLog(path string)
	allowLeaseTracking() (bool, int)
}

// An access point for a daemon to retrieve information such
// as status or metrics.
type AccessPoint struct {
	Type     string
	Address  string
	Port     int64
	Protocol protocoltype.ProtocolType
	Key      string
}

// Checks if two access points are equal.
func (ap *AccessPoint) IsEqual(other AccessPoint) bool {
	return ap.Type == other.Type &&
		ap.Address == other.Address &&
		ap.Port == other.Port &&
		ap.Protocol == other.Protocol &&
		ap.Key == other.Key
}

// String representation of an access point.
func (ap *AccessPoint) String() string {
	var b strings.Builder
	b.WriteString(ap.Type)
	b.WriteString(": ")
	b.WriteString(storkutil.HostWithPortURL(ap.Address, ap.Port, string(ap.Protocol)))
	if ap.Type == AccessPointControl {
		b.WriteString(" (auth key: ")
		if ap.Key != "" {
			b.WriteString("found")
		} else {
			b.WriteString("not found")
		}
		b.WriteString(")")
	}
	return b.String()
}

// Currently supported types are: "control" and "statistics".
const (
	AccessPointControl    = "control"
	AccessPointStatistics = "statistics"
)

type Daemon interface {
	GetName() daemonname.Name
	// Returns a first access point of a given type. If the access point is not
	// found, it returns nil.
	GetAccessPoint(apType string) *AccessPoint
	// Returns all access points of the daemon. There may be multiple access
	// points of the same type.
	GetAccessPoints() []AccessPoint
	// Checks if two daemon instances are the same. It is used to determine
	// whether the newly detected daemon is the same as the previously detected
	// daemon. In that case, the detected daemon is ignored.
	IsSame(other Daemon) bool
	// Called when the monitor newly detects the daemon.
	// It allows the daemon to perform initialization tasks
	// such as starting a background goroutine.
	Bootstrap() error
	// Called when the monitor no longer detects the daemon.
	// It allows the daemon to perform cleanup tasks such as
	// stopping a background goroutine.
	Cleanup() error
	// Performs periodic processing of the daemon, e.g., detect logs or
	// refresh zone inventory.
	RefreshState(context.Context, agentManager) error
	String() string
}

// Daemon information. This structure is embedded
// in daemon specific structures like KeaDaemon and Bind9Daemon.
type daemon struct {
	Name         daemonname.Name
	AccessPoints []AccessPoint
}

// Return the name of the daemon process.
func (d *daemon) GetName() daemonname.Name {
	return d.Name
}

// Returns all access points of the daemon.
func (d *daemon) GetAccessPoints() []AccessPoint {
	return d.AccessPoints
}

// Returns an access point of a given type. If the access point is not found,
// it returns nil.
func (d *daemon) GetAccessPoint(accessPointType string) *AccessPoint {
	for _, ap := range d.AccessPoints {
		if ap.Type == accessPointType {
			return &ap
		}
	}
	return nil
}

// String representation of a daemon.
func (d *daemon) String() string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("%s: ", d.Name))

	for i := 0; i < len(d.AccessPoints)-1; i++ {
		b.WriteString(d.AccessPoints[i].String())
		b.WriteString(", ")
	}
	if len(d.AccessPoints) > 0 {
		b.WriteString(d.AccessPoints[len(d.AccessPoints)-1].String())
	}

	return b.String()
}

// Checks if two daemons are the same. It checks the name and access
// points including their configuration.
func (d *daemon) IsSame(other Daemon) bool {
	if d.Name != other.GetName() {
		return false
	}

	otherAccessPoints := other.GetAccessPoints()
	if len(d.AccessPoints) != len(otherAccessPoints) {
		return false
	}

	// It is expected that the access points are always detected in the same
	// order.
	for i, otherAccessPoint := range otherAccessPoints {
		thisAccessPoint := d.AccessPoints[i]
		if !thisAccessPoint.IsEqual(otherAccessPoint) {
			return false
		}
	}
	return true
}

// An interface representing the DNS daemon.
type dnsDaemon interface {
	Daemon
	isSame(other dnsDaemon) bool
	getZoneInventory() zoneInventory
	getDetectedFiles() *detectedDaemonFiles
}

// An implementation providing common functionality for DNS daemons.
type dnsDaemonImpl struct {
	daemon
	zoneInventory zoneInventory
	detectedFiles *detectedDaemonFiles
}

// Checks if the embedded daemon instance has the same name and access points
// as the other daemon instance, and that the detected files are the same as
// well. It includes checking the paths, modification times, and sizes of the
// files.
func (d *dnsDaemonImpl) isSame(other dnsDaemon) bool {
	return other != nil && d.IsSame(other) && d.getDetectedFiles().isSame(other.getDetectedFiles())
}

// Returns the zone inventory.
func (d *dnsDaemonImpl) getZoneInventory() zoneInventory {
	return d.zoneInventory
}

// Returns the files associated with the detected daemon (e.g., configuration files).
func (d *dnsDaemonImpl) getDetectedFiles() *detectedDaemonFiles {
	return d.detectedFiles
}

// Bootstrap the DNS daemon. It starts the zone inventory if available.
func (d *dnsDaemonImpl) Bootstrap() error {
	if d.zoneInventory != nil {
		d.zoneInventory.start()
	}
	return nil
}

// Refreshes the DNS daemon state. It populates the zone inventory if
// it is not ready yet.
func (d *dnsDaemonImpl) RefreshState(ctx context.Context, agentMgr agentManager) error {
	if d.zoneInventory == nil || d.zoneInventory.getCurrentState().isReady() {
		return nil
	}
	var busyError *zoneInventoryBusyError
	if _, err := d.zoneInventory.populate(false); err != nil {
		switch {
		case errors.As(err, &busyError):
			// Inventory creation is in progress. This is not an error.
			return nil
		default:
			return errors.WithMessage(err, "failed to populate DNS zone inventory")
		}
	}
	return nil
}

// Called once before the daemon is removed. It stops the zone inventory.
func (d *dnsDaemonImpl) Cleanup() error {
	if d.zoneInventory != nil {
		d.zoneInventory.stop()
	}
	return nil
}

// The daemon monitor is responsible for detecting the daemons
// running in the operating system and periodically refreshing their states.
// They are available through assessors.
type Monitor interface {
	GetDaemons() []Daemon
	GetDaemonByAccessPoint(apType, address string, port int64) Daemon
	Start(context.Context, agentManager)
	Shutdown()
}

type monitor struct {
	requests                   chan chan []Daemon // input to monitor, ie. channel for receiving requests
	quit                       chan bool          // channel for stopping daemon monitor
	running                    bool
	wg                         *sync.WaitGroup
	commander                  storkutil.CommandExecutor
	processManager             *ProcessManager
	bind9FileParser            bind9FileParser
	explicitBind9ConfigPath    string
	pdnsConfigParser           pdnsConfigParser
	explicitPowerDNSConfigPath string
	keaHTTPClientConfig        HTTPClientConfig

	// List of detected daemons on the host.
	// Nil if the monitor has no perform detection yet.
	daemons []Daemon
}

// Returns an exported interface to the monitor. It used to start it as well, but this is now done
// by a dedicated method Start(). Make sure you call Start() before using daemon
// monitor.
func NewMonitor(explicitBind9ConfigPath string, explicitPowerDNSConfigPath string, keaHTTPClientConfig HTTPClientConfig) Monitor {
	return newMonitor(explicitBind9ConfigPath, explicitPowerDNSConfigPath, keaHTTPClientConfig)
}

// Creates a new monitor instance. It is used internally by the NewMonitor function and
// in the tests.
func newMonitor(explicitBind9ConfigPath string, explicitPowerDNSConfigPath string, keaHTTPClientConfig HTTPClientConfig) *monitor {
	return &monitor{
		requests:                   make(chan chan []Daemon),
		quit:                       make(chan bool),
		wg:                         &sync.WaitGroup{},
		commander:                  storkutil.NewSystemCommandExecutor(),
		processManager:             NewProcessManager(),
		bind9FileParser:            bind9config.NewParser(),
		pdnsConfigParser:           pdnsconfig.NewParser(),
		explicitBind9ConfigPath:    explicitBind9ConfigPath,
		explicitPowerDNSConfigPath: explicitPowerDNSConfigPath,
		keaHTTPClientConfig:        keaHTTPClientConfig,
		running:                    false,
		daemons:                    nil,
	}
}

// This function starts the actual monitor. This start is delayed in case we want to only
// do command line parameters parsing, e.g. to print version or help and quit.
func (sm *monitor) Start(ctx context.Context, storkAgent agentManager) {
	sm.wg.Add(1)
	go sm.run(ctx, storkAgent)
}

// Run the main loop of the monitor. It continually detects the daemons
// running on the host and refreshes their states.
func (sm *monitor) run(ctx context.Context, storkAgent agentManager) {
	log.Printf("Started daemon monitor")

	sm.running = true
	defer sm.wg.Done()

	// Run daemon detection one time immediately at startup.
	sm.detectDaemons(ctx)

	// Refresh states of all detected daemons.
	sm.refreshDaemons(ctx, storkAgent)

	// Prepare ticker.
	const detectionInterval = 10 * time.Second
	ticker := time.NewTicker(detectionInterval)
	defer ticker.Stop()

	for {
		select {
		case ret := <-sm.requests:
			// Process user request.
			ret <- sm.daemons

		case <-ticker.C:
			// Periodic detection.
			ticker.Stop()

			sm.detectDaemons(ctx)
			sm.refreshDaemons(ctx, storkAgent)

			// Reset ticker.
			ticker.Reset(detectionInterval)

		case <-sm.quit:
			// exit run
			log.Printf("Stopped daemon monitor")
			sm.running = false
			return
		}
	}
}

// Splits the daemons into newly started, untouched (already existed), untouched (duplicated) and stopped ones.
func splitDaemonsByTransition(previous, next []Daemon) (started, unchanged, unchangedDuplicated, stopped []Daemon) {
	// Daemons no longer running.
	stoppedMap := make(map[int]bool)
	for i := 0; i < len(previous); i++ {
		stoppedMap[i] = true
	}

	// Daemons newly started.
	startedMap := make(map[int]bool)
	for i := 0; i < len(next); i++ {
		startedMap[i] = true
	}

	// Daemons unchanged.
	unchangedMap := make(map[int]bool)
	unchangedDuplicatedMap := make(map[int]bool)

	for ip, p := range previous {
		for in, n := range next {
			// The daemon detecting functions may reuse existing daemon instances
			// when configuration files appear to not have changed. In this case,
			// instead of checking if the daemons are the same (that involves some
			// IO operations), it is enough to say if the daemons have equal pointers.
			if p == n || p.IsSame(n) {
				// Daemon is still running.
				stoppedMap[ip] = false
				startedMap[in] = false
				unchangedMap[ip] = true
				if p != n {
					// Only stop the duplicated daemon if it is not the same
					// as the unchanged daemon. Otherwise, we'd stop the unchanged
					// daemon.
					unchangedDuplicatedMap[in] = true
				}
				break
			}
		}
	}

	for ip, isStopped := range stoppedMap {
		if isStopped {
			stopped = append(stopped, previous[ip])
			log.Infof("Daemon stopped: %s", previous[ip].String())
		}
	}

	for in, isStarted := range startedMap {
		if isStarted {
			started = append(started, next[in])
			log.Infof("Daemon started: %s", next[in].String())
		}
	}

	for in, isUnchanged := range unchangedMap {
		if isUnchanged {
			unchanged = append(unchanged, previous[in])
		}
	}

	for in, isUnchangedDuplicated := range unchangedDuplicatedMap {
		if isUnchangedDuplicated {
			unchangedDuplicated = append(unchangedDuplicated, next[in])
		}
	}

	return
}

// Analyzes the processes running on the host and detects supported daemons.
func (sm *monitor) detectDaemons(ctx context.Context) {
	var daemons []Daemon

	// Lists processes running on the host and detectable by the monitor.
	processes, _ := sm.processManager.ListProcesses()

	for _, p := range processes {
		daemonName := p.getDaemonName()

		switch daemonName {
		case daemonname.DHCPv4, daemonname.DHCPv6, daemonname.D2, daemonname.CA:
			// Kea DHCP server.
			detectedDaemons, err := sm.detectKeaDaemons(ctx, p)
			if err != nil {
				log.WithField("daemon", daemonName).WithError(err).Warn("Failed to detect Kea daemon(s)")
				continue
			}
			daemons = append(daemons, detectedDaemons...)

		case daemonname.Bind9:
			// BIND 9 DNS server.
			detectedDaemon, err := sm.detectBind9Daemon(p)
			if err != nil {
				log.WithError(err).Warnf("Failed to detect BIND 9 DNS server daemon")
				continue
			}
			daemons = append(daemons, detectedDaemon)
		case daemonname.PDNS:
			// PowerDNS server.
			detectedDaemon, err := sm.detectPowerDNSDaemon(p)
			if err != nil {
				log.WithError(err).Warn("Failed to detect PowerDNS server daemon")
				continue
			}
			daemons = append(daemons, detectedDaemon)
		default:
			// This should never be the case given that we list only supported processes.
			log.Warnf("Unsupported daemon name %s", daemonName)
			continue
		}
	}

	if len(daemons) == 0 && (sm.daemons == nil || len(sm.daemons) != 0) {
		// It is a first detection when no daemon is detected.
		// Agent is starting up but no daemon to monitor has been detected.
		// Usually, the agent is installed with at least one monitored daemon.
		// The below message is printed for easier troubleshooting.
		log.Warn("No daemon detected for monitoring; please check if they are running, and Stork can communicate with them.")
		sm.daemons = []Daemon{}
	}

	startedDaemons, runningDaemons, duplicatedDaemons, stoppedDaemons := splitDaemonsByTransition(sm.daemons, daemons)
	newMonitorDaemons := []Daemon{} // Non-nil slice.
	newMonitorDaemons = append(newMonitorDaemons, runningDaemons...)

	for _, d := range stoppedDaemons {
		if err := d.Cleanup(); err != nil {
			log.WithError(err).WithField("daemon", d.String()).Warn("Failed to cleanup daemon")
		}
	}

	for _, d := range duplicatedDaemons {
		if err := d.Cleanup(); err != nil {
			log.WithError(err).WithField("daemon", d.String()).Warn("Failed to cleanup duplicated daemon")
		}
	}

	for _, d := range startedDaemons {
		if err := d.Bootstrap(); err != nil {
			log.WithError(err).WithField("daemon", d.String()).Warn("Failed to bootstrap daemon")
		}
		newMonitorDaemons = append(newMonitorDaemons, d)
	}

	sm.daemons = newMonitorDaemons
}

// Refreshes states of the detected daemons.
func (sm *monitor) refreshDaemons(ctx context.Context, storkAgent agentManager) {
	for _, d := range sm.daemons {
		if err := d.RefreshState(ctx, storkAgent); err != nil {
			log.WithError(err).WithField("daemon", d.String()).Warn("Failed to refresh state of the daemon")
		}
	}
}

// Get a list of detected daemons by a monitor.
func (sm *monitor) GetDaemons() []Daemon {
	ret := make(chan []Daemon)
	sm.requests <- ret
	daemons := <-ret
	return daemons
}

// Get a daemon from a monitor that matches provided params.
func (sm *monitor) GetDaemonByAccessPoint(apType, address string, port int64) Daemon {
	for _, d := range sm.GetDaemons() {
		for _, ap := range d.GetAccessPoints() {
			if ap.Type == apType && ap.Address == address && ap.Port == port {
				return d
			}
		}
	}
	return nil
}

// Shut down monitor. Stop background goroutines.
func (sm *monitor) Shutdown() {
	for _, d := range sm.GetDaemons() {
		err := d.Cleanup()
		if err != nil {
			log.WithError(err).Warnf("Failed to cleanup daemon %s", d.String())
		}
	}
	sm.quit <- true
	sm.wg.Wait()
}
