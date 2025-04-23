package bind9config

import storkutil "isc.org/stork/util"

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
			AdressMatchList: &AddressMatchList{
				Elements: []*AddressMatchListElement{
					{
						IPAddress: "127.0.0.1",
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
func (l ListenOnClauses) GetMatchingListenOn(port int64) *ListenOn {
	// For default port and no listen-on clauses, return the default
	// listen-on clause.
	if len(l) == 0 && port == 53 {
		return (*GetDefaultListenOnClauses())[0]
	}
	// Check listen-on clauses that include 127.0.0.1 or 0.0.0.0.
	for _, listenOn := range l {
		if listenOn.GetPort() == port && (listenOn.IncludesIPAddress("127.0.0.1") || listenOn.IncludesIPAddress("0.0.0.0")) {
			return listenOn
		}
	}
	// Check listen-on-v6 clauses that include ::1 or ::.
	for _, listenOn := range l {
		if listenOn.GetPort() == port && (listenOn.IncludesIPAddress("::1") || listenOn.IncludesIPAddress("::")) {
			return listenOn
		}
	}
	// Check listen-on clauses that include the specified port.
	for _, listenOn := range l {
		if listenOn.GetPort() == port {
			return listenOn
		}
	}
	// No match.
	return nil
}

// Gets the preferred IP address from the listen-on clause.
// The function prefers loopback and zero addresses.
func (l *ListenOn) GetPreferredIPAddress(allowTransferMatchList *AddressMatchList) string {
	if (l.IncludesIPAddress("127.0.0.1") || l.IncludesIPAddress("0.0.0.0")) && !allowTransferMatchList.ExcludesIPAddress("127.0.0.1") {
		return "127.0.0.1"
	}
	if (l.IncludesIPAddress("::1") || l.IncludesIPAddress("::")) && !allowTransferMatchList.ExcludesIPAddress("::1") {
		return "::1"
	}
	for _, element := range l.AdressMatchList.Elements {
		if element.IPAddress != "" && !element.Negation && !allowTransferMatchList.ExcludesIPAddress(element.IPAddress) {
			return element.IPAddress
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

// Checks if the listen-on clause includes the specified IP address.
func (l *ListenOn) IncludesIPAddress(ipAddress string) bool {
	for _, element := range l.AdressMatchList.Elements {
		if element.IPAddress == ipAddress && !element.Negation {
			return true
		}
	}
	return false
}
