package dbmodeltest

import (
	"isc.org/stork/datamodel/daemonname"
	"isc.org/stork/datamodel/protocoltype"
	dbmodel "isc.org/stork/server/database/model"
)

// A wrapper for a BIND 9 daemon.
type Bind9Server struct {
	Daemon
}

// A wrapper for a PowerDNS daemon.
type PowerDNSServer struct {
	Daemon
}

// Creates new BIND 9 daemon instance in the machine.
func (m *Machine) NewBind9Daemon() (*Bind9Server, error) {
	ap := []*dbmodel.AccessPoint{
		{
			Type:     dbmodel.AccessPointControl,
			Address:  "localhost",
			Port:     int64(getRandInt31()),
			Key:      "",
			Protocol: protocoltype.RNDC,
		},
		{
			Type:     dbmodel.AccessPointStatistics,
			Address:  "localhost",
			Port:     int64(getRandInt31()),
			Key:      "",
			Protocol: protocoltype.HTTPS,
		},
	}

	daemon := dbmodel.NewDaemon(&dbmodel.Machine{ID: m.ID}, daemonname.Bind9, true, ap)
	if err := dbmodel.AddDaemon(m.db, daemon); err != nil {
		return nil, err
	}

	return &Bind9Server{
		Daemon: Daemon{
			machine:  m,
			DaemonID: daemon.ID,
		},
	}, nil
}

// Creates a new PowerDNS daemon instance in the machine.
func (m *Machine) NewPowerDNSServer() (*PowerDNSServer, error) {
	ap := []*dbmodel.AccessPoint{{
		Type:     dbmodel.AccessPointControl,
		Address:  "localhost",
		Port:     int64(getRandInt31()),
		Key:      "",
		Protocol: protocoltype.HTTPS,
	}}

	daemon := dbmodel.NewDaemon(&dbmodel.Machine{ID: m.ID}, daemonname.PDNS, true, ap)
	if err := dbmodel.AddDaemon(m.db, daemon); err != nil {
		return nil, err
	}

	return &PowerDNSServer{
		Daemon{
			machine:  m,
			DaemonID: daemon.ID,
		},
	}, nil
}
