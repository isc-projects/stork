package ldap

import (
	"crypto/tls"
	"fmt"
	"log"
	"strings"

	goldap "github.com/go-ldap/ldap/v3"
	"github.com/pkg/errors"
	"isc.org/stork/server/authdata"
)

// Errors.
var (
	errLDAPSearchUnexpectedEntries = errors.New("LDAP search returned unexpected entries")
	errLDAPMissingUserAttribute    = errors.New("LDAP user entry is missing required attribute")
)

// High-level component to interact with the LDAP server. It provides
// domain-specific methods to perform various actions on LDAP. It internally
// converts the LDAP structures into Stork-compatible ones.
type LDAPController struct {
	settings Settings
	driver   LDAPDriver
}

// Constructs an LDAP controller instance. Accepts the settings and low-level
// driver to perform actions on LDAP.
func NewLDAPController(settings Settings, driver LDAPDriver) *LDAPController {
	return &LDAPController{settings: settings, driver: driver}
}

// Close the active connection.
func (c *LDAPController) Close() {
	c.driver.Close()
}

// Establishes connection to the LDAP server and configure the connection properties.
func (c *LDAPController) Connect() error {
	// Enable TLS if necessary.
	var tlsConfig *tls.Config
	if strings.HasPrefix(c.settings.DialURL, "ldaps://") {
		// The LDAP server may be configured with a self-signed (untrusted) SSL
		// certificate for testing purposes.
		if c.settings.TLSSkipServerVerification {
			log.Print(
				"LDAP hook is configured to skip verification of the TLS " +
					"server certificate - it is insecure and not recommended " +
					"for the production environment",
			)
		}
		tlsConfig = &tls.Config{
			InsecureSkipVerify: c.settings.TLSSkipServerVerification, //nolint:gosec
		}
	}

	// Connect to the server.
	// Setup connection.
	return c.driver.Dial(c.settings.DialURL, tlsConfig, c.settings.Timeout)
}

// Authorizes the current connection in the LDAP server as the bind user.
func (c *LDAPController) BindAsMaintenanceUser() error {
	return c.driver.SimpleBind(
		c.settings.BindUserDN,
		c.settings.BindPassword,
		c.settings.BindPassword == "",
	)
}

// Searches for some attributes in LDAP.
func (c *LDAPController) searchForAttributes(filter string, attributes []string) ([]*goldap.Entry, error) {
	searchRequest := goldap.NewSearchRequest(
		c.settings.Root,
		goldap.ScopeWholeSubtree, goldap.NeverDerefAliases, 0, 0, false,
		filter, attributes,
		nil,
	)

	searchResponse, err := c.driver.Search(searchRequest)
	if err != nil {
		return nil, errors.WithMessagef(err,
			"searching for attributes (%s) failed",
			strings.Join(attributes, ","),
		)
	}

	return searchResponse.Entries, nil
}

// Searches for attributes for a given user in LDAP.
func (c *LDAPController) searchForAttributesByUsername(username string, attributes []string) (*goldap.Entry, error) {
	entries, err := c.searchForAttributes(
		fmt.Sprintf(
			"(&(objectClass=%s)(%s=%s))",
			c.settings.AttributeNames.ObjectClassUser,
			c.settings.AttributeNames.UserID, goldap.EscapeFilter(username),
		),
		attributes,
	)
	if err != nil {
		return nil, errors.WithMessagef(err,
			"failed search for user (%s) attributes",
			username,
		)
	}
	if len(entries) == 0 {
		return nil, errors.WithMessagef(errLDAPSearchUnexpectedEntries, "no data found")
	}
	if len(entries) > 1 {
		return nil, errors.WithMessagef(
			errLDAPSearchUnexpectedEntries,
			"too many user entries returned for a user (%s), got %d",
			username, len(entries),
		)
	}
	return entries[0], nil
}

// Extract the value of a given attribute from the LDAP entry. It returns an
// empty string if the attribute is missing.
// Handles the special case for the "dn" attribute, which is not returned along
// with other attributes in the search result, but is a field of the entry.
func (c *LDAPController) getAttributeValue(entry *goldap.Entry, attributeName string) string {
	if attributeName == "dn" {
		return entry.DN
	}
	return entry.GetAttributeValue(attributeName)
}

// Searches for a given user profile in LDAP.
// Returns a user profile without groups.
func (c *LDAPController) SearchForUserProfile(username string) (*authdata.User, error) {
	attributes := []string{
		c.settings.AttributeNames.FirstName, c.settings.AttributeNames.LastName,
		c.settings.AttributeNames.Email, c.settings.AttributeNames.UniqueIdentifier,
	}

	entry, err := c.searchForAttributesByUsername(username, attributes)
	if err != nil {
		err = errors.WithMessagef(err, "searching for user profile failed")
		return nil, err
	}

	id := c.getAttributeValue(entry, c.settings.AttributeNames.UniqueIdentifier)
	if id == "" {
		return nil, errors.WithMessagef(
			errLDAPMissingUserAttribute,
			"missing unique identifier attribute (%s)",
			c.settings.AttributeNames.UniqueIdentifier,
		)
	}

	return &authdata.User{
		ID:       id,
		Login:    username,
		Email:    c.getAttributeValue(entry, c.settings.AttributeNames.Email),
		Name:     c.getAttributeValue(entry, c.settings.AttributeNames.FirstName),
		Lastname: c.getAttributeValue(entry, c.settings.AttributeNames.LastName),
	}, nil
}
