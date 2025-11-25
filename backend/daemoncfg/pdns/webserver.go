package pdnsconfig

import "net"

// Get webserver configuration from the configuration file. It returns the webserver
// address and port when api and webserver are enabled in the configuration file.
// Otherwise, it returns nil values. The default webserver and port are 127.0.0.1:8081.
func (c *Config) GetWebserverConfig() (*string, *int64, bool) {
	api := c.GetBool("api")
	if api == nil || !*api {
		return nil, nil, false
	}
	webserver := c.GetBool("webserver")
	if webserver == nil || !*webserver {
		return nil, nil, false
	}
	// Default address and port in PowerDNS.
	address := "127.0.0.1"
	port := int64(8081)
	if webserverAddress := c.GetString("webserver-address"); webserverAddress != nil {
		if ip := net.ParseIP(*webserverAddress); ip != nil {
			if ip.IsUnspecified() {
				if ip.To4() == nil {
					address = "::1"
				}
			} else {
				address = *webserverAddress
			}
		}
	}
	if webserverPort := c.GetInt64("webserver-port"); webserverPort != nil {
		port = *webserverPort
	}
	return &address, &port, true
}
