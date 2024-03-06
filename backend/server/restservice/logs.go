package restservice

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-openapi/runtime/middleware"
	log "github.com/sirupsen/logrus"

	dbmodel "isc.org/stork/server/database/model"
	"isc.org/stork/server/gen/models"
	"isc.org/stork/server/gen/restapi/operations/services"
	storkutil "isc.org/stork/util"
)

// Get tail of the specified log file.
func (r *RestAPI) GetLogTail(ctx context.Context, params services.GetLogTailParams) middleware.Responder {
	// We have ID of the log file to display. We need to get the details
	// of the file from the database.
	dbLogTarget, err := dbmodel.GetLogTargetByID(r.DB, params.ID)
	if err != nil {
		msg := fmt.Sprintf("Cannot get information about log file with ID %d from the database", params.ID)
		log.Error(msg)
		rsp := services.NewGetLogTailDefault(http.StatusInternalServerError).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}

	// Handle the case when referencing the non-existing file.
	if dbLogTarget == nil {
		msg := fmt.Sprintf("Log file with ID %d does not exist", params.ID)
		log.Warn(msg)
		rsp := services.NewGetLogTailDefault(http.StatusNotFound).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}

	// Currently we only support viewing log files.
	if dbLogTarget.Output == "stdout" || dbLogTarget.Output == "stderr" ||
		strings.HasPrefix(dbLogTarget.Output, "syslog") {
		msg := fmt.Sprintf("Viewing log from %s is not supported", dbLogTarget.Output)
		log.Warn(msg)
		rsp := services.NewGetLogTailDefault(http.StatusBadRequest).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}

	// Set the maximum length of the data fetched. Default is 4000 bytes.
	maxLength := int64(4000)
	if params.MaxLength != nil {
		maxLength = *params.MaxLength
	}

	// Send the request to the agent to tail the file.
	contents, err := r.Agents.TailTextFile(ctx, dbLogTarget.Daemon.App.Machine, dbLogTarget.Output, maxLength)

	errStr := ""
	if err != nil {
		errStr = err.Error()
	}

	// Everything ok. Return the response.
	tail := &models.LogTail{
		Machine: &models.AppMachine{
			ID:       dbLogTarget.Daemon.App.MachineID,
			Address:  dbLogTarget.Daemon.App.Machine.Address,
			Hostname: dbLogTarget.Daemon.App.Machine.State.Hostname,
		},
		AppID:           storkutil.Ptr(dbLogTarget.Daemon.App.ID),
		AppName:         storkutil.Ptr(dbLogTarget.Daemon.App.Name),
		AppType:         storkutil.Ptr(dbLogTarget.Daemon.App.Type.String()),
		LogTargetOutput: storkutil.Ptr(dbLogTarget.Output),
		Contents:        contents,
		Error:           errStr,
	}
	rsp := services.NewGetLogTailOK().WithPayload(tail)

	return rsp
}
