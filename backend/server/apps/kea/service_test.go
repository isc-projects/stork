package kea

import (
	"encoding/json"
	"testing"

	require "github.com/stretchr/testify/require"
	dbmodel "isc.org/stork/server/database/model"
)

func getTestConfigDHCP4() *dbmodel.KeaConfig {
	configStr := `{
        "Dhcp4": {
            "hooks-libraries": [
                {
                    "library": "libdhcp_ha.so",
                    "parameters": {
                        "high-availability": [{
                            "this-server-name": "server1",
                            "mode": "load-balancing",
                            "heartbeat-delay": 10000,
                            "max-response-delay": 10000,
                            "max-ack-delay": 5000,
                            "max-unacked-clients": 5,
                            "peers": [{
                                "name": "server1",
                                "url": "http://192.168.56.33:8000/",
                                "role": "primary",
                                "auto-failover": true
                            }, {
                                "name": "server2",
                                "url": "http://192.168.56.66:8000/",
                                "role": "secondary",
                                "auto-failover": true
                            }, {
                                "name": "server3",
                                "url": "http://192.168.56.99:8000/",
                                "role": "backup",
                                "auto-failover": false
                            }]
                        }]
                    }
                }
            ]
        }
    }`
	var config dbmodel.KeaConfig
	_ = json.Unmarshal([]byte(configStr), &config)
	return &config
}

func TestDetectHAServices(t *testing.T) {
	dbApp := dbmodel.App{
		Details: dbmodel.AppKea{
			Daemons: []*dbmodel.KeaDaemon{
				{
					Name:   "dhcp4",
					Config: getTestConfigDHCP4(),
				},
			},
		},
	}

	services := DetectHAServices(&dbApp)
	require.Len(t, services, 1)
}
