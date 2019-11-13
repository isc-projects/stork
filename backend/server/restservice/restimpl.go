package restservice

import (
	"fmt"
	"time"
	"context"

	log "github.com/sirupsen/logrus"
	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/runtime/middleware"

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

	state, err := r.Agents.GetState(dbMachine.Address)
	if err != nil {
		log.Warn(err)
		dbMachine.Error = "cannot get state of machine"
		err = r.Db.Update(dbMachine)
		if err != nil {
			log.Error(err)
		}
		m := MachineToRestApi(*dbMachine)
		rsp := services.NewGetMachineStateOK().WithPayload(m)
		return rsp
	}

	err = updateMachineFields(r.Db, dbMachine, state)
	if err != nil {
		rsp := services.NewGetMachineStateOK().WithPayload(&models.Machine{
			ID: int64(dbMachine.Id),
			Error: "cannot update machine in db",
		})
		return rsp
	}

	m := MachineToRestApi(*dbMachine)
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

	log.Infof("DB1 %+v", r.Db)
	dbMachines, total, err := dbmodel.GetMachines(r.Db, start, limit, text)
	if err != nil {
		log.Error(err)
		msg := "cannot get machines from db"
		rsp := services.NewCreateMachineDefault(500).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}


	for _, dbM := range dbMachines {
		mm := MachineToRestApi(dbM)
		machines = append(machines, mm)
	}

	m := models.Machines{
		Items: machines,
		Total: int64(total),
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
	m := MachineToRestApi(*dbMachine)
	rsp := services.NewGetMachineOK().WithPayload(m)
	return rsp
}

// Get one machine by ID where Stork Agent is running.
func (r *RestAPI) UpdateMachine(ctx context.Context, params services.UpdateMachineParams) middleware.Responder {
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
	dbMachine.Address = *params.Machine.Address
	err = r.Db.Update(dbMachine)
	if err != nil {
		msg := fmt.Sprintf("cannot update machine with id %d in db", params.ID)
		log.Error(err)
		rsp := services.NewUpdateMachineDefault(500).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}
	m := MachineToRestApi(*dbMachine)
	rsp := services.NewUpdateMachineOK().WithPayload(m)
	return rsp
}

func updateMachineFields(db *dbops.PgDB, dbMachine *dbmodel.Machine, m *agentcomm.State) error {
	dbMachine.AgentVersion = m.AgentVersion
	dbMachine.Cpus = m.Cpus
	dbMachine.CpusLoad = m.CpusLoad
	dbMachine.Memory = m.Memory
	dbMachine.Hostname = m.Hostname
	dbMachine.Uptime = m.Uptime
	dbMachine.UsedMemory = m.UsedMemory
	dbMachine.Os = m.Os
	dbMachine.Platform = m.Platform
	dbMachine.PlatformFamily = m.PlatformFamily
	dbMachine.PlatformVersion = m.PlatformVersion
	dbMachine.KernelVersion = m.KernelVersion
	dbMachine.KernelArch = m.KernelArch
	dbMachine.VirtualizationSystem = m.VirtualizationSystem
	dbMachine.VirtualizationRole = m.VirtualizationRole
	dbMachine.HostID = m.HostID
	dbMachine.LastVisited = m.LastVisited
	dbMachine.Error = m.Error
	return db.Update(dbMachine)
}

func MachineToRestApi(dbMachine dbmodel.Machine) *models.Machine {
	m := models.Machine{
		ID: int64(dbMachine.Id),
		Address: &dbMachine.Address,
		AgentVersion: dbMachine.AgentVersion,
		Cpus: dbMachine.Cpus,
		CpusLoad: dbMachine.CpusLoad,
		Memory: dbMachine.Memory,
		Hostname: dbMachine.Hostname,
		Uptime: dbMachine.Uptime,
		UsedMemory: dbMachine.UsedMemory,
		Os: dbMachine.Os,
		Platform: dbMachine.Platform,
		PlatformFamily: dbMachine.PlatformFamily,
		PlatformVersion: dbMachine.PlatformVersion,
		KernelVersion: dbMachine.KernelVersion,
		KernelArch: dbMachine.KernelArch,
		VirtualizationSystem: dbMachine.VirtualizationSystem,
		VirtualizationRole: dbMachine.VirtualizationRole,
		HostID: dbMachine.HostID,
		LastVisited: strfmt.DateTime(dbMachine.LastVisited),
		Error: dbMachine.Error,
	}
	return &m
}

// Add a machine where Stork Agent is running.
func (r *RestAPI) CreateMachine(ctx context.Context, params services.CreateMachineParams) middleware.Responder {
	addr := params.Machine.Address

	dbMachine, err := dbmodel.GetMachineByAddress(r.Db, *addr, true)
	if err == nil && dbMachine != nil && dbMachine.Deleted.IsZero() {
		msg := fmt.Sprintf("machine %s already exists", *addr)
		log.Warnf(msg)
		rsp := services.NewCreateMachineDefault(400).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}

	if dbMachine == nil {
		dbMachine = &dbmodel.Machine{Address: *addr}
		err = dbmodel.AddMachine(r.Db, dbMachine)
		if err != nil {
			msg := fmt.Sprintf("cannot store machine %s", *addr)
			log.Error(err)
			rsp := services.NewCreateMachineDefault(500).WithPayload(&models.APIError{
				Message: &msg,
			})
			return rsp
		}
	} else {
		dbMachine.Deleted = time.Time{}
	}

	state, err := r.Agents.GetState(*addr)
	if err != nil {
		log.Warn(err)
		dbMachine.Error = "cannot get state of machine"
		err = r.Db.Update(dbMachine)
		if err != nil {
			log.Error(err)
		}
		m := MachineToRestApi(*dbMachine)
		rsp := services.NewCreateMachineOK().WithPayload(m)
		return rsp
	}

	err = updateMachineFields(r.Db, dbMachine, state)
	if err != nil {
		rsp := services.NewCreateMachineOK().WithPayload(&models.Machine{
			ID: int64(dbMachine.Id),
			Address: addr,
			Error: "cannot update machine in db",
		})
		return rsp
	}

	m := MachineToRestApi(*dbMachine)
	rsp := services.NewCreateMachineOK().WithPayload(m)

	return rsp
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

