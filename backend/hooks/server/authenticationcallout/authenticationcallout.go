package authenticationcallout

import (
	"context"

	dbmodel "isc.org/stork/server/database/model"
	"isc.org/stork/server/gen/restapi/operations/users"
)

type AuthenticationCallout interface {
	Authenticate(ctx context.Context, params users.CreateSessionParams) (*dbmodel.SystemUser, error)
	Unauthenticate(ctx context.Context) error
}
