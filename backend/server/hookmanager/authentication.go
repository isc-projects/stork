package hookmanager

import (
	"context"
	"net/http"

	"github.com/pkg/errors"
	"isc.org/stork/hooks/server/authenticationcallouts"
	"isc.org/stork/hooksutil"
)

// Callout to authenticate the user based on HTTP request (headers, cookie)
// and the credentials provided in the login form (email, password).
func (hm *HookManager) Authenticate(ctx context.Context, request *http.Request, authenticationMethodID string, identifier, secret *string) (*authenticationcallouts.User, error) {
	type output struct {
		user *authenticationcallouts.User
		err  error
	}

	data := hooksutil.CallUntilSuccess(hm.GetExecutor(), func(carrier authenticationcallouts.AuthenticationCallouts) *output {
		if carrier.GetMetadata().GetID() != authenticationMethodID {
			// Go to next authentication callout.
			return nil
		}

		user, err := carrier.Authenticate(ctx, request, identifier, secret)
		return &output{
			user: user,
			err:  err,
		}
	})

	if data == nil {
		return nil, errors.Errorf("the '%s' authentication method is not supported", authenticationMethodID)
	}
	return data.user, data.err
}

// Callout to unauthenticate a user (close session). It accepts a context that
// should contain the session ID set up in the Authenticate callout.
func (hm *HookManager) Unauthenticate(ctx context.Context) error {
	return hooksutil.CallUntilSuccess(hm.GetExecutor(), func(carrier authenticationcallouts.AuthenticationCallouts) error {
		return carrier.Unauthenticate(ctx)
	})
}

func (hm *HookManager) GetAuthenticationMetadata() []authenticationcallouts.AuthenticationMetadata {
	return hooksutil.CallSequential(hm.GetExecutor(), func(carrier authenticationcallouts.AuthenticationCallouts) authenticationcallouts.AuthenticationMetadata {
		return carrier.GetMetadata()
	})
}
