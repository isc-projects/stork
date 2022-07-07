package main

import (
	"github.com/pkg/errors"
	"isc.org/stork/hooks/server/authenticationcallout"
	dbmodel "isc.org/stork/server/database/model"
	"isc.org/stork/server/gen/restapi/operations/users"
)

type callouts struct {
}

var _ authenticationcallout.AuthenticationCallout = (*callouts)(nil)

func (c *callouts) Close() error {
	return nil
}

func (c *callouts) GetAuthenticationDetails() *authenticationcallout.AuthenticationMethodDetails {
	return &authenticationcallout.AuthenticationMethodDetails{
		Name:        "LDAP",
		Description: "LDAP authentication",
	}
}

func (c *callouts) Authenticate(params users.CreateSessionParams) (*dbmodel.SystemUser, error) {
	if params.Credentials.Useremail == nil || params.Credentials.Userpassword == nil {
		return nil, errors.Errorf("missing email or password")
	}
	if *params.Credentials.Useremail != "secret" || *params.Credentials.Userpassword != "secret" {
		return nil, errors.Errorf("invalid user or password")
	}

	return &dbmodel.SystemUser{
		ID:       0,
		Login:    "Secretary",
		Email:    "secretary@office.com",
		Lastname: "Bob",
		Name:     "Alice",
		Password: "",
		Groups:   []*dbmodel.SystemGroup{},
	}, nil
}

func (c *callouts) Unauthenticate(user *dbmodel.SystemUser) error {
	return nil
}
