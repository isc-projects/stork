package configreview

import (
	"fmt"
	"math/big"
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
	if _, _, present := config.GetHookLibrary("libdhcp_stat_cmds"); !present {
		r, err := NewReport(ctx, "The Kea Statistics Commands library "+
			"(libdhcp_stat_cmds) provides commands for retrieving accurate "+
			"DHCP lease statistics for Kea DHCP servers. Stork sends these "+
			"commands to fetch lease statistics displayed in the dashboard, "+
			"subnet, and shared-network views. Stork found that {daemon} is "+
			"not using this hook library. Some statistics will not be "+
			"available until the library is loaded.").
			referencingDaemon(ctx.subjectDaemon).
			create()
		return r, err
	}
	return nil, nil
}

// The checker verifying if the lease_cmds hooks library is loaded.
func leaseCmdsPresence(ctx *ReviewContext) (*Report, error) {
	config := ctx.subjectDaemon.KeaDaemon.Config
	if _, _, present := config.GetHookLibrary("libdhcp_lease_cmds"); !present {
		r, err := NewReport(ctx, "The Kea Lease Commands library "+
			"(libdhcp_lease_cmds) provides commands for retrieving "+
			"DHCP leases from Kea DHCP servers. Stork sends these "+
			"commands to search for leases matching the criteria specified in the "+
			"lease search box and to check if the host reservations are used. "+
			"Stork found that {daemon} is not using this hook library. These "+
			"features will not be available until the library is loaded.").
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
	if _, _, present := config.GetHookLibrary("libdhcp_host_cmds"); !present {
		databases := config.GetAllDatabases()
		if len(databases.Hosts) > 0 {
			r, err := NewReport(ctx, "Kea can be configured to store host "+
				"reservations in a database. Stork can access these "+
				"reservations using the commands implemented in the Host "+
				"Commands hook library and make them available in the Host "+
				"Reservations view. It appears that the libdhcp_host_cmds "+
				"hook library is not loaded on {daemon}. Host reservations "+
				"from the database will not be visible in Stork until this "+
				"library is enabled.").
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
	// Get parsed shared-networks list.
	sharedNetworks := config.GetSharedNetworks(false)

	// Iterate over the shared-networks and check if any of them is
	// empty or contains only one subnet.
	emptyCount := int64(0)
	singleCount := int64(0)
	for _, net := range sharedNetworks {
		// Depending on whether there are no subnets or there is a single
		// subnet let's update the respective counters.
		switch len(net.GetSubnets()) {
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
		r, err := NewReport(ctx, fmt.Sprintf(
			"Kea {daemon} configuration includes %s. Shared networks create "+
				"overhead for a Kea server configuration and DHCP message "+
				"processing, affecting their performance. It is recommended "+
				"to remove any shared networks having none or a single "+
				"subnet and specify these subnets at the global "+
				"configuration level.",
			details,
		)).
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
	r, err := NewReport(ctx, fmt.Sprintf(
		"Kea {daemon} configuration includes %s without pools and host "+
			"reservations. The DHCP server will not assign any addresses to "+
			"the devices within this subnet. It is recommended to add some "+
			"pools or host reservations to this subnet or remove the subnet "+
			"from the configuration.",
		storkutil.FormatNoun(dispensableCount, "subnet", "s"),
	)).
		referencingDaemon(ctx.subjectDaemon).
		create()
	return r, err
}

// Implementation of a checker verifying if an IPv4 subnet can be removed
// because it includes no pools and no reservations.
func checkSubnet4Dispensable(ctx *ReviewContext) (*Report, error) {
	// Get parsed shared-networks list including top-level subnets.
	config := ctx.subjectDaemon.KeaDaemon.Config
	sharedNetworks := config.GetSharedNetworks(true)

	// Get hosts from the database when libdhcp_host_cmds hooks library is used.
	hostCmds, dbHosts, err := getDaemonHostsAndIndexBySubnet(ctx)
	if err != nil {
		return nil, err
	}
	// Iterate over the shared networks and check if they contain any
	// subnets that can be removed.
	dispensableCount := int64(0)
	for _, net := range sharedNetworks {
		for _, subnet := range net.GetSubnets() {
			if len(subnet.GetPools()) == 0 && len(subnet.GetReservations()) == 0 &&
				(!hostCmds || len(dbHosts[subnet.GetID()]) == 0) {
				dispensableCount++
			}
		}
	}
	return createSubnetDispensableReport(ctx, dispensableCount)
}

// Implementation of a checker verifying if an IPv6 subnet can be removed
// because it includes no pools, no prefix delegation pools and no reservations.
func checkSubnet6Dispensable(ctx *ReviewContext) (*Report, error) {
	// Get parsed shared-networks list including top level subnets.
	config := ctx.subjectDaemon.KeaDaemon.Config
	sharedNetworks := config.GetSharedNetworks(true)

	// Get hosts from the database when libdhcp_host_cmds hooks library is used.
	hostCmds, dbHosts, err := getDaemonHostsAndIndexBySubnet(ctx)
	if err != nil {
		return nil, err
	}
	// Iterate over the shared networks and check if they contain any
	// subnets that can be removed.
	dispensableCount := int64(0)
	for _, net := range sharedNetworks {
		for _, subnet := range net.GetSubnets() {
			// Empty address pools.
			if len(subnet.GetPools()) == 0 &&
				// Empty delegated prefix pools.
				len(subnet.GetPDPools()) == 0 &&
				// Empty host reservations
				len(subnet.GetReservations()) == 0 &&
				// Missing host cmds hook or empty DB host reservations.
				(!hostCmds || len(dbHosts[subnet.GetID()]) == 0) {
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
		return nil, errors.Errorf(
			"unsupported daemon %s",
			ctx.subjectDaemon.Name,
		)
	}
	if ctx.subjectDaemon.Name == dbmodel.DaemonNameDHCPv4 {
		return checkSubnet4Dispensable(ctx)
	}
	return checkSubnet6Dispensable(ctx)
}

// Fetch hosts for the tested daemon and index them by local subnet ID.
func getDaemonHostsAndIndexBySubnet(ctx *ReviewContext) (hostCmds bool, dbHosts map[int64][]dbmodel.Host, err error) {
	dbHosts = make(map[int64][]dbmodel.Host)
	if _, _, present := ctx.subjectDaemon.KeaDaemon.Config.GetHookLibrary("libdhcp_host_cmds"); present {
		hosts, _, err := dbmodel.GetHostsByDaemonID(
			ctx.db,
			ctx.subjectDaemon.ID,
			dbmodel.HostDataSourceAPI,
		)
		if err != nil {
			return present, dbHosts, err
		}
		for i, host := range hosts {
			if host.Subnet == nil {
				continue
			}
			for _, ls := range host.Subnet.LocalSubnets {
				if ls.DaemonID == ctx.subjectDaemon.ID && ls.LocalSubnetID != 0 {
					dbHosts[ls.LocalSubnetID] = append(dbHosts[ls.LocalSubnetID], hosts[i])
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
func isAnyIPReservationInPools(reservedAddresses []string, pools []keaconfig.Pool) bool {
	for _, address := range reservedAddresses {
		parsedReservation := storkutil.ParseIP(address)
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
func isAnyIPReservationInPDPools(addresses []string, pdPools []keaconfig.PDPool) bool {
	for _, address := range addresses {
		parsedReservation := storkutil.ParseIP(address)
		if parsedReservation == nil || !parsedReservation.Prefix {
			continue
		}
		for _, pdPool := range pdPools {
			if parsedReservation.IsInPrefixRange(pdPool.Prefix, pdPool.PrefixLen, pdPool.DelegatedLen) {
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
	// Get parsed shared-networks list with top-level subnets.
	config := ctx.subjectDaemon.KeaDaemon.Config
	sharedNetworks := config.GetSharedNetworks(true)

	// Get global host reservation mode settings.
	globalParameters := config.GetGlobalReservationParameters()

	// Get hosts from the database when libdhcp_host_cmds hooks library is used.
	_, dbHosts, err := getDaemonHostsAndIndexBySubnet(ctx)
	if err != nil {
		return nil, err
	}
	// Count the subnets for which it is feasible to enable out-of-pool
	// reservation mode.
	oopSubnetsCount := int64(0)
	for _, net := range sharedNetworks {
		for _, subnet := range net.GetSubnets() {
			// Check if out-of-pool host reservation mode has been enabled at
			// any level of inheritance from the subnet to the global scope.
			// If that mode has been already enabled there is nothing to do for
			// this subnet.
			if keaconfig.IsInAnyReservationModes(func(modes keaconfig.ReservationParameters) (bool, bool) {
				return modes.IsOutOfPool()
			}, subnet.GetSubnetParameters().ReservationParameters, net.GetSharedNetworkParameters().ReservationParameters, globalParameters) {
				continue
			}
			// If there are no reservations in this subnet there is nothing
			// to do.
			if len(subnet.GetReservations()) == 0 && len(dbHosts) == 0 {
				continue
			}
			inPool := false
			ipResrvExist := false
			// Check if at least one reservation is within a pool.
			for _, reservation := range subnet.GetReservations() {
				if len(reservation.IPAddress) > 0 {
					ipResrvExist = true
					// Check if the IP address belongs to any of the pools. If
					// it does, move to the next subnet.
					if isAnyAddressInPools([]string{reservation.IPAddress}, subnet.GetPools()) {
						inPool = true
						break
					}
				}
			}
			// If there is no in-pool reservation in the configured reservations
			// let's check if there are some in the host database.
			if !inPool {
				for _, dbHost := range dbHosts[subnet.GetID()] {
					ipResrvExist = true
					ipAddresses := dbHost.GetIPReservations()
					if len(ipAddresses) > 0 {
						if isAnyIPReservationInPools(ipAddresses, subnet.GetPools()) {
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
		r, err := NewReport(ctx, fmt.Sprintf(
			"Kea {daemon} configuration includes %s for which it is "+
				"recommended to use out-of-pool host-reservation mode. "+
				"Reservations specified for these subnets are outside the "+
				"dynamic address pools. Using out-of-pool reservation mode "+
				"prevents Kea from checking host-reservation existence when "+
				"allocating in-pool addresses, thus improving performance.",
			storkutil.FormatNoun(oopSubnetsCount, "subnet", "s"),
		)).
			referencingDaemon(ctx.subjectDaemon).
			create()
		return r, err
	}
	return nil, nil
}

// Check if any of the listed prefixes is within any of the prefix pools.
func isAnyPrefixInPools(prefixes []string, pools []keaconfig.PDPool) bool {
	for _, pd := range prefixes {
		parsedReservation := storkutil.ParseIP(pd)
		if parsedReservation == nil {
			continue
		}
		for _, pdPool := range pools {
			if parsedReservation.IsInPrefixRange(pdPool.Prefix, pdPool.PrefixLen, pdPool.DelegatedLen) {
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
	// Get shared networks list including root subnets.
	config := ctx.subjectDaemon.KeaDaemon.Config
	sharedNetworks := config.GetSharedNetworks(true)

	// Get global host reservation mode settings.
	globalParameters := config.GetGlobalReservationParameters()

	// Get hosts from the database when libdhcp_host_cmds hooks library is used.
	_, dbHosts, err := getDaemonHostsAndIndexBySubnet(ctx)
	if err != nil {
		return nil, err
	}
	// Count the subnets for which it is feasible to enable out-of-pool
	// reservation mode.
	oopSubnetsCount := int64(0)
	for _, net := range sharedNetworks {
		for _, subnet := range net.GetSubnets() {
			// Check if out-of-pool host reservation mode has been enabled at
			// any level of inheritance from the subnet to the global scope.
			// If that mode has been already enabled there is nothing to do for
			// this subnet.
			if keaconfig.IsInAnyReservationModes(func(modes keaconfig.ReservationParameters) (bool, bool) {
				return modes.IsOutOfPool()
			}, subnet.GetSubnetParameters().ReservationParameters, net.GetSharedNetworkParameters().ReservationParameters, globalParameters) {
				continue
			}
			// If there are no reservations in this subnet there is nothing
			// to do.
			if len(subnet.GetReservations()) == 0 && len(dbHosts) == 0 {
				continue
			}
			inPool := false
			ipResrvExist := false
			// Check if at least one reservation is within a pool.
			for _, reservation := range subnet.GetReservations() {
				if len(reservation.IPAddresses) > 0 || len(reservation.Prefixes) > 0 {
					ipResrvExist = true
					// Check if any of the IP addresses or delegated prefixes
					// belong to any of the pools. If so, move to the next subnet.
					if isAnyAddressInPools(reservation.IPAddresses, subnet.GetPools()) ||
						isAnyPrefixInPools(reservation.Prefixes, subnet.GetPDPools()) {
						inPool = true
						break
					}
				}
			}
			// If there is no in-pool reservation in the configured reservations
			// let's check if there are some in the host database.
			if !inPool {
				for _, dbHost := range dbHosts[subnet.GetID()] {
					ipResrvExist = true
					ipAddresses := dbHost.GetIPReservations()
					if len(ipAddresses) > 0 {
						if isAnyIPReservationInPools(ipAddresses, subnet.GetPools()) ||
							isAnyIPReservationInPDPools(ipAddresses, subnet.GetPDPools()) {
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
		r, err := NewReport(ctx, fmt.Sprintf(
			"Kea {daemon} configuration includes %s for which it is "+
				"recommended to use out-of-pool host-reservation mode. "+
				"Reservations specified for these subnets appear outside the "+
				"dynamic-address and/or prefix-delegation pools. Using "+
				"out-of-pool reservation mode prevents Kea from checking "+
				"host-reservation existence when allocating in-pool "+
				"addresses and delegated prefixes, thus improving performance.",
			storkutil.FormatNoun(oopSubnetsCount, "subnet", "s"),
		)).
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

type minimalSubnetPair struct {
	parent keaconfig.Subnet
	child  keaconfig.Subnet
}

// The checker validates that subnets (global or from shared networks) don't
// overlap.
func subnetsOverlapping(ctx *ReviewContext) (*Report, error) {
	if ctx.subjectDaemon.Name != dbmodel.DaemonNameDHCPv4 &&
		ctx.subjectDaemon.Name != dbmodel.DaemonNameDHCPv6 {
		return nil, errors.Errorf(
			"unsupported daemon %s", ctx.subjectDaemon.Name,
		)
	}

	config := ctx.subjectDaemon.KeaDaemon.Config

	// Global subnets.
	subnets := config.GetSubnets()

	// Subnets belonging to the shared networks.
	sharedNetworks := config.GetSharedNetworks(false)

	for _, sharedNetwork := range sharedNetworks {
		subnets = append(subnets, sharedNetwork.GetSubnets()...)
	}

	// Limits the overlaps count to avoid producing too huge review message.
	maxOverlaps := 10
	overlaps := findOverlaps(subnets, maxOverlaps)
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
		if overlap.parent.GetID() != 0 {
			parentID = fmt.Sprintf("[%d] ", overlap.parent.GetID())
		}
		childID := ""
		if overlap.child.GetID() != 0 {
			childID = fmt.Sprintf("[%d] ", overlap.child.GetID())
		}

		message := fmt.Sprintf("%d. %s%s is overlapped by %s%s", i+1,
			parentID, overlap.parent.GetPrefix(),
			childID, overlap.child.GetPrefix())
		overlappingMessages[i] = message
	}
	overlapMessage := strings.Join(overlappingMessages, "; ")

	return NewReport(ctx, fmt.Sprintf("Kea {daemon} configuration "+
		"includes%s %s. It means that the DHCP clients in different subnets "+
		"may be assigned the same IP addresses.\n%s", maxExceedMessage,
		storkutil.FormatNoun(int64(len(overlaps)), "overlapping subnet pair", "s"),
		overlapMessage)).referencingDaemon(ctx.subjectDaemon).create()
}

// Search for prefix overlaps in the provided set of subnets.
// The execution is stopped early if an expected name of founded overlaps is
// reached.
func findOverlaps(subnets []keaconfig.Subnet, maxOverlaps int) (overlaps []minimalSubnetPair) {
	// Pair of the subnet and its binary prefix.
	type subnetWithPrefix struct {
		subnet       keaconfig.Subnet
		binaryPrefix string
	}

	// Calculates the binary prefixes for all subnets.
	var subnetPrefixes []subnetWithPrefix

	for _, subnet := range subnets {
		cidr := storkutil.ParseIP(subnet.GetPrefix())
		if cidr == nil || !cidr.Prefix {
			continue
		}
		binaryPrefix := cidr.GetNetworkPrefixAsBinary()

		subnetPrefixes = append(subnetPrefixes, subnetWithPrefix{
			subnet:       subnet,
			binaryPrefix: binaryPrefix,
		})
	}

	// Sorts prefixes from the shortest (the most general masks) to the longest
	// (the most specific masks). If the prefix lengths are equal, sort by the
	// prefix address. If the prefixes are the same, sort by the subnet ID.
	sort.Slice(subnetPrefixes, func(i, j int) bool {
		if len(subnetPrefixes[i].binaryPrefix) != len(subnetPrefixes[j].binaryPrefix) {
			return len(subnetPrefixes[i].binaryPrefix) < len(subnetPrefixes[j].binaryPrefix)
		}

		if subnetPrefixes[i].binaryPrefix != subnetPrefixes[j].binaryPrefix {
			return subnetPrefixes[i].binaryPrefix < subnetPrefixes[j].binaryPrefix
		}

		return subnetPrefixes[i].subnet.GetID() <= subnetPrefixes[j].subnet.GetID()
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

	// Global subnets and shared networks.
	subnets := config.GetSubnets()
	sharedNetworks := config.GetSharedNetworks(false)

	for _, sharedNetwork := range sharedNetworks {
		subnets = append(subnets, sharedNetwork.GetSubnets()...)
	}

	maxIssues := 10
	var issues []string

	for _, subnet := range subnets {
		prefix, ok := getCanonicalPrefix(subnet.GetPrefix())
		if ok {
			continue
		}

		subnetID := ""
		if subnet.GetID() != 0 {
			subnetID = fmt.Sprintf("[%d] ", subnet.GetID())
		}

		issue := fmt.Sprintf(
			"%d. %s%s is invalid prefix",
			len(issues)+1,
			subnetID,
			subnet.GetPrefix(),
		)

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

	return NewReport(ctx, fmt.Sprintf("Kea {daemon} configuration "+
		"contains%s %s. Kea accepts non-canonical prefix forms, which may "+
		"lead to duplicates if two subnets have the same prefix specified in "+
		"different forms. Use canonical forms to ensure that Kea properly "+
		"identifies and validates subnet prefixes to avoid duplication or "+
		"overlap.\n%s", maxExceedMessage,
		storkutil.FormatNoun(int64(len(issues)), "non-canonical prefix", "es"),
		hintMessage)).referencingDaemon(ctx.subjectDaemon).create()
}

// Returns the prefix with zeros on masked bits. If it was already valid,
// return the true status.
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

// The checker verifies that the HA is running in multi-threading mode if
// Kea uses this mode.
func highAvailabilityMultiThreadingMode(ctx *ReviewContext) (*Report, error) {
	config := ctx.subjectDaemon.KeaDaemon.Config

	keaVersion := storkutil.ParseSemanticVersionOrLatest(ctx.subjectDaemon.Version)

	if !config.IsMultiThreadingEnabled(keaVersion) {
		// There is no HA configured.
		return nil, nil
	}

	_, haConfig, ok := config.GetHookLibraries().GetHAHookLibrary()
	if !ok {
		// There is no HA configured.
		return nil, nil
	}

	var disabled bool
	for _, rel := range haConfig.GetAllRelationships() {
		if !rel.IsMultiThreadingEnabled(keaVersion) {
			disabled = true
			break
		}
	}
	if !disabled {
		// HA+MT enabled.
		return nil, nil
	}

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

// The checker validates that High Availability peers don't use the HTTP port
// assigned to the Kea Control Agent when the dedicated listeners are enabled.
func highAvailabilityDedicatedPorts(ctx *ReviewContext) (*Report, error) {
	config := ctx.subjectDaemon.KeaDaemon.Config

	keaVersion := storkutil.ParseSemanticVersionOrLatest(ctx.subjectDaemon.Version)

	if !config.IsMultiThreadingEnabled(keaVersion) {
		// There is no HA configured.
		return nil, nil
	}

	_, haConfig, ok := config.GetHookLibraries().GetHAHookLibrary()
	if !ok {
		// There is no HA configured.
		return nil, nil
	}

	for _, rel := range haConfig.GetAllRelationships() {
		if rel.IsMultiThreadingEnabled(keaVersion) && !rel.IsDedicatedListenerEnabled(keaVersion) {
			// The dedicated listener is disabled.
			return NewReport(ctx, "The Kea {daemon} daemon is not configured to "+
				"use dedicated HTTP listeners to handle communication between HA "+
				"peers. They will communicate Kea Control Agent. It may cause "+
				"the bottlenecks that nullify any performance gains offered by "+
				"HA+MT. To avoid it, enable the dedicated HTTP listeners in the "+
				"multi-threading configuration of the High-Availability hook. "+
				"Remember that the dedicated listeners must be configured to use "+
				"the HTTP port different from the one used by the Kea Control "+
				"Agent.").referencingDaemon(ctx.subjectDaemon).create()
		}
	}

	// The loop checks if the subject daemon connects directly to the
	// dedicated listeners on the external peers.
	for _, rel := range haConfig.GetAllRelationships() {
		for _, peer := range rel.Peers {
			if !peer.IsValid() {
				// Invalid peer. Skip.
				continue
			}

			urlObj, err := url.Parse(*peer.URL)
			if err != nil {
				// It should never happen. Kea disallows invalid URLs.
				continue
			}

			peerPort, err := strconv.ParseInt(urlObj.Port(), 10, 64)
			if err != nil {
				// It should never happen. Kea disallows invalid URLs.
				continue
			}

			peerAddress := urlObj.Hostname()

			// Fetch the external peer machine from the database.
			accessPointType := dbmodel.AccessPointControl
			peerMachine, err := dbmodel.GetMachineByAddressAndAccessPointPort(
				ctx.db, peerAddress, peerPort, &accessPointType,
			)
			if err != nil {
				return nil, err
			}

			if peerMachine == nil {
				// Peer port doesn't collide with the CA port or the peer is not
				// monitored by Stork. Skip.
				continue
			}

			// Collect the Kea Control Agent daemons.
			// There is no possibility of binding the access point to a
			// specific CA daemon.
			var caDaemons []*dbmodel.Daemon
			for _, peerApp := range peerMachine.Apps {
				// Search for an application that contains the collided access point.
				for _, peerAccessPoint := range peerApp.AccessPoints {
					if peerAccessPoint.Port != peerPort {
						continue
					}

					for _, peerDaemon := range peerApp.Daemons {
						if peerDaemon.ID == ctx.subjectDaemon.ID {
							// Prevent referencing the subject daemon twice.
							continue
						}

						if peerDaemon.Name == dbmodel.DaemonNameCA {
							caDaemons = append(caDaemons, peerDaemon)
						}
					}
				}
			}

			report := NewReport(ctx, fmt.Sprintf("The {daemon} has enabled High "+
				"Availability hook configured to use dedicated HTTP "+
				"listeners but the connections to the HA '%s' peer with the '%s' "+
				"URL are performed over the Kea Control Agent omitting the "+
				"dedicated HTTP listener of this peer. It may cause the "+
				"bottlenecks that nullify any performance gains offered by HA+MT"+
				"You need to set the peer's HTTP '%d' port to the dedicated "+
				"listener's port.", *peer.Name, *peer.URL, peerPort)).
				referencingDaemon(ctx.subjectDaemon)
			for _, daemon := range caDaemons {
				report = report.referencingDaemon(daemon)
			}
			return report.create()
		}
	}

	return nil, nil
}

// The checker validates when a size of pool equals to the number of
// reservations.
func addressPoolsExhaustedByReservations(ctx *ReviewContext) (*Report, error) {
	if ctx.subjectDaemon.Name != dbmodel.DaemonNameDHCPv4 &&
		ctx.subjectDaemon.Name != dbmodel.DaemonNameDHCPv6 {
		return nil, errors.Errorf("unsupported daemon %s", ctx.subjectDaemon.Name)
	}

	type reportData struct {
		Subnet keaconfig.Subnet
		Pool   string
	}

	// Retrieve the subnet data (IPv4 and IPv6).
	config := ctx.subjectDaemon.KeaDaemon.Config
	subnets := config.GetSubnets()

	// Collected data to report.
	const maxIssues = 10
	var issues []reportData

	// Get hosts from the database when libdhcp_host_cmds hooks library is used.
	_, dbHosts, err := getDaemonHostsAndIndexBySubnet(ctx)
	if err != nil {
		return nil, err
	}

	for _, subnet := range subnets {
		// Parse all reservations in a subnet.
		reservedAddresses := []*storkutil.ParsedIP{}

		for _, reservation := range subnet.GetReservations() {
			if reservation.IPAddress != "" {
				parsedIP := storkutil.ParseIP(reservation.IPAddress)
				reservedAddresses = append(reservedAddresses, parsedIP)
			}
			for _, address := range reservation.IPAddresses {
				parsedIP := storkutil.ParseIP(address)
				reservedAddresses = append(reservedAddresses, parsedIP)
			}
		}

		for _, host := range dbHosts[subnet.GetID()] {
			for _, reservation := range host.GetIPReservations() {
				parsedIP := storkutil.ParseIP(reservation)
				if !parsedIP.Prefix {
					reservedAddresses = append(reservedAddresses, parsedIP)
				}
			}
		}

		// Iterate over the address pools and check if they are exhausted by
		// IP reservations.
		for _, pool := range subnet.GetPools() {
			// Parse a pool.
			lb, ub, err := storkutil.ParseIPRange(pool.Pool)
			if err != nil {
				continue
			}

			// Calculate the pool size.
			poolSize := storkutil.CalculateRangeSize(lb, ub)

			// Count the reservations in a pool.
			reservationsInPoolCount := storkutil.NewBigCounter(0)
			for _, address := range reservedAddresses {
				if address.IsInRange(lb, ub) {
					// Increment by one.
					reservationsInPoolCount.AddUint64(1)
				}
			}

			if poolSize.Cmp(reservationsInPoolCount.ToBigInt()) > 0 {
				// The pool size is greater than the number of reservations
				// within it.
				continue
			}

			// The pool size equals to the number of reservations. Add to report.
			issues = append(issues, reportData{subnet, pool.Pool})

			if len(issues) == maxIssues {
				// Found a maximum number of the affected pools. Early stop.
				break
			}
		}

		if len(issues) == maxIssues {
			// Found a maximum number of the affected pools. Early stop.
			break
		}
	}

	if len(issues) == 0 {
		// No affected pools found.
		return nil, nil
	}

	// Format affected pool messages.
	messages := make([]string, len(issues))
	for i, issue := range issues {
		subnetID := ""
		if issue.Subnet.GetID() != 0 {
			subnetID = fmt.Sprintf("[%d] ", issue.Subnet.GetID())
		}

		messages[i] = fmt.Sprintf(
			"%d. Pool '%s' of the '%s%s' subnet.",
			i+1, issue.Pool, subnetID, issue.Subnet.GetPrefix(),
		)
	}

	// Format the message about a count of affected pool messages.
	countMessage := fmt.Sprintf("First %d affected pools", maxIssues)
	if len(issues) < maxIssues {
		countMessage = fmt.Sprintf(
			"Found %s",
			storkutil.FormatNoun(
				int64(len(issues)),
				"affected pool",
				"s",
			),
		)
	}

	// Format the final report.
	return NewReport(ctx, fmt.Sprintf("Kea {daemon} configuration contains "+
		"address pools with the number of in-pool IP reservations equal "+
		"to their size. The devices lacking the reservations will not get "+
		"addresses from these pools. %s:\n%s",
		countMessage, strings.Join(messages, "\n"))).
		referencingDaemon(ctx.subjectDaemon).
		create()
}

// The checker validates when a size of delegated prefix pool equals to the
// number of reservations.
func delegatedPrefixPoolsExhaustedByReservations(ctx *ReviewContext) (*Report, error) {
	if ctx.subjectDaemon.Name != dbmodel.DaemonNameDHCPv4 &&
		ctx.subjectDaemon.Name != dbmodel.DaemonNameDHCPv6 {
		return nil, errors.Errorf("unsupported daemon %s", ctx.subjectDaemon.Name)
	}

	type reportData struct {
		Subnet keaconfig.Subnet
		Pool   string
	}

	// Retrieve the subnet data (IPv4 and IPv6).
	config := ctx.subjectDaemon.KeaDaemon.Config
	subnets := config.GetSubnets()

	// Collected data to report.
	const maxIssues = 10
	var issues []reportData

	// Get hosts from the database when libdhcp_host_cmds hooks library is used.
	_, dbHosts, err := getDaemonHostsAndIndexBySubnet(ctx)
	if err != nil {
		return nil, err
	}

	for _, subnet := range subnets {
		// Parse all reservations in a subnet.
		reservedPrefixes := []*storkutil.ParsedIP{}

		for _, reservation := range subnet.GetReservations() {
			for _, prefix := range reservation.Prefixes {
				parsedIP := storkutil.ParseIP(prefix)
				reservedPrefixes = append(reservedPrefixes, parsedIP)
			}
		}

		for _, host := range dbHosts[subnet.GetID()] {
			for _, reservation := range host.GetIPReservations() {
				parsedIP := storkutil.ParseIP(reservation)
				if parsedIP.Prefix {
					reservedPrefixes = append(reservedPrefixes, parsedIP)
				}
			}
		}

		// Iterate over the PD pools and check if they are exhausted by
		// IP reservations.
		for _, pool := range subnet.GetPDPools() {
			// Calculate the pool size.
			// Pool size = 2 power to the number of the wildcard bytes.
			poolSize := storkutil.CalculateDelegatedPrefixRangeSize(
				pool.PrefixLen, pool.DelegatedLen,
			)

			// Count the reservations in a pool.
			reservationsInPoolCount := storkutil.NewBigCounter(0)
			for _, prefix := range reservedPrefixes {
				if prefix.IsInPrefixRange(pool.Prefix, pool.PrefixLen, pool.DelegatedLen) {
					reservationsInPoolCount.AddUint64(1)
				}
			}

			if poolSize.Cmp(reservationsInPoolCount.ToBigInt()) > 0 {
				// The pool size is greater than the number of reservations
				// within it.
				continue
			}

			// The pool size equals to the number of reservations. Add to report.
			issues = append(issues, reportData{
				subnet,
				fmt.Sprintf(
					"%s del. %d",
					storkutil.FormatCIDRNotation(pool.Prefix, pool.PrefixLen),
					pool.DelegatedLen,
				),
			})

			if len(issues) == maxIssues {
				// Found a maximum number of the affected pools. Early stop.
				break
			}
		}

		if len(issues) == maxIssues {
			// Found a maximum number of the affected pools. Early stop.
			break
		}
	}

	if len(issues) == 0 {
		// No affected pools found.
		return nil, nil
	}

	// Format affected pool messages.
	messages := make([]string, len(issues))
	for i, issue := range issues {
		subnetID := ""
		if issue.Subnet.GetID() != 0 {
			subnetID = fmt.Sprintf("[%d] ", issue.Subnet.GetID())
		}

		messages[i] = fmt.Sprintf(
			"%d. Pool '%s' of the '%s%s' subnet.",
			i+1, issue.Pool, subnetID, issue.Subnet.GetPrefix(),
		)
	}

	// Format the message about a count of affected pool messages.
	countMessage := fmt.Sprintf("First %d affected pools", maxIssues)
	if len(issues) < maxIssues {
		countMessage = fmt.Sprintf(
			"Found %s",
			storkutil.FormatNoun(
				int64(len(issues)),
				"affected pool",
				"s",
			),
		)
	}

	// Format the final report.
	return NewReport(ctx, fmt.Sprintf("Kea {daemon} configuration contains "+
		"delegated prefix pools with the number of in-pool PD reservations equal "+
		"to their size. The devices lacking the reservations will not get "+
		"prefix from these pools. %s:\n%s",
		countMessage, strings.Join(messages, "\n"))).
		referencingDaemon(ctx.subjectDaemon).
		create()
}

// The checker validates that the subnet commands hook is not used mutually
// with the config backend.
func subnetCmdsAndConfigBackendMutualExclusion(ctx *ReviewContext) (*Report, error) {
	if ctx.subjectDaemon.Name != dbmodel.DaemonNameDHCPv4 &&
		ctx.subjectDaemon.Name != dbmodel.DaemonNameDHCPv6 {
		return nil, errors.Errorf("unsupported daemon %s", ctx.subjectDaemon.Name)
	}

	config := ctx.subjectDaemon.KeaDaemon.Config

	if _, _, present := config.GetHookLibrary("libdhcp_subnet_cmds"); !present {
		// Missing subnet commands hook.
		return nil, nil
	}

	databases := config.GetAllDatabases()
	if len(databases.Config) == 0 {
		// Missing a config backend.
		return nil, nil
	}

	return NewReport(
		ctx,
		"It is recommended that the 'subnet_cmds' hook library not be used "+
			"to manage subnets when the configuration backend is used as a "+
			"source of information about the subnets. The 'subnet_cmds' hook "+
			"library modifies the local subnets configuration in the server's "+
			"memory, not in the database. Use the 'cb_cmds' hook library to "+
			"manage the subnets information in the database instead.",
	).referencingDaemon(ctx.subjectDaemon).create()
}

// The checker validates that the Stork agent communicates with the Kea Control
// Agent using the HTTPS protocol when the HTTP authentication credentials
// (i.e., Basic Auth) are configured.
func credentialsOverHTTPS(ctx *ReviewContext) (*Report, error) {
	daemon := ctx.subjectDaemon
	if daemon.Name != dbmodel.DaemonNameCA {
		return nil, errors.Errorf("unsupported daemon %s", daemon.Name)
	}

	config := daemon.KeaDaemon.Config

	if config.Authentication == nil || !config.Authentication.IsBasicAuth() {
		// The Basic Auth credentials are not configured.
		return nil, nil
	}

	if config.UseSecureProtocol() {
		// The TLS is configured. All is OK.
		return nil, nil
	}

	// The Stork agent has HTTP credentials configured but communicates with
	// Kea over unsecure protocol.
	return NewReport(ctx, "The Kea Control Agent requires the Basic Auth "+
		"credentials but it accepts connections over unsecure protocol - TLS "+
		"is disabled. The communication between the Stork Agent and Kea Control "+
		"Agent is vulnerable to man-in-the-middle attacks and the credentials "+
		"may be stolen. "+
		"Configure the 'trust-anchor', 'cert-file', and 'key-file' "+
		"properties in the Kea Control Agent {daemon} configuration to use "+
		"the secure protocol.").referencingDaemon(daemon).create()
}

// The checker validates that the control sockets of Kea Control Agent are
// configured.
func controlSocketsCA(ctx *ReviewContext) (*Report, error) {
	if ctx.subjectDaemon.Name != dbmodel.DaemonNameCA {
		return nil, errors.Errorf("unsupported daemon %s", ctx.subjectDaemon.Name)
	}

	config := ctx.subjectDaemon.KeaDaemon.Config
	controlSockets := config.GetControlSockets()

	switch {
	case controlSockets == nil:
		return NewReport(ctx, "The control sockets are not specified in the "+
			"Kea Control Agent {daemon} configuration. It causes the Kea "+
			"Control Agent to not connect to the Kea daemons, so Stork cannot "+
			"monitor them. You need to provide the proper socket paths in the "+
			"\"control-sockets\" top-level entry.").
			referencingDaemon(ctx.subjectDaemon).create()
	case !controlSockets.HasAnyConfiguredDaemon():
		return NewReport(ctx, "The control sockets entry in the Kea Control "+
			"Agent {daemon} configuration is empty. It causes the Kea "+
			"Control Agent to not connect to the Kea daemons, so Stork cannot "+
			"monitor them. You need to provide the proper socket paths in the "+
			"\"control-sockets\" top-level entry.").
			referencingDaemon(ctx.subjectDaemon).create()
	case controlSockets.Dhcp4 == nil && controlSockets.Dhcp6 == nil:
		return NewReport(ctx, "The control sockets entry in the Kea Control "+
			"Agent {daemon} configuration doesn't contain path to any DHCP "+
			"daemon, so Stork cannot detect them. You need to provide the "+
			"proper socket paths in the \"dhcp4\" and/or \"dhcp6\" properties "+
			"of the \"control-sockets\" top-level entry.").
			referencingDaemon(ctx.subjectDaemon).create()
	default:
		return nil, nil
	}
}

// Gathering statistics is unavailable when all of the following conditions are met:
//  1. Kea's version is between 2.3.0 (inclusive, but I'm not sure exactly which
//     version the problem was introduced) and 2.5.3 (exclusive).
//  2. There is configured a subnet or shared network with more than 2^63-1
//     addresses or a delegated prefix pool with more than 2^63-1 prefixes.
//  3. The stat_cmds hook is loaded.
func gatheringStatisticsUnavailableDueToNumberOverflow(ctx *ReviewContext) (*Report, error) {
	// Check the daemon type.
	if ctx.subjectDaemon.Name != dbmodel.DaemonNameDHCPv4 &&
		ctx.subjectDaemon.Name != dbmodel.DaemonNameDHCPv6 {
		return nil, errors.Errorf("unsupported daemon %s", ctx.subjectDaemon.Name)
	}

	// Check the statistics hook presence.
	config := ctx.subjectDaemon.KeaDaemon.Config
	if _, _, present := config.GetHookLibrary("libdhcp_stat_cmds"); !present {
		// The stat hook is not loaded.
		return nil, nil
	}

	// Check the Kea version.
	daemonVersion := storkutil.ParseSemanticVersionOrLatest(ctx.subjectDaemon.Version)
	if daemonVersion.GreaterThanOrEqual(storkutil.NewSemanticVersion(2, 5, 3)) {
		// Fully supported version.
		return nil, nil
	}

	// Look for the subnets and shared networks with the number of addresses
	// that cause the statistics overflow.
	sharedNetworks := config.GetSharedNetworks(true)
	isOverflow, overflowReason, err := findSharedNetworkExceedingAddressLimit(sharedNetworks)
	if err != nil {
		return nil, err
	}

	// Look for the subnets and shared networks with the number of delegated
	// prefixes that cause the statistics overflow.
	if !isOverflow && ctx.subjectDaemon.Name == dbmodel.DaemonNameDHCPv6 {
		isOverflow, overflowReason = findSharedNetworkExceedingDelegatedPrefixLimit(sharedNetworks)
	}

	if !isOverflow {
		// No overflow detected.
		return nil, nil
	}

	if daemonVersion.LessThan(storkutil.NewSemanticVersion(2, 3, 0)) {
		// The gathering statistics works but the exact values are not accurate.
		return NewReport(ctx, fmt.Sprintf(
			"The Kea {daemon} daemon has configured some very large "+
				"pools. The installed Kea version doesn't handle the "+
				"statistics for so large pools properly. The "+
				"statistics presented by Stork and Prometheus/Grafana may "+
				"be inaccurate. Details: %s.", overflowReason,
		)).referencingDaemon(ctx.subjectDaemon).create()
	} else {
		// The gathering statistics doesn't work.
		return NewReport(ctx, fmt.Sprintf(
			"The Kea {daemon} daemon has configured some very large "+
				"pools. The installed Kea version doesn't handle the "+
				"statistics for so large pools properly. Stork "+
				"is unable to fetch them. Details: %s.", overflowReason,
		)).referencingDaemon(ctx.subjectDaemon).create()
	}
}

// Determines if any of the provided shared networks has more than 2^63-1
// addresses. It is expected that one of the shared networks represents the
// global subnet scope. The global subnet scope has no limit on the total
// number of addresses but still has a limit on the number of addresses in
// a single subnet.
// Returns the boolean value indicating if the overflow is detected, the
// reason of the overflow (overflow in a single subnet or shared network),
// and an error, if any.
func findSharedNetworkExceedingAddressLimit(sharedNetworks []keaconfig.SharedNetwork) (bool, string, error) {
	// Check the shared networks.
	for _, sharedNetwork := range sharedNetworks {
		sharedNetworkSize := big.NewInt(0)
		for _, subnet := range sharedNetwork.GetSubnets() {
			subnetSize := big.NewInt(0)
			for _, pool := range subnet.GetPools() {
				lower, upper, err := pool.GetBoundaries()
				if err != nil {
					return false, "", errors.WithMessagef(
						err,
						"could not parse the pool boundaries for the %s subnet",
						subnet.GetPrefix(),
					)
				}
				poolSize := storkutil.CalculateRangeSize(lower, upper)

				subnetSize.Add(
					subnetSize,
					poolSize,
				)

				if !subnetSize.IsInt64() {
					// Subnet itself cannot exceed the 2^63-1 addresses.
					return true, fmt.Sprintf(
						"the '%s' subnet has more than 2^63-1 addresses",
						subnet.GetPrefix(),
					), nil
				}
			}

			sharedNetworkSize.Add(
				sharedNetworkSize,
				subnetSize,
			)

			if sharedNetwork.GetName() != "" && !sharedNetworkSize.IsInt64() {
				// None of the shared networks can exceed the 2^63-1 addresses.
				// The global subnet scope has no limit.
				return true, fmt.Sprintf(
					"the '%s' shared network has more than 2^63-1 addresses",
					sharedNetwork.GetName(),
				), nil
			}
		}
	}
	return false, "", nil
}

// Determines if any of the provided shared networks has more than 2^63-1
// delegated prefixes. It is expected that one of the shared networks
// represents the global subnet scope. The global subnet scope has no limit
// on the total number of delegated prefixes but still have a limit on the
// number of delegated prefixes in a single subnet.
// Returns the boolean value indicating if the overflow is detected, and the
// reason of the overflow (overflow in a single subnet or shared network).
func findSharedNetworkExceedingDelegatedPrefixLimit(sharedNetworks []keaconfig.SharedNetwork) (bool, string) {
	// Check the shared networks.
	for _, sharedNetwork := range sharedNetworks {
		sharedNetworkSize := big.NewInt(0)
		for _, subnet := range sharedNetwork.GetSubnets() {
			subnetSize := big.NewInt(0)
			for _, pool := range subnet.GetPDPools() {
				poolSize := storkutil.CalculateDelegatedPrefixRangeSize(
					pool.PrefixLen, pool.DelegatedLen,
				)

				subnetSize.Add(
					subnetSize,
					poolSize,
				)

				if !subnetSize.IsInt64() {
					// Subnet itself cannot exceed the 2^63-1 addresses.
					return true, fmt.Sprintf(
						"the '%s' subnet has more than 2^63-1 delegated prefixes",
						subnet.GetPrefix(),
					)
				}
			}

			sharedNetworkSize.Add(
				sharedNetworkSize,
				subnetSize,
			)

			if sharedNetwork.GetName() != "" && !sharedNetworkSize.IsInt64() {
				// None of the shared networks can exceed the 2^63-1 addresses.
				// The global subnet scope has no limit.
				return true, fmt.Sprintf(
					"the '%s' shared network has more than 2^63-1 delegated prefixes",
					sharedNetwork.GetName(),
				)
			}
		}
	}
	return false, ""
}
