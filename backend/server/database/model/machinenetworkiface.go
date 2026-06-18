package dbmodel

import (
	"context"
	"slices"

	"github.com/go-pg/pg/v10"
	"github.com/pkg/errors"
	dbops "isc.org/stork/server/database"
)

// Represents a relation between an IP address on the machine and other tables.
type MachineNetworkInterfaceIPAddressRelation string

const (
	MachineNetworkInterfaceIPAddressRelationInterface MachineNetworkInterfaceIPAddressRelation = "Interface"
	MachineNetworkInterfaceIPAddressRelationMachine   MachineNetworkInterfaceIPAddressRelation = "Interface.Machine"
)

// Represents a single IP address detected on the machine's network interface.
type MachineNetworkInterfaceIPAddress struct {
	MachineNetworkInterfaceID int64  `pg:",pk"`
	IPAddress                 string `pg:",pk"`

	Interface *MachineNetworkInterface `pg:"machine_network_interface,rel:has-one,fk:machine_network_interface_id"`
}

// Represents a single network interface detected on the machine.
type MachineNetworkInterface struct {
	// Interface ID.
	ID int64
	// Machine ID.
	MachineID int64
	// Interface name.
	Name string
	// Bitwise combination of interface flags as described in https://pkg.go.dev/net#Flags.
	// Use net.Flags constants to test the flags (e.g., whether the interface is up or down).
	Flags uint32 `pg:",use_zero"`
	// Interface hardware address (e.g. MAC address).
	HardwareAddress []byte
	// List of IP addresses assigned to the interface.
	IPAddresses []MachineNetworkInterfaceIPAddress `pg:"rel:has-many"`
	// Machine associated with the interface.
	Machine *Machine `pg:"rel:has-one"`
}

// Updates network interfaces detected on the given machine, and the IP addresses
// associated with them. It starts new transaction if the transaction is not already
// started. It uses the existing transaction otherwise.
func upsertMachineNetworkInterfaces(tx *pg.Tx, machineID int64, interfaces ...MachineNetworkInterface) error {
	// Remove interfaces associated with the machine that are not
	// in the list of new interfaces. It will also remove the IP addresses
	// associated with them.
	q := tx.Model(&MachineNetworkInterface{}).
		Where("machine_id = ?", machineID)
	if len(interfaces) > 0 {
		// Delete the interfaces that are not in the list of new interfaces.
		var ifacesNames []string
		for _, iface := range interfaces {
			ifacesNames = append(ifacesNames, iface.Name)
		}
		q = q.Where("name NOT IN (?)", pg.In(ifacesNames))
	}
	_, err := q.Delete()
	if err != nil {
		return errors.Wrapf(err, "problem deleting network interfaces for machine %d", machineID)
	}
	// If the list of new interfaces is empty, there is nothing to do.
	if len(interfaces) == 0 {
		return nil
	}
	// Insert new interfaces and update the existing ones, if the hardware address or
	// flags changed.
	ifaces := make([]*MachineNetworkInterface, len(interfaces))
	for i, iface := range interfaces {
		ifaces[i] = &MachineNetworkInterface{
			MachineID:       machineID,
			Name:            iface.Name,
			Flags:           iface.Flags,
			HardwareAddress: iface.HardwareAddress,
			IPAddresses:     iface.IPAddresses,
		}
	}
	_, err = tx.Model(&ifaces).OnConflict("(machine_id, name) DO UPDATE").
		Set("flags = EXCLUDED.flags", "hardware_address = EXCLUDED.hardware_address").Insert()
	if err != nil {
		return errors.Wrapf(err, "problem inserting or updating network interfaces for machine %d", machineID)
	}

	var ipAddresses []MachineNetworkInterfaceIPAddress
	for _, iface := range ifaces {
		addrs := make([]string, len(iface.IPAddresses))
		for i, addr := range iface.IPAddresses {
			addrs[i] = addr.IPAddress
		}
		// Delete IP addresses that are not in the list of new IP addresses
		// for the given interface.
		query := tx.Model((*MachineNetworkInterfaceIPAddress)(nil)).
			Where("machine_network_interface_id = ?", iface.ID)

		if len(addrs) > 0 {
			query = query.Where("ip_address NOT IN (?)", pg.In(addrs))
		}
		_, err := query.Delete()
		if err != nil {
			return errors.Wrapf(err, "problem deleting IP addresses for machine network interface %d", iface.ID)
		}

		// Insert new IP addresses for the given interface.
		for _, addr := range iface.IPAddresses {
			ipAddresses = append(ipAddresses, MachineNetworkInterfaceIPAddress{
				MachineNetworkInterfaceID: iface.ID,
				IPAddress:                 addr.IPAddress,
			})
		}
	}

	if len(ipAddresses) > 0 {
		_, err = tx.Model(&ipAddresses).OnConflict("(machine_network_interface_id, ip_address) DO NOTHING").
			Insert()
		err = errors.Wrapf(err, "problem inserting host interfaces for machine %d", machineID)
	}
	return err
}

// Updates interfaces detected on the given machine. It removes the interfaces that
// are not in the list of new interfaces. It inserts new interfaces that are not
// already present in the database. It preserves IP addresses in the database that
// are present in the list of new IP addresses. It starts new transaction if the
// transaction is not already started. It uses the existing transaction otherwise.
func UpsertMachineNetworkInterfaces(dbi dbops.DBI, machineID int64, interfaces ...MachineNetworkInterface) error {
	if db, ok := dbi.(*pg.DB); ok {
		return db.RunInTransaction(context.Background(), func(tx *pg.Tx) error {
			return upsertMachineNetworkInterfaces(tx, machineID, interfaces...)
		})
	}
	return upsertMachineNetworkInterfaces(dbi.(*pg.Tx), machineID, interfaces...)
}

// Returns all IP addresses detected on all machines.
func GetMachineNetworkInterfaceIPAddresses(db *pg.DB, relations ...MachineNetworkInterfaceIPAddressRelation) ([]MachineNetworkInterfaceIPAddress, error) {
	var ipAddresses []MachineNetworkInterfaceIPAddress
	q := db.Model(&ipAddresses)
	for _, relation := range relations {
		q = q.Relation(string(relation))
	}
	// Order by IP addresses.
	q = q.OrderExpr("ip_address ASC")

	// Select IP addresses.
	err := q.Select()
	if err != nil && !errors.Is(err, pg.ErrNoRows) {
		return nil, errors.Wrapf(err, "problem getting host IP addresses for all machines")
	}
	return ipAddresses, nil
}

// Returns all machines having the given IP address on one of their network interfaces.
// Optionally, it can return interfaces and IP addresses associated with the machines.
// However, joining this extra information affects query performance, and should be
// avoided if possible.
func GetMachinesByNetworkInterfaceIPAddress(db *pg.DB, ipAddress string, relations ...MachineRelation) ([]Machine, error) {
	var machines []Machine
	q := db.Model(&machines).
		Join("JOIN machine_network_interface AS mni").
		JoinOn("machine.id = mni.machine_id").
		Join("JOIN machine_network_interface_ip_address AS ip").
		JoinOn("ip.machine_network_interface_id = mni.id").
		Where("ip.ip_address = ?", ipAddress)

	for _, relation := range relations {
		q = q.Relation(string(relation))
	}

	q = q.OrderExpr("machine.id ASC")

	if slices.Contains(relations, MachineRelationNetworkInterfacesIPAddresses) {
		q = q.OrderExpr("mni.name ASC")
		q = q.OrderExpr("ip.ip_address ASC")
	} else if slices.Contains(relations, MachineRelationNetworkInterfaces) {
		q = q.OrderExpr("mni.name ASC")
	}

	err := q.Select()
	if err != nil && !errors.Is(err, pg.ErrNoRows) {
		return nil, errors.Wrapf(err, "problem getting machines by IP address %s", ipAddress)
	}
	return machines, nil
}
