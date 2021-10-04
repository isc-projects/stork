package configreview

import (
	"context"
	"sync"
	"time"

	dbops "isc.org/stork/server/database"
	dbmodel "isc.org/stork/server/database/model"
)

type reviewContext struct {
	daemons []*dbmodel.Daemon
	reports []*report
}

func newReviewContext() *reviewContext {
	ctx := &reviewContext{}
	return ctx
}

type DispatchGroupSelector int

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

type dispatchGroup struct {
	producers []*producer
}

type Dispatcher struct {
	db             *dbops.PgDB
	groups         map[DispatchGroupSelector]*dispatchGroup
	wg             *sync.WaitGroup
	wg2            *sync.WaitGroup
	mutex          *sync.Mutex
	reviewDoneChan chan *reviewContext
	dispatchCtx    context.Context
	cancelDispatch context.CancelFunc
}

func (d *Dispatcher) newContext() *reviewContext {
	ctx := newReviewContext()
	return ctx
}

func (d *Dispatcher) awaitReports() {
	defer d.wg.Done()
	for {
		select {
		case ctx := <-d.reviewDoneChan:
			if ctx != nil {
			}

		case <-d.dispatchCtx.Done():
			allReviewsDone := new(bool)
			*allReviewsDone = false
			mutex := &sync.Mutex{}
			wg := &sync.WaitGroup{}
			wg.Add(1)

			go func() {
				defer wg.Done()
				for {
					select {
					case <-d.reviewDoneChan:
					default:
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
			d.wg2.Wait()

			mutex.Lock()
			*allReviewsDone = true
			mutex.Unlock()

			wg.Wait()

			return
		}
	}
}

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
	}
	return dispatcher
}

func (d *Dispatcher) Start() {
	d.wg.Add(1)
	go d.awaitReports()
}

func (d *Dispatcher) Stop() {
	d.cancelDispatch()
	d.wg.Wait()
}

func (d *Dispatcher) RegisterProducer(selector DispatchGroupSelector, producerName string, produceFn func(*reviewContext) *report) {
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

func (d *Dispatcher) LoadDefaultProducers() {
	d.RegisterProducer(KeaDHCPDaemon, "stat_cmds_presence", statCmdsPresence)
}

func (d *Dispatcher) runForDaemon(daemon *dbmodel.Daemon) {
	defer d.wg2.Done()

	ctx := d.newContext()
	ctx.daemons = append(ctx.daemons, daemon)

	dispatchGroupSelectors := getDispatchGroupSelectors(daemon.Name)

	for _, selector := range dispatchGroupSelectors {
		if group, ok := d.groups[selector]; ok {
			for i := range group.producers {
				select {
				case <-d.dispatchCtx.Done():
					return
				default:
					ctx.reports = append(ctx.reports, group.producers[i].produceFn(ctx))
				}
			}
		}
	}
	d.reviewDoneChan <- ctx
}

func (d *Dispatcher) BeginForDaemon(daemon *dbmodel.Daemon) error {
	d.wg2.Add(1)
	go d.runForDaemon(daemon)
	return nil
}
