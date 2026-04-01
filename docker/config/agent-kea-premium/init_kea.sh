#!/bin/sh
set -eu

kea-shell --service dhcp4 remote-server4-set << 'EOF'
"servers": [
    {
        "server-tag": "agent-kea-premium",
        "description": "A DHCP server for testing purposes."
    }
]
EOF

kea-shell --service dhcp4 remote-network4-set << 'EOF'
"shared-networks": [
    {
        "name": "palma"
    }
],
"server-tags": [ "agent-kea-premium" ]
EOF

kea-shell --service dhcp4 remote-subnet4-set << 'EOF'
"subnets": [
    {
        "id": 5,
        "subnet": "192.0.19.0/24",
        "shared-network-name": "palma",
        "pools": [ { "pool": "192.0.19.100-192.0.19.200" } ]
    }
],
"server-tags": [ "agent-kea-premium" ]
EOF

kea-shell --service dhcp4 remote-subnet4-set << 'EOF'
"subnets": [
    {
        "id": 6,
        "subnet": "192.0.20.0/24",
        "shared-network-name": null,
        "pools": [ { "pool": "192.0.20.100-192.0.20.200" } ]
    }
],
"server-tags": [ "all" ]
EOF
