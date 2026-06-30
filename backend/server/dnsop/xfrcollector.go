package dnsop

import (
	"context"
	"sync"
	"time"

	"github.com/go-pg/pg/v10"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	agentcomm "isc.org/stork/server/agentcomm"
	dbmodel "isc.org/stork/server/database/model"
)

// xfrCollector maintains streaming communication with a single agent and collects
// the zone transfer states the agent reports. The received zone transfer states
// are inserted into the the Stork database. If the connection with the agent fails,
// the collector tries to re-establish the connection. It uses a backoff mechanism
// to avoid overwhelming the agent with the requests during a temporary outage.
type xfrCollector struct {
	// Common database connection.
	db *pg.DB
	// Common connection pool to the agents.
	agents agentcomm.ConnectedAgents
	// The BIND 9 daemon instance to communicate with.
	daemon *dbmodel.Daemon
	// The cancellation function used to stop the goroutine that collects the zone transfer states.
	cancel context.CancelFunc
	// The channel used by the stop function to block during context cancellation.
	stopChan chan struct{}
	// The mutex to protect the collector state from concurrent access.
	mutex sync.Mutex
	// The backoff factor used to calculate the backoff duration on re-connect.
	// The factor is set to 1 second by default. It is multiplied by power of two
	// to calculate the backoff duration. So, initial backoff is 1s, then 2s,
	// then 4s, etc. The factor can be decreased in the unit tests to shorten
	// blocking time.
	backoffFactor time.Duration
}

// Instantiates a new collector instance. The owner is typically the dnsop.Manager
// providing the common database and the connected agents instances. The daemon
// points to the BIND 9 daemon instance to communicate with.
func newXFRCollector(owner ManagerAccessors, daemon *dbmodel.Daemon) *xfrCollector {
	return &xfrCollector{
		db:            owner.GetDB(),
		agents:        owner.GetConnectedAgents(),
		daemon:        daemon,
		stopChan:      nil,
		backoffFactor: 1 * time.Second,
	}
}

// The main goroutine implementation that receives the zone transfer states over
// stream. It is called internally by the start function. In case of an error, it
// tries to re-connect to the agent using the backoff mechanism. If the connection
// ends without an error, the function exits, as it indicates that the agent has
// no more data to return, or the context was cancelled.
func (xfrCollector *xfrCollector) collect(ctx context.Context) {
	backoff := xfrCollector.backoffFactor
	for {
		streamErred := false
		for xfr, err := range xfrCollector.agents.ReceiveZoneTransfers(ctx, xfrCollector.daemon, true) {
			if err != nil {
				var agentTrackingDisabledError *agentcomm.ZoneTransferTrackingDisabledOnAgentError
				switch {
				case errors.As(err, &agentTrackingDisabledError):
					// Zone transfer tracking is disabled on the agent. There is nothing to do here.
					log.Info(agentTrackingDisabledError.Error())
					return
				default:
					// Some other error.
					log.WithError(err).Error("Failed to receive zone transfer state from the agent")
					streamErred = true
				}
				break
			}
			// The connection was successfully established. Let's restart the backoff.
			backoff = xfrCollector.backoffFactor
			err = dbmodel.AddOrUpdateZoneTransferState(xfrCollector.db, &dbmodel.ZoneTransferState{
				DaemonID:       xfrCollector.daemon.ID,
				ViewName:       xfr.ViewName,
				ZoneName:       xfr.ZoneName,
				Serial:         xfr.Serial,
				Client:         xfr.Client,
				Server:         xfr.Server,
				MessagesCount:  xfr.MessagesCount,
				RecordsCount:   xfr.RecordsCount,
				BytesCount:     xfr.BytesCount,
				Duration:       xfr.Duration,
				Status:         xfr.Status,
				StartTime:      xfr.StartTime,
				CompletionTime: xfr.CompletionTime,
				Message:        xfr.Message,
			})
			if err != nil {
				var pgErr pg.Error
				if errors.As(err, &pgErr) && pgErr.Field('C') == "23503" && pgErr.Field('n') == "zone_transfer_state_daemon_id_fkey" {
					// Handle foreign key violation error. It indicates that the daemon with this ID no longer exists in the database.
					log.Warnf("The daemon with ID %d have been removed; stopping zone transfer monitoring for that daemon", xfrCollector.daemon.ID)
					return
				}
				log.WithError(err).Error("Failed to add zone transfer state into the database")
			}
		}
		if !streamErred {
			// If the stream ended cleanly, there is no reason to reconnect.
			return
		}
		// Wait for the backoff duration or until the context is cancelled (whichever happens first).
		select {
		case <-ctx.Done():
			return
		case <-time.After(backoff):
			// Increase the backoff duration for the possible next attempt.
			backoff = min(backoff*2, 30*xfrCollector.backoffFactor)
		}
	}
}

// Starts the collector in a goroutine. If the collector is already started, it
// is no-op.
func (xfrCollector *xfrCollector) start() {
	xfrCollector.mutex.Lock()
	defer xfrCollector.mutex.Unlock()
	if xfrCollector.stopChan != nil {
		// The collector is already started.
		return
	}
	// This channel will be used in the stop function to block until the
	// collector is fully stopped.
	xfrCollector.stopChan = make(chan struct{})
	ctx, cancel := context.WithCancel(context.Background())
	xfrCollector.cancel = cancel
	go func() {
		defer func() {
			// Release the states and unblock the stop function by closing
			// the channel.
			xfrCollector.mutex.Lock()
			defer xfrCollector.mutex.Unlock()
			xfrCollector.cancel = nil
			close(xfrCollector.stopChan)
			xfrCollector.stopChan = nil
		}()
		xfrCollector.collect(ctx)
	}()
}

// Stops the collector. If the collector is not started, it is no-op.
// It is a blocking call waiting until the collector is fully stopped.
func (xfrCollector *xfrCollector) stop() {
	xfrCollector.mutex.Lock()
	cancel := xfrCollector.cancel
	stopChan := xfrCollector.stopChan
	xfrCollector.mutex.Unlock()
	if cancel != nil && stopChan != nil {
		cancel()
		<-stopChan
		return
	}
}

// Checks if the collector has been started.
func (xfrCollector *xfrCollector) isActive() bool {
	xfrCollector.mutex.Lock()
	defer xfrCollector.mutex.Unlock()
	return xfrCollector.stopChan != nil
}
