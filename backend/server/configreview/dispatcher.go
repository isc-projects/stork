package configreview

import (
	"context"
	"sync"
	"time"

	pkgerrors "github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	dbops "isc.org/stork/server/database"
	dbmodel "isc.org/stork/server/database/model"
)

// Callback function invoked when configuration review is completed
// for a daemon.
type CallbackFunc func(int64, error)

// A wrapper around the config report returned by a producer. It
// supplies additional information to the report (i.e., name of
// the producer that generated the report).
type taggedReport struct {
	producerName string
	report       *report
}

// Review context is valid throughout a review of a daemon configuration.
// It holds information about the daemons which configurations are
// reviewed, and the review results output by the review report
// producers. Producers can modify the context information (e.g. fetch
// additional daemon configurations as necessary). The following
// fields are available in the context:
// - subjectDaemon: a daemon for which review is conducted
// - refDaemons: daemons fetched by the producers during the review
// (e.g. daemons which configurations are associated with the subject
// daemon configuration),
// - reports: configuration reports produced so far,
// - callback: user callback to invoke after the review,
// - internal: boolean flag indicating if the configuration review
// was triggered internally by the dispatcher or by an external call.
type reviewContext struct {
	subjectDaemon *dbmodel.Daemon
	refDaemons    []*dbmodel.Daemon
	reports       []taggedReport
	callback      CallbackFunc
	internal      bool
}

// Creates new review context instance.
func newReviewContext() *reviewContext {
	ctx := &reviewContext{}
	return ctx
}

// Dispatch group selector is used to segregate different configuration
// review producers by daemon types.
type DispatchGroupSelector int

// Enumeration listing supported configuration review producer groups.
// For instance, producers belonging to the KeaDHCPDaemon group are
// used for reviewing DHCPv4 and DHCPv6 servers configurations. The
// producers belonging to the KeaDaemon group are used to review
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

// Returns group selectors for selecting registered producers appropriate
// for the specified daemon name. For example, it returns EachDaemon,
// KeaDaemon, KeaDHCPDaemon and KeaDHCPv4Daemon selector for the "dhcp4"
// daemon. The corresponding producers will be used to review the
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
	return []DispatchGroupSelector{}
}

// Dispatch group is a group of producers registered for the particular
// dispatch group selector, e.g. for the KeaDHCPv4Daemon. Typically,
// producers from multiple dispatch groups are used for reviewing
// a single configuration.
type dispatchGroup struct {
	producers []*producer
}

// The dispatcher coordinates configuration reviews of all daemons.
// The dispatcher maintains a list of configuration review producers.
// They are segregated into dispatch groups invoked for different daemon
// types. Producers are registered and associated with the dispatch
// groups by their implementers, and the registration takes place before
// the dispatcher start. The producers perform specific checks on the
// selected part of the configuration and generate reports about issues.
// The configuration review can end with a list of reports or an empty
// list when no issues are found. A caller schedules a review by calling
// the BeginReview function. It schedules the review and returns
// immediately. The dispatcher runs the configuration through the
// registered producers in the background. The dispatcher inserts the
// reports into the database when the review finishes, replacing any
// existing reports for the daemon. More advanced producers can look
// into more than one daemon's configuration (e.g., to verify the
// consistency of the HA configuration between two partners).
type Dispatcher struct {
	// Database instance where configuration reports are stored.
	db *dbops.PgDB
	// Config review dispatch groups containing producers segregated
	// into groups by daemon types.
	groups map[DispatchGroupSelector]*dispatchGroup
	// Wait group used to gracefully stop the dispatcher when the server
	// is shutdown. It waits for the remaining work to complete.
	wg *sync.WaitGroup
	// Wait group used to synchronize the ongoing reviews.
	wg2 *sync.WaitGroup
	// Dispatcher main mutex.
	mutex *sync.Mutex
	// Channel for passing ready review reports to the worker
	// goroutine populating the reports into the database.
	reviewDoneChan chan *reviewContext
	// Context used for cancelling the worker goroutine when the
	// dispatcher is stopped.
	dispatchCtx context.Context
	// Function cancelling the worker goroutine.
	cancelDispatch context.CancelFunc
	// A map holding information about currently scheduled reviews.
	state map[int64]bool
}

// Creates new context instance when a review is scheduled.
func (d *Dispatcher) newContext() *reviewContext {
	ctx := newReviewContext()
	return ctx
}

// A worker function receiving ready configuration reports and populating
// them into the database.
func (d *Dispatcher) awaitReports() {
	defer d.wg.Done()
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
			allReviewsDone := new(bool)
			*allReviewsDone = false
			mutex := &sync.Mutex{}
			wg := &sync.WaitGroup{}
			wg.Add(1)

			// Launch a goroutine waiting for the outstanding reviews. It needs
			// a new goroutine because in the current goroutine we need to call
			// a blocking d.wg2.Wait() to wait for all outstanding reviews.
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
						done := false
						mutex.Lock()
						done = *allReviewsDone
						mutex.Unlock()
						if done {
							return
						}
						time.Sleep(50 * time.Millisecond)
					}
				}
			}()
			// Wait for the outstanding reviews to complete.
			d.wg2.Wait()

			// All reviews done. Let's signal it to the goroutine.
			mutex.Lock()
			*allReviewsDone = true
			mutex.Unlock()

			// Wait for the goroutine to finish.
			wg.Wait()

			return
		}
	}
}

// Goroutine performing configuration review for a daemon.
func (d *Dispatcher) runForDaemon(daemon *dbmodel.Daemon, internal bool, callback CallbackFunc) {
	defer d.wg2.Done()

	ctx := d.newContext()
	ctx.subjectDaemon = daemon
	ctx.callback = callback
	ctx.internal = internal

	dispatchGroupSelectors := getDispatchGroupSelectors(daemon.Name)

	for _, selector := range dispatchGroupSelectors {
		if group, ok := d.groups[selector]; ok {
			for i := range group.producers {
				report, err := group.producers[i].produceFn(ctx)
				if err != nil {
					log.Errorf("malformed report created by the config review producer %s: %+v",
						group.producers[i].name, err)
				}
				if report != nil {
					ctx.reports = append(ctx.reports, taggedReport{
						producerName: group.producers[i].name,
						report:       report,
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
func (d *Dispatcher) beginReview(daemon *dbmodel.Daemon, internal bool, callback CallbackFunc) bool {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	// Check if another review for this daemon has been already scheduled. Do not
	// schedule new review if one is already in progress.
	inProgress, ok := d.state[daemon.ID]
	if !ok || !inProgress {
		d.state[daemon.ID] = true
		d.wg2.Add(1)
		// Run the review in the background.
		go d.runForDaemon(daemon, internal, callback)
		return true
	}
	// Another review in progress - bail.
	return false
}

// Inserts new config review reports into the database.
func (d *Dispatcher) populateReports(ctx *reviewContext) error {
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

	// The following calls serve two purposes. Firstly, they lock the daeamons
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

	// Check if the configuration of any of the deamons has changed.
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
		for _, id := range r.report.refDaemons {
			assoc = append(assoc, &dbmodel.Daemon{
				ID: id,
			})
		}
		cr := &dbmodel.ConfigReport{
			ProducerName: r.producerName,
			Contents:     r.report.issue,
			DaemonID:     ctx.subjectDaemon.ID,
			RefDaemons:   assoc,
		}
		err = dbmodel.AddConfigReport(tx, cr)
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
func NewDispatcher(db *dbops.PgDB) *Dispatcher {
	ctx, cancel := context.WithCancel(context.Background())
	dispatcher := &Dispatcher{
		db:             db,
		groups:         make(map[DispatchGroupSelector]*dispatchGroup),
		wg:             &sync.WaitGroup{},
		wg2:            &sync.WaitGroup{},
		mutex:          &sync.Mutex{},
		reviewDoneChan: make(chan *reviewContext),
		dispatchCtx:    ctx,
		cancelDispatch: cancel,
		state:          make(map[int64]bool),
	}
	return dispatcher
}

// Registers new producer. A producer implements an algorithm to verify
// a single configuration piece (or aspect) and output a suitable report
// if it finds issues. It should return nil when no issues were found.
// Each producer is assigned a unique name so it will be possible to
// list available producers and/or selectively disable them.
func (d *Dispatcher) RegisterProducer(selector DispatchGroupSelector, producerName string, produceFn func(*reviewContext) (*report, error)) {
	if _, ok := d.groups[selector]; !ok {
		d.groups[selector] = &dispatchGroup{}
		d.groups[selector].producers = []*producer{}
	}
	d.groups[selector].producers = append(d.groups[selector].producers,
		&producer{
			name:      producerName,
			produceFn: produceFn,
		},
	)
}

// Registers default producers in this package. When new producer is
// implemented it should be included in this function.
func (d *Dispatcher) RegisterDefaultProducers() {
	d.RegisterProducer(KeaDHCPDaemon, "stat_cmds_presence", statCmdsPresence)
}

// Starts the dispatcher by launching the worker goroutine receiving
// config reviews and populating them into the database.
func (d *Dispatcher) Start() {
	d.wg.Add(1)
	go d.awaitReports()
}

// Stops the dispatcher gracefully. When there are any ongoing reviews,
// this function blocks until all reviews are completed.
func (d *Dispatcher) Shutdown() {
	d.cancelDispatch()
	d.wg.Wait()
}

// Begins a new review for a daemon. If the callback function is not
// nil, the callback is invoked when the review is completed. The
// returned boolean value indicates whether or not the review has
// been scheduled. It is not scheduled when there is another review
// for the daemon already in progress.
func (d *Dispatcher) BeginReview(daemon *dbmodel.Daemon, callback CallbackFunc) bool {
	return d.beginReview(daemon, false, callback)
}
