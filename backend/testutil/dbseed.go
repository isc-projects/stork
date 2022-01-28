package testutil

import (
	"fmt"
	"math"

	"github.com/go-pg/pg/v10"
	"github.com/pkg/errors"
	dbmodel "isc.org/stork/server/database/model"
)

// Configuration of the seed function with number of entries to generate.
type SeedConfig struct {
	Machines int
	// Number of apps per machine
	Apps int
	// Number of IPv4 subnets per app
	SubnetsV4 int
	// Number of IPv6 subnets per app
	SubnetsV6 int
	// Number of daemons per app
	Daemons int
	// Number of in-pool host reservations per subnet
	HostReservationsInPool int
	// Number of out-of-pool host reservations per subnet
	HostReservationsOutOfPool int

	// HostReservationsGlobalInPool      int
	// HostReservationsGlobalOutOfPool   int

	// Number of in-pool prefix reservations per IPv6 subnet
	PrefixReservationsInPool int
	// Number of out-of-pool prefix reservations per IPv6 subnet
	PrefixReservationsOutOfPool int

	// PrefixReservationsGlobalInPool    int
	// PrefixReservationsGlobalOutOfPool int
}

// The generator is simple and for is only for development use.
// It means that it has some limitations.
// This function returns false if the generated entries will be for sure invalid.
// But true output doesn't guarantee that results are valid.
// Be careful with huge generation inputs.
func (c *SeedConfig) validate() (string, bool) {
	if c.HostReservationsInPool > 100 {
		return "too many in-pool host reservations", false
	}

	if c.HostReservationsOutOfPool > 155 {
		return "too many out-of-pool host reservations", false
	}

	if c.PrefixReservationsInPool > 255 {
		return "too many in-pool prefix reservations", false
	}

	if c.PrefixReservationsOutOfPool > 255 {
		return "too many out-of-pool prefix reservations", false
	}

	if c.Machines*c.Apps*int(math.Max(float64(c.SubnetsV4), float64(c.SubnetsV6))) > 255*255 {
		return "too many total subnets", false
	}

	return "", true
}

// Helper structure to store index of the iteration.
type index struct {
	item  int
	total int
}

// Constructor of the index structure.
func newIndex(item, total int) *index {
	return &index{item, total}
}

// Combine two indices into one.
// It is useful to obtain unique index in multi-nested for loop.
// The combination order is important. The caller index should
// be higher-order then an argument index.
func (i *index) combine(other *index) *index {
	return newIndex(
		i.item*other.total+other.item,
		i.total*other.total,
	)
}

// Generate fake data and save them into database.
// Helper function to perform performance tests.
func Seed(db *pg.DB, config *SeedConfig) error {
	if msg, ok := config.validate(); !ok {
		return errors.Errorf("Configuration exceed the generator limitations: %s", msg)
	}
	for machineI := 0; machineI < config.Machines; machineI++ {
		machine := dbmodel.Machine{
			Address:    fmt.Sprintf("machine-%d", machineI),
			AgentPort:  int64(1024 + machineI),
			Authorized: true,
		}
		machineIdx := newIndex(machineI, config.Machines)

		if err := dbmodel.AddMachine(db, &machine); err != nil {
			return err
		}

		for appI := 0; appI < config.Apps; appI++ {
			appIdx := machineIdx.combine(newIndex(appI, config.Apps))

			accessPoint := &dbmodel.AccessPoint{
				Address: fmt.Sprintf("access-point-%d", appIdx.item),
				Port:    int64(1024 + appI),
				Type:    "control",
			}

			var daemons []*dbmodel.Daemon

			for daemonI := 0; daemonI < config.Daemons; daemonI++ {
				daemonIdx := appIdx.combine(newIndex(daemonI, config.Daemons))
				daemon := &dbmodel.Daemon{
					Pid:       int32(10 + daemonI),
					Name:      fmt.Sprintf("daemon-v4-%d", daemonIdx.item),
					Active:    true,
					Monitored: true,
					Uptime:    int64(daemonIdx.item),
				}
				daemons = append(daemons, daemon)
			}

			app := &dbmodel.App{
				MachineID:    machine.ID,
				Active:       true,
				Type:         dbmodel.AppTypeKea,
				Name:         fmt.Sprintf("app-%d", appIdx),
				AccessPoints: []*dbmodel.AccessPoint{accessPoint},
				Daemons:      daemons,
			}

			_, err := dbmodel.AddApp(db, app)
			if err != nil {
				return err
			}

			for subnetI := 0; subnetI < config.SubnetsV4; subnetI++ {
				subnetIdx := appIdx.combine(newIndex(subnetI, config.SubnetsV4))
				subnetPrefix := fmt.Sprintf("10.%d.%d", (subnetIdx.item-subnetIdx.item%256)/256, subnetIdx.item%256)
				subnet := &dbmodel.Subnet{
					Prefix:      fmt.Sprintf("%s.0/24", subnetPrefix),
					ClientClass: fmt.Sprintf("class-%02d-%02d", (subnetIdx.item-subnetIdx.item%100)/100, subnetIdx.item%100),
					AddressPools: []dbmodel.AddressPool{
						{
							LowerBound: fmt.Sprintf("%s.1", subnetPrefix),
							UpperBound: fmt.Sprintf("%s.101", subnetPrefix),
						},
					},
					AddrUtilization: int16(subnetIdx.item % 1000),
				}
				if err := dbmodel.AddSubnet(db, subnet); err != nil {
					return err
				}

				var reservations []dbmodel.IPReservation

				for reservationIdx := 0; reservationIdx < config.HostReservationsInPool; reservationIdx++ {
					reservation := dbmodel.IPReservation{
						Address: fmt.Sprintf("%s.%d", subnetPrefix, reservationIdx+1),
					}
					reservations = append(reservations, reservation)
				}

				for reservationIdx := 0; reservationIdx < config.HostReservationsOutOfPool; reservationIdx++ {
					reservation := dbmodel.IPReservation{
						Address: fmt.Sprintf("%s.%d", subnetPrefix, 102+reservationIdx),
					}
					reservations = append(reservations, reservation)
				}

				host := &dbmodel.Host{
					SubnetID:       subnet.ID,
					Hostname:       fmt.Sprintf("host-v4-%d", subnetIdx.item),
					IPReservations: reservations,
				}

				if err := dbmodel.AddHost(db, host); err != nil {
					return err
				}
			}

			for subnetI := 0; subnetI < config.SubnetsV6; subnetI++ {
				subnetIdx := appIdx.combine(newIndex(subnetI, config.SubnetsV6))
				subnetPrefix := fmt.Sprintf("30:%02x:%02x", (subnetIdx.item-subnetIdx.item%256)/256, subnetIdx.item%256)
				subnet := &dbmodel.Subnet{
					Prefix:      fmt.Sprintf("%s::/64", subnetPrefix),
					ClientClass: fmt.Sprintf("class-%02d-%02d", (subnetIdx.item-subnetIdx.item%100)/100, subnetIdx.item%100),
					AddressPools: []dbmodel.AddressPool{
						{
							LowerBound: fmt.Sprintf("%s::1", subnetPrefix),
							UpperBound: fmt.Sprintf("%s::101", subnetPrefix),
						},
					},
					PrefixPools: []dbmodel.PrefixPool{
						{
							Prefix:       fmt.Sprintf("%s:FF::/64", subnetPrefix),
							DelegatedLen: 80,
						},
					},
					AddrUtilization: int16(subnetIdx.item % 1000),
					PdUtilization:   int16(subnetIdx.item % 1000),
				}
				if err := dbmodel.AddSubnet(db, subnet); err != nil {
					return err
				}

				var reservations []dbmodel.IPReservation

				for reservationIdx := 0; reservationIdx < config.HostReservationsInPool; reservationIdx++ {
					reservation := dbmodel.IPReservation{
						Address: fmt.Sprintf("%s::%d", subnetPrefix, reservationIdx+1),
					}
					reservations = append(reservations, reservation)
				}

				for reservationIdx := 0; reservationIdx < config.HostReservationsOutOfPool; reservationIdx++ {
					reservation := dbmodel.IPReservation{
						Address: fmt.Sprintf("%s::%d", subnetPrefix, 102+reservationIdx),
					}
					reservations = append(reservations, reservation)
				}

				for reservationIdx := 0; reservationIdx < config.PrefixReservationsInPool; reservationIdx++ {
					reservation := dbmodel.IPReservation{
						Address: fmt.Sprintf("%s:FF:%02d::/80", subnetPrefix, reservationIdx+1),
					}
					reservations = append(reservations, reservation)
				}

				for reservationIdx := 0; reservationIdx < config.PrefixReservationsOutOfPool; reservationIdx++ {
					reservation := dbmodel.IPReservation{
						Address: fmt.Sprintf("%s:EE:%02d::/80", subnetPrefix, reservationIdx+1),
					}
					reservations = append(reservations, reservation)
				}

				host := &dbmodel.Host{
					SubnetID:       subnet.ID,
					Hostname:       fmt.Sprintf("host-v6-%d", subnetIdx.item),
					IPReservations: reservations,
				}

				if err := dbmodel.AddHost(db, host); err != nil {
					return err
				}
			}
		}
	}

	return nil
}
