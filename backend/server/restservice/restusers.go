package restservice

import (
	"context"
	"fmt"
	"strings"

	"github.com/go-openapi/runtime/middleware"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"isc.org/stork/server/database/model"
	"isc.org/stork/server/gen/models"
	"isc.org/stork/server/gen/restapi/operations/users"
)

// Creates new instance of the user model used by REST API from the
// user instance returned from the database.
func newRestUser(u dbmodel.SystemUser) *models.User {
	id := int64(u.Id)
	r := &models.User{
		Email:    &u.Email,
		Name:     &u.Name,
		ID:       &id,
		Lastname: &u.Lastname,
		Login:    &u.Login,
	}

	// Append an array of groups.
	for _, g := range u.Groups {
		if g.Id > 0 {
			r.Groups = append(r.Groups, int64(g.Id))
		}
	}

	return r
}

// Create new instance of the group model used by REST API from the
// group instance returned from the database.
func newRestGroup(g dbmodel.SystemGroup) *models.Group {
	id := int64(g.Id)
	r := &models.Group{
		ID:          &id,
		Name:        &g.Name,
		Description: &g.Description,
	}

	return r
}

// Attempts to login the user to the system.
func (r *RestAPI) CreateSession(ctx context.Context, params users.CreateSessionParams) middleware.Responder {
	user := &dbmodel.SystemUser{}
	login := *params.Useremail
	if strings.Contains(login, "@") {
		user.Email = login
	} else {
		user.Login = login
	}
	user.Password = *params.Userpassword

	ok, err := dbmodel.Authenticate(r.Db, user)
	if ok {
		err = r.SessionManager.LoginHandler(ctx, user)
	}

	if !ok || err != nil {
		if err != nil {
			log.Error(err)
		}
		return users.NewCreateSessionBadRequest()
	}

	rspUser := newRestUser(*user)
	return users.NewCreateSessionOK().WithPayload(rspUser)
}

// Attempts to logout a user from the system.
func (r *RestAPI) DeleteSession(ctx context.Context, params users.DeleteSessionParams) middleware.Responder {
	err := r.SessionManager.LogoutHandler(ctx)
	if err != nil {
		log.Error(err)
		return users.NewDeleteSessionBadRequest()
	}
	return users.NewDeleteSessionOK()
}

// Get users having an account in the system.
func (r *RestAPI) GetUsers(ctx context.Context, params users.GetUsersParams) middleware.Responder {
	systemUsers, total, err := dbmodel.GetUsers(r.Db, int(*params.Start), int(*params.Limit), dbmodel.SystemUserOrderById)
	if err != nil {
		log.WithFields(log.Fields{
			"start": int(*params.Start),
			"limit": int(*params.Limit),
		}).Errorf("failed to get users from the database with error: %s", err.Error())

		msg := "failed to get users from the database"
		rspErr := models.APIError{
			Message: &msg,
		}
		rsp := users.NewGetUsersDefault(500).WithPayload(&rspErr)
		return rsp
	}

	usersList := []*models.User{}
	for _, u := range systemUsers {
		usersList = append(usersList, newRestUser(*u))
	}

	u := models.Users{
		Items: usersList,
		Total: total,
	}
	rsp := users.NewGetUsersOK().WithPayload(&u)
	return rsp
}

// Returns user information by user ID.
func (r *RestAPI) GetUser(ctx context.Context, params users.GetUserParams) middleware.Responder {
	id := int(params.ID)
	su, err := dbmodel.GetUserById(r.Db, id)
	if err != nil {
		log.WithFields(log.Fields{
			"userid": id,
		}).Errorf("failed to fetch user with id %v from the database with error: %s", id,
			err.Error())

		msg := "failed to fetch user with id %v from the database"
		rspErr := models.APIError{
			Message: &msg,
		}
		rsp := users.NewGetUserDefault(500).WithPayload(&rspErr)
		return rsp

	} else if su == nil {
		msg := fmt.Sprintf("failed to find user with id %v in the database", id)
		log.WithFields(log.Fields{
			"userid": id,
		}).Error(msg)

		rspErr := models.APIError{
			Message: &msg,
		}
		rsp := users.NewGetUserDefault(404).WithPayload(&rspErr)
		return rsp
	}

	u := newRestUser(*su)
	return users.NewGetUserOK().WithPayload(u)
}

// Creates new user account in the database.
func (r *RestAPI) CreateUser(ctx context.Context, params users.CreateUserParams) middleware.Responder {
	u := params.Account.User
	p := params.Account.Password

	su := &dbmodel.SystemUser{
		Login:    *u.Login,
		Email:    *u.Email,
		Lastname: *u.Lastname,
		Name:     *u.Name,
		Password: string(p),
	}

	for _, gid := range u.Groups {
		su.Groups = append(su.Groups, &dbmodel.SystemGroup{Id: int(gid)})
	}

	err, con := dbmodel.CreateUser(r.Db, su)
	if err != nil {
		if con {
			log.WithFields(log.Fields{
				"login": *u.Login,
				"email": *u.Email,
			}).Infof("failed to create conflicting user account for user %s: %s", su.Identity(), err.Error())

			msg := "user account with provided login/email already exists"
			rspErr := models.APIError{
				Message: &msg,
			}
			return users.NewCreateUserDefault(409).WithPayload(&rspErr)

		} else {
			log.Errorf("failed to create new user account for user %s: %s", su.Identity(), err.Error())

			msg := fmt.Sprintf("failed to create new user account for user %s", su.Identity())
			rspErr := models.APIError{
				Message: &msg,
			}
			return users.NewCreateUserDefault(500).WithPayload(&rspErr)
		}
	}

	*u.ID = int64(su.Id)
	return users.NewCreateUserOK().WithPayload(u)
}

// Updates existing user account in the database.
func (r *RestAPI) UpdateUser(ctx context.Context, params users.UpdateUserParams) middleware.Responder {
	u := params.Account.User
	p := params.Account.Password

	su := &dbmodel.SystemUser{
		Id:       int(*u.ID),
		Login:    *u.Login,
		Email:    *u.Email,
		Lastname: *u.Lastname,
		Name:     *u.Name,
		Password: string(p),
	}

	for _, gid := range u.Groups {
		su.Groups = append(su.Groups, &dbmodel.SystemGroup{Id: int(gid)})
	}

	err, con := dbmodel.UpdateUser(r.Db, su)
	if con {
		log.WithFields(log.Fields{
			"userid": *u.ID,
		}).Infof("failed to update user account for user %s: %s", su.Identity(), err.Error())

		msg := "user account with provided login/email already exists"
		rspErr := models.APIError{
			Message: &msg,
		}
		rsp := users.NewUpdateUserDefault(409).WithPayload(&rspErr)
		return rsp

	} else if err != nil {
		log.WithFields(log.Fields{
			"userid": *u.ID,
			"login":  *u.Login,
			"email":  *u.Email,
		}).Errorf("failed to update user account for user %s: %s",
			su.Identity(), err.Error())

		msg := fmt.Sprintf("failed to update user account for user %s", su.Identity())
		rspErr := models.APIError{
			Message: &msg,
		}
		rsp := users.NewUpdateUserDefault(500).WithPayload(&rspErr)
		return rsp

	}

	return users.NewUpdateUserOK()
}

// Updates password of the given user in the database.
func (r *RestAPI) UpdateUserPassword(ctx context.Context, params users.UpdateUserPasswordParams) middleware.Responder {
	id := int(params.ID)
	passwords := params.Passwords

	// Try to change the password for the given user id. Including old password
	// for verification and the new password which will only be set if this
	// verification is successful.
	auth, err := dbmodel.ChangePassword(r.Db, id, string(passwords.Oldpassword),
		string(passwords.Newpassword))

	// Error is returned when something went wrong with the database communication
	// or something similar. The error is not returned when the current password
	// is not matching.
	if err != nil {
		err = errors.Wrapf(err, "failed to update password for user id %d", id)
		log.Error(err)

		msg := "database error while trying to update user password"
		rspErr := models.APIError{
			Message: &msg,
		}
		rsp := users.NewUpdateUserPasswordDefault(500).WithPayload(&rspErr)
		return rsp

	} else if !auth {
		log.Infof("specified current password is invalid while trying to update existing password for user id %d",
			id)

		msg := "invalid current password specified"
		rspErr := models.APIError{
			Message: &msg,
		}
		rsp := users.NewUpdateUserPasswordDefault(400).WithPayload(&rspErr)
		return rsp
	}

	// Password successfully changed.
	return users.NewUpdateUserPasswordOK()
}

// Get groups defined in the system.
func (r *RestAPI) GetGroups(ctx context.Context, params users.GetGroupsParams) middleware.Responder {
	systemGroups, err := dbmodel.GetGroups(r.Db)
	if err != nil {
		log.Errorf("failed to get groups from the database with error: %s", err.Error())

		msg := "failed to get groups from the database"
		rspErr := models.APIError{
			Message: &msg,
		}
		rsp := users.NewGetGroupsDefault(500).WithPayload(&rspErr)
		return rsp
	}

	groupsList := []*models.Group{}
	for _, g := range systemGroups {
		groupsList = append(groupsList, newRestGroup(*g))
	}

	g := models.Groups{
		Items: groupsList,
		Total: int64(len(groupsList)),
	}
	rsp := users.NewGetGroupsOK().WithPayload(&g)
	return rsp
}
