package dbmodel

import (
	"context"
	"math"
	"strings"

	"github.com/go-pg/pg/v10"
	"github.com/go-pg/pg/v10/orm"
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
		if lease == nil {
			err = pkgerrors.Wrapf(err, "cannot insert nil lease into database")
		} else {
			err = pkgerrors.Wrapf(err, "problem inserting lease for %s to (mac:%s/duid:%s/clientid:%s)", lease.IPAddress, lease.HWAddress, lease.DUID, lease.ClientID)
		}
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

// Container for values filtering leases fetched by page.
//
// FilterText searches by IP, DUID, Client ID, hardware address, hostname, etc.
type LeasesByPageFilters struct {
	MachineID     *int64
	SubnetID      *int64
	DaemonID      *int64
	LocalSubnetID *int64
	FilterText    *string
}

// Sort field which may be used in GetLeasesByPage.
type LeaseSortField string

// Valid lease sort fields.
const (
	LeaseSortFieldSubnet        LeaseSortField = "subnet"
	LeaseSortFieldHWAddr        LeaseSortField = "hw_address"
	LeaseSortFieldIPAddr        LeaseSortField = "address"
	LeaseSortFieldClientID      LeaseSortField = "client_id"
	LeaseSortFieldDUID          LeaseSortField = "duid"
	LeaseSortFieldCLTT          LeaseSortField = "cltt"
	LeaseSortFieldValidLifetime LeaseSortField = "valid_lifetime"
)

// Fetches a collection of leases from the database.
// Returns an ordered subset of hosts and the total number of hosts, or an error.
//
// The offset and limit specify the beginning of the page and the maximum number
// of leases to show per page.
//
// sortField indicates the database column to use for sorting the data. When
// sortField is the zero value, the lease ID is used for sorting.
//
// sortDir specifies the direction for the sort. If SortDirAny is provided,
// results will be sorted in ascending order.
func GetLeasesByPage(dbi dbops.DBI, offset, limit int64, filters LeasesByPageFilters, sortField string, sortDir SortDirEnum) ([]Lease, int64, error) {
	leases := []Lease{}
	q := dbi.Model(&leases)

	// Convert friendly API field names to database column names.
	var dbSortField string
	switch LeaseSortField(sortField) {
	case LeaseSortFieldSubnet:
		dbSortField = "subnet.prefix"
	case LeaseSortFieldHWAddr:
		dbSortField = "hw_address"
	case LeaseSortFieldIPAddr:
		dbSortField = "ip_address"
	case LeaseSortFieldClientID:
		dbSortField = "client_id"
	case LeaseSortFieldDUID:
		dbSortField = "duid"
	case LeaseSortFieldCLTT:
		dbSortField = "cltt"
	case LeaseSortFieldValidLifetime:
		dbSortField = "valid_lifetime"
	default:
		dbSortField = sortField
	}
	orderExpr, distinctOnFields := prepareOrderAndDistinctExpr("lease", dbSortField, sortDir, nil)
	q = q.DistinctOn(distinctOnFields)

	if filters.MachineID != nil && *filters.MachineID != 0 {
		q = q.Join("JOIN daemon").JoinOn("lease.daemon_id = daemon.id")
		q = q.Where("daemon.machine_id = ?", *filters.MachineID)
	}

	if filters.DaemonID != nil && *filters.DaemonID != 0 {
		q = q.Where("lease.daemon_id = ?", *filters.DaemonID)
	}

	if filters.SubnetID != nil && *filters.SubnetID != 0 {
		q = q.Where("lease.subnet_id = ?", *filters.SubnetID)
	}

	if filters.LocalSubnetID != nil && *filters.LocalSubnetID != 0 {
		q = q.Where("lease.local_subnet_id = ?", *filters.LocalSubnetID)
	}

	if filters.FilterText != nil && len(*filters.FilterText) > 0 {
		colonlessFilterExpr := "%" + strings.ReplaceAll(*filters.FilterText, ":", "") + "%"
		regFilterExpr := "%" + *filters.FilterText + "%"
		q = q.WhereGroup(func(q *orm.Query) (*orm.Query, error) {
			q = q.WhereOr("text(r.ip_address) ILIKE ?", regFilterExpr).
				WhereOr("encode(duid, 'hex') ILIKE ?", colonlessFilterExpr).
				WhereOr("encode(hw_address, 'hex') ILIKE ?", colonlessFilterExpr).
				WhereOr("encode(client_id, 'hex') ILIKE ?", colonlessFilterExpr).
				WhereOr("hostname ILIKE ?", regFilterExpr)
			return q, nil
		})
	}

	q = q.
		Relation("Daemon").
		Relation("Subnet")

	q = q.OrderExpr(orderExpr)
	q = q.
		Offset(int(offset)).
		Limit(int(limit))

	total, err := q.SelectAndCount()
	if err != nil {
		if pkgerrors.Is(err, pg.ErrNoRows) {
			return nil, 0, nil
		}
		err = pkgerrors.Wrapf(err, "problem getting leases by page")
	}
	return leases, int64(total), err
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
