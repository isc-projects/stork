package authenticationcallouts

import (
	"context"
	"io"
	"net/http"
)

// The metadata of the authentication method is used to display a selector on
// the login page.
type AuthenticationMetadata interface {
	// Returns unique, fixed ID of the authentication method.
	GetID() string
	// Returns a name of the authentication method.
	GetName() string
	// Returns a description of the authentication method.
	GetDescription() string
	// Returns an icon of the authentication method in the PNG format.
	// The resource should be open only on demand. Caller is responsible for
	// closing the reader.
	GetIcon() (io.ReadCloser, error)
}

// The metadata of the authentication method that uses the data from the login
// form. Currently, we expect that all methods implement this interface.
// The non-form flow (e.g.: OpenID Connect, Basic Auth, Bearer Token, JWT) are
// not supported.
type AuthenticationMetadataForm interface {
	// Returns a label for the identifier field in the login form on UI.
	GetIdentifierFormLabel() string
	// Returns a label for the secret field in the login form on UI.
	GetSecretFormLabel() string
}

// User group ID enum.
type UserGroupID int

// List of the user group IDs used in the server.
const (
	UserGroupIDSuperAdmin UserGroupID = 1
	UserGroupIDAdmin      UserGroupID = 2
)

// The logged user metadata. It's a data transfer object (DTO) to avoid using
// heavy dbmodel dependencies.
type User struct {
	// It must be a unique and persistent ID.
	ID       string
	Login    string
	Email    string
	Lastname string
	Name     string
	// It must contain internal Stork group IDs. It means that the hook should
	// map the authentication API identifiers.
	Groups []UserGroupID
}

// Set of callouts used to perform authentication.
type AuthenticationCallouts interface {
	// Called to perform authentication. It accepts an HTTP request (header,
	// cookie) and the credentials provided in the login form. Returns a user
	// metadata or error if an authentication failed.
	// A session ID (if applicable) may be stored in the context.
	Authenticate(ctx context.Context, request *http.Request, identifier, secret *string) (*User, error)
	// Called to perform unauthentication (closing the session). It accepts the
	// context passed previously to the authentication callout.
	Unauthenticate(ctx context.Context) error
	// Returns authentication metadata used to list the authentication methods
	// on the UI. The metadata object should also implement the interface
	// specific to the flow of providing credentials
	// (e.g.: "AuthenticationMetadataForm" - for form-based).
	// Note: Other flows are unsupported but expected in the future as
	// redirecting to external authentication provider, Single Sign On, Basic
	// Auth, Token-based authentication, or Multi-Factor authentication
	GetMetadata() AuthenticationMetadata
}
