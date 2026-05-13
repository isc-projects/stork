package oidc

import (
	storkutil "isc.org/stork/util"
)

// Specifies a list of comma-separated flags.
type CommaSeparatedStrings []string

// Specifies mapping between OIDC groups returned from token endpoint and Stork groups.
type GroupMapping struct {
	Admin      CommaSeparatedStrings `long:"oidc-group-admin" description:"The group returned from OIDC token endpoint that can be mapped to Stork 'admin' group; also accepts a comma-separated list of group names" env:"STORK_OIDC_GROUP_ADMIN" default:"stork-admin"`
	SuperAdmin CommaSeparatedStrings `long:"oidc-group-super-admin" description:"The group returned from OIDC token endpoint that can be mapped to Stork 'super-admin' group; also accepts a comma-separated list of group names" env:"STORK_OIDC_GROUP_SUPER_ADMIN" default:"stork-super-admin"`
	ReadOnly   CommaSeparatedStrings `long:"oidc-group-read-only" description:"The group returned from OIDC token endpoint that can be mapped to Stork 'read-only' group; also accepts a comma-separated list of group names" env:"STORK_OIDC_GROUP_READ_ONLY" default:"stork-read-only"`
}

// The main OIDC settings structure.
type Settings struct {
	IssuerURL            string                `long:"oidc-issuer-url" description:"The OID Provider Issuer URL used for OIDC discovery process." env:"STORK_OIDC_ISSUER_URL" default:""`
	ClientID             string                `long:"oidc-client-id" description:"Mandatory. Client ID registered at the OID Provider." env:"STORK_OIDC_CLIENT_ID" default:""`
	ClientSecret         string                `long:"oidc-client-secret" description:"Optional. Client secret provided by the OID Provider. Optional, because only some Providers require this in the OIDC process." env:"STORK_OIDC_CLIENT_SECRET" default:""`
	IdentityProviderName string                `long:"oidc-provider-name" description:"Optional. The OID Provider name that will be displayed on a Login page. Leave blank to display generic OAuth/OIDC name." env:"STORK_OIDC_PROVIDER_NAME" default:""`
	MandatoryAllowGroup  string                `long:"oidc-group-allow" description:"The mandatory group that user must belong to, to access Stork, empty for allow all users" env:"STORK_OIDC_GROUP_ALLOW" default:""`
	EnableGroupMapping   bool                  `long:"oidc-map-groups" description:"Enable mapping OIDC groups returned from token endpoint to Stork groups" env:"STORK_OIDC_MAP_GROUPS"`
	GroupMapping         GroupMapping          `group:"OIDC to Stork group mapping"`
	Scopes               CommaSeparatedStrings `long:"oidc-scopes" description:"Comma separated list of scopes sent in Authentication Request. Stork always sends 'openid' scope and this list is appended." env:"STORK_OIDC_SCOPES" default:"email,profile"`
	GroupsClaim          string                `long:"oidc-groups-claim" description:"Claim key used to retrieve user groups from OID Provider token endpoint. It must be configured if 'group-allow' or 'map-groups' setting is used" env:"STORK_OIDC_GROUPS_CLAIM" default:"groups"`
}

// Implementation of go-flags Unmarshaler interface. Unmarshals a comma-separated values in a string into a slice of strings.
// Supports backslash comma escaping.
func (flags *CommaSeparatedStrings) UnmarshalFlag(value string) error {
	*flags = storkutil.SplitByComma(value)
	return nil
}
