package main

import (
	"context"
	"errors"
	"net/http"

	"isc.org/stork/server/callouts/authenticationcallout"
)

type user struct {
	id       int64
	login    string
	email    string
	lastName string
	name     string
	groups   []int
}

func (u *user) GetID() int64 {
	return u.id
}

func (u *user) GetLogin() string {
	return u.login
}

func (u *user) GetEmail() string {
	return u.email
}

func (u *user) GetLastName() string {
	return u.lastName
}

func (u *user) GetName() string {
	return u.name
}

func (u *user) GetGroups() []int {
	return u.groups
}

type callout struct{}

func (c *callout) Close() error {
	return nil
}

var _ authenticationcallout.AuthenticationCallout = (*callout)(nil)

func (c *callout) Authenticate(ctx context.Context, request *http.Request, email, password *string) (*user, error) {
	if email == nil || password == nil {
		return nil, errors.New("missing email or password")
	}
	if *email != "secret" || *password != "secret" {
		return nil, errors.New("invalid user or password")
	}

	return &user{
		id:       1,
		login:    "Secretary",
		email:    "secretary@office.com",
		lastName: "Bob",
		name:     "Alice",
		groups:   []int{1},
	}, nil
}

func (c *callout) Unauthenticate(ctx context.Context) error {
	return nil
}
