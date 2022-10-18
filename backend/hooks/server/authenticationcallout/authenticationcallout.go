package authenticationcallout

import (
	"context"
	"net/http"

	"isc.org/stork/hooks"
)

// The logged user metadata. It's a data transfer object (DTO) to avoid using
// heavy dbmodel dependencies.
type User struct {
	// It must be a unique and persistent ID.
	ID       int64
	Login    string
	Email    string
	Lastname string
	Name     string
	// It must contain internal Stork group IDs. It means that the hook should
	// map the authentication API identifiers.
	Groups []int
}

// Set of callout points used to perform authentication.
type AuthenticationCallout interface {
	hooks.Callout
	// Called to perform authentication. It accepts an HTTP request (header,
	// cookie) and the credentials provided in the login form. Returns a user
	// metadata or error if an authentication failed.
	// A session ID (if applicable) may be stored in the context.
	Authenticate(ctx context.Context, request *http.Request, email, password *string) (*User, error)
	// Called to perform unauthentication (closing the session). It accepts the
	// context passed previously to the authentication callout point.
	Unauthenticate(ctx context.Context) error
}
