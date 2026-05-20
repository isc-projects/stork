package dbmodel

import (
	"context"

	"github.com/go-pg/pg/v10"
	"github.com/pkg/errors"
	dbops "isc.org/stork/server/database"
)

// Represents a relation between the machine IP address and other tables.
type MachineIPAddressRelation string

const (
	MachineIPAddressRelationMachine MachineIPAddressRelation = "Machine"
)

// Represents a single IP address detected on the machine.
type MachineIPAddress struct {
	ID        int64
	MachineID int64
	IPAddress string

	Machine *Machine `pg:"rel:has-one"`
}

// Updates IP addresses detected on the given machine in the transaction.
func upsertMachineIPAddresses(tx *pg.Tx, machineID int64, ipAddresses ...string) error {
	q := tx.Model(&MachineIPAddress{}).
		Where("machine_id = ?", machineID)
	if len(ipAddresses) > 0 {
		q = q.Where("ip_address NOT IN (?)", pg.In(ipAddresses))
	}
	_, err := q.Delete()
	if err != nil {
		return errors.Wrapf(err, "problem deleting IP addresses for machine %d", machineID)
	}
	if len(ipAddresses) == 0 {
		return nil
	}
	rows := make([]*MachineIPAddress, len(ipAddresses))
	for i, ipAddress := range ipAddresses {
		rows[i] = &MachineIPAddress{
			MachineID: machineID,
			IPAddress: ipAddress,
		}
	}
	_, err = tx.Model(&rows).OnConflict("DO NOTHING").Insert()
	return errors.Wrapf(err, "problem inserting IP addresses for machine %d", machineID)
}

// Updates IP addresses detected on the given machine. It removes IP addresses that
// are not in the list of new IP addresses. It inserts new IP addresses that are not
// already present in the database. It preserves IP addresses in the database that
// are present in the list of new IP addresses. It starts new transaction if the
// transaction is not already started. It uses the existing transaction otherwise.
func UpsertMachineIPAddresses(dbi dbops.DBI, machineID int64, ipAddresses ...string) error {
	if db, ok := dbi.(*pg.DB); ok {
		return db.RunInTransaction(context.Background(), func(tx *pg.Tx) error {
			return upsertMachineIPAddresses(tx, machineID, ipAddresses...)
		})
	}
	return upsertMachineIPAddresses(dbi.(*pg.Tx), machineID, ipAddresses...)
}

// Returns all IP addresses detected on all machines.
func GetMachineIPAddresses(db *pg.DB, relations ...MachineIPAddressRelation) ([]MachineIPAddress, error) {
	var machineIPAddresses []MachineIPAddress
	q := db.Model(&machineIPAddresses)
	for _, relation := range relations {
		q = q.Relation(string(relation))
	}
	// Order by IP addresses.
	ordExpr, _ := prepareOrderAndDistinctExpr("machine_ip_address", "ip_address", SortDirAsc, nil)
	q = q.OrderExpr(ordExpr)

	// Select IP addresses.
	err := q.Select()
	if err != nil && !errors.Is(err, pg.ErrNoRows) {
		return nil, errors.Wrapf(err, "problem getting IP addresses for all machines")
	}
	return machineIPAddresses, nil
}
