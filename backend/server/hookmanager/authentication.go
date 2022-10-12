package hookmanager

import (
	"context"
	"net/http"
	"reflect"

	"isc.org/stork/hooks/server/authenticationcallout"
	"isc.org/stork/hooksutil"
)

// Interface checks.
var _ authenticationcallout.AuthenticationCallout = (*HookManager)(nil)

func (hm *HookManager) HasAuthenticationHook() bool {
	return hm.executor.HasRegistered(reflect.TypeOf((*authenticationcallout.AuthenticationCallout)(nil)).Elem())
}

func (hm *HookManager) Authenticate(ctx context.Context, request *http.Request, email, password *string) (*authenticationcallout.User, error) {
	type output struct {
		user *authenticationcallout.User
		err  error
	}

	data := hooksutil.CallSingle(hm.executor, func(callout authenticationcallout.AuthenticationCallout) output {
		user, err := callout.Authenticate(ctx, request, email, password)
		return output{
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
