package authdata

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
