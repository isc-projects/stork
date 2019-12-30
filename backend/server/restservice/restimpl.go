package restservice

import (
	"fmt"
	"time"
	"context"

	log "github.com/sirupsen/logrus"
	"github.com/pkg/errors"
	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/runtime/middleware"
	"github.com/asaskevich/govalidator"

	"isc.org/stork"
	"isc.org/stork/server/database"
	"isc.org/stork/server/database/model"
	"isc.org/stork/server/agentcomm"
	"isc.org/stork/server/gen/models"
	"isc.org/stork/server/gen/restapi/operations/general"
	"isc.org/stork/server/gen/restapi/operations/services"
	"isc.org/stork/server/apps/kea"
)


// Get version of Stork server.
func (r *RestAPI) GetVersion(ctx context.Context, params general.GetVersionParams) middleware.Responder {
	d, err := strfmt.ParseDateTime("0001-01-01T00:00:00.000Z")
	if err != nil {
		fmt.Printf("problem\n")
	}
	bt := stork.BuildType
	v := stork.Version
	ver := models.Version{
		Date: &d,
		Type: &bt,
		Version: &v,
	}
	return general.NewGetVersionOK().WithPayload(&ver)
}

func machineToRestApi(dbMachine dbmodel.Machine) (*models.Machine, error) {
	var apps []*models.MachineApp
	for _, app := range dbMachine.Apps {
		active := true
		if app.Type == "kea" {
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
			ID: app.Id,
			Type: app.Type,
			Version: app.Meta.Version,
			Active: active,
		}
		apps = append(apps, &s)
	}

	m := models.Machine{
		ID: dbMachine.Id,
		Address: &dbMachine.Address,
		AgentPort: dbMachine.AgentPort,
		AgentVersion: dbMachine.State.AgentVersion,
		Cpus: dbMachine.State.Cpus,
		CpusLoad: dbMachine.State.CpusLoad,
		Memory: dbMachine.State.Memory,
		Hostname: dbMachine.State.Hostname,
		Uptime: dbMachine.State.Uptime,
		UsedMemory: dbMachine.State.UsedMemory,
		Os: dbMachine.State.Os,
		Platform: dbMachine.State.Platform,
		PlatformFamily: dbMachine.State.PlatformFamily,
		PlatformVersion: dbMachine.State.PlatformVersion,
		KernelVersion: dbMachine.State.KernelVersion,
		KernelArch: dbMachine.State.KernelArch,
		VirtualizationSystem: dbMachine.State.VirtualizationSystem,
		VirtualizationRole: dbMachine.State.VirtualizationRole,
		HostID: dbMachine.State.HostID,
		LastVisited: strfmt.DateTime(dbMachine.LastVisited),
		Error: dbMachine.Error,
		Apps: apps,
	}
	return &m, nil
}

// Get runtime state of indicated machine.
func (r *RestAPI) GetMachineState(ctx context.Context, params services.GetMachineStateParams) middleware.Responder {
	dbMachine, err := dbmodel.GetMachineById(r.Db, params.ID)
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

	ctx2, cancel := context.WithTimeout(ctx, 2 * time.Second)
	defer cancel()
	state, err := r.Agents.GetState(ctx2, dbMachine.Address, dbMachine.AgentPort)
	if err != nil {
		log.Warn(err)
		dbMachine.Error = "cannot get state of machine"
		err = r.Db.Update(dbMachine)
		if err != nil {
			log.Error(err)
			msg := "problem with updating record in database"
			rsp := services.NewGetMachineStateDefault(500).WithPayload(&models.APIError{
				Message: &msg,
			})
			return rsp
		}
		m, err := machineToRestApi(*dbMachine)
		if err != nil {
			log.Error(err)
			msg := "problem with serializing data"
			rsp := services.NewGetMachineStateDefault(500).WithPayload(&models.APIError{
				Message: &msg,
			})
			return rsp
		}

		rsp := services.NewGetMachineStateOK().WithPayload(m)
		return rsp
	}

	err = updateMachineFields(r.Db, dbMachine, state)
	if err != nil {
		log.Error(err)
		rsp := services.NewGetMachineStateOK().WithPayload(&models.Machine{
			ID: dbMachine.Id,
			Error: "cannot update machine in db",
		})
		return rsp
	}

	m, err := machineToRestApi(*dbMachine)
	if err != nil {
		log.Error(err)
		msg := "problem with serializing data"
		rsp := services.NewGetMachineStateDefault(500).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}
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
		"text": text,
		"app": app,
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
		mm, err := machineToRestApi(dbM)
		if err != nil {
			log.Error(err)
			msg := "problem with serializing data"
			rsp := services.NewGetMachinesDefault(500).WithPayload(&models.APIError{
				Message: &msg,
			})
			return rsp
		}
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
	dbMachine, err := dbmodel.GetMachineById(r.Db, params.ID)
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
	m, err := machineToRestApi(*dbMachine)
	if err != nil {
		log.Error(err)
		msg := "problem with serializing data"
		rsp := services.NewGetMachineDefault(500).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}
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

	ctx2, cancel := context.WithTimeout(ctx, 100 * time.Second)
	defer cancel()
	state, err := r.Agents.GetState(ctx2, addr, params.Machine.AgentPort)
	if err != nil {
		log.Warn(err)
		dbMachine.Error = "cannot get state of machine"
		err = r.Db.Update(dbMachine)
		if err != nil {
			log.Error(err)
			msg := "problem with updating record in database"
			rsp := services.NewGetMachineStateDefault(500).WithPayload(&models.APIError{
				Message: &msg,
			})
			return rsp
		}
		m, err := machineToRestApi(*dbMachine)
		if err != nil {
			log.Error(err)
			msg := "problem with serializing data"
			rsp := services.NewGetMachineDefault(500).WithPayload(&models.APIError{
				Message: &msg,
			})
			return rsp
		}
		rsp := services.NewCreateMachineOK().WithPayload(m)
		return rsp
	}

	err = updateMachineFields(r.Db, dbMachine, state)
	if err != nil {
		log.Error(err)
		rsp := services.NewCreateMachineOK().WithPayload(&models.Machine{
			ID: dbMachine.Id,
			Address: &addr,
			Error: "cannot update machine in db",
		})
		return rsp
	}

	m, err := machineToRestApi(*dbMachine)
	if err != nil {
		log.Error(err)
		msg := "problem with serializing data"
		rsp := services.NewGetMachineDefault(500).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}
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

	dbMachine, err := dbmodel.GetMachineById(r.Db, params.ID)
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
		if err == nil && dbMachine2 != nil && dbMachine2.Id != dbMachine.Id {
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
	m, err := machineToRestApi(*dbMachine)
	if err != nil {
		log.Error(err)
		msg := "problem with serializing data"
		rsp := services.NewUpdateMachineDefault(500).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}
	rsp := services.NewUpdateMachineOK().WithPayload(m)
	return rsp
}

func updateMachineFieldsKea(db *dbops.PgDB, dbMachine *dbmodel.Machine, dbAppsMap map[string]dbmodel.App, keaApp *agentcomm.AppKea) (err error) {
	var keaDaemons []dbmodel.KeaDaemon
	if keaApp != nil {
		for _, d := range keaApp.Daemons {
			keaDaemons = append(keaDaemons, dbmodel.KeaDaemon{
				Pid:             d.Pid,
				Name:            d.Name,
				Active:          d.Active,
				Version:         d.Version,
				ExtendedVersion: d.ExtendedVersion,
			})
		}
	}

	dbKeaApp, dbOk := dbAppsMap["kea"]
	if dbOk && keaApp != nil {
		// update app in db
		meta := dbmodel.AppMeta{
			Version: keaApp.Version,
		}
		dbKeaApp.Deleted = time.Time{} // undelete if it was deleted
		dbKeaApp.CtrlPort = keaApp.CtrlPort
		dbKeaApp.Active = keaApp.Active
		dbKeaApp.Meta = meta
		dt := dbKeaApp.Details.(dbmodel.AppKea)
		dt.ExtendedVersion = keaApp.ExtendedVersion
		dt.Daemons = keaDaemons
		err = db.Update(&dbKeaApp)
		if err != nil {
			return errors.Wrapf(err, "problem with updating app %v", dbKeaApp)
		}
	} else if dbOk && keaApp == nil {
		// delete app from db
		err = dbmodel.DeleteApp(db, &dbKeaApp)
		if err != nil {
			return err
		}
	} else if !dbOk && keaApp != nil {
		// add app to db
		dbKeaApp = dbmodel.App{
			MachineID: dbMachine.Id,
			Type:      "kea",
			CtrlPort:  keaApp.CtrlPort,
			Active:    keaApp.Active,
			Meta: dbmodel.AppMeta{
				Version: keaApp.Version,
			},
			Details: dbmodel.AppKea{
				ExtendedVersion: keaApp.ExtendedVersion,
				Daemons:         keaDaemons,
			},
		}
		err = dbmodel.AddApp(db, &dbKeaApp)
		if err != nil {
			return err
		}
	}
	return nil
}

func updateMachineFieldsBind9(db *dbops.PgDB, dbMachine *dbmodel.Machine, dbAppsMap map[string]dbmodel.App, bind9App *agentcomm.AppBind9) (err error) {
	var bind9Daemon dbmodel.Bind9Daemon

	if bind9App != nil {
		bind9Daemon = dbmodel.Bind9Daemon{
			Pid:     bind9App.Daemon.Pid,
			Name:    bind9App.Daemon.Name,
			Active:  bind9App.Daemon.Active,
			Version: bind9App.Daemon.Version,
		}
	}

	dbBind9App, dbOk := dbAppsMap["bind9"]
	if dbOk && bind9App != nil {
		// update app in db
		meta := dbmodel.AppMeta{
			Version: bind9App.Version,
		}
		dbBind9App.Deleted = time.Time{} // undelete if it was deleted
		dbBind9App.CtrlPort = bind9App.CtrlPort
		dbBind9App.Active = bind9App.Active
		dbBind9App.Meta = meta
		dt := dbBind9App.Details.(dbmodel.AppBind9)
		dt.Daemon = bind9Daemon
		err = db.Update(&dbBind9App)
		if err != nil {
			return errors.Wrapf(err, "problem with updating app %v", dbBind9App)
		}
	} else if dbOk && bind9App == nil {
		// delete app from db
		err = dbmodel.DeleteApp(db, &dbBind9App)
		if err != nil {
			return err
		}
	} else if !dbOk && bind9App != nil {
		// add app to db
		dbBind9App = dbmodel.App{
			MachineID: dbMachine.Id,
			Type:      "bind9",
			CtrlPort:  bind9App.CtrlPort,
			Active:    bind9App.Active,
			Meta: dbmodel.AppMeta{
				Version: bind9App.Version,
			},
			Details: dbmodel.AppBind9{
				Daemon: bind9Daemon,
			},
		}
		err = dbmodel.AddApp(db, &dbBind9App)
		if err != nil {
			return err
		}
	}
	return nil
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

	// update services associated with machine

	// get list of present services in db
	dbApps, err := dbmodel.GetAppsByMachine(db, dbMachine.Id)
	if err != nil {
		return err
	}

	dbAppsMap := make(map[string]dbmodel.App)
	for _, dbSrv := range dbApps {
		dbAppsMap[dbSrv.Type] = dbSrv
	}

	var keaApp *agentcomm.AppKea = nil
	var bind9App *agentcomm.AppBind9 = nil
	for _, app := range m.Apps {
		switch a := app.(type) {
		case *agentcomm.AppKea:
			keaApp = a
		case *agentcomm.AppBind9:
		 	bind9App = a
		default:
			log.Println("NOT IMPLEMENTED")
		}
	}

	err = updateMachineFieldsKea(db, dbMachine, dbAppsMap, keaApp)
	if err != nil {
		return err
	}

	err = updateMachineFieldsBind9(db, dbMachine, dbAppsMap, bind9App)
	if err != nil {
		return err
	}

	err = dbmodel.RefreshMachineFromDb(db, dbMachine)
	if err != nil {
		return err
	}

	return nil
}

// Add a machine where Stork Agent is running.
func (r *RestAPI) DeleteMachine(ctx context.Context, params services.DeleteMachineParams) middleware.Responder {
	dbMachine, err := dbmodel.GetMachineById(r.Db, params.ID)
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

func appToRestApi(dbApp *dbmodel.App, hooks map[string][]string) *models.App {
	app := models.App{
		ID: dbApp.Id,
		Type: dbApp.Type,
		CtrlAddress: dbApp.CtrlAddress,
		CtrlPort: dbApp.CtrlPort,
		Active: dbApp.Active,
		Version: dbApp.Meta.Version,
		Machine: &models.AppMachine{
			ID: dbApp.MachineID,
			Address: dbApp.Machine.Address,
			Hostname: dbApp.Machine.State.Hostname,
		},
	}

	isKeaApp := dbApp.Type == "kea"
	isBind9App := dbApp.Type == "bind9"

	if isKeaApp {
		var keaDaemons []*models.KeaDaemon
		for _, d := range dbApp.Details.(dbmodel.AppKea).Daemons {
			dmn := &models.KeaDaemon{
				Pid: int64(d.Pid),
				Name: d.Name,
				Active: d.Active,
				Version: d.Version,
				ExtendedVersion: d.ExtendedVersion,
				Hooks: []string{},
			}
			if hooks != nil {
				hooksList, ok := hooks[d.Name]
				if ok {
					dmn.Hooks = hooksList
				}
			}
			keaDaemons = append(keaDaemons, dmn)
		}

		app.Details = struct { models.AppKea; models.AppBind9 }{
			models.AppKea{
				ExtendedVersion: dbApp.Details.(dbmodel.AppKea).ExtendedVersion,
				Daemons:         keaDaemons,
			},
			models.AppBind9{},
		}
	}

	if isBind9App {
		bind9Daemon := &models.Bind9Daemon{
			Pid: int64(dbApp.Details.(dbmodel.AppBind9).Daemon.Pid),
			Name: dbApp.Details.(dbmodel.AppBind9).Daemon.Name,
			Active: dbApp.Details.(dbmodel.AppBind9).Daemon.Active,
			Version: dbApp.Details.(dbmodel.AppBind9).Daemon.Version,
		}

		app.Details = struct { models.AppKea; models.AppBind9 }{
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
		"text": text,
		"app": app,
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
		a := appToRestApi(&app, nil)
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
	dbApp, err := dbmodel.GetAppById(r.Db, params.ID)
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
	if dbApp.Type == "bind9" {
		a = appToRestApi(dbApp, nil)
	} else if dbApp.Type == "kea" {
		hooksByDaemon, err := kea.GetDaemonHooks(ctx, r.Agents, dbApp)
		if err != nil {
			log.Warn(err)
		}
		a = appToRestApi(dbApp, hooksByDaemon)
	}
	rsp := services.NewGetAppOK().WithPayload(a)
	return rsp
}

// Gets current status of services which the given application is associated with.
func (r *RestAPI) GetAppServicesStatus(ctx context.Context, params services.GetAppServicesStatusParams) middleware.Responder {
	dbApp, err := dbmodel.GetAppById(r.Db, params.ID)
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
	if dbApp.Type == "kea" {
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
						Role: s.HAServers.Local.Role,
						Scopes: s.HAServers.Local.Scopes,
						State: s.HAServers.Local.State,
					},
					RemoteServer: &models.KeaStatusHaServersRemoteServer{
						Age: s.HAServers.Remote.Age,
						InTouch: s.HAServers.Remote.InTouch,
						Role: s.HAServers.Remote.Role,
						Scopes: s.HAServers.Remote.LastScopes,
						State: s.HAServers.Remote.LastState,
					},
				}
			}

			serviceStatus := &models.ServiceStatus{
				Status: struct {
					models.KeaStatus
				} {
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
		KeaAppsTotal: 0,
		KeaAppsNotOk: 0,
		Bind9AppsTotal: 0,
		Bind9AppsNotOk: 0,
	}
	for _, dbApp := range dbApps {
		switch dbApp.Type {
		case "kea":
			appsStats.KeaAppsTotal += 1
			if !dbApp.Active {
				appsStats.KeaAppsNotOk += 1
			}
		case "bind9":
			appsStats.Bind9AppsTotal += 1
			if !dbApp.Active {
				appsStats.Bind9AppsNotOk += 1
			}
		}
	}

	rsp := services.NewGetAppsStatsOK().WithPayload(&appsStats)
	return rsp
}
