/**
 * Get total number of addresses in a subnet.
 * It is taken from DHCPv4 or DHCPv6 stats respectively.
 * In DHCPv6 if total is -1 in stats then max safe int is returned.
 */
export function getTotalAddresses(subnet) {
    // DHCPv4 or DHCPv6 stats
    const statName = subnet.subnet.includes('.') ? 'total-addresses' : 'total-nas'
    return subnet.localSubnets[0].stats[statName]
}

/**
 * Get assigned number of addresses in a subnet.
 * It is taken from DHCPv4 or DHCPv6 stats respectively.
 */
export function getAssignedAddresses(subnet) {
    const statName = subnet.subnet.includes('.') ? 'assigned-addresses' : 'assigned-nas'
    return subnet.localSubnets[0].stats[statName]
}

/**
 * Helper that converts parse the statistic values from string to big integer.
 * It necessary because the statistics can store large numbers and standard
 * JSON parser converts them to double precision number. It causes losing precision.
 */
export function parseSubnetsStatisticValues(subnets): void {
    subnets.items.forEach((s) => {
        s.localSubnets.forEach((l) => {
            Object.keys(l.stats).forEach((k) => {
                if (typeof l.stats[k] !== 'string') {
                    return
                }
                try {
                    l.stats[k] = BigInt(l.stats[k])
                } catch {
                    // Non-integer string
                    return
                }
            })
        })
    })
    return subnets
}
