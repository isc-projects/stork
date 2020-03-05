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
 * Get subnet utilization in % based on stats.
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
