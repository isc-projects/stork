package dbmodel

import (
	"errors"
)

// An error indicating that the searched entry was not found.
var ErrNotExists = errors.New("database entry not found")
