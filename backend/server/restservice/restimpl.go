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

func machineToRestApi(dbMachine dbmodel.Machine) *models.Machine {
	var services []*models.MachineService
	for _, srv := range dbMachine.Services {
		s := models.MachineService{
			ID: srv.Id,
			Type: srv.Type,
			Version: srv.Meta.Version,
		}
		services = append(services, &s)
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
		Services: services,
	}
	return &m
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
		}
		m := machineToRestApi(*dbMachine)
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

	m := machineToRestApi(*dbMachine)
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

	service := ""
	if params.Service != nil {
		service = *params.Service
	}

	log.WithFields(log.Fields{
		"start": start,
		"limit": limit,
		"text": text,
		"service": service,
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
		mm := machineToRestApi(dbM)
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
	m := machineToRestApi(*dbMachine)
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
		}
		m := machineToRestApi(*dbMachine)
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

	m := machineToRestApi(*dbMachine)
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
	m := machineToRestApi(*dbMachine)
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

	// update services associated with machine

	// get list of present services in db
	dbServices, err := dbmodel.GetServicesByMachine(db, dbMachine.Id)
	if err != nil {
		return err
	}

	dbServicesMap := make(map[string]dbmodel.Service)
	for _, dbSrv := range dbServices {
		dbServicesMap[dbSrv.Type] = dbSrv
	}

	var keaSrv *agentcomm.ServiceKea = nil
	//var bindSrv *agentcomm.ServiceBind
	for _, srv := range m.Services {
		switch s := srv.(type) {
		case *agentcomm.ServiceKea:
			keaSrv = s
		// case agentcomm.ServiceBind:
		// 	bindSrv = &s
		default:
			log.Println("NOT IMPLEMENTED")
		}
	}

	var keaDaemons []dbmodel.KeaDaemon
	if keaSrv != nil {
		for _, d := range keaSrv.Daemons {
			keaDaemons = append(keaDaemons, dbmodel.KeaDaemon{
				Pid: d.Pid,
				Name: d.Name,
				Active: d.Active,
				Version: d.Version,
				ExtendedVersion: d.ExtendedVersion,
			})
		}
	}

	dbKeaSrv, dbOk := dbServicesMap["kea"]
	if dbOk && keaSrv != nil {
		// update service in db
		meta := dbmodel.ServiceMeta{
			Version: keaSrv.Version,
		}
		dbKeaSrv.Deleted = time.Time{}  // undelete if it was deleted
		dbKeaSrv.CtrlPort = keaSrv.CtrlPort
		dbKeaSrv.Active = keaSrv.Active
		dbKeaSrv.Meta = meta
		dt := dbKeaSrv.Details.(dbmodel.ServiceKea)
		dt.ExtendedVersion = keaSrv.ExtendedVersion
		dt.Daemons = keaDaemons
		err = db.Update(&dbKeaSrv)
		if err != nil {
			return errors.Wrapf(err, "problem with updating service %v", dbKeaSrv)
		}
	} else if dbOk && keaSrv == nil {
		// delete service from db
		err = dbmodel.DeleteService(db, &dbKeaSrv)
		if err != nil {
			return err
		}
	} else if !dbOk && keaSrv != nil {
		// add service to db
		dbKeaSrv = dbmodel.Service{
			MachineID: dbMachine.Id,
			Type: "kea",
			CtrlPort: keaSrv.CtrlPort,
			Active: keaSrv.Active,
			Meta: dbmodel.ServiceMeta{
				Version: keaSrv.Version,
			},
			Details: dbmodel.ServiceKea{
				ExtendedVersion: keaSrv.ExtendedVersion,
				Daemons: keaDaemons,
			},
		}
		err = dbmodel.AddService(db, &dbKeaSrv)
		if err != nil {
			return err
		}
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

func serviceToRestApi(dbService dbmodel.Service) *models.Service {
	var daemons []*models.KeaDaemon
	for _, d := range dbService.Details.(dbmodel.ServiceKea).Daemons {
		daemons = append(daemons, &models.KeaDaemon{
			Pid: int64(d.Pid),
			Name: d.Name,
			Active: d.Active,
			Version: d.Version,
			ExtendedVersion: d.ExtendedVersion,
		})
	}
	s := models.Service{
		ID: dbService.Id,
		Type: dbService.Type,
		CtrlPort: dbService.CtrlPort,
		Active: dbService.Active,
		Version: dbService.Meta.Version,
		Details: struct {
			models.ServiceKea
			models.ServiceBind
		}{
			models.ServiceKea{
				ExtendedVersion: dbService.Details.(dbmodel.ServiceKea).ExtendedVersion,
				Daemons: daemons,
			},
			models.ServiceBind{},
		},
		Machine: &models.ServiceMachine{
			ID: dbService.MachineID,
			Address: dbService.Machine.Address,
			Hostname: dbService.Machine.State.Hostname,
		},
	}
	return &s
}

func (r *RestAPI) GetServices(ctx context.Context, params services.GetServicesParams) middleware.Responder {
	servicesLst := []*models.Service{}

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

	service := ""
	if params.Service != nil {
		service = *params.Service
	}

	log.WithFields(log.Fields{
		"start": start,
		"limit": limit,
		"text": text,
		"service": service,
	}).Info("query services")

	dbServices, total, err := dbmodel.GetServicesByPage(r.Db, start, limit, text, service)
	if err != nil {
		log.Error(err)
		msg := "cannot get services from db"
		rsp := services.NewGetServicesDefault(500).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}


	for _, dbS := range dbServices {
		ss := serviceToRestApi(dbS)
		servicesLst = append(servicesLst, ss)
	}

	s := models.Services{
		Items: servicesLst,
		Total: total,
	}
	rsp := services.NewGetServicesOK().WithPayload(&s)
	return rsp
}

func (r *RestAPI) GetService(ctx context.Context, params services.GetServiceParams) middleware.Responder {
	dbService, err := dbmodel.GetServiceById(r.Db, params.ID)
	if err != nil {
		msg := fmt.Sprintf("cannot get service with id %d from db", params.ID)
		log.Error(err)
		rsp := services.NewGetServiceDefault(500).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}
	if dbService == nil {
		msg := fmt.Sprintf("cannot find service with id %d", params.ID)
		rsp := services.NewGetServiceDefault(404).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}
	s := serviceToRestApi(*dbService)
	rsp := services.NewGetServiceOK().WithPayload(s)
	return rsp
}
