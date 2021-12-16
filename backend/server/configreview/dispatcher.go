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
// - internal: boolean flag indicating if the configuration review
// was triggered internally by the dispatcher or by an external call.
type ReviewContext struct {
	db            *dbops.PgDB
	subjectDaemon *dbmodel.Daemon
	refDaemons    []*dbmodel.Daemon
	reports       []taggedReport
	callback      CallbackFunc
	internal      bool
}

// Creates new review context instance.
func newReviewContext(db *dbops.PgDB, daemon *dbmodel.Daemon, internal bool, callback CallbackFunc) *ReviewContext {
	ctx := &ReviewContext{
		db:            db,
		subjectDaemon: daemon,
		callback:      callback,
		internal:      internal,
	}
	return ctx
}

// Dispatch group selector is used to segregate different configuration
// review checkers by daemon types.
type DispatchGroupSelector int

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
func getDispatchGroupSelectors(daemonName string) []DispatchGroupSelector {
	switch daemonName {
	case "dhcp4":
		return []DispatchGroupSelector{EachDaemon, KeaDaemon, KeaDHCPDaemon, KeaDHCPv4Daemon}
	case "dhcp6":
		return []DispatchGroupSelector{EachDaemon, KeaDaemon, KeaDHCPDaemon, KeaDHCPv6Daemon}
	case "ca":
		return []DispatchGroupSelector{EachDaemon, KeaDaemon, KeaCADaemon}
	case "d2":
		return []DispatchGroupSelector{EachDaemon, KeaDaemon, KeaD2Daemon}
	case "bind9":
		return []DispatchGroupSelector{EachDaemon, Bind9Daemon}
	}
	log.WithFields(log.Fields{
		"daemon_name": daemonName,
	}).Warn("Config review dispatcher was unable to recognize the daemon by name and assign any suitable dispatch groups. Please consult this issue with the ISC Stork Development Team.")

	return []DispatchGroupSelector{}
}

// Dispatch group is a group of checkers registered for the particular
// dispatch group selector, e.g. for the KeaDHCPv4Daemon. Typically,
// checkers from multiple dispatch groups are used for reviewing
// a single configuration.
type dispatchGroup struct {
	checkers []*checker
}

// Stringer implementation for a dispatchGroup. It lists checker names
// as a slice. It excludes checker function pointers making the output
// consistent across function runs. The dispatcher uses this function
// to create a signature.
func (d dispatchGroup) String() string {
	names := []string{}
	for _, checker := range d.checkers {
		names = append(names, checker.name)
	}
	return fmt.Sprintf("%+v", names)
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
}

// Dispatcher interface. The interface is used in the unit tests that
// require replacing the default implementation with a mock dispatcher.
type Dispatcher interface {
	RegisterChecker(selector DispatchGroupSelector, checkerName string, checkFn func(*ReviewContext) (*Report, error))
	UnregisterChecker(selector DispatchGroupSelector, checkerName string) bool
	GetSignature() string
	Start()
	Shutdown()
	BeginReview(daemon *dbmodel.Daemon, callback CallbackFunc) bool
	ReviewInProgress(daemonID int64) bool
}

// Creates new context instance when a review is scheduled. The daemon
// is a pointer to a daemon instance for which the review is
// performed. The internal flag indicates if the review has been
// initiated internally by the dispatcher. The callback is a
// callback function invoked after the review is completed.
func (d *dispatcherImpl) newContext(db *dbops.PgDB, daemon *dbmodel.Daemon, internal bool, callback CallbackFunc) *ReviewContext {
	ctx := newReviewContext(db, daemon, internal, callback)
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
					log.Errorf("problem with populating configuration review reports to the database for daemon %d: %+v",
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
							log.Errorf("problem with populating configuration review reports to the database for daemon %d: %+v",
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
func (d *dispatcherImpl) runForDaemon(daemon *dbmodel.Daemon, internal bool, callback CallbackFunc) {
	defer d.reviewWg.Done()

	ctx := d.newContext(d.db, daemon, internal, callback)

	dispatchGroupSelectors := getDispatchGroupSelectors(daemon.Name)

	for _, selector := range dispatchGroupSelectors {
		if group, ok := d.groups[selector]; ok {
			for i := range group.checkers {
				report, err := group.checkers[i].checkFn(ctx)
				if err != nil {
					log.Errorf("malformed report created by the config review checker %s: %+v",
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

// Internal function scheduling a new review. Comparing to the exported function, BeginReview,
// it accepts the additional "internal" parameter. It allows for marking the review as
// scheduled internally by the dispatcher. In that case the populateReports function
// takes slightly different path when it inserts new reports to the database. The
// BeginReview function internally calls the beginReview function with this flag set to
// false.
func (d *dispatcherImpl) beginReview(daemon *dbmodel.Daemon, internal bool, callback CallbackFunc) bool {
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
	go d.runForDaemon(daemon, internal, callback)
	return true
}

// Inserts new config review reports into the database.
func (d *dispatcherImpl) populateReports(ctx *ReviewContext) error {
	// Ensure that the state indicates that the review is no longer
	// in progress when this function returns.
	defer func() {
		d.mutex.Lock()
		defer d.mutex.Unlock()
		d.state[ctx.subjectDaemon.ID] = false
	}()

	daemons := append([]*dbmodel.Daemon{ctx.subjectDaemon}, ctx.refDaemons...)

	// Begin a new transaction for inserting the reports.
	tx, rollback, commit, err := dbops.Transaction(d.db)
	if err != nil {
		return err
	}
	defer rollback()

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
		return err
	}

	// Ensure that all daemons were in the database.
	if len(dbDaemons) != len(daemons) {
		return pkgerrors.New("some daemons with reviewed configuration are missing in the database")
	}

	// Check if the configuration of any of the daemons has changed.
	for _, dbDaemon := range dbDaemons {
		for _, daemon := range daemons {
			if daemon.ID == dbDaemon.ID {
				if daemon.KeaDaemon != nil && dbDaemon.KeaDaemon != nil &&
					daemon.KeaDaemon.ConfigHash != dbDaemon.KeaDaemon.ConfigHash {
					return pkgerrors.Errorf("Kea daemon %d configuration has changed since the config review began", daemon.ID)
				}
			}
		}
	}

	if !ctx.internal {
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
					return err
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
			return err
		}
	}

	// Add configuration review summary.
	// todo: add config review summary for BIND9. Currently we don't because
	// BIND9 does not include a config hash.
	if ctx.subjectDaemon.KeaDaemon != nil {
		configReview := &dbmodel.ConfigReview{
			ConfigHash: ctx.subjectDaemon.KeaDaemon.ConfigHash,
			Signature:  d.GetSignature(),
			DaemonID:   ctx.subjectDaemon.ID,
		}
		err = dbmodel.AddConfigReview(tx, configReview)
		if err != nil {
			return err
		}
	}

	err = commit()
	if err != nil {
		return err
	}

	if !ctx.internal {
		// If the review was scheduled externally, and we deleted configuration
		// reports for referenced daemons, we have to rebuild the reports for
		// those daemons. Let's schedule internal reviews for each of them.
		for i := range ctx.refDaemons {
			// Do not schedule the review for the subject daemon because we're
			// now doing its review.
			if ctx.refDaemons[i].ID != ctx.subjectDaemon.ID {
				_ = d.beginReview(ctx.refDaemons[i], true, ctx.callback)
			}
		}
	}

	return nil
}

// Creates new dispatcher instance.
func NewDispatcher(db *dbops.PgDB) Dispatcher {
	ctx, cancel := context.WithCancel(context.Background())
	dispatcher := &dispatcherImpl{
		db:             db,
		groups:         make(map[DispatchGroupSelector]*dispatchGroup),
		shutdownWg:     &sync.WaitGroup{},
		reviewWg:       &sync.WaitGroup{},
		mutex:          &sync.RWMutex{},
		reviewDoneChan: make(chan *ReviewContext),
		dispatchCtx:    ctx,
		cancelDispatch: cancel,
		state:          make(map[int64]bool),
		enforceSeq:     enforceDispatchSeq,
	}
	return dispatcher
}

// Registers new checker. A checker implements an algorithm to verify
// a single configuration piece (or aspect) and output a suitable report
// if it finds issues. It should return nil when no issues were found.
// Each checker is assigned a unique name so it will be possible to
// list available checkers and/or selectively disable them.
func (d *dispatcherImpl) RegisterChecker(selector DispatchGroupSelector, checkerName string, checkFn func(*ReviewContext) (*Report, error)) {
	if _, ok := d.groups[selector]; !ok {
		d.groups[selector] = &dispatchGroup{}
		d.groups[selector].checkers = []*checker{}
	}
	d.groups[selector].checkers = append(d.groups[selector].checkers,
		&checker{
			name:    checkerName,
			checkFn: checkFn,
		},
	)
}

// Unregisters a checker from a dispatch group. It returns a boolean
// value indicating if the matching checker was found and removed (if true).
func (d *dispatcherImpl) UnregisterChecker(selector DispatchGroupSelector, checkerName string) bool {
	if _, ok := d.groups[selector]; ok {
		for i := range d.groups[selector].checkers {
			if d.groups[selector].checkers[i].name == checkerName {
				newGroups := make([]*checker, 0)
				newGroups = append(newGroups, d.groups[selector].checkers[:i]...)
				newGroups = append(newGroups, d.groups[selector].checkers[i+1:]...)
				d.groups[selector].checkers = newGroups
				if len(d.groups[selector].checkers) == 0 {
					delete(d.groups, selector)
				}
				return true
			}
		}
	}
	return false
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
// for the daemon already in progress.
func (d *dispatcherImpl) BeginReview(daemon *dbmodel.Daemon, callback CallbackFunc) bool {
	log.WithFields(log.Fields{
		"daemon_id": daemon.ID,
		"name":      daemon.Name,
	}).Info("scheduling a new configuration review")
	return d.beginReview(daemon, false, callback)
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
	dispatcher.RegisterChecker(KeaDHCPDaemon, "stat_cmds_presence", statCmdsPresence)
	dispatcher.RegisterChecker(KeaDHCPDaemon, "host_cmds_presence", hostCmdsPresence)
	dispatcher.RegisterChecker(KeaDHCPDaemon, "shared_network_dispensable", sharedNetworkDispensable)
	dispatcher.RegisterChecker(KeaDHCPDaemon, "subnet_dispensable", subnetDispensable)
	dispatcher.RegisterChecker(KeaDHCPDaemon, "reservations_out_of_pool", reservationsOutOfPool)
}
