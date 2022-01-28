package dbmodel

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"github.com/go-pg/pg/v10"
	"github.com/go-pg/pg/v10/orm"
	pkgerrors "github.com/pkg/errors"
	dbops "isc.org/stork/server/database"
	storkutil "isc.org/stork/util"
)

// This structure reflects a row in the host_identifier table. It includes
// a value and type of the identifier used to match the client with a host. The
// following types are available: hw-address, duid, circuit-id, client-id
// and flex-id (same as those available in Kea).
type HostIdentifier struct {
	ID     int64
	Type   string
	Value  []byte
	HostID int64
}

// This structure reflects a row in the ip_reservation table. It represents
// a single IP address or prefix reservation associated with a selected host.
type IPReservation struct {
	ID      int64
	Address string
	HostID  int64
}

// Checks if reservation represents a prefix reservations.
func (r *IPReservation) IsPrefixReservation() bool {
	if !strings.Contains(r.Address, "/") {
		return false
	}
	// IPv4
	singleSuffix := "/32"
	if strings.Contains(r.Address, ":") {
		// IPv6
		singleSuffix = "/128"
	}
	// IPv6
	return !strings.HasSuffix(r.Address, singleSuffix)
}

// This structure reflects a row in the host table. The host may be associated
// with zero, one or multiple IP reservations. It may also be associated with
// one or more identifiers which are used for matching DHCP clients with the
// host.
type Host struct {
	ID        int64 `pg:",pk"`
	CreatedAt time.Time
	SubnetID  int64
	Subnet    *Subnet `pg:"rel:has-one"`

	Hostname string

	HostIdentifiers []HostIdentifier `pg:"rel:has-many"`
	IPReservations  []IPReservation  `pg:"rel:has-many"`

	LocalHosts []LocalHost `pg:"rel:has-many"`

	// This flag is used to indicate that some changes have been applied to
	// the Host instance locally and that these changes should be applied in
	// the database too. It also indicates that the new app should be
	// associated with the host upon the call to the CommitSubnetHostsIntoDB.
	UpdateOnCommit bool `pg:"-"`
}

// This structure links a host entry stored in the database with a daemon from
// which it has been retrieved. It provides M:N relationship between hosts
// and daemons.
type LocalHost struct {
	HostID     int64   `pg:",pk"`
	DaemonID   int64   `pg:",pk"`
	Daemon     *Daemon `pg:"rel:has-one"`
	Host       *Host   `pg:"rel:has-one"`
	DataSource string
}

// Associates a host with DHCP with host identifiers.
func addHostIdentifiers(tx *pg.Tx, host *Host) error {
	for i, id := range host.HostIdentifiers {
		identifier := id
		identifier.HostID = host.ID
		_, err := tx.Model(&identifier).
			OnConflict("(type, host_id) DO UPDATE").
			Set("value = EXCLUDED.value").
			Insert()
		if err != nil {
			err = pkgerrors.Wrapf(err, "problem with adding host identifier with type %s for host with id %d",
				identifier.Type, host.ID)
			return err
		}
		host.HostIdentifiers[i] = identifier
	}
	return nil
}

// Associates a host with IP reservations.
func addIPReservations(tx *pg.Tx, host *Host) error {
	for i, r := range host.IPReservations {
		reservation := r
		reservation.HostID = host.ID
		_, err := tx.Model(&reservation).
			OnConflict("DO NOTHING").
			Insert()
		if err != nil {
			err = pkgerrors.Wrapf(err, "problem with adding IP reservation %s for host with id %d",
				reservation.Address, host.ID)
			return err
		}
		host.IPReservations[i] = reservation
	}
	return nil
}

// Adds new host, its reservations and identifiers into the database in
// a transaction.
func addHost(tx *pg.Tx, host *Host) error {
	// Add the host and fetch its id.
	_, err := tx.Model(host).Insert()
	if err != nil {
		err = pkgerrors.Wrapf(err, "problem with adding new host")
		return err
	}
	// Associate the host with the given id with its identifiers.
	err = addHostIdentifiers(tx, host)
	if err != nil {
		return err
	}
	// Associate the host with the given id with its reservations.
	err = addIPReservations(tx, host)
	if err != nil {
		return err
	}
	return nil
}

// Adds new host, its reservations and identifiers into the database.
// It begins a new transaction when dbi has a *pg.DB type or uses an
// existing transaction when dbi has a *pg.Tx type.
func AddHost(dbi dbops.DBI, host *Host) error {
	if db, ok := dbi.(*pg.DB); ok {
		return db.RunInTransaction(context.Background(), func(tx *pg.Tx) error {
			return addHost(tx, host)
		})
	}
	return addHost(dbi.(*pg.Tx), host)
}

// Updates a host, its reservations and identifiers in the database
// in a transaction.
func updateHost(tx *pg.Tx, host *Host) error {
	// Collect updated identifiers.
	hostIDTypes := []string{}
	for _, hostID := range host.HostIdentifiers {
		hostIDTypes = append(hostIDTypes, hostID.Type)
	}
	q := tx.Model((*HostIdentifier)(nil)).
		Where("host_identifier.host_id = ?", host.ID)
	// If the new reservation has any host identifiers exclude them from
	// the deleted ones. Otherwise, delete all reservations belonging to
	// the old host version.
	if len(hostIDTypes) > 0 {
		q = q.Where("host_identifier.type NOT IN (?)", pg.In(hostIDTypes))
	}
	_, err := q.Delete()
	if err != nil {
		err = pkgerrors.Wrapf(err, "problem with deleting host identifiers for host %d", host.ID)
		return err
	}
	// Add or update host identifiers.
	err = addHostIdentifiers(tx, host)
	if err != nil {
		return pkgerrors.WithMessagef(err, "problem with updating host with id %d", host.ID)
	}

	// Collect updated identifiers.
	ipAddresses := []string{}
	for _, resrv := range host.IPReservations {
		ipAddresses = append(ipAddresses, resrv.Address)
	}
	q = tx.Model((*IPReservation)(nil)).
		Where("ip_reservation.host_id = ?", host.ID)
	// If the new reservation has some reserved IP addresses exclude them
	// from the deleted ones. Otherwise, delete all IP addresses belonging
	// to the old host version.
	if len(ipAddresses) > 0 {
		q = q.Where("ip_reservation.address NOT IN (?)", pg.In(ipAddresses))
	}
	_, err = q.Delete()

	if err != nil {
		err = pkgerrors.Wrapf(err, "problem with deleting IP reservations for host %d", host.ID)
		return err
	}
	// Add or update host reservations.
	err = addIPReservations(tx, host)
	if err != nil {
		return pkgerrors.WithMessagef(err, "problem with updating host with id %d", host.ID)
	}

	// Update the host information.
	result, err := tx.Model(host).WherePK().Update()
	if err != nil {
		err = pkgerrors.Wrapf(err, "problem with updating host with id %d", host.ID)
	} else if result.RowsAffected() <= 0 {
		err = pkgerrors.Wrapf(ErrNotExists, "host with id %d does not exist", host.ID)
	}
	return err
}

// Updates a host, its reservations and identifiers in the database.
func UpdateHost(dbi pg.DBI, host *Host) error {
	if db, ok := dbi.(*pg.DB); ok {
		return db.RunInTransaction(context.Background(), func(tx *pg.Tx) error {
			return updateHost(tx, host)
		})
	}
	return updateHost(dbi.(*pg.Tx), host)
}

// Fetch the host by ID.
func GetHost(dbi dbops.DBI, hostID int64) (*Host, error) {
	host := &Host{}
	err := dbi.Model(host).
		Relation("HostIdentifiers", func(q *orm.Query) (*orm.Query, error) {
			return q.Order("host_identifier.id ASC"), nil
		}).
		Relation("IPReservations", func(q *orm.Query) (*orm.Query, error) {
			return q.Order("ip_reservation.id ASC"), nil
		}).
		Relation("Subnet").
		Relation("LocalHosts.Daemon.App").
		Where("host.id = ?", hostID).
		Select()
	if err != nil {
		if errors.Is(err, pg.ErrNoRows) {
			return nil, nil
		}
		err = pkgerrors.Wrapf(err, "problem with getting a host with id %d", hostID)
		return nil, err
	}
	return host, err
}

// Fetch all hosts having address reservations belonging to a specific family
// or all reservations regardless of the family.
func GetAllHosts(dbi dbops.DBI, family int) ([]Host, error) {
	hosts := []Host{}
	q := dbi.Model(&hosts).DistinctOn("id")

	// Let's be liberal and allow other values than 0 too. The only special
	// ones are 4 and 6.
	if family == 4 || family == 6 {
		q = q.Join("INNER JOIN ip_reservation AS r ON r.host_id = host.id")
		q = q.Where("family(r.address) = ?", family)
	}

	// Include host identifiers and IP reservations.
	q = q.
		Relation("HostIdentifiers", func(q *orm.Query) (*orm.Query, error) {
			return q.Order("host_identifier.id ASC"), nil
		}).
		Relation("IPReservations", func(q *orm.Query) (*orm.Query, error) {
			return q.Order("ip_reservation.id ASC"), nil
		}).
		Relation("LocalHosts").
		OrderExpr("id ASC")

	err := q.Select()
	if err != nil {
		if errors.Is(err, pg.ErrNoRows) {
			return nil, nil
		}
		err = pkgerrors.Wrapf(err, "problem with getting all hosts for family %d", family)
		return nil, err
	}
	return hosts, err
}

// Fetches a collection of hosts by subnet ID. This function may be sometimes
// used within a transaction. In particular, when we're synchronizing hosts
// fetched from the Kea hosts backend in multiple chunks.`.
func GetHostsBySubnetID(dbi dbops.DBI, subnetID int64) ([]Host, error) {
	hosts := []Host{}
	q := dbi.Model(&hosts).
		Relation("HostIdentifiers", func(q *orm.Query) (*orm.Query, error) {
			return q.Order("host_identifier.id ASC"), nil
		}).
		Relation("IPReservations", func(q *orm.Query) (*orm.Query, error) {
			return q.Order("ip_reservation.id ASC"), nil
		}).
		Relation("LocalHosts").
		OrderExpr("id ASC")

	// Subnet ID is never zero, it may be NULL. The reason for it is that we
	// have a foreign key that requires subnet to exist for non NULL value.
	// This constraint allows for NULL subnet_id though. Therefore, searching
	// for a host with subnet_id of zero is really searching for a host with
	// the NULL value.
	if subnetID == 0 {
		q = q.Where("host.subnet_id IS NULL")
	} else {
		q = q.Where("host.subnet_id = ?", subnetID)
	}

	err := q.Select()
	if err != nil {
		if errors.Is(err, pg.ErrNoRows) {
			return nil, nil
		}
		err = pkgerrors.Wrapf(err, "problem with getting hosts by subnet ID %d", subnetID)
		return nil, err
	}
	return hosts, err
}

// Fetches a collection of hosts by daemon ID and optionally filters by a
// data source.
func GetHostsByDaemonID(dbi dbops.DBI, daemonID int64, dataSource string) ([]Host, int64, error) {
	hosts := []Host{}
	q := dbi.Model(&hosts).
		Join("INNER JOIN local_host AS lh ON host.id = lh.host_id").
		Relation("HostIdentifiers", func(q *orm.Query) (*orm.Query, error) {
			return q.Order("host_identifier.id ASC"), nil
		}).
		Relation("IPReservations", func(q *orm.Query) (*orm.Query, error) {
			return q.Order("ip_reservation.id ASC"), nil
		}).
		Relation("LocalHosts").
		Relation("Subnet.LocalSubnets").
		OrderExpr("id ASC").
		Where("lh.daemon_id = ?", daemonID)

	// Optionally filter by a data source.
	if len(dataSource) > 0 {
		q = q.Where("lh.data_source = ?", dataSource)
	}

	total, err := q.SelectAndCount()
	if err != nil {
		if errors.Is(err, pg.ErrNoRows) {
			return nil, 0, nil
		}
		err = pkgerrors.Wrapf(err, "problem with getting hosts for daemon %d", daemonID)
	}
	return hosts, int64(total), err
}

// Fetches a collection of hosts from the database. The offset and
// limit specify the beginning of the page and the maximum size of the
// page. The appID, if different than 0, is used to fetch hosts whose
// local hosts belong to indicated app. The optional subnetID is used
// to fetch hosts belonging to the particular IPv4 or IPv6 subnet. If
// this value is set to nil all subnets are returned.  The value of 0
// indicates that only global hosts are to be returned. Filtering text
// allows for searching hosts by reserved IP addresses, host identifiers
// (using hexadecimal digits or a textual format) and hostnames. It is
// allowed to specify colons while searching for hosts by host identifiers.
// If global flag is true then only hosts from the global scope are
// returned (i.e. not assigned to any subnet), if false then only hosts
// from subnets are returned. sortField allows indicating sort column
// in database and sortDir allows selection the order of sorting. If
// sortField is empty then id is used for sorting. If SortDirAny is
// used then ASC order is used.
func GetHostsByPage(dbi dbops.DBI, offset, limit int64, appID int64, subnetID *int64, filterText *string, global *bool, sortField string, sortDir SortDirEnum) ([]Host, int64, error) {
	hosts := []Host{}
	q := dbi.Model(&hosts)

	// prepare distinct on expression to include sort field, otherwise distinct on will fail
	distingOnFields := "host.id"
	if sortField != "" && sortField != "id" && sortField != "host.id" {
		distingOnFields = sortField + ", " + distingOnFields
	}
	q = q.DistinctOn(distingOnFields)

	// When filtering by appID we also need the local_host table as it holds the
	// application identifier.
	if appID != 0 {
		q = q.Join("INNER JOIN local_host AS lh ON host.id = lh.host_id")
		q = q.Join("INNER JOIN daemon AS d ON lh.daemon_id = d.id")
		q = q.Where("d.app_id = ?", appID)
	}

	// filter by subnet id
	if subnetID != nil && *subnetID != 0 {
		// Get hosts for matching subnet id.
		q = q.Where("subnet_id = ?", *subnetID)
	}

	// filter global or non-global hosts
	if (global != nil && *global) || (subnetID != nil && *subnetID == 0) {
		q = q.WhereOr("host.subnet_id IS NULL")
	}
	if global != nil && !*global {
		q = q.WhereOr("host.subnet_id IS NOT NULL")
	}

	// filter by text
	if filterText != nil && len(*filterText) > 0 {
		// It is possible that the user is typing a search text with colons
		// for host identifiers. We need to remove them because they are
		// not present in the database.
		colonlessFilterText := strings.ReplaceAll(*filterText, ":", "")
		q = q.Join("INNER JOIN ip_reservation AS r ON r.host_id = host.id")
		q = q.Join("INNER JOIN host_identifier AS i ON i.host_id = host.id")
		q = q.WhereGroup(func(q *orm.Query) (*orm.Query, error) {
			q = q.WhereOr("text(r.address) ILIKE ?", "%"+*filterText+"%").
				WhereOr("i.type::text ILIKE ?", "%"+*filterText+"%").
				WhereOr("encode(i.value, 'hex') ILIKE ?", "%"+colonlessFilterText+"%").
				WhereOr("encode(i.value, 'escape') ILIKE ?", "%"+*filterText+"%").
				WhereOr("host.hostname ILIKE ?", "%"+*filterText+"%")
			return q, nil
		})
	}

	q = q.
		Relation("HostIdentifiers", func(q *orm.Query) (*orm.Query, error) {
			return q.Order("host_identifier.id ASC"), nil
		}).
		Relation("IPReservations", func(q *orm.Query) (*orm.Query, error) {
			return q.Order("ip_reservation.id ASC"), nil
		}).
		Relation("LocalHosts").
		Relation("LocalHosts.Daemon.App").
		Relation("LocalHosts.Daemon.App.Machine").
		Relation("LocalHosts.Daemon.App.AccessPoints")

	// Only join the subnet if querying all hosts or hosts belonging to a
	// given subnet.
	if subnetID == nil || *subnetID > 0 {
		q = q.Relation("Subnet")
	}

	// prepare sorting expression, offset and limit
	ordExpr := prepareOrderExpr("host", sortField, sortDir)
	q = q.OrderExpr(ordExpr)
	q = q.Offset(int(offset))
	q = q.Limit(int(limit))

	total, err := q.SelectAndCount()
	if err != nil {
		if errors.Is(err, pg.ErrNoRows) {
			return nil, 0, nil
		}
		err = pkgerrors.Wrapf(err, "problem with getting hosts by page")
	}
	return hosts, int64(total), err
}

// Delete host, host identifiers and reservations by id.
func DeleteHost(dbi dbops.DBI, hostID int64) error {
	host := &Host{
		ID: hostID,
	}
	result, err := dbi.Model(host).WherePK().Delete()
	if err != nil {
		err = pkgerrors.Wrapf(err, "problem with deleting a host with id %d", hostID)
	} else if result.RowsAffected() <= 0 {
		err = pkgerrors.Wrapf(ErrNotExists, "host with id %d does not exist", hostID)
	}
	return err
}

// Associates a daemon with the host having a specified ID in a
// transaction. Internally, the association is made via the local_host
// table which holds information about the host from the given daemon
// perspective. The source argument indicates whether the host
// information was fetched from the daemon's configuration or via the
// command.
func addDaemonToHost(tx *pg.Tx, host *Host, daemonID int64, source string) error {
	localHost := LocalHost{
		HostID:     host.ID,
		DaemonID:   daemonID,
		DataSource: source,
	}
	q := tx.Model(&localHost).
		OnConflict("(daemon_id, host_id) DO UPDATE").
		Set("daemon_id = EXCLUDED.daemon_id").
		Set("data_source = EXCLUDED.data_source")

	_, err := q.Insert()
	if err != nil {
		err = pkgerrors.Wrapf(err, "problem with associating the daemon %d with the host %d",
			daemonID, host.ID)
	}
	return err
}

// Associates a daemon with the host having a specified ID.
// It begins a new transaction when dbi has a *pg.DB type or uses an
// existing transaction when dbi has a *pg.Tx type.
func AddDaemonToHost(dbi dbops.DBI, host *Host, daemonID int64, source string) error {
	if db, ok := dbi.(*pg.DB); ok {
		return db.RunInTransaction(context.Background(), func(tx *pg.Tx) error {
			return addDaemonToHost(tx, host, daemonID, source)
		})
	}
	return addDaemonToHost(dbi.(*pg.Tx), host, daemonID, source)
}

// Dissociates a daemon from the hosts. The dataSource designates a data
// source from which the deleted hosts were fetched. If it is an empty value
// the hosts from all sources are deleted. The first returned value indicates
// if any row was removed from the local_host table.
func DeleteDaemonFromHosts(dbi dbops.DBI, daemonID int64, dataSource string) (int64, error) {
	q := dbi.Model((*LocalHost)(nil)).
		Where("daemon_id = ?", daemonID)

	if len(dataSource) > 0 {
		q = q.Where("data_source = ?", dataSource)
	}

	result, err := q.Delete()
	if err != nil && !errors.Is(err, pg.ErrNoRows) {
		err = pkgerrors.Wrapf(err, "problem with deleting a daemon %d from hosts", daemonID)
		return 0, err
	}
	return int64(result.RowsAffected()), nil
}

// Deletes hosts which are not associated with any apps. Returns deleted host
// count and an error.
func DeleteOrphanedHosts(dbi dbops.DBI) (int64, error) {
	subquery := dbi.Model(&[]LocalHost{}).
		Column("id").
		Limit(1).
		Where("host.id = local_host.host_id")
	result, err := dbi.Model(&[]Host{}).
		Where("(?) IS NULL", subquery).
		Delete()
	if err != nil {
		err = pkgerrors.Wrapf(err, "problem with deleting orphaned hosts")
		return 0, err
	}
	return int64(result.RowsAffected()), nil
}

// Iterates over the list of hosts and commits them into the database. The hosts
// can be associated with a subnet or can be made global.
func commitHostsIntoDB(tx *pg.Tx, hosts []Host, subnetID int64, daemon *Daemon, source string) (err error) {
	for i := range hosts {
		hosts[i].SubnetID = subnetID
		newHost := (hosts[i].ID == 0)
		if newHost {
			err = AddHost(tx, &hosts[i])
			if err != nil {
				err = pkgerrors.WithMessagef(err, "unable to add detected host to the database")
				return err
			}
		} else if hosts[i].UpdateOnCommit {
			err = UpdateHost(tx, &hosts[i])
			if err != nil {
				err = pkgerrors.WithMessagef(err, "unable to update detected host in the database")
				return err
			}
		}
		if newHost || hosts[i].UpdateOnCommit {
			err = AddDaemonToHost(tx, &hosts[i], daemon.ID, source)
			if err != nil {
				err = pkgerrors.WithMessagef(err, "unable to associate detected host with Kea daemon having id %d",
					daemon.ID)
				return err
			}
		}
	}
	return nil
}

// Iterates over the list of hosts and commits them as global hosts.
func CommitGlobalHostsIntoDB(tx *pg.Tx, hosts []Host, daemon *Daemon, source string) (err error) {
	return commitHostsIntoDB(tx, hosts, 0, daemon, source)
}

// Iterates over the hosts belonging to the given subnet and stores them
// or updates in the database.
func CommitSubnetHostsIntoDB(tx *pg.Tx, subnet *Subnet, daemon *Daemon, source string) (err error) {
	return commitHostsIntoDB(tx, subnet.Hosts, subnet.ID, daemon, source)
}

// This function checks if the given host includes a reservation for the
// given address.
func (host Host) HasIPAddress(ipAddress string) bool {
	for _, r := range host.IPReservations {
		hostCidr, err := storkutil.MakeCIDR(r.Address)
		if err != nil {
			continue
		}
		argCidr, err := storkutil.MakeCIDR(ipAddress)
		if err != nil {
			return false
		}
		if hostCidr == argCidr {
			return true
		}
	}
	return false
}

// This function checks if the given host has specified identifier and if
// the identifier value matches. The first returned value indicates if the
// identifiers exists. The second one indicates if the value matches.
func (host Host) HasIdentifier(idType string, identifier []byte) (bool, bool) {
	for _, i := range host.HostIdentifiers {
		if idType == i.Type {
			if bytes.Equal(i.Value, identifier) {
				return true, true
			}
			return true, false
		}
	}
	return false, false
}

// This function checks if the given host has an identifier of a given type.
func (host Host) HasIdentifierType(idType string) bool {
	for _, i := range host.HostIdentifiers {
		if idType == i.Type {
			return true
		}
	}
	return false
}

// Checks if two hosts have the same IP reservations.
func (host Host) HasEqualIPReservations(other *Host) bool {
	if len(host.IPReservations) != len(other.IPReservations) {
		return false
	}

	for _, o := range other.IPReservations {
		if !host.HasIPAddress(o.Address) {
			return false
		}
	}

	return true
}

// Checks if two hosts are equal.
func (host Host) Equal(other *Host) bool {
	if len(host.HostIdentifiers) != len(other.HostIdentifiers) {
		return false
	}

	for _, o := range other.HostIdentifiers {
		if _, ok := host.HasIdentifier(o.Type, o.Value); !ok {
			return false
		}
	}
	return host.HasEqualIPReservations(other)
}

// Converts host identifier value to a string of hexadecimal digits.
func (id HostIdentifier) ToHex(separator string) string {
	// Convert binary value to hexadecimal value.
	encoded := hex.EncodeToString(id.Value)
	// If no separator specified, return what we have.
	if len(separator) == 0 {
		return encoded
	}
	var separated string
	// Iterate over pairs of hexadecimal digits and insert separator
	// between them.
	for i := 0; i < len(encoded); i += 2 {
		if len(separated) > 0 {
			separated += separator
		}
		separated += encoded[i : i+2]
	}
	return separated
}
