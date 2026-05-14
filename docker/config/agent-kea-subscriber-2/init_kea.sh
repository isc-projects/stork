#!/bin/sh
set -eu

kea-shell --service dhcp4 remote-server4-set << 'EOF'
"servers": [
    {
        "server-tag": "agent-kea-subscriber-2",
        "description": "A DHCP server for testing purposes."
    }
]
EOF

kea-shell --service dhcp4 remote-subnet4-set << 'EOF'
"subnets": [
    {
        "id": 7,
        "subnet": "192.0.21.0/24",
        "shared-network-name": null,
        "pools": [ { "pool": "192.0.21.100-192.0.21.200" } ]
    }
],
"server-tags": [ "agent-kea-subscriber-2" ]
EOF
