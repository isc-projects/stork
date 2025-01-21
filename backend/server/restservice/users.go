package restservice

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-openapi/runtime/middleware"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"isc.org/stork/hooks/server/authenticationcallouts"
	dbmodel "isc.org/stork/server/database/model"
	"isc.org/stork/server/gen/models"
	"isc.org/stork/server/gen/restapi/operations/users"
	storkutil "isc.org/stork/util"
)

// Creates new instance of the user model used by REST API from the
// user instance returned from the database.
func newRestUser(u dbmodel.SystemUser) *models.User {
	id := int64(u.ID)
	r := &models.User{
		Email:                  u.Email,
		Name:                   u.Name,
		ID:                     &id,
		Lastname:               u.Lastname,
		Login:                  u.Login,
		AuthenticationMethodID: &u.AuthenticationMethodID,
		ExternalID:             u.ExternalID,
		Groups:                 []int64{},
		ChangePassword:         u.ChangePassword,
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

// The internal authentication flow based on the login and password stored in
// the database.
func (r *RestAPI) internalAuthentication(params users.CreateSessionParams) (*dbmodel.SystemUser, error) {
	user := &dbmodel.SystemUser{}
	var identifier, secret string

	if params.Credentials.Identifier != nil {
		identifier = *params.Credentials.Identifier
	}

	if strings.Contains(identifier, "@") {
		user.Email = identifier
	} else {
		user.Login = identifier
	}

	if params.Credentials.Secret != nil {
		secret = *params.Credentials.Secret
	}

	ok, err := dbmodel.Authenticate(r.DB, user, secret)
	if !ok {
		return nil, err
	}
	return user, err
}

// The internal authentication flow handled by the hooks.
func (r *RestAPI) externalAuthentication(ctx context.Context, params users.CreateSessionParams) (*dbmodel.SystemUser, error) {
	calloutUser, err := r.HookManager.Authenticate(
		ctx,
		params.HTTPRequest,
		*params.Credentials.AuthenticationMethodID,
		params.Credentials.Identifier,
		params.Credentials.Secret,
	)

	if calloutUser == nil || err != nil {
		return nil, errors.WithMessage(err, "cannot authenticate a user")
	}

	groupIDMapping := map[authenticationcallouts.UserGroupID]int{
		authenticationcallouts.UserGroupIDSuperAdmin: dbmodel.SuperAdminGroupID,
		authenticationcallouts.UserGroupIDAdmin:      dbmodel.AdminGroupID,
	}

	var groups []*dbmodel.SystemGroup
	for _, g := range calloutUser.Groups {
		systemGroupID, ok := groupIDMapping[g]
		if !ok {
			log.Warningf("Unknown group returned from hook: %d", g)
			continue
		}

		groups = append(groups, &dbmodel.SystemGroup{
			ID: systemGroupID,
		})
	}

	systemUser := &dbmodel.SystemUser{
		Login:                  calloutUser.Login,
		Email:                  calloutUser.Email,
		Lastname:               calloutUser.Lastname,
		Name:                   calloutUser.Name,
		Groups:                 groups,
		AuthenticationMethodID: *params.Credentials.AuthenticationMethodID,
		ExternalID:             calloutUser.ID,
		ChangePassword:         false,
	}

	conflict, err := dbmodel.CreateUser(r.DB, systemUser)
	if conflict {
		var dbUser *dbmodel.SystemUser
		dbUser, err = dbmodel.GetUserByExternalID(
			r.DB,
			*params.Credentials.AuthenticationMethodID,
			calloutUser.ID,
		)
		if err != nil {
			return nil, errors.Errorf("cannot fetch the internal user profile")
		}

		systemUser.ID = dbUser.ID

		if calloutUser.Groups == nil {
			// The groups are not managed by the hook.
			systemUser.Groups = dbUser.Groups
		}

		_, err = dbmodel.UpdateUser(r.DB, systemUser)
	}
	return systemUser, err
}

// Attempts to login the user to the system.
func (r *RestAPI) CreateSession(ctx context.Context, params users.CreateSessionParams) middleware.Responder {
	var systemUser *dbmodel.SystemUser
	var err error

	if params.Credentials == nil {
		log.Warning(("Cannot authenticate a user due to missing credentials"))
		return users.NewCreateSessionBadRequest()
	}

	// Extract the authentication method and normalize the value.
	var authenticationMethod string
	if params.Credentials.AuthenticationMethodID == nil || *params.Credentials.AuthenticationMethodID == "" {
		authenticationMethod = dbmodel.AuthenticationMethodIDInternal
	} else {
		authenticationMethod = *params.Credentials.AuthenticationMethodID
	}

	if authenticationMethod == dbmodel.AuthenticationMethodIDInternal {
		systemUser, err = r.internalAuthentication(params)
	} else {
		systemUser, err = r.externalAuthentication(ctx, params)
	}

	if systemUser == nil && err == nil {
		log.
			WithField("method", authenticationMethod).
			WithField("identifier", *params.Credentials.Identifier).
			Error("User not found, cannot authenticate")
		return users.NewCreateSessionBadRequest()
	} else if err != nil {
		log.
			WithError(err).
			WithField("method", authenticationMethod).
			WithField("identifier", *params.Credentials.Identifier).
			Error("Cannot authenticate a user")
		return users.NewCreateSessionBadRequest()
	}

	err = r.SessionManager.LoginHandler(ctx, systemUser)
	if err != nil {
		log.
			WithError(err).
			WithField("identifier", *params.Credentials.Identifier).
			Error("Cannot log in a user")
		return users.NewCreateSessionBadRequest()
	}

	rspUser := newRestUser(*systemUser)
	return users.NewCreateSessionOK().WithPayload(rspUser)
}

// Attempts to logout a user from the system.
func (r *RestAPI) DeleteSession(ctx context.Context, params users.DeleteSessionParams) middleware.Responder {
	ok, user := r.SessionManager.Logged(ctx)
	if !ok {
		return users.NewDeleteSessionBadRequest()
	}

	err := r.SessionManager.LogoutHandler(ctx)
	if err != nil {
		log.WithError(err).Error("Cannot logout user")
		return users.NewDeleteSessionBadRequest()
	}

	_ = r.HookManager.Unauthenticate(ctx, user.AuthenticationMethodID)
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
	var start int64
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
		}).WithError(err).Error("Failed to get users from the database")

		msg := "Failed to get users from the database"
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
			"userID": id,
		}).WithError(err).Errorf("Failed to fetch user with ID %d from the database", id)

		msg := fmt.Sprintf("Failed to fetch user with ID %d from the database", id)
		rspErr := models.APIError{
			Message: &msg,
		}
		rsp := users.NewGetUserDefault(http.StatusInternalServerError).WithPayload(&rspErr)
		return rsp
	}
	if su == nil {
		msg := fmt.Sprintf("Failed to find user with ID %d in the database", id)
		log.WithField("userID", id).Error(msg)

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
		log.Warn("Failed to create new user account: missing data")

		msg := "Failed to create new user account: missing data"
		rspErr := models.APIError{
			Message: &msg,
		}
		return users.NewCreateUserDefault(http.StatusBadRequest).WithPayload(&rspErr)
	}
	u := params.Account.User
	p := params.Account.Password

	if u == nil || p == nil || u.Name == "" || u.Lastname == "" {
		msg := "Failed to create new user account: missing data"
		log.Warn(msg)
		rspErr := models.APIError{Message: &msg}
		return users.NewCreateUserDefault(http.StatusBadRequest).WithPayload(&rspErr)
	} else if u.Login == "" && u.Email == "" {
		msg := "Failed to create new user account: missing identifier"
		log.Warn(msg)
		rspErr := models.APIError{Message: &msg}
		return users.NewCreateUserDefault(http.StatusBadRequest).WithPayload(&rspErr)
	}

	su := &dbmodel.SystemUser{
		Login:          u.Login,
		Email:          u.Email,
		Lastname:       u.Lastname,
		Name:           u.Name,
		ChangePassword: u.ChangePassword,
	}

	for _, gid := range u.Groups {
		su.Groups = append(su.Groups, &dbmodel.SystemGroup{ID: int(gid)})
	}

	con, err := dbmodel.CreateUserWithPassword(r.DB, su, string(*p))
	if err != nil {
		if con {
			log.
				WithField("user", su.Identity()).
				WithError(err).
				Info("Failed to create conflicting user account")

			msg := "User account with provided login/email already exists"
			rspErr := models.APIError{
				Message: &msg,
			}
			return users.NewCreateUserDefault(http.StatusConflict).WithPayload(&rspErr)
		}
		log.
			WithField("user", su.Identity()).
			WithError(err).
			Error("Failed to create new user account")

		msg := fmt.Sprintf("Failed to create new user account for user %s", su.Identity())
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
		log.Warn("Failed to update user account: missing data")

		msg := "Failed to update user account: missing data"
		rspErr := models.APIError{
			Message: &msg,
		}
		return users.NewUpdateUserDefault(http.StatusBadRequest).WithPayload(&rspErr)
	}
	u := params.Account.User
	p := params.Account.Password

	if u == nil || u.ID == nil {
		msg := "Failed to update user account: missing data"
		log.Warn(msg)
		rspErr := models.APIError{
			Message: &msg,
		}
		return users.NewUpdateUserDefault(http.StatusBadRequest).WithPayload(&rspErr)
	} else if u.Login == "" && u.Email == "" {
		msg := "Failed to create new user account: missing identifier"
		log.Warn(msg)
		rspErr := models.APIError{Message: &msg}
		return users.NewCreateUserDefault(http.StatusBadRequest).WithPayload(&rspErr)
	}

	isInternal := u.AuthenticationMethodID == nil || *u.AuthenticationMethodID == dbmodel.AuthenticationMethodIDInternal
	if isInternal && (u.Name == "" || u.Lastname == "") {
		msg := "Failed to update user account: missing first or last name"
		log.Warn(msg)
		rspErr := models.APIError{
			Message: &msg,
		}
		return users.NewUpdateUserDefault(http.StatusBadRequest).WithPayload(&rspErr)
	}

	su := &dbmodel.SystemUser{
		ID:             int(*u.ID),
		Login:          u.Login,
		Email:          u.Email,
		Lastname:       u.Lastname,
		Name:           u.Name,
		ChangePassword: u.ChangePassword,
	}

	for _, gid := range u.Groups {
		su.Groups = append(su.Groups, &dbmodel.SystemGroup{ID: int(gid)})
	}

	con, err := dbmodel.UpdateUser(r.DB, su)
	if con {
		log.WithField("userID", *u.ID).WithError(err).Infof("Failed to update user account for user %s", su.Identity())

		msg := "User account with provided login/email already exists"
		rspErr := models.APIError{
			Message: &msg,
		}
		rsp := users.NewUpdateUserDefault(http.StatusConflict).WithPayload(&rspErr)
		return rsp
	} else if err != nil {
		log.WithFields(log.Fields{
			"userID": *u.ID,
			"login":  u.Login,
			"email":  u.Email,
		}).WithError(err).Errorf("Failed to update user account for user %s", su.Identity())

		msg := fmt.Sprintf("Failed to update user account for user %s", su.Identity())
		rspErr := models.APIError{
			Message: &msg,
		}
		rsp := users.NewUpdateUserDefault(http.StatusInternalServerError).WithPayload(&rspErr)
		return rsp
	}

	password := ""
	if p != nil {
		password = string(*p)
	}
	if password != "" {
		err = dbmodel.SetPassword(r.DB, int(*u.ID), password)
		if err != nil {
			log.WithFields(log.Fields{
				"userID": *u.ID,
				"login":  u.Login,
				"email":  u.Email,
			}).WithError(err).Errorf("Failed to update password for user %s", su.Identity())

			msg := fmt.Sprintf("Failed to update password for user %s", su.Identity())
			rspErr := models.APIError{
				Message: &msg,
			}
			rsp := users.NewUpdateUserDefault(http.StatusInternalServerError).WithPayload(&rspErr)
			return rsp
		}
	}

	return users.NewUpdateUserOK()
}

// Deletes existing user account from the database.
func (r *RestAPI) DeleteUser(ctx context.Context, params users.DeleteUserParams) middleware.Responder {
	id := int(params.ID)

	_, currentUser := r.SessionManager.Logged(ctx)
	if currentUser.ID == id {
		log.WithField("userID", id).Infof("Failed to delete user account for logged in user %s", currentUser.Identity())

		msg := "User account with provided login/email tries to delete itself"
		rspErr := models.APIError{
			Message: &msg,
		}
		rsp := users.NewDeleteUserDefault(http.StatusBadRequest).WithPayload(&rspErr)
		return rsp
	}

	su, err := dbmodel.GetUserByID(r.DB, id)
	if err != nil {
		log.WithField("userID", id).
			WithError(err).
			Errorf("Failed to fetch user with ID %d from the database", id)

		msg := fmt.Sprintf("Failed to fetch user with ID %d from the database", id)
		rspErr := models.APIError{
			Message: &msg,
		}
		rsp := users.NewDeleteUserDefault(http.StatusInternalServerError).WithPayload(&rspErr)
		return rsp
	}
	if su == nil {
		msg := fmt.Sprintf("Failed to find user with ID %d in the database", id)
		log.WithField("userID", id).Error(msg)

		rspErr := models.APIError{
			Message: &msg,
		}
		rsp := users.NewDeleteUserDefault(http.StatusNotFound).WithPayload(&rspErr)
		return rsp
	}

	err = dbmodel.DeleteUser(r.DB, su)
	if err != nil {
		log.WithFields(log.Fields{
			"userID": id,
			"login":  su.Login,
			"email":  su.Email,
		}).WithError(err).Errorf("Failed to delete user account for user %s",
			su.Identity())

		msg := fmt.Sprintf("Failed to delete user account for user %s", su.Identity())
		rspErr := models.APIError{
			Message: &msg,
		}
		rsp := users.NewDeleteUserDefault(http.StatusInternalServerError).WithPayload(&rspErr)
		return rsp
	}

	err = r.SessionManager.LogoutUser(ctx, su)
	if err != nil {
		log.WithFields(log.Fields{
			"userID": id,
			"login":  su.Login,
			"email":  su.Email,
		}).WithError(err).Errorf("Failed to logout user account for user %s", su.Identity())

		msg := fmt.Sprintf("Failed to logout user account for user %s", su.Identity())
		rspErr := models.APIError{
			Message: &msg,
		}
		rsp := users.NewDeleteUserDefault(http.StatusInternalServerError).WithPayload(&rspErr)
		return rsp
	}

	u := newRestUser(*su)
	return users.NewDeleteUserOK().WithPayload(u)
}

// Updates password of the given user in the database.
func (r *RestAPI) UpdateUserPassword(ctx context.Context, params users.UpdateUserPasswordParams) middleware.Responder {
	id := int(params.ID)
	passwords := params.Passwords
	if passwords == nil || passwords.Newpassword == nil || passwords.Oldpassword == nil {
		log.Warnf("Failed to update password for user ID %d: missing data", id)

		msg := "Failed to update password for user: missing data"
		rspErr := models.APIError{
			Message: &msg,
		}
		rsp := users.NewUpdateUserPasswordDefault(http.StatusBadRequest).WithPayload(&rspErr)
		return rsp
	}

	// Try to change the password for the given user id. Including old password
	// for verification and the new password which will only be set if this
	// verification is successful.
	err := dbmodel.ChangePassword(r.DB, id, string(*passwords.Oldpassword),
		string(*passwords.Newpassword))

	// Error is returned when something went wrong with the database communication
	// or something similar. The error is not returned when the current password
	// is not matching.
	if errors.Is(err, dbmodel.ErrInvalidPassword) {
		log.Infof("Specified invalid current password while trying to update existing password for user ID %d",
			id)

		rspErr := models.APIError{
			Message: storkutil.Ptr("Invalid current password specified"),
		}
		rsp := users.NewUpdateUserPasswordDefault(http.StatusBadRequest).WithPayload(&rspErr)
		return rsp
	} else if err != nil {
		log.WithError(err).Errorf("Failed to update password for user ID %d", id)
		rspErr := models.APIError{
			Message: storkutil.Ptr("Database error while trying to update user password"),
		}
		rsp := users.NewUpdateUserPasswordDefault(http.StatusInternalServerError).WithPayload(&rspErr)
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
	var start int64
	if params.Start != nil {
		start = *params.Start
	}

	var limit int64 = 10
	if params.Limit != nil {
		limit = *params.Limit
	}

	groups, err := r.getGroups(start, limit, params.Text, "", dbmodel.SortDirAny)
	if err != nil {
		log.Errorf("Failed to get groups from the database with error: %s", err.Error())

		msg := "Failed to get groups from the database"
		rspErr := models.APIError{
			Message: &msg,
		}
		rsp := users.NewGetGroupsDefault(http.StatusInternalServerError).WithPayload(&rspErr)
		return rsp
	}

	rsp := users.NewGetGroupsOK().WithPayload(groups)
	return rsp
}

// Get authentication methods supported by server. Endpoint is allowed without log in.
func (r *RestAPI) GetAuthenticationMethods(ctx context.Context, params users.GetAuthenticationMethodsParams) middleware.Responder {
	metadata := r.HookManager.GetAuthenticationMetadata()

	var methods []*models.AuthenticationMethod

	for _, meta := range metadata {
		method := &models.AuthenticationMethod{
			ID:          meta.GetID(),
			Description: meta.GetDescription(),
			Name:        meta.GetName(),
		}

		if metaForm, ok := meta.(authenticationcallouts.AuthenticationMetadataForm); ok {
			method.FormLabelIdentifier = metaForm.GetIdentifierFormLabel()
			method.FormLabelSecret = metaForm.GetSecretFormLabel()
		}

		methods = append(methods, method)
	}

	methods = append(methods, &models.AuthenticationMethod{
		ID:                  "internal",
		Name:                "Internal",
		Description:         "Internal Stork authentication based on credentials from the internal database",
		FormLabelIdentifier: "Email/Login",
		FormLabelSecret:     "Password",
	})

	return users.NewGetAuthenticationMethodsOK().WithPayload(&models.AuthenticationMethods{
		Items: methods,
		Total: int64(len(methods)),
	})
}
