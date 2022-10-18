package main

import (
	"context"
	"errors"
	"net/http"

	"isc.org/stork/hooks/server/authenticationcallout"
)

type callout struct{}

var _ authenticationcallout.AuthenticationCallout = (*callout)(nil)

func (c *callout) Close() error {
	return nil
}

func (c *callout) Authenticate(ctx context.Context, request *http.Request, email, password *string) (*authenticationcallout.User, error) {
	if email == nil || password == nil {
		return nil, errors.New("missing email or password")
	}
	if *email != "secret" || *password != "secret" {
		return nil, errors.New("invalid user or password")
	}

	return &authenticationcallout.User{
		ID:       1,
		Login:    "Secretary",
		Email:    "secretary@office.com",
		Lastname: "Bob",
		Name:     "Alice",
		Groups:   []int{1},
	}, nil
}

func (c *callout) Unauthenticate(ctx context.Context) error {
	return nil
}
