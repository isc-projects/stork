package authenticationcallout

import (
	dbmodel "isc.org/stork/server/database/model"
	"isc.org/stork/server/gen/restapi/operations/users"
)

type AuthenticationMethodDetails struct {
	Name        string
	Description string
}

type AuthenticationCallout interface {
	GetAuthenticationDetails() *AuthenticationMethodDetails
	Authenticate(params users.CreateSessionParams) (*dbmodel.SystemUser, error)
	Unauthenticate(user *dbmodel.SystemUser) error
}
