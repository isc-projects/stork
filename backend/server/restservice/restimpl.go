package restservice

import (
	"fmt"
	"context"

	log "github.com/sirupsen/logrus"
	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/runtime/middleware"

	"isc.org/stork/server/gen/models"
	"isc.org/stork/server/gen/restapi/operations"
)


// Get version of Stork server.
func (r *RestAPI) GetVersion(ctx context.Context, params operations.GetVersionParams) middleware.Responder {
	t := "stable"
	v := "0.1.0"
	d, err := strfmt.ParseDateTime("0001-01-01T00:00:00.000Z")
	if err != nil {
		fmt.Printf("problem\n")
	}
	var ver models.Version
	ver.Date = &d
	ver.Type = &t
	ver.Version = &v
	return operations.NewGetVersionOK().WithPayload(&ver)
}

// Get version from indicated agent.
func (r *RestAPI) GetAgentVersion(ctx context.Context, params operations.GetAgentVersionParams) middleware.Responder {
	ver, err := r.Agents.GetVersion(params.AgentIP)
	if err != nil {
		log.Printf("%+v", err)
		msg := "problems with connecting to agent"
		rspErr := models.Error{
			Code: 500,
			Message: &msg,
		}
		return operations.NewGetAgentVersionDefault(500).WithPayload(&rspErr)
	}

	log.Printf("AGENT %v, ver %s", params.AgentIP, ver)

	rspVer := models.AgentVersion{Version: &ver}

	return operations.NewGetAgentVersionOK().WithPayload(&rspVer)
}
