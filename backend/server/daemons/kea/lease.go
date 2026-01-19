package kea

import (
	"context"
	"net"
	"reflect"
	"slices"

	errors "github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	keactrl "isc.org/stork/daemonctrl/kea"
	keadata "isc.org/stork/daemondata/kea"
	"isc.org/stork/datamodel/daemonname"
	"isc.org/stork/server/agentcomm"
	dbops "isc.org/stork/server/database"
	dbmodel "isc.org/stork/server/database/model"
	storkutil "isc.org/stork/util"
)

// Structure representing a response to a command fetching a
// single DHCPv4 lease from the Kea server.
type Lease4GetResponse struct {
	keactrl.ResponseHeader
	Arguments *dbmodel.Lease `json:"arguments,omitempty"`
}

// Structure representing a response to a command fetching a
// single DHCPv6 lease from the Kea server.
type Lease6GetResponse struct {
	keactrl.ResponseHeader
	Arguments *dbmodel.Lease `json:"arguments,omitempty"`
}

// Structure representing arguments of a response to a command
// fetching multiple DHCP leases from the Kea server.
type LeaseGetMultipleResponseArgs struct {
	Leases []dbmodel.Lease
}

// Structure representing a response to a command fetching multiple
// DHCP leases from the Kea server.
type LeaseGetMultipleResponse struct {
	keactrl.ResponseHeader
	Arguments *LeaseGetMultipleResponseArgs `json:"arguments,omitempty"`
}

// Validates a response from a Kea daemon to the commands fetching
// leases, e.g. lease4-get-by-hw-address. It checks that the response
// comprises the Success status and that arguments map is not nil.
func validateGetLeasesResponse(commandName keactrl.CommandName, result keactrl.ResponseResult, arguments interface{}) error {
	if result == keactrl.ResponseError {
		return errors.Errorf("error returned by Kea in response to %s command", commandName)
	}
	if result == keactrl.ResponseCommandUnsupported {
		return errors.Errorf("%s command unsupported", commandName)
	}
	argumentsType := reflect.TypeOf(arguments)
	if argumentsType != nil && argumentsType.Kind() == reflect.Ptr {
		if reflect.ValueOf(arguments).IsNil() {
			return errors.Errorf("response to %s command lacks arguments", commandName)
		}
	}
	return nil
}

// Sends a lease4-get command with ip-address argument specifying a searched lease.
// If the lease is found, the pointer to it is returned. If the lease does not
// exist, a nil pointer and nil error are returned.
func GetLease4ByIPAddress(agents agentcomm.ConnectedAgents, daemon *dbmodel.Daemon, ipAddress string) (lease *dbmodel.Lease, err error) {
	command := keactrl.NewCommandLease4Get(ipAddress)
	var response Lease4GetResponse
	ctx := context.Background()
	respResult, err := agents.ForwardToKeaOverHTTP(ctx, daemon, []keactrl.SerializableCommand{command}, &response)
	if err != nil {
		return lease, err
	}
	if err := respResult.GetFirstError(); err != nil {
		return lease, err
	}
	if response.Result == keactrl.ResponseEmpty {
		return lease, nil
	}
	if err = validateGetLeasesResponse("lease4-get", response.Result, response.Arguments); err != nil {
		return lease, err
	}
	lease = response.Arguments
	lease.DaemonID = daemon.ID
	lease.Daemon = daemon
	return lease, nil
}

// Sends a lease6-get command with type and ip-address arguments specifying
// searched lease type and IP address. If the lease is found, the pointer to
// it is returned. If the lease does not exist, a nil pointer and nil error
// are returned.
func GetLease6ByIPAddress(agents agentcomm.ConnectedAgents, daemon *dbmodel.Daemon, leaseType keactrl.LeaseType, ipAddress string) (lease *dbmodel.Lease, err error) {
	command := keactrl.NewCommandLease6Get(leaseType, ipAddress)
	var response Lease6GetResponse
	ctx := context.Background()
	respResult, err := agents.ForwardToKeaOverHTTP(ctx, daemon, []keactrl.SerializableCommand{command}, &response)
	if err != nil {
		return lease, err
	}
	if respResult.Error != nil {
		return lease, respResult.Error
	}
	if response.Result == keactrl.ResponseEmpty {
		return lease, nil
	}
	if err = validateGetLeasesResponse(keactrl.Lease6Get, response.Result, response.Arguments); err != nil {
		return lease, err
	}
	lease = response.Arguments
	lease.DaemonID = daemon.ID
	lease.Daemon = daemon
	return lease, nil
}

// This is a generic function querying a Kea server for leases by specified lease
// properties: hw-address, client-id, DUID or hostname. The type of the property
// is unknown to the function and therefore it sends multiple commands to the Kea
// server using the property value as an input to different commands. It is up
// to the caller to decide which commands this function should send to Kea. For
// example, if the property value is 01:01:01:01, the caller should select the
// lease4-get-by-hw-address, lease4-get-by-client-id and lease6-get-by-duid
// commands. The specified commands are combined in a single gRPC transaction
// to minimize the number of roundtrips between the Stork Server and an agent.
func getLeasesByProperties(agents agentcomm.ConnectedAgents, daemon *dbmodel.Daemon, propertyValue string, commandNames ...keactrl.CommandName) (leases []dbmodel.Lease, warns bool, err error) {
	var commands []keactrl.SerializableCommand
	for _, commandName := range commandNames {
		var command *keactrl.Command
		propertyValue := propertyValue // Make a copy to ensure changes do not affect other commands.
		switch commandName {
		case keactrl.Lease4GetByHWAddress:
			// Searching by empty MAC address is allowed when trying to find declined leases.
			// If the value is non-empty, it has to be properly formatted.
			if len(propertyValue) > 0 {
				// When searching by MAC address we must ensure that it has the format
				// expected by Kea, i.e. 01:02:03:04:05:06.
				if formattedPropertyValue, ok := storkutil.FormatMACAddress(propertyValue); ok {
					propertyValue = formattedPropertyValue
				} else {
					return leases, false, errors.Errorf("invalid format of the property %s used to get leases by MAC address from Kea", propertyValue)
				}
			}
			command = keactrl.NewCommandLease4GetByHWAddress(propertyValue)
		case keactrl.Lease4GetByClientID:
			command = keactrl.NewCommandLease4GetByClientID(propertyValue)
		case keactrl.Lease6GetByDUID:
			// Kea does not accept empty DUIDs. Empty DUID in Kea is represented by 1 zero byte (Kea < 2.3.8) or 3 zero bytes (Kea >= 2.3.8).
			if len(propertyValue) == 0 {
				semver := storkutil.ParseSemanticVersionOrLatest(daemon.Version)
				if semver.LessThan(storkutil.NewSemanticVersion(2, 3, 8)) {
					propertyValue = "0"
				} else {
					propertyValue = "00:00:00"
				}
			}
			command = keactrl.NewCommandLease6GetByDUID(propertyValue)
		case keactrl.Lease4GetByHostname:
			command = keactrl.NewCommandLease4GetByHostname(propertyValue)
		case keactrl.Lease6GetByHostname:
			command = keactrl.NewCommandLease6GetByHostname(propertyValue)
		case keactrl.Lease4GetByState:
			command = keactrl.NewCommandLease4GetByState(propertyValue)
		case keactrl.Lease6GetByState:
			command = keactrl.NewCommandLease6GetByState(propertyValue)
		default:
			continue
		}
		commands = append(commands, command)
	}

	// A caller specified no commands or command names were invalid.
	if len(commands) == 0 {
		return leases, false, nil
	}

	// Create container for responses to each command sent.
	var responses []any
	for range commands {
		var response LeaseGetMultipleResponse
		responses = append(responses, &response)
	}

	ctx := context.Background()

	// Send all commands to Kea via Stork Agent.
	respResult, err := agents.ForwardToKeaOverHTTP(ctx, daemon, commands, responses...)
	if err != nil {
		return leases, false, err
	}

	if respResult.Error != nil {
		return leases, false, respResult.Error
	}

	// Validate responses to all commands.
	for i, r := range responses {
		response := r.(*LeaseGetMultipleResponse)

		// Ignore empty response. It is valid but there are no leases,
		// so there is nothing more to do.
		if response.Result != keactrl.ResponseEmpty {
			if err = validateGetLeasesResponse(commands[i].GetCommand(), response.Result, response.Arguments); err != nil {
				// Log an error and continue. Maybe there is a communication problem
				// with one daemon, but the other one is still operational.
				log.WithError(err).Warn("Validation of response to command fetching leases failed")
				warns = true
			} else {
				leases = append(leases, response.Arguments.Leases...)
			}
		}
	}
	for i := range leases {
		leases[i].DaemonID = daemon.ID
		leases[i].Daemon = daemon
	}
	return leases, warns, nil
}

// Sends lease4-get-by-hw-address command to Kea.
func GetLeases4ByHWAddress(agents agentcomm.ConnectedAgents, dbDaemon *dbmodel.Daemon, hwAddress string) (leases []dbmodel.Lease, err error) {
	leases, _, err = getLeasesByProperties(agents, dbDaemon, hwAddress, "lease4-get-by-hw-address")
	return leases, err
}

// Sends lease4-get-by-client-id command to Kea.
func GetLeases4ByClientID(agents agentcomm.ConnectedAgents, dbDaemon *dbmodel.Daemon, clientID string) (leases []dbmodel.Lease, err error) {
	leases, _, err = getLeasesByProperties(agents, dbDaemon, clientID, "lease4-get-by-client-id")
	return leases, err
}

// Sends lease4-get-by-hostname command to Kea.
func GetLeases4ByHostname(agents agentcomm.ConnectedAgents, dbDaemon *dbmodel.Daemon, hostname string) (leases []dbmodel.Lease, err error) {
	leases, _, err = getLeasesByProperties(agents, dbDaemon, hostname, "lease4-get-by-hostname")
	return leases, err
}

// Sends lease6-get-by-duid command to Kea.
func GetLeases6ByDUID(agents agentcomm.ConnectedAgents, dbDaemon *dbmodel.Daemon, duid string) (leases []dbmodel.Lease, err error) {
	leases, _, err = getLeasesByProperties(agents, dbDaemon, duid, "lease6-get-by-duid")
	return leases, err
}

// Sends lease6-get-by-hostname command to Kea.
func GetLeases6ByHostname(agents agentcomm.ConnectedAgents, dbDaemon *dbmodel.Daemon, hostname string) (leases []dbmodel.Lease, err error) {
	leases, _, err = getLeasesByProperties(agents, dbDaemon, hostname, "lease6-get-by-hostname")
	return leases, err
}

// Convenience function checking if a given daemon has the libdhcp_lease_cmds
// hooks library configured.
func hasLeaseCmdsHook(daemon *dbmodel.Daemon) bool {
	if daemon != nil && daemon.KeaDaemon != nil && daemon.KeaDaemon.Config != nil {
		if _, _, ok := daemon.KeaDaemon.Config.GetHookLibrary("libdhcp_lease_cmds"); ok {
			return true
		}
	}
	return false
}

// Attempts to find a lease on the Kea servers by specified text.
// It expects that the text is an IP address, MAC address, client
// identifier, or hostname matching a lease. The server contacts
// all Kea servers, which may potentially have the lease. If
// multiple servers have the same lease (e.g. in HA configuration),
// it returns all those lease instances. The Kea servers that
// returned an error response are returned in the second value.
// Such failures do not preclude the function from returning
// leases found on other servers, but the caller becomes aware
// that some leases may not be included due to the communication
// errors with some servers. The third returned value indicates
// a general error, e.g. issues with Stork database communication.
func FindLeases(db *dbops.PgDB, agents agentcomm.ConnectedAgents, text string) (leases []dbmodel.Lease, erredDaemons []*dbmodel.Daemon, err error) {
	// Recognize if the text comprises an IP address or some identifier,
	// e.g. MAC address or client identifier.
	const (
		ipv4 = iota
		ipv6
		identifier
		hostname
	)
	// By default query by hostname.
	queryType := hostname
	if ip := net.ParseIP(text); ip != nil {
		// It is an IP address. If it converts to 4 bytes, it is
		// an IPv4 address. Otherwise, it is an IPv6 address.
		if ip.To4() != nil {
			queryType = ipv4
		} else {
			queryType = ipv6
		}
	} else if storkutil.IsHexIdentifier(text) {
		// It is a string of hexadecimal digits, so it must be one
		// of the identifiers.
		queryType = identifier
	}

	// Get Kea daemons from the database. We will send commands to these
	// daemons to find leases.
	daemons, err := dbmodel.GetDHCPDaemons(db)
	if err != nil {
		err = errors.WithMessagef(err, "failed to fetch Kea daemons while searching for leases by %s", text)
		return leases, erredDaemons, err
	}

	for i := range daemons {
		daemonError := false
		daemon := &daemons[i]

		if !hasLeaseCmdsHook(daemon) {
			continue
		}

		switch queryType {
		case ipv4:
			if daemon.Name != daemonname.DHCPv4 {
				// This is not a DHCPv4 daemon, so skip it.
				continue
			}
			// This is an IPv4 address, so send the command to the DHCPv4 server.
			lease, err := GetLease4ByIPAddress(agents, daemon, text)
			if err != nil {
				daemonError = true
				log.WithError(err).Error("Failed to fetch lease by IPv4 address from Kea")
			} else if lease != nil {
				leases = append(leases, *lease)
			}
		case ipv6:
			if daemon.Name != daemonname.DHCPv6 {
				// This is not a DHCPv6 daemon, so skip it.
				continue
			}
			// This is an IPv6 address (or prefix), so send the command to the
			// DHCPv6 server instead.
			for _, leaseType := range []keactrl.LeaseType{keactrl.LeaseTypeNA, keactrl.LeaseTypePD} {
				lease, err := GetLease6ByIPAddress(agents, daemon, leaseType, text)
				if err != nil {
					daemonError = true
					log.WithError(err).Error("Failed to fetch lease by IPv6 address from Kea")
				} else if lease != nil {
					leases = append(leases, *lease)
					// If we found a lease by IP address there is no reason to
					// query by delegated prefix because the IP address/prefix
					// must be unique in the database.
					break
				}
			}
		default:
			// The remaining cases are to query by identifier or hostname. They share
			// lots of common code, so they are combined in their own switch statement.
			var commands []keactrl.CommandName

			// From Kea 2.3.8, the DUID identifier must be at least 3 bytes long.
			// If the value is shorter, it must be a hostname.
			if queryType == identifier && storkutil.CountHexIdentifierBytes(text) < 3 {
				semver := storkutil.ParseSemanticVersionOrLatest(daemon.Version)
				if semver.GreaterThanOrEqual(storkutil.NewSemanticVersion(2, 3, 8)) {
					// It isn't an identifier in the Kea 2.3.8+ sense.
					queryType = hostname
				}
			}

			switch queryType {
			case identifier:
				if daemon.Name == daemonname.DHCPv4 {
					commands = append(commands, "lease4-get-by-hw-address", "lease4-get-by-client-id")
				}
				if daemon.Name == daemonname.DHCPv6 {
					commands = append(commands, "lease6-get-by-duid")
				}
			default:
				if daemon.Name == daemonname.DHCPv4 {
					commands = append(commands, "lease4-get-by-hostname")
				}
				if daemon.Name == daemonname.DHCPv6 {
					commands = append(commands, "lease6-get-by-hostname")
				}
			}
			// Search for leases by identifier or hostname.
			leasesByProperties, warns, err := getLeasesByProperties(agents, daemon, text, commands...)
			daemonError = warns
			if err != nil {
				daemonError = true
				log.WithError(err).Errorf("Failed to fetch leases from Kea [%d] %s daemon", daemon.ID, daemon.Name)
			} else {
				leases = append(leases, leasesByProperties...)
			}
		}
		if daemonError {
			erredDaemons = append(erredDaemons, daemon)
		}
	}
	return leases, erredDaemons, nil
}

// Attempts to find declined leases on the Kea servers. Prior to v3.1.5, Kea
// provided no API to search the leases by state. However, the declined leases
// were stored with HW address and DUID empty. Thus, for Kea 3.1.4 and earlier,
// this function sends lease4-get-by-hw-address and lease6-get-by-duid with
// empty hw-address and empty duid parameters respectively. Next, it removes the
// leases which are not in the declined state from the result. The Kea servers
// that returned an error response are returned in the second value. Such
// failures do not preclude the function from returning leases found on other
// servers, but the caller becomes aware that some leases may not be included
// due to the communication errors with some servers. The third returned value
// indicates a general error, e.g. issues with Stork database communication.
//
// For daemons running Kea 3.1.5 or newer, it sends the "lease[46]-get-by-state"
// comman and returns the results.
func FindDeclinedLeases(db *dbops.PgDB, agents agentcomm.ConnectedAgents) (leases []dbmodel.Lease, erredDaemons []*dbmodel.Daemon, err error) {
	// Get all Kea daemons.
	daemons, err := dbmodel.GetDHCPDaemons(db)
	if err != nil {
		err = errors.WithMessagef(err, "failed to fetch Kea daemons while searching for declined leases")
		return leases, erredDaemons, err
	}

	// Kea version with leases[46]-get-by-state.
	keaVerWithState := storkutil.NewSemanticVersion(3, 1, 5)
	keaVerWhereBroken := storkutil.NewSemanticVersion(3, 1, 0)

	// Send appropriate commands to each daemon.
	for i := range daemons {
		daemon := &daemons[i]
		daemonError := false

		if !hasLeaseCmdsHook(daemon) {
			// This daemon does not have the libdhcp_lease_cmds hook library configured
			// so it cannot return leases.
			continue
		}

		var commands []keactrl.CommandName

		// Future enhancement: remove all of the version checks and preserve only the
		// supportsGetByState code once Kea 3.1.4 is no longer supported (April 2026).
		daemonVer, err := storkutil.ParseSemanticVersion(daemon.Version)
		supportsGetByState := true
		if err != nil {
			supportsGetByState = false
		}
		if daemonVer.LessThan(keaVerWithState) {
			supportsGetByState = false
			if daemonVer.GreaterThan(keaVerWhereBroken) {
				log.WithField("version", daemon.Version).
					Warn("Kea versions 3.1.1 through 3.1.4 do not support any method of querying for declined leases.  Please downgrade to the previous stable version or upgrade to 3.1.5 (or newer) if you rely on this feature.")
				erredDaemons = append(erredDaemons, daemon)
				continue
			}
		}

		if supportsGetByState {
			if daemon.Name == daemonname.DHCPv4 {
				commands = append(commands, "lease4-get-by-state")
			}
			if daemon.Name == daemonname.DHCPv6 {
				commands = append(commands, "lease6-get-by-state")
			}

			leasesByState, warns, err := getLeasesByProperties(agents, daemon, "declined", commands...)
			daemonError = warns
			if err != nil {
				daemonError = true
				log.WithError(err).Warnf("Failed to fetch leases from Kea [%d] %s daemon", daemon.ID, daemon.Name)
			} else {
				leases = slices.Concat(leases, leasesByState)
			}
		} else {
			if daemon.Name == daemonname.DHCPv4 {
				commands = append(commands, "lease4-get-by-hw-address")
			}
			if daemon.Name == daemonname.DHCPv6 {
				commands = append(commands, "lease6-get-by-duid")
			}

			// Send these commands with empty hw-address and empty duid.
			leasesByProperties, warns, err := getLeasesByProperties(agents, daemon, "", commands...)
			daemonError = warns
			if err != nil {
				daemonError = true
				log.WithError(err).Warnf("Failed to fetch leases from Kea [%d] %s daemon", daemon.ID, daemon.Name)
			} else {
				for j := range leasesByProperties {
					// Only return the leases in the declined state.
					if leasesByProperties[j].State == keadata.LeaseStateDeclined {
						leases = append(leases, leasesByProperties[j])
					}
				}
			}
		}
		if daemonError {
			erredDaemons = append(erredDaemons, daemon)
		}
	}
	return leases, erredDaemons, nil
}

// Selects leases not matching specified host reservation. It compares DHCP
// identifiers in the lease with host identifiers. If match is not found,
// the lease is considered in conflict with the host and returned in the
// conflicts slice. This mechanism has a limitation that lease conflicts
// can't be detected if the host contains flex-id or circuit-id identifiers.
// Lease information does not contain any indication if the lease has been
// assigned using any of these identifiers.
func findHostLeaseConflicts(host *dbmodel.Host, leases []dbmodel.Lease) (conflicts []int64) {
	if host.HasIdentifierType("circuit-id") || host.HasIdentifierType("flex-id") {
		return conflicts
	}

	// Detect conflicting leases.
	for i := range leases {
		lease := leases[i]
		ids := []struct {
			leaseID    string
			hostIDType string
		}{
			{
				leaseID:    lease.ClientID,
				hostIDType: "client-id",
			},
			{
				leaseID:    lease.DUID,
				hostIDType: "duid",
			},
			{
				leaseID:    lease.HWAddress,
				hostIDType: "hw-address",
			},
		}
		conflict := true
		for _, id := range ids {
			// If the identifier is present in the lease, let's check
			// if it matches any in the host reservation.
			if len(id.leaseID) > 0 {
				if _, equal := host.HasIdentifier(id.hostIDType, storkutil.HexToBytes(id.leaseID)); equal {
					conflict = false
					break
				}
			}
		}
		if conflict {
			conflicts = append(conflicts, lease.ID)
		}
	}
	return conflicts
}

// Attempts to find leases for a given host reservation. An error is returned
// only if there is a problem with database communication. If the host doesn't
// exist, no leases are returned. This function will send commands to all
// monitored Kea servers querying for leases assigned to the given host.
// If there is a communication problem with any of the Kea servers, the details
// of the server are recorded in the erredDaemons slice.
func FindLeasesByHostID(db *dbops.PgDB, agents agentcomm.ConnectedAgents, hostID int64) (leases []dbmodel.Lease, conflicts []int64, erredDaemons []*dbmodel.Daemon, err error) {
	host, err := dbmodel.GetHost(db, hostID)
	if err != nil {
		err = errors.WithMessagef(err, "failed to fetch host with ID %d while searching for its leases", hostID)
		return leases, conflicts, erredDaemons, err
	}
	if host == nil {
		return leases, conflicts, erredDaemons, err
	}

	// Get Kea daemons from the database. We will send commands to these
	// daemons to find leases.
	daemons, err := dbmodel.GetDHCPDaemons(db)
	if err != nil {
		err = errors.WithMessagef(err, "failed to fetch Kea daemons while searching for leases for host ID %d", hostID)
		return leases, conflicts, erredDaemons, err
	}

	currentLeaseID := int64(1)
	for i := range daemons {
		daemon := &daemons[i]
		if !hasLeaseCmdsHook(daemon) {
			// This daemon does not have the libdhcp_lease_cmds hook library configured
			// so it cannot return leases.
			continue
		}

		// Monitor if a daemon returned an error. We stop sending commands to the
		// daemon it first returns an error.
		dhcp4Error := false
		dhcp6Error := false
		daemonError := false
		// Go over all IP reservations and send appropriate commands to the daemon
		// for each of them.
		for _, address := range host.GetIPReservations() {
			parsedIP := storkutil.ParseIP(address)
			if parsedIP == nil {
				// This is rather impossible condition, but let's be safe.
				continue
			}
			// Determine if this is IPv4 or IPv6 lease.
			switch parsedIP.Protocol {
			case storkutil.IPv4:
				if daemon.Name != daemonname.DHCPv4 {
					// This is not a DHCPv4 daemon, so skip it.
					continue
				}

				if !dhcp4Error {
					lease, err := GetLease4ByIPAddress(agents, daemon, parsedIP.NetworkPrefix)
					if err != nil {
						dhcp4Error = true
						log.WithError(err).Warn("failed to get lease by IPv4 address from Kea")
					} else if lease != nil {
						lease.ID = currentLeaseID
						currentLeaseID++
						leases = append(leases, *lease)
					}
				}
			case storkutil.IPv6:
				if daemon.Name != daemonname.DHCPv6 {
					// This is not a DHCPv6 daemon, so skip it.
					continue
				}

				if !dhcp6Error {
					// These commands distinguish between IA_NA and IA_PD. A caller
					// must specify the lease type.
					leaseType := keactrl.LeaseTypeNA
					if parsedIP.Prefix {
						leaseType = keactrl.LeaseTypePD
					}
					lease, err := GetLease6ByIPAddress(agents, daemon, leaseType, parsedIP.NetworkPrefix)
					if err != nil {
						dhcp6Error = true
						log.WithError(err).Warn("failed to get lease by IPv6 address from Kea")
					} else if lease != nil {
						lease.ID = currentLeaseID
						currentLeaseID++
						leases = append(leases, *lease)
					}
				}
			default:
				// Again, this is impossible condition.
				continue
			}
			// The daemon returned an error. Maybe the server is unavailable. We don't
			// want to send more commands to a daemon returning an error because there
			// is a minimal chance it will reply with success.
			if dhcp4Error || dhcp6Error {
				if !daemonError {
					// Record a daemon for which the error was returned.
					erredDaemons = append(erredDaemons, daemon)
					daemonError = true
				}
				// If both daemons returned an error, stop sending any commands to
				// this daemon.
				if dhcp4Error && dhcp6Error {
					break
				}
			}
		}
	}

	// Detect conflicting leases, i.e. leases assigned to clients having different
	// DHCP identifiers than those in our host reservations.
	conflicts = findHostLeaseConflicts(host, leases)

	return leases, conflicts, erredDaemons, err
}
