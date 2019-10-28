package dbmodel

// Represents a user held in system_user table in the database.
type SystemUser struct {
	Id           int
	Email        string
	Lastname     string
	Name         string
	PasswordHash string
}

