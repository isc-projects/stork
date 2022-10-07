package authenticationcallout

import (
	"context"
	"net/http"
)

type User struct {
	ID       int64
	Login    string
	Email    string
	Lastname string
	Name     string
	Groups   []int
}

type AuthenticationCallout interface {
	Authenticate(ctx context.Context, request *http.Request, email, password *string) (*User, error)
	Unauthenticate(ctx context.Context) error
}
