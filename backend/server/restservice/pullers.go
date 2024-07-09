package restservice

import (
	"context"
	"fmt"
	"net/http"
	"reflect"
	"time"

	"github.com/go-openapi/runtime/middleware"
	"github.com/go-openapi/strfmt"
	"isc.org/stork/server/agentcomm"
	"isc.org/stork/server/gen/models"
	"isc.org/stork/server/gen/restapi/operations/settings"
)

// Allows accessing the metadata of the periodic puller.
type pullerMetadata interface {
	GetName() string
	GetIntervalSettingName() string
	GetInterval() time.Duration
	GetLastInvokedAt() time.Time
	GetLastFinishedAt() time.Time
}

var _ pullerMetadata = (*agentcomm.PeriodicPuller)(nil)

// Returns a list of puller statuses.
func (r *RestAPI) GetPullers(ctx context.Context, params settings.GetPullersParams) middleware.Responder {
	v := reflect.ValueOf(*r.Pullers)

	pullers := []*models.Puller{}

	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		if !field.CanInterface() || field.IsNil() {
			continue
		}
		puller, ok := field.Interface().(pullerMetadata)
		if !ok {
			continue
		}

		metadata := &models.Puller{
			Name:           puller.GetName(),
			ID:             puller.GetIntervalSettingName(),
			Interval:       int64(puller.GetInterval().Truncate(time.Second).Seconds()),
			LastInvokedAt:  strfmt.DateTime(puller.GetLastInvokedAt()),
			LastFinishedAt: strfmt.DateTime(puller.GetLastFinishedAt()),
		}

		pullers = append(pullers, metadata)
	}

	rsp := settings.NewGetPullersOK().WithPayload(&models.Pullers{
		Items: pullers,
		Total: int64(len(pullers)),
	})
	return rsp
}

// Returns a specific puller status.
func (r *RestAPI) GetPuller(ctx context.Context, params settings.GetPullerParams) middleware.Responder {
	v := reflect.ValueOf(*r.Pullers)

	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		if !field.CanInterface() || field.IsNil() {
			continue
		}

		puller, ok := field.Interface().(pullerMetadata)
		if !ok {
			continue
		}

		pullerID := puller.GetIntervalSettingName()

		if params.ID != pullerID {
			continue
		}

		metadata := &models.Puller{
			Name:           puller.GetName(),
			ID:             puller.GetIntervalSettingName(),
			Interval:       int64(puller.GetInterval().Truncate(time.Second).Seconds()),
			LastInvokedAt:  strfmt.DateTime(puller.GetLastInvokedAt()),
			LastFinishedAt: strfmt.DateTime(puller.GetLastFinishedAt()),
		}

		rsp := settings.NewGetPullerOK().WithPayload(metadata)
		return rsp
	}

	msg := fmt.Sprintf("Cannot get puller with ID %s", params.ID)
	rsp := settings.NewGetPullerDefault(http.StatusNotFound).WithPayload(&models.APIError{
		Message: &msg,
	})
	return rsp
}
