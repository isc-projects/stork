package bind9config

import (
	"strconv"
)

const (
	defaultControlsPort           int64 = 953
	defaultStatisticsChannelsPort int64 = 80
)

// Returns an address and port which the agent can connect to based on the
// inet clause. If port is an asterisk, or not specified, the default port
// specified as an argument is used. If the address is an asterisk or is zero,
// the address is set to "localhost".
func (i *InetClause) GetConnectableAddressAndPort(defaultPort int64) (address string, port int64) {
	address = i.Address
	port = defaultPort
	if i.Port != nil && *i.Port != "*" {
		if parsedPort, err := strconv.Atoi(*i.Port); err == nil {
			port = int64(parsedPort)
		}
	}
	if address == "*" || address == "0.0.0.0" || address == "::" {
		address = "localhost"
	}
	return
}
