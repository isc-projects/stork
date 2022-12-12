package hookmanager

import (
	"context"
	"net/http"
	"reflect"

	"isc.org/stork/hooks/server/authenticationcallouts"
	"isc.org/stork/hooksutil"
)

// Interface checks.
var _ authenticationcallouts.AuthenticationCallouts = (*HookManager)(nil)

// Checks if the authentication hook was registered.
func (hm *HookManager) HasAuthenticationHook() bool {
	return hm.GetExecutor().HasRegistered(reflect.TypeOf((*authenticationcallouts.AuthenticationCallouts)(nil)).Elem())
}

// Callout point to authenticate the user based on HTTP request (headers, cookie)
// and the credentials provided in the login form (email, password).
func (hm *HookManager) Authenticate(ctx context.Context, request *http.Request, email, password *string) (*authenticationcallouts.User, error) {
	type output struct {
		user *authenticationcallouts.User
		err  error
	}

	// We assume that only one authentication hook can be used.
	// It's a design decision. Technically, it's possible to authorize different
	// users with different methods.
	data := hooksutil.CallSingle(hm.GetExecutor(), func(callout authenticationcallouts.AuthenticationCallouts) output {
		user, err := callout.Authenticate(ctx, request, email, password)
		return output{
			user: user,
			err:  err,
		}
	})
	return data.user, data.err
}

// Callout point to unauthenticate a user (close session). It accepts a context
// that should contain the session IDCallout point to unauthenticate a user
// (close session). It accepts a context that should contain the session ID set
// up in the Authenticate callout point.
func (hm *HookManager) Unauthenticate(ctx context.Context) error {
	return hooksutil.CallSingle(hm.GetExecutor(), func(callout authenticationcallouts.AuthenticationCallouts) error {
		return callout.Unauthenticate(ctx)
	})
}
