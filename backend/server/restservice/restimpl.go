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
	"isc.org/stork/server/agentcomm"
	"isc.org/stork/server/apps/bind9"
	"isc.org/stork/server/apps/kea"
	dbops "isc.org/stork/server/database"
	dbmodel "isc.org/stork/server/database/model"
	"isc.org/stork/server/gen/models"
	dhcp "isc.org/stork/server/gen/restapi/operations/d_h_c_p"
	"isc.org/stork/server/gen/restapi/operations/general"
	"isc.org/stork/server/gen/restapi/operations/services"
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

func machineToRestAPI(dbMachine dbmodel.Machine) *models.Machine {
	var apps []*models.MachineApp
	for _, app := range dbMachine.Apps {
		active := true
		if app.Type == dbmodel.AppTypeKea {
			if app.Active {
				for _, d := range app.Details.(dbmodel.AppKea).Daemons {
					if !d.Active {
						active = false
						break
					}
				}
			} else {
				active = false
			}
		}
		s := models.MachineApp{
			ID:      app.ID,
			Type:    app.Type,
			Version: app.Meta.Version,
			Active:  active,
		}
		apps = append(apps, &s)
	}

	m := models.Machine{
		ID:                   dbMachine.ID,
		Address:              &dbMachine.Address,
		AgentPort:            dbMachine.AgentPort,
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

// appCompare compares two apps on equality.  Two apps are considered equal if
// their type matches and if they have the same control port.  Return true if
// equal, false otherwise.
func appCompare(dbApp *dbmodel.App, app *agentcomm.App) bool {
	if dbApp.Type != app.Type {
		return false
	}

	var controlPortEqual bool
	for _, pt1 := range dbApp.AccessPoints {
		if pt1.Type != dbmodel.AccessPointControl {
			continue
		}
		for _, pt2 := range app.AccessPoints {
			if pt2.Type != dbmodel.AccessPointControl {
				continue
			}

			if pt1.Port == pt2.Port {
				controlPortEqual = true
				break
			}
		}

		// If a match is found, we can break.
		if controlPortEqual {
			break
		}
	}

	return controlPortEqual
}

func getMachineAndAppsState(ctx context.Context, db *dbops.PgDB, dbMachine *dbmodel.Machine, agents agentcomm.ConnectedAgents) string {
	ctx2, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	// get state of machine from agent
	state, err := agents.GetState(ctx2, dbMachine.Address, dbMachine.AgentPort)
	if err != nil {
		log.Warn(err)
		dbMachine.Error = "cannot get state of machine"
		err = db.Update(dbMachine)
		if err != nil {
			log.Error(err)
			return "problem with updating record in database"
		}
		return ""
	}

	// store machine's state in db
	err = updateMachineFields(db, dbMachine, state)
	if err != nil {
		log.Error(err)
		return "cannot update machine in db"
	}

	// If there are any new apps then get their state and add to db.
	// Old ones are just updated.
	oldAppsList := dbMachine.Apps
	dbMachine.Apps = []*dbmodel.App{}
	for _, app := range state.Apps {
		// look for old app
		var dbApp *dbmodel.App = nil
		for _, dbApp2 := range oldAppsList {
			if appCompare(dbApp2, app) {
				dbApp = dbApp2
				break
			}
		}
		// if no old app in db then prepare new record
		if dbApp == nil {
			var accessPoints []*dbmodel.AccessPoint

			for _, point := range app.AccessPoints {
				accessPoints = append(accessPoints, &dbmodel.AccessPoint{
					Type:    point.Type,
					Address: point.Address,
					Port:    point.Port,
					Key:     point.Key,
				})
			}

			dbApp = &dbmodel.App{
				ID:           0,
				MachineID:    dbMachine.ID,
				Machine:      dbMachine,
				Type:         app.Type,
				AccessPoints: accessPoints,
			}
		} else {
			dbApp.Machine = dbMachine
		}

		switch app.Type {
		case dbmodel.AppTypeKea:
			kea.GetAppState(ctx2, agents, dbApp)
			err = kea.CommitAppIntoDB(db, dbApp)
		case dbmodel.AppTypeBind9:
			bind9.GetAppState(ctx2, agents, dbApp)
			err = bind9.CommitAppIntoDB(db, dbApp)
		default:
			err = nil
		}

		if err != nil {
			log.Error(err)
			return "problem with storing application state in the database"
		}

		log.Printf("committed information about %s app running on %s to database",
			dbApp.Type, dbMachine.Address)

		// add app to machine's apps list
		dbMachine.Apps = append(dbMachine.Apps, dbApp)
	}

	// delete missing apps
	for _, dbApp := range oldAppsList {
		found := false
		for _, app := range state.Apps {
			if appCompare(dbApp, app) {
				found = true
				break
			}
		}
		if !found {
			err = dbmodel.DeleteApp(db, dbApp)
			if err != nil {
				log.Error(err)
			}
			log.Printf("deleted %s app on %s", dbApp.Type, dbMachine.Address)
		}
	}

	return ""
}

// Get runtime state of indicated machine.
func (r *RestAPI) GetMachineState(ctx context.Context, params services.GetMachineStateParams) middleware.Responder {
	dbMachine, err := dbmodel.GetMachineByID(r.Db, params.ID)
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

	errStr := getMachineAndAppsState(ctx, r.Db, dbMachine, r.Agents)
	if errStr != "" {
		rsp := services.NewGetMachineStateDefault(http.StatusInternalServerError).WithPayload(&models.APIError{
			Message: &errStr,
		})
		return rsp
	}

	m := machineToRestAPI(*dbMachine)
	rsp := services.NewGetMachineStateOK().WithPayload(m)

	return rsp
}

// Get machines where Stork Agent is running.
func (r *RestAPI) GetMachines(ctx context.Context, params services.GetMachinesParams) middleware.Responder {
	machines := []*models.Machine{}

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

	dbMachines, total, err := dbmodel.GetMachinesByPage(r.Db, start, limit, text)
	if err != nil {
		log.Error(err)
		msg := "cannot get machines from db"
		rsp := services.NewGetMachinesDefault(http.StatusInternalServerError).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}

	for _, dbM := range dbMachines {
		mm := machineToRestAPI(dbM)
		machines = append(machines, mm)
	}

	m := models.Machines{
		Items: machines,
		Total: total,
	}
	rsp := services.NewGetMachinesOK().WithPayload(&m)
	return rsp
}

// Get one machine by ID where Stork Agent is running.
func (r *RestAPI) GetMachine(ctx context.Context, params services.GetMachineParams) middleware.Responder {
	dbMachine, err := dbmodel.GetMachineByID(r.Db, params.ID)
	if err != nil {
		msg := fmt.Sprintf("cannot get machine with id %d from db", params.ID)
		log.Error(err)
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
	m := machineToRestAPI(*dbMachine)
	rsp := services.NewGetMachineOK().WithPayload(m)
	return rsp
}

// Add a machine where Stork Agent is running.
func (r *RestAPI) CreateMachine(ctx context.Context, params services.CreateMachineParams) middleware.Responder {
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

	dbMachine, err := dbmodel.GetMachineByAddressAndAgentPort(r.Db, addr, params.Machine.AgentPort)
	if err == nil && dbMachine != nil {
		msg := fmt.Sprintf("machine %s:%d already exists", addr, params.Machine.AgentPort)
		log.Warnf(msg)
		rsp := services.NewCreateMachineDefault(http.StatusBadRequest).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}

	if dbMachine == nil {
		dbMachine = &dbmodel.Machine{Address: addr, AgentPort: params.Machine.AgentPort}
		err = dbmodel.AddMachine(r.Db, dbMachine)
		if err != nil {
			msg := fmt.Sprintf("cannot store machine %s", addr)
			log.Error(err)
			rsp := services.NewCreateMachineDefault(http.StatusInternalServerError).WithPayload(&models.APIError{
				Message: &msg,
			})
			return rsp
		}
	}

	errStr := getMachineAndAppsState(ctx, r.Db, dbMachine, r.Agents)
	if errStr != "" {
		rsp := services.NewCreateMachineDefault(http.StatusInternalServerError).WithPayload(&models.APIError{
			Message: &errStr,
		})
		return rsp
	}

	m := machineToRestAPI(*dbMachine)
	log.Printf("machineToRestAPI  %+v", m)
	rsp := services.NewCreateMachineOK().WithPayload(m)

	return rsp
}

// Get one machine by ID where Stork Agent is running.
func (r *RestAPI) UpdateMachine(ctx context.Context, params services.UpdateMachineParams) middleware.Responder {
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

	dbMachine, err := dbmodel.GetMachineByID(r.Db, params.ID)
	if err != nil {
		msg := fmt.Sprintf("cannot get machine with id %d from db", params.ID)
		log.Error(err)
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
		dbMachine2, err := dbmodel.GetMachineByAddressAndAgentPort(r.Db, addr, params.Machine.AgentPort)
		if err == nil && dbMachine2 != nil && dbMachine2.ID != dbMachine.ID {
			msg := fmt.Sprintf("machine with address %s:%d already exists",
				*params.Machine.Address, params.Machine.AgentPort)
			rsp := services.NewUpdateMachineDefault(http.StatusBadRequest).WithPayload(&models.APIError{
				Message: &msg,
			})
			return rsp
		}
	}

	// copy fields
	dbMachine.Address = addr
	dbMachine.AgentPort = params.Machine.AgentPort
	err = r.Db.Update(dbMachine)
	if err != nil {
		msg := fmt.Sprintf("cannot update machine with id %d in db", params.ID)
		log.Error(err)
		rsp := services.NewUpdateMachineDefault(http.StatusInternalServerError).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}
	m := machineToRestAPI(*dbMachine)
	rsp := services.NewUpdateMachineOK().WithPayload(m)
	return rsp
}

func updateMachineFields(db *dbops.PgDB, dbMachine *dbmodel.Machine, m *agentcomm.State) error {
	// update state fields in machine
	dbMachine.State.AgentVersion = m.AgentVersion
	dbMachine.State.Cpus = m.Cpus
	dbMachine.State.CpusLoad = m.CpusLoad
	dbMachine.State.Memory = m.Memory
	dbMachine.State.Hostname = m.Hostname
	dbMachine.State.Uptime = m.Uptime
	dbMachine.State.UsedMemory = m.UsedMemory
	dbMachine.State.Os = m.Os
	dbMachine.State.Platform = m.Platform
	dbMachine.State.PlatformFamily = m.PlatformFamily
	dbMachine.State.PlatformVersion = m.PlatformVersion
	dbMachine.State.KernelVersion = m.KernelVersion
	dbMachine.State.KernelArch = m.KernelArch
	dbMachine.State.VirtualizationSystem = m.VirtualizationSystem
	dbMachine.State.VirtualizationRole = m.VirtualizationRole
	dbMachine.State.HostID = m.HostID
	dbMachine.LastVisitedAt = m.LastVisitedAt
	dbMachine.Error = m.Error
	err := db.Update(dbMachine)
	if err != nil {
		return errors.Wrapf(err, "problem with updating machine %+v", dbMachine)
	}
	return nil
}

// Add a machine where Stork Agent is running.
func (r *RestAPI) DeleteMachine(ctx context.Context, params services.DeleteMachineParams) middleware.Responder {
	dbMachine, err := dbmodel.GetMachineByID(r.Db, params.ID)
	if err == nil && dbMachine == nil {
		rsp := services.NewDeleteMachineOK()
		return rsp
	} else if err != nil {
		msg := fmt.Sprintf("cannot delete machine %d", params.ID)
		log.Error(err)
		rsp := services.NewDeleteMachineDefault(http.StatusInternalServerError).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}

	err = dbmodel.DeleteMachine(r.Db, dbMachine)
	if err != nil {
		msg := fmt.Sprintf("cannot delete machine %d", params.ID)
		log.Error(err)
		rsp := services.NewDeleteMachineDefault(http.StatusInternalServerError).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}

	rsp := services.NewDeleteMachineOK()

	return rsp
}

func appToRestAPI(dbApp *dbmodel.App) *models.App {
	app := models.App{
		ID:      dbApp.ID,
		Type:    dbApp.Type,
		Active:  dbApp.Active,
		Version: dbApp.Meta.Version,
		Machine: &models.AppMachine{
			ID:       dbApp.MachineID,
			Address:  dbApp.Machine.Address,
			Hostname: dbApp.Machine.State.Hostname,
		},
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

	if isKeaApp {
		var keaDaemons []*models.KeaDaemon
		for _, d := range dbApp.Details.(dbmodel.AppKea).Daemons {
			dmn := &models.KeaDaemon{
				Pid:             int64(d.Pid),
				Name:            d.Name,
				Active:          d.Active,
				Version:         d.Version,
				ExtendedVersion: d.ExtendedVersion,
				Uptime:          d.Uptime,
				ReloadedAt:      strfmt.DateTime(d.ReloadedAt),
				Hooks:           []string{},
			}
			hooksByDaemon := kea.GetDaemonHooks(dbApp)
			if hooksByDaemon != nil {
				hooksList, ok := hooksByDaemon[d.Name]
				if ok {
					dmn.Hooks = hooksList
				}
			}
			keaDaemons = append(keaDaemons, dmn)
		}

		app.Details = struct {
			models.AppKea
			models.AppBind9
		}{
			models.AppKea{
				ExtendedVersion: dbApp.Details.(dbmodel.AppKea).ExtendedVersion,
				Daemons:         keaDaemons,
			},
			models.AppBind9{},
		}
	}

	if isBind9App {
		bind9Daemon := &models.Bind9Daemon{
			Pid:           int64(dbApp.Details.(dbmodel.AppBind9).Daemon.Pid),
			Name:          dbApp.Details.(dbmodel.AppBind9).Daemon.Name,
			Active:        dbApp.Details.(dbmodel.AppBind9).Daemon.Active,
			Version:       dbApp.Details.(dbmodel.AppBind9).Daemon.Version,
			Uptime:        dbApp.Details.(dbmodel.AppBind9).Daemon.Uptime,
			ReloadedAt:    strfmt.DateTime(dbApp.Details.(dbmodel.AppBind9).Daemon.ReloadedAt),
			ZoneCount:     dbApp.Details.(dbmodel.AppBind9).Daemon.ZoneCount,
			AutoZoneCount: dbApp.Details.(dbmodel.AppBind9).Daemon.AutomaticZoneCount,
			CacheHits:     dbApp.Details.(dbmodel.AppBind9).Daemon.CacheHits,
			CacheMisses:   dbApp.Details.(dbmodel.AppBind9).Daemon.CacheMisses,
			CacheHitRatio: dbApp.Details.(dbmodel.AppBind9).Daemon.CacheHitRatio,
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

func (r *RestAPI) GetApps(ctx context.Context, params services.GetAppsParams) middleware.Responder {
	appsLst := []*models.App{}

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
	}).Info("query apps")

	dbApps, total, err := dbmodel.GetAppsByPage(r.Db, start, limit, text, app)
	if err != nil {
		log.Error(err)
		msg := "cannot get apps from db"
		rsp := services.NewGetAppsDefault(http.StatusInternalServerError).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}

	for _, dbA := range dbApps {
		app := dbA
		a := appToRestAPI(&app)
		appsLst = append(appsLst, a)
	}

	a := models.Apps{
		Items: appsLst,
		Total: total,
	}
	rsp := services.NewGetAppsOK().WithPayload(&a)
	return rsp
}

func (r *RestAPI) GetApp(ctx context.Context, params services.GetAppParams) middleware.Responder {
	dbApp, err := dbmodel.GetAppByID(r.Db, params.ID)
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
		a = appToRestAPI(dbApp)
	} else if dbApp.Type == dbmodel.AppTypeKea {
		a = appToRestAPI(dbApp)
	}
	rsp := services.NewGetAppOK().WithPayload(a)
	return rsp
}

// Gets current status of services which the given application is associated with.
func (r *RestAPI) GetAppServicesStatus(ctx context.Context, params services.GetAppServicesStatusParams) middleware.Responder {
	dbApp, err := dbmodel.GetAppByID(r.Db, params.ID)
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

	servicesStatus := &models.ServicesStatus{}

	// If this is Kea application, get the Kea DHCP servers status which possibly
	// includes HA status.
	if dbApp.Type == dbmodel.AppTypeKea {
		keaServices, err := dbmodel.GetDetailedServicesByAppID(r.Db, dbApp.ID)
		if err != nil {
			log.Error(err)
			msg := fmt.Sprintf("cannot get status of the app with id %d", params.ID)
			rsp := services.NewGetAppServicesStatusDefault(http.StatusInternalServerError).WithPayload(&models.APIError{
				Message: &msg,
			})
			return rsp
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
			now := time.Now().UTC()
			for i, t := range []time.Time{ha.PrimaryStatusCollectedAt, ha.SecondaryStatusCollectedAt} {
				// If status time hasn't been set yet, return a negative age value to
				// indicate that it cannot be displayed.
				if t.IsZero() || now.Before(t) {
					age[i] = -1
				} else {
					age[i] = int64(now.Sub(t).Seconds())
				}
			}
			// Format failover times into string.
			failoverTime := make([]string, 2)
			for i, t := range []time.Time{ha.PrimaryLastFailoverAt, ha.SecondaryLastFailoverAt} {
				// Only display the non-zero failover times and the times that are
				// before current time.
				if !t.IsZero() && now.After(t) {
					failoverTime[i] = t.Format(time.UnixDate)
				}
			}
			keaStatus.HaServers = &models.KeaStatusHaServers{
				PrimaryServer: &models.KeaStatusHaServersPrimaryServer{
					ID:           ha.PrimaryID,
					Age:          age[0],
					InTouch:      ha.PrimaryReachable,
					Role:         "primary",
					Scopes:       ha.PrimaryLastScopes,
					State:        ha.PrimaryLastState,
					FailoverTime: failoverTime[0],
				},
				SecondaryServer: &models.KeaStatusHaServersSecondaryServer{
					ID:           ha.SecondaryID,
					Age:          age[1],
					InTouch:      ha.SecondaryReachable,
					Role:         secondaryRole,
					Scopes:       ha.SecondaryLastScopes,
					State:        ha.SecondaryLastState,
					FailoverTime: failoverTime[1],
				},
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
	}

	rsp := services.NewGetAppServicesStatusOK().WithPayload(servicesStatus)
	return rsp
}

// Get statistics about applications.
func (r *RestAPI) GetAppsStats(ctx context.Context, params services.GetAppsStatsParams) middleware.Responder {
	dbApps, err := dbmodel.GetAllApps(r.Db)
	if err != nil {
		msg := fmt.Sprintf("cannot get all apps from db")
		log.Error(err)
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
		msg := "cannot get IPv4 subnets from the db"
		log.Error(err)
		rsp := dhcp.NewGetDhcpOverviewDefault(http.StatusInternalServerError).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}

	subnets6, err := r.getSubnets(0, 5, 0, 6, nil, "addr_utilization", dbmodel.SortDirDesc)
	if err != nil {
		msg := "cannot get IPv6 subnets from the db"
		log.Error(err)
		rsp := dhcp.NewGetDhcpOverviewDefault(http.StatusInternalServerError).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}

	// get list of mostly utilized shared networks
	sharedNetworks4, err := r.getSharedNetworks(0, 5, 0, 4, nil, "addr_utilization", dbmodel.SortDirDesc)
	if err != nil {
		msg := "cannot get IPv4 shared networks from the db"
		log.Error(err)
		rsp := dhcp.NewGetDhcpOverviewDefault(http.StatusInternalServerError).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}

	sharedNetworks6, err := r.getSharedNetworks(0, 5, 0, 6, nil, "addr_utilization", dbmodel.SortDirDesc)
	if err != nil {
		msg := "cannot get IPv6 shared networks from the db"
		log.Error(err)
		rsp := dhcp.NewGetDhcpOverviewDefault(http.StatusInternalServerError).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}

	// get dhcp statistics
	stats, err := dbmodel.GetAllStats(r.Db)
	if err != nil {
		msg := "cannot get statistics from db"
		log.Error(err)
		rsp := dhcp.NewGetDhcpOverviewDefault(http.StatusInternalServerError).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}

	dhcp4Stats := &models.Dhcp4Stats{
		AssignedAddresses: stats["assigned-addreses"],
		TotalAddresses:    stats["total-addreses"],
		DeclinedAddresses: stats["declined-addreses"],
	}
	dhcp6Stats := &models.Dhcp6Stats{
		AssignedNAs: stats["assigned-nas"],
		TotalNAs:    stats["total-nas"],
		AssignedPDs: stats["assigned-pds"],
		TotalPDs:    stats["total-pds"],
		DeclinedNAs: stats["declined-nas"],
	}

	// get kea apps and daemons statuses
	dbApps, err := dbmodel.GetAppsByType(r.Db, dbmodel.AppTypeKea)
	if err != nil {
		msg := "cannot get statistics from db"
		log.Error(err)
		rsp := dhcp.NewGetDhcpOverviewDefault(http.StatusInternalServerError).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}

	var dhcpDaemons []*models.DhcpDaemon
	for _, dbApp := range dbApps {
		for _, dbDaemon := range dbApp.Details.(dbmodel.AppKea).Daemons {
			if !strings.HasPrefix(dbDaemon.Name, "dhcp") {
				continue
			}
			daemon := &models.DhcpDaemon{
				MachineID:       dbApp.MachineID,
				Machine:         dbApp.Machine.State.Hostname,
				AppVersion:      dbApp.Meta.Version,
				AppID:           dbApp.ID,
				Name:            dbDaemon.Name,
				Active:          dbDaemon.Active,
				Lps15min:        0,
				Lps24h:          0,
				AddrUtilization: 0,
				HaState:         "load-balancing",
				Uptime:          dbDaemon.Uptime,
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
