package testutil

import _ "embed"

//go:embed config/kea/keaconfig_test_dhcp4_all_keys.json
var AllKeysDHCPv4JSON []byte

//go:embed config/kea/keaconfig_test_dhcp6_all_keys.json
var AllKeysDHCPv6JSON []byte
