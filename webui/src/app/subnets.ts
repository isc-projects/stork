/**
 * Get total number of addresses in a subnet.
 * It is taken from DHCPv4 or DHCPv6 stats respectively.
 * In DHCPv6 if total is -1 in stats then max safe int is returned.
 */
export function getTotalAddresses(subnet) {
    if (subnet.subnet.includes('.')) {
        // DHCPv4 stats
        return subnet.localSubnets[0].stats['total-addresses']
    } else {
        // DHCPv6 stats
        let total = subnet.localSubnets[0].stats['total-nas']
        if (total === -1) {
            total = Number.MAX_SAFE_INTEGER
        }
        return total
    }
}

/**
 * Get assigned number of addresses in a subnet.
 * It is taken from DHCPv4 or DHCPv6 stats respectively.
 */
export function getAssignedAddresses(subnet) {
    if (subnet.subnet.includes('.')) {
        // DHCPv4 stats
        return subnet.localSubnets[0].stats['assigned-addresses']
    } else {
        // DHCPv6 stats
        return subnet.localSubnets[0].stats['assigned-nas']
    }
}
