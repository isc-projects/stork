package configreview

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	pkgerrors "github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	dbops "isc.org/stork/server/database"
	dbmodel "isc.org/stork/server/database/model"
	storkutil "isc.org/stork/util"
)

// A constant value bumped to enforce new config reviews after
// the server is started. Typically, it should be bumped when
// an implementation of any checker was modified but the
// dispatch groups were not changed.
const enforceDispatchSeq = 1

// Callback function invoked when configuration review is completed
// for a daemon. The first argument holds an ID of a daemon for
// which the review has been performed. The second argument holds an
// error which occurred during the review or nil.
type CallbackFunc func(int64, error)

// A type of an event triggering config review.
type Trigger string

// Collection of triggers.
type Triggers []Trigger

// Types of events triggering config review. Typically, a review is
// triggered when daemon configuration change has been detected.
// However, some config checkers also use other source of data that,
// if modified, should also trigger config reviews. Another way to
// trigger new config review is a manual run from the the UI. The
// special type, internal run, is used internally by the dispatcher.
const (
	// Config review is triggered internally by the dispatcher.
	internalRun Trigger = "internal"
	// Config review is triggered manually by a user over REST API.
	ManualRun Trigger = "manual"
	// Config review is triggered as a result of the configuration
	// change.
	ConfigModified Trigger = "config change"
	// Config review is triggered as a result of the hosts modifications
	// in the host database.
	DBHostsModified Trigger = "host reservations change"
)

// Returns default config review triggers. They are by default used by
// all configuration checkers:
// - ManualRun
// - ConfigModified.
func GetDefaultTriggers() Triggers {
	return Triggers{ManualRun, ConfigModified}
}

// Convenience function returning a combination of default triggers and
// the triggers specified as the function arguments.
func ExtendDefaultTriggers(triggers ...Trigger) Triggers {
	return append(GetDefaultTriggers(), triggers...)
}

// A wrapper around the config report returned by a checker. It
// supplies additional information to the report (i.e., name of
// the checker that generated the report).
type taggedReport struct {
	checkerName string
	report      *Report
}

// Review context is valid throughout a review of a daemon configuration.
// It holds information about the daemons which configurations are
// reviewed, and the review results output by the review report
// checkers. Checkers can modify the context information (e.g. fetch
// additional daemon configurations as necessary). The following
// fields are available in the context:
// - db: pointer to the database instance
// - subjectDaemon: a daemon for which review is conducted
// - refDaemons: daemons fetched by the checkers during the review
// (e.g. daemons which configurations are associated with the subject
// daemon configuration),
// - reports: configuration reports produced so far,
// - callback: user callback to invoke after the review,
// - trigger: a trigger that started the current review.
type ReviewContext struct {
	db            *dbops.PgDB
	subjectDaemon *dbmodel.Daemon
	refDaemons    []*dbmodel.Daemon
	reports       []taggedReport
	callback      CallbackFunc
	trigger       Trigger
}

// Creates new review context instance.
func newReviewContext(db *dbops.PgDB, daemon *dbmodel.Daemon, trigger Trigger, callback CallbackFunc) *ReviewContext {
	ctx := &ReviewContext{
		db:            db,
		subjectDaemon: daemon,
		callback:      callback,
		trigger:       trigger,
	}
	return ctx
}

// Dispatch group selector is used to segregate different configuration
// review checkers by daemon types.
type DispatchGroupSelector int

// A slice of the DispatchGroupSelector values.
type DispatchGroupSelectors []DispatchGroupSelector

// Enumeration listing supported configuration review checker groups.
// For instance, checkers belonging to the KeaDHCPDaemon group are
// used for reviewing DHCPv4 and DHCPv6 servers configurations. The
// checkers belonging to the KeaDaemon group are used to review
// the configuration parts shared by all Kea daemons. And so on...
const (
	EachDaemon DispatchGroupSelector = iota
	KeaDaemon
	KeaCADaemon
	KeaDHCPDaemon
	KeaDHCPv4Daemon
	KeaDHCPv6Daemon
	KeaD2Daemon
	Bind9Daemon
)

// Returns group selectors for selecting registered checkers appropriate
// for the specified daemon name. For example, it returns EachDaemon,
// KeaDaemon, KeaDHCPDaemon and KeaDHCPv4Daemon selector for the "dhcp4"
// daemon. The corresponding checkers will be used to review the
// daemon configuration.
func getDispatchGroupSelectors(daemonName string) DispatchGroupSelectors {
	switch daemonName {
	case "dhcp4":
		return DispatchGroupSelectors{EachDaemon, KeaDaemon, KeaDHCPDaemon, KeaDHCPv4Daemon}
	case "dhcp6":
		return DispatchGroupSelectors{EachDaemon, KeaDaemon, KeaDHCPDaemon, KeaDHCPv6Daemon}
	case "ca":
		return DispatchGroupSelectors{EachDaemon, KeaDaemon, KeaCADaemon}
	case "d2":
		return DispatchGroupSelectors{EachDaemon, KeaDaemon, KeaD2Daemon}
	case "bind9":
		return DispatchGroupSelectors{EachDaemon, Bind9Daemon}
	}
	log.WithFields(log.Fields{
		"daemon_name": daemonName,
	}).Warn("Config review dispatcher was unable to recognize the daemon by name and assign any suitable dispatch groups. Please notify the ISC Stork Development Team about this issue.")

	return DispatchGroupSelectors{}
}

// Dispatch group is a group of checkers registered for the particular
// dispatch group selector, e.g. for the KeaDHCPv4Daemon. Typically,
// checkers from multiple dispatch groups are used for reviewing
// a single configuration.
type dispatchGroup struct {
	checkers []*checker
	// A map of triggers to the counts of checkers using them.
	triggerRefCounts map[Trigger]int64
}

// Creates new dispatch group instance.
func newDispatchGroup() *dispatchGroup {
	return &dispatchGroup{
		triggerRefCounts: make(map[Trigger]int64),
	}
}

func (g *dispatchGroup) appendChecker(checker *checker) {
	g.checkers = append(g.checkers, checker)
	for _, trigger := range checker.triggers {
		if _, ok := g.triggerRefCounts[trigger]; !ok {
			g.triggerRefCounts[trigger] = 0
		}
		g.triggerRefCounts[trigger]++
	}
}

// Stringer implementation for a dispatchGroup. It lists checker names
// as a slice. It excludes checker function pointers making the output
// consistent across function runs. The dispatcher uses this function
// to create a signature.
func (g dispatchGroup) String() string {
	names := []string{}
	for _, checker := range g.checkers {
		names = append(names, checker.name)
	}
	return fmt.Sprintf("%+v", names)
}

// Checks if the dispatch group has checkers registered that are launched
// for the specified trigger.
func (g *dispatchGroup) hasCheckersForTrigger(trigger Trigger) bool {
	if count, ok := g.triggerRefCounts[trigger]; ok {
		return count > 0
	}
	return false
}

// The dispatcher coordinates configuration reviews of all daemons.
// The dispatcher maintains a list of configuration review checkers.
// They are segregated into dispatch groups invoked for different daemon
// types. Checkers are registered and associated with the dispatch
// groups by their implementers, and the registration takes place before
// the dispatcher start. The checkers perform specific checks on the
// selected part of the configuration and generate reports about issues.
// The configuration review can end with a list of reports or an empty
// list when no issues are found. A caller schedules a review by calling
// the BeginReview function. It schedules the review and returns
// immediately. The dispatcher runs the configuration through the
// registered checkers in the background. The dispatcher inserts the
// reports into the database when the review finishes, replacing any
// existing reports for the daemon. More advanced checkers can look
// into more than one daemon's configuration (e.g., to verify the
// consistency of the HA configuration between two partners).
type dispatcherImpl struct {
	// Database instance where configuration reports are stored.
	db *dbops.PgDB
	// Config review dispatch groups containing checkers segregated
	// into groups by daemon types.
	groups map[DispatchGroupSelector]*dispatchGroup
	// Wait group used to gracefully stop the dispatcher when the server
	// is shutdown. It waits for the remaining work to complete.
	shutdownWg *sync.WaitGroup
	// Wait group used to synchronize the ongoing reviews.
	reviewWg *sync.WaitGroup
	// Dispatcher main mutex.
	mutex *sync.RWMutex
	// Channel for passing ready review reports to the worker
	// goroutine populating the reports into the database.
	reviewDoneChan chan *ReviewContext
	// Context used for cancelling the worker goroutine when the
	// dispatcher is stopped.
	dispatchCtx context.Context
	// Function cancelling the worker goroutine.
	cancelDispatch context.CancelFunc
	// A map holding information about currently scheduled reviews.
	state map[int64]bool
	// Current value of the enforceDispatchSeq.
	enforceSeq int
	// Checker controller manages the state of configuration checkers.
	checkerController checkerController
}

// Dispatcher interface. The interface is used in the unit tests that
// require replacing the default implementation with a mock dispatcher.
type Dispatcher interface {
	RegisterChecker(selector DispatchGroupSelector, checkerName string, triggers Triggers, checkFn func(*ReviewContext) (*Report, error))
	UnregisterChecker(selector DispatchGroupSelector, checkerName string) bool
	GetCheckersMetadata(daemonID int64, daemonName string) []*CheckerMetadata
	GetSignature() string
	Start()
	Shutdown()
	BeginReview(daemon *dbmodel.Daemon, trigger Trigger, callback CallbackFunc) bool
	ReviewInProgress(daemonID int64) bool
}

// Creates new context instance when a review is scheduled. The daemon
// is a pointer to a daemon instance for which the review is
// performed. The trigger as a trigger that started the current
// review. The callback is a callback function invoked after the
// review is completed.
func (d *dispatcherImpl) newContext(db *dbops.PgDB, daemon *dbmodel.Daemon, trigger Trigger, callback CallbackFunc) *ReviewContext {
	ctx := newReviewContext(db, daemon, trigger, callback)
	return ctx
}

// A worker function receiving ready configuration reports and populating
// them into the database.
func (d *dispatcherImpl) awaitReports() {
	defer d.shutdownWg.Done()
	for {
		// Wait for ready reviews or for a signal that the dispatcher is
		// being stopped.
		select {
		case ctx := <-d.reviewDoneChan:
			if ctx != nil {
				// Received a review context which means that the review of
				// a daemon's configuration is completed and the reports can
				// be inserted into the database.
				err := d.populateReports(ctx)
				if err != nil {
					log.Errorf("Problem populating configuration review reports to the database for daemon %d: %+v",
						ctx.subjectDaemon.ID, err)
				} else {
					log.WithFields(log.Fields{
						"daemon_id":     ctx.subjectDaemon.ID,
						"reports_count": len(ctx.reports),
					}).Info("configuration review completed")
				}
				// Notify a caller that the review is finished if the caller
				// supplied a callback function.
				if ctx.callback != nil {
					ctx.callback(ctx.subjectDaemon.ID, err)
				}
			}
		case <-d.dispatchCtx.Done():
			// The dispatcher is being stopped. There may be some reviews
			// still in progress. We need to wait for them to complete.
			allReviewsDone := int32(0)
			wg := &sync.WaitGroup{}
			wg.Add(1)

			// Launch a goroutine waiting for the outstanding reviews. It needs
			// a new goroutine because in the current goroutine we need to call
			// a blocking d.reviewWg.Wait() to wait for all outstanding reviews.
			go func() {
				defer wg.Done()
				for {
					select {
					case ctx := <-d.reviewDoneChan:
						// Next outstanding review finished.
						err := d.populateReports(ctx)
						if err != nil {
							log.Errorf("Problem populating configuration review reports to the database for daemon %d: %+v",
								ctx.subjectDaemon.ID, err)
						} else {
							log.WithFields(log.Fields{
								"daemon_id":     ctx.subjectDaemon.ID,
								"reports_count": len(ctx.reports),
							}).Info("configuration review completed")
						}
						if ctx.callback != nil {
							ctx.callback(ctx.subjectDaemon.ID, nil)
						}
					default:
						// The default case is non-blocking. This code path is
						// taken when there are no new ready reviews sent over
						// the reviewDoneChan. Let's check if there are any
						// reviews ongoing. If there are none, we're done.
						// Otherwise, wait for another 50ms to avoid active
						// waiting hammering the server.
						if atomic.LoadInt32(&allReviewsDone) != 0 {
							return
						}
						time.Sleep(50 * time.Millisecond)
					}
				}
			}()
			// Wait for the outstanding reviews to complete.
			d.reviewWg.Wait()

			// All reviews done. Let's signal it to the goroutine.
			atomic.StoreInt32(&allReviewsDone, 1)

			// Wait for the goroutine to finish.
			wg.Wait()

			return
		}
	}
}

// Goroutine performing configuration review for a daemon.
func (d *dispatcherImpl) runForDaemon(daemon *dbmodel.Daemon, trigger Trigger, dispatchGroupSelectors DispatchGroupSelectors, callback CallbackFunc) {
	defer d.reviewWg.Done()

	ctx := d.newContext(d.db, daemon, trigger, callback)

	// If this is an internal run, the dispatch group selectors haven't
	// been determined in the beginReview function.
	var selectors DispatchGroupSelectors
	if trigger == internalRun {
		selectors = getDispatchGroupSelectors(daemon.Name)
	} else {
		selectors = dispatchGroupSelectors
	}

	for _, selector := range selectors {
		if group := d.getGroup(selector); group != nil {
			for i := range group.checkers {
				report, err := group.checkers[i].checkFn(ctx)
				if err != nil {
					log.Errorf("Malformed report created by the config review checker %s: %+v",
						group.checkers[i].name, err)
				}
				if report != nil {
					ctx.reports = append(ctx.reports, taggedReport{
						checkerName: group.checkers[i].name,
						report:      report,
					})
				}
			}
		}
	}
	d.reviewDoneChan <- ctx
}

// Internal function scheduling a new review. Comparing to the exported function,
// BeginReview, it also accepts "internalRun" trigger value. In that case the
// populateReports function takes slightly different path when it inserts new
// reports to the database.
func (d *dispatcherImpl) beginReview(daemon *dbmodel.Daemon, trigger Trigger, callback CallbackFunc) bool {
	var dispatchGroupSelectors DispatchGroupSelectors

	// The specified trigger indicates why the review is scheduled. The internally
	// scheduled review is unconditional. In some cases the review may be skipped
	// when none of the current config checkers are activated for this trigger.
	shouldRun := (trigger == internalRun)
	if !shouldRun {
		// Not an internal run. See if there are any checkers for this trigger.
		dispatchGroupSelectors = getDispatchGroupSelectors(daemon.Name)
		for _, selector := range dispatchGroupSelectors {
			if group := d.getGroup(selector); group != nil {
				if group.hasCheckersForTrigger(trigger) {
					shouldRun = true
					break
				}
			}
		}
	}
	if !shouldRun {
		return false
	}

	if trigger != internalRun {
		log.WithFields(log.Fields{
			"daemon_id": daemon.ID,
			"name":      daemon.Name,
			"trigger":   trigger,
		}).Info("scheduling a new configuration review")
	}

	// Check if another review for this daemon has been already scheduled. Do not
	// schedule new review if one is already in progress.
	d.mutex.RLock()
	inProgress, ok := d.state[daemon.ID]
	d.mutex.RUnlock()
	if ok && inProgress {
		// Another review in progress. Do not run another one.
		return false
	}

	d.mutex.Lock()
	inProgress, ok = d.state[daemon.ID]
	if ok && inProgress {
		d.mutex.Unlock()
		return false
	}
	d.state[daemon.ID] = true
	d.mutex.Unlock()

	d.reviewWg.Add(1)
	// Run the review in the background.
	go d.runForDaemon(daemon, trigger, dispatchGroupSelectors, callback)
	return true
}

// Inserts new config review reports into the database.
func (d *dispatcherImpl) populateReports(ctx *ReviewContext) (err error) {
	// Ensure that the state indicates that the review is no longer
	// in progress when this function returns.
	defer func() {
		d.mutex.Lock()
		defer d.mutex.Unlock()
		d.state[ctx.subjectDaemon.ID] = false
	}()

	daemons := append([]*dbmodel.Daemon{ctx.subjectDaemon}, ctx.refDaemons...)

	// Begin a new transaction for inserting the reports.
	tx, err := d.db.Begin()
	if err != nil {
		return
	}
	defer dbops.RollbackOnError(tx, &err)

	// The following calls serve two purposes. Firstly, they lock the daemons
	// and other information in the database, so no other transaction can
	// modify or delete the daemon while we insert the review reports.
	// Secondly, we get the most current information about the daemons to
	// see if the information in the database is consistent with our copy
	// which we were using to perform the review. In theory, the configuration
	// could have been modified while we were performing the review. We can
	// see if it was the case by comparing the configuration hashes.
	var dbDaemons []*dbmodel.Daemon
	if ctx.subjectDaemon.KeaDaemon == nil {
		dbDaemons, err = dbmodel.GetDaemonsForUpdate(tx, daemons)
	} else {
		dbDaemons, err = dbmodel.GetKeaDaemonsForUpdate(tx, daemons)
	}
	if err != nil {
		return
	}

	// Ensure that all daemons were in the database.
	if len(dbDaemons) != len(daemons) {
		err = pkgerrors.New("some daemons with reviewed configuration are missing from the database")
		return
	}

	// Check if the configuration of any of the daemons has changed.
	for _, dbDaemon := range dbDaemons {
		for _, daemon := range daemons {
			if daemon.ID == dbDaemon.ID {
				if daemon.KeaDaemon != nil && dbDaemon.KeaDaemon != nil &&
					daemon.KeaDaemon.ConfigHash != dbDaemon.KeaDaemon.ConfigHash {
					err = pkgerrors.Errorf("Kea daemon %d configuration has changed since the config review began", daemon.ID)
					return
				}
			}
		}
	}

	if ctx.trigger != internalRun {
		// Delete configuration reports for all daemons involved in our review.
		// It includes the reports for daemons only referenced in the review
		// (e.g., a HA peer's configuration). For those daemons, we will have to
		// run configuration reviews internally (at the end of this function) to
		// ensure they have up-to-date reports. Note that we delete the reports
		// for the referenced daemons because configuration change of the subject
		// daemon can also affect reviews for the referenced daemons.
		for i, daemon := range daemons {
			// A subject daemon can appear twice in this slice. Let's ensure
			// we delete the config reports for this daemon only once.
			if i == 0 || daemon.ID != ctx.subjectDaemon.ID {
				err = dbmodel.DeleteConfigReportsByDaemonID(tx, daemon.ID)
				if err != nil {
					return
				}
			}
		}
	}

	// Add configuration reports.
	for _, r := range ctx.reports {
		var assoc []*dbmodel.Daemon
		for _, id := range r.report.refDaemonIDs {
			assoc = append(assoc, &dbmodel.Daemon{
				ID: id,
			})
		}
		cr := &dbmodel.ConfigReport{
			CheckerName: r.checkerName,
			Content:     r.report.content,
			DaemonID:    r.report.daemonID,
			RefDaemons:  assoc,
		}
		err = dbmodel.AddConfigReport(tx, cr)
		if err != nil {
			return
		}
	}

	// Add configuration review summary.
	// todo: add config review summary for BIND9. Currently we don't because
	// BIND 9 does not include a config hash.
	if ctx.subjectDaemon.KeaDaemon != nil {
		configReview := &dbmodel.ConfigReview{
			ConfigHash: ctx.subjectDaemon.KeaDaemon.ConfigHash,
			Signature:  d.GetSignature(),
			DaemonID:   ctx.subjectDaemon.ID,
		}
		err = dbmodel.AddConfigReview(tx, configReview)
		if err != nil {
			return
		}
	}

	err = tx.Commit()
	if err != nil {
		return
	}

	if ctx.trigger != internalRun {
		// If the review was scheduled externally, and we deleted configuration
		// reports for referenced daemons, we have to rebuild the reports for
		// those daemons. Let's schedule internal reviews for each of them.
		for i := range ctx.refDaemons {
			// Do not schedule the review for the subject daemon because we're
			// now doing its review.
			if ctx.refDaemons[i].ID != ctx.subjectDaemon.ID {
				_ = d.beginReview(ctx.refDaemons[i], internalRun, ctx.callback)
			}
		}
	}

	return err
}

// Returns dispatch group indicated by the selector or nil when such group
// does not exist.
func (d *dispatcherImpl) getGroup(selector DispatchGroupSelector) *dispatchGroup {
	if g, ok := d.groups[selector]; ok {
		return g
	}
	return nil
}

// Creates new dispatcher instance.
func NewDispatcher(db *dbops.PgDB) Dispatcher {
	ctx, cancel := context.WithCancel(context.Background())
	dispatcher := &dispatcherImpl{
		db:                db,
		groups:            make(map[DispatchGroupSelector]*dispatchGroup),
		shutdownWg:        &sync.WaitGroup{},
		reviewWg:          &sync.WaitGroup{},
		mutex:             &sync.RWMutex{},
		reviewDoneChan:    make(chan *ReviewContext),
		dispatchCtx:       ctx,
		cancelDispatch:    cancel,
		state:             make(map[int64]bool),
		enforceSeq:        enforceDispatchSeq,
		checkerController: newCheckerController(),
	}
	return dispatcher
}

// Registers new checker. A checker implements an algorithm to verify
// a single configuration piece (or aspect) and output a suitable report
// if it finds issues. It should return nil when no issues were found.
// Each checker is assigned a unique name so it will be possible to
// list available checkers and/or selectively disable them.
func (d *dispatcherImpl) RegisterChecker(selector DispatchGroupSelector, checkerName string, triggers Triggers, checkFn func(*ReviewContext) (*Report, error)) {
	group := d.getGroup(selector)
	if group == nil {
		group = newDispatchGroup()
		d.groups[selector] = group
	}

	group.appendChecker(
		&checker{
			name:     checkerName,
			triggers: triggers,
			checkFn:  checkFn,
		},
	)
}

// Unregisters a checker from a dispatch group. It returns a boolean
// value indicating if the matching checker was found and removed (if true).
func (d *dispatcherImpl) UnregisterChecker(selector DispatchGroupSelector, checkerName string) bool {
	if group := d.getGroup(selector); group != nil {
		for i := range group.checkers {
			if group.checkers[i].name == checkerName {
				// When we're removing a checker we should decrease the appropriate
				// reference counters of the triggers it was using. If the reference
				// counter becomes 0, the dispatcher no longer runs reviews for
				// these triggers.
				for _, trigger := range group.checkers[i].triggers {
					if _, ok := group.triggerRefCounts[trigger]; ok {
						group.triggerRefCounts[trigger]--
					}
				}
				group.checkers = append(group.checkers[:i], group.checkers[i+1:]...)
				if len(group.checkers) == 0 {
					delete(d.groups, selector)
				}
				return true
			}
		}
	}
	return false
}

// Returns the metadata of all registered configuration checkers for a given
// daemon. Metadata is a set of values describing the checker, i.e., name of
// the checker, list of triggers and selectors, and state of checker (enabled
// or disabled). The metadata is temporary objects releated to a specific
// daemon and cannot be used to execute the checker function. If the daemon ID
// equals to zero then it returns all registered checkers and global states.
func (d *dispatcherImpl) GetCheckersMetadata(daemonID int64, daemonName string) []*CheckerMetadata {
	isDaemonFetch := daemonID != 0

	checkers := make(map[string]*checker)
	selectors := make(map[string]DispatchGroupSelectors)

	var availableSelectors map[DispatchGroupSelector]bool
	if isDaemonFetch {
		groupSelectors := getDispatchGroupSelectors(daemonName)
		availableSelectors = make(map[DispatchGroupSelector]bool, len(groupSelectors))
		for _, selector := range groupSelectors {
			availableSelectors[selector] = true
		}
	}

	for selector, group := range d.groups {
		if isDaemonFetch {
			if _, ok := availableSelectors[selector]; !ok {
				continue
			}
		}

		for _, checker := range group.checkers {
			if _, ok := selectors[checker.name]; !ok {
				selectors[checker.name] = DispatchGroupSelectors{}
			}
			checkers[checker.name] = checker
			selectors[checker.name] = append(selectors[checker.name], selector)
		}
	}

	metadata := make([]*CheckerMetadata, len(checkers))
	for _, checker := range checkers {
		isEnabled := d.checkerController.IsCheckerEnabledForDaemon(daemonID, checker.name)
		m := newCheckerMetadataFromChecker(checker, selectors[checker.name], isEnabled)
		metadata = append(metadata, m)
	}

	return metadata
}

// Returns dispatcher's signature. The signature is a hash function output
// which depends on the registered checkers and dispatch groups. Comparing
// this signature with signatures stored in the database for already performed
// reviews is useful to determine whether new reviews are required. The
// signature changes whenever new checkers are registered. It does not
// change when an implementation of any existing checker has been modified.
// In this case, bump up the enforceDispatchSeq constant value to enforce
// generation of a new signature and new config reviews.
func (d *dispatcherImpl) GetSignature() string {
	return storkutil.Fnv128(fmt.Sprintf("%d:%+v", d.enforceSeq, d.groups))
}

// Starts the dispatcher by launching the worker goroutine receiving
// config reviews and populating them into the database.
func (d *dispatcherImpl) Start() {
	log.Info("starting the configuration review dispatcher")
	d.shutdownWg.Add(1)
	go d.awaitReports()
}

// Stops the dispatcher gracefully. When there are any ongoing reviews,
// this function blocks until all reviews are completed.
func (d *dispatcherImpl) Shutdown() {
	log.Info("stopping the configuration review dispatcher")
	d.cancelDispatch()
	d.shutdownWg.Wait()
	log.Info("stopped the configuration review dispatcher")
}

// Begins a new review for a daemon. If the callback function is not
// nil, the callback is invoked when the review is completed. The
// returned boolean value indicates whether or not the review has
// been scheduled. It is not scheduled when there is another review
// for the daemon already in progress, there are no checkers activated
// for the given trigger or the trigger is set to "internalRun".
func (d *dispatcherImpl) BeginReview(daemon *dbmodel.Daemon, trigger Trigger, callback CallbackFunc) bool {
	if trigger == internalRun {
		return false
	}
	return d.beginReview(daemon, trigger, callback)
}

// Checks if the review for the specified daemon is in progress.
func (d *dispatcherImpl) ReviewInProgress(daemonID int64) bool {
	d.mutex.RLock()
	defer d.mutex.RUnlock()
	inProgress, ok := d.state[daemonID]
	return ok && inProgress
}

// Registers default checkers in this package. When new checker is
// implemented it should be included in this function.
func RegisterDefaultCheckers(dispatcher Dispatcher) {
	dispatcher.RegisterChecker(KeaDHCPDaemon, "stat_cmds_presence", GetDefaultTriggers(), statCmdsPresence)
	dispatcher.RegisterChecker(KeaDHCPDaemon, "host_cmds_presence", GetDefaultTriggers(), hostCmdsPresence)
	dispatcher.RegisterChecker(KeaDHCPDaemon, "dispensable_shared_network", GetDefaultTriggers(), sharedNetworkDispensable)
	dispatcher.RegisterChecker(KeaDHCPDaemon, "dispensable_subnet", ExtendDefaultTriggers(DBHostsModified), subnetDispensable)
	dispatcher.RegisterChecker(KeaDHCPDaemon, "out_of_pool_reservation", ExtendDefaultTriggers(DBHostsModified), reservationsOutOfPool)
	dispatcher.RegisterChecker(KeaDHCPDaemon, "overlapping_subnet", GetDefaultTriggers(), subnetsOverlapping)
	dispatcher.RegisterChecker(KeaDHCPDaemon, "canonical_prefix", GetDefaultTriggers(), canonicalPrefixes)
}
