package restservice

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/asaskevich/govalidator"
	"github.com/go-openapi/runtime/middleware"
	"github.com/go-openapi/strfmt"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"isc.org/stork"
	keaconfig "isc.org/stork/daemoncfg/kea"
	"isc.org/stork/daemondata/bind9stats"
	"isc.org/stork/datamodel/daemonname"
	"isc.org/stork/pki"
	"isc.org/stork/server/certs"
	"isc.org/stork/server/daemons"
	"isc.org/stork/server/daemons/kea"
	dbops "isc.org/stork/server/database"
	dbmodel "isc.org/stork/server/database/model"
	"isc.org/stork/server/dumper"
	"isc.org/stork/server/gen/models"
	dhcp "isc.org/stork/server/gen/restapi/operations/d_h_c_p"
	"isc.org/stork/server/gen/restapi/operations/general"
	"isc.org/stork/server/gen/restapi/operations/services"
	storkutil "isc.org/stork/util"
)

// Get version of Stork Server.
func (r *RestAPI) GetVersion(ctx context.Context, params general.GetVersionParams) middleware.Responder {
	bd := stork.BuildDate
	v := stork.Version
	ver := models.Version{
		Date:    &bd,
		Version: &v,
	}
	return general.NewGetVersionOK().WithPayload(&ver)
}

// Tries to send HTTP GET to STORK_REST_VERSIONS_URL to retrieve versions metadata file containing information about current ISC software versions.
// If the response to the HTTP request is successful, it tries to unmarshal received data.
// If it succeeds, pointer to the DaemonsVersions is returned. Non-nil error is returned in case of any fail.
func (r *RestAPI) getOnlineVersionsJSON() (*models.AppsVersions, error) {
	url := r.Settings.VersionsURL
	accept := "application/json"
	userAgent := fmt.Sprintf("ISC Stork / %s built on %s", stork.Version, stork.BuildDate)

	req, err := http.NewRequestWithContext(context.Background(), "GET", url, nil)
	if err != nil {
		err = errors.Wrapf(err, "could not create HTTP GET request to %s", url)
		return nil, err
	}

	req.Header.Add("Accept", accept)
	req.Header.Add("User-Agent", userAgent)
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		err = errors.Wrapf(err, "problem sending HTTP GET request to %s", url)
		return nil, err
	}

	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		err = errors.Wrapf(err, "problem reading received online versions metadata file response body")
		return nil, err
	}
	return unmarshalVersionsJSONData(&body, models.VersionsDataSourceOnline)
}

// Tries to read versions.json local file containing information about current ISC software versions
// and then it tries to unmarshal read data.
// If it succeeds, pointer to the AppsVersions is returned. Non-nil error is returned in case of any fail.
func getOfflineVersionsJSON() (*models.AppsVersions, error) {
	// Find the location of the JSON file with software versions metadata.
	searchPaths := []string{}

	// Path to the JSON file with software versions metadata is computed relative to the executable.
	var (
		executable string
		err        error
	)
	if executable, err = os.Executable(); err == nil {
		if executable, err = filepath.EvalSymlinks(executable); err == nil {
			exDir := filepath.Dir(executable)
			searchPaths = append(searchPaths, filepath.Join(exDir, "..", "..", "etc", "stork", "versions.json")) // relative path when Stork was installed from package or with 'rake install' task
			searchPaths = append(searchPaths, filepath.Join(exDir, "..", "..", "..", "etc", "versions.json"))    // relative path when running Stork server with 'rake run' task - typical for DEV
		}
	}
	if err != nil {
		return nil, errors.Wrapf(err, "problem determining the executable path to locate the JSON file with software versions metadata")
	}

	jsonFile := storkutil.GetFirstExistingPathOrDefault(VersionsJSONPath, searchPaths...)

	// Open JSON file.
	file, err := os.Open(jsonFile)
	if err != nil {
		err = errors.Wrapf(err, "problem opening the JSON file with software versions metadata")
		return nil, err
	}
	defer file.Close()

	// Read the contents of the file.
	bytes, err := io.ReadAll(file)
	if err != nil {
		err = errors.Wrapf(err, "problem reading the contents of the JSON file with software versions metadata")
		return nil, err
	}
	return unmarshalVersionsJSONData(&bytes, models.VersionsDataSourceOffline)
}

// Deserializes bytes data into ReportAppsVersions struct, converts and returns the data in REST API format.
func unmarshalVersionsJSONData(bytes *[]byte, mode models.VersionsDataSource) (*models.AppsVersions, error) {
	// Unmarshal the JSON to custom struct.
	s := ReportAppsVersions{}
	err := json.Unmarshal(*bytes, &s)
	if err != nil {
		err = errors.Wrapf(err, "problem unmarshalling contents of the %s JSON file with software versions metadata", mode)
		return nil, err
	}

	bind9, err := daemonVersionMetadataToRestAPI(*s.Bind9)
	if err != nil {
		err = errors.WithMessagef(err, "problem converting BIND 9 data from the %s JSON file with software versions metadata", mode)
		return nil, err
	}
	kea, err := daemonVersionMetadataToRestAPI(*s.Kea)
	if err != nil {
		err = errors.WithMessagef(err, "problem converting Kea data from the %s JSON file with software versions metadata", mode)
		return nil, err
	}
	stork, err := daemonVersionMetadataToRestAPI(*s.Stork)
	if err != nil {
		err = errors.WithMessagef(err, "problem converting Stork data from the %s JSON file with software versions metadata", mode)
		return nil, err
	}

	parsedTime, err := time.Parse("2006-01-02", *s.Date)
	if err != nil {
		err = errors.Wrapf(err, "problem parsing date from the %s JSON file with software versions metadata", mode)
		return nil, err
	}
	dataDate := strfmt.Date(parsedTime)

	// Prepare REST API response.
	appsVersions := models.AppsVersions{
		Date:       &dataDate,
		Bind9:      bind9,
		Kea:        kea,
		Stork:      stork,
		DataSource: mode,
	}
	return &appsVersions, nil
}

// Get information about current ISC software versions.
func (r *RestAPI) GetSoftwareVersions(ctx context.Context, params general.GetSoftwareVersionsParams) middleware.Responder {
	onlineModeEnabled, err := dbmodel.GetSettingBool(r.DB, "enable_online_software_versions")
	if err != nil {
		log.WithError(err).Error("Problem reading boolean setting enable_online_software_versions")
		onlineModeEnabled = false
	}
	if onlineModeEnabled {
		appsVersions, err := r.getOnlineVersionsJSON()
		if err == nil {
			return general.NewGetSoftwareVersionsOK().WithPayload(appsVersions)
		}
		log.WithError(err).Error("Problem processing online versions metadata file data; falling back to offline mode")
	} else {
		log.Warn("online mode of software version checking disabled")
	}

	appsVersions, err := getOfflineVersionsJSON()
	if err == nil {
		return general.NewGetSoftwareVersionsOK().WithPayload(appsVersions)
	}
	log.WithError(err).Error("Problem processing offline versions.json data")
	errMsg := "Error parsing the contents of the JSON file with software versions metadata"
	rsp := general.NewGetSoftwareVersionsDefault(http.StatusInternalServerError).WithPayload(&models.APIError{
		Message: &errMsg,
	})
	return rsp
}

// Convert db machine to rest structure.
func (r *RestAPI) machineToRestAPI(dbMachine dbmodel.Machine) *models.Machine {
	daemons := []*models.AnyDaemon{}

	for _, dbDaemon := range dbMachine.Daemons {
		daemon := r.daemonToRestAPI(dbDaemon)
		daemons = append(daemons, daemon)
	}

	m := models.Machine{
		ID:                   dbMachine.ID,
		Address:              &dbMachine.Address,
		AgentPort:            dbMachine.AgentPort,
		Authorized:           dbMachine.Authorized,
		AgentToken:           dbMachine.AgentToken,
		AgentVersion:         dbMachine.State.AgentVersion,
		Cpus:                 dbMachine.State.Cpus,
		CpusLoad:             dbMachine.State.CpusLoad,
		Memory:               dbMachine.State.Memory,
		Hostname:             dbMachine.State.Hostname,
		Uptime:               dbMachine.State.Uptime,
		UsedMemory:           dbMachine.State.UsedMemory,
		Os:                   dbMachine.State.Os,
		Platform:             dbMachine.State.Platform,
		PlatformFamily:       dbMachine.State.PlatformFamily,
		PlatformVersion:      dbMachine.State.PlatformVersion,
		KernelVersion:        dbMachine.State.KernelVersion,
		KernelArch:           dbMachine.State.KernelArch,
		VirtualizationSystem: dbMachine.State.VirtualizationSystem,
		VirtualizationRole:   dbMachine.State.VirtualizationRole,
		HostID:               dbMachine.State.HostID,
		LastVisitedAt:        convertToOptionalDatetime(dbMachine.LastVisitedAt),
		Error:                dbMachine.Error,
		Daemons:              daemons,
	}
	return &m
}

// Convert db machine to minimalistic rest structure.
func (r *RestAPI) simpleMachineToRestAPI(dbMachine dbmodel.Machine) *models.SimpleMachine {
	daemons := []*models.SimpleDaemon{}
	for _, dbDaemon := range dbMachine.Daemons {
		daemon := r.simpleDaemonToRestAPI(dbDaemon)
		daemons = append(daemons, daemon)
	}

	// Return only minimal information about the machine and add software versions
	// data for the Daemons.
	m := models.SimpleMachine{
		ID:           dbMachine.ID,
		Address:      dbMachine.Address,
		AgentPort:    dbMachine.AgentPort,
		AgentVersion: dbMachine.State.AgentVersion,
		Hostname:     dbMachine.State.Hostname,
		Daemons:      daemons,
	}
	return &m
}

// Get runtime state of indicated machine.
func (r *RestAPI) GetMachineState(ctx context.Context, params services.GetMachineStateParams) middleware.Responder {
	dbMachine, err := dbmodel.GetMachineByID(r.DB, params.ID)
	if err != nil {
		msg := fmt.Sprintf("Cannot get machine with ID %d from db", params.ID)
		log.WithError(err).Error(msg)
		rsp := services.NewGetMachineStateDefault(http.StatusInternalServerError).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}
	if dbMachine == nil {
		msg := fmt.Sprintf("Cannot find machine with ID %d", params.ID)
		rsp := services.NewGetMachineStateDefault(http.StatusNotFound).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}

	errStr := daemons.UpdateMachineAndDaemonsState(ctx, r.DB, dbMachine, r.Agents, r.EventCenter, r.ReviewDispatcher, r.DHCPOptionDefinitionLookup)
	if errStr != "" {
		rsp := services.NewGetMachineStateDefault(http.StatusInternalServerError).WithPayload(&models.APIError{
			Message: &errStr,
		})
		return rsp
	}

	m := r.machineToRestAPI(*dbMachine)
	rsp := services.NewGetMachineStateOK().WithPayload(m)

	return rsp
}

// Get machines from database based on params and convert them to rest structures.
func (r *RestAPI) getMachines(offset, limit int64, filterText *string, authorized *bool, sortField string, sortDir dbmodel.SortDirEnum) (*models.Machines, error) {
	dbMachines, total, err := dbmodel.GetMachinesByPage(r.DB, offset, limit, filterText, authorized, sortField, sortDir)
	if err != nil {
		return nil, err
	}
	machines := &models.Machines{
		Total: total,
		Items: []*models.Machine{},
	}

	for _, dbM := range dbMachines {
		m := r.machineToRestAPI(dbM)
		machines.Items = append(machines.Items, m)
	}

	return machines, nil
}

// Get machines where Stork Agent is running.
func (r *RestAPI) GetMachines(ctx context.Context, params services.GetMachinesParams) middleware.Responder {
	var start int64
	if params.Start != nil {
		start = *params.Start
	}

	var limit int64 = 10
	if params.Limit != nil {
		limit = *params.Limit
	}

	text := ""
	if params.Text != nil {
		text = *params.Text
	}

	sortField := ""
	if params.SortField != nil {
		sortField = *params.SortField
	}

	sortDir := dbmodel.SortDirAny
	if params.SortDir != nil {
		sortDir = dbmodel.SortDirEnum(*params.SortDir)
	}

	log.WithFields(log.Fields{
		"start":     start,
		"limit":     limit,
		"text":      text,
		"sortField": sortField,
		"sortDir":   sortDir,
	}).Info("query machines")

	machines, err := r.getMachines(start, limit, params.Text, params.Authorized, sortField, sortDir)
	if err != nil {
		msg := "Cannot get machines from db"
		log.WithError(err).Error(msg)
		rsp := services.NewGetMachinesDefault(http.StatusInternalServerError).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}
	rsp := services.NewGetMachinesOK().WithPayload(machines)
	return rsp
}

// Returns a list of all authorized machines' ids and addresses/names. A client calls this
// function to create a drop down list with available machines or to validate user's input
// against machines' names available in the system.
func (r *RestAPI) GetMachinesDirectory(ctx context.Context, params services.GetMachinesDirectoryParams) middleware.Responder {
	authorized := true
	dbMachines, err := dbmodel.GetAllMachinesNoRelations(r.DB, &authorized)
	if err != nil {
		msg := "Cannot get machines directory from the database"
		log.WithError(err).Error(msg)
		rsp := services.NewGetMachinesDefault(http.StatusInternalServerError).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}

	machines := &models.Machines{
		Total: int64(len(dbMachines)),
	}
	for i := range dbMachines {
		machine := models.Machine{
			ID:      dbMachines[i].ID,
			Address: &dbMachines[i].Address,
		}
		machines.Items = append(machines.Items, &machine)
	}

	rsp := services.NewGetMachinesDirectoryOK().WithPayload(machines)
	return rsp
}

// Returns a list of all authorized machines' ids and daemons versions.
func (r *RestAPI) GetMachinesVersions(ctx context.Context, params services.GetMachinesVersionsParams) middleware.Responder {
	authorized := true
	dbMachines, err := dbmodel.GetAllMachinesWithRelations(r.DB, &authorized, dbmodel.MachineRelationDaemonAccessPoints)
	if err != nil {
		msg := "Cannot get machines daemons versions from the database"
		log.WithError(err).Error(msg)
		rsp := services.NewGetMachinesVersionsDefault(http.StatusInternalServerError).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}

	machines := &models.SimpleMachines{
		Total: int64(len(dbMachines)),
	}
	for i := range dbMachines {
		machine := r.simpleMachineToRestAPI(dbMachines[i])
		machines.Items = append(machines.Items, machine)
	}

	rsp := services.NewGetMachinesVersionsOK().WithPayload(machines)
	return rsp
}

// Return the number of the unauthorized machines.
func (r *RestAPI) GetUnauthorizedMachinesCount(ctx context.Context, params services.GetUnauthorizedMachinesCountParams) middleware.Responder {
	count, err := dbmodel.GetUnauthorizedMachinesCount(r.DB)
	if err != nil {
		msg := "Cannot get a number of the unauthorized machines from the database"
		log.WithError(err).Error(msg)
		rsp := services.NewGetUnauthorizedMachinesCountDefault(http.StatusInternalServerError).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}
	rsp := services.NewGetUnauthorizedMachinesCountOK().WithPayload(int64(count))
	return rsp
}

// Check server token provided by user in agent registration
// procedure. If it is empty and it is allowed to be empty then it is
// accepted (true is returned). Otherwise provided token is compared
// with the one from server's database. If they do not match then
// false is returned. The second returned thing is HTTP error code,
// the last is error message.
func (r *RestAPI) checkServerToken(serverToken string, allowEmpty bool) (bool, int, string) {
	// if token can be empty and it is empty then return that machine is not authorized
	// and no error
	if allowEmpty && serverToken == "" {
		return false, 0, ""
	}
	dbServerToken, err := dbmodel.GetSecret(r.DB, dbmodel.SecretServerToken)
	if err != nil {
		msg := "Cannot retrieve server token from database"
		log.WithError(err).Error(msg)
		return false, http.StatusInternalServerError, msg
	}
	if dbServerToken == nil {
		msg := "Server internal problem - server token is empty"
		log.Error(msg)
		return false, http.StatusInternalServerError, msg
	}
	dbServerTokenStr := string(dbServerToken)

	// if provided token does not match then machine is not authorized and return error
	if serverToken != dbServerTokenStr {
		msg := "Provided server token is wrong"
		log.Error(msg)
		return false, http.StatusBadRequest, msg
	}
	// tokens match so machine is authorized
	return true, 0, ""
}

// Get one machine by ID where Stork Agent is running.
func (r *RestAPI) GetMachine(ctx context.Context, params services.GetMachineParams) middleware.Responder {
	dbMachine, err := dbmodel.GetMachineByID(r.DB, params.ID)
	if err != nil {
		msg := fmt.Sprintf("Cannot get machine with ID %d from db", params.ID)
		log.WithError(err).Error(msg)
		rsp := services.NewGetMachineDefault(http.StatusInternalServerError).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}
	if dbMachine == nil {
		msg := fmt.Sprintf("Cannot find machine with ID %d", params.ID)
		rsp := services.NewGetMachineDefault(http.StatusNotFound).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}
	m := r.machineToRestAPI(*dbMachine)
	rsp := services.NewGetMachineOK().WithPayload(m)
	return rsp
}

// Add a machine where Stork Agent is running.
func (r *RestAPI) CreateMachine(ctx context.Context, params services.CreateMachineParams) middleware.Responder {
	if params.Machine == nil || params.Machine.Address == nil {
		log.Warnf("Cannot create machine: missing parameters")
		msg := "Missing parameters"
		rsp := services.NewCreateMachineDefault(http.StatusBadRequest).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}
	addr := *params.Machine.Address
	if !govalidator.IsHost(*params.Machine.Address) {
		log.Warnf("Problem parsing address %s", addr)
		msg := "Cannot parse address"
		rsp := services.NewCreateMachineDefault(http.StatusBadRequest).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}
	if params.Machine.AgentPort <= 0 || params.Machine.AgentPort > 65535 {
		log.Warnf("Bad agent port %d", params.Machine.AgentPort)
		msg := "Bad port"
		rsp := services.NewCreateMachineDefault(http.StatusBadRequest).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}
	if params.Machine.AgentCSR == nil {
		msg := "Agent CSR cannot be empty"
		log.Warn(msg)
		rsp := services.NewCreateMachineDefault(http.StatusBadRequest).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}

	if *params.Machine.AgentToken == "" {
		msg := "Agent token cannot be empty"
		log.Warn(msg)
		rsp := services.NewCreateMachineDefault(http.StatusBadRequest).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}

	dbMachine, err := dbmodel.GetMachineByAddressAndAgentPort(r.DB, addr, params.Machine.AgentPort)
	if err != nil {
		msg := fmt.Sprintf("Problem finding machine %s:%d in database", addr, params.Machine.AgentPort)
		log.Warn(msg)
		rsp := services.NewCreateMachineDefault(http.StatusInternalServerError).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}

	rootCertPEM, err := dbmodel.GetSecret(r.DB, dbmodel.SecretCACert)
	if err != nil {
		msg := "Problem loading server CA cert"
		log.WithError(err).Error(msg)
		rsp := services.NewCreateMachineDefault(http.StatusInternalServerError).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}

	serverCertPEM, err := dbmodel.GetSecret(r.DB, dbmodel.SecretServerCert)
	if err != nil {
		msg := "Problem loading server cert"
		log.WithError(err).Error(msg)
		rsp := services.NewCreateMachineDefault(http.StatusInternalServerError).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}
	serverCertFingerprint, err := pki.CalculateFingerprintFromPEM(serverCertPEM)
	if err != nil {
		msg := "Problem calculating fingerprint of server cert"
		log.WithError(err).Error(msg)
		rsp := services.NewCreateMachineDefault(http.StatusInternalServerError).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}

	machineAuthorized := false

	// Check if a machine is already registered with a provided agent token
	if dbMachine != nil && dbMachine.AgentToken == *params.Machine.AgentToken {
		// Check if the CA cert has been updated since the machine was last
		// registered. If the CA was updated, the agent cert must be
		// re-generated but we want to keep the authorization status.
		var caCertFingerprint [32]byte
		if params.Machine.CaCertFingerprint != "" {
			// The legacy machines that don't send the serial number in the
			// registration request will receive a new agent certificate on
			// every run. This shouldn't be a big problem as long as the
			// authorization status is preserved.
			rawFingerprint := storkutil.HexToBytes(params.Machine.CaCertFingerprint)
			if len(rawFingerprint) == 32 {
				caCertFingerprint = [32]byte(rawFingerprint)
			} else {
				rsp := services.NewCreateMachineDefault(http.StatusBadRequest).WithPayload(&models.APIError{
					Message: storkutil.Ptr("Invalid CA cert fingerprint"),
				})
				return rsp
			}
		}

		rootCertFingerprint, err := pki.CalculateFingerprintFromPEM(rootCertPEM)
		if err != nil {
			msg := "Problem calculating fingerprint of server CA cert"
			log.WithError(err).Error(msg)
			rsp := services.NewCreateMachineDefault(http.StatusInternalServerError).WithPayload(&models.APIError{
				Message: &msg,
			})
			return rsp
		}

		if rootCertFingerprint == caCertFingerprint {
			link := fmt.Sprintf("/machines/%d", dbMachine.ID)
			rsp := services.NewCreateMachineConflict().
				WithLocation(link).
				WithPayload(&models.ExistingMachineResp{
					ID:                    dbMachine.ID,
					ServerCertFingerprint: storkutil.BytesToHex(serverCertFingerprint[:]),
				})
			return rsp
		}

		// Preserve the current authorization status because the host and agent
		// token are correct.
		machineAuthorized = dbMachine.Authorized
	}

	if !machineAuthorized {
		// check server token
		var (
			httpRspCode int
			rspMsg      string
		)
		machineAuthorized, httpRspCode, rspMsg = r.checkServerToken(params.Machine.ServerToken, true)
		if httpRspCode != 0 {
			rsp := services.NewCreateMachineDefault(httpRspCode).WithPayload(&models.APIError{
				Message: &rspMsg,
			})
			return rsp
		}
	}

	// sign agent cert
	agentCSR := []byte(*params.Machine.AgentCSR)
	certSerialNumber, err := dbmodel.GetNewCertSerialNumber(r.DB)
	if err != nil {
		msg := "Problem generating serial number for cert"
		log.WithError(err).Error(msg)
		rsp := services.NewCreateMachineDefault(http.StatusInternalServerError).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}
	// TODO: consider providing a database query which returns multiple
	// secrets to avoid multiple database roundtrips.
	rootKeyPEM, err := dbmodel.GetSecret(r.DB, dbmodel.SecretCAKey)
	if err != nil {
		msg := "Problem loading server CA private key"
		log.WithError(err).Error(msg)
		rsp := services.NewCreateMachineDefault(http.StatusInternalServerError).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}

	agentCertPEM, agentCertFingerprint, paramsErr, innerErr := pki.SignCert(agentCSR, certSerialNumber, rootCertPEM, rootKeyPEM)
	if paramsErr != nil {
		msg := "Problem with agent CSR"
		log.WithError(paramsErr).Error(msg)
		rsp := services.NewCreateMachineDefault(http.StatusBadRequest).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}
	if innerErr != nil {
		msg := "Problem signing agent CSR"
		log.WithError(innerErr).Error(msg)
		rsp := services.NewCreateMachineDefault(http.StatusInternalServerError).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}

	// We can't create new machine and pull machines' states at the same time. This may
	// put heavy workload on the server and it may also result in conflicts. Temporarily
	// disable the puller while the new machine is being added.
	if r.Pullers != nil && r.Pullers.StatePuller != nil {
		r.Pullers.StatePuller.Pause()
		defer r.Pullers.StatePuller.Unpause()
	}

	if dbMachine == nil {
		if r.EndpointControl.IsDisabled(EndpointOpCreateNewMachine) {
			log.Info("Machine registration prevented because it is administratively disabled")
			rsp := services.NewCreateMachineDefault(http.StatusForbidden).WithPayload(&models.APIError{
				Message: storkutil.Ptr("Machine registration is administratively disabled"),
			})
			return rsp
		}

		dbMachine = &dbmodel.Machine{
			Address:         addr,
			AgentPort:       params.Machine.AgentPort,
			AgentToken:      *params.Machine.AgentToken,
			CertFingerprint: agentCertFingerprint,
			Authorized:      machineAuthorized,
		}
		err = dbmodel.AddMachine(r.DB, dbMachine)
		if err != nil {
			msg := fmt.Sprintf("Cannot store machine %s", addr)
			log.WithError(err).Error(msg)
			rsp := services.NewCreateMachineDefault(http.StatusInternalServerError).WithPayload(&models.APIError{
				Message: &msg,
			})
			return rsp
		}
		r.EventCenter.AddInfoEvent("added {machine}", dbmodel.SSERegistration, dbMachine)
	} else {
		dbMachine.AgentToken = *params.Machine.AgentToken
		dbMachine.CertFingerprint = agentCertFingerprint
		dbMachine.Authorized = machineAuthorized
		err = dbmodel.UpdateMachine(r.DB, dbMachine)
		if err != nil {
			msg := fmt.Sprintf("Cannot update machine %s in database", addr)
			log.WithError(err).Error(msg)
			rsp := services.NewCreateMachineDefault(http.StatusInternalServerError).WithPayload(&models.APIError{
				Message: &msg,
			})
			return rsp
		}
		r.EventCenter.AddInfoEvent("re-registered {machine}", dbMachine)
	}

	m := &models.NewMachineResp{
		ID:                    dbMachine.ID,
		ServerCACert:          string(rootCertPEM),
		AgentCert:             string(agentCertPEM),
		ServerCertFingerprint: storkutil.BytesToHex(serverCertFingerprint[:]),
	}
	rsp := services.NewCreateMachineOK().WithPayload(m)

	return rsp
}

// Ping given machine, i.e. check connectivity.
func (r *RestAPI) PingMachine(ctx context.Context, params services.PingMachineParams) middleware.Responder {
	// find machine in db
	dbMachine, err := dbmodel.GetMachineByIDWithRelations(r.DB, params.ID)
	if err != nil {
		msg := fmt.Sprintf("Cannot get machine with ID %d from db", params.ID)
		log.WithError(err).Error(msg)
		rsp := services.NewPingMachineDefault(http.StatusInternalServerError).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}
	if dbMachine == nil {
		msg := fmt.Sprintf("Cannot find machine with ID %d", params.ID)
		rsp := services.NewPingMachineDefault(http.StatusNotFound).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}

	// check server token
	_, httpRspCode, rspMsg := r.checkServerToken(params.Ping.ServerToken, false)
	if httpRspCode != 0 {
		rsp := services.NewPingMachineDefault(httpRspCode).WithPayload(&models.APIError{
			Message: &rspMsg,
		})
		return rsp
	}

	// check agent token
	if params.Ping.AgentToken != dbMachine.AgentToken {
		msg := "Provided agent token is wrong"
		log.Error(msg)
		rsp := services.NewPingMachineDefault(http.StatusBadRequest).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}

	// ping machine
	ctx2, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	err = r.Agents.Ping(ctx2, dbMachine)
	if err != nil {
		msg := "Cannot ping machine"
		log.WithError(err).Error(msg)
		rsp := services.NewPingMachineDefault(http.StatusInternalServerError).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}

	// Communication with an agent established, so get machine's state.
	errStr := daemons.UpdateMachineAndDaemonsState(ctx2, r.DB, dbMachine, r.Agents, r.EventCenter, r.ReviewDispatcher, r.DHCPOptionDefinitionLookup)
	if errStr != "" {
		rsp := services.NewPingMachineDefault(http.StatusInternalServerError).WithPayload(&models.APIError{
			Message: &errStr,
		})
		return rsp
	}

	rsp := services.NewPingMachineOK()

	return rsp
}

// Get one machine by ID where Stork Agent is running.
func (r *RestAPI) UpdateMachine(ctx context.Context, params services.UpdateMachineParams) middleware.Responder {
	if params.Machine == nil || params.Machine.Address == nil {
		log.Warnf("Cannot update machine: missing parameters")
		msg := "Missing parameters"
		rsp := services.NewUpdateMachineDefault(http.StatusBadRequest).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}
	addr := *params.Machine.Address
	if !govalidator.IsHost(*params.Machine.Address) {
		log.Warnf("Problem parsing address %s", addr)
		msg := "Cannot parse address"
		rsp := services.NewUpdateMachineDefault(http.StatusBadRequest).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}
	if params.Machine.AgentPort <= 0 || params.Machine.AgentPort > 65535 {
		log.Warnf("Bad agent port %d", params.Machine.AgentPort)
		msg := "Bad port"
		rsp := services.NewUpdateMachineDefault(http.StatusBadRequest).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}

	dbMachine, err := dbmodel.GetMachineByID(r.DB, params.ID)
	if err != nil {
		msg := fmt.Sprintf("Cannot get machine with ID %d from db", params.ID)
		log.WithError(err).Error(msg)
		rsp := services.NewUpdateMachineDefault(http.StatusInternalServerError).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}
	if dbMachine == nil {
		msg := fmt.Sprintf("Cannot find machine with ID %d", params.ID)
		rsp := services.NewUpdateMachineDefault(http.StatusNotFound).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}

	// check if there is no duplicate
	if dbMachine.Address != addr || dbMachine.AgentPort != params.Machine.AgentPort {
		dbMachine2, err := dbmodel.GetMachineByAddressAndAgentPort(r.DB, addr, params.Machine.AgentPort)
		if err == nil && dbMachine2 != nil && dbMachine2.ID != dbMachine.ID {
			msg := fmt.Sprintf("Machine with address %s:%d already exists",
				*params.Machine.Address, params.Machine.AgentPort)
			rsp := services.NewUpdateMachineDefault(http.StatusBadRequest).WithPayload(&models.APIError{
				Message: &msg,
			})
			return rsp
		}
	}

	// if machine authorization is changed then this action requires super-admin group
	if dbMachine.Authorized != params.Machine.Authorized {
		_, dbUser := r.SessionManager.Logged(ctx)
		if !dbUser.InGroup(&dbmodel.SystemGroup{ID: dbmodel.SuperAdminGroupID}) {
			msg := "User is forbidden to change machine authorization"
			rsp := services.NewUpdateMachineDefault(http.StatusForbidden).WithPayload(&models.APIError{
				Message: &msg,
			})
			return rsp
		}
	}

	// copy fields
	dbMachine.Address = addr
	dbMachine.AgentPort = params.Machine.AgentPort
	prevAuthorized := dbMachine.Authorized
	dbMachine.Authorized = params.Machine.Authorized
	_, err = r.DB.Model(dbMachine).WherePK().Update()
	if err != nil {
		log.WithError(err).Error("Cannot update machine")
		msg := fmt.Sprintf("Cannot update machine with ID %d in db", params.ID)
		rsp := services.NewUpdateMachineDefault(http.StatusInternalServerError).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}

	// as we just authorized machine so get its state now
	if !prevAuthorized && dbMachine.Authorized {
		ctx2, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
		errStr := daemons.UpdateMachineAndDaemonsState(ctx2, r.DB, dbMachine, r.Agents, r.EventCenter, r.ReviewDispatcher, r.DHCPOptionDefinitionLookup)
		if errStr != "" {
			rsp := services.NewUpdateMachineDefault(http.StatusInternalServerError).WithPayload(&models.APIError{
				Message: &errStr,
			})
			return rsp
		}
	}

	m := r.machineToRestAPI(*dbMachine)
	rsp := services.NewUpdateMachineOK().WithPayload(m)
	return rsp
}

// Get machine's server token. It is used by user during manual agent registration.
func (r *RestAPI) GetMachinesServerToken(ctx context.Context, params services.GetMachinesServerTokenParams) middleware.Responder {
	// only super-admin can get server token
	_, dbUser := r.SessionManager.Logged(ctx)
	if !dbUser.InGroup(&dbmodel.SystemGroup{ID: dbmodel.SuperAdminGroupID}) {
		msg := "User is forbidden to get server token"
		rsp := services.NewGetMachinesServerTokenDefault(http.StatusForbidden).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}

	// get server token from database
	dbServerToken, err := dbmodel.GetSecret(r.DB, dbmodel.SecretServerToken)
	if err != nil {
		msg := "Cannot retrieve server token from database"
		log.WithError(err).Error(msg)
		rsp := services.NewGetMachinesServerTokenDefault(http.StatusInternalServerError).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}
	if dbServerToken == nil {
		msg := "Server internal problem - server token is empty"
		log.Error(msg)
		rsp := services.NewGetMachinesServerTokenDefault(http.StatusInternalServerError).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}
	tokenRsp := &services.GetMachinesServerTokenOKBody{
		Token: string(dbServerToken),
	}
	rsp := services.NewGetMachinesServerTokenOK().WithPayload(tokenRsp)
	return rsp
}

// Regenerate machines server token.
func (r *RestAPI) RegenerateMachinesServerToken(ctx context.Context, params services.RegenerateMachinesServerTokenParams) middleware.Responder {
	// only super-admin can get server token
	_, dbUser := r.SessionManager.Logged(ctx)
	if !dbUser.InGroup(&dbmodel.SystemGroup{ID: dbmodel.SuperAdminGroupID}) {
		msg := "User is forbidden to generate new server token"
		rsp := services.NewGetMachinesServerTokenDefault(http.StatusForbidden).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}

	// generate new server token
	dbServerToken, err := certs.GenerateServerToken(r.DB)
	if err != nil {
		msg := "Cannot regenerate server token"
		log.WithError(err).Error(msg)
		rsp := services.NewRegenerateMachinesServerTokenDefault(http.StatusInternalServerError).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}
	tokenRsp := &services.RegenerateMachinesServerTokenOKBody{
		Token: string(dbServerToken),
	}
	rsp := services.NewRegenerateMachinesServerTokenOK().WithPayload(tokenRsp)
	return rsp
}

// Delete a machine where Stork Agent is running.
func (r *RestAPI) DeleteMachine(ctx context.Context, params services.DeleteMachineParams) middleware.Responder {
	// Only super-admin can delete a machine.
	_, dbUser := r.SessionManager.Logged(ctx)
	if !dbUser.InGroup(&dbmodel.SystemGroup{ID: dbmodel.SuperAdminGroupID}) {
		msg := "User is forbidden to delete a machine"
		rsp := services.NewDeleteMachineDefault(http.StatusForbidden).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}

	dbMachine, err := dbmodel.GetMachineByIDWithRelations(r.DB, params.ID, dbmodel.MachineRelationDaemons)
	if err == nil && dbMachine == nil {
		rsp := services.NewDeleteMachineOK()
		return rsp
	} else if err != nil {
		msg := fmt.Sprintf("Cannot delete machine %d", params.ID)
		log.WithError(err).Error(msg)
		rsp := services.NewDeleteMachineDefault(http.StatusInternalServerError).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}

	err = dbmodel.DeleteMachine(r.DB, dbMachine)
	if err != nil {
		msg := fmt.Sprintf("Cannot delete machine %d", params.ID)
		log.WithError(err).Error(msg)
		rsp := services.NewDeleteMachineDefault(http.StatusInternalServerError).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}

	r.EventCenter.AddInfoEvent("removed {machine}", dbMachine)

	rsp := services.NewDeleteMachineOK()

	return rsp
}

// Return a single machine dump archive. It is intended for easily sharing the configuration
// for diagnostic purposes. The archive contains the database dumps and some log files.
func (r *RestAPI) GetMachineDump(ctx context.Context, params services.GetMachineDumpParams) middleware.Responder {
	dump, err := dumper.DumpMachine(r.DB, r.Agents, params.ID)
	if err != nil {
		status := http.StatusInternalServerError
		statusMessage := fmt.Sprintf("Cannot dump machine %d", params.ID)
		log.WithError(err).Error(statusMessage)
		rsp := services.NewGetMachineDumpDefault(status).WithPayload(&models.APIError{
			Message: &statusMessage,
		})
		return rsp
	}

	if dump == nil {
		status := http.StatusNotFound
		statusMessage := fmt.Sprintf("Cannot find machine %d", params.ID)
		rsp := services.NewGetMachineDumpDefault(status).WithPayload(&models.APIError{
			Message: &statusMessage,
		})
		return rsp
	}

	dispositionHeaderValue := fmt.Sprintf(
		"attachment; filename=\"stork-machine-%d-dump_%s.tar.gz\"",
		params.ID,
		strings.ReplaceAll(time.Now().UTC().Format(time.RFC3339), ":", "-"),
	)

	rsp := services.
		NewGetMachineDumpOK().
		WithContentType("application/gzip").
		WithContentDisposition(dispositionHeaderValue).
		WithPayload(dump)
	return rsp
}

// Merges information carried in the Kea database configuration into the map of
// existing database configurations formatted to be sent over the REST API.
// This function is called by the getKeaStorages function.
func mergeKeaDatabase(keaDatabase *keaconfig.Database, dataType string, existingDatabases *map[string]*models.KeaDaemonDatabase) {
	id := fmt.Sprintf("%s:%s@%s", keaDatabase.Type, keaDatabase.Name, keaDatabase.Host)
	if existingDatabase, ok := (*existingDatabases)[id]; ok {
		existingDatabase.DataTypes = append(existingDatabase.DataTypes, dataType)
	} else {
		newDatabase := &models.KeaDaemonDatabase{
			BackendType: keaDatabase.Type,
			Database:    keaDatabase.Name,
			Host:        keaDatabase.Host,
			DataTypes:   []string{dataType},
		}
		(*existingDatabases)[id] = newDatabase
	}
}

// Parses Kea configuration, discovers all configured backends and returns them.
// They are returned in two structures, one containing files and one containing
// information about database connections in use. The first structure contains
// files used by Memfile lease database backend and Forensic Logging hooks library.
// If neither of them is used, this structure is empty.
func getKeaStorages(config keaconfig.DatabaseConfig) ([]*models.File, []*models.KeaDaemonDatabase) {
	databases := []*models.KeaDaemonDatabase{}
	files := []*models.File{}
	foundDatabases := make(map[string]*models.KeaDaemonDatabase)
	keaDatabases := config.GetAllDatabases()
	// Leases.
	if keaDatabases.Lease != nil {
		if keaDatabases.Lease.Type == "memfile" {
			// Storing leases in a lease file.
			persist := true
			if keaDatabases.Lease.Persist != nil && !*keaDatabases.Lease.Persist {
				persist = false
			}
			files = append(files, &models.File{
				Filename: keaDatabases.Lease.Name,
				Filetype: "Lease file",
				// This is in every file response, but only makes sense for lease files.
				// Limitations of the subset of JSON Schema used by OpenAPI 2.0 prevent
				// me from writing an `anyOf` schema that could exclude it from files
				// where it is not relevant.
				Persist: persist,
			})
		} else {
			// Storing leases in a database.
			mergeKeaDatabase(keaDatabases.Lease, "Leases", &foundDatabases)
		}
	}
	// Host reservations.
	for i := range keaDatabases.Hosts {
		mergeKeaDatabase(&keaDatabases.Hosts[i], "Host Reservations", &foundDatabases)
	}
	// Config backend.
	for i := range keaDatabases.Config {
		mergeKeaDatabase(&keaDatabases.Config[i], "Config Backend", &foundDatabases)
	}
	// Forensic logging.
	if keaDatabases.Forensic != nil {
		// If the type is empty (or missing) or equals to "logfile", the logs
		// are saved to a file. If it is "syslog", the logs are sent to syslog.
		// If it is anything else, the logs are saved to a database.
		// See src/lib/dhcpsrv/legal_log_mgr.cc@LegalLogMgr::parseConfig in Kea
		// source code for more details.
		switch keaDatabases.Forensic.Type {
		case "", "logfile", "syslog":
			// Not logging to a database.
			files = append(files, &models.File{
				Filename: keaDatabases.Forensic.Path,
				Filetype: "Forensic Logging",
				Persist:  keaDatabases.Forensic.Type != "syslog",
			})
		default:
			// Logging to a database.
			mergeKeaDatabase(keaDatabases.Forensic, "Forensic Logging", &foundDatabases)
		}
	}
	// Return found databases as a list.
	for _, d := range foundDatabases {
		databases = append(databases, d)
	}
	return files, databases
}

// Converts the daemon to REST API format.
func (r *RestAPI) daemonToRestAPI(dbDaemon *dbmodel.Daemon) *models.AnyDaemon {
	var accessPoints []*models.AccessPoint
	for _, point := range dbDaemon.AccessPoints {
		accessPoints = append(accessPoints, &models.AccessPoint{
			Type:     string(point.Type),
			Address:  point.Address,
			Port:     point.Port,
			Protocol: string(point.Protocol),
		})
	}

	daemon := &models.AnyDaemon{
		Daemon: models.Daemon{
			AccessPoints:    accessPoints,
			Active:          dbDaemon.Active,
			ExtendedVersion: dbDaemon.ExtendedVersion,
			ID:              dbDaemon.ID,
			MachineID:       dbDaemon.MachineID,
			Monitored:       dbDaemon.Monitored,
			Name:            string(dbDaemon.Name),
			Label:           dbDaemon.GetLabel(),
			Pid:             int64(dbDaemon.Pid),
			ReloadedAt:      convertToOptionalDatetime(dbDaemon.ReloadedAt),
			Uptime:          dbDaemon.Uptime,
			Version:         dbDaemon.Version,
		},
	}

	switch {
	case dbDaemon.Name.IsKea():
		if dbDaemon.KeaDaemon != nil && dbDaemon.KeaDaemon.Config != nil {
			daemon.Files, daemon.Backends = getKeaStorages(dbDaemon.KeaDaemon.Config.Config)
		}

		if hooks := kea.GetDaemonHooks(dbDaemon); len(hooks) > 0 {
			daemon.Hooks = hooks
		} else {
			daemon.Hooks = []string{}
		}

		daemon.LogTargets = []*models.LogTarget{}
		for _, logTarget := range dbDaemon.LogTargets {
			daemon.LogTargets = append(daemon.LogTargets, &models.LogTarget{
				ID:       logTarget.ID,
				Name:     logTarget.Name,
				Severity: logTarget.Severity,
				Output:   logTarget.Output,
			})
		}
	case dbDaemon.Name == daemonname.Bind9:
		var namedStats bind9stats.Bind9NamedStats
		if dbDaemon.Bind9Daemon != nil {
			namedStats = dbDaemon.Bind9Daemon.Stats.NamedStats
			daemon.ZoneCount = dbDaemon.Bind9Daemon.Stats.ZoneCount
			daemon.AutoZoneCount = dbDaemon.Bind9Daemon.Stats.AutomaticZoneCount
		}
		var views []*models.Bind9DaemonView
		for name, view := range namedStats.Views {
			queryHits := view.Resolver.CacheStats["QueryHits"]
			queryMisses := view.Resolver.CacheStats["QueryMisses"]
			queryTotal := float64(queryHits) + float64(queryMisses)
			var queryHitRatio float64
			if queryTotal > 0 {
				queryHitRatio = float64(queryHits) / queryTotal
			}
			views = append(views, &models.Bind9DaemonView{
				Name:          name,
				QueryHits:     queryHits,
				QueryMisses:   queryMisses,
				QueryHitRatio: queryHitRatio,
			})
		}
		// Sort views by name. Otherwise they will be returned in the random
		// order of a map.
		sort.Slice(views, func(i, j int) bool {
			return views[i].Name < views[j].Name
		})

		daemon.Views = views
	case dbDaemon.Name == daemonname.PDNS:
		if dbDaemon.PDNSDaemon != nil {
			daemon.URL = dbDaemon.PDNSDaemon.Details.URL
			daemon.ConfigURL = dbDaemon.PDNSDaemon.Details.ConfigURL
			daemon.ZonesURL = dbDaemon.PDNSDaemon.Details.ZonesURL
			daemon.AutoprimariesURL = dbDaemon.PDNSDaemon.Details.AutoprimariesURL
		}
	}

	if dbDaemon.Machine != nil {
		daemon.MachineLabel = dbDaemon.Machine.GetLabel()
		agentStats := r.Agents.GetConnectedAgentStatsWrapper(dbDaemon.Machine.Address, dbDaemon.Machine.AgentPort)
		if agentStats != nil {
			defer agentStats.Close()
			daemon.AgentCommErrors = agentStats.GetStats().GetTotalAgentErrorCount()

			switch {
			case dbDaemon.Name.IsKea():
				keaStats := agentStats.GetStats().GetKeaStats()
				daemon.DaemonCommErrors = keaStats.GetErrorCount(dbDaemon.Name)
				keaVersion := storkutil.ParseSemanticVersionOrLatest(dbDaemon.Version)
				if keaVersion.LessThan(storkutil.SemanticVersion{Major: 3, Minor: 0, Patch: 0}) {
					// Communication via Kea Control Agent.
					daemon.CaCommErrors = keaStats.GetErrorCount(daemonname.CA)
				} else {
					// Communication directly to the daemon.
					daemon.CaCommErrors = 0
				}
			case dbDaemon.Name == daemonname.Bind9:
				bind9Errors := agentStats.GetStats().GetBind9Stats()
				daemon.DaemonCommErrors = bind9Errors.GetErrorCount(dbmodel.AccessPointControl)
				daemon.StatsCommErrors = bind9Errors.GetErrorCount(dbmodel.AccessPointStatistics)
			}
		}
	}

	return daemon
}

// Converts the Kea daemon to REST API format.
func (r *RestAPI) keaDaemonToRestAPI(dbDaemon *dbmodel.Daemon) *models.KeaDaemon {
	daemon := r.daemonToRestAPI(dbDaemon)
	return &models.KeaDaemon{
		Daemon:           daemon.Daemon,
		KeaDaemonDetails: daemon.KeaDaemonDetails,
	}
}

// Converts database daemon structure to minimalistic REST API format.
func (r *RestAPI) simpleDaemonToRestAPI(dbDaemon *dbmodel.Daemon) *models.SimpleDaemon {
	daemon := &models.SimpleDaemon{
		ID:      dbDaemon.ID,
		Name:    string(dbDaemon.Name),
		Label:   dbDaemon.GetLabel(),
		Version: dbDaemon.Version,
		Active:  dbDaemon.Active,
	}

	return daemon
}

// Internal function to get daemons from the database with given filter.
func (r *RestAPI) getDaemons(offset, limit int64, filterText *string, sortField string, sortDir dbmodel.SortDirEnum, daemonNames ...daemonname.Name) (*models.Daemons, error) {
	dbDaemons, total, err := dbmodel.GetDaemonsByPage(r.DB, offset, limit, filterText, sortField, sortDir, daemonNames...)
	if err != nil {
		return nil, err
	}

	var daemons []*models.AnyDaemon
	for _, dbDaemon := range dbDaemons {
		daemon := r.daemonToRestAPI(&dbDaemon)
		daemons = append(daemons, daemon)
	}
	return &models.Daemons{
		Items: daemons,
		Total: total,
	}, nil
}

// Searches for daemons that meet the given filter conditions. The results
// are paginated.
func (r *RestAPI) GetDaemons(ctx context.Context, params services.GetDaemonsParams) middleware.Responder {
	var start int64
	if params.Start != nil {
		start = *params.Start
	}

	var limit int64 = 10
	if params.Limit != nil {
		limit = *params.Limit
	}

	sortField := ""
	if params.SortField != nil {
		sortField = *params.SortField
	}

	sortDir := dbmodel.SortDirAny
	if params.SortDir != nil {
		sortDir = dbmodel.SortDirEnum(*params.SortDir)
	}

	log.WithFields(log.Fields{
		"start":     start,
		"limit":     limit,
		"text":      params.Text,
		"daemons":   params.Daemons,
		"sortField": sortField,
		"sortDir":   sortDir,
	}).Info("query daemons")

	daemonNames := make([]daemonname.Name, 0, len(params.Daemons))
	for _, dn := range params.Daemons {
		daemonNames = append(daemonNames, daemonname.Name(dn))
	}

	daemons, err := r.getDaemons(start, limit, params.Text, sortField, sortDir, daemonNames...)
	if err != nil {
		msg := "Cannot get daemons from db"
		log.WithError(err).Error(msg)
		rsp := services.NewGetDaemonsDefault(http.StatusInternalServerError).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}
	rsp := services.NewGetDaemonsOK().WithPayload(daemons)
	return rsp
}

// Returns a list of all daemons' ids and names. A client calls this function to create a
// drop down list with available daemons or to validate user's input against daemons' names
// available in the system.
func (r *RestAPI) GetDaemonsDirectory(ctx context.Context, params services.GetDaemonsDirectoryParams) middleware.Responder {
	dbDaemons, err := dbmodel.GetAllDaemonsWithRelations(r.DB, dbmodel.DaemonRelationMachine)
	if err != nil {
		msg := "Cannot get daemons directory from the database"
		log.WithError(err).Error(msg)
		rsp := services.NewGetDaemonsDirectoryDefault(http.StatusInternalServerError).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}

	daemons := &models.SimpleDaemons{}
	for _, dbDaemon := range dbDaemons {
		daemon := r.simpleDaemonToRestAPI(&dbDaemon)
		daemon.Machine = r.simpleMachineToRestAPI(*dbDaemon.Machine)
		daemon.MachineID = dbDaemon.MachineID
		daemons.Items = append(daemons.Items, daemon)
	}
	daemons.Total = int64(len(daemons.Items))

	rsp := services.NewGetDaemonsDirectoryOK().WithPayload(daemons)
	return rsp
}

// Returns a list of daemons for which the server discovered some communication problems.
// It includes a lack of communication with the agent or the daemons behind it.
func (r *RestAPI) GetDaemonsWithCommunicationIssues(ctx context.Context, params services.GetDaemonsWithCommunicationIssuesParams) middleware.Responder {
	// Get all daemons with a minimal set of relations.
	dbDaemons, err := dbmodel.GetAllDaemonsWithRelations(r.DB, dbmodel.DaemonRelationMachine, dbmodel.DaemonRelationAccessPoints)
	if err != nil {
		msg := "Cannot get daemons from the database"
		log.WithError(err).Error(msg)
		rsp := services.NewGetDaemonsWithCommunicationIssuesDefault(http.StatusInternalServerError).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}

	daemons := []*models.AnyDaemon{}
	for _, dbDaemon := range dbDaemons {
		if !dbDaemon.Monitored {
			continue
		}

		daemon := r.daemonToRestAPI(&dbDaemon)
		if daemon.AgentCommErrors > 0 || daemon.DaemonCommErrors > 0 || daemon.CaCommErrors > 0 || daemon.StatsCommErrors > 0 {
			daemons = append(daemons, daemon)
		}
	}
	// Send the list.
	rsp := services.NewGetDaemonsWithCommunicationIssuesOK().WithPayload(&models.Daemons{
		Items: daemons,
		Total: int64(len(daemons)),
	})
	return rsp
}

// Returns a daemon for a given ID or HTTP 404 status if it's missing.
func (r *RestAPI) GetDaemon(ctx context.Context, params services.GetDaemonParams) middleware.Responder {
	dbDaemon, err := dbmodel.GetDaemonByID(r.DB, params.ID)
	if err != nil {
		msg := fmt.Sprintf("Cannot get daemon with ID %d from db", params.ID)
		log.WithError(err).Error(msg)
		rsp := services.NewGetDaemonDefault(http.StatusInternalServerError).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}
	if dbDaemon == nil {
		msg := fmt.Sprintf("Daemon with ID %d not found", params.ID)
		rsp := services.NewGetDaemonDefault(http.StatusNotFound).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}

	daemon := r.daemonToRestAPI(dbDaemon)
	rsp := services.NewGetDaemonOK().WithPayload(daemon)
	return rsp
}

// Gets current status of services for the given Kea daemon.
func getKeaServicesStatus(db *dbops.PgDB, daemon *dbmodel.Daemon) *models.ServicesStatus {
	servicesStatus := &models.ServicesStatus{}

	daemonServices, err := dbmodel.GetDetailedServicesByDaemonID(db, daemon.ID)
	if err != nil {
		log.WithError(err).Error("cannot get kea services from db")
		return nil
	}

	for _, s := range daemonServices {
		if s.HAService == nil {
			continue
		}
		ha := s.HAService
		keaStatus := models.KeaStatus{
			Daemon: string(ha.HAType),
		}
		secondaryRole := "secondary"
		if ha.HAMode == dbmodel.HAModeHotStandby {
			secondaryRole = "standby"
		}
		// Calculate age.
		age := make([]int64, 2)
		statusTime := make([]*strfmt.DateTime, 2)
		now := storkutil.UTCNow()
		for i, t := range []time.Time{ha.PrimaryStatusCollectedAt, ha.SecondaryStatusCollectedAt} {
			// If status time hasn't been set yet, return a negative age value to
			// indicate that it cannot be displayed.
			if t.IsZero() || now.Before(t) {
				age[i] = -1
			} else {
				age[i] = int64(now.Sub(t).Seconds())
				datetime := strfmt.DateTime(t)
				statusTime[i] = &datetime
			}
		}
		// Format failover times into string.
		failoverTime := make([]*strfmt.DateTime, 2)
		for i, t := range []time.Time{ha.PrimaryLastFailoverAt, ha.SecondaryLastFailoverAt} {
			// Only display the non-zero failover times and the times that are
			// before current time.
			if !t.IsZero() && now.After(t) {
				datetime := strfmt.DateTime(t)
				failoverTime[i] = &datetime
			}
		}
		// Get the label for daemons taking part in HA.
		labels := make([]string, 2)

		for i := range s.Daemons {
			switch s.Daemons[i].ID {
			case ha.PrimaryID:
				labels[0] = s.Daemons[i].GetLabel()
			case ha.SecondaryID:
				labels[0] = s.Daemons[i].GetLabel()
			}
		}
		// Get the communication state value.
		commInterrupted := make([]int64, 2)
		for i, c := range []*bool{ha.PrimaryCommInterrupted, ha.SecondaryCommInterrupted} {
			if c == nil {
				// Negative value indicates that the communication state is unknown.
				// Quite possibly that we're running earlier Kea version that doesn't
				// provide this information.
				commInterrupted[i] = -1
			} else if *c {
				// Communication interrupted.
				commInterrupted[i] = 1
			}
		}
		keaStatus.HaServers = &models.KeaStatusHaServers{
			Relationship: ha.Relationship,
			PrimaryServer: &models.KeaHAServerStatus{
				Age:                age[0],
				FailoverTime:       failoverTime[0],
				ID:                 ha.PrimaryID,
				Label:              labels[0],
				InTouch:            ha.PrimaryReachable,
				Role:               "primary",
				Scopes:             ha.PrimaryLastScopes,
				State:              ha.PrimaryLastState,
				StatusTime:         statusTime[0],
				CommInterrupted:    commInterrupted[0],
				ConnectingClients:  ha.PrimaryConnectingClients,
				UnackedClients:     ha.PrimaryUnackedClients,
				UnackedClientsLeft: ha.PrimaryUnackedClientsLeft,
				AnalyzedPackets:    ha.PrimaryAnalyzedPackets,
			},
		}

		// Including the information about the secondary server only
		// makes sense for load-balancing and hot-standby mode.
		if ha.HAMode != "passive-backup" {
			keaStatus.HaServers.SecondaryServer = &models.KeaHAServerStatus{
				Age:                age[1],
				Label:              labels[1],
				FailoverTime:       failoverTime[1],
				ID:                 ha.SecondaryID,
				InTouch:            ha.SecondaryReachable,
				Role:               secondaryRole,
				Scopes:             ha.SecondaryLastScopes,
				State:              ha.SecondaryLastState,
				StatusTime:         statusTime[1],
				CommInterrupted:    commInterrupted[1],
				ConnectingClients:  ha.SecondaryConnectingClients,
				UnackedClients:     ha.SecondaryUnackedClients,
				UnackedClientsLeft: ha.SecondaryUnackedClientsLeft,
				AnalyzedPackets:    ha.SecondaryAnalyzedPackets,
			}
		}

		serviceStatus := &models.ServiceStatus{
			Status: struct {
				models.KeaStatus
			}{
				keaStatus,
			},
		}
		servicesStatus.Items = append(servicesStatus.Items, serviceStatus)
	}

	return servicesStatus
}

// Gets current status of services which the given daemons is associated with.
func (r *RestAPI) GetDaemonServicesStatus(ctx context.Context, params services.GetDaemonServicesStatusParams) middleware.Responder {
	daemon, err := dbmodel.GetDaemonByID(r.DB, params.ID)
	if err != nil {
		msg := fmt.Sprintf("Cannot get daemon with ID %d from db", params.ID)
		log.WithError(err).Error(msg)
		rsp := services.NewGetDaemonServicesStatusDefault(http.StatusInternalServerError).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}

	var servicesStatus *models.ServicesStatus
	// If this is Kea daemon, get the Kea DHCP servers status which possibly
	// includes HA status.
	if daemon.Name.IsKea() {
		servicesStatus = getKeaServicesStatus(r.DB, daemon)
		if servicesStatus == nil {
			msg := fmt.Sprintf("Cannot get status of daemon with ID %d", daemon.ID)
			rsp := services.NewGetDaemonServicesStatusDefault(http.StatusInternalServerError).WithPayload(&models.APIError{
				Message: &msg,
			})
			return rsp
		}
	} else {
		servicesStatus = &models.ServicesStatus{}
	}

	rsp := services.NewGetDaemonServicesStatusOK().WithPayload(servicesStatus)
	return rsp
}

// Get statistics about daemon.
func (r *RestAPI) GetDaemonsStats(ctx context.Context, params services.GetDaemonsStatsParams) middleware.Responder {
	// The second argument indicates that only basic information about the daemons
	// should be returned, i.e. the information stored in the daemon table.
	dbDaemons, err := dbmodel.GetAllDaemons(r.DB)
	if err != nil {
		msg := "Cannot get all daemons from db"
		log.WithError(err).Error(msg)
		rsp := services.NewGetDaemonsStatsDefault(http.StatusInternalServerError).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}

	daemonsStats := models.DaemonsStats{
		DhcpDaemonsNotOk: 0,
		DhcpDaemonsTotal: 0,
		DNSDaemonsTotal:  0,
		DNSDaemonsNotOk:  0,
	}
	for _, dbDaemon := range dbDaemons {
		switch {
		case dbDaemon.Name.IsKea():
			daemonsStats.DhcpDaemonsTotal++
			if !dbDaemon.Active {
				daemonsStats.DhcpDaemonsNotOk++
			}
		case dbDaemon.Name.IsDNS():
			daemonsStats.DNSDaemonsTotal++
			if !dbDaemon.Active {
				daemonsStats.DNSDaemonsNotOk++
			}
		}
	}

	rsp := services.NewGetDaemonsStatsOK().WithPayload(&daemonsStats)
	return rsp
}

// Get DHCP overview.
func (r *RestAPI) GetDhcpOverview(ctx context.Context, params dhcp.GetDhcpOverviewParams) middleware.Responder {
	// get list of mostly utilized subnets
	filters := &dbmodel.SubnetsByPageFilters{}
	filters.SetIPv4Family()

	subnets4, err := r.getSubnets(0, 5, filters, "addr_utilization", dbmodel.SortDirDesc)
	if err != nil {
		msg := "Cannot get IPv4 subnets from db"
		log.WithError(err).Error(msg)
		rsp := dhcp.NewGetDhcpOverviewDefault(http.StatusInternalServerError).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}

	filters.SetIPv6Family()
	subnets6, err := r.getSubnets(0, 5, filters, "addr_utilization", dbmodel.SortDirDesc)
	if err != nil {
		msg := "Cannot get IPv6 subnets from db"
		log.WithError(err).Error(msg)
		rsp := dhcp.NewGetDhcpOverviewDefault(http.StatusInternalServerError).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}

	// get list of mostly utilized shared networks
	sharedNetworks4, err := r.getSharedNetworks(0, 5, 0, 4, nil, "addr_utilization", dbmodel.SortDirDesc)
	if err != nil {
		msg := "Cannot get IPv4 shared networks from db"
		log.WithError(err).Error(msg)
		rsp := dhcp.NewGetDhcpOverviewDefault(http.StatusInternalServerError).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}

	sharedNetworks6, err := r.getSharedNetworks(0, 5, 0, 6, nil, "addr_utilization", dbmodel.SortDirDesc)
	if err != nil {
		msg := "Cannot get IPv6 shared networks from db"
		log.WithError(err).Error(msg)
		rsp := dhcp.NewGetDhcpOverviewDefault(http.StatusInternalServerError).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}

	// get dhcp statistics
	stats, err := dbmodel.GetAllStats(r.DB)
	if err != nil {
		msg := "Cannot get statistics from db"
		log.WithError(err).Error(msg)
		rsp := dhcp.NewGetDhcpOverviewDefault(http.StatusInternalServerError).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}

	dhcp4Stats := &models.Dhcp4Stats{
		AssignedAddresses: fmt.Sprint(stats[dbmodel.StatNameAssignedAddresses]),
		TotalAddresses:    fmt.Sprint(stats[dbmodel.StatNameTotalAddresses]),
		DeclinedAddresses: fmt.Sprint(stats[dbmodel.StatNameDeclinedAddresses]),
	}
	dhcp6Stats := &models.Dhcp6Stats{
		AssignedNAs: fmt.Sprint(stats[dbmodel.StatNameAssignedNAs]),
		TotalNAs:    fmt.Sprint(stats[dbmodel.StatNameTotalNAs]),
		AssignedPDs: fmt.Sprint(stats[dbmodel.StatNameAssignedPDs]),
		TotalPDs:    fmt.Sprint(stats[dbmodel.StatNameTotalPDs]),
		DeclinedNAs: fmt.Sprint(stats[dbmodel.StatNameDeclinedNAs]),
	}

	// get kea daemons and daemons statuses
	dbDaemons, err := dbmodel.GetDaemonsByName(r.DB, daemonname.DHCPv4, daemonname.DHCPv6)
	if err != nil {
		msg := "Cannot get DHCP daemons from db"
		log.WithError(err).Error(msg)
		rsp := dhcp.NewGetDhcpOverviewDefault(http.StatusInternalServerError).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}

	var dhcpDaemons []*models.DhcpDaemon
	for _, dbDaemon := range dbDaemons {
		if !dbDaemon.Monitored {
			// do not show not monitored daemons (ie. show only monitored services)
			continue
		}
		// TODO: Currently Kea supports only one HA relationship per daemon.
		// Until we extend Kea to support multiple relationships per daemon
		// or integrate ISC DHCP with Stork, the number of HA states returned
		// will be 0 or 1. Therefore, we take the first HA state if it exists
		// and return it over the REST API.
		var (
			haEnabled               bool
			haRelationshipOverviews []*models.DhcpDaemonHARelationshipOverview
			haState                 string
			haFailureAt             *strfmt.DateTime
		)
		if overview := dbDaemon.GetHAOverview(); len(overview) > 0 {
			haEnabled = true
			for i := range overview {
				haState = overview[i].State
				if !overview[0].LastFailureAt.IsZero() {
					haFailureAt = convertToOptionalDatetime(overview[0].LastFailureAt)
				}
				haRelationshipOverviews = append(haRelationshipOverviews, &models.DhcpDaemonHARelationshipOverview{
					HaState:     haState,
					HaFailureAt: haFailureAt,
				})
			}
		}

		daemon := r.daemonToRestAPI(&dbDaemon)
		dhcpDaemon := &models.DhcpDaemon{
			Daemon:       daemon.Daemon,
			CaCommErrors: daemon.CaCommErrors,
			Rps1:         dbDaemon.KeaDaemon.KeaDHCPDaemon.Stats.RPS1,
			Rps2:         dbDaemon.KeaDaemon.KeaDHCPDaemon.Stats.RPS2,
			HaEnabled:    haEnabled,
			HaOverview:   haRelationshipOverviews,
		}
		dhcpDaemons = append(dhcpDaemons, dhcpDaemon)
	}

	// combine gathered information
	overview := &models.DhcpOverview{
		Subnets4:        subnets4,
		Subnets6:        subnets6,
		SharedNetworks4: sharedNetworks4,
		SharedNetworks6: sharedNetworks6,
		Dhcp4Stats:      dhcp4Stats,
		Dhcp6Stats:      dhcp6Stats,
		DhcpDaemons:     dhcpDaemons,
	}

	rsp := dhcp.NewGetDhcpOverviewOK().WithPayload(overview)
	return rsp
}

// Update a daemon.
func (r *RestAPI) UpdateDaemon(ctx context.Context, params services.UpdateDaemonParams) middleware.Responder {
	dbDaemon, err := dbmodel.GetDaemonByID(r.DB, params.ID)
	if err != nil {
		msg := fmt.Sprintf("Cannot get daemon with ID %d from db", params.ID)
		log.WithError(err).Error(msg)
		rsp := services.NewUpdateDaemonDefault(http.StatusInternalServerError).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}
	if dbDaemon == nil {
		msg := fmt.Sprintf("Cannot find daemon with ID %d", params.ID)
		rsp := services.NewUpdateDaemonDefault(http.StatusNotFound).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}

	oldMonitored := dbDaemon.Monitored

	dbDaemon.Monitored = params.Daemon.Monitored

	err = dbmodel.UpdateDaemon(r.DB, dbDaemon)
	if err != nil {
		msg := fmt.Sprintf("Failed to update daemon with ID %d", params.ID)
		rsp := services.NewUpdateDaemonDefault(http.StatusInternalServerError).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}

	_, dbUser := r.SessionManager.Logged(ctx)

	if oldMonitored != params.Daemon.Monitored {
		if params.Daemon.Monitored {
			r.EventCenter.AddInfoEvent("{user} enabled monitoring {daemon}", dbUser, dbDaemon, dbDaemon.Machine)
		} else {
			r.EventCenter.AddWarningEvent("{user} disabled monitoring {daemon}", dbUser, dbDaemon, dbDaemon.Machine)
		}
	}

	rsp := services.NewUpdateDaemonOK()
	return rsp
}

// Delete a daemon. Only superadmin can do it.
func (r *RestAPI) DeleteDaemon(ctx context.Context, params services.DeleteDaemonParams) middleware.Responder {
	ok, dbUser := r.SessionManager.Logged(ctx)
	if !ok || !dbUser.InGroup(&dbmodel.SystemGroup{ID: dbmodel.SuperAdminGroupID}) {
		msg := "User is forbidden to delete daemon"
		rsp := services.NewDeleteDaemonDefault(http.StatusForbidden).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}

	dbDaemon, err := dbmodel.GetDaemonByID(r.DB, params.ID)
	if err != nil {
		msg := fmt.Sprintf("Cannot get daemon with ID %d from db", params.ID)
		log.WithError(err).Error(msg)
		rsp := services.NewDeleteDaemonDefault(http.StatusInternalServerError).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}
	if dbDaemon == nil {
		msg := fmt.Sprintf("Cannot find daemon with ID %d", params.ID)
		rsp := services.NewDeleteDaemonDefault(http.StatusNotFound).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}

	err = dbmodel.DeleteDaemon(r.DB, dbDaemon)
	if err != nil {
		msg := fmt.Sprintf("Failed to delete daemon with ID %d", params.ID)
		rsp := services.NewDeleteDaemonDefault(http.StatusInternalServerError).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}

	r.EventCenter.AddInfoEvent("{user} deleted {daemon} of {machine}", dbUser, dbDaemon, dbDaemon.Machine)

	rsp := services.NewDeleteDaemonOK()
	return rsp
}

// Returns the authentication key assigned to the given access point.
// If there is no authentication key assigned, returns an empty string.
func (r *RestAPI) GetAccessPointKey(ctx context.Context, params services.GetAccessPointKeyParams) middleware.Responder {
	_, dbUser := r.SessionManager.Logged(ctx)
	if !dbUser.InGroup(&dbmodel.SystemGroup{ID: dbmodel.SuperAdminGroupID}) {
		msg := "User is forbidden to get access point key"
		rsp := services.NewGetAccessPointKeyDefault(http.StatusForbidden).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}

	daemon, err := dbmodel.GetDaemonByID(r.DB, params.DaemonID)
	if err != nil {
		msg := "Cannot retrieve daemon from database"
		log.WithError(err).Error(msg)
		rsp := services.NewGetAccessPointKeyDefault(http.StatusInternalServerError).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}
	if daemon == nil {
		msg := fmt.Sprintf("Cannot find daemon with ID %d", params.DaemonID)
		rsp := services.NewGetAccessPointKeyDefault(http.StatusNotFound).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}

	accessPoint, err := daemon.GetAccessPoint(dbmodel.AccessPointType(params.Type))
	if err != nil {
		msg := "Cannot retrieve access point from database"
		log.WithError(err).Error(msg)
		rsp := services.NewGetAccessPointKeyDefault(http.StatusInternalServerError).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}

	if accessPoint == nil {
		msg := fmt.Sprintf("Cannot find access point with Daemon ID %d and type %s", params.DaemonID, params.Type)
		rsp := services.NewGetAccessPointKeyDefault(http.StatusNotFound).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}

	rsp := services.NewGetAccessPointKeyOK().WithPayload(accessPoint.Key)
	return rsp
}
