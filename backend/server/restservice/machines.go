package restservice

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/asaskevich/govalidator"
	"github.com/go-openapi/runtime/middleware"
	"github.com/go-openapi/strfmt"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"isc.org/stork"
	"isc.org/stork/pki"
	"isc.org/stork/server/agentcomm"
	"isc.org/stork/server/apps"
	"isc.org/stork/server/apps/kea"
	"isc.org/stork/server/certs"
	dbops "isc.org/stork/server/database"
	dbmodel "isc.org/stork/server/database/model"
	"isc.org/stork/server/gen/models"
	dhcp "isc.org/stork/server/gen/restapi/operations/d_h_c_p"
	"isc.org/stork/server/gen/restapi/operations/general"
	"isc.org/stork/server/gen/restapi/operations/services"
	storkutil "isc.org/stork/util"
)

// Get version of Stork server.
func (r *RestAPI) GetVersion(ctx context.Context, params general.GetVersionParams) middleware.Responder {
	bd := stork.BuildDate
	v := stork.Version
	ver := models.Version{
		Date:    &bd,
		Version: &v,
	}
	return general.NewGetVersionOK().WithPayload(&ver)
}

// Convert db machine to rest structure.
func (r *RestAPI) machineToRestAPI(dbMachine dbmodel.Machine) *models.Machine {
	var apps []*models.App
	for _, app := range dbMachine.Apps {
		a := r.appToRestAPI(app)
		apps = append(apps, a)
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
		LastVisitedAt:        strfmt.DateTime(dbMachine.LastVisitedAt),
		Error:                dbMachine.Error,
		Apps:                 apps,
	}
	return &m
}

// Get runtime state of indicated machine.
func (r *RestAPI) GetMachineState(ctx context.Context, params services.GetMachineStateParams) middleware.Responder {
	dbMachine, err := dbmodel.GetMachineByID(r.DB, params.ID)
	if err != nil {
		msg := fmt.Sprintf("cannot get machine with id %d from db", params.ID)
		log.Error(err)
		rsp := services.NewGetMachineStateDefault(http.StatusInternalServerError).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}
	if dbMachine == nil {
		msg := fmt.Sprintf("cannot find machine with id %d", params.ID)
		rsp := services.NewGetMachineStateDefault(http.StatusNotFound).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}

	errStr := apps.GetMachineAndAppsState(ctx, r.DB, dbMachine, r.Agents, r.EventCenter)
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
	}

	for _, dbM := range dbMachines {
		m := r.machineToRestAPI(dbM)
		machines.Items = append(machines.Items, m)
	}

	return machines, nil
}

// Get machines where Stork Agent is running.
func (r *RestAPI) GetMachines(ctx context.Context, params services.GetMachinesParams) middleware.Responder {
	var start int64 = 0
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

	app := ""
	if params.App != nil {
		app = *params.App
	}

	log.WithFields(log.Fields{
		"start": start,
		"limit": limit,
		"text":  text,
		"app":   app,
	}).Info("query machines")

	machines, err := r.getMachines(start, limit, params.Text, params.Authorized, "", dbmodel.SortDirAny)
	if err != nil {
		log.Error(err)
		msg := "cannot get machines from db"
		rsp := services.NewGetMachinesDefault(http.StatusInternalServerError).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}
	rsp := services.NewGetMachinesOK().WithPayload(machines)
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
		log.Error(err)
		msg := "cannot retrieve server token from database"
		return false, http.StatusInternalServerError, msg
	}
	if dbServerToken == nil {
		msg := "server internal problem - server token is empty"
		log.Error(msg)
		return false, http.StatusInternalServerError, msg
	}
	dbServerTokenStr := string(dbServerToken)

	// if provided token does not match then machine is not authorized and return error
	if serverToken != dbServerTokenStr {
		msg := "provided server token is wrong"
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
		log.Error(err)
		msg := fmt.Sprintf("cannot get machine with id %d from db", params.ID)
		rsp := services.NewGetMachineDefault(http.StatusInternalServerError).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}
	if dbMachine == nil {
		msg := fmt.Sprintf("cannot find machine with id %d", params.ID)
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
		log.Warnf("cannot create machine: missing parameters")
		msg := "missing parameters"
		rsp := services.NewCreateMachineDefault(http.StatusBadRequest).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}
	addr := *params.Machine.Address
	if !govalidator.IsHost(*params.Machine.Address) {
		log.Warnf("problem with parsing address %s", addr)
		msg := "cannot parse address"
		rsp := services.NewCreateMachineDefault(http.StatusBadRequest).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}
	if params.Machine.AgentPort <= 0 || params.Machine.AgentPort > 65535 {
		log.Warnf("bad agent port %d", params.Machine.AgentPort)
		msg := "bad port"
		rsp := services.NewCreateMachineDefault(http.StatusBadRequest).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}
	if params.Machine.AgentCSR == nil {
		msg := "agent CSR cannot be empty"
		log.Warnf(msg)
		rsp := services.NewCreateMachineDefault(http.StatusBadRequest).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}

	dbMachine, err := dbmodel.GetMachineByAddressAndAgentPort(r.DB, addr, params.Machine.AgentPort)
	if err != nil {
		msg := fmt.Sprintf("problem with finding machine %s:%d in database", addr, params.Machine.AgentPort)
		log.Warnf(msg)
		rsp := services.NewCreateMachineDefault(http.StatusInternalServerError).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}

	// check server token
	machineAuthorized, httpRspCode, rspMsg := r.checkServerToken(params.Machine.ServerToken, true)
	if httpRspCode != 0 {
		rsp := services.NewCreateMachineDefault(httpRspCode).WithPayload(&models.APIError{
			Message: &rspMsg,
		})
		return rsp
	}

	// sign agent cert
	agentCSR := []byte(*params.Machine.AgentCSR)
	certSerialNumber, err := dbmodel.GetNewCertSerialNumber(r.DB)
	if err != nil {
		log.Error(err)
		msg := "problem with generating serial number for cert"
		rsp := services.NewCreateMachineDefault(http.StatusInternalServerError).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}
	rootKeyPEM, err := dbmodel.GetSecret(r.DB, dbmodel.SecretCAKey)
	if err != nil {
		log.Error(err)
		msg := "problem with loading server CA private key"
		rsp := services.NewCreateMachineDefault(http.StatusInternalServerError).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}
	rootCertPEM, err := dbmodel.GetSecret(r.DB, dbmodel.SecretCACert)
	if err != nil {
		log.Error(err)
		msg := "problem with loading server CA cert"
		rsp := services.NewCreateMachineDefault(http.StatusInternalServerError).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}
	agentCertPEM, agentCertFingerprint, paramsErr, innerErr := pki.SignCert(agentCSR, certSerialNumber, rootCertPEM, rootKeyPEM)
	if paramsErr != nil {
		log.Error(paramsErr)
		msg := "problem with agent CSR"
		rsp := services.NewCreateMachineDefault(http.StatusBadRequest).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}
	if innerErr != nil {
		log.Error(innerErr)
		msg := "problem with signing agent CSR"
		rsp := services.NewCreateMachineDefault(http.StatusInternalServerError).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}

	// We can't create new machine and pull machines' states at the same time. This may
	// put heavy workload on the server and it may also result in conflicts. Temporarily
	// disable the puller while the new machine is being added.
	if r.Pullers != nil && r.Pullers.AppsStatePuller != nil {
		r.Pullers.AppsStatePuller.Pause()
		defer r.Pullers.AppsStatePuller.Unpause()
	}

	if dbMachine == nil {
		dbMachine = &dbmodel.Machine{
			Address:         addr,
			AgentPort:       params.Machine.AgentPort,
			AgentToken:      params.Machine.AgentToken,
			CertFingerprint: agentCertFingerprint,
			Authorized:      machineAuthorized,
		}
		err = dbmodel.AddMachine(r.DB, dbMachine)
		if err != nil {
			log.Error(err)
			msg := fmt.Sprintf("cannot store machine %s", addr)
			rsp := services.NewCreateMachineDefault(http.StatusInternalServerError).WithPayload(&models.APIError{
				Message: &msg,
			})
			return rsp
		}
		r.EventCenter.AddInfoEvent("added {machine}", dbMachine)
	} else {
		dbMachine.AgentToken = params.Machine.AgentToken
		dbMachine.CertFingerprint = agentCertFingerprint
		dbMachine.Authorized = machineAuthorized
		err = dbmodel.UpdateMachine(r.DB, dbMachine)
		if err != nil {
			log.Error(err)
			msg := fmt.Sprintf("cannot update machine %s in database", addr)
			rsp := services.NewCreateMachineDefault(http.StatusInternalServerError).WithPayload(&models.APIError{
				Message: &msg,
			})
			return rsp
		}
		r.EventCenter.AddInfoEvent("re-registered {machine}", dbMachine)
	}

	m := &models.NewMachineResp{
		ID:           dbMachine.ID,
		ServerCACert: string(rootCertPEM),
		AgentCert:    string(agentCertPEM),
	}
	rsp := services.NewCreateMachineOK().WithPayload(m)

	return rsp
}

// Ping given machine, i.e. check connectivity.
func (r *RestAPI) PingMachine(ctx context.Context, params services.PingMachineParams) middleware.Responder {
	// find machine in db
	dbMachine, err := dbmodel.GetMachineByID(r.DB, params.ID)
	if err != nil {
		msg := fmt.Sprintf("cannot get machine with id %d from db", params.ID)
		log.Error(err)
		rsp := services.NewPingMachineDefault(http.StatusInternalServerError).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}
	if dbMachine == nil {
		msg := fmt.Sprintf("cannot find machine with id %d", params.ID)
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
		msg := fmt.Sprintf("provided agent token is wrong (%s vs %s)", params.Ping.AgentToken, dbMachine.AgentToken)
		log.Error(msg)
		rsp := services.NewPingMachineDefault(http.StatusBadRequest).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}

	// ping machine
	ctx2, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	err = r.Agents.Ping(ctx2, dbMachine.Address, dbMachine.AgentPort)
	if err != nil {
		msg := "cannot ping machine"
		log.Error(err)
		rsp := services.NewPingMachineDefault(http.StatusInternalServerError).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}

	// so as all is ok then get state from the machine
	errStr := apps.GetMachineAndAppsState(ctx2, r.DB, dbMachine, r.Agents, r.EventCenter)
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
		log.Warnf("cannot update machine: missing parameters")
		msg := "missing parameters"
		rsp := services.NewUpdateMachineDefault(http.StatusBadRequest).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}
	addr := *params.Machine.Address
	if !govalidator.IsHost(*params.Machine.Address) {
		log.Warnf("problem with parsing address %s", addr)
		msg := "cannot parse address"
		rsp := services.NewUpdateMachineDefault(http.StatusBadRequest).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}
	if params.Machine.AgentPort <= 0 || params.Machine.AgentPort > 65535 {
		log.Warnf("bad agent port %d", params.Machine.AgentPort)
		msg := "bad port"
		rsp := services.NewUpdateMachineDefault(http.StatusBadRequest).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}

	dbMachine, err := dbmodel.GetMachineByID(r.DB, params.ID)
	if err != nil {
		log.Error(err)
		msg := fmt.Sprintf("cannot get machine with id %d from db", params.ID)
		rsp := services.NewUpdateMachineDefault(http.StatusInternalServerError).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}
	if dbMachine == nil {
		msg := fmt.Sprintf("cannot find machine with id %d", params.ID)
		rsp := services.NewUpdateMachineDefault(http.StatusNotFound).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}

	// check if there is no duplicate
	if dbMachine.Address != addr || dbMachine.AgentPort != params.Machine.AgentPort {
		dbMachine2, err := dbmodel.GetMachineByAddressAndAgentPort(r.DB, addr, params.Machine.AgentPort)
		if err == nil && dbMachine2 != nil && dbMachine2.ID != dbMachine.ID {
			msg := fmt.Sprintf("machine with address %s:%d already exists",
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
			msg := "user is forbidden to change machine authorization"
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
	err = r.DB.Update(dbMachine)
	if err != nil {
		log.Errorf("cannot update machine: %s", err)
		msg := fmt.Sprintf("cannot update machine with id %d in db", params.ID)
		rsp := services.NewUpdateMachineDefault(http.StatusInternalServerError).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}

	// as we just authorized machine so get its state now
	if !prevAuthorized && dbMachine.Authorized {
		ctx2, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
		errStr := apps.GetMachineAndAppsState(ctx2, r.DB, dbMachine, r.Agents, r.EventCenter)
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

// Get machines server token. It is used by user during manual agent registration.
func (r *RestAPI) GetMachinesServerToken(ctx context.Context, params services.GetMachinesServerTokenParams) middleware.Responder {
	// only super-admin can get server token
	_, dbUser := r.SessionManager.Logged(ctx)
	if !dbUser.InGroup(&dbmodel.SystemGroup{ID: dbmodel.SuperAdminGroupID}) {
		msg := "user is forbidden to get server token"
		rsp := services.NewGetMachinesServerTokenDefault(http.StatusForbidden).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}

	// get server token from database
	dbServerToken, err := dbmodel.GetSecret(r.DB, dbmodel.SecretServerToken)
	if err != nil {
		log.Error(err)
		msg := "cannot retrieve server token from database"
		rsp := services.NewGetMachinesServerTokenDefault(http.StatusInternalServerError).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}
	if dbServerToken == nil {
		msg := "server internal problem - server token is empty"
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
		msg := "user is forbidden to generate new server token"
		rsp := services.NewGetMachinesServerTokenDefault(http.StatusForbidden).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}

	// generate new server token
	dbServerToken, err := certs.GenerateServerToken(r.DB)
	if err != nil {
		log.Error(err)
		msg := "cannot regenerate server token"
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

// Add a machine where Stork Agent is running.
func (r *RestAPI) DeleteMachine(ctx context.Context, params services.DeleteMachineParams) middleware.Responder {
	dbMachine, err := dbmodel.GetMachineByID(r.DB, params.ID)
	if err == nil && dbMachine == nil {
		rsp := services.NewDeleteMachineOK()
		return rsp
	} else if err != nil {
		log.Error(err)
		msg := fmt.Sprintf("cannot delete machine %d", params.ID)
		rsp := services.NewDeleteMachineDefault(http.StatusInternalServerError).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}

	err = dbmodel.DeleteMachine(r.DB, dbMachine)
	if err != nil {
		log.Error(err)
		msg := fmt.Sprintf("cannot delete machine %d", params.ID)
		rsp := services.NewDeleteMachineDefault(http.StatusInternalServerError).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}

	r.EventCenter.AddInfoEvent("removed {machine}", dbMachine)

	rsp := services.NewDeleteMachineOK()

	return rsp
}

func (r *RestAPI) appToRestAPI(dbApp *dbmodel.App) *models.App {
	app := models.App{
		ID:      dbApp.ID,
		Name:    dbApp.Name,
		Type:    dbApp.Type,
		Version: dbApp.Meta.Version,
		Machine: &models.AppMachine{
			ID: dbApp.MachineID,
		},
	}
	if dbApp.Machine != nil {
		app.Machine.Address = dbApp.Machine.Address
		app.Machine.Hostname = dbApp.Machine.State.Hostname
	}

	var accessPoints []*models.AppAccessPoint
	for _, point := range dbApp.AccessPoints {
		accessPoints = append(accessPoints, &models.AppAccessPoint{
			Type:    point.Type,
			Address: point.Address,
			Port:    point.Port,
			Key:     point.Key,
		})
	}
	app.AccessPoints = accessPoints

	isKeaApp := dbApp.Type == dbmodel.AppTypeKea
	isBind9App := dbApp.Type == dbmodel.AppTypeBind9

	agentErrors := int64(0)
	var agentStats *agentcomm.AgentStats
	var accessPoint *dbmodel.AccessPoint
	if dbApp.Machine != nil {
		agentStats = r.Agents.GetConnectedAgentStats(dbApp.Machine.Address, dbApp.Machine.AgentPort)
		if agentStats != nil {
			agentErrors = agentStats.CurrentErrors
			accessPoint, _ = dbApp.GetAccessPoint(dbmodel.AccessPointControl)
		}
	}

	if isKeaApp {
		var keaStats *agentcomm.AgentKeaCommStats
		if agentStats != nil && accessPoint != nil {
			keaStats, _ = agentStats.AppCommStats[agentcomm.AppCommStatsKey{
				Address: accessPoint.Address,
				Port:    accessPoint.Port,
			}].(*agentcomm.AgentKeaCommStats)
		}
		var keaDaemons []*models.KeaDaemon
		for _, d := range dbApp.Daemons {
			dmn := &models.KeaDaemon{
				ID:              d.ID,
				Pid:             int64(d.Pid),
				Name:            d.Name,
				Active:          d.Active,
				Monitored:       d.Monitored,
				Version:         d.Version,
				ExtendedVersion: d.ExtendedVersion,
				Uptime:          d.Uptime,
				ReloadedAt:      strfmt.DateTime(d.ReloadedAt),
				Hooks:           []string{},
				AgentCommErrors: agentErrors,
			}
			if keaStats != nil {
				dmn.CaCommErrors = keaStats.CurrentErrorsCA
				dmn.DaemonCommErrors = keaStats.CurrentErrorsDaemons[d.Name]
			}

			hooksByDaemon := kea.GetDaemonHooks(dbApp)
			if hooksByDaemon != nil {
				hooksList, ok := hooksByDaemon[d.Name]
				if ok {
					dmn.Hooks = hooksList
				}
			}

			for _, logTarget := range d.LogTargets {
				dmn.LogTargets = append(dmn.LogTargets, &models.LogTarget{
					ID:       logTarget.ID,
					Name:     logTarget.Name,
					Severity: logTarget.Severity,
					Output:   logTarget.Output,
				})
			}
			keaDaemons = append(keaDaemons, dmn)
		}

		app.Details = struct {
			models.AppKea
			models.AppBind9
		}{
			models.AppKea{
				ExtendedVersion: dbApp.Meta.ExtendedVersion,
				Daemons:         keaDaemons,
			},
			models.AppBind9{},
		}
	}

	if isBind9App {
		var queryHitRatio float64
		var queryHits int64
		var queryMisses int64
		namedStats := dbApp.Daemons[0].Bind9Daemon.Stats.NamedStats
		if namedStats != nil {
			view, okView := namedStats.Views["_default"]
			if okView {
				queryHits = view.Resolver.CacheStats["QueryHits"]
				queryMisses = view.Resolver.CacheStats["QueryMisses"]
				queryTotal := float64(queryHits) + float64(queryMisses)
				if queryTotal > 0 {
					queryHitRatio = float64(queryHits) / queryTotal
				}
			}
		}

		bind9Daemon := &models.Bind9Daemon{
			ID:              dbApp.Daemons[0].ID,
			Pid:             int64(dbApp.Daemons[0].Pid),
			Name:            dbApp.Daemons[0].Name,
			Active:          dbApp.Daemons[0].Active,
			Monitored:       dbApp.Daemons[0].Monitored,
			Version:         dbApp.Daemons[0].Version,
			Uptime:          dbApp.Daemons[0].Uptime,
			ReloadedAt:      strfmt.DateTime(dbApp.Daemons[0].ReloadedAt),
			ZoneCount:       dbApp.Daemons[0].Bind9Daemon.Stats.ZoneCount,
			AutoZoneCount:   dbApp.Daemons[0].Bind9Daemon.Stats.AutomaticZoneCount,
			QueryHits:       queryHits,
			QueryMisses:     queryMisses,
			QueryHitRatio:   queryHitRatio,
			AgentCommErrors: agentErrors,
		}
		var bind9Stats *agentcomm.AgentBind9CommStats
		if agentStats != nil && accessPoint != nil {
			if bind9Stats, _ = agentStats.AppCommStats[agentcomm.AppCommStatsKey{
				Address: accessPoint.Address,
				Port:    accessPoint.Port,
			}].(*agentcomm.AgentBind9CommStats); bind9Stats != nil {
				bind9Daemon.RndcCommErrors = bind9Stats.CurrentErrorsRNDC
				bind9Daemon.StatsCommErrors = bind9Stats.CurrentErrorsStats
			}
		}
		app.Details = struct {
			models.AppKea
			models.AppBind9
		}{
			models.AppKea{},
			models.AppBind9{
				Daemon: bind9Daemon,
			},
		}
	}

	return &app
}

func (r *RestAPI) getApps(offset, limit int64, filterText *string, appType string, sortField string, sortDir dbmodel.SortDirEnum) (*models.Apps, error) {
	dbApps, total, err := dbmodel.GetAppsByPage(r.DB, offset, limit, filterText, appType, sortField, sortDir)
	if err != nil {
		return nil, err
	}
	apps := &models.Apps{
		Total: total,
	}
	for _, dbA := range dbApps {
		app := dbA
		a := r.appToRestAPI(&app)
		apps.Items = append(apps.Items, a)
	}
	return apps, nil
}

func (r *RestAPI) GetApps(ctx context.Context, params services.GetAppsParams) middleware.Responder {
	var start int64 = 0
	if params.Start != nil {
		start = *params.Start
	}

	var limit int64 = 10
	if params.Limit != nil {
		limit = *params.Limit
	}

	appType := ""
	if params.App != nil {
		appType = *params.App
	}

	log.WithFields(log.Fields{
		"start": start,
		"limit": limit,
		"text":  params.Text,
		"app":   appType,
	}).Info("query apps")

	apps, err := r.getApps(start, limit, params.Text, appType, "", dbmodel.SortDirAny)
	if err != nil {
		log.Error(err)
		msg := "cannot get apps from db"
		rsp := services.NewGetAppsDefault(http.StatusInternalServerError).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}
	rsp := services.NewGetAppsOK().WithPayload(apps)
	return rsp
}

func (r *RestAPI) GetApp(ctx context.Context, params services.GetAppParams) middleware.Responder {
	dbApp, err := dbmodel.GetAppByID(r.DB, params.ID)
	if err != nil {
		msg := fmt.Sprintf("cannot get app with id %d from db", params.ID)
		log.Error(err)
		rsp := services.NewGetAppDefault(http.StatusInternalServerError).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}
	if dbApp == nil {
		msg := fmt.Sprintf("cannot find app with id %d", params.ID)
		rsp := services.NewGetAppDefault(http.StatusNotFound).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}

	var a *models.App
	if dbApp.Type == dbmodel.AppTypeBind9 {
		a = r.appToRestAPI(dbApp)
	} else if dbApp.Type == dbmodel.AppTypeKea {
		a = r.appToRestAPI(dbApp)
	}
	rsp := services.NewGetAppOK().WithPayload(a)
	return rsp
}

// Gets current status of services for a given Kea application.
func getKeaServicesStatus(db *dbops.PgDB, app *dbmodel.App) *models.ServicesStatus {
	servicesStatus := &models.ServicesStatus{}

	keaServices, err := dbmodel.GetDetailedServicesByAppID(db, app.ID)
	if err != nil {
		log.Error(err)

		return nil
	}

	for _, s := range keaServices {
		if s.HAService == nil {
			continue
		}
		ha := s.HAService
		keaStatus := models.KeaStatus{
			Daemon: ha.HAType,
		}
		secondaryRole := "secondary"
		if ha.HAMode == "hot-standby" {
			secondaryRole = "standby"
		}
		// Calculate age.
		age := make([]int64, 2)
		statusTime := make([]strfmt.DateTime, 2)
		now := storkutil.UTCNow()
		for i, t := range []time.Time{ha.PrimaryStatusCollectedAt, ha.SecondaryStatusCollectedAt} {
			// If status time hasn't been set yet, return a negative age value to
			// indicate that it cannot be displayed.
			if t.IsZero() || now.Before(t) {
				age[i] = -1
			} else {
				age[i] = int64(now.Sub(t).Seconds())
				statusTime[i] = strfmt.DateTime(t)
			}
		}
		// Format failover times into string.
		failoverTime := make([]strfmt.DateTime, 2)
		for i, t := range []time.Time{ha.PrimaryLastFailoverAt, ha.SecondaryLastFailoverAt} {
			// Only display the non-zero failover times and the times that are
			// before current time.
			if !t.IsZero() && now.After(t) {
				failoverTime[i] = strfmt.DateTime(t)
			}
		}
		// Get the control addresses and app ids for daemons taking part in HA.
		controlAddress := make([]string, 2)
		appID := make([]int64, 2)
		for i := range s.Daemons {
			if s.Daemons[i].ID == ha.PrimaryID {
				ap, _ := s.Daemons[i].App.GetAccessPoint("control")
				if ap != nil {
					controlAddress[0] = ap.Address
				}
				appID[0] = s.Daemons[i].App.ID
			} else if s.Daemons[i].ID == ha.SecondaryID {
				ap, _ := s.Daemons[i].App.GetAccessPoint("control")
				if ap != nil {
					controlAddress[1] = ap.Address
				}
				appID[1] = s.Daemons[i].App.ID
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
			PrimaryServer: &models.KeaHAServerStatus{
				Age:                age[0],
				AppID:              appID[0],
				ControlAddress:     controlAddress[0],
				FailoverTime:       failoverTime[0],
				ID:                 ha.PrimaryID,
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
				AppID:              appID[1],
				ControlAddress:     controlAddress[1],
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

// Gets current status of services which the given application is associated with.
func (r *RestAPI) GetAppServicesStatus(ctx context.Context, params services.GetAppServicesStatusParams) middleware.Responder {
	dbApp, err := dbmodel.GetAppByID(r.DB, params.ID)
	if err != nil {
		log.Error(err)
		msg := fmt.Sprintf("cannot get app with id %d from the database", params.ID)
		rsp := services.NewGetAppServicesStatusDefault(http.StatusInternalServerError).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}

	if dbApp == nil {
		msg := fmt.Sprintf("cannot find app with id %d", params.ID)
		log.Warn(errors.New(msg))
		rsp := services.NewGetAppDefault(http.StatusNotFound).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}

	var servicesStatus *models.ServicesStatus

	// If this is Kea application, get the Kea DHCP servers status which possibly
	// includes HA status.
	if dbApp.Type == dbmodel.AppTypeKea {
		servicesStatus = getKeaServicesStatus(r.DB, dbApp)
		if servicesStatus == nil {
			msg := fmt.Sprintf("cannot get status of the app with id %d", dbApp.ID)
			rsp := services.NewGetAppServicesStatusDefault(http.StatusInternalServerError).WithPayload(&models.APIError{
				Message: &msg,
			})
			return rsp
		}
	} else {
		servicesStatus = &models.ServicesStatus{}
	}

	rsp := services.NewGetAppServicesStatusOK().WithPayload(servicesStatus)
	return rsp
}

// Get statistics about applications.
func (r *RestAPI) GetAppsStats(ctx context.Context, params services.GetAppsStatsParams) middleware.Responder {
	dbApps, err := dbmodel.GetAllApps(r.DB)
	if err != nil {
		log.Error(err)
		msg := "cannot get all apps from db"
		rsp := services.NewGetAppsStatsDefault(http.StatusInternalServerError).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}

	appsStats := models.AppsStats{
		KeaAppsTotal:   0,
		KeaAppsNotOk:   0,
		Bind9AppsTotal: 0,
		Bind9AppsNotOk: 0,
	}
	for _, dbApp := range dbApps {
		switch dbApp.Type {
		case dbmodel.AppTypeKea:
			appsStats.KeaAppsTotal++
			if !dbApp.Active {
				appsStats.KeaAppsNotOk++
			}
		case dbmodel.AppTypeBind9:
			appsStats.Bind9AppsTotal++
			if !dbApp.Active {
				appsStats.Bind9AppsNotOk++
			}
		}
	}

	rsp := services.NewGetAppsStatsOK().WithPayload(&appsStats)
	return rsp
}

// Get DHCP overview.
func (r *RestAPI) GetDhcpOverview(ctx context.Context, params dhcp.GetDhcpOverviewParams) middleware.Responder {
	// get list of mostly utilized subnets
	subnets4, err := r.getSubnets(0, 5, 0, 4, nil, "addr_utilization", dbmodel.SortDirDesc)
	if err != nil {
		log.Error(err)
		msg := "cannot get IPv4 subnets from the db"
		rsp := dhcp.NewGetDhcpOverviewDefault(http.StatusInternalServerError).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}

	subnets6, err := r.getSubnets(0, 5, 0, 6, nil, "addr_utilization", dbmodel.SortDirDesc)
	if err != nil {
		log.Error(err)
		msg := "cannot get IPv6 subnets from the db"
		rsp := dhcp.NewGetDhcpOverviewDefault(http.StatusInternalServerError).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}

	// get list of mostly utilized shared networks
	sharedNetworks4, err := r.getSharedNetworks(0, 5, 0, 4, nil, "addr_utilization", dbmodel.SortDirDesc)
	if err != nil {
		log.Error(err)
		msg := "cannot get IPv4 shared networks from the db"
		rsp := dhcp.NewGetDhcpOverviewDefault(http.StatusInternalServerError).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}

	sharedNetworks6, err := r.getSharedNetworks(0, 5, 0, 6, nil, "addr_utilization", dbmodel.SortDirDesc)
	if err != nil {
		log.Error(err)
		msg := "cannot get IPv6 shared networks from the db"
		rsp := dhcp.NewGetDhcpOverviewDefault(http.StatusInternalServerError).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}

	// get dhcp statistics
	stats, err := dbmodel.GetAllStats(r.DB)
	if err != nil {
		log.Error(err)
		msg := "cannot get statistics from db"
		rsp := dhcp.NewGetDhcpOverviewDefault(http.StatusInternalServerError).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}

	dhcp4Stats := &models.Dhcp4Stats{
		AssignedAddresses: stats["assigned-addresses"],
		TotalAddresses:    stats["total-addresses"],
		DeclinedAddresses: stats["declined-addresses"],
	}
	dhcp6Stats := &models.Dhcp6Stats{
		AssignedNAs: stats["assigned-nas"],
		TotalNAs:    stats["total-nas"],
		AssignedPDs: stats["assigned-pds"],
		TotalPDs:    stats["total-pds"],
		DeclinedNAs: stats["declined-nas"],
	}

	// get kea apps and daemons statuses
	dbApps, err := dbmodel.GetAppsByType(r.DB, dbmodel.AppTypeKea)
	if err != nil {
		log.Error(err)
		msg := "cannot get statistics from db"
		rsp := dhcp.NewGetDhcpOverviewDefault(http.StatusInternalServerError).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}

	var dhcpDaemons []*models.DhcpDaemon
	for _, dbApp := range dbApps {
		for _, dbDaemon := range dbApp.Daemons {
			if !strings.HasPrefix(dbDaemon.Name, "dhcp") {
				continue
			}
			if !dbDaemon.Monitored {
				// do not show not monitored daemons (ie. show only monitored services)
				continue
			}
			// todo: Currently Kea supports only one HA relationship per daemon.
			// Until we extend Kea to support multiple relationships per daemon
			// or integrate ISC DHCP with Stork, the number of HA states returned
			// will be 0 or 1. Therefore, we take the first HA state if it exists
			// and return it over the REST API.
			var (
				haEnabled   bool
				haState     string
				haFailureAt strfmt.DateTime
			)
			if haOverview := dbDaemon.GetHAOverview(); len(haOverview) > 0 {
				haEnabled = true
				haState = haOverview[0].State
				if !haOverview[0].LastFailureAt.IsZero() {
					haFailureAt = strfmt.DateTime(haOverview[0].LastFailureAt)
				}
			}
			agentErrors := int64(0)
			caErrors := int64(0)
			daemonErrors := int64(0)
			agentStats := r.Agents.GetConnectedAgentStats(dbApp.Machine.Address, dbApp.Machine.AgentPort)
			if agentStats != nil {
				agentErrors = agentStats.CurrentErrors
				accessPoint, _ := dbApp.GetAccessPoint(dbmodel.AccessPointControl)
				if accessPoint != nil {
					if keaStats, ok := agentStats.AppCommStats[agentcomm.AppCommStatsKey{
						Address: accessPoint.Address,
						Port:    accessPoint.Port,
					}].(*agentcomm.AgentKeaCommStats); ok {
						caErrors = keaStats.CurrentErrorsCA
						daemonErrors = keaStats.CurrentErrorsDaemons[dbDaemon.Name]
					}
				}
			}
			daemon := &models.DhcpDaemon{
				MachineID:        dbApp.MachineID,
				Machine:          dbApp.Machine.State.Hostname,
				AppVersion:       dbApp.Meta.Version,
				AppID:            dbApp.ID,
				AppName:          dbApp.Name,
				Name:             dbDaemon.Name,
				Active:           dbDaemon.Active,
				Monitored:        dbDaemon.Monitored,
				Rps1:             int64(dbDaemon.KeaDaemon.KeaDHCPDaemon.Stats.RPS1),
				Rps2:             int64(dbDaemon.KeaDaemon.KeaDHCPDaemon.Stats.RPS2),
				AddrUtilization:  0,
				HaEnabled:        haEnabled,
				HaState:          haState,
				HaFailureAt:      haFailureAt,
				Uptime:           dbDaemon.Uptime,
				AgentCommErrors:  agentErrors,
				CaCommErrors:     caErrors,
				DaemonCommErrors: daemonErrors,
			}
			dhcpDaemons = append(dhcpDaemons, daemon)
		}
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
		log.Error(err)
		msg := fmt.Sprintf("cannot get daemon with id %d from db", params.ID)
		rsp := services.NewUpdateDaemonDefault(http.StatusInternalServerError).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}
	if dbDaemon == nil {
		msg := fmt.Sprintf("cannot find daemon with id %d", params.ID)
		rsp := services.NewUpdateDaemonDefault(http.StatusNotFound).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}

	oldMonitored := dbDaemon.Monitored

	dbDaemon.Monitored = params.Daemon.Monitored

	err = dbmodel.UpdateDaemon(r.DB, dbDaemon)
	if err != nil {
		msg := fmt.Sprintf("failed updating daemon with id %d", params.ID)
		rsp := services.NewUpdateDaemonDefault(http.StatusInternalServerError).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}

	_, dbUser := r.SessionManager.Logged(ctx)

	if oldMonitored != params.Daemon.Monitored {
		if params.Daemon.Monitored {
			r.EventCenter.AddInfoEvent("{user} enabled monitoring {daemon}", dbUser, dbDaemon, dbDaemon.App, dbDaemon.App.Machine)
		} else {
			r.EventCenter.AddWarningEvent("{user} disabled monitoring {daemon}", dbUser, dbDaemon, dbDaemon.App, dbDaemon.App.Machine)
		}
	}

	rsp := services.NewUpdateDaemonOK()
	return rsp
}
