package dbmodel

import (
	"time"

	"github.com/go-pg/pg/v9"
	"github.com/go-pg/pg/v9/orm"
	errors "github.com/pkg/errors"

	dbops "isc.org/stork/server/database"
)

// This structure reflects a row in the host_identifier table. It includes
// a value and type of the identifier used to match the client with a host. The
// following types are available: hw-address, duid, circuit-id, client-id
// and flex-id (same as those available in Kea).
type HostIdentifier struct {
	ID     int64
	Type   string `pg:"id_type"`
	Value  []byte `pg:"id_value"`
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
	ID           int64 `pg:",pk"`
	CreatedAt time.Time
	SubnetID  int64

	HostIdentifiers []HostIdentifier
	IPReservations  []IPReservation
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
}

// Associates a host with DHCP with host identifiers.
func addHostIdentifiers(tx *pg.Tx, host *Host) error {
	for i, id := range host.HostIdentifiers {
		identifier := id
		identifier.HostID = host.ID
		_, err := tx.Model(&identifier).
			OnConflict("(id_type, host_id) DO UPDATE").
			Set("id_value = EXCLUDED.id_value").
			Insert()
		if err != nil {
			err = errors.Wrapf(err, "problem with adding host identifier with type %s for host with id %d",
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
			err = errors.Wrapf(err, "problem with adding IP reservation %s for host with id %d",
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
		err = errors.WithMessagef(err, "problem with starting transaction for adding new host")
		return err
	}
	defer rollback()

	// Add the host and fetch its id.
	_, err = tx.Model(host).Insert()
	if err != nil {
		err = errors.Wrapf(err, "problem with adding new host")
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
		err = errors.WithMessagef(err, "problem with committing new host")
	}

	return err
}

// Updates a host, its reservations and identifiers in the database.
func UpdateHost(dbIface interface{}, host *Host) error {
	tx, rollback, commit, err := dbops.Transaction(dbIface)
	if err != nil {
		err = errors.WithMessagef(err, "problem with starting transaction for updating host with id %d",
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
		Where("host_identifier.id_type NOT IN (?)", pg.In(hostIDTypes)).
		Delete()
	if err != nil {
		err = errors.Wrapf(err, "problem with deleting host identifiers for host %d", host.ID)
		return err
	}
	// Add or update host identifiers.
	err = addHostIdentifiers(tx, host)
	if err != nil {
		return errors.WithMessagef(err, "problem with updating host with id %d", host.ID)
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
		err = errors.Wrapf(err, "problem with deleting IP reservations for host %d", host.ID)
		return err
	}
	// Add or update host reservations.
	err = addIPReservations(tx, host)
	if err != nil {
		return errors.WithMessagef(err, "problem with updating host with id %d", host.ID)
	}

	// Update the host information.
	_, err = tx.Model(host).WherePK().Update()
	if err != nil {
		err = errors.Wrapf(err, "problem with updating host with id %d", host.ID)
		return err
	}

	// Everything is fine. Commit the changes.
	err = commit()
	if err != nil {
		err = errors.WithMessagef(err, "problem with committing updated host with id %d", host.ID)
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
		Where("host.id = ?", hostID).
		Select()

	if err != nil {
		if err == pg.ErrNoRows {
			return nil, nil
		}
		err = errors.Wrapf(err, "problem with getting a host with id %d", hostID)
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
		OrderExpr("id ASC")

	err := q.Select()

	if err != nil {
		if err == pg.ErrNoRows {
			return nil, nil
		}
		err = errors.Wrapf(err, "problem with getting all hosts for family %d", family)
		return nil, err
	}
	return hosts, err
}

// Delete host, host identifiers and reservations by id.
func DeleteHost(db *pg.DB, hostID int64) error {
	host := &Host{
		ID: hostID,
	}
	_, err := db.Model(host).WherePK().Delete()
	if err != nil {
		err = errors.Wrapf(err, "problem with deleting a host with id %d", hostID)
	}
	return err
}
