package restservice

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/go-openapi/runtime/middleware"
	log "github.com/sirupsen/logrus"

	"isc.org/stork/server/config"
	dbmodel "isc.org/stork/server/database/model"
	"isc.org/stork/server/gen/models"
	dhcp "isc.org/stork/server/gen/restapi/operations/d_h_c_p"
	storkutil "isc.org/stork/util"
)

// Converts host reservation fetched from the database to the format
// used in REST API.
func convertFromHost(dbHost *dbmodel.Host) *models.Host {
	host := &models.Host{
		ID:       dbHost.ID,
		SubnetID: dbHost.SubnetID,
		Hostname: dbHost.Hostname,
	}
	// Include subnet prefix if this is subnet specific host.
	if dbHost.Subnet != nil {
		host.SubnetPrefix = dbHost.Subnet.Prefix
	}
	// Convert DHCP host identifiers.
	for _, dbHostID := range dbHost.HostIdentifiers {
		hostID := models.HostIdentifier{
			IDType:     dbHostID.Type,
			IDHexValue: dbHostID.ToHex(":"),
		}
		host.HostIdentifiers = append(host.HostIdentifiers, &hostID)
	}
	// Convert IP reservations.
	for _, dbHostIP := range dbHost.IPReservations {
		parsedIP := storkutil.ParseIP(dbHostIP.Address)
		if parsedIP == nil {
			continue
		}
		hostIP := models.IPReservation{
			Address: parsedIP.NetworkAddress,
		}
		if parsedIP.Prefix {
			host.PrefixReservations = append(host.PrefixReservations, &hostIP)
		} else {
			host.AddressReservations = append(host.AddressReservations, &hostIP)
		}
	}
	// Append local hosts containing associations of the host with
	// daemons.
	for _, dbLocalHost := range dbHost.LocalHosts {
		localHost := models.LocalHost{
			AppID:      dbLocalHost.Daemon.AppID,
			AppName:    dbLocalHost.Daemon.App.Name,
			DataSource: dbLocalHost.DataSource,
		}
		host.LocalHosts = append(host.LocalHosts, &localHost)
	}
	return host
}

// Convert host reservation from the format used in REST API to a
// database host representation.
func convertToHost(restHost *models.Host) (*dbmodel.Host, error) {
	var err error
	host := &dbmodel.Host{
		ID:       restHost.ID,
		SubnetID: restHost.SubnetID,
		Hostname: restHost.Hostname,
	}
	// Convert DHCP host identifiers.
	for _, hid := range restHost.HostIdentifiers {
		hostID := dbmodel.HostIdentifier{
			Type:  hid.IDType,
			Value: storkutil.HexToBytes(hid.IDHexValue),
		}
		host.HostIdentifiers = append(host.HostIdentifiers, hostID)
	}
	// Convert IP reservations.
	for _, r := range append(restHost.PrefixReservations, restHost.AddressReservations...) {
		ipr := dbmodel.IPReservation{
			Address: r.Address,
		}
		host.IPReservations = append(host.IPReservations, ipr)
	}
	// Convert local hosts containing associations of the host with daemons.
	for _, lh := range restHost.LocalHosts {
		localHost := dbmodel.LocalHost{
			DaemonID:   lh.DaemonID,
			DataSource: lh.DataSource,
		}
		localHost.DHCPOptionSet, err = flattenDHCPOptions("", lh.Options)
		if err != nil {
			return nil, err
		}
		host.LocalHosts = append(host.LocalHosts, localHost)
	}
	return host, nil
}

// Fetches host reservations from the database and converts to the data formats
// used in REST API.
func (r *RestAPI) getHosts(offset, limit, appID int64, subnetID *int64, filterText *string, global *bool, sortField string, sortDir dbmodel.SortDirEnum) (*models.Hosts, error) {
	// Get the hosts from the database.
	dbHosts, total, err := dbmodel.GetHostsByPage(r.DB, offset, limit, appID, subnetID, filterText, global, sortField, sortDir)
	if err != nil {
		return nil, err
	}

	hosts := &models.Hosts{
		Total: total,
	}

	// Convert hosts fetched from the database to REST.
	for i := range dbHosts {
		host := convertFromHost(&dbHosts[i])
		hosts.Items = append(hosts.Items, host)
	}

	return hosts, nil
}

// Get list of hosts with specifying an offset and a limit. The hosts can be fetched
// for a given subnet and with filtering by search text.
func (r *RestAPI) GetHosts(ctx context.Context, params dhcp.GetHostsParams) middleware.Responder {
	var start int64 = 0
	if params.Start != nil {
		start = *params.Start
	}

	var limit int64 = 10
	if params.Limit != nil {
		limit = *params.Limit
	}

	var appID int64 = 0
	if params.AppID != nil {
		appID = *params.AppID
	}

	// get hosts from db
	hosts, err := r.getHosts(start, limit, appID, params.SubnetID, params.Text, params.Global, "", dbmodel.SortDirAny)
	if err != nil {
		msg := "Problem fetching hosts from the database"
		log.Error(err)
		rsp := dhcp.NewGetHostsDefault(http.StatusInternalServerError).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}

	// Everything fine.
	rsp := dhcp.NewGetHostsOK().WithPayload(hosts)
	return rsp
}

// Get a host by ID.
func (r *RestAPI) GetHost(ctx context.Context, params dhcp.GetHostParams) middleware.Responder {
	// Find a host in the database.
	dbHost, err := dbmodel.GetHost(r.DB, params.ID)
	if err != nil {
		// Error while communicating with the database.
		msg := fmt.Sprintf("Problem fetching host reservation with ID %d from db", params.ID)
		log.Error(err)
		rsp := dhcp.NewGetHostDefault(http.StatusInternalServerError).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}
	if dbHost == nil {
		// Host not found.
		msg := fmt.Sprintf("Cannot find host reservation with ID %d", params.ID)
		rsp := dhcp.NewGetHostDefault(http.StatusNotFound).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}
	// Host found. Convert it to the format used in REST API.
	host := convertFromHost(dbHost)
	rsp := dhcp.NewGetHostOK().WithPayload(host)
	return rsp
}

// Implements the POST call to create new transaction for adding a new host
// reservation (hosts/new/transaction/new).
func (r *RestAPI) CreateHostBegin(ctx context.Context, params dhcp.CreateHostBeginParams) middleware.Responder {
	// A list of Kea DHCP daemons will be needed in the user form,
	// so the user can select which servers send the reservation to.
	daemons, err := dbmodel.GetKeaDHCPDaemons(r.DB)
	if err != nil {
		msg := "problem with fetching Kea daemons from the database"
		log.Error(err)
		rsp := dhcp.NewCreateHostBeginDefault(http.StatusInternalServerError).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}
	// Convert daemons list to REST API format.
	respDaemons := []*models.KeaDaemon{}
	for i := range daemons {
		if daemons[i].KeaDaemon != nil && daemons[i].KeaDaemon.Config != nil {
			// Filter the daemons with host_cmds hook library.
			if _, _, exists := daemons[i].KeaDaemon.Config.GetHooksLibrary("libdhcp_host_cmds"); exists {
				respDaemons = append(respDaemons, keaDaemonToRestAPI(&daemons[i]))
			}
		}
	}
	// If there are no daemons with host_cmds hooks library loaded there is no way
	// to add new host reservation. In that case, we don't begin a transaction.
	if len(respDaemons) == 0 {
		msg := "unable to begin transaction for adding new host because there are no Kea servers with host_cmds hooks library available"
		log.Error(msg)
		rsp := dhcp.NewCreateHostBeginDefault(http.StatusBadRequest).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}
	// Host reservations are typically associated with subnets. The
	// user needs a current list of available subnets.
	subnets, err := dbmodel.GetAllSubnets(r.DB, 0)
	if err != nil {
		msg := "problem with fetching subnets from the database"
		log.Error(err)
		rsp := dhcp.NewCreateHostBeginDefault(http.StatusInternalServerError).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}
	// Convert subnets list to REST API format.
	respSubnets := []*models.Subnet{}
	for i := range subnets {
		respSubnets = append(respSubnets, subnetToRestAPI(&subnets[i]))
	}
	// Get the logged user's ID.
	ok, user := r.SessionManager.Logged(ctx)
	if !ok {
		msg := "unable to begin transaction because user is not logged in"
		log.Error("problem with creating transaction context because user has no session")
		rsp := dhcp.NewCreateHostBeginDefault(http.StatusForbidden).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}
	// Create configuration context.
	cctx, err := r.ConfigManager.CreateContext(int64(user.ID))
	if err != nil {
		msg := "problem with creating transaction context"
		log.Error(err)
		rsp := dhcp.NewCreateHostBeginDefault(http.StatusInternalServerError).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}
	// Retrieve the generated context ID.
	cctxID, ok := config.GetValueAsInt64(cctx, config.ContextIDKey)
	if !ok {
		msg := "problem with retrieving context ID for a transaction"
		log.Error(msg)
		rsp := dhcp.NewCreateHostBeginDefault(http.StatusInternalServerError).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}
	// Remember the context, i.e. new transaction has been succcessfully created.
	_ = r.ConfigManager.RememberContext(cctx, time.Minute*10)

	// Return transaction ID, apps and subnets to the user.
	contents := &models.CreateHostBeginResponse{
		ID:      cctxID,
		Daemons: respDaemons,
		Subnets: respSubnets,
	}
	rsp := dhcp.NewCreateHostBeginOK().WithPayload(contents)
	return rsp
}

// Implements the POST call to apply and commit host reservation (hosts/new/transaction/{id}/submit).
func (r *RestAPI) CreateHostSubmit(ctx context.Context, params dhcp.CreateHostSubmitParams) middleware.Responder {
	// Make sure that the host information is present.
	if params.Host == nil {
		msg := "host information not specified"
		log.Errorf(msg)
		rsp := dhcp.NewCreateHostSubmitDefault(http.StatusBadRequest).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}
	// Get the user ID and recover the transaction context.
	ok, user := r.SessionManager.Logged(ctx)
	if !ok {
		msg := "unable to submit because user is not logged in"
		log.Error("problem with recovering transaction context because user has no session")
		rsp := dhcp.NewCreateHostSubmitDefault(http.StatusForbidden).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}
	// Retrieve the context from the config manager.
	cctx, _ := r.ConfigManager.RecoverContext(params.ID, int64(user.ID))
	if cctx == nil {
		msg := "transaction expired"
		log.Errorf("problem with recovering transaction context for transaction ID %d and user ID %d", params.ID, user.ID)
		rsp := dhcp.NewCreateHostSubmitDefault(http.StatusNotFound).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}
	// Convert host information from REST API to database format.
	host, err := convertToHost(params.Host)
	if err != nil {
		msg := "error parsing specified host reservation"
		log.Error(err)
		rsp := dhcp.NewCreateHostSubmitDefault(http.StatusBadRequest).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}
	err = host.PopulateDaemons(r.DB)
	if err != nil {
		msg := "specified host is associated with daemons that no longer exist"
		log.Error(err)
		rsp := dhcp.NewCreateHostSubmitDefault(http.StatusNotFound).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}
	err = host.PopulateSubnet(r.DB)
	if err != nil {
		msg := "problem with retrieving subnet association with the host"
		log.Error(err)
		rsp := dhcp.NewCreateHostSubmitDefault(http.StatusInternalServerError).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}
	// Apply the host information (create Kea commands).
	cctx, err = r.ConfigManager.GetKeaModule().ApplyHostAdd(cctx, host)
	if err != nil {
		msg := "problem with applying host information"
		log.Error(err)
		rsp := dhcp.NewCreateHostSubmitDefault(http.StatusInternalServerError).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}
	// Send the commands to Kea servers.
	cctx, err = r.ConfigManager.Commit(cctx)
	if err != nil {
		msg := fmt.Sprintf("problem with committing host information: %s", err)
		log.Error(err)
		rsp := dhcp.NewCreateHostSubmitDefault(http.StatusConflict).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}
	// Everything ok. Cleanup and send OK to the client.
	r.ConfigManager.Done(cctx)
	rsp := dhcp.NewCreateHostSubmitOK()
	return rsp
}

// Implements the DELETE call to cancel adding new reservation (hosts/new/transaction{id}). It
// removes the specified transaction from the config manager, if the transaction exists.
func (r *RestAPI) CreateHostDelete(ctx context.Context, params dhcp.CreateHostDeleteParams) middleware.Responder {
	// Get the user ID and recover the transaction context.
	ok, user := r.SessionManager.Logged(ctx)
	if !ok {
		msg := "unable to cancel transaction because user is not logged in"
		log.Error("problem with recovering transaction context because user has no session")
		rsp := dhcp.NewCreateHostDeleteDefault(http.StatusForbidden).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}
	// Retrieve the context from the config manager.
	cctx, _ := r.ConfigManager.RecoverContext(params.ID, int64(user.ID))
	if cctx == nil {
		msg := "transaction expired"
		log.Errorf("problem with recovering transaction context for transaction ID %d and user ID %d", params.ID, user.ID)
		rsp := dhcp.NewCreateHostDeleteDefault(http.StatusNotFound).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}
	r.ConfigManager.Done(cctx)
	rsp := dhcp.NewCreateHostDeleteOK()
	return rsp
}
