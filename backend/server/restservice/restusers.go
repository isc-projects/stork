package restservice

import (
	"context"
	"fmt"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/go-openapi/runtime/middleware"

	"isc.org/stork/server/database/model"
	"isc.org/stork/server/gen/models"
	"isc.org/stork/server/gen/restapi/operations/users"
)

// Creates new instance of the user model used by REST API from the
// user instance returned from the database.
func NewRestUser(u dbmodel.SystemUser) *models.User {
	id := int64(u.Id)
	r := &models.User{
		Email: &u.Email,
		Name: &u.Name,
		ID: &id,
		Lastname: &u.Lastname,
		Login: &u.Login,
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

	ok, err := dbmodel.Authenticate(r.Db, user);
	if ok {
		err = r.SessionManager.LoginHandler(ctx, user)
	}

	if !ok || err != nil {
		if err != nil {
			log.Error(err)
		}
		return users.NewCreateSessionBadRequest()
	}

	rspUserId := int64(user.Id)
	rspUser := models.User{
		ID: &rspUserId,
		Login: &user.Login,
		Email: &user.Email,
		Name: &user.Name,
		Lastname: &user.Lastname,
	}

	return users.NewCreateSessionOK().WithPayload(&rspUser)
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
	systemUsers, total, err := dbmodel.GetUsersByPage(r.Db, int(*params.Start), int(*params.Limit), dbmodel.SystemUserOrderById)
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
		usersList = append(usersList, NewRestUser(u))
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

	u := NewRestUser(*su)
	return users.NewGetUserOK().WithPayload(u)
}

// Creates new user account in the database.
func (r *RestAPI) CreateUser(ctx context.Context, params users.CreateUserParams) middleware.Responder {
	u := params.Account.User
	p := params.Account.Password

	su := &dbmodel.SystemUser{
		Login: *u.Login,
		Email: *u.Email,
		Lastname: *u.Lastname,
		Name: *u.Name,
		Password: string(p),
	}
	err, con := su.Persist(r.Db)
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
		Id: int(*u.ID),
		Login: *u.Login,
		Email: *u.Email,
		Lastname: *u.Lastname,
		Name: *u.Name,
		Password: string(p),
	}
	err, con := su.Persist(r.Db)
	if (con) {
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
			"login": *u.Login,
			"email": *u.Email,
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
