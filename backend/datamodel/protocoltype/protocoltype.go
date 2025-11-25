package protocoltype

// Supported protocol types in communication between the Stork agent and
// daemons.
type ProtocolType string

const (
	Unspecified ProtocolType = ""
	HTTP        ProtocolType = "http"
	HTTPS       ProtocolType = "https"
	Socket      ProtocolType = "unix"
	RNDC        ProtocolType = "rndc"
)

// Indicates whether the protocol type is secure.
func (pt ProtocolType) IsSecure() bool {
	return pt == HTTPS || pt == RNDC
}

// Parses the protocol type from string. It returns false if the
// protocol type is not recognized.
func Parse(protocolType string) (ProtocolType, bool) {
	switch protocolType {
	case string(HTTP):
		return HTTP, true
	case string(HTTPS):
		return HTTPS, true
	case string(Socket):
		return Socket, true
	case string(RNDC):
		return RNDC, true
	default:
		return Unspecified, false
	}
}
