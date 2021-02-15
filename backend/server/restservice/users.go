package restservice

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-openapi/runtime/middleware"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	dbmodel "isc.org/stork/server/database/model"
	"isc.org/stork/server/gen/models"
	"isc.org/stork/server/gen/restapi/operations/users"
)

// Creates new instance of the user model used by REST API from the
// user instance returned from the database.
func newRestUser(u dbmodel.SystemUser) *models.User {
	id := int64(u.ID)
	r := &models.User{
		Email:    &u.Email,
		Name:     &u.Name,
		ID:       &id,
		Lastname: &u.Lastname,
		Login:    &u.Login,
	}

	// Append an array of groups.
	for _, g := range u.Groups {
		if g.ID > 0 {
			r.Groups = append(r.Groups, int64(g.ID))
		}
	}

	return r
}

// Create new instance of the group model used by REST API from the
// group instance returned from the database.
func newRestGroup(g dbmodel.SystemGroup) *models.Group {
	id := int64(g.ID)
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

	var login string
	if params.Credentials.Useremail != nil {
		login = *params.Credentials.Useremail
	}
	if strings.Contains(login, "@") {
		user.Email = login
	} else {
		user.Login = login
	}
	if params.Credentials.Userpassword != nil {
		user.Password = *params.Credentials.Userpassword
	}

	ok, err := dbmodel.Authenticate(r.DB, user)
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

func (r *RestAPI) getUsers(offset, limit int64, filterText *string, sortField string, sortDir dbmodel.SortDirEnum) (*models.Users, error) {
	dbUsers, total, err := dbmodel.GetUsersByPage(r.DB, offset, limit, filterText, sortField, sortDir)
	if err != nil {
		return nil, err
	}
	users := &models.Users{
		Total: total,
	}
	for _, u := range dbUsers {
		users.Items = append(users.Items, newRestUser(u))
	}
	return users, nil
}

// Get users having an account in the system.
func (r *RestAPI) GetUsers(ctx context.Context, params users.GetUsersParams) middleware.Responder {
	var start int64 = 0
	if params.Start != nil {
		start = *params.Start
	}

	var limit int64 = 10
	if params.Limit != nil {
		limit = *params.Limit
	}

	usersRsp, err := r.getUsers(start, limit, params.Text, "", dbmodel.SortDirAny)
	if err != nil {
		log.WithFields(log.Fields{
			"start": start,
			"limit": limit,
		}).Errorf("failed to get users from the database with error: %s", err.Error())

		msg := "failed to get users from the database"
		rspErr := models.APIError{
			Message: &msg,
		}
		rsp := users.NewGetUsersDefault(http.StatusInternalServerError).WithPayload(&rspErr)
		return rsp
	}

	rsp := users.NewGetUsersOK().WithPayload(usersRsp)
	return rsp
}

// Returns user information by user ID.
func (r *RestAPI) GetUser(ctx context.Context, params users.GetUserParams) middleware.Responder {
	id := int(params.ID)
	su, err := dbmodel.GetUserByID(r.DB, id)
	if err != nil {
		log.WithFields(log.Fields{
			"userid": id,
		}).Errorf("failed to fetch user with id %v from the database with error: %s", id,
			err.Error())

		msg := "failed to fetch user with id %v from the database"
		rspErr := models.APIError{
			Message: &msg,
		}
		rsp := users.NewGetUserDefault(http.StatusInternalServerError).WithPayload(&rspErr)
		return rsp
	} else if su == nil {
		msg := fmt.Sprintf("failed to find user with id %v in the database", id)
		log.WithFields(log.Fields{
			"userid": id,
		}).Error(msg)

		rspErr := models.APIError{
			Message: &msg,
		}
		rsp := users.NewGetUserDefault(http.StatusNotFound).WithPayload(&rspErr)
		return rsp
	}

	u := newRestUser(*su)
	return users.NewGetUserOK().WithPayload(u)
}

// Creates new user account in the database.
func (r *RestAPI) CreateUser(ctx context.Context, params users.CreateUserParams) middleware.Responder {
	if params.Account == nil {
		log.Warnf("failed to create new user account: missing data")

		msg := "failed to create new user account: missing data"
		rspErr := models.APIError{
			Message: &msg,
		}
		return users.NewCreateUserDefault(http.StatusBadRequest).WithPayload(&rspErr)
	}
	u := params.Account.User
	p := params.Account.Password

	if u == nil || u.Login == nil || u.Email == nil || u.Lastname == nil || u.Name == nil {
		log.Warnf("failed to create new user account: missing data")

		msg := "failed to create new user account: missing data"
		rspErr := models.APIError{
			Message: &msg,
		}
		return users.NewCreateUserDefault(http.StatusBadRequest).WithPayload(&rspErr)
	}

	su := &dbmodel.SystemUser{
		Login:    *u.Login,
		Email:    *u.Email,
		Lastname: *u.Lastname,
		Name:     *u.Name,
		Password: string(p),
	}

	for _, gid := range u.Groups {
		su.Groups = append(su.Groups, &dbmodel.SystemGroup{ID: int(gid)})
	}

	con, err := dbmodel.CreateUser(r.DB, su)
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
			return users.NewCreateUserDefault(http.StatusConflict).WithPayload(&rspErr)
		}
		log.Errorf("failed to create new user account for user %s: %s", su.Identity(), err.Error())

		msg := fmt.Sprintf("failed to create new user account for user %s", su.Identity())
		rspErr := models.APIError{
			Message: &msg,
		}
		return users.NewCreateUserDefault(http.StatusInternalServerError).WithPayload(&rspErr)
	}

	*u.ID = int64(su.ID)
	return users.NewCreateUserOK().WithPayload(u)
}

// Updates existing user account in the database.
func (r *RestAPI) UpdateUser(ctx context.Context, params users.UpdateUserParams) middleware.Responder {
	if params.Account == nil {
		log.Warnf("failed to update user account: missing data")

		msg := "failed to update user account: missing data"
		rspErr := models.APIError{
			Message: &msg,
		}
		return users.NewUpdateUserDefault(http.StatusBadRequest).WithPayload(&rspErr)
	}
	u := params.Account.User
	p := params.Account.Password

	if u == nil || u.ID == nil || u.Login == nil || u.Email == nil || u.Lastname == nil || u.Name == nil {
		log.Warnf("failed to update user account: missing data")

		msg := "failed to update user account: missing data"
		rspErr := models.APIError{
			Message: &msg,
		}
		return users.NewUpdateUserDefault(http.StatusBadRequest).WithPayload(&rspErr)
	}

	su := &dbmodel.SystemUser{
		ID:       int(*u.ID),
		Login:    *u.Login,
		Email:    *u.Email,
		Lastname: *u.Lastname,
		Name:     *u.Name,
		Password: string(p),
	}

	for _, gid := range u.Groups {
		su.Groups = append(su.Groups, &dbmodel.SystemGroup{ID: int(gid)})
	}

	con, err := dbmodel.UpdateUser(r.DB, su)
	if con {
		log.WithFields(log.Fields{
			"userid": *u.ID,
		}).Infof("failed to update user account for user %s: %s", su.Identity(), err.Error())

		msg := "user account with provided login/email already exists"
		rspErr := models.APIError{
			Message: &msg,
		}
		rsp := users.NewUpdateUserDefault(http.StatusConflict).WithPayload(&rspErr)
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
		rsp := users.NewUpdateUserDefault(http.StatusInternalServerError).WithPayload(&rspErr)
		return rsp
	}

	return users.NewUpdateUserOK()
}

// Updates password of the given user in the database.
func (r *RestAPI) UpdateUserPassword(ctx context.Context, params users.UpdateUserPasswordParams) middleware.Responder {
	id := int(params.ID)
	passwords := params.Passwords
	if passwords == nil {
		log.Warnf("failed to update password for user id %d: missing data", id)

		msg := "failed to update password for user: missing data"
		rspErr := models.APIError{
			Message: &msg,
		}
		rsp := users.NewUpdateUserPasswordDefault(http.StatusBadRequest).WithPayload(&rspErr)
		return rsp
	}

	// Try to change the password for the given user id. Including old password
	// for verification and the new password which will only be set if this
	// verification is successful.
	auth, err := dbmodel.ChangePassword(r.DB, id, string(passwords.Oldpassword),
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
		rsp := users.NewUpdateUserPasswordDefault(http.StatusInternalServerError).WithPayload(&rspErr)
		return rsp
	} else if !auth {
		log.Infof("specified current password is invalid while trying to update existing password for user id %d",
			id)

		msg := "invalid current password specified"
		rspErr := models.APIError{
			Message: &msg,
		}
		rsp := users.NewUpdateUserPasswordDefault(http.StatusBadRequest).WithPayload(&rspErr)
		return rsp
	}

	// Password successfully changed.
	return users.NewUpdateUserPasswordOK()
}

func (r *RestAPI) getGroups(offset, limit int64, filterText *string, sortField string, sortDir dbmodel.SortDirEnum) (*models.Groups, error) {
	dbGroups, total, err := dbmodel.GetGroupsByPage(r.DB, offset, limit, filterText, sortField, sortDir)
	if err != nil {
		return nil, err
	}

	groups := &models.Groups{
		Total: total,
	}
	for _, g := range dbGroups {
		groups.Items = append(groups.Items, newRestGroup(g))
	}
	return groups, nil
}

// Get groups defined in the system.
func (r *RestAPI) GetGroups(ctx context.Context, params users.GetGroupsParams) middleware.Responder {
	var start int64 = 0
	if params.Start != nil {
		start = *params.Start
	}

	var limit int64 = 10
	if params.Limit != nil {
		limit = *params.Limit
	}

	groups, err := r.getGroups(start, limit, params.Text, "", dbmodel.SortDirAny)
	if err != nil {
		log.Errorf("failed to get groups from the database with error: %s", err.Error())

		msg := "failed to get groups from the database"
		rspErr := models.APIError{
			Message: &msg,
		}
		rsp := users.NewGetGroupsDefault(http.StatusInternalServerError).WithPayload(&rspErr)
		return rsp
	}

	rsp := users.NewGetGroupsOK().WithPayload(groups)
	return rsp
}
