package kea

import (
	"context"

	errors "github.com/pkg/errors"

	keactrl "isc.org/stork/appctrl/kea"
	keadata "isc.org/stork/appdata/kea"
	"isc.org/stork/server/agentcomm"
	dbmodel "isc.org/stork/server/database/model"
)

// Structure representing arguments of a response to a command
// fetching multiple DHCPv4 leases from the Kea server.
type Lease4GetResponseArgs struct {
	Leases []keadata.Lease4
}

// Structure representing a response to a command fetching multiple
// DHCPv4 leases from the Kea server.
type Lease4GetResponse struct {
	keactrl.ResponseHeader
	Arguments *Lease4GetResponseArgs `json:"arguments,omitempty"`
}

// Structure representing arguments of a response to a command
// fetching multiple DHCPv6 leases from the Kea server.
type Lease6GetResponseArgs struct {
	Leases []keadata.Lease6
}

// Structure representing a response to a command fetching multiple
// DHCPv6 leases from the Kea server.
type Lease6GetResponse struct {
	keactrl.ResponseHeader
	Arguments *Lease6GetResponseArgs `json:"arguments,omitempty"`
}

// Validates a response from a Kea daemon to the commands fetching
// multiple leases, e.g. lease4-get-by-hw-address. It checks that
// the response comprises the Success status and that arguments
// map is not nil.
func validateGetLeasesResponse(commandName string, result int, arguments interface{}) error {
	if result == keactrl.ResponseError {
		return errors.Errorf("error returned by Kea in response to %s command", commandName)
	}
	if result == keactrl.ResponseCommandUnsupported {
		return errors.Errorf("%s command unsupported", commandName)
	}
	if arguments == nil {
		return errors.Errorf("response to %s command lacks arguments", commandName)
	}
	return nil
}

// Sends a specified command to the DHCPv4 server to fetch leases by one of the following
// properties: hw-address, client-id or hostname. The agents argument contains a pointer
// to the agents monitored by Stork. The dbApp is a pointer to the app identifying to
// which agent the server sends this command. The propertyName is one of the following
// hw-address, client-id or hostname.
func getLeases4ByProperty(agents agentcomm.ConnectedAgents, dbApp *dbmodel.App, commandName, propertyName, propertyValue string) (leases []keadata.Lease4, err error) {
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
	response := make([]Lease4GetResponse, 1)
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
	if err = validateGetLeasesResponse(commandName, response[0].Result, response[0].Arguments); err != nil {
		return leases, err
	}
	leases = response[0].Arguments.Leases
	return leases, nil
}

// Sends a specified command to the DHCPv6 server to fetch leases by one of the following
// properties: duid or hostname. The agents argument contains a pointer to the agents
// monitored by Stork. The dbApp is a pointer to the app identifying to which agent the
// server sends this command. The propertyName is one of the following duid or hostname.
func getLeases6ByProperty(agents agentcomm.ConnectedAgents, dbApp *dbmodel.App, commandName, propertyName, propertyValue string) (leases []keadata.Lease6, err error) {
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
	response := make([]Lease6GetResponse, 1)
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
	if err = validateGetLeasesResponse(commandName, response[0].Result, response[0].Arguments); err != nil {
		return leases, err
	}
	leases = response[0].Arguments.Leases
	return leases, nil
}

// Sends lease4-get-by-hw-address command to Kea.
func GetLeases4ByHWAddress(agents agentcomm.ConnectedAgents, dbApp *dbmodel.App, hwaddress string) (leases []keadata.Lease4, err error) {
	return getLeases4ByProperty(agents, dbApp, "lease4-get-by-hw-address", "hw-address", hwaddress)
}

// Sends lease4-get-by-client-id command to Kea.
func GetLeases4ByClientID(agents agentcomm.ConnectedAgents, dbApp *dbmodel.App, clientID string) (leases []keadata.Lease4, err error) {
	return getLeases4ByProperty(agents, dbApp, "lease4-get-by-client-id", "client-id", clientID)
}

// Sends lease4-get-by-hostname command to Kea.
func GetLeases4ByHostname(agents agentcomm.ConnectedAgents, dbApp *dbmodel.App, hostname string) (leases []keadata.Lease4, err error) {
	return getLeases4ByProperty(agents, dbApp, "lease4-get-by-hostname", "hostname", hostname)
}

// Sends lease6-get-by-duid command to Kea.
func GetLeases6ByDUID(agents agentcomm.ConnectedAgents, dbApp *dbmodel.App, duid string) (leases []keadata.Lease6, err error) {
	return getLeases6ByProperty(agents, dbApp, "lease6-get-by-duid", "duid", duid)
}

// Sends lease6-get-by-hostname command to Kea.
func GetLeases6ByHostname(agents agentcomm.ConnectedAgents, dbApp *dbmodel.App, hostname string) (leases []keadata.Lease6, err error) {
	return getLeases6ByProperty(agents, dbApp, "lease6-get-by-hostname", "hostname", hostname)
}
