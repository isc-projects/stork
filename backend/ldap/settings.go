package ldap

import (
	"time"

	storkutil "isc.org/stork/util"
)

// LDAP has no strict user structure, and it can use various schemas. The
// values define the property names used by the current LDAP schema.
type LDAPAttributeNames struct {
	ObjectClassGroup string `long:"object-class-group" description:"The name of the group object class in the group schema (preferred 'groupOfNames' or 'groupOfUniqueNames')" env:"OBJECT_CLASS_GROUP" default:"groupOfNames"`
	GroupMember      string `long:"object-class-group-member" description:"The name of the member property in the group object class (for 'groupOfUniqueNames' use 'uniqueMember')" env:"OBJECT_CLASS_GROUP_MEMBER" default:"member"`
	GroupCommonName  string `long:"object-class-group-common-name" description:"The name of the common name property in the group object class" env:"OBJECT_CLASS_GROUP_COMMON_NAME" default:"cn"`
	ObjectClassUser  string `long:"object-class-user" description:"The name of the user object class in the user schema" env:"OBJECT_CLASS_USER" default:"organizationalPerson"`
	UserID           string `long:"object-class-user-id" description:"The name of the ID property in the user object class" env:"OBJECT_CLASS_USER_ID" default:"uid"`
	FirstName        string `long:"object-class-user-first-name" description:"The name of the first name property in the user object class, optional" env:"OBJECT_CLASS_USER_FIRST_NAME" default:"givenName"`
	LastName         string `long:"object-class-user-last-name" description:"The name of the last name property in the user object class, optional" env:"OBJECT_CLASS_USER_LAST_NAME" default:"sn"`
	Email            string `long:"object-class-user-email" description:"The name of the email property in the user object class, optional" env:"OBJECT_CLASS_USER_EMAIL" default:"mail"`
	UniqueIdentifier string `long:"object-class-user-unique-identifier" description:"The name of the unique, persistent identifier property in the user object class, optional" env:"OBJECT_CLASS_USER_UNIQUE_IDENTIFIER" default:"entryUUID"`
}

// Specifies a list of comma-separated flags.
type CommaSeparatedStrings []string

// Specifies mapping between LDAP and Stork groups.
// The values are the common names of the LDAP groups.
type GroupMapping struct {
	Admin      CommaSeparatedStrings `long:"group-admin" description:"The LDAP group corresponding to Stork 'admin' group; also accepts a comma-separated list of group names" env:"GROUP_ADMIN" default:"stork-admin"`
	SuperAdmin CommaSeparatedStrings `long:"group-super-admin" description:"The LDAP group corresponding to Stork 'super-admin' group; also accepts a comma-separated list of group names" env:"GROUP_SUPER_ADMIN" default:"stork-super-admin"`
	ReadOnly   CommaSeparatedStrings `long:"group-read-only" description:"The LDAP group corresponding to Stork 'read-only' group; also accepts a comma-separated list of group names" env:"GROUP_READ_ONLY" default:"stork-read-only"`
}

// The main settings structure.
type Settings struct {
	DialURL                   string             `long:"url" description:"The LDAP server access URL (use ldaps:// protocol to connect over TLS)" env:"URL" default:"ldap://127.0.0.1:1389"`
	Root                      string             `long:"root" description:"The LDAP root for login user" env:"ROOT" default:"dc=example,dc=org"`
	BindUserDN                string             `long:"bind-userdn" description:"The maintenance userdn used to bind to the server for reading user profiles" env:"BIND_USERDN" default:"cn=admin,dc=example,dc=org"`
	BindPassword              string             `long:"bind-password" description:"The maintenance password used to bind to the server for reading user profiles" env:"BIND_PASSWORD" default:"adminpassword"`
	TLSSkipServerVerification bool               `long:"skip-tls-server-verification" description:"Skip the TLS server certificate verification - not recommended for the production environments" env:"SKIP_SERVER_TLS_VERIFICATION"`
	MandatoryAllowGroup       string             `long:"group-allow" description:"The mandatory group that must be assigned to user to access Stork, empty for allow all users" env:"GROUP_ALLOW" default:""`
	EnableGroupMapping        bool               `long:"map-groups" description:"Enable mapping LDAP groups into Stork groups" env:"MAP_GROUPS"`
	GroupMapping              GroupMapping       `group:"LDAP to Stork group mapping"`
	AttributeNames            LDAPAttributeNames `group:"LDAP schema attributes"`
	Debug                     bool               `long:"debug" description:"Enable additional debug information about connection to LDAP server" env:"DEBUG"`
	Timeout                   time.Duration      `long:"timeout" description:"The LDAP server connection timeout" env:"TIMEOUT" default:"30s"`
}

// Implementation of go-flags Unmarshaler interface. Unmarshals a comma-separated values in a string into a slice of strings.
// Supports backslash comma escaping.
func (flags *CommaSeparatedStrings) UnmarshalFlag(value string) error {
	*flags = append(*flags, storkutil.SplitByComma(value)...)
	return nil
}
