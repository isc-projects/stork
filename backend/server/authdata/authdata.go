package authdata

import "io"

// User group ID enum.
type UserGroupID int

// List of the user group IDs used in the server.
const (
	UserGroupIDSuperAdmin UserGroupID = 1
	UserGroupIDAdmin      UserGroupID = 2
	UserGroupIDReadOnly   UserGroupID = 3
)

// The authenticated user metadata. It's a data transfer object (DTO) to avoid using
// heavy dbmodel dependencies when exchanging authenticated user data between
// external authenticator and core server code.
type User struct {
	// It must be a unique and persistent ID.
	ID       string
	Login    string
	Email    string
	Lastname string
	Name     string
	// It must contain internal Stork group IDs. It means that the external authenticator should
	// map the authentication API identifiers.
	Groups                  []UserGroupID
	ExternallyManagedGroups bool // If external authenticator wants to manage user groups, this should be set to true.
}

// The metadata of the authentication method is used to display a selector on
// the login page. All external authentication methods (LDAP, OIDC, etc.)
// should implement this interface.
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
