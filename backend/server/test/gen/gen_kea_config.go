// Config generator for test purposes
package storktestgen

import (
	"fmt"
	"math/rand"
)

// Generate Kea configuration with specific number of subnets.
// It is a port of the "main" function from "gen_kea_config.py" file.
func GenerateKeaV4Config(n int) map[string]interface{} {
	inner := 0
	outer := 0
	if n/256 > 0 {
		inner = 255
		outer = n / 256
	} else {
		inner = 0
		outer = n
	}

	config := map[string]interface{}{
		"Dhcp4": map[string]interface{}{
			"interfaces-config": map[string]interface{}{
				"interfaces": []interface{}{"eth0"},
			},
			"control-socket": map[string]interface{}{
				"socket-type": "unix",
				"socket-name": "/tmp/kea4-ctrl-socket",
			},
			"lease-database": map[string]interface{}{
				"type":         "memfile",
				"lfc-interval": 3600,
			},
			"expired-leases-processing": map[string]interface{}{
				"reclaim-timer-wait-time":         10,
				"flush-reclaimed-timer-wait-time": 25,
				"hold-reclaimed-time":             3600,
				"max-reclaim-leases":              100,
				"max-reclaim-time":                250,
				"unwarned-reclaim-cycles":         5,
			},

			"renew-timer":    90,
			"rebind-timer":   120,
			"valid-lifetime": 180,

			"reservations": []interface{}{
				map[string]interface{}{
					"hw-address": "ee:ee:ee:ee:ee:ee",
					"ip-address": "10.0.0.123",
				},
				map[string]interface{}{
					"client-id":  "aa:aa:aa:aa:aa:aa",
					"ip-address": "10.0.0.222",
				},
			},

			"option-data": []interface{}{
				map[string]interface{}{
					"name": "domain-name-servers",
					"data": "192.0.2.1, 192.0.2.2",
				},
				map[string]interface{}{
					"code": 15,
					"data": "example.org",
				},
				map[string]interface{}{
					"name": "domain-search",
					"data": "mydomain.example.com, example.com",
				},
				map[string]interface{}{
					"name": "boot-file-name",
					"data": "EST5EDT4\\,M3.2.0/02:00\\,M11.1.0/02:00",
				},
				map[string]interface{}{
					"name": "default-ip-ttl",
					"data": "0xf0",
				},
			},
			"client-classes": []interface{}{
				map[string]interface{}{
					"name": "class-00-00",
					"test": "substring(hexstring(pkt4.mac,':'),0,5) == '00:00'",
				},
				map[string]interface{}{
					"name": "class-01-00",
					"test": "substring(hexstring(pkt4.mac,':'),0,5) == '01:00'",
				},
				map[string]interface{}{
					"name": "class-01-01",
					"test": "substring(hexstring(pkt4.mac,':'),0,5) == '01:01'",
				},
				map[string]interface{}{
					"name": "class-01-02",
					"test": "substring(hexstring(pkt4.mac,':'),0,5) == '01:02'",
				},
				map[string]interface{}{
					"name": "class-01-03",
					"test": "substring(hexstring(pkt4.mac,':'),0,5) == '01:03'",
				},
				map[string]interface{}{
					"name": "class-01-04",
					"test": "substring(hexstring(pkt4.mac,':'),0,5) == '01:04'",
				},
				map[string]interface{}{
					"name": "class-02-00",
					"test": "substring(hexstring(pkt4.mac,':'),0,5) == '02:00'",
				},
				map[string]interface{}{
					"name": "class-02-01",
					"test": "substring(hexstring(pkt4.mac,':'),0,5) == '02:01'",
				},
				map[string]interface{}{
					"name": "class-02-02",
					"test": "substring(hexstring(pkt4.mac,':'),0,5) == '02:02'",
				},
			},
			"hooks-libraries": []interface{}{
				map[string]interface{}{
					"library": "/usr/lib/x8664-linux-gnu/kea/hooks/libdhcpLeaseCmds.so",
				},
				map[string]interface{}{
					"library": "/usr/lib/x8664-linux-gnu/kea/hooks/libdhcpStatCmds.so",
				},
			},

			"subnet4": generateV4Subnet(outer, inner),

			"loggers": []interface{}{
				map[string]interface{}{
					"name": "kea-dhcp4",
					"outputOptions": []interface{}{
						map[string]interface{}{
							"output":  "stdout",
							"pattern": "%-5p %m\n",
						},
						map[string]interface{}{
							"output": "/tmp/kea-dhcp4.log",
						},
					},
					"severity":   "DEBUG",
					"debuglevel": 0,
				},
			},
		},
	}

	return config
}

// Generate IPv4 subnets.
// It accepts two arguments that specify the number of created outer and inner networks.
// Subnets have random option data and subsequent ID.
func generateV4Subnet(rangeOfOuterScope int, rangeOfInnerScope int) interface{} {
	var subnets []interface{}
	netmask := 8
	if rangeOfInnerScope != 0 {
		netmask = 16
	}

	optionData4 := []interface{}{
		map[string]interface{}{"code": 2, "data": "50", "name": "time-offset", "space": "dhcp4"},
		map[string]interface{}{"code": 3, "data": "100.100.100.10,50.50.50.5", "name": "routers", "space": "dhcp4"},
		map[string]interface{}{"code": 4, "data": "199.199.199.1,199.199.199.2", "name": "time-servers", "space": "dhcp4"},
		map[string]interface{}{"code": 5, "data": "199.199.199.1,100.100.100.1", "name": "name-servers", "space": "dhcp4"},
		map[string]interface{}{"code": 6, "data": "199.199.199.1,100.100.100.1", "name": "domain-name-servers", "space": "dhcp4"},
		map[string]interface{}{"code": 7, "data": "199.199.199.1,100.100.100.1", "name": "log-servers", "space": "dhcp4"},
		map[string]interface{}{"code": 76, "data": "199.1.1.1,200.1.1.2", "name": "streettalk-directory-assistance-server", "space": "dhcp4"},
		map[string]interface{}{"code": 19, "csv-format": true, "data": "True", "name": "ip-forwarding", "space": "dhcp4"},
		map[string]interface{}{"code": 20, "data": "True", "name": "non-local-source-routing", "space": "dhcp4"},
		map[string]interface{}{"code": 29, "data": "False", "name": "perform-mask-discovery", "space": "dhcp4"},
	}

	for outerScope := 1; outerScope <= rangeOfOuterScope; outerScope++ {
		for innerScope := 0; innerScope <= rangeOfInnerScope; innerScope++ {
			subnetNetmask := 255
			if netmask == 16 {
				subnetNetmask = innerScope
			}
			subnet := map[string]interface{}{
				"pools": []interface{}{
					map[string]interface{}{
						"pool": fmt.Sprintf("%d.%d.0.4-%d.%d.255.254",
							outerScope, innerScope, outerScope, subnetNetmask),
					},
				},
				"subnet":       fmt.Sprintf("%d.%d.0.0/%d", outerScope, innerScope, netmask),
				"option-data":  optionData4[rand.Intn(len(optionData4))], //nolint:gosec
				"client-class": "class-00-00",
				"relay": map[string]interface{}{
					"ip-addresses": []string{"172.100.0.200"},
				},
				"id": (outerScope-1)*(rangeOfInnerScope+1) + innerScope + 1,
			}
			subnets = append(subnets, subnet)
		}
	}

	return subnets
}
