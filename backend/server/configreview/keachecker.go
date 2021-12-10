package configreview

import (
	"fmt"

	"github.com/pkg/errors"
	dbmodel "isc.org/stork/server/database/model"
)

// The checker verifying if the stat_cmds hooks library is loaded.
func statCmdsPresence(ctx *ReviewContext) (*Report, error) {
	config := ctx.subjectDaemon.KeaDaemon.Config
	if _, _, present := config.GetHooksLibrary("libdhcp_stat_cmds"); !present {
		r, err := NewReport(ctx, "The Kea Statistics Commands library (libdhcp_stat_cmds) provides commands for retrieving accurate DHCP lease statistics for Kea DHCP servers. Stork sends these commands to fetch lease statistics displayed in the dashboard, subnet, and shared network views. Stork found that {daemon} is not using this hook library. Some statistics will not be available until the library is loaded.").
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
			r, err := NewReport(ctx, "Kea can be configured to store host reservations in a database. Stork can access these reservations using the commands implemented in the Host Commands hook library and make them available in the Host Reservations view. It appears that the libdhcp_host_cmds hooks library is not loaded on {daemon}. Host reservations from the database will not be visible in Stork until this library is enabled.").
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
	emptyCount := 0
	singleCount := 0
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
			details = fmt.Sprintf("%d empty shared network", emptyCount)
			if emptyCount > 1 {
				details += "s"
			}
		}
		if singleCount > 0 {
			if len(details) > 0 {
				details += " and "
			}
			details += fmt.Sprintf("%d shared network", singleCount)
			if singleCount > 1 {
				details += "s"
			}
			details += " with only a single subnet"
		}
		r, err := NewReport(ctx, fmt.Sprintf("Kea {daemon} configuration comprises %s. Using shared networks creates an overhead for a Kea server configuration and DHCP message processing, affecting its performance. It is recommended to remove the shared networks with none or a single subnet and specify these subnets at the global configuration level.", details)).
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
func createSubnetDispensableReport(ctx *ReviewContext, dispensableCount int) (*Report, error) {
	if dispensableCount == 0 {
		return nil, nil
	}
	r, err := NewReport(ctx, fmt.Sprintf("Kea {daemon} configuration comprises %d subnets without pools and host reservations. The DHCP server will not assign any addresses to the devices within this subnet. It is recommended to add some pools or host reservations to this subnet or remove the subnet from the configuration.", dispensableCount)).
		referencingDaemon(ctx.subjectDaemon).
		create()
	return r, err
}

// Implementation of a checker verifying if an IPv4 subnet can be removed
// because it includes no pools and no reservations.
func checkSubnet4Dispensable(ctx *ReviewContext) (*Report, error) {
	type subnet4 struct {
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

	// Iterate over the shared networks and check if they contain any
	// subnets that can be removed.
	dispensableCount := 0
	for _, net := range *decodedSharedNetworks {
		for _, subnet := range net.Subnet4 {
			if len(subnet.Pools) == 0 && len(subnet.Reservations) == 0 {
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

	// Iterate over the shared networks and check if they contain any
	// subnets that can be removed.
	dispensableCount := 0
	for _, net := range *decodedSharedNetworks {
		for _, subnet := range net.Subnet6 {
			if len(subnet.Pools) == 0 && len(subnet.PDPools) == 0 && len(subnet.Reservations) == 0 {
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
	// Skip the check if the host_cmds hooks library is loaded.
	if _, _, present := ctx.subjectDaemon.KeaDaemon.Config.GetHooksLibrary("libdhcp_host_cmds"); present {
		return nil, nil
	}
	if ctx.subjectDaemon.Name == dbmodel.DaemonNameDHCPv4 {
		return checkSubnet4Dispensable(ctx)
	}
	return checkSubnet6Dispensable(ctx)
}
