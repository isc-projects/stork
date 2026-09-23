package ldap

import (
	"crypto/tls"
	"errors"
	"fmt"
	"log"
	"strings"

	goldap "github.com/go-ldap/ldap/v3"
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
		return nil, fmt.Errorf(
			"searching for attributes (%s) failed: %w",
			strings.Join(attributes, ","),
			err,
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
		return nil, fmt.Errorf(
			"failed search for user (%s) attributes: %w",
			username, err,
		)
	}
	if len(entries) == 0 {
		return nil, fmt.Errorf("no data found: %w", errLDAPSearchUnexpectedEntries)
	}
	if len(entries) > 1 {
		return nil, fmt.Errorf(
			"too many user entries returned for a user (%s), got %d: %w",
			username, len(entries), errLDAPSearchUnexpectedEntries,
		)
	}
	return entries[0], nil
}

// Searches for a given user DN in LDAP.
func (c *LDAPController) SearchForUserDN(username string) (string, error) {
	entry, err := c.searchForAttributesByUsername(username, []string{"dn"})
	if err != nil {
		err = fmt.Errorf("searching for user DN failed: %w", err)
		return "", err
	}

	return entry.DN, nil
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
		err = fmt.Errorf("searching for user profile failed: %w", err)
		return nil, err
	}

	id := c.getAttributeValue(entry, c.settings.AttributeNames.UniqueIdentifier)
	if id == "" {
		return nil, fmt.Errorf(
			"missing unique identifier attribute (%s): %w",
			c.settings.AttributeNames.UniqueIdentifier,
			errLDAPMissingUserAttribute,
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

// Authorizes the current connection in the LDAP server as the provided user.
// Accepts the user DN (not username) and password.
// Returns an error if the password is invalid.
func (c *LDAPController) BindAsUser(userDN, password string) error {
	// Bind as the user to verify their password
	return c.driver.SimpleBind(userDN, password, false)
}

// Searches for a group membership of a given user.
// Accepts user DN. Returns LDAP groups mapped into Stork groups and the
// occurrence of the allow group or error.
func (c *LDAPController) SearchForUserGroupMembership(userDN string) ([]authdata.UserGroupID, bool, error) {
	entries, err := c.searchForAttributes(
		fmt.Sprintf(
			"(&(objectClass=%s)(%s=%s))",
			c.settings.AttributeNames.ObjectClassGroup,
			c.settings.AttributeNames.GroupMember,
			goldap.EscapeFilter(userDN),
		),
		[]string{c.settings.AttributeNames.GroupCommonName},
	)
	if err != nil {
		return nil, false, fmt.Errorf(
			"searching for group membership failed: %w",
			err,
		)
	}

	groups := []authdata.UserGroupID{}
	allowed := false
	for _, entry := range entries {
		groupCommonName := c.getAttributeValue(entry, c.settings.AttributeNames.GroupCommonName)
		if groupCommonName == c.settings.MandatoryAllowGroup {
			allowed = true
		}
		c.checkMultipleGroupMembership(groupCommonName, &groups)
	}
	return groups, allowed, nil
}

// Utility function which checks whether given group common name is any of
// the configured mapped LDAP groups. The hook allows to configure multiple LDAP
// groups for mapping to each of Stork groups - they must be separated by a comma -
// so this function checks all the multiple LDAP groups.
// It takes pointer to slice of Stork group IDs and mutates the slice by appending there all found
// group memberships of given group common name.
func (c *LDAPController) checkMultipleGroupMembership(groupCN string, groups *[]authdata.UserGroupID) {
	for _, readOnlyGroup := range c.settings.GroupMapping.ReadOnly {
		if readOnlyGroup == groupCN {
			*groups = append(*groups, authdata.UserGroupIDReadOnly)
			break
		}
	}
	for _, adminGroup := range c.settings.GroupMapping.Admin {
		if adminGroup == groupCN {
			*groups = append(*groups, authdata.UserGroupIDAdmin)
			break
		}
	}
	for _, superAdminGroup := range c.settings.GroupMapping.SuperAdmin {
		if superAdminGroup == groupCN {
			*groups = append(*groups, authdata.UserGroupIDSuperAdmin)
			break
		}
	}
}
