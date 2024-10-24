package restservice

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/go-openapi/runtime/middleware"
	"github.com/go-openapi/strfmt"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	keaconfig "isc.org/stork/appcfg/kea"
	"isc.org/stork/server/apps/kea"
	"isc.org/stork/server/config"
	"isc.org/stork/server/configreview"
	dbmodel "isc.org/stork/server/database/model"
	"isc.org/stork/server/gen/models"
	dhcp "isc.org/stork/server/gen/restapi/operations/d_h_c_p"
	"isc.org/stork/server/gen/restapi/operations/services"
	storkutil "isc.org/stork/util"
)

// Get daemon config. Only Kea daemon supported.
func (r *RestAPI) GetDaemonConfig(ctx context.Context, params services.GetDaemonConfigParams) middleware.Responder {
	dbDaemon, err := dbmodel.GetDaemonByID(r.DB, params.ID)
	if err != nil {
		log.Error(err)
		msg := fmt.Sprintf("Cannot get daemon with ID %d from db", params.ID)
		rsp := services.NewGetDaemonConfigDefault(http.StatusInternalServerError).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}
	if dbDaemon == nil {
		msg := fmt.Sprintf("Cannot find daemon with ID %d", params.ID)
		rsp := services.NewGetDaemonConfigDefault(http.StatusBadRequest).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}

	if dbDaemon.KeaDaemon == nil {
		msg := fmt.Sprintf("Daemon with ID %d is not a Kea daemon", params.ID)
		rsp := services.NewGetDaemonConfigDefault(http.StatusBadRequest).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}
	if dbDaemon.KeaDaemon.Config == nil {
		msg := fmt.Sprintf("Config not assigned for daemon with ID %d", params.ID)
		rsp := services.NewGetDaemonConfigDefault(http.StatusNotFound).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}

	_, dbUser := r.SessionManager.Logged(ctx)
	if !dbUser.InGroup(&dbmodel.SystemGroup{ID: dbmodel.SuperAdminGroupID}) {
		dbDaemon.KeaDaemon.Config.HideSensitiveData()
	}

	var options *models.DHCPOptions
	if dbDaemon.KeaDaemon.Config.IsDHCPv4() || dbDaemon.KeaDaemon.Config.IsDHCPv6() {
		ipType := storkutil.IPType(4)
		if dbDaemon.KeaDaemon.Config.IsDHCPv6() {
			ipType = storkutil.IPType(6)
		}

		var convertedOptions []dbmodel.DHCPOption
		for _, option := range dbDaemon.KeaDaemon.Config.GetDHCPOptions() {
			convertedOption, err := dbmodel.NewDHCPOptionFromKea(
				option, ipType, r.DHCPOptionDefinitionLookup,
			)
			if err != nil {
				continue
			}
			convertedOptions = append(convertedOptions, *convertedOption)
		}

		options = &models.DHCPOptions{
			OptionsHash: keaconfig.NewHasher().Hash(convertedOptions),
			Options:     r.unflattenDHCPOptions(convertedOptions, "", 0),
		}
	}

	rsp := services.NewGetDaemonConfigOK().WithPayload(&models.KeaDaemonConfig{
		DaemonID:   dbDaemon.GetID(),
		AppID:      dbDaemon.App.GetID(),
		AppName:    dbDaemon.App.GetName(),
		AppType:    dbDaemon.GetAppType().String(),
		DaemonName: dbDaemon.GetName(),
		Editable:   dbDaemon.Monitored && dbDaemon.Active,
		Config:     dbDaemon.KeaDaemon.Config,
		Options:    options,
	})
	return rsp
}

// Get configuration review reports for a specified daemon. Only Kea
// daemons are currently supported. The daemon id value is mandatory.
// The start and limit values are optional. They are used to retrieve
// paged configuration review reports for a daemon. If they are not
// specified, all configuration reports are returned. When the
// configuration review is in progress for the specified daemon it
// returns HTTP Accepted status code. When the review hasn't been
// yet performed for the daemon, it returns HTTP No Content status
// code. If the review is available it returns HTTP OK status code.
func (r *RestAPI) GetDaemonConfigReports(ctx context.Context, params services.GetDaemonConfigReportsParams) middleware.Responder {
	// If the review is in progress return HTTP Accepted status
	// code to indicate that the caller can try again soon to
	// get the new reports.
	if r.ReviewDispatcher.ReviewInProgress(params.ID) {
		rsp := services.NewGetDaemonConfigReportsAccepted()
		return rsp
	}

	// Get the basic information about the last review.
	review, err := dbmodel.GetConfigReviewByDaemonID(r.DB, params.ID)
	if err != nil {
		log.Error(err)
		msg := fmt.Sprintf("Cannot get configuration review for daemon with ID %d from db", params.ID)
		rsp := services.NewGetDaemonConfigReportsDefault(http.StatusInternalServerError).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}
	if review == nil {
		// If the information is not present it means that the daemon
		// configuration has never been reviewed. HTTP No Content status
		// indicates it to the client.
		rsp := services.NewGetDaemonConfigReportsNoContent()
		return rsp
	}

	start := int64(0)
	if params.Start != nil {
		start = *params.Start
	}

	limit := int64(0)
	if params.Limit != nil {
		limit = *params.Limit
	}

	issuesOnly := params.IssuesOnly != nil && *params.IssuesOnly
	dbReports, total, err := dbmodel.GetConfigReportsByDaemonID(r.DB, start, limit, params.ID, issuesOnly)
	if err != nil {
		log.Error(err)
		msg := fmt.Sprintf("Cannot get configuration review reports for daemon with ID %d from db", params.ID)
		rsp := services.NewGetDaemonConfigReportsDefault(http.StatusInternalServerError).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}

	var totalReports int64
	var totalIssues int64
	if issuesOnly {
		totalIssues = total
		totalReports, err = dbmodel.CountConfigReportsByDaemonID(r.DB, params.ID, false)
	} else {
		totalIssues, err = dbmodel.CountConfigReportsByDaemonID(r.DB, params.ID, true)
		totalReports = total
	}
	if err != nil {
		log.Error(err)
		msg := fmt.Sprintf("Cannot count configuration review reports for daemon with ID %d from db", params.ID)
		rsp := services.NewGetDaemonConfigReportsDefault(http.StatusInternalServerError).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}

	configReports := &models.ConfigReports{
		Review: &models.ConfigReview{
			ID:        review.ID,
			DaemonID:  review.DaemonID,
			CreatedAt: strfmt.DateTime(review.CreatedAt),
		},
		Total:        total,
		TotalIssues:  totalIssues,
		TotalReports: totalReports,
	}

	for _, dbReport := range dbReports {
		report := &models.ConfigReport{
			ID:        dbReport.ID,
			CreatedAt: strfmt.DateTime(dbReport.CreatedAt),
			Checker:   dbReport.CheckerName,
			Content:   dbReport.Content,
		}
		configReports.Items = append(configReports.Items, report)
	}

	rsp := services.NewGetDaemonConfigReportsOK().WithPayload(configReports)
	return rsp
}

// Begins daemon configuration review on demand.
func (r *RestAPI) PutDaemonConfigReview(ctx context.Context, params services.PutDaemonConfigReviewParams) middleware.Responder {
	// Try to get the daemon information from the database.
	daemon, err := dbmodel.GetDaemonByID(r.DB, params.ID)
	if err != nil {
		log.Error(err)
		msg := fmt.Sprintf("Cannot get daemon with ID %d from db", params.ID)
		rsp := services.NewPutDaemonConfigReviewDefault(http.StatusInternalServerError).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}
	// If the daemon doesn't exist there is nothing to do. Return the
	// HTTP Bad Request status.
	if daemon == nil {
		msg := fmt.Sprintf("Cannot find daemon with ID %d", params.ID)
		rsp := services.NewPutDaemonConfigReviewDefault(http.StatusBadRequest).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}
	// Config review is currently only supported for Kea.
	if daemon.KeaDaemon == nil {
		msg := fmt.Sprintf("Daemon with ID %d is not a Kea daemon", params.ID)
		rsp := services.NewPutDaemonConfigReviewDefault(http.StatusBadRequest).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}
	// Config must be present to perform the review.
	if daemon.KeaDaemon.Config == nil {
		msg := fmt.Sprintf("Configuration not found for daemon with ID %d", params.ID)
		rsp := services.NewPutDaemonConfigReviewDefault(http.StatusBadRequest).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}

	// Begin the review but do not wait for the result.
	_ = r.ReviewDispatcher.BeginReview(daemon, configreview.Triggers{configreview.ManualRun}, nil)

	// Inform the caller that the review request has been "accepted".
	rsp := services.NewPutDaemonConfigReviewAccepted()
	return rsp
}

// Converts the internal config checker metadata to the REST API
// structure.
func convertConfigCheckerMetadataToRestAPI(metadata []*configreview.CheckerMetadata) *models.ConfigCheckers {
	checkers := make([]*models.ConfigChecker, len(metadata))
	for i, m := range metadata {
		var selectors []string
		for _, selector := range m.Selectors {
			selectors = append(selectors, selector.String())
		}

		var triggers []string
		for _, trigger := range m.Triggers {
			triggers = append(triggers, string(trigger))
		}

		checkers[i] = &models.ConfigChecker{
			Name:            storkutil.Ptr(m.Name),
			Selectors:       selectors,
			State:           m.State,
			Triggers:        triggers,
			GloballyEnabled: storkutil.Ptr(m.GloballyEnabled),
		}
	}

	payload := &models.ConfigCheckers{
		Items: checkers,
		Total: int64(len(checkers)),
	}

	return payload
}

// Converts the config checker state from RestAPI to the internal type.
func convertConfigCheckerStateFromRestAPI(state models.ConfigCheckerState) (configreview.CheckerState, bool) {
	switch state {
	case models.ConfigCheckerStateEnabled:
		return configreview.CheckerStateEnabled, true
	case models.ConfigCheckerStateDisabled:
		return configreview.CheckerStateDisabled, true
	case models.ConfigCheckerStateInherit:
		return configreview.CheckerStateInherit, true
	default:
		log.WithField("state", state).Error("Received unknown config checker state")
		return configreview.CheckerStateEnabled, false
	}
}

// Returns global config checkers metadata.
func (r *RestAPI) GetGlobalConfigCheckers(ctx context.Context, params services.GetGlobalConfigCheckersParams) middleware.Responder {
	metadata, err := r.ReviewDispatcher.GetCheckersMetadata(nil)
	if err != nil {
		log.Error(err)
		msg := "cannot get the global checkers metadata"
		rsp := services.NewGetGlobalConfigCheckersDefault(http.StatusInternalServerError).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}

	payload := convertConfigCheckerMetadataToRestAPI(metadata)
	rsp := services.NewGetGlobalConfigCheckersOK().WithPayload(payload)
	return rsp
}

// Returns the config checkers metadata for a given daemon.
func (r *RestAPI) GetDaemonConfigCheckers(ctx context.Context, params services.GetDaemonConfigCheckersParams) middleware.Responder {
	daemon, err := dbmodel.GetDaemonByID(r.DB, params.ID)
	if err != nil {
		log.Error(err)
		msg := fmt.Sprintf("Cannot get daemon with ID %d from db", params.ID)
		rsp := services.NewGetDaemonConfigCheckersDefault(http.StatusInternalServerError).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}
	if daemon == nil {
		msg := fmt.Sprintf("Cannot find daemon with ID %d", params.ID)
		rsp := services.NewGetDaemonConfigCheckersDefault(http.StatusBadRequest).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}

	metadata, err := r.ReviewDispatcher.GetCheckersMetadata(daemon)
	if err != nil {
		log.Error(err)
		msg := fmt.Sprintf("Cannot get checkers metadata for daemon (ID: %d, Name: %s)", daemon.ID, daemon.Name)
		rsp := services.NewGetDaemonConfigCheckersDefault(http.StatusInternalServerError).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}

	payload := convertConfigCheckerMetadataToRestAPI(metadata)

	rsp := services.NewGetDaemonConfigCheckersOK().WithPayload(payload)
	return rsp
}

// Modifies the checker preferences for a given daemon. The changes are
// persistent. It returns a list of actual config checker metadata for given
// daemon.
func (r *RestAPI) PutDaemonConfigCheckerPreferences(ctx context.Context, params services.PutDaemonConfigCheckerPreferencesParams) middleware.Responder {
	daemon, err := dbmodel.GetDaemonByID(r.DB, params.ID)
	if err != nil {
		log.Error(err)
		msg := fmt.Sprintf("Cannot get daemon with ID %d from db", params.ID)
		rsp := services.NewPutDaemonConfigCheckerPreferencesDefault(http.StatusInternalServerError).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}
	if daemon == nil {
		msg := fmt.Sprintf("Cannot find daemon with ID %d", params.ID)
		rsp := services.NewPutDaemonConfigCheckerPreferencesDefault(http.StatusBadRequest).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}

	var newOrUpdatedPreferences []*dbmodel.ConfigCheckerPreference
	var deletedPreferences []*dbmodel.ConfigCheckerPreference
	for _, change := range params.Changes.Items {
		apiState := models.ConfigCheckerState(change.State.(string))
		state, ok := convertConfigCheckerStateFromRestAPI(apiState)
		if !ok {
			log.Errorf("Received unknown checker state %s", apiState)
			msg := fmt.Sprintf("Cannot parse the checker state %s", apiState)
			rsp := services.NewPutDaemonConfigCheckerPreferencesDefault(http.StatusBadRequest).WithPayload(&models.APIError{
				Message: &msg,
			})
			return rsp
		}

		err = r.ReviewDispatcher.SetCheckerState(daemon, change.Name, state)
		if err != nil {
			log.Error(err)
			msg := fmt.Sprintf("Cannot set the state for the %s checker", change.Name)
			rsp := services.NewPutDaemonConfigCheckerPreferencesDefault(http.StatusInternalServerError).WithPayload(&models.APIError{
				Message: &msg,
			})
			return rsp
		}

		if state == configreview.CheckerStateInherit {
			deletedPreferences = append(
				deletedPreferences,
				dbmodel.NewDaemonConfigCheckerPreference(daemon.ID, change.Name, true),
			)
		} else {
			newOrUpdatedPreferences = append(
				newOrUpdatedPreferences,
				dbmodel.NewDaemonConfigCheckerPreference(
					daemon.ID,
					change.Name,
					state == configreview.CheckerStateEnabled,
				),
			)
		}
	}

	err = dbmodel.CommitCheckerPreferences(r.DB, newOrUpdatedPreferences, deletedPreferences)
	if err != nil {
		log.Error(err)
		msg := "Cannot commit the config checker changes into DB"
		rsp := services.NewPutDaemonConfigCheckerPreferencesDefault(http.StatusInternalServerError).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}

	metadata, err := r.ReviewDispatcher.GetCheckersMetadata(daemon)
	if err != nil {
		log.Error(err)
		msg := fmt.Sprintf("Cannot get checkers metadata for daemon (ID: %d, Name: %s)", daemon.ID, daemon.Name)
		rsp := services.NewPutDaemonConfigCheckerPreferencesDefault(http.StatusInternalServerError).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}

	payload := convertConfigCheckerMetadataToRestAPI(metadata)

	rsp := services.NewPutDaemonConfigCheckerPreferencesOK().WithPayload(payload)
	return rsp
}

// Modifies the global checker preferences. The changes are persistent.
// It returns a list of actual global config checker metadata.
func (r *RestAPI) PutGlobalConfigCheckerPreferences(ctx context.Context, params services.PutGlobalConfigCheckerPreferencesParams) middleware.Responder {
	var newOrUpdatedPreferences []*dbmodel.ConfigCheckerPreference
	var deletedPreferences []*dbmodel.ConfigCheckerPreference

	for _, change := range params.Changes.Items {
		apiState := models.ConfigCheckerState(change.State.(string))
		if state, ok := convertConfigCheckerStateFromRestAPI(apiState); ok {
			err := r.ReviewDispatcher.SetCheckerState(nil, change.Name, state)
			if err != nil {
				log.Error(err)
				msg := fmt.Sprintf("Cannot set the global state for the %s checker", change.Name)
				rsp := services.NewPutDaemonConfigCheckerPreferencesDefault(http.StatusInternalServerError).WithPayload(&models.APIError{
					Message: &msg,
				})
				return rsp
			}

			if state == configreview.CheckerStateDisabled {
				newOrUpdatedPreferences = append(
					newOrUpdatedPreferences,
					dbmodel.NewGlobalConfigCheckerPreference(change.Name),
				)
			} else {
				deletedPreferences = append(
					deletedPreferences,
					dbmodel.NewGlobalConfigCheckerPreference(change.Name),
				)
			}
		}
	}

	err := dbmodel.CommitCheckerPreferences(r.DB, newOrUpdatedPreferences, deletedPreferences)
	if err != nil {
		log.Error(err)
		msg := "Cannot commit the config checker changes into DB"
		rsp := services.NewPutDaemonConfigCheckerPreferencesDefault(http.StatusInternalServerError).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}

	metadata, err := r.ReviewDispatcher.GetCheckersMetadata(nil)
	if err != nil {
		log.Error(err)
		msg := "Cannot get global checkers metadata for daemon"
		rsp := services.NewPutDaemonConfigCheckerPreferencesDefault(http.StatusInternalServerError).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}

	payload := convertConfigCheckerMetadataToRestAPI(metadata)

	rsp := services.NewGetDaemonConfigCheckersOK().WithPayload(payload)
	return rsp
}

// Deletes the Kea daemon config hashes effectively causing the Stork server
// to fetch and update Kea configurations in the Stork server's database.
func (r *RestAPI) DeleteKeaDaemonConfigHashes(ctx context.Context, params services.DeleteKeaDaemonConfigHashesParams) middleware.Responder {
	err := dbmodel.DeleteKeaDaemonConfigHashes(r.DB)
	if err != nil {
		msg := "Cannot reset Kea configurations"
		log.WithError(err).Error(msg)
		rsp := services.NewDeleteKeaDaemonConfigHashesDefault(http.StatusInternalServerError).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}
	rsp := services.NewDeleteKeaDaemonConfigHashesOK()
	return rsp
}

// Implements the POST call to create new transaction for updating global
// Kea configurations (kea-global-parameters/transaction).
func (r *RestAPI) UpdateKeaGlobalParametersBegin(ctx context.Context, params dhcp.UpdateKeaGlobalParametersBeginParams) middleware.Responder {
	// Create configuration context.
	_, user := r.SessionManager.Logged(ctx)
	cctx, err := r.ConfigManager.CreateContext(int64(user.ID))
	if err != nil {
		msg := "Problem with creating transaction context"
		log.WithError(err).Error(msg)
		rsp := dhcp.NewUpdateKeaGlobalParametersBeginDefault(http.StatusInternalServerError).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}
	cctx, err = r.ConfigManager.GetKeaModule().BeginGlobalParametersUpdate(cctx, params.Request.DaemonIds)
	if err != nil {
		var (
			someDaemonsNotFound *config.SomeDaemonsNotFoundError
			lock                *config.LockError
		)
		switch {
		case errors.As(err, &someDaemonsNotFound):
			// Failed to find some of the daemons.
			msg := "Unable to update the Kea global parameters because some of the specified daemons do not exist"
			log.Error(msg)
			rsp := dhcp.NewUpdateKeaGlobalParametersBeginDefault(http.StatusBadRequest).WithPayload(&models.APIError{
				Message: &msg,
			})
			return rsp
		case errors.As(err, &lock):
			// Failed to lock daemons.
			msg := "Unable to edit Kea global parameters because they may be currently edited by another user"
			log.WithError(err).Error(msg)
			rsp := dhcp.NewUpdateKeaGlobalParametersBeginDefault(http.StatusLocked).WithPayload(&models.APIError{
				Message: &msg,
			})
			return rsp
		default:
			// Other error.
			msg := "Problem with initializing transaction for an update of the Kea configs"
			log.WithError(err).Error(msg)
			rsp := dhcp.NewUpdateKeaGlobalParametersBeginDefault(http.StatusInternalServerError).WithPayload(&models.APIError{
				Message: &msg,
			})
			return rsp
		}
	}

	state, _ := config.GetTransactionState[kea.ConfigRecipe](cctx)
	daemons := state.Updates[0].Recipe.KeaDaemonsBeforeConfigUpdate

	// Retrieve the generated context ID.
	cctxID, ok := config.GetValueAsInt64(cctx, config.ContextIDKey)
	if !ok {
		msg := "problem with retrieving context ID for a transaction to update Kea configs"
		log.Error(msg)
		rsp := dhcp.NewUpdateKeaGlobalParametersBeginDefault(http.StatusInternalServerError).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}
	// Remember the context, i.e. new transaction has been successfully created.
	_ = r.ConfigManager.RememberContext(cctx, time.Minute*10)

	var configs []*models.KeaDaemonConfig
	for _, daemon := range daemons {
		if daemon.KeaDaemon == nil {
			continue
		}

		var options *models.DHCPOptions
		if daemon.KeaDaemon.Config.IsDHCPv4() || daemon.KeaDaemon.Config.IsDHCPv6() {
			ipType := storkutil.IPType(4)
			if daemon.KeaDaemon.Config.IsDHCPv6() {
				ipType = storkutil.IPType(6)
			}

			var convertedOptions []dbmodel.DHCPOption
			for _, option := range daemon.KeaDaemon.Config.GetDHCPOptions() {
				convertedOption, err := dbmodel.NewDHCPOptionFromKea(
					option, ipType, r.DHCPOptionDefinitionLookup,
				)
				if err != nil {
					continue
				}
				convertedOptions = append(convertedOptions, *convertedOption)
			}

			options = &models.DHCPOptions{
				OptionsHash: keaconfig.NewHasher().Hash(convertedOptions),
				Options:     r.unflattenDHCPOptions(convertedOptions, "", 0),
			}
		}

		configs = append(configs, &models.KeaDaemonConfig{
			AppID:         daemon.GetAppID(),
			AppName:       daemon.App.GetName(),
			AppType:       "kea",
			DaemonID:      daemon.ID,
			DaemonName:    daemon.Name,
			DaemonVersion: daemon.Version,
			Config:        daemon.KeaDaemon.Config,
			Options:       options,
		})
	}

	// Return transaction ID and daemons to the user.
	contents := &models.UpdateKeaDaemonsGlobalParametersBeginResponse{
		ID:      cctxID,
		Configs: configs,
	}
	rsp := dhcp.NewUpdateKeaGlobalParametersBeginOK().WithPayload(contents)
	return rsp
}

// Implements the POST call and commits updated global Kea configurations
// (kea-global-parameters/transaction/{id}/submit).
func (r *RestAPI) UpdateKeaGlobalParametersSubmit(ctx context.Context, params dhcp.UpdateKeaGlobalParametersSubmitParams) middleware.Responder {
	// Retrieve the context from the config manager.
	_, user := r.SessionManager.Logged(ctx)
	cctx, _ := r.ConfigManager.RecoverContext(params.ID, int64(user.ID))
	if cctx == nil {
		msg := "Transaction expired for the Kea configs update"
		log.Errorf("Problem with recovering transaction context for transaction ID %d and user ID %d", params.ID, user.ID)
		rsp := dhcp.NewUpdateKeaGlobalParametersSubmitDefault(http.StatusNotFound).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}

	// Configs are mandatory
	if len(params.Request.Configs) == 0 {
		msg := "No configs for update have been specified"
		log.Error(msg)
		rsp := dhcp.NewUpdateKeaGlobalParametersSubmitDefault(http.StatusBadRequest).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}

	var settableConfigs []config.AnnotatedEntity[*keaconfig.SettableConfig]
	for i := range params.Request.Configs {
		receivedConfig := params.Request.Configs[i]
		var settableConfig *keaconfig.SettableConfig
		switch receivedConfig.DaemonName {
		case dbmodel.DaemonNameDHCPv4:
			settableConfig = keaconfig.NewSettableDHCPv4Config()
		case dbmodel.DaemonNameDHCPv6:
			settableConfig = keaconfig.NewSettableDHCPv6Config()
		case dbmodel.DaemonNameD2:
			settableConfig = keaconfig.NewSettableD2Config()
		default:
			settableConfig = keaconfig.NewSettableCtrlAgentConfig()
		}

		partialConfig := receivedConfig.PartialConfig

		// Set DHCP parameters. We ignore errors from setting the respective
		// values because we ensure here to set appropriate values depending
		// on the DHCP server kind.
		if settableConfig.IsDHCPv4() || settableConfig.IsDHCPv6() {
			// Common DHCP parameters.
			_ = settableConfig.SetAllocator(partialConfig.Allocator)
			_ = settableConfig.SetCacheThreshold(partialConfig.CacheThreshold)
			_ = settableConfig.SetDDNSGeneratedPrefix(partialConfig.DdnsGeneratedPrefix)
			_ = settableConfig.SetDDNSOverrideClientUpdate(partialConfig.DdnsOverrideClientUpdate)
			_ = settableConfig.SetDDNSOverrideNoUpdate(partialConfig.DdnsOverrideNoUpdate)
			_ = settableConfig.SetDDNSQualifyingSuffix(partialConfig.DdnsQualifyingSuffix)
			_ = settableConfig.SetDDNSReplaceClientName(partialConfig.DdnsReplaceClientName)
			_ = settableConfig.SetDDNSSendUpdates(partialConfig.DdnsSendUpdates)
			_ = settableConfig.SetDDNSTTLPercent(partialConfig.DdnsTTLPercent)
			_ = settableConfig.SetDDNSUpdateOnRenew(partialConfig.DdnsUpdateOnRenew)
			_ = settableConfig.SetDDNSUseConflictResolution(partialConfig.DdnsUseConflictResolution)
			_ = settableConfig.SetDDNSConflictResolutionMode(partialConfig.DdnsConflictResolutionMode)
			_ = settableConfig.SetDHCPDDNSEnableUpdates(partialConfig.DhcpDdnsEnableUpdates)
			_ = settableConfig.SetDHCPDDNSMaxQueueSize(partialConfig.DhcpDdnsMaxQueueSize)
			_ = settableConfig.SetDHCPDDNSNCRFormat(partialConfig.DhcpDdnsNcrFormat)
			_ = settableConfig.SetDHCPDDNSNCRProtocol(partialConfig.DhcpDdnsNcrProtocol)
			_ = settableConfig.SetDHCPDDNSSenderIP(partialConfig.DhcpDdnsSenderIP)
			_ = settableConfig.SetDHCPDDNSSenderPort(partialConfig.DhcpDdnsSenderPort)
			_ = settableConfig.SetDHCPDDNSServerIP(partialConfig.DhcpDdnsServerIP)
			_ = settableConfig.SetDHCPDDNSServerPort(partialConfig.DhcpDdnsServerPort)
			_ = settableConfig.SetELPFlushReclaimedTimerWaitTime(partialConfig.ExpiredFlushReclaimedTimerWaitTime)
			_ = settableConfig.SetELPHoldReclaimedTime(partialConfig.ExpiredHoldReclaimedTime)
			_ = settableConfig.SetELPMaxReclaimLeases(partialConfig.ExpiredMaxReclaimLeases)
			_ = settableConfig.SetELPMaxReclaimTime(partialConfig.ExpiredMaxReclaimTime)
			_ = settableConfig.SetELPReclaimTimerWaitTime(partialConfig.ExpiredReclaimTimerWaitTime)
			_ = settableConfig.SetELPUnwarnedReclaimCycles(partialConfig.ExpiredUnwarnedReclaimCycles)
			_ = settableConfig.SetEarlyGlobalReservationsLookup(partialConfig.EarlyGlobalReservationsLookup)
			_ = settableConfig.SetHostReservationIdentifiers(partialConfig.HostReservationIdentifiers)
			_ = settableConfig.SetReservationsGlobal(partialConfig.ReservationsGlobal)
			_ = settableConfig.SetReservationsInSubnet(partialConfig.ReservationsInSubnet)
			_ = settableConfig.SetReservationsOutOfPool(partialConfig.ReservationsOutOfPool)
			_ = settableConfig.SetValidLifetime(partialConfig.ValidLifetime)

			options, err := r.flattenDHCPOptions("", partialConfig.Options, 0)
			if err != nil {
				msg := fmt.Sprintf("Problem with flattening DHCP options: %s", err)
				log.WithError(err).Error(msg)
				rsp := dhcp.NewUpdateKeaGlobalParametersSubmitDefault(http.StatusBadRequest).WithPayload(&models.APIError{
					Message: &msg,
				})
				return rsp
			}

			singleOptions := make([]keaconfig.SingleOptionData, 0, len(options))
			for _, option := range options {
				singleOption, err := keaconfig.CreateSingleOptionData(
					receivedConfig.DaemonID,
					r.DHCPOptionDefinitionLookup,
					option,
				)
				if err != nil {
					msg := fmt.Sprintf(
						"Problem with creating Kea representation of the DHCP option (code: %d, space: %s)",
						option.Code,
						option.Space,
					)
					log.WithError(err).Error(msg)

					// Include the exact error message in the API response.
					// It is safe because the error contains only the field
					// parser results and doesn't reveal any internal issues.
					msg = fmt.Sprintf("%s: %v", msg, err)

					rsp := dhcp.NewUpdateKeaGlobalParametersSubmitDefault(http.StatusBadRequest).WithPayload(&models.APIError{
						Message: &msg,
					})
					return rsp
				}
				singleOptions = append(singleOptions, *singleOption)
			}
			_ = settableConfig.SetDHCPOptions(singleOptions)

			if settableConfig.IsDHCPv4() {
				// DHCPv4 specific parameters.
				_ = settableConfig.SetAuthoritative(partialConfig.Authoritative)
				_ = settableConfig.SetEchoClientID(partialConfig.EchoClientID)
			} else {
				// DHCPv6 specific parameters.
				_ = settableConfig.SetPDAllocator(partialConfig.PdAllocator)
			}
		}
		settableConfigs = append(settableConfigs, *config.NewAnnotatedEntity(receivedConfig.DaemonID, settableConfig))
	}
	var err error
	cctx, err = r.ConfigManager.GetKeaModule().ApplyGlobalParametersUpdate(cctx, settableConfigs)
	if err != nil {
		var invalidConfigs *config.InvalidConfigsError
		switch {
		case errors.As(err, &invalidConfigs):
			// Invalid configs applied.
			msg := "Problem with applying Kea global parameters because invalid set of configurations have been specified"
			log.WithError(err).Error(msg)
			rsp := dhcp.NewUpdateKeaGlobalParametersSubmitDefault(http.StatusBadRequest).WithPayload(&models.APIError{
				Message: &msg,
			})
			return rsp
		default:
			// Other error.
			msg := "Problem with applying Kea global parameters"
			log.WithError(err).Error(msg)
			rsp := dhcp.NewUpdateKeaGlobalParametersSubmitDefault(http.StatusInternalServerError).WithPayload(&models.APIError{
				Message: &msg,
			})
			return rsp
		}
	}
	// Send the commands to Kea servers.
	cctx, err = r.ConfigManager.Commit(cctx)
	if err != nil {
		msg := fmt.Sprintf("Problem with committing Kea config: %s", err)
		log.WithError(err).Error(msg)
		rsp := dhcp.NewUpdateKeaGlobalParametersSubmitDefault(http.StatusConflict).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}

	// Everything ok. Cleanup and send OK to the client.
	r.ConfigManager.Done(cctx)
	rsp := dhcp.NewUpdateKeaGlobalParametersSubmitOK()
	return rsp
}

// Implements the DELETE call to cancel updating Kea global parameters (kea-global-parameters/transaction/{id}).
// It removes the specified transaction from the config manager, if the transaction exists.
func (r *RestAPI) UpdateKeaGlobalParametersDelete(ctx context.Context, params dhcp.UpdateKeaGlobalParametersDeleteParams) middleware.Responder {
	// Retrieve the context from the config manager.
	_, user := r.SessionManager.Logged(ctx)
	cctx, _ := r.ConfigManager.RecoverContext(params.ID, int64(user.ID))
	if cctx == nil {
		msg := "Transaction expired for updating Kea global parameters"
		log.Errorf("Problem with recovering transaction context for transaction ID %d and user ID %d", params.ID, user.ID)
		rsp := dhcp.NewUpdateKeaGlobalParametersDeleteDefault(http.StatusNotFound).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}
	r.ConfigManager.Done(cctx)
	rsp := dhcp.NewUpdateKeaGlobalParametersDeleteOK()
	return rsp
}
