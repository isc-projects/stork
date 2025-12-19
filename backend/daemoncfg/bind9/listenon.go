package bind9config

import storkutil "isc.org/stork/util"

var _ formattedElement = (*ListenOn)(nil)

// ListenOn is the clause specifying the addresses the servers listens on the
// DNS requests. It also contains additional options.
//
// The listen-on clause has the following format:
//
//	listen-on [ port <integer> ] [ proxy <string> ] [ tls <string> ] [ http <string> ] { <address_match_element>; ... };
//
// See: https://bind9.readthedocs.io/en/latest/reference.html#namedconf-statement-listen-on
type ListenOn struct {
	Variant          string            `parser:"@( 'listen-on' | 'listen-on-v6' )"`
	Port             *int64            `parser:"( 'port' @Ident )?"`
	Proxy            *string           `parser:"( 'proxy' ( @String | @Ident ) )?"`
	TLS              *string           `parser:"( 'tls' ( @String | @Ident ) )?"`
	HTTP             *string           `parser:"( 'http' ( @String | @Ident ) )?"`
	AddressMatchList *AddressMatchList `parser:"'{' @@ '}'"`
}

// Defines a collection of listen-on and listen-on-v6 clauses.
// These clauses can be specified multiple times in the configuration file.
// This object is used to extract best matching listen-on clauses from the
// collection.
type ListenOnClauses []*ListenOn

// Gets a default listen-on clause encapsulated in a slice. The default
// clause includes the address of 127.0.0.1 and port 53.
func GetDefaultListenOnClauses() *ListenOnClauses {
	return &ListenOnClauses{
		&ListenOn{
			AddressMatchList: &AddressMatchList{
				Elements: []*AddressMatchListElement{
					{
						IPAddressOrACLName: "127.0.0.1",
					},
				},
			},
			Port: storkutil.Ptr(int64(53)),
		},
	}
}

// Attempts to find a listen-on clause that matches the specified port,
// typically extracted from the allow-transfer clause. This search
// prefers listen-on clauses enabling listening on local loopback
// addresses.
func (l ListenOnClauses) GetMatchingListenOnClause(port int64) *ListenOn {
	// For default port and no listen-on clauses, return the default
	// listen-on clause.
	if len(l) == 0 && port == 53 {
		return (*GetDefaultListenOnClauses())[0]
	}
	// Check listen-on clauses that include 127.0.0.1 or 0.0.0.0.
	for _, listenOn := range l {
		if listenOn.GetPort() == port && !listenOn.Includes("none") && (listenOn.Includes("127.0.0.1") || listenOn.Includes("0.0.0.0")) {
			return listenOn
		}
	}
	// Check listen-on-v6 clauses that include ::1 or ::.
	for _, listenOn := range l {
		if listenOn.GetPort() == port && !listenOn.Includes("none") && (listenOn.Includes("::1") || listenOn.Includes("::")) {
			return listenOn
		}
	}
	// Check listen-on clauses that include any.
	for _, listenOn := range l {
		if listenOn.GetPort() == port && !listenOn.Includes("none") && listenOn.Includes("any") {
			return listenOn
		}
	}
	// Check listen-on clauses that include the specified port.
	for _, listenOn := range l {
		if listenOn.GetPort() == port && !listenOn.Includes("none") {
			return listenOn
		}
	}
	// No match.
	return nil
}

// Gets the preferred IP address from the listen-on clause.
// The function prefers loopback and zero addresses.
func (l *ListenOn) GetPreferredIPAddress(allowTransferMatchList *AddressMatchList) string {
	switch l.Variant {
	case "listen-on":
		if (l.Includes("127.0.0.1") || l.Includes("0.0.0.0")) || l.Includes("any") && !allowTransferMatchList.Excludes("127.0.0.1") {
			return "127.0.0.1"
		}
	case "listen-on-v6":
		if (l.Includes("::1") || l.Includes("::")) || l.Includes("any") && !allowTransferMatchList.Excludes("::1") {
			return "::1"
		}
	}
	for _, element := range l.AddressMatchList.Elements {
		if element.IPAddressOrACLName != "" && storkutil.IsIPAddress(element.IPAddressOrACLName) && !element.Negation && !allowTransferMatchList.Excludes(element.IPAddressOrACLName) {
			return element.IPAddressOrACLName
		}
	}
	return ""
}

// Gets the port from the listen-on clause. If the port is not specified,
// the default port 53 is returned.
func (l *ListenOn) GetPort() int64 {
	if l.Port != nil {
		return *l.Port
	}
	return 53
}

// Checks if the listen-on clause includes the specified IP address or ACL name.
func (l *ListenOn) Includes(ipAddressOrACLName string) bool {
	for _, element := range l.AddressMatchList.Elements {
		if element.IPAddressOrACLName == ipAddressOrACLName && !element.Negation {
			return true
		}
	}
	return false
}

// Returns the serialized BIND 9 configuration for the listen-on and listen-on-v6 clauses.
func (l *ListenOn) getFormattedOutput(filter *Filter) formatterOutput {
	clause := newFormatterClause(l.Variant)
	if l.Port != nil {
		clause.addTokenf(`port %d`, l.GetPort())
	}
	if l.Proxy != nil {
		clause.addTokenf(`proxy %s`, *l.Proxy)
	}
	if l.TLS != nil {
		clause.addTokenf(`tls %s`, *l.TLS)
	}
	if l.HTTP != nil {
		clause.addTokenf(`http %s`, *l.HTTP)
	}
	clauseScope := clause.addScope()
	if l.AddressMatchList != nil {
		for _, element := range l.AddressMatchList.Elements {
			clauseScope.add(element.getFormattedOutput(filter))
		}
	}
	return clause
}
