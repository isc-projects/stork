#!/bin/sh
set -eu

kea-shell --service dhcp6 remote-server6-set << 'EOF'
"servers": [
    {
        "server-tag": "agent-kea-subscriber-1",
        "description": "A DHCP server for testing purposes."
    }
]
EOF

kea-shell --service dhcp6 remote-network6-set << 'EOF'
"shared-networks": [
    {
        "name": "palma"
    }
],
"server-tags": [ "all" ]
EOF

kea-shell --service dhcp6 remote-subnet6-set << 'EOF'
"subnets": [
    {
        "id": 405,
        "subnet": "2000:db8:19::/64",
        "shared-network-name": "palma",
        "pools": [ { "pool": "2000:db8:19::100-2000:db8:19::200" } ],
        "client-class": "class-11-01-eth11-RELAY",
        "relay": {
            "ip-addresses": [ "3011:db8:1::200" ]
        }
    }
],
"server-tags": [ "all" ]
EOF

kea-shell --service dhcp6 remote-subnet6-set << 'EOF'
"subnets": [
    {
        "id": 406,
        "subnet": "2000:db8:20::/64",
        "shared-network-name": null,
        "pools": [ { "pool": "2000:db8:20::100-2000:db8:20::200" } ],
        "client-class": "class-11-02-eth11-RELAY",
        "relay": {
            "ip-addresses": [ "3011:db8:1::200" ]
        }
    }
],
"server-tags": [ "agent-kea-subscriber-1" ]
EOF
