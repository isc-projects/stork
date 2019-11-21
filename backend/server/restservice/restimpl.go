package restservice

import (
	"fmt"
	"net"
	"time"
	"context"
	"net/url"

	log "github.com/sirupsen/logrus"
	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/runtime/middleware"
	"github.com/pkg/errors"

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

	ctx2, cancel := context.WithTimeout(ctx, 2 * time.Second)
	defer cancel()
	state, err := r.Agents.GetState(ctx2, dbMachine.Address)
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

func checkMachineAddress(address string) (string, error, string) {
	// Use net/url to parse the address. Trick: pretend http url address.
	// This way we can parse any case of address:
	// - localhost
	// - localhost:8080
	// - 127.0.0.1:8080
	// - [2001:0db8:85a3:0000:0000:8a2e:0370:7334]:8080
	httpAddr := "http://" + strings.TrimSpace(address)
	u, err := url.Parse(httpAddr)
	if err != nil {
		return "", errors.Wrapf(err, "problem with parsing address %v", address), "cannot parse address"
	}
	host := u.Hostname()
	port := u.Port()
	if port == "" {
		port = stork.DEFAULT_AGENT_PORT
	}

	return net.JoinHostPort(host, port), nil, ""
}

// Get one machine by ID where Stork Agent is running.
func (r *RestAPI) UpdateMachine(ctx context.Context, params services.UpdateMachineParams) middleware.Responder {
	address, err, errStr := checkMachineAddress(*params.Machine.Address)
	if err != nil {
		log.Warn(err)
		rsp := services.NewUpdateMachineDefault(400).WithPayload(&models.APIError{
			Message: &errStr,
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
	if dbMachine.Address != address {
		dbMachine2, err := dbmodel.GetMachineByAddress(r.Db, address, false)
		if err == nil && dbMachine2.Id != dbMachine.Id {
			msg := fmt.Sprintf("machine with address %s already exists", *params.Machine.Address)
			rsp := services.NewUpdateMachineDefault(400).WithPayload(&models.APIError{
				Message: &msg,
			})
			return rsp
		}
	}

	// copy fields
	dbMachine.Address = address
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
	return db.Update(dbMachine)
}

func MachineToRestApi(dbMachine dbmodel.Machine) *models.Machine {
	m := models.Machine{
		ID: int64(dbMachine.Id),
		Address: &dbMachine.Address,
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
	}
	return &m
}

// Add a machine where Stork Agent is running.
func (r *RestAPI) CreateMachine(ctx context.Context, params services.CreateMachineParams) middleware.Responder {
	addr, err, errStr := checkMachineAddress(*params.Machine.Address)
	if err != nil {
		log.Warn(err)
		rsp := services.NewCreateMachineDefault(400).WithPayload(&models.APIError{
			Message: &errStr,
		})
		return rsp
	}

	dbMachine, err := dbmodel.GetMachineByAddress(r.Db, addr, true)
	if err == nil && dbMachine != nil && dbMachine.Deleted.IsZero() {
		msg := fmt.Sprintf("machine %s already exists", addr)
		log.Warnf(msg)
		rsp := services.NewCreateMachineDefault(400).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}

	if dbMachine == nil {
		dbMachine = &dbmodel.Machine{Address: addr}
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
	state, err := r.Agents.GetState(ctx2, addr)
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
			Address: &addr,
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

