package configreview

import (
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	keaconfig "isc.org/stork/appcfg/kea"
	dbmodel "isc.org/stork/server/database/model"
	storkutil "isc.org/stork/util"
)

// The checker verifying if the stat_cmds hooks library is loaded.
func statCmdsPresence(ctx *ReviewContext) (*Report, error) {
	config := ctx.subjectDaemon.KeaDaemon.Config
	if _, _, present := config.GetHooksLibrary("libdhcp_stat_cmds"); !present {
		r, err := NewReport(ctx, "The Kea Statistics Commands library (libdhcp_stat_cmds) provides commands for retrieving accurate DHCP lease statistics for Kea DHCP servers. Stork sends these commands to fetch lease statistics displayed in the dashboard, subnet, and shared-network views. Stork found that {daemon} is not using this hook library. Some statistics will not be available until the library is loaded.").
			referencingDaemon(ctx.subjectDaemon).
			create()
		return r, err
	}
	return nil, nil
}

// The checker verifying if the host_cmds hooks library is loaded when
// host backend is in use.
func hostCmdsPresence(ctx *ReviewContext) (*Report, error) {
	config := ctx.subjectDaemon.KeaDaemon.Config
	if _, _, present := config.GetHooksLibrary("libdhcp_host_cmds"); !present {
		databases := config.GetAllDatabases()
		if len(databases.Hosts) > 0 {
			r, err := NewReport(ctx, "Kea can be configured to store host reservations in a database. Stork can access these reservations using the commands implemented in the Host Commands hook library and make them available in the Host Reservations view. It appears that the libdhcp_host_cmds hook library is not loaded on {daemon}. Host reservations from the database will not be visible in Stork until this library is enabled.").
				referencingDaemon(ctx.subjectDaemon).
				create()
			return r, err
		}
	}
	return nil, nil
}

// The checker verifying if a shared network can be removed because it
// is empty or contains only one subnet.
func sharedNetworkDispensable(ctx *ReviewContext) (*Report, error) {
	config := ctx.subjectDaemon.KeaDaemon.Config
	// Create a structure into which the shared networks will be decoded.
	decodedSharedNetworks := &[]struct {
		Name    string
		Subnet4 []struct {
			Subnet string
		}
		Subnet6 []struct {
			Subnet string
		}
	}{}
	// Parse shared-networks list.
	err := config.DecodeSharedNetworks(decodedSharedNetworks)
	if err != nil {
		return nil, err
	}
	// Iterate over the shared-networks and check if any of them is
	// empty or contains only one subnet.
	emptyCount := int64(0)
	singleCount := int64(0)
	for _, net := range *decodedSharedNetworks {
		// Fetch the number of subnets in the shared network.
		var subnetsCount int
		switch ctx.subjectDaemon.Name {
		case dbmodel.DaemonNameDHCPv4:
			subnetsCount = len(net.Subnet4)
		case dbmodel.DaemonNameDHCPv6:
			subnetsCount = len(net.Subnet6)
		default:
			return nil, errors.Errorf("unsupported daemon %s", ctx.subjectDaemon.Name)
		}
		// Depending on whether there are no subnets or there is a single subnet
		// let's update the respective counters.
		switch subnetsCount {
		case 0:
			emptyCount++
		case 1:
			singleCount++
		}
	}

	// Create a report only if there is at least one empty shared network
	// or a shared network with only one subnet.
	if emptyCount > 0 || singleCount > 0 {
		details := ""
		if emptyCount > 0 {
			details = storkutil.FormatNoun(emptyCount, "empty shared network", "s")
		}
		if singleCount > 0 {
			if len(details) > 0 {
				details += " and "
			}
			details += storkutil.FormatNoun(singleCount, "shared network", "s")
			details += " with only a single subnet"
		}
		r, err := NewReport(ctx, fmt.Sprintf("Kea {daemon} configuration includes %s. Shared networks create overhead for a Kea server configuration and DHCP message processing, affecting their performance. It is recommended to remove any shared networks having none or a single subnet and specify these subnets at the global configuration level.", details)).
			referencingDaemon(ctx.subjectDaemon).
			create()
		return r, err
	}
	// There are no empty shared networks nor shared networks with
	// a single subnet.
	return nil, nil
}

// Creates a report for a checker verifying if a subnet can be removed
// because it contains no pools and no reservations.
func createSubnetDispensableReport(ctx *ReviewContext, dispensableCount int64) (*Report, error) {
	if dispensableCount == 0 {
		return nil, nil
	}
	r, err := NewReport(ctx, fmt.Sprintf("Kea {daemon} configuration includes %s without pools and host reservations. The DHCP server will not assign any addresses to the devices within this subnet. It is recommended to add some pools or host reservations to this subnet or remove the subnet from the configuration.", storkutil.FormatNoun(dispensableCount, "subnet", "s"))).
		referencingDaemon(ctx.subjectDaemon).
		create()
	return r, err
}

// Implementation of a checker verifying if an IPv4 subnet can be removed
// because it includes no pools and no reservations.
func checkSubnet4Dispensable(ctx *ReviewContext) (*Report, error) {
	type subnet4 struct {
		ID     int64
		Subnet string
		Pools  []struct {
			Pool string
		}
		Reservations []struct {
			IPAddress string
		}
	}
	// Create a structure into which the shared networks will be decoded.
	type sharedNetwork struct {
		Name    string
		Subnet4 []subnet4
	}
	decodedSharedNetworks := &[]sharedNetwork{}

	// Parse shared-networks list.
	config := ctx.subjectDaemon.KeaDaemon.Config
	err := config.DecodeSharedNetworks(decodedSharedNetworks)
	if err != nil {
		return nil, err
	}

	// Parse top-level subnets.
	var decodedSubnets4 []subnet4
	err = config.DecodeTopLevelSubnets(&decodedSubnets4)
	if err != nil {
		return nil, err
	}

	// Create an artificial shared network comprising the top-level
	// subnets. It will make the code below more readable.
	*decodedSharedNetworks = append(*decodedSharedNetworks, sharedNetwork{
		Subnet4: decodedSubnets4,
	})
	// Get hosts from the database when libdhcp_host_cmds hooks library is used.
	hostCmds, dbHosts, err := getDaemonHostsAndIndexBySubnet(ctx)
	if err != nil {
		return nil, err
	}
	// Iterate over the shared networks and check if they contain any
	// subnets that can be removed.
	dispensableCount := int64(0)
	for _, net := range *decodedSharedNetworks {
		for _, subnet := range net.Subnet4 {
			if len(subnet.Pools) == 0 && len(subnet.Reservations) == 0 &&
				(!hostCmds || len(dbHosts[subnet.ID]) == 0) {
				dispensableCount++
			}
		}
	}
	return createSubnetDispensableReport(ctx, dispensableCount)
}

// Implementation of a checker verifying if an IPv6 subnet can be removed
// because it includes no pools, no prefix delegation pools and no reservations.
func checkSubnet6Dispensable(ctx *ReviewContext) (*Report, error) {
	type subnet6 struct {
		ID     int64
		Subnet string
		Pools  []struct {
			Pool string
		}
		PDPools []struct {
			Prefix       string
			PrefixLen    int
			DelegatedLen int
		}
		Reservations []struct {
			IPAddresses []string
		}
	}
	// Create a structure into which the shared networks will be decoded.
	type sharedNetwork struct {
		Name    string
		Subnet6 []subnet6
	}
	decodedSharedNetworks := &[]sharedNetwork{}

	// Parse shared-networks list.
	config := ctx.subjectDaemon.KeaDaemon.Config
	err := config.DecodeSharedNetworks(decodedSharedNetworks)
	if err != nil {
		return nil, err
	}

	// Parse top-level subnets.
	var decodedSubnets6 []subnet6
	err = config.DecodeTopLevelSubnets(&decodedSubnets6)
	if err != nil {
		return nil, err
	}

	// Create an artificial shared network comprising the top-level
	// subnets. It will make the code below more readable.
	*decodedSharedNetworks = append(*decodedSharedNetworks, sharedNetwork{
		Subnet6: decodedSubnets6,
	})
	// Get hosts from the database when libdhcp_host_cmds hooks library is used.
	hostCmds, dbHosts, err := getDaemonHostsAndIndexBySubnet(ctx)
	if err != nil {
		return nil, err
	}
	// Iterate over the shared networks and check if they contain any
	// subnets that can be removed.
	dispensableCount := int64(0)
	for _, net := range *decodedSharedNetworks {
		for _, subnet := range net.Subnet6 {
			if len(subnet.Pools) == 0 && len(subnet.PDPools) == 0 && len(subnet.Reservations) == 0 &&
				(!hostCmds || len(dbHosts[subnet.ID]) == 0) {
				dispensableCount++
			}
		}
	}
	return createSubnetDispensableReport(ctx, dispensableCount)
}

// The checker verifying if a subnet can be removed because it includes
// no pools and no reservations. The check is skipped when the host_cmds
// hook library is loaded because host reservations may be present in
// the database.
func subnetDispensable(ctx *ReviewContext) (*Report, error) {
	if ctx.subjectDaemon.Name != dbmodel.DaemonNameDHCPv4 &&
		ctx.subjectDaemon.Name != dbmodel.DaemonNameDHCPv6 {
		return nil, errors.Errorf("unsupported daemon %s", ctx.subjectDaemon.Name)
	}
	if ctx.subjectDaemon.Name == dbmodel.DaemonNameDHCPv4 {
		return checkSubnet4Dispensable(ctx)
	}
	return checkSubnet6Dispensable(ctx)
}

// Fetch hosts for the tested daemon and index them by local subnet ID.
func getDaemonHostsAndIndexBySubnet(ctx *ReviewContext) (hostCmds bool, dbHosts map[int64][]dbmodel.Host, err error) {
	dbHosts = make(map[int64][]dbmodel.Host)
	if ctx.db == nil {
		return false, dbHosts, nil
	}
	if _, _, present := ctx.subjectDaemon.KeaDaemon.Config.GetHooksLibrary("libdhcp_host_cmds"); present {
		hosts, _, err := dbmodel.GetHostsByDaemonID(ctx.db, ctx.subjectDaemon.ID, dbmodel.HostDataSourceAPI)
		if err != nil {
			return present, dbHosts, err
		}
		for i, host := range hosts {
			if host.Subnet != nil {
				for _, ls := range host.Subnet.LocalSubnets {
					if ls.DaemonID == ctx.subjectDaemon.ID && ls.LocalSubnetID != 0 {
						dbHosts[ls.LocalSubnetID] = append(dbHosts[ls.LocalSubnetID], hosts[i])
					}
				}
			}
		}
		return present, dbHosts, nil
	}
	return false, dbHosts, nil
}

// Check if any of the listed addresses is within any of the address pools.
func isAnyAddressInPools(addresses []string, pools []keaconfig.Pool) bool {
	for _, ip := range addresses {
		parsedReservation := storkutil.ParseIP(ip)
		if parsedReservation == nil {
			continue
		}
		for _, pool := range pools {
			lb, ub, err := storkutil.ParseIPRange(pool.Pool)
			if err != nil {
				continue
			}
			if parsedReservation.IsInRange(lb, ub) {
				// We found an IP reservation that is within a pool.
				return true
			}
		}
	}
	return false
}

// Check if any of the listed IP reservations is within any of the address pools.
func isAnyIPReservationInPools(reservations []dbmodel.IPReservation, pools []keaconfig.Pool) bool {
	for _, reservation := range reservations {
		parsedReservation := storkutil.ParseIP(reservation.Address)
		if parsedReservation == nil || parsedReservation.Prefix {
			continue
		}
		for _, pool := range pools {
			lb, ub, err := storkutil.ParseIPRange(pool.Pool)
			if err != nil {
				continue
			}
			if parsedReservation.IsInRange(lb, ub) {
				// We found an IP reservation that is within a pool.
				return true
			}
		}
	}
	return false
}

// Check if any of the listed IP reservations is within any of the prefix pools.
func isAnyIPReservationInPDPools(reservations []dbmodel.IPReservation, pdpools []keaconfig.PdPool) bool {
	for _, reservation := range reservations {
		parsedReservation := storkutil.ParseIP(reservation.Address)
		if parsedReservation == nil || !parsedReservation.Prefix {
			continue
		}
		for _, pdpool := range pdpools {
			if parsedReservation.IsInPrefixRange(pdpool.Prefix, pdpool.PrefixLen, pdpool.DelegatedLen) {
				// We found an IP reservation that is within a prefix pool.
				return true
			}
		}
	}
	return false
}

// Implementation of a checker suggesting the use of out-of-pool host reservation
// mode when there are IPv4 subnets with all host reservations outside of the
// dynamic pools.
func checkDHCPv4ReservationsOutOfPool(ctx *ReviewContext) (*Report, error) {
	type subnet4 struct {
		ID           int64
		Subnet       string
		Pools        []keaconfig.Pool
		Reservations []struct {
			IPAddress string
		}
		keaconfig.ReservationModes
	}
	// Create a structure into which the shared networks will be decoded.
	type sharedNetwork struct {
		Name    string
		Subnet4 []subnet4
		keaconfig.ReservationModes
	}
	// Parse shared-networks list.
	var decodedSharedNetworks []sharedNetwork
	config := ctx.subjectDaemon.KeaDaemon.Config
	err := config.DecodeSharedNetworks(&decodedSharedNetworks)
	if err != nil {
		return nil, err
	}
	// Parse top-level subnets.
	var decodedSubnets4 []subnet4
	err = config.DecodeTopLevelSubnets(&decodedSubnets4)
	if err != nil {
		return nil, err
	}
	// Create an artificial shared network comprising the top-level
	// subnets. It will make the code below more readable.
	decodedSharedNetworks = append(decodedSharedNetworks, sharedNetwork{
		Subnet4: decodedSubnets4,
	})
	// Get global host reservation mode settings.
	globalModes := config.GetGlobalReservationModes()
	if globalModes == nil {
		return nil, errors.New("problem getting global reservation modes from Kea configuration")
	}
	// Get hosts from the database when libdhcp_host_cmds hooks library is used.
	_, dbHosts, err := getDaemonHostsAndIndexBySubnet(ctx)
	if err != nil {
		return nil, err
	}
	// Count the subnets for which it is feasible to enable out-of-pool
	// reservation mode.
	oopSubnetsCount := int64(0)
	for _, net := range decodedSharedNetworks {
		for _, subnet := range net.Subnet4 {
			// Check if out-of-pool host reservation mode has been enabled at
			// any level of inheritance from the subnet to the global scope.
			// If that mode has been already enabled there is nothing to do for
			// this subnet.
			if keaconfig.IsInAnyReservationModes(func(modes keaconfig.ReservationModes) (bool, bool) {
				return modes.IsOutOfPool()
			}, subnet.ReservationModes, net.ReservationModes, *globalModes) {
				continue
			}
			// If there are no reservations in this subnet there is nothing
			// to do.
			if len(subnet.Reservations) == 0 && len(dbHosts) == 0 {
				continue
			}
			inPool := false
			ipResrvExist := false
			// Check if at least one reservation is within a pool.
			for _, reservation := range subnet.Reservations {
				if len(reservation.IPAddress) > 0 {
					ipResrvExist = true
					// Check if the IP address belongs to any of the pools. If
					// it does, move to the next subnet.
					if isAnyAddressInPools([]string{reservation.IPAddress}, subnet.Pools) {
						inPool = true
						break
					}
				}
			}
			// If there is no in-pool reservation in the configured reservations
			// let's check if there are some in the host database.
			if !inPool {
				for _, dbHost := range dbHosts[subnet.ID] {
					ipResrvExist = true
					if len(dbHost.IPReservations) > 0 {
						if isAnyIPReservationInPools(dbHost.IPReservations, subnet.Pools) {
							inPool = true
							break
						}
					}
				}
			}
			// Didn't find in-pool reservations in the configuration file and in
			// the host database.
			if ipResrvExist && !inPool {
				// No in-pool reservation.
				oopSubnetsCount++
			}
		}
	}

	if oopSubnetsCount > 0 {
		r, err := NewReport(ctx, fmt.Sprintf("Kea {daemon} configuration includes %s for which it is recommended to use out-of-pool host-reservation mode. Reservations specified for these subnets are outside the dynamic address pools. Using out-of-pool reservation mode prevents Kea from checking host-reservation existence when allocating in-pool addresses, thus improving performance.", storkutil.FormatNoun(oopSubnetsCount, "subnet", "s"))).
			referencingDaemon(ctx.subjectDaemon).
			create()
		return r, err
	}
	return nil, nil
}

// Check if any of the listed prefixes is within any of the prefix pools.
func isAnyPrefixInPools(prefixes []string, pools []keaconfig.PdPool) bool {
	for _, pd := range prefixes {
		parsedReservation := storkutil.ParseIP(pd)
		if parsedReservation == nil {
			continue
		}
		for _, pdpool := range pools {
			if parsedReservation.IsInPrefixRange(pdpool.Prefix, pdpool.PrefixLen, pdpool.DelegatedLen) {
				// We found a prefix reservation that is within a pool.
				return true
			}
		}
	}
	return false
}

// Implementation of a checker suggesting the use of out-of-pool host reservation
// mode when there are IPv6 subnets with all host reservations outside of the
// dynamic pools.
func checkDHCPv6ReservationsOutOfPool(ctx *ReviewContext) (*Report, error) {
	type subnet6 struct {
		ID           int64
		Subnet       string
		Pools        []keaconfig.Pool
		PDPools      []keaconfig.PdPool
		Reservations []struct {
			IPAddresses []string
			Prefixes    []string
		}
		keaconfig.ReservationModes
	}
	// Create a structure into which the shared networks will be decoded.
	type sharedNetwork struct {
		Name    string
		Subnet6 []subnet6
		keaconfig.ReservationModes
	}
	// Parse shared-networks list.
	var decodedSharedNetworks []sharedNetwork
	config := ctx.subjectDaemon.KeaDaemon.Config
	err := config.DecodeSharedNetworks(&decodedSharedNetworks)
	if err != nil {
		return nil, err
	}
	// Parse top-level subnets.
	var decodedSubnets6 []subnet6
	err = config.DecodeTopLevelSubnets(&decodedSubnets6)
	if err != nil {
		return nil, err
	}
	// Create an artificial shared network comprising the top-level
	// subnets. It will make the code below more readable.
	decodedSharedNetworks = append(decodedSharedNetworks, sharedNetwork{
		Subnet6: decodedSubnets6,
	})
	// Get global host reservation mode settings.
	globalModes := config.GetGlobalReservationModes()
	if globalModes == nil {
		return nil, errors.New("problem getting global reservation modes from Kea configuration")
	}
	// Get hosts from the database when libdhcp_host_cmds hooks library is used.
	_, dbHosts, err := getDaemonHostsAndIndexBySubnet(ctx)
	if err != nil {
		return nil, err
	}
	// Count the subnets for which it is feasible to enable out-of-pool
	// reservation mode.
	oopSubnetsCount := int64(0)
	for _, net := range decodedSharedNetworks {
		for _, subnet := range net.Subnet6 {
			// Check if out-of-pool host reservation mode has been enabled at
			// any level of inheritance from the subnet to the global scope.
			// If that mode has been already enabled there is nothing to do for
			// this subnet.
			if keaconfig.IsInAnyReservationModes(func(modes keaconfig.ReservationModes) (bool, bool) {
				return modes.IsOutOfPool()
			}, subnet.ReservationModes, net.ReservationModes, *globalModes) {
				continue
			}
			// If there are no reservations in this subnet there is nothing
			// to do.
			if len(subnet.Reservations) == 0 && len(dbHosts) == 0 {
				continue
			}
			inPool := false
			ipResrvExist := false
			// Check if at least one reservation is within a pool.
			for _, reservation := range subnet.Reservations {
				if len(reservation.IPAddresses) > 0 || len(reservation.Prefixes) > 0 {
					ipResrvExist = true
					// Check if any of the IP addresses or delegated prefixes belong to any
					// of the pools. If so, move to the next subnet.
					if isAnyAddressInPools(reservation.IPAddresses, subnet.Pools) ||
						isAnyPrefixInPools(reservation.Prefixes, subnet.PDPools) {
						inPool = true
						break
					}
				}
			}
			// If there is no in-pool reservation in the configured reservations
			// let's check if there are some in the host database.
			if !inPool {
				for _, dbHost := range dbHosts[subnet.ID] {
					ipResrvExist = true
					if len(dbHost.IPReservations) > 0 {
						if isAnyIPReservationInPools(dbHost.IPReservations, subnet.Pools) ||
							isAnyIPReservationInPDPools(dbHost.IPReservations, subnet.PDPools) {
							inPool = true
							break
						}
					}
				}
			}
			// Didn't find in-pool reservations in the configuration file and in
			// the host database.
			if ipResrvExist && !inPool {
				// No in-pool reservation.
				oopSubnetsCount++
			}
		}
	}

	if oopSubnetsCount > 0 {
		r, err := NewReport(ctx, fmt.Sprintf("Kea {daemon} configuration includes %s for which it is recommended to use out-of-pool host-reservation mode. Reservations specified for these subnets appear outside the dynamic-address and/or prefix-delegation pools. Using out-of-pool reservation mode prevents Kea from checking host-reservation existence when allocating in-pool addresses and delegated prefixes, thus improving performance.", storkutil.FormatNoun(oopSubnetsCount, "subnet", "s"))).
			referencingDaemon(ctx.subjectDaemon).
			create()
		return r, err
	}
	return nil, nil
}

// The checker suggesting the use of out-of-pool host reservation mode
// when there are subnets with all host reservations outside of the
// dynamic pools.
func reservationsOutOfPool(ctx *ReviewContext) (*Report, error) {
	if ctx.subjectDaemon.Name != dbmodel.DaemonNameDHCPv4 &&
		ctx.subjectDaemon.Name != dbmodel.DaemonNameDHCPv6 {
		return nil, errors.Errorf("unsupported daemon %s", ctx.subjectDaemon.Name)
	}
	if ctx.subjectDaemon.Name == dbmodel.DaemonNameDHCPv4 {
		return checkDHCPv4ReservationsOutOfPool(ctx)
	}
	return checkDHCPv6ReservationsOutOfPool(ctx)
}

type minimalSubnet struct {
	ID     int64
	Subnet string
}

type minimalSubnetPair struct {
	parent minimalSubnet
	child  minimalSubnet
}

// The checker validates that subnets (global or from shared networks) don't overlap.
func subnetsOverlapping(ctx *ReviewContext) (*Report, error) {
	if ctx.subjectDaemon.Name != dbmodel.DaemonNameDHCPv4 &&
		ctx.subjectDaemon.Name != dbmodel.DaemonNameDHCPv6 {
		return nil, errors.Errorf("unsupported daemon %s", ctx.subjectDaemon.Name)
	}

	config := ctx.subjectDaemon.KeaDaemon.Config

	var decodedSubnets []minimalSubnet
	// Global subnets.
	err := config.DecodeTopLevelSubnets(&decodedSubnets)
	if err != nil {
		return nil, err
	}

	// Subnets belonging to the shared networks.
	type minimalSharedNetwork struct {
		Subnet4 []minimalSubnet
		Subnet6 []minimalSubnet
	}
	var decodedSharedNetworks []minimalSharedNetwork
	err = config.DecodeSharedNetworks(&decodedSharedNetworks)
	if err != nil {
		return nil, err
	}

	for _, sharedNetwork := range decodedSharedNetworks {
		decodedSubnets = append(decodedSubnets, sharedNetwork.Subnet4...)
		decodedSubnets = append(decodedSubnets, sharedNetwork.Subnet6...)
	}

	// Limits the overlaps count to avoid producing too huge review message.
	maxOverlaps := 10
	overlaps := findOverlaps(decodedSubnets, maxOverlaps)
	if len(overlaps) == 0 {
		return nil, nil
	}

	maxExceedMessage := ""
	if len(overlaps) == maxOverlaps {
		maxExceedMessage = " at least"
	}

	overlappingMessages := make([]string, len(overlaps))
	for i, overlap := range overlaps {
		parentID := ""
		if overlap.parent.ID != 0 {
			parentID = fmt.Sprintf(" (subnet-id %d)", overlap.parent.ID)
		}
		childID := ""
		if overlap.child.ID != 0 {
			childID = fmt.Sprintf(" (subnet-id %d)", overlap.child.ID)
		}

		message := fmt.Sprintf("%d. %s%s is overlapped by %s%s", i+1,
			overlap.parent.Subnet, parentID,
			overlap.child.Subnet, childID)
		overlappingMessages[i] = message
	}
	overlapMessage := strings.Join(overlappingMessages, "; ")

	return NewReport(ctx, fmt.Sprintf("Kea {daemon} configuration includes%s %s. "+
		"It means that the DHCP clients in different subnets may be assigned the same IP addresses.\n%s",
		maxExceedMessage, storkutil.FormatNoun(int64(len(overlaps)), "overlapping subnet pair", "s"), overlapMessage)).
		referencingDaemon(ctx.subjectDaemon).
		create()
}

// Search for prefix overlaps in the provided set of subnets.
// The execution is stopped early if an expected name of founded overlaps is
// reached.
func findOverlaps(subnets []minimalSubnet, maxOverlaps int) (overlaps []minimalSubnetPair) {
	// Pair of the subnet and its binary prefix.
	type subnetWithPrefix struct {
		subnet       minimalSubnet
		binaryPrefix string
	}

	// Calculates the binary prefixes for all subnets.
	subnetPrefixes := make([]subnetWithPrefix, len(subnets))

	for i, subnet := range subnets {
		cidr := storkutil.ParseIP(subnet.Subnet)
		if cidr == nil || !cidr.Prefix {
			continue
		}
		binaryPrefix := cidr.GetNetworkPrefixAsBinary()

		subnetPrefixes[i] = subnetWithPrefix{
			subnet:       subnet,
			binaryPrefix: binaryPrefix,
		}
	}

	// Sorts prefixes from the shortest (the most general masks) to the longest
	// (the most specific masks).
	sort.Slice(subnetPrefixes, func(i, j int) bool {
		return len(subnetPrefixes[i].binaryPrefix) <= len(subnetPrefixes[j].binaryPrefix)
	})

	for outerIdx, outer := range subnetPrefixes {
		for innerIdx, inner := range subnetPrefixes {
			// The prefixes are sorted by length. The prefix length is equal to
			// the subnet mask in bits. For a given prefix with length X, we
			// need only check the prefixes with lengths equal to or greater
			// than X. It means that we need to check only the following
			// prefixes.
			if outerIdx >= innerIdx {
				continue
			}

			// Checks if the outer prefix contains the inner prefix. It happens
			// when the inner prefix's binary representation starts with the
			// outer prefix's binary representation.
			if strings.HasPrefix(inner.binaryPrefix, outer.binaryPrefix) {
				overlaps = append(overlaps, minimalSubnetPair{
					parent: outer.subnet,
					child:  inner.subnet,
				})

				// Checks if the overlap limit is exceed.
				if len(overlaps) == maxOverlaps {
					return
				}
			}
		}
	}
	return overlaps
}

// The checker validates that all subnet prefixes are in canonical form.
func canonicalPrefixes(ctx *ReviewContext) (*Report, error) {
	if ctx.subjectDaemon.Name != dbmodel.DaemonNameDHCPv4 &&
		ctx.subjectDaemon.Name != dbmodel.DaemonNameDHCPv6 {
		return nil, errors.Errorf("unsupported daemon %s", ctx.subjectDaemon.Name)
	}

	config := ctx.subjectDaemon.KeaDaemon.Config

	var decodedSubnets []minimalSubnet
	// Global subnets.
	err := config.DecodeTopLevelSubnets(&decodedSubnets)
	if err != nil {
		return nil, err
	}

	// Subnets belonging to the shared networks.
	type minimalSharedNetwork struct {
		Subnet4 []minimalSubnet
		Subnet6 []minimalSubnet
	}
	var decodedSharedNetworks []minimalSharedNetwork
	err = config.DecodeSharedNetworks(&decodedSharedNetworks)
	if err != nil {
		return nil, err
	}

	for _, sharedNetwork := range decodedSharedNetworks {
		decodedSubnets = append(decodedSubnets, sharedNetwork.Subnet4...)
		decodedSubnets = append(decodedSubnets, sharedNetwork.Subnet6...)
	}

	maxIssues := 10
	var issues []string

	for _, decodedSubnet := range decodedSubnets {
		prefix, ok := getCanonicalPrefix(decodedSubnet.Subnet)
		if ok {
			continue
		}

		subnetID := ""
		if decodedSubnet.ID != 0 {
			subnetID = fmt.Sprintf("[%d] ", decodedSubnet.ID)
		}

		issue := fmt.Sprintf("%d. %s%s is invalid prefix", len(issues)+1, subnetID, decodedSubnet.Subnet)

		if prefix != "" {
			issue = fmt.Sprintf("%s, expected: %s", issue, prefix)
		}

		issues = append(issues, issue)

		if len(issues) == maxIssues {
			break
		}
	}

	if len(issues) == 0 {
		return nil, nil
	}

	maxExceedMessage := ""
	if len(issues) == maxIssues {
		maxExceedMessage = " at least"
	}

	hintMessage := strings.Join(issues, "; ")

	return NewReport(ctx, fmt.Sprintf("Kea {daemon} configuration contains%s %s. "+
		"Kea accepts non-canonical prefix forms, which may lead to duplicates "+
		"if two subnets have the same prefix specified in different forms. "+
		"Use canonical forms to ensure that Kea properly identifies and "+
		"validates subnet prefixes to avoid duplication or overlap.\n%s",
		maxExceedMessage, storkutil.FormatNoun(int64(len(issues)), "non-canonical prefix", "es"), hintMessage)).
		referencingDaemon(ctx.subjectDaemon).
		create()
}

// Returns the prefix with zeros on masked bits. If it was already valid, return the true status.
func getCanonicalPrefix(prefix string) (string, bool) {
	candidate := storkutil.ParseIP(prefix)
	if candidate == nil {
		return "", false
	}
	expected := storkutil.ParseIP(candidate.NetworkPrefix)
	for i, b := range candidate.IP {
		if b != expected.IP[i] {
			return candidate.GetNetworkPrefixWithLength(), false
		}
	}
	return candidate.GetNetworkPrefixWithLength(), true
}

// The checker validates that the HA is running in multithreading mode if the
// Kea uses this mode.
func highAvailabilityMultithreadingMode(ctx *ReviewContext) (*Report, error) {
	if ctx.subjectDaemon.Name != dbmodel.DaemonNameDHCPv4 &&
		ctx.subjectDaemon.Name != dbmodel.DaemonNameDHCPv6 {
		return nil, errors.Errorf("unsupported daemon %s", ctx.subjectDaemon.Name)
	}

	config := ctx.subjectDaemon.KeaDaemon.Config

	multiThreadingConfig := config.GetMultiThreadingEntry()

	if multiThreadingConfig == nil ||
		multiThreadingConfig.EnableMultiThreading == nil ||
		!*multiThreadingConfig.EnableMultiThreading {
		// The top-level multi-threading is not configured or disabled.
		return nil, nil
	}

	_, haConfig, ok := config.GetHAHooksLibrary()
	if !ok {
		// There is no HA configured.
		return nil, nil
	}

	// Suggest to enable HA+MT.
	haMultiThreadingConfig := haConfig.MultiThreading
	if haMultiThreadingConfig == nil ||
		haMultiThreadingConfig.EnableMultiThreading == nil ||
		!*haMultiThreadingConfig.EnableMultiThreading {
		// The HA-level multi-threading is not configured or disabled.
		return NewReport(ctx, "The Kea {daemon} daemon is configured to work "+
			"in multi-threading mode, but the High Availability hooks use "+
			"single-thread mode. You can set the 'multi-threading' parameter "+
			"to true in the HA hook configuration to enable the "+
			"multi-threading mode and improve the performance of the "+
			"communication between HA servers.").
			referencingDaemon(ctx.subjectDaemon).
			create()
	}

	// Hint to configure proper HTTP ports.

	// Collect all CA daemons in a given application.
	var caDaemons []*dbmodel.Daemon
	for _, daemon := range ctx.subjectDaemon.App.Daemons {
		if daemon.Name == dbmodel.DaemonNameCA {
			caDaemons = append(caDaemons, daemon)
		}
	}

	for _, peer := range haConfig.Peers {
		if !peer.IsSet() {
			// Invalid peer. Skip.
			continue
		}

		urlObj, err := url.Parse(*peer.URL)
		if err != nil {
			// It should never happen. Kea disallows invalid URLs.
			continue
		}

		port, err := strconv.ParseInt(urlObj.Port(), 10, 64)
		if err != nil {
			// It should never happen. Kea disallows invalid URLs.
			continue
		}

		for _, accessPoint := range ctx.subjectDaemon.App.AccessPoints {
			if accessPoint.Address != urlObj.Hostname() {
				// There is no port collision due to the ports belong to different hosts.
				continue
			}
			if accessPoint.Port == port {
				// Port collision.
				report := NewReport(ctx, fmt.Sprintf("The HA '%s' peer with the '%s' URL is "+
					"configured to use the same '%d' HTTP port as the '%s' "+
					"access point. It may cause the bottlenecks that nullify "+
					"any performance gains offered by HA+MT",
					*peer.Name, *peer.URL, port, accessPoint.Type)).
					referencingDaemon(ctx.subjectDaemon)

				for _, daemon := range caDaemons {
					report = report.referencingDaemon(daemon)
				}

				return report.create()
			}
		}
	}

	return nil, nil
}
