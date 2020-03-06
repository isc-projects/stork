/**
 * Get total number of addresses in a subnet.
 * It is taken from DHCPv4 or DHCPv6 stats respectively.
 * In DHCPv6 if total is -1 in stats then max safe int is returned.
 */
export function getTotalAddresses(subnet) {
    if (subnet.subnet.includes('.')) {
        // DHCPv4 stats
        return subnet.stats['total-addreses']
    } else {
        // DHCPv6 stats
        let total = subnet.stats['total-nas']
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
        return subnet.stats['assigned-addreses']
    } else {
        // DHCPv6 stats
        return subnet.stats['assigned-nas']
    }
}

/**
 * Get subnet utilization based on stats. A percentage is returned as floored int.
 */
export function getSubnetUtilization(subnet) {
    let utilization = 0.0
    if (!subnet.stats) {
        return utilization
    }
    const total = getTotalAddresses(subnet)
    const assigned = getAssignedAddresses(subnet)
    utilization = (100 * assigned) / total
    return Math.floor(utilization)
}
