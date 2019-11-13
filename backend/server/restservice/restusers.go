package restservice

import (
	"context"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/go-openapi/runtime/middleware"

	"isc.org/stork/server/database/model"
	"isc.org/stork/server/gen/models"
	"isc.org/stork/server/gen/restapi/operations/users"
)

// Creates new instance of the user model used by REST API from the
// user instance returned from the database.
func NewRestUser(u *dbmodel.SystemUser) *models.User {
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

	ok, err := dbmodel.Authenticate(r.PgDB, user);
	if ok {
		err = r.SessionManager.LoginHandler(ctx, user)
	}

	if !ok || err != nil {
		if err != nil {
			log.Printf("%+v", err)
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
		log.Printf("%+v", err)
		return users.NewDeleteSessionBadRequest()
	}
	return users.NewDeleteSessionOK()
}

// Get users having an account in the system.
func (r *RestAPI) GetUsers(ctx context.Context, params users.GetUsersParams) middleware.Responder {
	system_users, err := dbmodel.GetUsers(r.PgDB, int(*params.Start), int(*params.Limit), dbmodel.SystemUserOrderById)
	if err != nil {
		msg := err.Error()
		rspErr := models.APIError{
			Code: 500,
			Message: &msg,
		}
		rsp := users.NewGetUsersDefault(500).WithPayload(&rspErr)
		return rsp
	}

	usersList := []*models.User{}
	for _, u := range system_users {
		usersList = append(usersList, NewRestUser(&u))
	}

	u := models.Users{
		Items: usersList,
		Total: int64(len(usersList)),
	}
	rsp := users.NewGetUsersOK().WithPayload(&u)
	return rsp
}

