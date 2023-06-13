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

	ok, data := hooksutil.CallSequentialUntilProcessed(hm.GetExecutor(), func(carrier authenticationcallouts.AuthenticationCallouts) (hooksutil.CallStatus, *output) {
		if carrier.GetMetadata().GetID() != authenticationMethodID {
			// Go to next authentication callout.
			return hooksutil.CallStatusSkipped, nil
		}

		user, err := carrier.Authenticate(ctx, request, identifier, secret)
		err = errors.Wrap(err, "error occurred in the Authenticate callout")
		return hooksutil.CallStatusProcessed, &output{
			user: user,
			err:  err,
		}
	})

	if !ok {
		return nil, errors.Errorf("the '%s' authentication method is not supported", authenticationMethodID)
	}
	return data.user, data.err
}

// Callout to unauthenticate a user (close session). It can be used to notify
// the external authentication provider.
func (hm *HookManager) Unauthenticate(ctx context.Context, authenticationMethodID string) error {
	_, err := hooksutil.CallSequentialUntilProcessed(hm.GetExecutor(), func(carrier authenticationcallouts.AuthenticationCallouts) (hooksutil.CallStatus, error) {
		if carrier.GetMetadata().GetID() != authenticationMethodID {
			// Go to next authentication callout.
			return hooksutil.CallStatusSkipped, nil
		}
		return hooksutil.CallStatusProcessed, carrier.Unauthenticate(ctx)
	})
	err = errors.Wrap(err, "error occurred in the Unauthenticate callout")
	return err
}

// Callout to obtain the metadata of the authentication method provided by a hook.
func (hm *HookManager) GetAuthenticationMetadata() []authenticationcallouts.AuthenticationMetadata {
	return hooksutil.CallSequential(hm.GetExecutor(), func(carrier authenticationcallouts.AuthenticationCallouts) authenticationcallouts.AuthenticationMetadata {
		return carrier.GetMetadata()
	})
}
