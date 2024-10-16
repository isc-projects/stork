package restservice

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/go-openapi/runtime/middleware"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	keaconfig "isc.org/stork/appcfg/kea"
	"isc.org/stork/server/apps/kea"
	"isc.org/stork/server/config"
	dbmodel "isc.org/stork/server/database/model"
	"isc.org/stork/server/gen/models"
	dhcp "isc.org/stork/server/gen/restapi/operations/d_h_c_p"
	storkutil "isc.org/stork/util"
)

// Converts Kea specific configuration parameters for a shared network to a format used
// in the REST API. This function is called to convert the shared-network level parameters
// for a subnet and/or shared network.
func convertSharedNetworkParametersToRestAPI(keaParameters *keaconfig.SharedNetworkParameters) *models.KeaConfigSubnetDerivedParameters {
	parameters := &models.KeaConfigSubnetDerivedParameters{
		KeaConfigCacheParameters: models.KeaConfigCacheParameters{
			CacheThreshold: keaParameters.CacheThreshold,
			CacheMaxAge:    keaParameters.CacheMaxAge,
		},
		KeaConfigClientClassParameters: models.KeaConfigClientClassParameters{
			ClientClass:          storkutil.NullifyEmptyString(keaParameters.ClientClass),
			RequireClientClasses: keaParameters.RequireClientClasses,
		},
		KeaConfigDdnsParameters: models.KeaConfigDdnsParameters{
			DdnsGeneratedPrefix:        storkutil.NullifyEmptyString(keaParameters.DDNSGeneratedPrefix),
			DdnsOverrideClientUpdate:   keaParameters.DDNSOverrideClientUpdate,
			DdnsOverrideNoUpdate:       keaParameters.DDNSOverrideNoUpdate,
			DdnsQualifyingSuffix:       storkutil.NullifyEmptyString(keaParameters.DDNSQualifyingSuffix),
			DdnsReplaceClientName:      storkutil.NullifyEmptyString(keaParameters.DDNSReplaceClientName),
			DdnsSendUpdates:            keaParameters.DDNSSendUpdates,
			DdnsUpdateOnRenew:          keaParameters.DDNSUpdateOnRenew,
			DdnsUseConflictResolution:  keaParameters.DDNSUseConflictResolution,
			DdnsConflictResolutionMode: keaParameters.DDNSConflictResolutionMode,
			DdnsTTLPercent:             keaParameters.DDNSTTLPercent,
		},
		KeaConfigHostnameCharParameters: models.KeaConfigHostnameCharParameters{
			HostnameCharReplacement: storkutil.NullifyEmptyString(keaParameters.HostnameCharReplacement),
			HostnameCharSet:         storkutil.NullifyEmptyString(keaParameters.HostnameCharSet),
		},
		KeaConfigPreferredLifetimeParameters: models.KeaConfigPreferredLifetimeParameters{
			MaxPreferredLifetime: keaParameters.MaxPreferredLifetime,
			MinPreferredLifetime: keaParameters.MinPreferredLifetime,
			PreferredLifetime:    keaParameters.PreferredLifetime,
		},
		KeaConfigReservationParameters: models.KeaConfigReservationParameters{
			ReservationMode:       storkutil.NullifyEmptyString(keaParameters.ReservationMode),
			ReservationsGlobal:    keaParameters.ReservationsGlobal,
			ReservationsInSubnet:  keaParameters.ReservationsInSubnet,
			ReservationsOutOfPool: keaParameters.ReservationsOutOfPool,
		},
		KeaConfigTimerParameters: models.KeaConfigTimerParameters{
			CalculateTeeTimes: keaParameters.CalculateTeeTimes,
			RebindTimer:       keaParameters.RebindTimer,
			RenewTimer:        keaParameters.RenewTimer,
			T1Percent:         keaParameters.T1Percent,
			T2Percent:         keaParameters.T2Percent,
		},
		KeaConfigValidLifetimeParameters: models.KeaConfigValidLifetimeParameters{
			MaxValidLifetime: keaParameters.MaxValidLifetime,
			MinValidLifetime: keaParameters.MinValidLifetime,
			ValidLifetime:    keaParameters.ValidLifetime,
		},
		KeaConfigAssortedSubnetParameters: models.KeaConfigAssortedSubnetParameters{
			Allocator:         storkutil.NullifyEmptyString(keaParameters.Allocator),
			Authoritative:     keaParameters.Authoritative,
			BootFileName:      storkutil.NullifyEmptyString(keaParameters.BootFileName),
			Interface:         storkutil.NullifyEmptyString(keaParameters.Interface),
			InterfaceID:       storkutil.NullifyEmptyString(keaParameters.InterfaceID),
			MatchClientID:     keaParameters.MatchClientID,
			NextServer:        storkutil.NullifyEmptyString(keaParameters.NextServer),
			PdAllocator:       storkutil.NullifyEmptyString(keaParameters.PDAllocator),
			RapidCommit:       keaParameters.RapidCommit,
			ServerHostname:    storkutil.NullifyEmptyString(keaParameters.ServerHostname),
			StoreExtendedInfo: keaParameters.StoreExtendedInfo,
		},
	}
	if keaParameters.Relay != nil {
		parameters.Relay = &models.KeaConfigAssortedSubnetParametersRelay{
			IPAddresses: keaParameters.Relay.IPAddresses,
		}
	}
	return parameters
}

// Creates a REST API representation of a shared network from a database model.
func (r *RestAPI) convertSharedNetworkToRestAPI(sn *dbmodel.SharedNetwork) *models.SharedNetwork {
	subnets := []*models.Subnet{}
	// Exclude the subnets that are not attached to any app. This shouldn't
	// be the case but let's be safe.
	for i := range sn.Subnets {
		subnet := r.convertSubnetToRestAPI(&sn.Subnets[i])
		subnets = append(subnets, subnet)
	}
	// Create shared network.
	sharedNetwork := &models.SharedNetwork{
		ID:               sn.ID,
		Name:             sn.Name,
		Universe:         int64(sn.Family),
		Subnets:          subnets,
		AddrUtilization:  float64(sn.AddrUtilization) / 10,
		PdUtilization:    float64(sn.PdUtilization) / 10,
		Stats:            sn.Stats,
		StatsCollectedAt: convertToOptionalDatetime(sn.StatsCollectedAt),
	}

	for _, lsn := range sn.LocalSharedNetworks {
		localSharedNetwork := &models.LocalSharedNetwork{
			AppID:    lsn.Daemon.App.ID,
			DaemonID: lsn.Daemon.ID,
			AppName:  lsn.Daemon.App.Name,
		}
		keaParameters := lsn.KeaParameters
		if keaParameters != nil {
			if localSharedNetwork.KeaConfigSharedNetworkParameters == nil {
				localSharedNetwork.KeaConfigSharedNetworkParameters = &models.KeaConfigSharedNetworkParameters{}
			}
			localSharedNetwork.KeaConfigSharedNetworkParameters.SharedNetworkLevelParameters = convertSharedNetworkParametersToRestAPI(keaParameters)
			localSharedNetwork.KeaConfigSharedNetworkParameters.SharedNetworkLevelParameters.OptionsHash = lsn.DHCPOptionSet.Hash
			localSharedNetwork.KeaConfigSharedNetworkParameters.SharedNetworkLevelParameters.Options = r.unflattenDHCPOptions(lsn.DHCPOptionSet.Options, "", 0)
		}

		// Global configuration parameters.
		if lsn.Daemon != nil && lsn.Daemon.KeaDaemon != nil && lsn.Daemon.KeaDaemon.Config != nil &&
			(lsn.Daemon.KeaDaemon.Config.IsDHCPv4() || lsn.Daemon.KeaDaemon.Config.IsDHCPv6()) {
			cfg := lsn.Daemon.KeaDaemon.Config
			if localSharedNetwork.KeaConfigSharedNetworkParameters == nil {
				localSharedNetwork.KeaConfigSharedNetworkParameters = &models.KeaConfigSharedNetworkParameters{}
			}
			localSharedNetwork.KeaConfigSharedNetworkParameters.GlobalParameters = convertGlobalSubnetParametersToRestAPI(cfg)
			var convertedOptions []dbmodel.DHCPOption
			for _, option := range cfg.GetDHCPOptions() {
				convertedOption, err := dbmodel.NewDHCPOptionFromKea(option, storkutil.IPType(sn.Family), r.DHCPOptionDefinitionLookup)
				if err != nil {
					continue
				}
				convertedOptions = append(convertedOptions, *convertedOption)
			}
			localSharedNetwork.KeaConfigSharedNetworkParameters.GlobalParameters.OptionsHash = keaconfig.NewHasher().Hash(convertedOptions)
			localSharedNetwork.KeaConfigSharedNetworkParameters.GlobalParameters.Options = r.unflattenDHCPOptions(convertedOptions, "", 0)
		}
		sharedNetwork.LocalSharedNetworks = append(sharedNetwork.LocalSharedNetworks, localSharedNetwork)
	}

	return sharedNetwork
}

// Convert shared network from the format used in REST API to a database shared network
// representation. It is used when Stork user modifies or creates new shared network.
// Thus, it doesn't populate shared network statistics as it is not specified by Stork user.
// It is pulled from the Kea servers periodically.
func (r *RestAPI) convertSharedNetworkFromRestAPI(restSharedNetwork *models.SharedNetwork) (*dbmodel.SharedNetwork, error) {
	subnets := []dbmodel.Subnet{}
	// Exclude the subnets that are not attached to any app. This shouldn't
	// be the case but let's be safe.
	for i := range restSharedNetwork.Subnets {
		subnet, err := r.convertSubnetFromRestAPI(restSharedNetwork.Subnets[i])
		if err != nil {
			return nil, err
		}
		subnets = append(subnets, *subnet)
	}
	sharedNetwork := &dbmodel.SharedNetwork{
		ID:      restSharedNetwork.ID,
		Name:    restSharedNetwork.Name,
		Family:  int(restSharedNetwork.Universe),
		Subnets: subnets,
	}
	// Convert local shared network containing associations of the shared network with daemons.
	for _, lsn := range restSharedNetwork.LocalSharedNetworks {
		localSharedNetwork := &dbmodel.LocalSharedNetwork{
			DaemonID: lsn.DaemonID,
		}

		if lsn.KeaConfigSharedNetworkParameters != nil && lsn.KeaConfigSharedNetworkParameters.SharedNetworkLevelParameters != nil {
			keaParameters := lsn.KeaConfigSharedNetworkParameters.SharedNetworkLevelParameters
			localSharedNetwork.KeaParameters = &keaconfig.SharedNetworkParameters{
				CacheParameters: keaconfig.CacheParameters{
					CacheThreshold: keaParameters.CacheThreshold,
					CacheMaxAge:    keaParameters.CacheMaxAge,
				},
				ClientClassParameters: keaconfig.ClientClassParameters{
					ClientClass:          storkutil.NullifyEmptyString(keaParameters.ClientClass),
					RequireClientClasses: keaParameters.RequireClientClasses,
				},
				DDNSParameters: keaconfig.DDNSParameters{
					DDNSGeneratedPrefix:        storkutil.NullifyEmptyString(keaParameters.DdnsGeneratedPrefix),
					DDNSOverrideClientUpdate:   keaParameters.DdnsOverrideClientUpdate,
					DDNSOverrideNoUpdate:       keaParameters.DdnsOverrideNoUpdate,
					DDNSQualifyingSuffix:       storkutil.NullifyEmptyString(keaParameters.DdnsQualifyingSuffix),
					DDNSReplaceClientName:      storkutil.NullifyEmptyString(keaParameters.DdnsReplaceClientName),
					DDNSSendUpdates:            keaParameters.DdnsSendUpdates,
					DDNSUpdateOnRenew:          keaParameters.DdnsUpdateOnRenew,
					DDNSUseConflictResolution:  keaParameters.DdnsUseConflictResolution,
					DDNSConflictResolutionMode: keaParameters.DdnsConflictResolutionMode,
					DDNSTTLPercent:             keaParameters.DdnsTTLPercent,
				},
				HostnameCharParameters: keaconfig.HostnameCharParameters{
					HostnameCharReplacement: storkutil.NullifyEmptyString(keaParameters.HostnameCharReplacement),
					HostnameCharSet:         storkutil.NullifyEmptyString(keaParameters.HostnameCharSet),
				},
				PreferredLifetimeParameters: keaconfig.PreferredLifetimeParameters{
					MaxPreferredLifetime: keaParameters.MaxPreferredLifetime,
					MinPreferredLifetime: keaParameters.MinPreferredLifetime,
					PreferredLifetime:    keaParameters.PreferredLifetime,
				},
				ReservationParameters: keaconfig.ReservationParameters{
					ReservationMode:       storkutil.NullifyEmptyString(keaParameters.ReservationMode),
					ReservationsGlobal:    keaParameters.ReservationsGlobal,
					ReservationsInSubnet:  keaParameters.ReservationsInSubnet,
					ReservationsOutOfPool: keaParameters.ReservationsOutOfPool,
				},
				TimerParameters: keaconfig.TimerParameters{
					CalculateTeeTimes: keaParameters.CalculateTeeTimes,
					RebindTimer:       keaParameters.RebindTimer,
					RenewTimer:        keaParameters.RenewTimer,
					T1Percent:         keaParameters.T1Percent,
					T2Percent:         keaParameters.T2Percent,
				},
				ValidLifetimeParameters: keaconfig.ValidLifetimeParameters{
					MaxValidLifetime: keaParameters.MaxValidLifetime,
					MinValidLifetime: keaParameters.MinValidLifetime,
					ValidLifetime:    keaParameters.ValidLifetime,
				},
				Allocator:         storkutil.NullifyEmptyString(keaParameters.Allocator),
				Authoritative:     keaParameters.Authoritative,
				BootFileName:      storkutil.NullifyEmptyString(keaParameters.BootFileName),
				Interface:         storkutil.NullifyEmptyString(keaParameters.Interface),
				InterfaceID:       storkutil.NullifyEmptyString(keaParameters.InterfaceID),
				MatchClientID:     keaParameters.MatchClientID,
				NextServer:        storkutil.NullifyEmptyString(keaParameters.NextServer),
				PDAllocator:       storkutil.NullifyEmptyString(keaParameters.PdAllocator),
				RapidCommit:       keaParameters.RapidCommit,
				ServerHostname:    storkutil.NullifyEmptyString(keaParameters.ServerHostname),
				StoreExtendedInfo: keaParameters.StoreExtendedInfo,
			}
			if keaParameters.Relay != nil {
				localSharedNetwork.KeaParameters.Relay = &keaconfig.Relay{
					IPAddresses: keaParameters.Relay.IPAddresses,
				}
			}
			// DHCP options.
			options, err := r.flattenDHCPOptions("", lsn.KeaConfigSharedNetworkParameters.SharedNetworkLevelParameters.Options, 0)
			if err != nil {
				return nil, err
			}
			localSharedNetwork.DHCPOptionSet.SetDHCPOptions(options, keaconfig.NewHasher())
		}
		sharedNetwork.SetLocalSharedNetwork(localSharedNetwork)
	}
	return sharedNetwork, nil
}

// Get the list of shared networks for the given set of parameters.
func (r *RestAPI) getSharedNetworks(offset, limit, appID, family int64, filterText *string, sortField string, sortDir dbmodel.SortDirEnum) (*models.SharedNetworks, error) {
	// get shared networks from db
	dbSharedNetworks, total, err := dbmodel.GetSharedNetworksByPage(r.DB, offset, limit, appID, family, filterText, sortField, sortDir)
	if err != nil {
		return nil, err
	}

	// prepare response
	sharedNetworks := &models.SharedNetworks{
		Total: total,
	}

	// go through shared networks and their subnets from db and change their format to ReST one
	for i := range dbSharedNetworks {
		sharedNetworks.Items = append(sharedNetworks.Items, r.convertSharedNetworkToRestAPI(&dbSharedNetworks[i]))
	}

	return sharedNetworks, nil
}

// Get list of DHCP shared networks. The list can be filtered by app ID, DHCP version and text.
func (r *RestAPI) GetSharedNetworks(ctx context.Context, params dhcp.GetSharedNetworksParams) middleware.Responder {
	var start int64
	if params.Start != nil {
		start = *params.Start
	}

	var limit int64 = 10
	if params.Limit != nil {
		limit = *params.Limit
	}

	var appID int64
	if params.AppID != nil {
		appID = *params.AppID
	}

	var dhcpVer int64
	if params.DhcpVersion != nil {
		dhcpVer = *params.DhcpVersion
	}

	// get shared networks from db
	sharedNetworks, err := r.getSharedNetworks(start, limit, appID, dhcpVer, params.Text, "", dbmodel.SortDirAsc)
	if err != nil {
		msg := "Cannot get shared network from db"
		log.Error(err)
		rsp := dhcp.NewGetSharedNetworksDefault(http.StatusInternalServerError).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}

	rsp := dhcp.NewGetSharedNetworksOK().WithPayload(sharedNetworks)
	return rsp
}

// Returns the detailed shared network information including the shared network and
// global DHCP configuration parameters. The returned information is sufficient to
// open a form for editing the shared network.
func (r *RestAPI) GetSharedNetwork(ctx context.Context, params dhcp.GetSharedNetworkParams) middleware.Responder {
	dbSharedNetwork, err := dbmodel.GetSharedNetwork(r.DB, params.ID)
	if err != nil {
		// Error while communicating with the database.
		msg := fmt.Sprintf("Problem fetching shared network with ID %d from db", params.ID)
		log.WithError(err).Error(msg)
		rsp := dhcp.NewGetSharedNetworkDefault(http.StatusInternalServerError).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}

	if dbSharedNetwork == nil {
		// Subnet not found.
		msg := fmt.Sprintf("Cannot find shared network with ID %d", params.ID)
		log.Error(msg)
		rsp := dhcp.NewGetSharedNetworkDefault(http.StatusNotFound).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}

	sharedNetwork := r.convertSharedNetworkToRestAPI(dbSharedNetwork)
	rsp := dhcp.NewGetSharedNetworkOK().WithPayload(sharedNetwork)
	return rsp
}

// Common function that implements the POST calls to apply and commit a new
// or updated shared network. The ctx parameter is the REST API context. The
// transactionID is the identifier of the current configuration transaction
// used by the function to recover the transaction context. The restSharedNetwork
// is the pointer to the shared network specified by the user. It is converted
// by this function to the database model. The applyFunc is the function of the Kea
// config module that applies the specified shared network. It is one of the
// ApplySharedNetworkAdd or ApplySharedNetworkUpdate, depending on whether the
// new shared network is created (via CreateSharedNetworkSubmit) or updated
// (via UpdateSharedNetworkSubmit). The apply functions receive the transaction
// context and a pointer to the shared network. They return the updated context
// and error. This function returns the HTTP error code if an error occurs or 0
// when there is no error. It also returns an ID of the created or modified shared
// network. Finally, it returns an error string to be included in the HTTP response
// or an empty string if there is no error.
func (r *RestAPI) commonCreateOrUpdateSharedNetworkSubmit(ctx context.Context, transactionID int64, restSharedNetwork *models.SharedNetwork, applyFunc func(context.Context, *dbmodel.SharedNetwork) (context.Context, error)) (int, int64, string) {
	// Make sure that the shared network information is present.
	if restSharedNetwork == nil {
		msg := "Shared network information not specified"
		log.Errorf("Problem with submitting a shared network because the shared network information is missing")
		return http.StatusBadRequest, 0, msg
	}
	// Retrieve the context from the config manager.
	_, user := r.SessionManager.Logged(ctx)
	cctx, _ := r.ConfigManager.RecoverContext(transactionID, int64(user.ID))
	if cctx == nil {
		msg := "Transaction expired for the shared network update"
		log.Errorf("Problem with recovering transaction context for transaction ID %d and user ID %d", transactionID, user.ID)
		return http.StatusNotFound, 0, msg
	}

	// Convert shared network information from REST API to database format.
	sharedNetwork, err := r.convertSharedNetworkFromRestAPI(restSharedNetwork)
	if err != nil {
		msg := "Error parsing specified shared network"
		log.WithError(err).Error(msg)
		return http.StatusBadRequest, 0, msg
	}
	err = sharedNetwork.PopulateDaemons(r.DB)
	if err != nil {
		msg := "Specified shared network is associated with daemons that no longer exist"
		log.WithError(err).Error(msg)
		return http.StatusNotFound, 0, msg
	}
	// Apply the shared network information (create Kea commands).
	cctx, err = applyFunc(cctx, sharedNetwork)
	if err != nil {
		msg := fmt.Sprintf("Problem with applying shared network information: %s", err)
		log.WithError(err).Error(msg)
		return http.StatusInternalServerError, 0, msg
	}
	// Send the commands to Kea servers.
	cctx, err = r.ConfigManager.Commit(cctx)
	if err != nil {
		msg := fmt.Sprintf("Problem with committing shared network information: %s", err)
		log.WithError(err).Error(msg)
		return http.StatusConflict, 0, msg
	}
	sharedNetworkID := restSharedNetwork.ID
	if sharedNetworkID == 0 {
		recipe, err := config.GetRecipeForUpdate[kea.ConfigRecipe](cctx, 0)
		if err != nil {
			msg := "Problem recovering shared network ID from the context"
			log.WithError(err).Error(msg)
			return http.StatusInternalServerError, 0, msg
		}
		if recipe.SharedNetworkID != nil {
			sharedNetworkID = *recipe.SharedNetworkID
		}
	}
	// Everything ok. Cleanup and send OK to the client.
	r.ConfigManager.Done(cctx)
	return 0, sharedNetworkID, ""
}

// Common function that implements the DELETE calls to cancel adding new
// or updating a shared network. It removes the specified transaction from the
// config manager, if the transaction exists. It returns the HTTP error code
// if an error occurs or 0 when there is no error. In addition it returns an
// error string to be included in the HTTP response or an empty string if there
// is no error.
func (r *RestAPI) commonCreateOrUpdateSharedNetworkDelete(ctx context.Context, transactionID int64) (int, string) {
	// Retrieve the context from the config manager.
	_, user := r.SessionManager.Logged(ctx)
	cctx, _ := r.ConfigManager.RecoverContext(transactionID, int64(user.ID))
	if cctx == nil {
		msg := "Transaction expired for the shared network update"
		log.Errorf("Problem with recovering transaction context for transaction ID %d and user ID %d", transactionID, user.ID)
		return http.StatusNotFound, msg
	}
	r.ConfigManager.Done(cctx)
	return 0, ""
}

// Implements the POST call to create new transaction for creating a new
// shared network (shared-networks/new/transaction).
func (r *RestAPI) CreateSharedNetworkBegin(ctx context.Context, params dhcp.CreateSharedNetworkBeginParams) middleware.Responder {
	// Execute the common part between create and update operations. It retrieves
	// the daemons and creates a transaction context.
	respDaemons, respIPv4SharedNetworks, respIPv6SharedNetworks, respClientClasses, cctx, code, msg := r.commonCreateOrUpdateNetworkBegin(ctx)
	if code != 0 {
		// Error case.
		rsp := dhcp.NewCreateSubnetBeginDefault(code).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}
	// Begin the transaction.
	var err error
	cctx, err = r.ConfigManager.GetKeaModule().BeginSharedNetworkAdd(cctx)
	if err != nil {
		msg := "Problem with initializing transaction for creating a shared network"
		log.WithError(err).Error(msg)
		rsp := dhcp.NewCreateSharedNetworkBeginDefault(http.StatusInternalServerError).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}
	// Retrieve the generated context ID.
	cctxID, ok := config.GetValueAsInt64(cctx, config.ContextIDKey)
	if !ok {
		msg := "problem with retrieving context ID for a transaction to create a shared network"
		log.Error(msg)
		rsp := dhcp.NewCreateSharedNetworkBeginDefault(http.StatusInternalServerError).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}
	// Remember the context, i.e. new transaction has been successfully created.
	_ = r.ConfigManager.RememberContext(cctx, time.Minute*10)

	// Return transaction ID, daemons and client classes to the user.
	contents := &models.CreateSharedNetworkBeginResponse{
		ID:            cctxID,
		Daemons:       respDaemons,
		ClientClasses: respClientClasses,
	}
	for _, sn := range respIPv4SharedNetworks {
		contents.SharedNetworks4 = append(contents.SharedNetworks4, sn.Name)
	}
	for _, sn := range respIPv6SharedNetworks {
		contents.SharedNetworks6 = append(contents.SharedNetworks6, sn.Name)
	}
	rsp := dhcp.NewCreateSharedNetworkBeginOK().WithPayload(contents)
	return rsp
}

// Implements the POST call and commits a new shared network
// (shared-networks/new/transaction/{id}/submit).
func (r *RestAPI) CreateSharedNetworkSubmit(ctx context.Context, params dhcp.CreateSharedNetworkSubmitParams) middleware.Responder {
	code, sharedNetworkID, msg := r.commonCreateOrUpdateSharedNetworkSubmit(ctx, params.ID, params.SharedNetwork, r.ConfigManager.GetKeaModule().ApplySharedNetworkAdd)
	if code != 0 {
		// Error case.
		rsp := dhcp.NewCreateSharedNetworkSubmitDefault(code).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}
	contents := &models.CreateSharedNetworkSubmitResponse{
		SharedNetworkID: sharedNetworkID,
	}
	rsp := dhcp.NewCreateSharedNetworkSubmitOK().WithPayload(contents)
	return rsp
}

// Implements the DELETE call to cancel creating a shared network
// (shared-networks/new/transaction/{id}).
// It removes the specified transaction from the config manager,
// if the transaction exists.
func (r *RestAPI) CreateSharedNetworkDelete(ctx context.Context, params dhcp.CreateSharedNetworkDeleteParams) middleware.Responder {
	if code, msg := r.commonCreateOrUpdateSharedNetworkDelete(ctx, params.ID); code != 0 {
		// Error case.
		rsp := dhcp.NewCreateSharedNetworkDeleteDefault(code).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}
	rsp := dhcp.NewCreateSharedNetworkDeleteOK()
	return rsp
}

// Implements the POST call to create new transaction for updating an
// existing shared network (shared-networks/{sharedNetworkId}/transaction).
func (r *RestAPI) UpdateSharedNetworkBegin(ctx context.Context, params dhcp.UpdateSharedNetworkBeginParams) middleware.Responder {
	// Execute the common part between create and update operations. It retrieves
	// the daemons and creates a transaction context.
	respDaemons, respIPv4SharedNetworks, respIPv6SharedNetworks, respClientClasses, cctx, code, msg := r.commonCreateOrUpdateNetworkBegin(ctx)
	if code != 0 {
		// Error case.
		rsp := dhcp.NewUpdateSharedNetworkBeginDefault(code).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}
	// Begin shared network update transaction. It retrieves current shared network
	// information and locks daemons for updates.
	var err error
	cctx, err = r.ConfigManager.GetKeaModule().BeginSharedNetworkUpdate(cctx, params.SharedNetworkID)
	if err != nil {
		var (
			sharedNetworkNotFound *config.SharedNetworkNotFoundError
			lock                  *config.LockError
			hooksNotConfigured    *config.NoSubnetCmdsHookError
		)
		switch {
		case errors.As(err, &sharedNetworkNotFound):
			// Failed to find shared network.
			msg := fmt.Sprintf("Unable to edit the shared network with ID %d because it cannot be found", params.SharedNetworkID)
			log.Error(msg)
			rsp := dhcp.NewUpdateSharedNetworkBeginDefault(http.StatusBadRequest).WithPayload(&models.APIError{
				Message: &msg,
			})
			return rsp
		case errors.As(err, &lock):
			// Failed to lock daemons.
			msg := fmt.Sprintf("Unable to edit the shared network with ID %d because it may be currently edited by another user", params.SharedNetworkID)
			log.WithError(err).Error(msg)
			rsp := dhcp.NewUpdateSharedNetworkBeginDefault(http.StatusLocked).WithPayload(&models.APIError{
				Message: &msg,
			})
			return rsp
		case errors.As(err, &hooksNotConfigured):
			// Lack of the libdhcp_subnet_cmds hook.
			msg := "Unable to update shared network configuration because some daemons lack libdhcp_subnet_cmds hook library"
			log.Error(msg)
			rsp := dhcp.NewUpdateSharedNetworkBeginDefault(http.StatusBadRequest).WithPayload(&models.APIError{
				Message: &msg,
			})
			return rsp
		default:
			// Other error.
			msg := fmt.Sprintf("Problem with initializing transaction for an update of the shared network with ID %d", params.SharedNetworkID)
			log.WithError(err).Error(msg)
			rsp := dhcp.NewUpdateSharedNetworkBeginDefault(http.StatusInternalServerError).WithPayload(&models.APIError{
				Message: &msg,
			})
			return rsp
		}
	}
	state, _ := config.GetTransactionState[kea.ConfigRecipe](cctx)
	sharedNetwork := state.Updates[0].Recipe.SharedNetworkBeforeUpdate

	// Retrieve the generated context ID.
	cctxID, ok := config.GetValueAsInt64(cctx, config.ContextIDKey)
	if !ok {
		msg := "problem with retrieving context ID for a transaction to update a shared network"
		log.Error(msg)
		rsp := dhcp.NewUpdateSubnetBeginDefault(http.StatusInternalServerError).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}
	// Remember the context, i.e. new transaction has been successfully created.
	_ = r.ConfigManager.RememberContext(cctx, time.Minute*10)

	// Return transaction ID and daemons to the user.
	contents := &models.UpdateSharedNetworkBeginResponse{
		ID:            cctxID,
		SharedNetwork: r.convertSharedNetworkToRestAPI(sharedNetwork),
		Daemons:       respDaemons,
		ClientClasses: respClientClasses,
	}
	for _, sn := range respIPv4SharedNetworks {
		contents.SharedNetworks4 = append(contents.SharedNetworks4, sn.Name)
	}
	for _, sn := range respIPv6SharedNetworks {
		contents.SharedNetworks6 = append(contents.SharedNetworks6, sn.Name)
	}
	rsp := dhcp.NewUpdateSharedNetworkBeginOK().WithPayload(contents)
	return rsp
}

// Implements the POST call and commits an updated shared network
// (shared-networks/{sharedNetworkId}/transaction/{id}/submit).
func (r *RestAPI) UpdateSharedNetworkSubmit(ctx context.Context, params dhcp.UpdateSharedNetworkSubmitParams) middleware.Responder {
	if code, _, msg := r.commonCreateOrUpdateSharedNetworkSubmit(ctx, params.ID, params.SharedNetwork, r.ConfigManager.GetKeaModule().ApplySharedNetworkUpdate); code != 0 {
		// Error case.
		rsp := dhcp.NewUpdateSharedNetworkSubmitDefault(code).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}
	rsp := dhcp.NewUpdateSharedNetworkSubmitOK()
	return rsp
}

// Implements the DELETE call to cancel updating a shared network
// (shared-networks/{sharedNetworkId}/transaction/{id}).
// It removes the specified transaction from the config manager,
// if the transaction exists.
func (r *RestAPI) UpdateSharedNetworkDelete(ctx context.Context, params dhcp.UpdateSharedNetworkDeleteParams) middleware.Responder {
	if code, msg := r.commonCreateOrUpdateSharedNetworkDelete(ctx, params.ID); code != 0 {
		// Error case.
		rsp := dhcp.NewUpdateSharedNetworkDeleteDefault(code).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}
	rsp := dhcp.NewUpdateSharedNetworkDeleteOK()
	return rsp
}

// Implements the DELETE call for a shared network (shared-networks/{id}). It sends suitable
// commands to the Kea servers owning the shared network. Deleting shared network is not
// transactional. It could be implemented as a transaction with first REST API call ensuring
// that the shared network still exists in Stork database and locking configuration changes
// for the daemons owning the shared network. However, it seems to be too much overhead with
// little gain. If the shared network doesn't exist this call will return an error anyway.
func (r *RestAPI) DeleteSharedNetwork(ctx context.Context, params dhcp.DeleteSharedNetworkParams) middleware.Responder {
	dbSharedNetwork, err := dbmodel.GetSharedNetwork(r.DB, params.ID)
	if err != nil {
		// Error while communicating with the database.
		msg := fmt.Sprintf("Problem fetching shared network with ID %d from db", params.ID)
		log.WithError(err).Error(msg)
		rsp := dhcp.NewDeleteSharedNetworkDefault(http.StatusInternalServerError).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}
	if dbSharedNetwork == nil {
		// Shared network not found.
		msg := fmt.Sprintf("Cannot find a shared network with ID %d", params.ID)
		rsp := dhcp.NewDeleteSharedNetworkDefault(http.StatusNotFound).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}
	// Create configuration context.
	_, user := r.SessionManager.Logged(ctx)
	cctx, err := r.ConfigManager.CreateContext(int64(user.ID))
	if err != nil {
		msg := "Problem with creating transaction context for deleting the shared network"
		log.WithError(err).Error(err)
		rsp := dhcp.NewDeleteSharedNetworkDefault(http.StatusInternalServerError).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}
	// Create Kea commands to delete the shared network.
	cctx, err = r.ConfigManager.GetKeaModule().ApplySharedNetworkDelete(cctx, dbSharedNetwork)
	if err != nil {
		msg := "Problem with preparing commands for deleting the shared network"
		log.WithError(err).Error(msg)
		rsp := dhcp.NewDeleteSharedNetworkDefault(http.StatusInternalServerError).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}
	// Send the commands to Kea servers.
	_, err = r.ConfigManager.Commit(cctx)
	if err != nil {
		msg := fmt.Sprintf("Problem with deleting a shared network: %s", err)
		log.WithError(err).Error(msg)
		rsp := dhcp.NewDeleteSharedNetworkDefault(http.StatusConflict).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}
	// Send OK to the client.
	rsp := dhcp.NewDeleteSharedNetworkOK()
	return rsp
}
