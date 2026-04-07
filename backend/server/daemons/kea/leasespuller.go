package kea

import (
	"context"
	"maps"
	"net/netip"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/go-pg/pg/v10"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"isc.org/stork/server/agentcomm"
	storkutil "isc.org/stork/util"

	"isc.org/stork/datamodel/daemonname"
	dbmodel "isc.org/stork/server/database/model"
)

// Leases puller is responsible for fetching lease data from Kea via the agents.
type LeasesPuller struct {
	*agentcomm.PeriodicPuller
	db *pg.DB
}

// Create a LeasesPuller object that, in the background, pulls lease records
// from Kea. The retreived records are added to the database.
func NewLeasesPuller(db *pg.DB, agents agentcomm.ConnectedAgents) (*LeasesPuller, error) {
	puller := LeasesPuller{
		nil,
		db,
	}
	periodicPuller, err := agentcomm.NewPeriodicPuller(db, agents, "Kea Leases puller", "kea_leases_puller_interval", puller.pullLeases)
	if err != nil {
		return nil, err
	}
	puller.PeriodicPuller = periodicPuller

	return &puller, nil
}

// Shut down the LeasesPuller. It ends the goroutine that pulls lease records.
func (puller *LeasesPuller) Shutdown() {
	puller.PeriodicPuller.Shutdown()
}

// A unique key for identifying Keas that are talking to the same database or writing to the same leasefile.  In order to use this effectively, EITHER:
//   - set `daemonName`, `machine` and `leasefilePath`, leaving all other fields
//     at the zero value;
//   - set `daemonName`, `dbHost`, `dbPort`, and `dbName`, leaving all other
//     fields at the zero value;
//   - set `daemonName`, `machine`, `dbHost`, `dbPort`, and `dbName`, leaving
//     all other fields set at the zero value (only if dbHost is
//     localhost-like); or
//   - set `daemonName` and `unique` to a non-zero unique value
//
// If two daemons with the same name (dhcpv4, dhcpv6, d2, ca, bind9, pdns) are
// looking at the same leasefilePath on the same machine, they are using the
// same lease database. If two daemons with the same name are looking at the
// same database host, port, and database name, then they are using the same
// lease database. (If a database host looks localhost-like, include the machine
// ID to catch daemons running on different machines both pointed at different
// RDMBS installations on localhost.) Changing any one of those values (very
// likely) means that they are pointed at different databases. `unique` is
// provided as an escape hatch to deal gracefully with uncommon Kea
// configurations (persist=false, any future enhancement to add a new lease
// database type).
//
// Known edge cases where this fails:
//   - Using nftables/iptables to redirect two external ports to the same RDBMS
//     (incorrectly sees them as different)
//   - Using CNAMEs to point two hosts at the same IP (incorrectly sees them as
//     different)
//   - Using multiple NICs or multicast tricks to point two IPs at the same
//     computer (incorrectly sees them as different)
//   - Mounting a filesystem on multiple machines using SMB or NFS (incorrectly
//     sees them as different)
//   - Running an agent on a host and having it monitor multiple Keas writing
//     leasefiles in containers (incorrectly sees them as the same)
type leaseDBUniqueKey struct {
	daemonName    daemonname.Name
	machine       int64
	leasefilePath string
	dbHost        string
	dbName        string
	unique        int
}

// Determine whether the provided host is likely to be localhost, without doing
// any DNS lookups.
func checkIsLocalhost(host string) bool {
	var hostAddr netip.Addr
	hostAddrPort, err := netip.ParseAddrPort(host)
	if err != nil {
		hostAddr, err = netip.ParseAddr(host)
		if err != nil {
			lowercase := strings.ToLower(host)
			return lowercase == "localhost" || strings.HasPrefix(lowercase, "localhost:")
		}
	} else {
		hostAddr = hostAddrPort.Addr()
	}
	return hostAddr.IsLoopback()
}

// Filter a list of daemons down to only daemons which use different databases.
// The filtering is done deterministically, so that repeated invocations of this
// function over time will choose the same daemon from a set of duplicates each
// time. (This is done by ID, choosing the lowest ID from the set.)
//
// onlyMemfile is an additional filter to help me during development because the
// agent doesn't support any database other than memfile yet.
//
// Precondition: the daemons slice is sorted by ID ascending.
func filterDaemons(daemons []dbmodel.Daemon, onlyMemfile bool) []*dbmodel.Daemon {
	// Take only the first daemon which uses each unique database.
	uniqueCtr := 1
	selectedDaemons := map[leaseDBUniqueKey]*dbmodel.Daemon{}
	for _, daemon := range daemons {
		if daemon.KeaDaemon == nil || daemon.KeaDaemon.Config == nil {
			log.WithField("daemon_id", daemon.ID).Debug("Ignoring non-Kea or configless daemon")
			continue
		}
		databases := daemon.KeaDaemon.Config.GetAllDatabases()
		if databases.Lease == nil {
			log.WithField("daemon_id", daemon.ID).Debug("Ignoring daemon with no lease database")
			continue
		}
		key := leaseDBUniqueKey{
			daemonName: daemon.Name,
		}
		switch {
		case databases.Lease.Type == "memfile" && ((databases.Lease.Persist != nil && *databases.Lease.Persist) || (databases.Lease.Persist == nil)):
			// TODO: get full memfile path from status-get response
			key.leasefilePath = databases.Lease.Path
			key.machine = daemon.MachineID
		case databases.Lease.Type == "mysql" || databases.Lease.Type == "postgresql":
			if onlyMemfile {
				continue
			}
			if checkIsLocalhost(databases.Lease.Host) {
				key.machine = daemon.MachineID
			}
			key.dbHost = databases.Lease.Host
			key.dbName = databases.Lease.Name
			// TODO: add this to the parsed part of the config.
			// key.dbPort = databases.Lease.Port
		default:
			key.unique = uniqueCtr
			uniqueCtr++
		}
		_, ok := selectedDaemons[key]
		if ok {
			// There's already a daemon matching this key in the map, so don't
			// overwrite it.
			continue
		}
		selectedDaemons[key] = &daemon
	}

	return slices.Collect(maps.Values(selectedDaemons))
}

// Pull leases from all monitored Kea daemons and store them in the database.
func (puller *LeasesPuller) pullLeases() error {
	log.Debug("beginning lease pull")
	beginTimestamp := time.Now()
	// Get a list of all of the DHCP daemons from the database. This function
	// must continue to return the daemons sorted by ID ascending, otherwise the
	// same sort needs to be applied in `filterDaemons()`.
	daemons, err := dbmodel.GetDHCPDaemons(puller.db)
	if err != nil {
		return err
	}

	selectedDaemons := filterDaemons(daemons, true)
	selectedDaemonsCount := len(selectedDaemons)
	log.Debug("lease: daemons filtered")

	// Get lease records from each daemon.
	var wg sync.WaitGroup
	errorPipe := make(chan error, selectedDaemonsCount)
	for _, daemon := range selectedDaemons {
		wg.Go(func() {
			err := puller.getLeasesFromDaemon(daemon)
			if err != nil {
				log.WithError(err).Warnf("Could not retrieve leases from daemon %d", daemon.ID)
				errorPipe <- err
			} else {
				errorPipe <- nil
			}
		})
	}
	log.Debug("lease: waiting for pull")
	wg.Wait()
	close(errorPipe)
	log.Debug("lease: wait complete; processing results")
	var errors []error
	daemonsOkCnt := 0
	for err := range errorPipe {
		log.Debug("lease: ranging over errorPipe")
		if err == nil {
			daemonsOkCnt++
		} else {
			errors = append(errors, err)
		}
	}
	log.Debug("lease: loop finished")
	elapsed := time.Since(beginTimestamp)
	log.WithField("successful_daemons", selectedDaemonsCount-len(errors)).
		WithField("total_daemons", selectedDaemonsCount).
		WithField("duration", elapsed).
		Info("completed lease pulling")
	return storkutil.CombineErrors("errors occurred while trying to pull leases from one or more daemons", errors)
}

func (puller *LeasesPuller) getLeasesFromDaemon(daemon *dbmodel.Daemon) error {
	if daemon.KeaDaemon == nil || !daemon.Active {
		log.WithField("daemon_id", daemon.ID).Debug("Skipping daemon because it is not Kea or it is inactive")
		return nil
	}

	if !daemon.Name.IsDHCP() {
		log.WithField("daemon_id", daemon.ID).Debug("Skipping daemon because it is not a DHCPD")
		return nil
	}

	ctx := context.Background()
	// TODO: review performance of this code to see if batching would help
	return puller.db.RunInTransaction(ctx, func(tx *pg.Tx) error {
		var maxCLTT uint64
		err := tx.Model((*dbmodel.Lease)(nil)).
			ColumnExpr("MAX(?)", pg.Ident("cltt")).
			Where("daemon_id = ?", daemon.ID).
			Select(&maxCLTT)
		if err != nil {
			log.WithError(err).WithField("daemon_id", daemon.ID).Error("Failed to fetch last CLTT from database for daemon")
			maxCLTT = 0
		}
		for response, err := range puller.Agents.ReceiveKeaLeases(ctx, daemon, maxCLTT) {
			switch {
			case err != nil:
				return err
			case response == nil:
				return errors.New("unexpected nil in response stream of Kea leases")
			default:
				// Non-error response.
				// TODO: get real lease ID
				err := dbmodel.AddLease(tx, dbmodel.LeaseFromGRPC(response.Lease, daemon.ID, 0))
				if err != nil {
					return err
				}
			}
		}
		return nil
	})
}
