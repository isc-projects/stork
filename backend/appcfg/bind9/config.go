package bind9config

import (
	"net"
	"path/filepath"
	"strconv"

	"github.com/pkg/errors"
)

const DefaultViewName = "_default"

func (c *Config) GetOptions() *Options {
	for _, statement := range c.Statements {
		if statement.Options != nil {
			return statement.Options
		}
	}
	return nil
}

// Returns the view with the given name or nil if the view is not found.
func (c *Config) GetView(viewName string) *View {
	for _, statement := range c.Statements {
		if statement.View != nil && statement.View.Name == viewName {
			return statement.View
		}
	}
	return nil
}

func (c *Config) GetZone(zoneName string) *Zone {
	for _, statement := range c.Statements {
		if statement.Zone != nil && statement.Zone.Name == zoneName {
			return statement.Zone
		}
	}
	return nil
}

// Returns the key with the given name or nil if the key is not found.
func (c *Config) GetKey(keyID string) *Key {
	for _, statement := range c.Statements {
		if statement.Key != nil && statement.Key.Name == keyID {
			return statement.Key
		}
	}
	return nil
}

// Returns the ACL with the given name or nil if the ACL is not found.
func (c *Config) GetACL(aclName string) *ACL {
	for _, statement := range c.Statements {
		if statement.ACL != nil && statement.ACL.Name == aclName {
			return statement.ACL
		}
	}
	return nil
}

// Recursively searches for a key in the address-match-list. If the list
// contains references to other ACLs, it searches for a key in the referenced
// ACLs. It protects against infinite recursion by limiting the depth of the
// search to 5 levels.
func (c *Config) getKeyFromAddressMatchList(level int, addressMatchList *AddressMatchList) (*Key, error) {
	if level > 5 {
		// Too much recursion.
		return nil, errors.New("too much recursion in address-match-list")
	}
	for _, element := range addressMatchList.Elements {
		switch {
		case element.Negation:
			// Skip the element that contains negation (!).
			continue
		case element.KeyID != "":
			// Find a key by specified name.
			return c.GetKey(element.KeyID), nil
		case element.ACL != nil:
			// Recursively search for a key in the inline ACL.
			return c.getKeyFromAddressMatchList(level+1, element.ACL.AddressMatchList)
		case element.ACLName != "":
			// Recursively search for a key in the referenced ACL.
			acl := c.GetACL(element.ACLName)
			if acl != nil {
				return c.getKeyFromAddressMatchList(level+1, acl.AddressMatchList)
			}
		default:
			continue
		}
	}
	// The key was not found.
	return nil, nil
}

// Gets credentials for the zone transfer for the given zone in the default view.
func (c *Config) getAXFRCredentialsForDefaultView(zoneName string) (address *string, keyName *string, algorithm *string, secret *string, err error) {
	// The allow-transfer enables zone transfers and can be specified at the
	// zone or global level.
	var allowTransfer *AllowTransfer

	// Get the zone and check if it contains allow-transfer clause. The zone-level
	// allow-transfer overrides the global one.
	zone := c.GetZone(zoneName)
	if zone != nil {
		allowTransfer = zone.GetAllowTransfer()
	}

	var options *Options
	if allowTransfer == nil {
		// If allow-transfer was not specified at the zone level, check if it is
		// specified at the global level.
		if options = c.GetOptions(); options != nil {
			allowTransfer = options.GetAllowTransfer()
		}
	}

	// If allow-transfer is disabled (at zone level or globally) we cannot do
	// AXFR. Return an error as it requires administrative action.
	if allowTransfer == nil || allowTransfer.IsDisabled() {
		return nil, nil, nil, nil, errors.Errorf("failed to get AXFR credentials for zone %s: allow-transfer is disabled", zoneName)
	}

	// The allow-transfer may specify the key that is allowed to run the zone transfer.
	// If that key is specified the client will use it.
	var key *Key
	if key, err = c.getKeyFromAddressMatchList(0, allowTransfer.AddressMatchList); err != nil {
		return nil, nil, nil, nil, errors.WithMessagef(err, "failed to get AXFR credentials for zone %s", zoneName)
	}

	if key != nil {
		// Key is optional when the zone is in the default view. If it is specified, let's get
		// the key details.
		keyName = &key.Name
		if algorithm, secret, err = key.GetAlgorithmSecret(); err != nil {
			return nil, nil, nil, nil, errors.WithMessagef(err, "failed to get AXFR credentials for zone %s", zoneName)
		}
	}

	// The allow-transfer clause may optionally specify the port number. The client should send
	// the request to this port number. The default port is 53.
	port := int64(53)
	if allowTransfer.Port != nil {
		port = *allowTransfer.Port
	}

	// We may be getting the options for the first time.
	if options == nil {
		options = c.GetOptions()
	}

	// The listen-on settings are optional. If they are not specified, named will try
	// to listen on all interfaces, including the loopback. Hence, let's start with the
	// default listen-on settings: 127.0.0.1:53.
	listenOnSet := GetDefaultListenOnClauses()
	if options != nil {
		listenOnSet = options.GetListenOnSet()
	}

	// The listenOnSet may contain multiple listen-on clauses. We need to find the
	// clause matching our desired port. Also, preferably it should be a local
	// loopback address.
	listenOn := listenOnSet.GetMatchingListenOn(port)
	if listenOn == nil {
		return nil, nil, nil, nil, errors.Errorf("failed to get AXFR credentials for zone %s: allow-transfer port %d does not match any listen-on setting", zoneName, port)
	}

	// Return the address and port to connect to.
	preferredIPAddress := listenOn.GetPreferredIPAddress(allowTransfer.AddressMatchList)
	if preferredIPAddress == "" {
		return nil, nil, nil, nil, errors.Errorf("failed to get AXFR credentials for zone %s: allow-transfer port %d does not match any listen-on setting", zoneName, port)
	}
	addr := net.JoinHostPort(preferredIPAddress, strconv.Itoa(int(listenOn.GetPort())))
	return &addr, keyName, algorithm, secret, nil
}

// Gets credentials for the zone transfer for the given view and zone.
func (c *Config) getAXFRCredentialsForView(viewName string, zoneName string) (address *string, keyName *string, algorithm *string, secret *string, err error) {
	// View is required as it may contain the match-clients clause. This clause should
	// contain a reference to the key the client should use to discriminate between the
	// zones from different views.
	view := c.GetView(viewName)
	if view == nil {
		return nil, nil, nil, nil, errors.Errorf("failed to get AXFR credentials for view %s, zone %s: view does not exist", viewName, zoneName)
	}
	matchClients := view.GetMatchClients()

	// The allow-transfer enables zone transfers and can be specified at the
	// zone, view or global level.
	var allowTransfer *AllowTransfer

	// Try to get the zone. The zone is optional because it may be specified outside of the
	// configuration file. If it is specified, it may contain the allow-transfer clause.
	zone := view.GetZone(zoneName)
	if zone != nil {
		allowTransfer = zone.GetAllowTransfer()
	}

	// If allow-transfer was not specified at the zone level, check if it is
	// specified at the view level.
	if allowTransfer == nil {
		allowTransfer = view.GetAllowTransfer()
	}

	var options *Options
	if allowTransfer == nil {
		// If allow-transfer was specified neither at the zone level nor view level,
		// check if it is specified at the global level.
		if options = c.GetOptions(); options != nil {
			allowTransfer = options.GetAllowTransfer()
		}
	}

	// If allow-transfer is disabled (at zone, view level or globally) we cannot do
	// AXFR. Return an error as it requires administrative action.
	if allowTransfer == nil || allowTransfer.IsDisabled() {
		return nil, nil, nil, nil, errors.Errorf("failed to get AXFR credentials for view %s, zone %s: allow-transfer is disabled", viewName, zoneName)
	}

	// The allow-transfer may specify the key that is allowed to run the zone transfer.
	// If that key is specified the client will use it.

	// Check if the match-clients clause contains a reference to the key.
	var key *Key
	if matchClients != nil {
		key, err = c.getKeyFromAddressMatchList(0, matchClients.AddressMatchList)
		if err != nil {
			return nil, nil, nil, nil, err
		}
	}

	if key == nil {
		// Allow transfer may specify the key that is allowed to run the zone transfer.
		// If this key is specified the client will use it.
		key, err = c.getKeyFromAddressMatchList(0, allowTransfer.AddressMatchList)
		if err != nil {
			return nil, nil, nil, nil, err
		}
	}

	// The key is required when dealing with views. Otherwise, it is not possible to
	// discriminate between the zones from different views.
	if key == nil {
		return nil, nil, nil, nil, errors.Errorf("failed to get AXFR credentials for view %s, zone %s: no key found", viewName, zoneName)
	}

	// Get the key details.
	keyName = &key.Name
	algorithm, secret, err = key.GetAlgorithmSecret()
	if err != nil {
		return nil, nil, nil, nil, errors.WithMessagef(err, "failed to get AXFR credentials for zone %s", zoneName)
	}

	// The allow-transfer clause may optionally specify the port number. The client should send
	// the request to this port number. The default port is 53.
	port := int64(53)
	if allowTransfer.Port != nil {
		port = *allowTransfer.Port
	}

	// We may be getting the options for the first time.
	if options == nil {
		options = c.GetOptions()
	}

	// The listen-on settings are optional. If they are not specified, named will try
	// to listen on all interfaces, including the loopback. Hence, let's start with the
	// default listen-on settings: 127.0.0.1:53.
	listenOnSet := GetDefaultListenOnClauses()
	if options != nil {
		listenOnSet = options.GetListenOnSet()
	}

	// The listenOnSet may contain multiple listen-on clauses. We need to find the
	// clause matching our desired port. Also, preferably it should be a local
	// loopback address.
	listenOn := listenOnSet.GetMatchingListenOn(port)
	if listenOn == nil {
		return nil, nil, nil, nil, errors.Errorf("failed to get AXFR credentials for zone %s: allow-transfer port %d does not match any listen-on setting", zoneName, port)
	}

	// Return the address and port to connect to.
	preferredIPAddress := listenOn.GetPreferredIPAddress(allowTransfer.AddressMatchList)
	if preferredIPAddress == "" {
		return nil, nil, nil, nil, errors.Errorf("failed to get AXFR credentials for zone %s: allow-transfer port %d does not match any listen-on setting", zoneName, port)
	}
	addr := net.JoinHostPort(preferredIPAddress, strconv.Itoa(int(listenOn.GetPort())))
	return &addr, keyName, algorithm, secret, nil
}

// Gets the target address and the required credentials for the zone transfer.
// A caller specifies the view name and zone name. If the default view is specified,
// the zone is expected to be defined outside of the view. Otherwise, the zone should
// be defined inside of the specified view. This function also allows that the zone
// is not specified in the configuration file. That's because the zone may be created
// dynamically. The specified view, however, must be present in the configuration file.
//
// If non-default view is specified, the function expects that there is a TSIG key
// associated with the view. This association can be realized via match-clients clause
// at the view level or via allow-transfer at the view or zone level. If the key is not
// found for the view, the function will return an error. The address and port of the
// server is determined using listen-on and/or listen-on-v6 clauses. The local loopback
// addresses are preferred. If these settings are not explicitly specified, the local
// loopback address with default port 53 is assumed.
//
// If the default view is specified, the logic is slightly different. Firstly, the zone
// is specified at the global scope, so no view-level settings are taken into account.
// Secondly, the TSIG key is not mandatory. If it is not found, the nil values are returned
// for the respective key-specific arguments. The allow-transfer can be specified at the
// zone level or globally.
//
// It has to be noted that this function is trying its best to find suitable credentials
// but due to the complexity of the BIND 9 configuration file, some of the corner cases
// may not be handled. It typically involves the cross check between the ACLs and the
// listen-on clauses. Especially when they refer to ACLs. As a result, some AXFR attempts
// may fail in BIND 9 when misconfigurations aren't caught by Stork.
func (c *Config) GetAXFRCredentials(viewName string, zoneName string) (address *string, keyName *string, algorithm *string, secret *string, err error) {
	if viewName != DefaultViewName {
		return c.getAXFRCredentialsForView(viewName, zoneName)
	}
	return c.getAXFRCredentialsForDefaultView(zoneName)
}

// Returns the API key for the statistics channel. This key is included in
// the X-API-Key header. It is unused for BIND 9.
func (c *Config) GetAPIKey() string {
	return ""
}

// Checks if the zone is RPZ.
func (c *Config) IsRPZ(viewName string, zoneName string) bool {
	var responsePolicy *ResponsePolicy
	if viewName == DefaultViewName {
		if options := c.GetOptions(); options != nil {
			responsePolicy = options.GetResponsePolicy()
		}
	} else if view := c.GetView(viewName); view != nil {
		responsePolicy = view.GetResponsePolicy()
	}
	return responsePolicy != nil && responsePolicy.IsRPZ(zoneName)
}

// Returns the key associated with the given view or nil if the view is not found.
// The key can be associated with the view via match-clients clause and the global
// ACLs.
func (c *Config) GetZoneKey(viewName string, zoneName string) (*Key, error) {
	view := c.GetView(viewName)
	if view == nil {
		return nil, nil
	}
	// Check if there is match-clients clause. Use it if present.
	for _, clause := range view.Clauses {
		if clause.MatchClients != nil {
			return c.getKeyFromAddressMatchList(0, clause.MatchClients.AddressMatchList)
		}
	}
	// If there was no match-clients clause, look for the allow-transfer clause
	zone := view.GetZone(zoneName)
	if zone == nil {
		return nil, nil
	}
	// If there was no match-clients clause, check if there is allow-transfer
	// clause. Use it if present.
	for _, clause := range view.Clauses {
		if clause.AllowTransfer != nil {
			return c.getKeyFromAddressMatchList(0, clause.AllowTransfer.AddressMatchList)
		}
	}
	return nil, nil
}

// Returns the algorithm and secret from the given key.
func (key *Key) GetAlgorithmSecret() (algorithm *string, secret *string, err error) {
	for _, clause := range key.Clauses {
		if clause.Algorithm != "" {
			algorithm = &clause.Algorithm
		}
		if clause.Secret != "" {
			secret = &clause.Secret
		}
	}
	if algorithm == nil || secret == nil {
		err = errors.Errorf("no algorithm or secret found in key %s", key.Name)
	}
	return
}

// Expands the configuration by including the contents of the included files.
// The baseDir is a path prepended to the path of the included files when their
// paths are relative.
func (c *Config) Expand(baseDir string) (*Config, error) {
	expanded := &Config{
		sourcePath: c.sourcePath,
	}
	// Go over the top-level statements and identify the include statements.
	for _, statement := range c.Statements {
		if statement.Include != nil {
			// Found an include statement.
			path := statement.Include.Path
			if !filepath.IsAbs(path) {
				// Use the absolute path to the config file.
				path = filepath.Join(baseDir, path)
			}
			// Clean the path so it may be compared with the source file path to
			// avoid the cycles.
			path = filepath.Clean(path)
			if path == c.sourcePath {
				// If the included file points to the including file, skip expanding it.
				// One could consider returning an error but we want the parser to be
				// liberal. Stork wants to be able to look into the file contents rather
				// than validate it.
				expanded.Statements = append(expanded.Statements, statement)
				continue
			}
			// Parse the included file.
			parsedInclude, err := NewParser().ParseFile(path)
			if err != nil {
				return nil, err
			}
			// Append the parsed statements to the parent file.
			expanded.Statements = append(expanded.Statements, parsedInclude.Statements...)
		} else {
			// This is not an include statement. Append it as is.
			expanded.Statements = append(expanded.Statements, statement)
		}
	}
	return expanded, nil
}
