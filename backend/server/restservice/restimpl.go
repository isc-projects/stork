package restservice

import (
	"context"
	"fmt"
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
		if app.Type == dbmodel.KeaAppType {
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
		LastVisited:          strfmt.DateTime(dbMachine.LastVisited),
		Error:                dbMachine.Error,
		Apps:                 apps,
	}
	return &m
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
			if dbApp2.Type == app.Type && dbApp2.CtrlPort == app.CtrlPort {
				dbApp = dbApp2
				break
			}
		}
		// if no old app in db then prepare new record
		if dbApp == nil {
			dbApp = &dbmodel.App{
				ID:          0,
				MachineID:   dbMachine.ID,
				Machine:     dbMachine,
				Type:        app.Type,
				CtrlAddress: app.CtrlAddress,
				CtrlPort:    app.CtrlPort,
			}
		} else {
			dbApp.Machine = dbMachine
		}

		switch app.Type {
		case dbmodel.KeaAppType:
			kea.GetAppState(ctx2, agents, dbApp)
		case dbmodel.Bind9AppType:
			bind9.GetAppState(ctx2, agents, dbApp)
		}

		// either add new app record to db or update old one
		if dbApp.ID == 0 {
			err = dbmodel.AddApp(db, dbApp)
			if err != nil {
				log.Error(err)
				return "problem with storing application state in database"
			}
			log.Printf("added %s app on %s", dbApp.Type, dbMachine.Address)
		} else {
			err = db.Update(dbApp)
			if err != nil {
				log.Error(err)
				return "problem with storing application state in database"
			}
			log.Printf("updated %s app on %s", dbApp.Type, dbMachine.Address)
		}

		// add app to machine's apps list
		dbMachine.Apps = append(dbMachine.Apps, dbApp)
	}

	// delete missing apps
	for _, dbApp := range oldAppsList {
		found := false
		for _, app := range state.Apps {
			if dbApp.Type == app.Type && dbApp.CtrlPort == app.CtrlPort {
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
		rsp := services.NewGetMachineStateDefault(500).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}
	if dbMachine == nil {
		msg := fmt.Sprintf("cannot find machine with id %d", params.ID)
		rsp := services.NewGetMachineStateDefault(404).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}

	errStr := getMachineAndAppsState(ctx, r.Db, dbMachine, r.Agents)
	if errStr != "" {
		rsp := services.NewGetMachineStateDefault(500).WithPayload(&models.APIError{
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
		rsp := services.NewGetMachinesDefault(500).WithPayload(&models.APIError{
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
		rsp := services.NewGetMachineDefault(500).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}
	if dbMachine == nil {
		msg := fmt.Sprintf("cannot find machine with id %d", params.ID)
		rsp := services.NewGetMachineDefault(404).WithPayload(&models.APIError{
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
		rsp := services.NewCreateMachineDefault(400).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}
	if params.Machine.AgentPort <= 0 || params.Machine.AgentPort > 65535 {
		log.Warnf("bad agent port %d", params.Machine.AgentPort)
		msg := "bad port"
		rsp := services.NewCreateMachineDefault(400).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}

	dbMachine, err := dbmodel.GetMachineByAddressAndAgentPort(r.Db, addr, params.Machine.AgentPort, true)
	if err == nil && dbMachine != nil && dbMachine.Deleted.IsZero() {
		msg := fmt.Sprintf("machine %s:%d already exists", addr, params.Machine.AgentPort)
		log.Warnf(msg)
		rsp := services.NewCreateMachineDefault(400).WithPayload(&models.APIError{
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
			rsp := services.NewCreateMachineDefault(500).WithPayload(&models.APIError{
				Message: &msg,
			})
			return rsp
		}
	} else {
		dbMachine.Deleted = time.Time{}
	}

	errStr := getMachineAndAppsState(ctx, r.Db, dbMachine, r.Agents)
	if errStr != "" {
		rsp := services.NewCreateMachineDefault(500).WithPayload(&models.APIError{
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
		rsp := services.NewUpdateMachineDefault(400).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}
	if params.Machine.AgentPort <= 0 || params.Machine.AgentPort > 65535 {
		log.Warnf("bad agent port %d", params.Machine.AgentPort)
		msg := "bad port"
		rsp := services.NewUpdateMachineDefault(400).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}

	dbMachine, err := dbmodel.GetMachineByID(r.Db, params.ID)
	if err != nil {
		msg := fmt.Sprintf("cannot get machine with id %d from db", params.ID)
		log.Error(err)
		rsp := services.NewUpdateMachineDefault(500).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}
	if dbMachine == nil {
		msg := fmt.Sprintf("cannot find machine with id %d", params.ID)
		rsp := services.NewUpdateMachineDefault(404).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}

	// check if there is no duplicate
	if dbMachine.Address != addr || dbMachine.AgentPort != params.Machine.AgentPort {
		dbMachine2, err := dbmodel.GetMachineByAddressAndAgentPort(r.Db, addr, params.Machine.AgentPort, false)
		if err == nil && dbMachine2 != nil && dbMachine2.ID != dbMachine.ID {
			msg := fmt.Sprintf("machine with address %s:%d already exists",
				*params.Machine.Address, params.Machine.AgentPort)
			rsp := services.NewUpdateMachineDefault(400).WithPayload(&models.APIError{
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
		rsp := services.NewUpdateMachineDefault(500).WithPayload(&models.APIError{
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
	dbMachine.LastVisited = m.LastVisited
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
		rsp := services.NewDeleteMachineDefault(500).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}

	err = dbmodel.DeleteMachine(r.Db, dbMachine)
	if err != nil {
		msg := fmt.Sprintf("cannot delete machine %d", params.ID)
		log.Error(err)
		rsp := services.NewDeleteMachineDefault(500).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}

	rsp := services.NewDeleteMachineOK()

	return rsp
}

func appToRestAPI(dbApp *dbmodel.App) *models.App {
	app := models.App{
		ID:          dbApp.ID,
		Type:        dbApp.Type,
		CtrlAddress: dbApp.CtrlAddress,
		CtrlPort:    dbApp.CtrlPort,
		Active:      dbApp.Active,
		Version:     dbApp.Meta.Version,
		Machine: &models.AppMachine{
			ID:       dbApp.MachineID,
			Address:  dbApp.Machine.Address,
			Hostname: dbApp.Machine.State.Hostname,
		},
	}

	isKeaApp := dbApp.Type == dbmodel.KeaAppType
	isBind9App := dbApp.Type == dbmodel.Bind9AppType

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
			Pid:     int64(dbApp.Details.(dbmodel.AppBind9).Daemon.Pid),
			Name:    dbApp.Details.(dbmodel.AppBind9).Daemon.Name,
			Active:  dbApp.Details.(dbmodel.AppBind9).Daemon.Active,
			Version: dbApp.Details.(dbmodel.AppBind9).Daemon.Version,
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
		rsp := services.NewGetAppsDefault(500).WithPayload(&models.APIError{
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
		rsp := services.NewGetAppDefault(500).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}
	if dbApp == nil {
		msg := fmt.Sprintf("cannot find app with id %d", params.ID)
		rsp := services.NewGetAppDefault(404).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}

	var a *models.App
	if dbApp.Type == dbmodel.Bind9AppType {
		a = appToRestAPI(dbApp)
	} else if dbApp.Type == dbmodel.KeaAppType {
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
		rsp := services.NewGetAppServicesStatusDefault(500).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}

	if dbApp == nil {
		msg := fmt.Sprintf("cannot find app with id %d", params.ID)
		log.Warn(errors.New(msg))
		rsp := services.NewGetAppDefault(404).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}

	servicesStatus := &models.ServicesStatus{}

	// If this is Kea application, get the Kea DHCP servers status which possibly
	// includes HA status.
	if dbApp.Type == dbmodel.KeaAppType {
		status, err := kea.GetDHCPStatus(ctx, r.Agents, dbApp)
		if err != nil {
			log.Error(err)
			msg := fmt.Sprintf("cannot get status of the app with id %d", params.ID)
			rsp := services.NewGetAppServicesStatusDefault(500).WithPayload(&models.APIError{
				Message: &msg,
			})
			return rsp
		}

		for _, s := range status {
			keaStatus := models.KeaStatus{
				Pid:    s.Pid,
				Uptime: s.Uptime,
				Reload: s.Reload,
				Daemon: s.Daemon,
			}

			if s.HAServers != nil {
				keaStatus.HaServers = &models.KeaStatusHaServers{
					LocalServer: &models.KeaStatusHaServersLocalServer{
						Role:   s.HAServers.Local.Role,
						Scopes: s.HAServers.Local.Scopes,
						State:  s.HAServers.Local.State,
					},
					RemoteServer: &models.KeaStatusHaServersRemoteServer{
						Age:     s.HAServers.Remote.Age,
						InTouch: s.HAServers.Remote.InTouch,
						Role:    s.HAServers.Remote.Role,
						Scopes:  s.HAServers.Remote.LastScopes,
						State:   s.HAServers.Remote.LastState,
					},
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
		rsp := services.NewGetAppsStatsDefault(500).WithPayload(&models.APIError{
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
		case dbmodel.KeaAppType:
			appsStats.KeaAppsTotal++
			if !dbApp.Active {
				appsStats.KeaAppsNotOk++
			}
		case dbmodel.Bind9AppType:
			appsStats.Bind9AppsTotal++
			if !dbApp.Active {
				appsStats.Bind9AppsNotOk++
			}
		}
	}

	rsp := services.NewGetAppsStatsOK().WithPayload(&appsStats)
	return rsp
}
