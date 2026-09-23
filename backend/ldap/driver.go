package ldap

import (
	"crypto/tls"
	"time"

	goldap "github.com/go-ldap/ldap/v3"
	"github.com/pkg/errors"
)

// The go-ldap proxy interface for the mocking purposes.
type LDAPDriver interface {
	Dial(url string, tls *tls.Config, timeout time.Duration) error
	Close()
	SimpleBind(userDN, password string, allowEmptyPassword bool) error
	Search(request *goldap.SearchRequest) (*goldap.SearchResult, error)
}

// Low-level component to communicate with the LDAP server.
// It is a production implementation of the driver that passes through all
// commands to the go-ldap library.
type libraryDriver struct {
	connection      *goldap.Conn
	enableDebugging bool
}

// Construct new LDAP driver for the production purposes.
func NewLDAPLibraryDriver(enableDebugging bool) LDAPDriver {
	return &libraryDriver{enableDebugging: enableDebugging}
}

// Establishes connection to the LDAP server.
func (d *libraryDriver) Dial(url string, tls *tls.Config, timeout time.Duration) error {
	var opts []goldap.DialOpt
	if tls != nil {
		opts = append(opts, goldap.DialWithTLSConfig(tls))
	}

	connection, err := goldap.DialURL(url, opts...)
	if err != nil {
		err = errors.Errorf("cannot establish connection to the LDAP server (%s): %w", url, err)
		return err
	}

	connection.Debug.Enable(d.enableDebugging)
	connection.SetTimeout(timeout)

	d.connection = connection
	return nil
}

// Closes the connection to the LDAP server.
func (d *libraryDriver) Close() {
	if d.connection != nil {
		d.connection.Close()
		d.connection = nil
	}
}

// Authorizes in the LDAP server using the given credentials.
func (d *libraryDriver) SimpleBind(userDN, password string, allowEmptyPassword bool) error {
	_, err := d.connection.SimpleBind(&goldap.SimpleBindRequest{
		Username:           userDN,
		Password:           password,
		AllowEmptyPassword: allowEmptyPassword,
	})
	if err != nil {
		return errors.Errorf("cannot bind the '%s' user: %w", userDN, err)
	}
	return nil
}

// Performs searching in the LDAP server.
func (d *libraryDriver) Search(request *goldap.SearchRequest) (*goldap.SearchResult, error) {
	return d.connection.Search(request)
}
