package restservice

import (
	"context"
	"fmt"
	"net/http"

	"github.com/go-openapi/runtime/middleware"
	"github.com/go-openapi/strfmt"
	"isc.org/stork/server/configmigrator"
	"isc.org/stork/server/gen/models"
	dhcp "isc.org/stork/server/gen/restapi/operations/d_h_c_p"
	storkutil "isc.org/stork/util"
)

// Converts the status returned by a service to the format used in REST API.
func (r *RestAPI) convertMigrationStatusToRestAPI(status *configmigrator.MigrationStatus) *models.MigrationStatus {
	errs := []*models.MigrationError{}
	for _, err := range status.Errors {
		errs = append(errs, &models.MigrationError{
			Error: err.Error.Error(),
			ID:    err.ID,
			Label: err.Label,
			Type:  string(err.Type),
		})
	}
	var generalError *string
	if status.GeneralError != nil {
		generalError = storkutil.Ptr(status.GeneralError.Error())
	}

	// Retrieve some details from the context passed when the migration was started.
	var userLogin string
	var userID int64
	if ok, user := r.SessionManager.Logged(status.Context); ok {
		userLogin = user.Login
		userID = int64(user.ID)
	}

	return &models.MigrationStatus{
		Canceling:   status.Canceling,
		AuthorID:    userID,
		AuthorLogin: userLogin,
		ElapsedTime: strfmt.Duration(status.ElapsedTime),
		EndDate:     convertToOptionalDatetime(status.EndDate),
		Errors: &models.MigrationErrors{
			Items: errs,
			Total: int64(len(errs)),
		},
		EstimatedLeftTime:   strfmt.Duration(status.EstimatedLeftTime),
		GeneralError:        generalError,
		ID:                  int64(status.ID),
		ProcessedItemsCount: status.ProcessedItemsCount,
		TotalItemsCount:     status.TotalItemsCount,
		StartDate:           strfmt.DateTime(status.StartDate),
	}
}

// Implements the GET call to retrieve the statuses of all (ongoing or
// completed) migrations.
func (r *RestAPI) GetMigrations(ctx context.Context, params dhcp.GetMigrationsParams) middleware.Responder {
	// Fetch migration statuses from the migration service.
	statuses := r.MigrationService.GetMigrations()

	// Convert statuses to REST API format.
	respStatuses := []*models.MigrationStatus{}
	for _, status := range statuses {
		respStatuses = append(respStatuses, r.convertMigrationStatusToRestAPI(status))
	}

	// Send the statuses to the client.
	rsp := dhcp.NewGetMigrationsOK().WithPayload(&models.MigrationStatuses{
		Items: respStatuses,
		Total: int64(len(respStatuses)),
	})
	return rsp
}

// Implements the DELETE call to remove all finished migrations.
func (r *RestAPI) DeleteFinishedMigrations(ctx context.Context, params dhcp.DeleteFinishedMigrationsParams) middleware.Responder {
	// Remove finished migrations from the migration service.
	r.MigrationService.ClearFinishedMigrations()

	// Send OK response to the client.
	rsp := dhcp.NewDeleteFinishedMigrationsOK()
	return rsp
}

// Implements the GET call to retrieve the status of a specific migration.
func (r *RestAPI) GetMigration(ctx context.Context, params dhcp.GetMigrationParams) middleware.Responder {
	// Fetch migration status from the migration service.
	status, ok := r.MigrationService.GetMigration(configmigrator.MigrationIdentifier(params.ID))
	if !ok {
		msg := fmt.Sprintf("Cannot find migration status with ID %d", params.ID)
		rsp := dhcp.NewGetMigrationDefault(http.StatusNotFound).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}

	migrationStatus := r.convertMigrationStatusToRestAPI(status)
	rsp := dhcp.NewGetMigrationOK().WithPayload(migrationStatus)
	return rsp
}

// Implements the PUT call to cancel an ongoing migration.
func (r *RestAPI) PutMigration(ctx context.Context, params dhcp.PutMigrationParams) middleware.Responder {
	// Attempt to cancel the migration.
	status, ok := r.MigrationService.StopMigration(configmigrator.MigrationIdentifier(params.ID))
	if !ok {
		msg := fmt.Sprintf("Cannot find migration status with ID %d", params.ID)
		rsp := dhcp.NewPutMigrationDefault(http.StatusNotFound).WithPayload(&models.APIError{
			Message: &msg,
		})
		return rsp
	}

	// Send OK response to the client.
	rsp := dhcp.NewPutMigrationOK().WithPayload(r.convertMigrationStatusToRestAPI(status))
	return rsp
}
