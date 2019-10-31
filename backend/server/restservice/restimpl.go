package restservice

import (
	"fmt"
	"context"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/runtime/middleware"

	"isc.org/stork"
	"isc.org/stork/server/database/model"
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
	state, err := r.Agents.GetState(params.Address)
	if err != nil {
		log.Printf("%+v", err)
		msg := "problems with connecting to agent"
		rspErr := models.APIError{
			Code: 500,
			Message: &msg,
		}
		return services.NewGetMachineStateDefault(500).WithPayload(&rspErr)
	}

	rspState := models.Machine{
		Address: &params.Address,
		Cpus: state.Cpus,
		CpusLoad: state.CpusLoad,
		Memory: state.Memory,
		Hostname: state.Hostname,
		Uptime: state.Uptime,
		UsedMemory: state.UsedMemory,
		Error: state.Error,
		LastVisited: strfmt.DateTime(state.LastVisited),
	}

	return services.NewGetMachineStateOK().WithPayload(&rspState)
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

	addr := "10.2.3.4"
	machines = append(machines, &models.Machine{
		Address: &addr,
		Cpus: 4,
		CpusLoad: "0.10 0.20 0.30",
		Memory: 4,
		Hostname: "mach1.isc.org",
		Uptime: 123,
		UsedMemory: 78,
	})

	m := models.Machines{
		Items: machines,
		Total: 10,
	}
	rsp := services.NewGetMachinesOK().WithPayload(&m)
	return rsp
}

// Add a machine where Stork Agent is running.
func (r *RestAPI) CreateMachine(ctx context.Context, params services.CreateMachineParams) middleware.Responder {
	log.Info("create machine")

	addr := params.Machine.Address

	m := models.Machine{Address: addr}

	state, err := r.Agents.GetState(*addr)
	if err != nil {
		m.Error = "cannot get state of machine"
		rsp := services.NewCreateMachineOK().WithPayload(&m)
		return rsp
	}
	log.Infof("stat %+v", state)

	m.AgentVersion = state.AgentVersion
	m.Cpus = state.Cpus
	m.CpusLoad = state.CpusLoad
	m.Memory = state.Memory
	m.Hostname = state.Hostname
	m.Uptime = state.Uptime
	m.UsedMemory = state.UsedMemory
	m.Os = state.Os
	m.Platform = state.Platform
	m.PlatformFamily = state.PlatformFamily
	m.PlatformVersion = state.PlatformVersion
	m.KernelVersion = state.KernelVersion
	m.KernelArch = state.KernelArch
	m.VirtualizationSystem = state.VirtualizationSystem
	m.VirtualizationRole = state.VirtualizationRole
	m.HostID = state.HostID
	m.LastVisited = strfmt.DateTime(state.LastVisited)
	m.Error = state.Error
	rsp := services.NewCreateMachineOK().WithPayload(&m)

	return rsp
}

// Attempts to login the user to the system.
func (r *RestAPI) PostSessions(ctx context.Context, params operations.PostSessionsParams) middleware.Responder {
	user := &dbmodel.SystemUser{}
	login := *params.Useremail
	if strings.Contains(login, "@") {
		user.Email = login
	} else {
		user.Login = login
	}
	user.Password = *params.Userpassword

	ok, err := dbmodel.Authenticate(r.PgDB, user);
	if ok {
		err = r.SessionManager.LoginHandler(ctx)
	}

	if !ok || err != nil {
		if err != nil {
			log.Printf("%+v", err)
		}
		return operations.NewPostSessionsBadRequest()
	}

	rspUserId := int64(user.Id)
	rspUser := models.SystemUser{
		ID: &rspUserId,
		Login: &user.Login,
		Email: &user.Email,
		Firstname: &user.Name,
		Lastname: &user.Lastname,
	}

	return operations.NewPostSessionsOK().WithPayload(&rspUser)
}

// Attempts to logout a user from the system.
func (r *RestAPI) DeleteSessions(ctx context.Context, params operations.DeleteSessionsParams) middleware.Responder {
	err := r.SessionManager.LogoutHandler(ctx)
	if err != nil {
		log.Printf("%+v", err)
		return operations.NewDeleteSessionsBadRequest()
	}
	return operations.NewDeleteSessionsOK()
}
