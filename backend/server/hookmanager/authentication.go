package hookmanager

import (
	"context"
	"reflect"

	"isc.org/stork/hooks/server/authenticationcallout"
	"isc.org/stork/hooksutil"
	dbmodel "isc.org/stork/server/database/model"
	"isc.org/stork/server/gen/restapi/operations/users"
)

// Interface checks.
var _ authenticationcallout.AuthenticationCallout = (*HookManager)(nil)

func (hm *HookManager) HasAuthenticationHook() bool {
	return hm.executor.HasRegistered(reflect.TypeOf((*authenticationcallout.AuthenticationCallout)(nil)).Elem())
}

func (hm *HookManager) Authenticate(ctx context.Context, params users.CreateSessionParams) (*dbmodel.SystemUser, error) {
	data := hooksutil.CallSingle(hm.executor, func(callout authenticationcallout.AuthenticationCallout) struct {
		user *dbmodel.SystemUser
		err  error
	} {
		user, err := callout.Authenticate(ctx, params)
		return struct {
			user *dbmodel.SystemUser
			err  error
		}{
			user: user,
			err:  err,
		}
	})
	return data.user, data.err
}

func (hm *HookManager) Unauthenticate(ctx context.Context) error {
	return hooksutil.CallSingle(hm.executor, func(callout authenticationcallout.AuthenticationCallout) error {
		return callout.Unauthenticate(ctx)
	})
}
