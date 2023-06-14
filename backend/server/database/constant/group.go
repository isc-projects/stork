package constant

// User group ID enum.
type UserGroupID int

// List of the user group IDs used in the server.
const (
	SuperAdminGroupID UserGroupID = 1
	AdminGroupID      UserGroupID = 2
)
