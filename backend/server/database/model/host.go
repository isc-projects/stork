package dbmodel

import (
	"bytes"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"github.com/go-pg/pg/v9"
	"github.com/go-pg/pg/v9/orm"
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

// This structure reflects a row in the host table. The host may be associated
// with zero, one or multiple IP reservations. It may also be associated with
// one or more identifiers which are used for matching DHCP clients with the
// host.
type Host struct {
	ID        int64 `pg:",pk"`
	CreatedAt time.Time
	SubnetID  int64
	Subnet    *Subnet

	Hostname string

	HostIdentifiers []HostIdentifier
	IPReservations  []IPReservation

	LocalHosts []LocalHost

	// This flag is used to indicate that some changes have been applied to
	// the Host instance locally and that these changes should be applied in
	// the database too. It also indicates that the new app should be
	// associated with the host upon the call to the CommitSubnetHostsIntoDB.
	UpdateOnCommit bool `pg:"-"`
}

// This structure links a host entry stored in the database with an app from
// which it has been retrieved. It provides M:N relationship between hosts
// and apps.
type LocalHost struct {
	AppID      int64 `pg:",pk"`
	HostID     int64 `pg:",pk"`
	App        *App
	Host       *Host
	DataSource string
	UpdateSeq  int64
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

// Adds new host, its reservations and identifiers into the database.
func AddHost(dbIface interface{}, host *Host) error {
	tx, rollback, commit, err := dbops.Transaction(dbIface)
	if err != nil {
		err = pkgerrors.WithMessagef(err, "problem with starting transaction for adding new host")
		return err
	}
	defer rollback()

	// Add the host and fetch its id.
	_, err = tx.Model(host).Insert()
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

	// Everything is fine, commit the changes.
	err = commit()
	if err != nil {
		err = pkgerrors.WithMessagef(err, "problem with committing new host")
	}

	return err
}

// Updates a host, its reservations and identifiers in the database.
func UpdateHost(dbIface interface{}, host *Host) error {
	tx, rollback, commit, err := dbops.Transaction(dbIface)
	if err != nil {
		err = pkgerrors.WithMessagef(err, "problem with starting transaction for updating host with id %d",
			host.ID)
		return err
	}
	defer rollback()

	// Collect updated set of identifiers.
	hostIDTypes := []string{}
	for _, hostID := range host.HostIdentifiers {
		hostIDTypes = append(hostIDTypes, hostID.Type)
	}
	// Delete all existing identifiers for the host which are not present in
	// the new set of identifiers.
	_, err = tx.Model((*HostIdentifier)(nil)).
		Where("host_identifier.host_id = ?", host.ID).
		Where("host_identifier.type NOT IN (?)", pg.In(hostIDTypes)).
		Delete()
	if err != nil {
		err = pkgerrors.Wrapf(err, "problem with deleting host identifiers for host %d", host.ID)
		return err
	}
	// Add or update host identifiers.
	err = addHostIdentifiers(tx, host)
	if err != nil {
		return pkgerrors.WithMessagef(err, "problem with updating host with id %d", host.ID)
	}

	// Delete all existing reservations for the host which are not present in
	// the new set of reservations.
	ipAddresses := []string{}
	for _, resrv := range host.IPReservations {
		ipAddresses = append(ipAddresses, resrv.Address)
	}
	// Delete all existing reservations for the host which are not present in
	// the new set of reservations.
	_, err = tx.Model((*IPReservation)(nil)).
		Where("ip_reservation.host_id = ?", host.ID).
		Where("ip_reservation.address NOT IN (?)", pg.In(ipAddresses)).
		Delete()
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
	_, err = tx.Model(host).WherePK().Update()
	if err != nil {
		err = pkgerrors.Wrapf(err, "problem with updating host with id %d", host.ID)
		return err
	}

	// Everything is fine. Commit the changes.
	err = commit()
	if err != nil {
		err = pkgerrors.WithMessagef(err, "problem with committing updated host with id %d", host.ID)
	}

	return err
}

// Fetch the host by ID.
func GetHost(db *pg.DB, hostID int64) (*Host, error) {
	host := &Host{}
	err := db.Model(host).
		Relation("HostIdentifiers", func(q *orm.Query) (*orm.Query, error) {
			return q.Order("host_identifier.id ASC"), nil
		}).
		Relation("IPReservations", func(q *orm.Query) (*orm.Query, error) {
			return q.Order("ip_reservation.id ASC"), nil
		}).
		Relation("LocalHosts.App").
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
func GetAllHosts(db *pg.DB, family int) ([]Host, error) {
	hosts := []Host{}
	q := db.Model(&hosts).DistinctOn("id")

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
func GetHostsBySubnetID(dbIface interface{}, subnetID int64) ([]Host, error) {
	hosts := []Host{}

	tx, rollback, _, err := dbops.Transaction(dbIface)
	if err != nil {
		err = pkgerrors.WithMessagef(err, "problem with starting transaction for getting hosts for subnet with id %d", subnetID)
		return hosts, err
	}
	defer rollback()

	q := tx.Model(&hosts).
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

	err = q.Select()
	if err != nil {
		if errors.Is(err, pg.ErrNoRows) {
			return nil, nil
		}
		err = pkgerrors.Wrapf(err, "problem with getting hosts by subnet ID %d", subnetID)
		return nil, err
	}
	return hosts, err
}

// Fetches a collection of hosts from the database. The offset and
// limit specify the beginning of the page and the maximum size of the
// page. The appID, if different than 0, is used to fetch hosts whose
// local hosts belong to indicated app. The optional subnetID is used
// to fetch hosts belonging to the particular IPv4 or IPv6 subnet. If
// this value is set to nil all subnets are returned.  The value of 0
// indicates that only global hosts are to be returned. Filtering text
// allows for searching hosts by reserved IP addresses, host identifiers
// specified using hexadecimal digits and hostnames. It is allowed to
// specify colons while searching for hosts by host identifiers. If
// global flag is true then only hosts from the global scope are returned
// (i.e. not assigned to any subnet), if false then only hosts from
// subnets are returned. sortField allows indicating sort column in
// database and sortDir allows selection the order of sorting. If
// sortField is empty then id is used for sorting. If SortDirAny is
// used then ASC order is used.
func GetHostsByPage(db *pg.DB, offset, limit int64, appID int64, subnetID *int64, filterText *string, global *bool, sortField string, sortDir SortDirEnum) ([]Host, int64, error) {
	hosts := []Host{}
	q := db.Model(&hosts)

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
		q = q.Where("lh.app_id = ?", appID)
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
			q = q.WhereOr("text(r.address) LIKE ?", "%"+*filterText+"%").
				WhereOr("i.type::text LIKE ?", "%"+*filterText+"%").
				WhereOr("encode(i.value, 'hex') LIKE ?", "%"+colonlessFilterText+"%").
				WhereOr("host.hostname LIKE ?", "%"+*filterText+"%")
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
		Relation("LocalHosts.App").
		Relation("LocalHosts.App.Machine").
		Relation("LocalHosts.App.AccessPoints")

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
func DeleteHost(db *pg.DB, hostID int64) error {
	host := &Host{
		ID: hostID,
	}
	_, err := db.Model(host).WherePK().Delete()
	if err != nil {
		err = pkgerrors.Wrapf(err, "problem with deleting a host with id %d", hostID)
	}
	return err
}

// Delete associations of the apps with hosts for which the sequence number is
// different than the specified sequence number. These are usually hosts which
// are no longer present in any of the apps monitored by Stork. The dataSource
// is optional and it indicates the sources of hosts to be removed.
func DeleteLocalHostsWithOtherSeq(db *pg.DB, seq int64, dataSource string) error {
	q := db.Model(&LocalHost{}).
		Where("local_host.update_seq != ?", seq)
	if len(dataSource) > 0 {
		q = q.Where("local_host.data_source = ?", dataSource)
	}
	_, err := q.Delete()
	if err != nil {
		err = pkgerrors.Wrapf(err, "problem with deleting associations between apps and hosts for sequence number %d and data source type %s", seq, dataSource)
	}
	return err
}

// Associates an applicatiopn with the host having a specified ID. Internally,
// the association is made via the local_host table which holds information
// about the host from the given app perspective. The source argument
// indicates whether the host information was fetched from the app's configuration
// or via the command.
func AddAppToHost(dbIface interface{}, host *Host, app *App, source string, seq int64) error {
	tx, rollback, commit, err := dbops.Transaction(dbIface)
	if err != nil {
		err = pkgerrors.WithMessagef(err, "problem with starting transaction for associating an app %d with the host %d",
			app.ID, host.ID)
		return err
	}
	defer rollback()

	localHost := LocalHost{
		AppID:      app.ID,
		HostID:     host.ID,
		DataSource: source,
		UpdateSeq:  seq,
	}

	q := tx.Model(&localHost).
		OnConflict("(app_id, host_id) DO UPDATE").
		Set("data_source = EXCLUDED.data_source")

	for _, lh := range host.LocalHosts {
		if lh.AppID == app.ID && lh.UpdateSeq == 0 {
			q = q.Set("update_seq = EXCLUDED.update_seq")
		}
	}

	_, err = q.Insert()
	if err != nil {
		err = pkgerrors.Wrapf(err, "problem with associating the app %d with the host %d",
			app.ID, host.ID)
		return err
	}

	err = commit()
	if err != nil {
		err = pkgerrors.WithMessagef(err, "problem with committing transaction associating the app %d with the host %d",
			app.ID, host.ID)
	}
	return err
}

// Iterates over the list of hosts and commits them into the database. The hosts
// can be associated with a subnet or can be made global.
func commitHostsIntoDB(tx *pg.Tx, hosts []Host, subnetID int64, app *App, source string, seq int64) (err error) {
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
			err = AddAppToHost(tx, &hosts[i], app, source, seq)
			if err != nil {
				err = pkgerrors.WithMessagef(err, "unable to associate detected host with Kea app having id %d",
					app.ID)
				return err
			}
		}
	}
	return nil
}

// Iterates over the list of hosts and commits them as global hosts.
func CommitGlobalHostsIntoDB(tx *pg.Tx, hosts []Host, app *App, source string, seq int64) (err error) {
	return commitHostsIntoDB(tx, hosts, 0, app, source, seq)
}

// Iterates over the hosts belonging to the given subnet and stores them
// or updates in the database.
func CommitSubnetHostsIntoDB(tx *pg.Tx, subnet *Subnet, app *App, source string, seq int64) (err error) {
	return commitHostsIntoDB(tx, subnet.Hosts, subnet.ID, app, source, seq)
}

// This function returns next value of the bulk_update_seq. The returned value
// is used during bulk updates of data within a database. For example, when
// inserting or updating many host reservations, sequence value for all host
// reservations tha have been inserted or updated must be set to the same
// sequence value. This allows for distinguishing the hosts that haven't been
// updated. In such case, they can be removed.
// todo: This function may be used to a separate file. For now it is here, because
// the only supported use case is for hosts.
func GetNextBulkUpdateSeq(db *dbops.PgDB) (seq int64, err error) {
	_, err = db.QueryOne(pg.Scan(&seq), "SELECT nextval('bulk_update_seq')")
	if err != nil {
		err = pkgerrors.Wrapf(err, "problem with getting next bulk update sequence number")
	}
	return seq, err
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
