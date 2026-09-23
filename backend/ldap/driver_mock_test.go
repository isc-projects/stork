package ldap_test

import (
	"crypto/tls"
	"time"

	goldap "github.com/go-ldap/ldap/v3"
)

// Contains the name of the called method and its arguments.
type call struct {
	name      string
	arguments map[string]any
}

// The mock of LDAPDriver for testing purposes. It records all calls to the
// methods and allows to return predefined values.
type mockDriver struct {
	calls             []call
	returnDial        error
	returnSimpleBind  error
	returnSearchValue *goldap.SearchResult
	returnSearchErr   error
}

// Constructs new mock driver.
func newMockDriver() *mockDriver {
	return &mockDriver{}
}

// Mocks the Dial method. Records the call and returns the predefined value.
func (d *mockDriver) Dial(url string, tls *tls.Config, timeout time.Duration) error {
	d.calls = append(d.calls, call{"Dial", map[string]any{
		"url":     url,
		"tls":     tls,
		"timeout": timeout,
	}})
	return d.returnDial
}

// Mocks the Close method. Records the call.
func (d *mockDriver) Close() {
	d.calls = append(d.calls, call{"Close", map[string]any{}})
}

// Mocks the SimpleBind method. Records the call and returns the predefined value.
func (d *mockDriver) SimpleBind(userDN, password string, allowEmptyPassword bool) error {
	d.calls = append(d.calls, call{"SimpleBind", map[string]any{
		"userDN":             userDN,
		"password":           password,
		"allowEmptyPassword": allowEmptyPassword,
	}})
	return d.returnSimpleBind
}

// Mocks the Search method. Records the call and returns the predefined value.
func (d *mockDriver) Search(request *goldap.SearchRequest) (*goldap.SearchResult, error) {
	d.calls = append(d.calls, call{"Search", map[string]any{"request": request}})
	return d.returnSearchValue, d.returnSearchErr
}
