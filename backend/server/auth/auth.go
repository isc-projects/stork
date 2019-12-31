package auth

import (
	"net/http"
	"regexp"
	"strconv"

	dbmodel "isc.org/stork/server/database/model"
)

// Checks if the given user is permitted to access a resource. Currently the
// access pattern is very simple, the super-admin user can access all
// resources. The admin-user can access all resources except those related
// to users management.
func Authorize(user *dbmodel.SystemUser, req *http.Request) (ok bool, err error) {
	// If there is no user (possibly the user has not signed in), the user
	// does not belong to any groups or the request is nil, reject access to
	// the resource.
	if user == nil || len(user.Groups) == 0 || req == nil {
		return false, nil
	}

	// If the user is super-admin he can access all resources.
	if user.InGroup(&dbmodel.SystemGroup{ID: dbmodel.SuperAdminGroupID}) {
		return true, nil
	}

	// If the user does not belong to the super-admin group and trying to
	// access the user specific information, check if the data the user
	// is trying to access belong to this user. If not, reject access.
	if ok, _ := regexp.Match(`users/{0,}`, []byte(req.URL.Path)); ok {
		ok, _ := regexp.Match(`users/{1,}`+strconv.Itoa(user.ID)+`/*`, []byte(req.URL.Path))
		return ok, nil
	}

	// All other resources can be accessed by the admin user.
	if user.InGroup(&dbmodel.SystemGroup{ID: dbmodel.AdminGroupID}) {
		return true, err
	}

	// User who doesn't belong to any group is not allowed to access
	// system resources.
	return false, nil
}
