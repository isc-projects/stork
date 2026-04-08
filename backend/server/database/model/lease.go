package dbmodel

import (
	"context"
	"math"

	"github.com/go-pg/pg/v10"
	pkgerrors "github.com/pkg/errors"

	agentapi "isc.org/stork/api"
	keadata "isc.org/stork/daemondata/kea"
	dbops "isc.org/stork/server/database"
	storkutil "isc.org/stork/util"
)

// Extends basic Lease information with database specific information.
type Lease struct {
	ID int64

	keadata.Lease

	DaemonID int64
	Daemon   *Daemon `pg:"rel:has-one"`
	// Stork's subnet ID.
	SubnetID int64
	Subnet   *Subnet `pg:"rel:has-one"`
}

// Adds a new lease into the database within a transaction.
func addLease(tx *pg.Tx, lease *Lease) (err error) {
	// Add the subnet first.
	_, err = tx.Model(lease).Insert()
	if err != nil {
		err = pkgerrors.Wrapf(err, "problem inserting lease for %s to (mac:%s/duid:%s/clientid:%s)", lease.IPAddress, lease.HWAddress, lease.DUID, lease.ClientID)
		return err
	}
	return nil
}

// Adds a lease into the database. If `dbi` is a transaction, this function
// uses it as-is. If `dbi` is a DB, it makes a new transaction before adding
// the lease.
func AddLease(dbi dbops.DBI, lease *Lease) error {
	if db, ok := dbi.(*pg.DB); ok {
		return db.RunInTransaction(context.Background(), func(tx *pg.Tx) error {
			return addLease(tx, lease)
		})
	}
	return addLease(dbi.(*pg.Tx), lease)
}

// Get a lease by its ID.
func GetLeaseByID(dbi dbops.DBI, leaseID int64) (*Lease, error) {
	lease := &Lease{}
	err := dbi.Model(lease).
		Relation("Daemon").
		Relation("Subnet").
		Where("lease.id = ?", leaseID).
		Select()
	if err != nil {
		if pkgerrors.Is(err, pg.ErrNoRows) {
			return nil, nil
		}
		err = pkgerrors.Wrapf(err, "problem getting lease with ID %d", leaseID)
		return nil, err
	}
	return lease, err
}

// Create a model.Lease from the gRPC Lease structure.
func NewLeaseFromGRPC(grpc *agentapi.Lease, daemonID, subnetID int64) *Lease {
	if grpc == nil {
		return nil
	}
	if grpc.ValidLifetime > math.MaxUint32 {
		return nil
	}
	if grpc.PrefixLen > math.MaxUint8 {
		return nil
	}
	if grpc.Family != 4 && grpc.Family != 6 {
		return nil
	}
	ipv := storkutil.IPv4
	if grpc.Family == 6 {
		ipv = storkutil.IPv6
	}
	return &Lease{
		0,
		keadata.Lease{
			Family:        ipv,
			IPAddress:     grpc.IpAddress,
			HWAddress:     grpc.HwAddress,
			DUID:          grpc.Duid,
			CLTT:          grpc.Cltt,
			ValidLifetime: uint32(grpc.ValidLifetime),
			LocalSubnetID: grpc.SubnetID,
			State:         int(grpc.State),
			PrefixLength:  uint8(grpc.PrefixLen),
		},
		daemonID,
		nil,
		subnetID,
		nil,
	}
}
