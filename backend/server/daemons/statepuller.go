package daemons

import (
	"context"
	"time"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	keaconfig "isc.org/stork/daemoncfg/kea"
	"isc.org/stork/datamodel/daemonname"
	"isc.org/stork/server/agentcomm"
	"isc.org/stork/server/configreview"
	"isc.org/stork/server/daemons/bind9"
	"isc.org/stork/server/daemons/kea"
	"isc.org/stork/server/daemons/pdns"
	dbops "isc.org/stork/server/database"
	dbmodel "isc.org/stork/server/database/model"
	"isc.org/stork/server/eventcenter"
	storkutil "isc.org/stork/util"
)

// Instance of the puller which periodically checks the status of the Kea daemons.
// Besides basic status information the High Availability status is fetched.
type StatePuller struct {
	*agentcomm.PeriodicPuller
	EventCenter                eventcenter.EventCenter
	ReviewDispatcher           configreview.Dispatcher
	DHCPOptionDefinitionLookup keaconfig.DHCPOptionDefinitionLookup
}

// Create an instance of the puller which periodically checks the status of
// the Kea daemons.
func NewStatePuller(db *dbops.PgDB, agents agentcomm.ConnectedAgents, eventCenter eventcenter.EventCenter, reviewDispatcher configreview.Dispatcher, lookup keaconfig.DHCPOptionDefinitionLookup) (*StatePuller, error) {
	puller := &StatePuller{
		EventCenter:                eventCenter,
		ReviewDispatcher:           reviewDispatcher,
		DHCPOptionDefinitionLookup: lookup,
	}
	periodicPuller, err := agentcomm.NewPeriodicPuller(db, agents, "State Puller",
		"state_puller_interval", puller.pullData)
	if err != nil {
		return nil, err
	}
	puller.PeriodicPuller = periodicPuller
	return puller, nil
}

// Stops the timer triggering status checks.
func (puller *StatePuller) Shutdown() {
	puller.PeriodicPuller.Shutdown()
}

// Gets the status of machines and their daemons and stores useful information in the database.
func (puller *StatePuller) pullData() error {
	// get list of all authorized machines from database
	authorized := true
	dbMachines, err := dbmodel.GetAllMachines(puller.DB, &authorized)
	if err != nil {
		return err
	}

	// get state from machines and their daemons
	var lastErr error
	okCnt := 0
	for _, dbM := range dbMachines {
		dbM2 := dbM
		ctx := context.Background()
		errStr := UpdateMachineAndDaemonsState(ctx, puller.DB, &dbM2, puller.Agents, puller.EventCenter, puller.ReviewDispatcher, puller.DHCPOptionDefinitionLookup)
		if errStr != "" {
			lastErr = errors.New(errStr)
			log.WithError(lastErr).Errorf("Error occurred while getting info from machine %d", dbM2.ID)
		} else {
			okCnt++
		}
	}
	log.Printf("Completed pulling information from machines: %d/%d succeeded", okCnt, len(dbMachines))
	return lastErr
}

// Store updated machine fields in to database.
func updateMachineFields(db *dbops.PgDB, dbMachine *dbmodel.Machine, m *agentcomm.State) error {
	// update state fields in machine
	dbMachine.State.AgentVersion = m.AgentVersion
	dbMachine.State.Cpus = m.Cpus
	dbMachine.State.CpusLoad = m.CpusLoad
	dbMachine.State.Memory = m.Memory
	dbMachine.State.Hostname = m.Hostname
	dbMachine.State.Uptime = m.Uptime
	dbMachine.State.UsedMemory = m.UsedMemory
	dbMachine.State.Os = m.Os
	dbMachine.State.Platform = m.Platform
	dbMachine.State.PlatformFamily = m.PlatformFamily
	dbMachine.State.PlatformVersion = m.PlatformVersion
	dbMachine.State.KernelVersion = m.KernelVersion
	dbMachine.State.KernelArch = m.KernelArch
	dbMachine.State.VirtualizationSystem = m.VirtualizationSystem
	dbMachine.State.VirtualizationRole = m.VirtualizationRole
	dbMachine.State.HostID = m.HostID
	dbMachine.LastVisitedAt = m.LastVisitedAt
	dbMachine.Error = m.Error
	err := dbmodel.UpdateMachine(db, dbMachine)
	if err != nil {
		return errors.Wrapf(err, "problem updating machine %+v", dbMachine)
	}
	return nil
}

// daemonCompare compares two daemons for equality. Two daemons are considered
// equal if their type matches and if they have the same control port. Return
// true if equal, false otherwise.
func daemonCompare(dbDaemon *dbmodel.Daemon, grpcDaemon *agentcomm.Daemon) bool {
	if dbDaemon.Name != grpcDaemon.Name {
		return false
	}
	if len(dbDaemon.AccessPoints) != len(grpcDaemon.AccessPoints) {
		return false
	}
	accessPointIndex := map[dbmodel.AccessPointType]*dbmodel.AccessPoint{}
	for _, pt := range dbDaemon.AccessPoints {
		accessPointIndex[pt.Type] = pt
	}

	for _, grpcPt := range grpcDaemon.AccessPoints {
		dbPt, ok := accessPointIndex[grpcPt.Type]
		if !ok {
			return false
		}

		if dbPt.Port != grpcPt.Port || dbPt.Address != grpcPt.Address || dbPt.Key != grpcPt.Key || dbPt.Protocol != grpcPt.Protocol {
			return false
		}
	}

	return true
}

// For each provided discovered daemon, try to find a matching daemon in the
// database. If it is found, use it, otherwise create a new daemon.
func mergeNewAndOldDaemons(dbMachine *dbmodel.Machine, discoveredDaemons []*agentcomm.Daemon) []*dbmodel.Daemon {
	oldDaemons := dbMachine.Daemons
	oldMatchedIndices := map[int]struct{}{}
	// We preserve all old daemons. Some of them may not be discovered anymore
	// if they are temporarily or permanently down. There is not simple and
	// reliable way to distinguish between these two cases, so we keep the old
	// daemons until they are explicitly deleted by the administrator.
	var mergedDaemons []*dbmodel.Daemon
	mergedDaemons = append(mergedDaemons, dbMachine.Daemons...)

DISCOVERED_LOOP:
	for _, discoveredDaemon := range discoveredDaemons {
		for i, oldDaemon := range oldDaemons {
			if _, ok := oldMatchedIndices[i]; ok {
				// This old daemon has already been matched with some
				// discovered daemon.
				continue
			}

			if daemonCompare(oldDaemon, discoveredDaemon) {
				oldMatchedIndices[i] = struct{}{}
				continue DISCOVERED_LOOP
			}
		}

		// The daemon was not found in the old daemons, so create a new one.
		accessPoints := make([]*dbmodel.AccessPoint, len(discoveredDaemon.AccessPoints))
		for i, point := range discoveredDaemon.AccessPoints {
			accessPoints[i] = &dbmodel.AccessPoint{
				Type:     point.Type,
				Address:  point.Address,
				Port:     point.Port,
				Key:      point.Key,
				Protocol: point.Protocol,
			}
		}

		newDaemon := dbmodel.NewDaemon(dbMachine, discoveredDaemon.Name, true, accessPoints)
		mergedDaemons = append(mergedDaemons, newDaemon)
	}

	return mergedDaemons
}

// Retrieve remotely machine and its daemons state, and store it in the database.
func UpdateMachineAndDaemonsState(ctx context.Context, db *dbops.PgDB, dbMachine *dbmodel.Machine, agents agentcomm.ConnectedAgents, eventCenter eventcenter.EventCenter, reviewDispatcher configreview.Dispatcher, lookup keaconfig.DHCPOptionDefinitionLookup) string {
	ctx2, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	// get state of machine from agent
	state, err := agents.GetState(ctx2, dbMachine)
	if err != nil {
		log.WithError(err).Warn("Cannot get state of machine")
		dbMachine.Error = "Cannot get state of machine"
		err = dbmodel.UpdateMachine(db, dbMachine)
		if err != nil {
			msg := "Problem updating record in database"
			log.WithError(err).Error(msg)
			return msg
		}
		return ""
	}

	agentVersion, err := storkutil.ParseSemanticVersion(state.AgentVersion)
	if err != nil {
		log.WithError(err).Errorf("Cannot parse agent version: %s", state.AgentVersion)
		return "Cannot parse agent version"
	}

	if agentVersion.LessThan(storkutil.NewSemanticVersion(2, 3, 2)) {
		// The agent communicates through the Kea CA.
		for _, daemon := range state.Daemons {
			if daemon.Name != daemonname.CA {
				continue
			}

			additionalDaemons, err := getDaemonsFromKeaCAConfig(ctx, agents, daemon)
			if err != nil {
				msg := "Cannot get daemons from Kea CA configuration"
				log.WithError(err).Error(msg)
				return msg
			}

			// Append the additional daemons to the list of daemons.
			state.Daemons = append(state.Daemons, additionalDaemons...)
		}
	}

	// The Stork server doesn't gather the Stork agent configuration, so we cannot
	// detect its change. It used to compare the current agent state and the database
	// entry to merely recognize the HTTP credentials state change but this
	// parameter has been removed from the agent state. The following variable is
	// a placeholder for the possible future implementation of the Stork agent
	// configuration change detection.
	isStorkAgentChanged := false

	// store machine's state in db
	err = updateMachineFields(db, dbMachine, state)
	if err != nil {
		msg := "Cannot update machine in db"
		log.WithError(err).Error(msg)
		return msg
	}

	// take old daemons from db and new daemons fetched from the machine
	// and match them and prepare a list of all daemons
	mergedDaemons := mergeNewAndOldDaemons(dbMachine, state.Daemons)

	// Group daemons by Kea, BIND 9, PowerDNS, etc.
	// It is ordered map because some existing unit tests depend on the order
	// of processing the daemons.
	nameToTypeMapping := map[daemonname.Name]string{
		daemonname.DHCPv4: "kea",
		daemonname.DHCPv6: "kea",
		daemonname.CA:     "kea",
		daemonname.D2:     "kea",
		daemonname.Bind9:  "bind9",
		daemonname.PDNS:   "pdns",
	}

	mergedDaemonsByType := storkutil.NewOrderedMap[string, []*dbmodel.Daemon]()
	for _, daemon := range mergedDaemons {
		daemonType, ok := nameToTypeMapping[daemon.Name]
		if !ok {
			log.Warnf("Unknown daemon type %s", daemon.Name)
			continue
		}

		typeDaemons, ok := mergedDaemonsByType.Get(daemonType)
		if !ok {
			typeDaemons = []*dbmodel.Daemon{}
		}
		typeDaemons = append(typeDaemons, daemon)
		mergedDaemonsByType.Set(daemonType, typeDaemons)
	}
	// List of all daemons belonging to the machine with updated state.
	allDaemons := make([]*dbmodel.Daemon, 0, len(mergedDaemons))
	// go through all daemons and store their changes in database
	for _, entry := range mergedDaemonsByType.GetEntries() {
		daemonType := entry.Key
		mergedDaemons := entry.Value

		// get daemon state from the machine
		switch daemonType {
		case "kea":
			var states []kea.DaemonStateMeta
			var enhancedDaemons []*dbmodel.Daemon
			for _, daemon := range mergedDaemons {
				enhancedDaemon, state := kea.GetDaemonWithRefreshedState(ctx2, agents, daemon)
				enhancedDaemons = append(enhancedDaemons, enhancedDaemon)
				states = append(states, state)
			}

			err = kea.CommitDaemonsIntoDB(db, enhancedDaemons, eventCenter, states, lookup)

			if err == nil {
				for i, daemon := range enhancedDaemons {
					state := states[i]
					// Let's now identify new daemons or the daemons with updated
					// configurations and schedule configuration reviews for them
					conditionallyBeginKeaConfigReviews(daemon, state, reviewDispatcher, isStorkAgentChanged)
					allDaemons = append(allDaemons, daemon)
				}
			}
		case "bind9":
			for _, daemon := range mergedDaemons {
				bind9.GetDaemonState(ctx2, agents, daemon, eventCenter)
				err = bind9.CommitDaemonIntoDB(db, daemon, eventCenter)
				if err != nil {
					break
				}
				allDaemons = append(allDaemons, daemon)
			}
		case "pdns":
			for _, daemon := range mergedDaemons {
				pdns.GetDaemonState(ctx2, agents, daemon, eventCenter)
				err = pdns.CommitDaemonIntoDB(db, daemon, eventCenter)
				if err != nil {
					break
				}
				allDaemons = append(allDaemons, daemon)
			}
		default:
			err = nil
		}

		if err != nil {
			log.WithError(err).Errorf("Cannot store daemon state")
			return "Problem storing daemon state in the database"
		}
	}

	// add all daemons to machine's daemons list - it will be used in ReST API functions
	// to return state of machine and its daemons
	dbMachine.Daemons = allDaemons

	return ""
}

// This function checks if a new config review should be performed. It is
// performed when daemon's configuration or dispatcher's signature has changed.
func conditionallyBeginKeaConfigReviews(daemon *dbmodel.Daemon, state kea.DaemonStateMeta, reviewDispatcher configreview.Dispatcher, storkAgentConfigChanged bool) {
	// Let's make sure that the config pointer is set. It can be nil
	// when the daemon is inactive.
	if daemon.KeaDaemon == nil || daemon.KeaDaemon.Config == nil {
		return
	}

	var triggers configreview.Triggers
	if storkAgentConfigChanged {
		triggers = append(triggers, configreview.StorkAgentConfigModified)
	}

	isConfigModified := true
	if !state.IsConfigChanged {
		if daemon.ConfigReview != nil &&
			daemon.ConfigReview.Signature == reviewDispatcher.GetSignature() {
			// Configuration of this daemon hasn't changed and the dispatcher has
			// no checkers modified since the last review. Skip the config modified trigger.
			isConfigModified = false
		}
	}
	if isConfigModified {
		triggers = append(triggers, configreview.ConfigModified)
	}

	if len(triggers) != 0 {
		_ = reviewDispatcher.BeginReview(daemon, triggers, nil)
	}
}

// Reads the daemons from the Kea CA configuration file.
// It is expected that the provided daemon is the Kea CA daemon.
func getDaemonsFromKeaCAConfig(ctx context.Context, agents agentcomm.ConnectedAgents, daemon *agentcomm.Daemon) ([]*agentcomm.Daemon, error) {
	// Fetch the Kea CA configuration to retrieve a list of running
	// daemons.
	config, err := kea.GetConfig(ctx, agents, daemon)
	if err != nil {
		return nil, errors.WithMessage(err, "cannot get Kea CA configuration")
	}

	daemonNames := config.GetManagementControlSockets().GetManagedDaemonNames()
	var daemons []*agentcomm.Daemon
	for _, name := range daemonNames {
		if name == daemonname.CA {
			continue
		}

		daemons = append(daemons, &agentcomm.Daemon{
			Name: name,
			// Communication with this daemon is done through the Kea CA.
			AccessPoints: daemon.AccessPoints,
			Machine:      daemon.Machine,
		})
	}
	return daemons, nil
}
