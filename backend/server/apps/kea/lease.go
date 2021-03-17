package kea

import (
	"context"
	"fmt"
	"net"
	"reflect"

	errors "github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	keactrl "isc.org/stork/appctrl/kea"
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
// fetching multiple DHCPv4 leases from the Kea server.
type Lease4GetMultipleResponseArgs struct {
	Leases []dbmodel.Lease
}

// Structure representing a response to a command fetching multiple
// DHCPv4 leases from the Kea server.
type Lease4GetMultipleResponse struct {
	keactrl.ResponseHeader
	Arguments *Lease4GetMultipleResponseArgs `json:"arguments,omitempty"`
}

// Structure representing arguments of a response to a command
// fetching multiple DHCPv6 leases from the Kea server.
type Lease6GetMultipleResponseArgs struct {
	Leases []dbmodel.Lease
}

// Structure representing a response to a command fetching multiple
// DHCPv6 leases from the Kea server.
type Lease6GetMultipleResponse struct {
	keactrl.ResponseHeader
	Arguments *Lease6GetMultipleResponseArgs `json:"arguments,omitempty"`
}

// Validates a response from a Kea daemon to the commands fetching
// leases, e.g. lease4-get-by-hw-address. It checks that the response
// comprises the Success status and that arguments map is not nil.
func validateGetLeasesResponse(commandName string, result int, arguments interface{}) error {
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
func GetLease4ByIPAddress(agents agentcomm.ConnectedAgents, dbApp *dbmodel.App, ipaddress string) (lease *dbmodel.Lease, err error) {
	daemons, err := keactrl.NewDaemons("dhcp4")
	if err != nil {
		return lease, err
	}
	arguments := map[string]interface{}{
		"ip-address": ipaddress,
	}
	command, err := keactrl.NewCommand("lease4-get", daemons, &arguments)
	if err != nil {
		return lease, err
	}
	response := make([]Lease4GetResponse, 1)
	ctx := context.Background()
	respResult, err := agents.ForwardToKeaOverHTTP(ctx, dbApp, []*keactrl.Command{command}, &response)
	if err != nil {
		return lease, err
	}
	if respResult.Error != nil {
		return lease, respResult.Error
	}
	if len(response) == 0 {
		return lease, errors.Errorf("invalid response to lease4-get command received")
	}
	if response[0].Result == keactrl.ResponseEmpty {
		return lease, nil
	}
	if err = validateGetLeasesResponse("lease4-get", response[0].Result, response[0].Arguments); err != nil {
		return lease, err
	}
	lease = response[0].Arguments
	lease.AppID = dbApp.ID
	lease.App = dbApp
	return lease, nil
}

// Sends a lease6-get command with type and ip-address arguments specifying
// searched lease type and IP address. If the lease is found, the pointer to
// it is returned. If the lease does not exist, a nil pointer and nil error
// are returned.
func GetLease6ByIPAddress(agents agentcomm.ConnectedAgents, dbApp *dbmodel.App, leaseType, ipaddress string) (lease *dbmodel.Lease, err error) {
	daemons, err := keactrl.NewDaemons("dhcp6")
	if err != nil {
		return lease, err
	}
	arguments := map[string]interface{}{
		"ip-address": ipaddress,
		"type":       leaseType,
	}
	command, err := keactrl.NewCommand("lease6-get", daemons, &arguments)
	if err != nil {
		return lease, err
	}
	response := make([]Lease6GetResponse, 1)
	ctx := context.Background()
	respResult, err := agents.ForwardToKeaOverHTTP(ctx, dbApp, []*keactrl.Command{command}, &response)
	if err != nil {
		return lease, err
	}
	if respResult.Error != nil {
		return lease, respResult.Error
	}
	if len(response) == 0 {
		return lease, errors.Errorf("invalid response to lease6-get command received")
	}
	if response[0].Result == keactrl.ResponseEmpty {
		return lease, nil
	}
	if err = validateGetLeasesResponse("lease6-get", response[0].Result, response[0].Arguments); err != nil {
		return lease, err
	}
	lease = response[0].Arguments
	lease.AppID = dbApp.ID
	lease.App = dbApp
	return lease, nil
}

// Sends a specified command to the DHCPv4 server to fetch leases by one of the following
// properties: hw-address, client-id or hostname. The agents argument contains a pointer
// to the agents monitored by Stork. The dbApp is a pointer to the app identifying to
// which agent the server sends this command. The propertyName is one of the following
// hw-address, client-id or hostname.
func getLeases4ByProperty(agents agentcomm.ConnectedAgents, dbApp *dbmodel.App, commandName, propertyName, propertyValue string) (leases []dbmodel.Lease, err error) {
	daemons, err := keactrl.NewDaemons("dhcp4")
	if err != nil {
		return leases, err
	}
	arguments := map[string]interface{}{
		propertyName: propertyValue,
	}
	command, err := keactrl.NewCommand(commandName, daemons, &arguments)
	if err != nil {
		return leases, err
	}
	response := make([]Lease4GetMultipleResponse, 1)
	ctx := context.Background()
	respResult, err := agents.ForwardToKeaOverHTTP(ctx, dbApp, []*keactrl.Command{command}, &response)
	if err != nil {
		return leases, err
	}
	if respResult.Error != nil {
		return leases, respResult.Error
	}
	if len(response) == 0 {
		return leases, errors.Errorf("invalid response to %s command received", commandName)
	}
	if response[0].Result == keactrl.ResponseEmpty {
		return leases, nil
	}
	if err = validateGetLeasesResponse(commandName, response[0].Result, response[0].Arguments); err != nil {
		return leases, err
	}
	leases = response[0].Arguments.Leases
	for i := range leases {
		leases[i].AppID = dbApp.ID
		leases[i].App = dbApp
	}
	return leases, nil
}

// Sends a specified command to the DHCPv6 server to fetch leases by one of the following
// properties: duid or hostname. The agents argument contains a pointer to the agents
// monitored by Stork. The dbApp is a pointer to the app identifying to which agent the
// server sends this command. The propertyName is one of the following duid or hostname.
func getLeases6ByProperty(agents agentcomm.ConnectedAgents, dbApp *dbmodel.App, commandName, propertyName, propertyValue string) (leases []dbmodel.Lease, err error) {
	daemons, err := keactrl.NewDaemons("dhcp6")
	if err != nil {
		return leases, err
	}
	arguments := map[string]interface{}{
		propertyName: propertyValue,
	}
	command, err := keactrl.NewCommand(commandName, daemons, &arguments)
	if err != nil {
		return leases, err
	}
	response := make([]Lease6GetMultipleResponse, 1)
	ctx := context.Background()
	respResult, err := agents.ForwardToKeaOverHTTP(ctx, dbApp, []*keactrl.Command{command}, &response)
	if err != nil {
		return leases, err
	}
	if respResult.Error != nil {
		return leases, respResult.Error
	}
	if len(response) == 0 {
		return leases, errors.Errorf("invalid response to %s command received", commandName)
	}
	if response[0].Result == keactrl.ResponseEmpty {
		return leases, nil
	}
	if err = validateGetLeasesResponse(commandName, response[0].Result, response[0].Arguments); err != nil {
		return leases, err
	}
	leases = response[0].Arguments.Leases
	for i := range leases {
		leases[i].AppID = dbApp.ID
		leases[i].App = dbApp
	}
	return leases, nil
}

// Sends lease4-get-by-hw-address and lease4-get-by-client-id to Kea combined
// in a single gRPC command. An error response returned to first command
// yelds a function error to avoid sending second command that is unlikely to
// succeed. If the first command succeeds but the second one fails, an error
// is returned and empty leases slice.
func GetLeases4ByIdentifier(agents agentcomm.ConnectedAgents, dbApp *dbmodel.App, identifier string) (leases []dbmodel.Lease, err error) {
	daemons, err := keactrl.NewDaemons("dhcp4")
	if err != nil {
		return leases, err
	}
	// Prepare lease4-get-by-hw-address and lease4-by-client-id commands.
	var commands []*keactrl.Command
	for _, arg := range []string{"hw-address", "client-id"} {
		arguments := map[string]interface{}{
			arg: identifier,
		}
		command, err := keactrl.NewCommand(fmt.Sprintf("lease4-get-by-%s", arg), daemons, &arguments)
		if err != nil {
			return leases, err
		}
		commands = append(commands, command)
	}
	// Prepare containers for responses.
	responseHWAddress := make([]Lease4GetMultipleResponse, 1)
	responseClientID := make([]Lease4GetMultipleResponse, 1)
	ctx := context.Background()
	respResult, err := agents.ForwardToKeaOverHTTP(ctx, dbApp, commands, &responseHWAddress, &responseClientID)
	if err != nil {
		return leases, err
	}
	if respResult.Error != nil {
		return leases, respResult.Error
	}
	// Validate responses to both commands.
	for i, response := range [][]Lease4GetMultipleResponse{responseHWAddress, responseClientID} {
		if len(response) == 0 {
			return []dbmodel.Lease{}, errors.Errorf("invalid response received from Kea to %s command", commands[i].Command)
		}
		if response[0].Result != keactrl.ResponseEmpty {
			if err = validateGetLeasesResponse(commands[i].Command, response[0].Result, response[0].Arguments); err != nil {
				return []dbmodel.Lease{}, err
			}
			leases = append(leases, response[0].Arguments.Leases...)
		}
	}
	for i := range leases {
		leases[i].AppID = dbApp.ID
		leases[i].App = dbApp
	}
	return leases, nil
}

// Sends lease4-get-by-hw-address command to Kea.
func GetLeases4ByHWAddress(agents agentcomm.ConnectedAgents, dbApp *dbmodel.App, hwaddress string) (leases []dbmodel.Lease, err error) {
	return getLeases4ByProperty(agents, dbApp, "lease4-get-by-hw-address", "hw-address", hwaddress)
}

// Sends lease4-get-by-client-id command to Kea.
func GetLeases4ByClientID(agents agentcomm.ConnectedAgents, dbApp *dbmodel.App, clientID string) (leases []dbmodel.Lease, err error) {
	return getLeases4ByProperty(agents, dbApp, "lease4-get-by-client-id", "client-id", clientID)
}

// Sends lease4-get-by-hostname command to Kea.
func GetLeases4ByHostname(agents agentcomm.ConnectedAgents, dbApp *dbmodel.App, hostname string) (leases []dbmodel.Lease, err error) {
	return getLeases4ByProperty(agents, dbApp, "lease4-get-by-hostname", "hostname", hostname)
}

// Sends lease6-get-by-duid command to Kea.
func GetLeases6ByDUID(agents agentcomm.ConnectedAgents, dbApp *dbmodel.App, duid string) (leases []dbmodel.Lease, err error) {
	return getLeases6ByProperty(agents, dbApp, "lease6-get-by-duid", "duid", duid)
}

// Sends lease6-get-by-hostname command to Kea.
func GetLeases6ByHostname(agents agentcomm.ConnectedAgents, dbApp *dbmodel.App, hostname string) (leases []dbmodel.Lease, err error) {
	return getLeases6ByProperty(agents, dbApp, "lease6-get-by-hostname", "hostname", hostname)
}

func hasLeaseCmdsHook(app *dbmodel.App, daemonName string) bool {
	daemon := app.GetDaemonByName(daemonName)
	if daemon != nil && daemon.KeaDaemon != nil && daemon.KeaDaemon.Config != nil {
		if _, _, ok := daemon.KeaDaemon.Config.GetHooksLibrary("libdhcp_lease_cmds"); ok {
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
// it returns all that lease instances. The  Kea servers which
// returned an error response are returned as a second parameter.
// Such failures do not preclude the function from returning
// leases found on other servers, but the caller becomes aware
// that some leases may not be included due to the communication
// errors with some servers. The third returned parameter
// indicates a general error, e.g. issues with Stork database
// communication.
func FindLeases(db *dbops.PgDB, agents agentcomm.ConnectedAgents, text string) (leases []dbmodel.Lease, erredApps []*dbmodel.App, err error) {
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

	// Get Kea apps from the database. We will send commands to these
	// apps to find leases.
	apps, err := dbmodel.GetAppsByType(db, dbmodel.AppTypeKea)
	if err != nil {
		err = errors.WithMessagef(err, "failed to fetch Kea apps while searching for leases by %s", text)
		return leases, erredApps, err
	}

	for i := range apps {
		appError := false
		// Send DHCPv4 specific queries if the lease_cmds hook is installed.
		if queryType != ipv6 && hasLeaseCmdsHook(&apps[i], dbmodel.DaemonNameDHCPv4) {
			switch queryType {
			case ipv4:
				// This is an IPv4 address, so send the command to the DHCPv4 server.
				lease, err := GetLease4ByIPAddress(agents, &apps[i], text)
				if err != nil {
					appError = true
					log.Warn(err)
				}
				if lease != nil {
					leases = append(leases, *lease)
				}
			case identifier:
				// It is an identifier, so let's query the server using known identifier types.
				leases4ByID, err := GetLeases4ByIdentifier(agents, &apps[i], text)
				if err != nil {
					appError = true
					log.Warn(err)
				}
				leases = append(leases, leases4ByID...)
			default:
				// It is neither an IP address nor an identifier. Let's try to query by hostname.
				leases4ByHostname, err := GetLeases4ByHostname(agents, &apps[i], text)
				if err != nil {
					appError = true
					log.Warn(err)
				}
				leases = append(leases, leases4ByHostname...)
			}
		}
		// Send DHCPv6 specific queries if the lease_cmds hook is installed.
		if queryType != ipv4 && hasLeaseCmdsHook(&apps[i], dbmodel.DaemonNameDHCPv6) {
			switch queryType {
			case ipv6:
				// This is an IPv6 address (or prefix), so send the command to the
				// DHCPv6 server instead.
				for _, leaseType := range []string{"IA_NA", "IA_PD"} {
					lease, err := GetLease6ByIPAddress(agents, &apps[i], leaseType, text)
					if err != nil {
						appError = true
						log.Warn(err)
					}
					if lease != nil {
						leases = append(leases, *lease)
						// If we found a lease by IP address there is no reason to
						// query by delegated prefix because the IP address/prefix
						// must be unique in the database.
						break
					}
				}
			case identifier:
				leases6ByID, err := GetLeases6ByDUID(agents, &apps[i], text)
				if err != nil {
					appError = true
					log.Warn(err)
				}
				leases = append(leases, leases6ByID...)
			default:
				leases6ByHostname, err := GetLeases6ByHostname(agents, &apps[i], text)
				if err != nil {
					appError = true
					log.Warn(err)
				}
				leases = append(leases, leases6ByHostname...)
			}
		}
		if appError {
			erredApps = append(erredApps, &apps[i])
		}
	}
	return leases, erredApps, nil
}
