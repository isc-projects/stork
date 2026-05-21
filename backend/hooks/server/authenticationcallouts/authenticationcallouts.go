package authenticationcallouts

import (
	"context"
	"net/http"

	"isc.org/stork/server/authdata"
)

// The metadata of the authentication method that uses the data from the login
// form. Currently, we expect that LDAP method implements this interface.
// The non-form flow (e.g.: OpenID Connect, Basic Auth, Bearer Token, JWT) should
// not implement it.
type AuthenticationMetadataForm interface {
	// Returns a label for the identifier field in the login form on UI.
	GetIdentifierFormLabel() string
	// Returns a label for the secret field in the login form on UI.
	GetSecretFormLabel() string
}

// Set of callouts used to perform authentication.
type AuthenticationCallouts interface {
	// Called to perform authentication. It accepts an HTTP request (header,
	// cookie) and the credentials provided in the login form. Returns a user
	// metadata or error if an authentication failed.
	// A session ID (if applicable) may be stored in the context.
	Authenticate(ctx context.Context, request *http.Request, identifier, secret *string) (*authdata.User, error)
	// Called to perform unauthentication (closing the session). It accepts the
	// context passed previously to the authentication callout.
	Unauthenticate(ctx context.Context) error
	// Returns authentication metadata used to list the authentication methods
	// on the UI. The metadata object should also implement the interface
	// specific to the flow of providing credentials
	// (e.g.: "AuthenticationMetadataForm" - for form-based).
	GetMetadata() authdata.AuthenticationMetadata
}
