package auth

import (
	"isc.org/stork/server/database/model"
	"net/http"
	"regexp"
	"strconv"
)

func Authorize(user *dbmodel.SystemUser, req *http.Request) (ok bool, err error) {
	if user == nil || len(user.Groups) == 0 || req == nil {
		return false, nil
	}

	if user.InGroup(&dbmodel.SystemGroup{Id: dbmodel.SuperAdminGroupId}) {
		return true, nil
	}

	if ok, _ := regexp.Match(`users/{0,}`, []byte(req.URL.Path)); ok {
		ok, _ := regexp.Match(`users/{1,}`+strconv.Itoa(user.Id)+`/*`, []byte(req.URL.Path))
		return ok, nil
	}

	return true, err
}
