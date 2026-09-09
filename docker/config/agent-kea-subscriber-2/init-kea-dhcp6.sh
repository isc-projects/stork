#!/bin/sh
set -eu

kea-shell --service dhcp6 remote-server6-set << 'EOF'
"servers": [
    {
        "server-tag": "agent-kea-subscriber-2",
        "description": "A DHCP server for testing purposes."
    }
]
EOF

kea-shell --service dhcp6 remote-subnet6-set << 'EOF'
"subnets": [
    {
        "id": 407,
        "subnet": "2000:db8:21::/64",
        "shared-network-name": null,
        "pools": [ { "pool": "2000:db8:21::100-2000:db8:21::200" } ],
        "client-class": "class-12-01-eth12-RELAY",
        "relay": {
            "ip-addresses": [ "3012:db8:1::200" ]
        }
    }
],
"server-tags": [ "agent-kea-subscriber-2" ]
EOF
