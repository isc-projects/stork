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
	dhcpmodel "isc.org/stork/datamodel/dhcp"
	dbops "isc.org/stork/server/database"
	storkutil "isc.org/stork/util"
)

// Source of the host information, i.e. configuration file or API (host_cmds).
type HostDataSource string

// Valid host data sources.
const (
	HostDataSourceUnspecified HostDataSource = ""
	HostDataSourceConfig      HostDataSource = "config"
	HostDataSourceAPI         HostDataSource = "api"
)

// Returns true if the data source is API.
func (s HostDataSource) IsAPI() bool {
	return s == HostDataSourceAPI
}

// Returns true if the data source is configuration file.
func (s HostDataSource) IsConfig() bool {
	return s == HostDataSourceConfig
}

// Returns true if the data source is unspecified.
func (s HostDataSource) IsUnspecified() bool {
	return s == HostDataSourceUnspecified
}

// Returns true if the data source is specified.
func (s HostDataSource) IsSpecified() bool {
	return !s.IsUnspecified()
}

// Converts HostDataSource to string.
func (s HostDataSource) String() string {
	return string(s)
}

// Creates HostDataSource instance from string. It returns an error
// when specified string is neither "api" nor "config".
func ParseHostDataSource(s string) (hds HostDataSource, err error) {
	hds = HostDataSource(s)
	if hds != HostDataSourceConfig && hds != HostDataSourceAPI {
		err = pkgerrors.Errorf("unsupported host data source '%s'", s)
	}
	return
}

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
// a single IP address or prefix reservation associated with a selected local
// host.
type IPReservation struct {
	ID          int64
	Address     string
	LocalHostID int64
}

// Checks if reservation is a delegated prefix.
func (r *IPReservation) IsPrefix() bool {
	ip := storkutil.ParseIP(r.Address)
	if ip == nil {
		return false
	}
	return ip.Prefix
}

// This structure reflects a row in the host table. It may also be associated with
// one or more identifiers which are used for matching DHCP clients with the
// host.
type Host struct {
	ID        int64 `pg:",pk"`
	CreatedAt time.Time
	SubnetID  int64
	Subnet    *Subnet `pg:"rel:has-one"`

	HostIdentifiers []HostIdentifier `pg:"rel:has-many"`

	LocalHosts []LocalHost `pg:"rel:has-many"`
}

// This structure links a host entry stored in the database with a daemon from
// which it has been retrieved. It provides M:N relationship between hosts
// and daemons.
// The local host may be associated with zero, one or multiple IP reservations.
type LocalHost struct {
	DHCPOptionSet
	ID         int64
	HostID     int64
	DaemonID   int64
	DataSource HostDataSource

	Daemon *Daemon `pg:"rel:has-one"`
	Host   *Host   `pg:"rel:has-one"`

	ClientClasses  []string `pg:",array"`
	NextServer     string
	ServerHostname string
	BootFileName   string

	// Reservation data.
	Hostname       string
	IPReservations []IPReservation `pg:"rel:has-many"`
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
			err = pkgerrors.Wrapf(err, "problem adding host identifier with type %s for host with ID %d",
				identifier.Type, host.ID)
			return err
		}
		host.HostIdentifiers[i] = identifier
	}
	return nil
}

// Associates a host with IP reservations.
func addIPReservations(tx *pg.Tx, host *Host) error {
	for _, lh := range host.LocalHosts {
		for i := range lh.IPReservations {
			r := &lh.IPReservations[i]
			r.LocalHostID = lh.ID
			_, err := tx.Model(r).
				OnConflict("DO NOTHING").
				Insert()
			if err != nil {
				err = pkgerrors.Wrapf(err, "problem adding IP reservation %s for host with ID %d",
					r.Address, host.ID)
				return err
			}
		}
	}
	return nil
}

// Adds new host, its reservations and identifiers into the database in
// a transaction.
func addHost(tx *pg.Tx, host *Host) error {
	// Add the host and fetch its id.
	_, err := tx.Model(host).Insert()
	if err != nil {
		err = pkgerrors.Wrapf(err, "problem adding new host")
		return err
	}
	// Associate the host with the given id with its identifiers.
	err = addHostIdentifiers(tx, host)
	if err != nil {
		return err
	}
	return nil
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
		err = pkgerrors.Wrapf(err, "problem deleting host identifiers for host %d", host.ID)
		return err
	}
	// Add or update host identifiers.
	err = addHostIdentifiers(tx, host)
	if err != nil {
		return pkgerrors.WithMessagef(err, "problem updating host with ID %d", host.ID)
	}

	// Update the host information.
	result, err := tx.Model(host).WherePK().ExcludeColumn("created_at").Update()
	if err != nil {
		err = pkgerrors.Wrapf(err, "problem updating host with ID %d", host.ID)
	} else if result.RowsAffected() <= 0 {
		err = pkgerrors.Wrapf(ErrNotExists, "host with ID %d does not exist", host.ID)
	}
	return err
}

// Attempts to update a host and its local hosts with in an existing transaction.
func updateHostWithLocalHosts(tx *pg.Tx, host *Host) error {
	err := updateHost(tx, host)
	if err != nil {
		return err
	}
	// Delete current associations of the host with the daemons.
	_, err = tx.Model((*LocalHost)(nil)).
		Where("host_id = ?", host.ID).
		Delete()
	if err != nil {
		return pkgerrors.Wrapf(err, "problem deleting daemons from host %d", host.ID)
	}
	// Add new associations.
	err = addHostReferences(tx, host)
	return err
}

// Attempts to update a host and its local hosts within a transaction. If the dbi
// does not point to a transaction, a new transaction is started.
func UpdateHost(dbi dbops.DBI, host *Host) error {
	if db, ok := dbi.(*pg.DB); ok {
		return db.RunInTransaction(context.Background(), func(tx *pg.Tx) error {
			return updateHostWithLocalHosts(tx, host)
		})
	}
	return updateHostWithLocalHosts(dbi.(*pg.Tx), host)
}

// Fetch the host by ID.
func GetHost(dbi dbops.DBI, hostID int64) (*Host, error) {
	host := &Host{}
	err := dbi.Model(host).
		Relation("HostIdentifiers", func(q *orm.Query) (*orm.Query, error) {
			return q.Order("host_identifier.id ASC"), nil
		}).
		Relation("LocalHosts.IPReservations", func(q *orm.Query) (*orm.Query, error) {
			return q.Order("ip_reservation.id ASC"), nil
		}).
		Relation("Subnet.LocalSubnets").
		Relation("LocalHosts.Daemon.App.Machine").
		Relation("LocalHosts.Daemon.App.AccessPoints").
		Where("host.id = ?", hostID).
		Select()
	if err != nil {
		if errors.Is(err, pg.ErrNoRows) {
			return nil, nil
		}
		err = pkgerrors.Wrapf(err, "problem getting host with ID %d", hostID)
		return nil, err
	}
	return host, err
}

// Fetch all hosts having address reservations belonging to a specific family
// or all reservations regardless of the family.
func GetAllHosts(dbi dbops.DBI, family int) ([]Host, error) {
	var hosts []Host
	q := dbi.Model(&hosts).DistinctOn("id")

	// Let's be liberal and allow other values than 0 too. The only special
	// ones are 4 and 6.
	if family == 4 || family == 6 {
		q = q.Join("INNER JOIN local_host AS lh").JoinOn("host.id = lh.host_id")
		q = q.Join("INNER JOIN ip_reservation AS r").JoinOn("r.local_host_id = lh.id")
		q = q.Where("family(r.address) = ?", family)
	}

	// Include host identifiers and IP reservations.
	q = q.
		Relation("HostIdentifiers", func(q *orm.Query) (*orm.Query, error) {
			return q.Order("host_identifier.id ASC"), nil
		}).
		Relation("LocalHosts.IPReservations", func(q *orm.Query) (*orm.Query, error) {
			return q.Order("ip_reservation.id ASC"), nil
		}).
		OrderExpr("id ASC")

	err := q.Select()
	if err != nil {
		if errors.Is(err, pg.ErrNoRows) {
			return nil, nil
		}
		err = pkgerrors.Wrapf(err, "problem getting all hosts for family %d", family)
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
		Relation("LocalHosts.IPReservations", func(q *orm.Query) (*orm.Query, error) {
			return q.Order("ip_reservation.id ASC"), nil
		}).
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
		err = pkgerrors.Wrapf(err, "problem getting hosts by subnet ID %d", subnetID)
		return nil, err
	}
	return hosts, err
}

// Fetches a collection of hosts by daemon ID and optionally filters by a
// data source.
func GetHostsByDaemonID(dbi dbops.DBI, daemonID int64, dataSource HostDataSource) ([]Host, int64, error) {
	hosts := []Host{}
	q := dbi.Model(&hosts).
		Join("INNER JOIN local_host AS lh ON host.id = lh.host_id").
		Relation("HostIdentifiers", func(q *orm.Query) (*orm.Query, error) {
			return q.Order("host_identifier.id ASC"), nil
		}).
		Relation("LocalHosts.IPReservations", func(q *orm.Query) (*orm.Query, error) {
			return q.Order("ip_reservation.id ASC"), nil
		}).
		Relation("Subnet.LocalSubnets").
		OrderExpr("id ASC").
		Where("lh.daemon_id = ?", daemonID)

	// Optionally filter by a data source.
	if dataSource.IsSpecified() {
		q = q.Where("lh.data_source = ?", dataSource)
	}

	total, err := q.SelectAndCount()
	if err != nil {
		if errors.Is(err, pg.ErrNoRows) {
			return nil, 0, nil
		}
		err = pkgerrors.Wrapf(err, "problem getting hosts for daemon %d", daemonID)
	}
	return hosts, int64(total), err
}

// Container for values filtering hosts fetched by page.
//
// The AppID, if different than 0, is used to fetch hosts whose local hosts belong to
// the indicated app.
//
// The optional SubnetID is used to fetch hosts belonging to the particular IPv4 or
// IPv6 subnet. If this value is set to nil all subnets are returned. The value of 0
// indicates that only global hosts are to be returned.
//
// The LocalSubnetID filters hosts by the subnet ID specified in the Kea configuration.
//
// Filtering text allows for searching hosts by reserved IP addresses, host identifiers
// (using hexadecimal digits or a textual format) and hostnames. It is allowed to specify
// colons while searching for hosts by host identifiers. If Global flag is true then only
// hosts from the global scope are returned (i.e. not assigned to any subnet), if false
// then only hosts from subnets are returned.
//
// The DHCPDataConflict flag indicates whether to return hosts with
// conflicted (different) DHCP data in related Kea configurations.

// The DHCPDataDuplicate flag indicates whether to return hosts with
// duplicated (the same) DHCP data in related Kea configurations.
type HostsByPageFilters struct {
	AppID             *int64
	SubnetID          *int64
	LocalSubnetID     *int64
	FilterText        *string
	Global            *bool
	DHCPDataConflict  *bool
	DHCPDataDuplicate *bool
}

// Fetches a collection of hosts from the database.
// Returns the chunk of hosts, the total number of hosts or an error.
//
// The offset and limit specify the beginning of the page and the maximum size of the
// page.
//
// sortField allows indicating sort column in database and sortDir allows selection the
// order of sorting. If sortField is empty then id is used for sorting. If SortDirAny is
// used then ASC order is used.
//
// This function exceeds the maximum cognitive complexity (> 65), but it is a
// consequence of a number of filters, not the complexity of the function logic.
func GetHostsByPage(dbi dbops.DBI, offset, limit int64, filters HostsByPageFilters, sortField string, sortDir SortDirEnum) ([]Host, int64, error) { //nolint: gocyclo
	hosts := []Host{}
	q := dbi.Model(&hosts)

	// Prepare distinct on expression to include sort field, otherwise distinct
	// on will fail.
	distinctOnFields := "host.id"
	if sortField != "" && sortField != "id" && sortField != "host.id" {
		distinctOnFields = sortField + ", " + distinctOnFields
	}
	q = q.DistinctOn(distinctOnFields)

	// Join to the local host table.
	if (filters.AppID != nil && *filters.AppID != 0) || (filters.FilterText != nil && len(*filters.FilterText) > 0) {
		q = q.Join("JOIN local_host").JoinOn("host.id = local_host.host_id")
	}

	// Filter by app ID.
	// When filtering by appID we also need the local_host
	// table as it holds the application identifier.
	if filters.AppID != nil && *filters.AppID != 0 {
		q = q.Join("JOIN daemon").JoinOn("local_host.daemon_id = daemon.id")
		q = q.Where("daemon.app_id = ?", *filters.AppID)
	}

	// Filter by conflict or duplicate.
	if filters.DHCPDataConflict != nil || filters.DHCPDataDuplicate != nil {
		// This subquery appends a "conflict" column with a boolean value
		// indicating whether the DHCP data are inconsistent in the local hosts
		// of the host.
		// The conflict column is:
		//
		// - TRUE if the DHCP data are inconsistent in the local hosts (there
		//   are at least two local hosts for a given host with different DHCP
		//   data)
		// - FALSE if they are consistent/duplicated (all local hosts for a
		//   given host have the same DHCP data)
		// - NULL if there is only one local host.
		//
		// The DHCP data are considered inconsistent if at least one of the
		// following fields is different in the local hosts:
		//
		// - DHCP options (checked by comparing the hash of the DHCP option)
		// - Boot options: next server, server hostname, boot file name
		// - Client classes
		//
		// To compare a particular DHCP data field, we use the following idiom:
		//     max(field) != min(field).
		// In PostgresSQL the max and min functions accept any type.
		// If the values for all local hosts are the same, the max and min
		// functions return the same value, so the comparison returns FALSE
		// (no conflict). If there are different values, the comparison returns
		// TRUE (conflict). It doesn't matter which value is returned by the
		// max and min functions.
		// I didn't find a better way to compare arbitrary values in a single
		// query that would support any number of referenced local hosts.
		//
		// The IS DISTINCT FROM operator is used because some of the fields
		// may be NULL.
		//
		// The conflict column has NULL value if there is only one local host
		// for a given host. This is because the HAVING clause filters out this
		// kind of entries, so there are no corresponding rows in the subquery
		// and the subquery results are joined with LEFT JOIN.
		reservedSubquery := dbi.Model((*IPReservation)(nil)).
			Column("local_host_id").
			ColumnExpr("array_agg(address ORDER BY address) AS addresses").
			Group("local_host_id")

		conflictSubquery := dbi.Model((*struct {
			tableName struct{} `pg:"local_host"`
			HostID    int64
			Conflict  bool
		})(nil)).
			Join("LEFT JOIN (?) AS reserved", reservedSubquery).JoinOn("local_host.id = reserved.local_host_id").
			DistinctOn("host_id").
			Column("host_id").
			ColumnExpr(`
				   COUNT(DISTINCT COALESCE(dhcp_option_set_hash, '')) > 1
				OR COUNT(DISTINCT COALESCE(client_classes, '{}')) > 1
				OR COUNT(DISTINCT COALESCE(next_server, '')) > 1
				OR COUNT(DISTINCT COALESCE(server_hostname, '')) > 1
				OR COUNT(DISTINCT COALESCE(boot_file_name, '')) > 1
				OR COUNT(DISTINCT COALESCE(hostname, '')) > 1
				OR COUNT(DISTINCT COALESCE(addresses, '{}')) > 1
				AS conflict`).
			Group("host_id", "daemon_id").
			Having("COUNT(*) > 1").
			Order("host_id")

		q = q.Join("LEFT JOIN (?) AS duplicate", conflictSubquery).
			JoinOn("host.id = duplicate.host_id")

		// Joined 'conflict' column is TRUE if the DHCP data are inconsistent
		// in the local hosts, FALSE if they are consistent/duplicated, or
		// NULL if there is only one local host.
		q.WhereGroup(func(q *orm.Query) (*orm.Query, error) {
			// Glossary:
			// Conflicted - the hosts that have at least two local hosts with
			// 		different DHCP data.
			// Duplicated - the hosts that have at least two local hosts and
			// 		all of them have the same DHCP data.
			// Single - the hosts that have only one local host.
			//
			// Filter by conflict -> Return conflicted.
			// Filter by duplicate -> Return duplicated.
			// Filter by conflict AND duplicate -> Return conflicted and duplicated.
			// Filter by NOT conflict -> Return duplicated and single.
			// Filter by NOT duplicate -> Return conflicted and single.
			// Filter by NOT conflict AND NOT duplicate -> Return single.
			// Filter by conflict AND NOT duplicate -> Return conflicted and single.
			// Filter by NOT conflict AND duplicate -> Return duplicated and single.
			// Note: Above AND doesn't mean logical AND but rather a combination of filters.

			// Filter by conflict or filter by NOT duplicate.
			if (filters.DHCPDataConflict != nil && *filters.DHCPDataConflict) ||
				(filters.DHCPDataConflict == nil && filters.DHCPDataDuplicate != nil && !*filters.DHCPDataDuplicate) {
				q = q.WhereOr("duplicate.conflict = TRUE")
			}

			// Filter by duplicate or filter by NOT conflict.
			if (filters.DHCPDataDuplicate != nil && *filters.DHCPDataDuplicate) ||
				(filters.DHCPDataDuplicate == nil && filters.DHCPDataConflict != nil && !*filters.DHCPDataConflict) {
				q = q.WhereOr("duplicate.conflict = FALSE")
			}

			// Filter by NOT conflict or filter by NOT duplicate.
			if (filters.DHCPDataConflict != nil && !*filters.DHCPDataConflict) || (filters.DHCPDataDuplicate != nil && !*filters.DHCPDataDuplicate) {
				q = q.WhereOr("duplicate.conflict IS NULL")
			}
			return q, nil
		})
	}

	// Filter by subnet ID.
	if filters.SubnetID != nil && *filters.SubnetID != 0 {
		// Get hosts for matching subnet ID.
		q = q.Where("host.subnet_id = ?", *filters.SubnetID)
	}

	// Filter by local subnet ID.
	if filters.LocalSubnetID != nil && *filters.LocalSubnetID != 0 {
		q = q.Join("JOIN local_subnet").JoinOn("local_subnet.subnet_id = host.subnet_id")
		q = q.Where("local_subnet.local_subnet_id = ?", *filters.LocalSubnetID)
	}

	// Filter global or non-global hosts.
	if (filters.Global != nil && *filters.Global) || (filters.SubnetID != nil && *filters.SubnetID == 0) {
		q = q.Where("host.subnet_id IS NULL")
	}
	if filters.Global != nil && !*filters.Global {
		q = q.Where("host.subnet_id IS NOT NULL")
	}

	// Filter by text.
	if filters.FilterText != nil && len(*filters.FilterText) > 0 {
		// It is possible that the user is typing a search text with colons
		// for host identifiers. We need to remove them because they are
		// not present in the database.
		colonlessFilterText := strings.ReplaceAll(*filters.FilterText, ":", "")
		q = q.Join("JOIN host_identifier AS i").JoinOn("i.host_id = host.id")
		q = q.Join("LEFT JOIN ip_reservation AS r").JoinOn("r.local_host_id = local_host.id")
		q = q.WhereGroup(func(q *orm.Query) (*orm.Query, error) {
			q = q.WhereOr("text(r.address) ILIKE ?", "%"+*filters.FilterText+"%").
				WhereOr("i.type::text ILIKE ?", "%"+*filters.FilterText+"%").
				WhereOr("encode(i.value, 'hex') ILIKE ?", "%"+colonlessFilterText+"%").
				WhereOr("encode(i.value, 'escape') ILIKE ?", "%"+*filters.FilterText+"%").
				WhereOr("local_host.hostname ILIKE ?", "%"+*filters.FilterText+"%")
			return q, nil
		})
	}

	q = q.
		Relation("HostIdentifiers", func(q *orm.Query) (*orm.Query, error) {
			return q.Order("host_identifier.id ASC"), nil
		}).
		Relation("LocalHosts.IPReservations", func(q *orm.Query) (*orm.Query, error) {
			return q.Order("ip_reservation.id ASC"), nil
		}).
		Relation("LocalHosts", func(q *orm.Query) (*orm.Query, error) {
			return q.Order("local_host.id ASC"), nil
		}).
		Relation("LocalHosts.Daemon.App").
		Relation("LocalHosts.Daemon.App.Machine").
		Relation("LocalHosts.Daemon.App.AccessPoints")

	// Only join the subnet if querying all hosts or hosts belonging to a
	// given subnet.
	if filters.SubnetID == nil || *filters.SubnetID > 0 {
		q = q.Relation("Subnet")
	}

	// Prepare sorting expression, offset and limit.
	ordExpr := prepareOrderExpr("host", sortField, sortDir)
	q = q.OrderExpr(ordExpr)
	q = q.Offset(int(offset))
	q = q.Limit(int(limit))

	total, err := q.SelectAndCount()
	if err != nil {
		if errors.Is(err, pg.ErrNoRows) {
			return nil, 0, nil
		}
		err = pkgerrors.Wrapf(err, "problem getting hosts by page")
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
		err = pkgerrors.Wrapf(err, "problem deleting the host with ID %d", hostID)
	} else if result.RowsAffected() <= 0 {
		err = pkgerrors.Wrapf(ErrNotExists, "host with ID %d does not exist", hostID)
	}
	return err
}

// Iterates over the LocalHost instances of a Host and inserts or updates them
// and their IP reservations in the database.
func addHostReferences(tx *pg.Tx, host *Host) error {
	for i := range host.LocalHosts {
		host.LocalHosts[i].HostID = host.ID
		q := tx.Model(&host.LocalHosts[i]).
			OnConflict("(daemon_id, host_id, data_source) DO UPDATE").
			Set("client_classes = EXCLUDED.client_classes").
			Set("dhcp_option_set = EXCLUDED.dhcp_option_set").
			Set("dhcp_option_set_hash = EXCLUDED.dhcp_option_set_hash").
			Set("next_server = EXCLUDED.next_server").
			Set("server_hostname = EXCLUDED.server_hostname").
			Set("boot_file_name = EXCLUDED.boot_file_name")

		_, err := q.Insert()
		if err != nil {
			return pkgerrors.Wrapf(err, "problem associating the daemon %d with the host %d",
				host.LocalHosts[i].DaemonID, host.ID)
		}
	}

	return addIPReservations(tx, host)
}

// Attempts to add a host, its local hosts and IP reservations within an
// existing transaction.
func addHostWithReferences(tx *pg.Tx, host *Host) error {
	err := addHost(tx, host)
	if err != nil {
		return err
	}
	err = addHostReferences(tx, host)
	return err
}

// Attempts to add a host, its local hosts and IP reservations within a
// transaction. If the dbi does not point to a transaction, a new transaction
// is started.
func AddHost(dbi dbops.DBI, host *Host) error {
	if db, ok := dbi.(*pg.DB); ok {
		return db.RunInTransaction(context.Background(), func(tx *pg.Tx) error {
			return addHostWithReferences(tx, host)
		})
	}
	return addHostWithReferences(dbi.(*pg.Tx), host)
}

// Dissociates a daemon from the hosts. The dataSource designates a data
// source from which the deleted hosts were fetched. If it is an empty value
// the hosts from all sources are deleted. The first returned value indicates
// if any row was removed from the local_host table.
func DeleteDaemonFromHosts(dbi dbops.DBI, daemonID int64, dataSource HostDataSource) (int64, error) {
	q := dbi.Model((*LocalHost)(nil)).
		Where("daemon_id = ?", daemonID)

	if dataSource.IsSpecified() {
		q = q.Where("data_source = ?", dataSource)
	}

	result, err := q.Delete()
	if err != nil && !errors.Is(err, pg.ErrNoRows) {
		err = pkgerrors.Wrapf(err, "problem deleting the daemon %d from hosts", daemonID)
		return 0, err
	}
	return int64(result.RowsAffected()), nil
}

// Dissociates a daemon from the given host and data source. The dataSource
// designates a data source from which the deleted hosts were fetched. If it
// is an empty value the hosts from all sources are deleted. The first returned
// value indicates if any row was removed from the local_host table.
func DeleteDaemonsFromHost(dbi dbops.DBI, hostID int64, dataSource HostDataSource) (int64, error) {
	q := dbi.Model((*LocalHost)(nil)).
		Where("host_id = ?", hostID)

	if dataSource.IsSpecified() {
		q = q.Where("data_source = ?", dataSource)
	}

	result, err := q.Delete()
	if err != nil && !errors.Is(err, pg.ErrNoRows) {
		err = pkgerrors.Wrapf(err, "problem deleting the daemons from the host %d and the '%s' data source", hostID, dataSource)
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
		err = pkgerrors.Wrapf(err, "problem deleting orphaned hosts")
		return 0, err
	}
	return int64(result.RowsAffected()), nil
}

// Iterates over the list of hosts and commits them into the database. The hosts
// can be associated with a subnet or can be made global. The committed hosts
// must already include associations with the daemons and other information
// specific to daemons, e.g., DHCP options.
func commitHostsIntoDB(tx *pg.Tx, hosts []Host, subnetID int64) (err error) {
	for i := range hosts {
		hosts[i].SubnetID = subnetID
		if hosts[i].ID == 0 {
			err = addHostWithReferences(tx, &hosts[i])
			if err != nil {
				err = pkgerrors.WithMessagef(err, "unable to add detected host to the database")
				return err
			}
		} else {
			err = updateHostWithLocalHosts(tx, &hosts[i])
			if err != nil {
				err = pkgerrors.WithMessagef(err, "unable to update detected host in the database")
				return err
			}
		}
	}
	return nil
}

// Iterates over the list of hosts and commits them as global hosts.
func CommitGlobalHostsIntoDB(dbi dbops.DBI, hosts []Host) (err error) {
	if db, ok := dbi.(*pg.DB); ok {
		err = db.RunInTransaction(context.Background(), func(tx *pg.Tx) error {
			return commitHostsIntoDB(tx, hosts, 0)
		})
		return
	}
	return commitHostsIntoDB(dbi.(*pg.Tx), hosts, 0)
}

// Iterates over the hosts belonging to the given subnet and stores them
// or updates in the database.
func CommitSubnetHostsIntoDB(tx *pg.Tx, subnet *Subnet) (err error) {
	return commitHostsIntoDB(tx, subnet.Hosts, subnet.ID)
}

// This function checks if the given host includes a reservation for the
// given address.
func (host Host) HasIPAddress(ipAddress string) bool {
	for _, hostAddress := range host.GetIPReservations() {
		hostCidr, err := storkutil.MakeCIDR(hostAddress)
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
	thisIPReservations := host.GetIPReservations()
	otherIPReservations := other.GetIPReservations()

	if len(thisIPReservations) != len(otherIPReservations) {
		return false
	}

	for _, o := range otherIPReservations {
		if !host.HasIPAddress(o) {
			return false
		}
	}

	return true
}

// Checks if two Host instances describe the same host. The host is
// the same when it has equal host identifiers.
func (host Host) IsSame(other *Host) bool {
	if len(host.HostIdentifiers) != len(other.HostIdentifiers) {
		return false
	}

	for _, o := range other.HostIdentifiers {
		if _, ok := host.HasIdentifier(o.Type, o.Value); !ok {
			return false
		}
	}

	return true
}

// Returns local host instance for the daemon ID or nil.
func (host Host) GetLocalHost(daemonID int64) *LocalHost {
	for i := range host.LocalHosts {
		if host.LocalHosts[i].DaemonID == daemonID {
			return &host.LocalHosts[i]
		}
	}
	return nil
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

// Count out-of-pool IPv4 and IPv6 addresses for all subnets.
// Output is a mapping between subnet ID and count.
// The function assumes that the reservation can be only in
// the subnet in which it is defined. If it is outside this
// subnet it is considered out-of-pool, even if it happens to overlap
// with another subnet.
func CountOutOfPoolAddressReservations(dbi dbops.DBI) (map[int64]uint64, error) {
	// Output row.
	// Out-of-pool count per subnet.
	var res []struct {
		SubnetID int64
		// Stork uses the int64 data type for the host reservation ID.
		// It means that we expect at most 2^63-1 out-of-pool reservations.
		Oop uint64
	}

	// Select the unique reserved IP addresses for each host.
	reservedSubquery := dbi.Model((*IPReservation)(nil)).
		Join("LEFT JOIN local_host").JoinOn("ip_reservation.local_host_id = local_host.id").
		Group("local_host.host_id", "ip_reservation.address").
		DistinctOn("local_host.host_id, ip_reservation.address").
		Column("local_host.host_id", "ip_reservation.address")

	// Check if IP reservation address is in any subnet pool
	inAnyPoolSubquery := dbi.Model((*AddressPool)(nil)).
		// We don't need any data from this query, we check only row existence
		ColumnExpr("1").
		Join("JOIN local_subnet").JoinOn("address_pool.local_subnet_id = local_subnet.id").
		// We assume that the reservation can be only in
		// the subnet in which it is defined
		Where("local_subnet.subnet_id = host.subnet_id").
		// Is it in a pool? - from lower to upper bands inclusively
		Where("reserved.address BETWEEN address_pool.lower_bound AND address_pool.upper_bound").
		// We want only to know if the address is in at least one pool
		Limit(1)

	// Find out-of-pool host reservations.
	err := dbi.Model().TableExpr("(?) AS reserved", reservedSubquery).
		Column("host.subnet_id").
		ColumnExpr("COUNT(*) AS oop").
		Join("LEFT JOIN host").JoinOn("reserved.host_id = host.id").
		// Exclude global reservations
		Where("host.subnet_id IS NOT NULL").
		// The IP reservation table contains the address and prefix reservations both.
		// In this query, we check out-of-pool address reservations.
		// We need to exclude prefix reservations. We take into account
		// only IPv4 reservations (as IPv4 has no prefix concept) and
		// single IPv6 hosts - entries with 128 mask length (128 mask length
		// implies that it's an IPv6 address).
		WhereGroup(func(q *pg.Query) (*pg.Query, error) {
			return q.
				Where("family(reserved.address) = 4").
				WhereOr("masklen(reserved.address) = 128"), nil
		}).
		// Is it out of all pools? - Is it not in any pool?
		Where("NOT EXISTS (?)", inAnyPoolSubquery).
		// Group out-of-pool reservations per subnet
		// and count them (in SELECT)
		Group("host.subnet_id").
		Select(&res)
	if err != nil {
		return nil, pkgerrors.Wrap(err, "cannot count out-of-pool addresses")
	}

	countsPerSubnet := make(map[int64]uint64)

	for _, row := range res {
		countsPerSubnet[row.SubnetID] = row.Oop
	}

	return countsPerSubnet, nil
}

// Count out-of-pool prefixes for all subnets.
// Output is a mapping between subnet ID and count.
// The function assumes that the reservation can be only in
// the subnet in which it is defined. If it is outside this
// subnet then it is outside all subnets.
func CountOutOfPoolPrefixReservations(dbi dbops.DBI) (map[int64]uint64, error) {
	// Output row.
	// Out-of-pool count per subnet.
	var res []struct {
		SubnetID int64
		// Stork uses the int64 data type for the host reservation ID.
		// It means that we expect at most 2^63-1 out-of-pool reservations.
		Oop uint64
	}

	// Select the unique reserved IP addresses for each host.
	reservedSubquery := dbi.Model((*IPReservation)(nil)).
		Join("LEFT JOIN local_host").JoinOn("ip_reservation.local_host_id = local_host.id").
		Group("local_host.host_id", "ip_reservation.address").
		DistinctOn("local_host.host_id, ip_reservation.address").
		Column("local_host.host_id", "ip_reservation.address")

	// Check if prefix reservation is in any prefix pool
	inAnyPrefixPoolSubquery := dbi.Model((*PrefixPool)(nil)).
		// We don't need any data from this query, we check only row existence
		ColumnExpr("1").
		Join("JOIN local_subnet").JoinOn("prefix_pool.local_subnet_id = local_subnet.id").
		// We assume that the reservation can be only in
		// the subnet in which it is defined
		Where("local_subnet.subnet_id = host.subnet_id").
		// Reserved prefix is in prefix pool if it is contained by the prefix of the pool
		// and if the reserved prefix length is narrower than the delegation length.
		// For example for pool 3001::/48 and delegation length equals to 64:
		// - Prefix 3001:42::/80 is not in the pool, because it has different prefix.
		// - Prefixes 3001::/48 and 3001::/62 are not in the pool. They are in an expected network
		// (has the same 48 starting bits), but the mask lengths are less then 64.
		// - Prefixes 3001::/64 and 3001::/80 are in the pool. They are in an expected network
		// and the mask lengths are greater or equals 64.
		// The `<<=` is an operator that check if the left CIDR is contained within right CIDR.
		Where("reserved.address <<= prefix_pool.prefix AND masklen(reserved.address) >= prefix_pool.delegated_len").
		// We want only to know if the address is in at least one pool
		Limit(1)

	// Find out-of-pool host reservations.
	err := dbi.Model().TableExpr("(?) AS reserved", reservedSubquery).
		Column("host.subnet_id").
		ColumnExpr("COUNT(*) AS oop").
		Join("LEFT JOIN host").JoinOn("reserved.host_id = host.id").
		// Exclude global reservations
		Where("host.subnet_id IS NOT NULL").
		// The IP reservation table contains the address and prefix reservations both.
		// In this query, we check out-of-pool prefix reservations.
		// We need to exclude address reservations. We take into account
		// only IPv6 reservations (as only IPv6 has prefix concept) and
		// non single IPv6 hosts - entries with mask length less then 128 (128 mask length
		// implies that they are IPv6 addresses).
		Where("family(reserved.address) = 6").
		Where("masklen(reserved.address) != 128").
		// Is it out of all pools? - Is it not in any pool?
		Where("NOT EXISTS (?)", inAnyPrefixPoolSubquery).
		// Group out-of-pool reservations per subnet
		// and count them (in SELECT)
		Group("host.subnet_id").
		Select(&res)
	if err != nil {
		return nil, pkgerrors.Wrap(err, "cannot count out-of-pool prefixes")
	}

	countsPerSubnet := make(map[int64]uint64)

	for _, row := range res {
		countsPerSubnet[row.SubnetID] = row.Oop
	}

	return countsPerSubnet, nil
}

// Count global reservations of IPv4 and IPv6 addresses and delegated prefixes.
// We assume that global reservations are always out-of-pool.
// It's possible to define in-pool global reservation, but it's not recommended.
// The query without this assumption is very inefficient.
func CountGlobalReservations(dbi dbops.DBI) (ipv4Addresses, ipv6Addresses, prefixes uint64, err error) {
	// Output row.
	var res struct {
		Ipv4Addresses uint64
		Ipv6Addresses uint64
		Prefixes      uint64
	}

	// Select the unique reserved IP addresses for each host.
	reservedAddressesSubquery := dbi.Model((*IPReservation)(nil)).
		Join("LEFT JOIN local_host").JoinOn("ip_reservation.local_host_id = local_host.id").
		Group("local_host.host_id", "ip_reservation.address").
		DistinctOn("local_host.host_id, ip_reservation.address").
		Column("local_host.host_id", "ip_reservation.address")

	err = dbi.Model().TableExpr("(?) AS reserved", reservedAddressesSubquery).
		// Window functions aren't supported well by Go-PG
		ColumnExpr("COUNT(reserved.address) FILTER (WHERE family(reserved.address) = 4) AS ipv4_addresses").
		ColumnExpr("COUNT(reserved.address) FILTER (WHERE family(reserved.address) = 6 AND masklen(reserved.address) = 128) AS ipv6_addresses").
		ColumnExpr("COUNT(reserved.address) FILTER (WHERE family(reserved.address) = 6 AND masklen(reserved.address) != 128) AS prefixes").
		Join("LEFT JOIN host").JoinOn("reserved.host_id = host.id").
		// Include only global reservations
		Where("host.subnet_id IS NULL").
		Select(&res)
	err = pkgerrors.Wrap(err, "cannot count global out-of-pool reservations")

	ipv4Addresses = res.Ipv4Addresses
	ipv6Addresses = res.Ipv6Addresses
	prefixes = res.Prefixes
	return
}

// Implementation of the keaconfig.Host interface - used in conversions
// between Host and keaconfig.Reservation.

// Returns host identifiers.
func (host Host) GetHostIdentifiers() (identifiers []struct {
	Type  string
	Value []byte
},
) {
	for _, ids := range host.HostIdentifiers {
		identifiers = append(identifiers, struct {
			Type  string
			Value []byte
		}{
			Type:  ids.Type,
			Value: ids.Value,
		})
	}
	return
}

// Returns reserved IP addresses and prefixes.
// If the same address is reserved on multiple daemons (as usually happens)
// it is returned only once.
func (host Host) GetIPReservations() []string {
	ips := make(map[string]bool)

	for _, lh := range host.LocalHosts {
		for _, r := range lh.IPReservations {
			ips[r.Address] = true
		}
	}

	uniqueIPs := make([]string, 0, len(ips))
	for ip := range ips {
		uniqueIPs = append(uniqueIPs, ip)
	}

	return uniqueIPs
}

// Returns reserved hostname.
// All daemons should have the same hostname reserved for a given identifier
// (host). We assume it is the case and return the hostname from the first
// local host.
func (host Host) GetHostname() string {
	if len(host.LocalHosts) == 0 {
		return ""
	}
	return host.LocalHosts[0].Hostname
}

// Returns reserved client classes.
func (host Host) GetClientClasses(daemonID int64) (clientClasses []string) {
	if lh := host.GetLocalHost(daemonID); lh != nil {
		clientClasses = lh.ClientClasses
	}
	return
}

// Returns reserved next server address.
func (host Host) GetNextServer(daemonID int64) (nextServer string) {
	if lh := host.GetLocalHost(daemonID); lh != nil {
		nextServer = lh.NextServer
	}
	return
}

// Returns reserved server hostname.
func (host Host) GetServerHostname(daemonID int64) (serverHostname string) {
	if lh := host.GetLocalHost(daemonID); lh != nil {
		serverHostname = lh.ServerHostname
	}
	return
}

// Returns reserved boot file name.
func (host Host) GetBootFileName(daemonID int64) (bootFileName string) {
	if lh := host.GetLocalHost(daemonID); lh != nil {
		bootFileName = lh.BootFileName
	}
	return
}

// Returns DHCP options associated with the host and for a specified
// daemon ID.
func (host Host) GetDHCPOptions(daemonID int64) (options []dhcpmodel.DHCPOptionAccessor) {
	for _, lh := range host.LocalHosts {
		if lh.DaemonID == daemonID {
			for _, o := range lh.DHCPOptionSet.Options {
				options = append(options, o)
			}
		}
	}
	return
}

// Returns local subnet ID for a specified daemon. It returns an error
// if the specified daemon is not associated with the host. It returns 0
// if the host is not associated with a subnet (global host reservation case).
func (host Host) GetSubnetID(daemonID int64) (subnetID int64, err error) {
	if host.Subnet != nil {
		for _, ls := range host.Subnet.LocalSubnets {
			if ls.DaemonID == daemonID {
				subnetID = ls.LocalSubnetID
				return
			}
		}
		err = pkgerrors.Errorf("local subnet id not found in host %d for daemon %d", host.ID, daemonID)
	}
	return
}

// Fetches daemon information for each daemon ID within the local hosts.
// The host information can be partial when it is created from the request
// received over the REST API. In particular, the LocalHosts can merely
// contain DaemonID values and the Daemon pointers can be nil. In order
// to initialize Daemon pointers, this function fetches the daemons from
// the database and assigns them to the respective LocalHost instances.
// If any of the daemons does not exist or an error occurs, the host
// is not updated.
func (host Host) PopulateDaemons(dbi dbops.DBI) error {
	var daemons []*Daemon
	for _, lh := range host.LocalHosts {
		// DaemonID is required for this function to run.
		if lh.DaemonID == 0 {
			return pkgerrors.Errorf("problem with populating daemons: host %d lacks daemon ID", host.ID)
		}
		daemon, err := GetDaemonByID(dbi, lh.DaemonID)
		if err != nil {
			return pkgerrors.WithMessage(err, "problem with populating daemons")
		}
		// Daemon does not exist.
		if daemon == nil {
			return pkgerrors.Errorf("problem with populating daemons for host %d: daemon %d does not exist", host.ID, lh.DaemonID)
		}
		daemons = append(daemons, daemon)
	}
	// Everything fine. Assign fetched daemons to the host.
	for i := range host.LocalHosts {
		host.LocalHosts[i].Daemon = daemons[i]
	}
	return nil
}

// Fetches subnet information for a non-zero subnet ID in the host. The
// host information can be partial when it is created from the request
// received over the REST API. This function can be called to initialize
// the Subnet structure in the host with the full information about the
// subnet the host belongs to. This function is no-op when subnet ID is
// 0 or when the Subnet pointer is already non-nil. Otherwise, it fetches
// the relevant subnet information from the database. If the subnet
// doesn't exist, an error is returned.
func (host *Host) PopulateSubnet(dbi dbops.DBI) error {
	if host.SubnetID != 0 && host.Subnet == nil {
		subnet, err := GetSubnet(dbi, host.SubnetID)
		if err != nil {
			return pkgerrors.WithMessagef(err, "problem with populating subnet %d for host %d", host.SubnetID, host.ID)
		}
		if subnet == nil {
			return pkgerrors.Errorf("problem with populating subnet %d for host %d because such subnet does not exist", host.SubnetID, host.ID)
		}
		host.Subnet = subnet
	}
	return nil
}

// Sets LocalHost instance for the Host. If the corresponding LocalHost
// (having the same daemon ID and data source) already exists, it is replaced
// with the specified instance. Otherwise, the instance is appended to the
// slice of LocalHosts.
func (host *Host) AddOrUpdateLocalHost(localHost LocalHost) {
	for i, lh := range host.LocalHosts {
		if lh.DaemonID == localHost.DaemonID && lh.DataSource == localHost.DataSource {
			host.LocalHosts[i] = localHost
			return
		}
	}
	host.LocalHosts = append(host.LocalHosts, localHost)
}

// Combines two hosts into a single host by copying LocalHost data from
// the other host. It returns a boolean value indicating whether or not
// joining the hosts was successful. It returns false when joined hosts
// are not the same ones (have different identifiers, hostnames etc.).
func (host *Host) Join(other *Host) bool {
	if !host.IsSame(other) {
		return false
	}
	for i := range other.LocalHosts {
		host.AddOrUpdateLocalHost(other.LocalHosts[i])
	}
	return true
}
