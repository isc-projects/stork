package dbmodel

import (
	"fmt"
	"hash/adler32"

	"github.com/go-pg/pg/v10"
	"github.com/pkg/errors"
	"isc.org/stork/datamodel/daemonname"
)

// TODO: Code implemented in this file is a temporary solution for virtual applications.

// The virtual app is a solution to preserve backward compatibility with the
// legacy REST API. It mimics a legacy App instance for a given daemon.
type VirtualApp struct {
	ID   int64
	Name string
	Type VirtualAppType
}

type VirtualAppType string

const (
	VirtualAppTypeKea   VirtualAppType = "kea"
	VirtualAppTypeBind9 VirtualAppType = "bind9"
	VirtualAppTypePDNS  VirtualAppType = "pdns"
)

// Returns a virtual app for a given daemon.
func (d Daemon) GetVirtualApp() *VirtualApp {
	var appID int64
	if accessPoint, err := d.GetAccessPoint(AccessPointControl); err == nil {
		appID, _ = getVirtualAppID(accessPoint, d.MachineID)
	}

	var appType VirtualAppType
	switch {
	case d.Name.IsKea():
		appType = VirtualAppTypeKea
	case d.Name == daemonname.Bind9:
		appType = VirtualAppTypeBind9
	case d.Name == daemonname.PDNS:
		appType = VirtualAppTypePDNS
	}

	var appName string
	if d.Machine != nil {
		appName = fmt.Sprintf("%s@%s%%%d", appType, d.Machine.Address, appID)
	} else {
		appName = fmt.Sprintf("%s%%%d", appType, appID)
	}

	return &VirtualApp{
		ID:   appID,
		Name: appName,
		Type: appType,
	}
}

// Returns a virtual app ID for a given access point and machine.
// The app ID can be derived only for control access points.
//
// The app ID must be deterministic, be the same for the daemons from
// the same machine and having the same control access point and unique
// in other cases.
func getVirtualAppID(accessPoint *AccessPoint, machineID int64) (int64, error) {
	if accessPoint.Type != AccessPointControl {
		return 0, errors.Errorf(
			"cannot derive virtual app ID for non-control access point: %v",
			accessPoint,
		)
	}

	checksum := adler32.Checksum([]byte(fmt.Sprintf(
		"%d:%s:%d",
		machineID,
		accessPoint.Address,
		accessPoint.Port,
	)))
	return int64(checksum), nil
}

// Returns daemons generating a given (virtual) app ID. The app ID is derived
// from the control access point of the daemon. If multiple daemons share the
// same control access point, they will have the same app ID and all of them
// will be returned.
func GetDaemonsByVirtualAppID(dbi pg.DBI, appID int64) (daemons []*Daemon, err error) {
	var accessPoints []AccessPoint
	err = dbi.Model(&accessPoints).
		Relation("Daemon").
		Where("type = ?", AccessPointControl).
		Select()
	if err != nil {
		return nil, errors.Wrapf(err, "problem selecting control access points")
	}

	var matchingDaemonIDs []int64
	for _, ap := range accessPoints {
		actualAppID, err := getVirtualAppID(&ap, ap.Daemon.MachineID)
		if err != nil {
			return nil, err
		}

		if actualAppID == appID {
			matchingDaemonIDs = append(matchingDaemonIDs, ap.DaemonID)
		}
	}

	if len(matchingDaemonIDs) == 0 {
		// No matching access points, return empty result.
		return []*Daemon{}, nil
	}

	err = dbi.Model(&daemons).
		Relation(DaemonRelationAccessPoints).
		Relation(DaemonRelationMachine).
		Relation(DaemonRelationKeaDHCPDaemon).
		Relation(DaemonRelationBind9Daemon).
		Relation(DaemonRelationPDNSDaemon).
		Where("daemon.id IN (?)", pg.In(matchingDaemonIDs)).
		OrderExpr("daemon.id ASC").
		Select()

	if errors.Is(err, pg.ErrNoRows) {
		return daemons, nil
	} else if err != nil {
		return nil, errors.Wrapf(err, "problem selecting daemons for virtual app ID %d", appID)
	}

	return daemons, nil
}

// Returns a machine ID related to the virtual app.
// Returns zero if no matching machine is found.
func GetMachineIDByVirtualAppID(dbi pg.DBI, appID int64) (machineID int64, err error) {
	var accessPoints []AccessPoint
	err = dbi.Model(&accessPoints).
		Relation("Daemon").
		Where("type = ?", AccessPointControl).
		Select()
	if err != nil {
		return 0, errors.Wrapf(err, "problem selecting control access points")
	}

	var matchingDaemonID int64
	for _, ap := range accessPoints {
		actualAppID, err := getVirtualAppID(&ap, ap.Daemon.MachineID)
		if err != nil {
			return 0, err
		}

		if actualAppID == appID {
			matchingDaemonID = ap.DaemonID
			break
		}
	}

	if matchingDaemonID == 0 {
		// No matching access point, return empty result.
		return 0, nil
	}

	err = dbi.Model((*Daemon)(nil)).
		Where("daemon.id = ?", matchingDaemonID).
		Column("daemon.machine_id").
		Select(&machineID)

	if errors.Is(err, pg.ErrNoRows) {
		return 0, nil
	} else if err != nil {
		return 0, errors.Wrapf(err, "problem selecting machine ID for virtual app ID %d", appID)
	}

	return machineID, err
}
