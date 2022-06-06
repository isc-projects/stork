import { SharedNetwork, Subnet } from './backend'

/**
 * Get total number of addresses in a subnet.
 * It is taken from DHCPv4 or DHCPv6 stats respectively.
 * In DHCPv6 if total is -1 in stats then max safe int is returned.
 */
export function getTotalAddresses(subnet: Subnet | SharedNetwork): number | bigint | null {
    // DHCPv4 or DHCPv6 stats
    if (subnet.stats == null) {
        return BigInt(0)
    }
    if ('total-addresses' in subnet.stats) {
        return subnet.stats['total-addresses']
    } else {
        return subnet.stats['total-nas']
    }
}

/**
 * Get assigned number of addresses in a subnet.
 * It is taken from DHCPv4 or DHCPv6 stats respectively.
 */
export function getAssignedAddresses(subnet: Subnet | SharedNetwork): number | bigint | null {
    // DHCPv4 or DHCPv6 stats
    if (subnet.stats == null) {
        return BigInt(0)
    }
    if ('total-addresses' in subnet.stats) {
        return subnet.stats['assigned-addresses']
    } else {
        return subnet.stats['assigned-nas']
    }
}

/**
 * Helper that converts the statistic values from string to big integer.
 * It is necessary because the statistics can store large numbers and standard
 * JSON parser converts them to double precision number. It causes losing precision.
 */
export function parseSubnetsStatisticValues(subnets: Subnet[] | SharedNetwork[]): void {
    subnets.forEach((s) => {
        if (s.stats == null) {
            return
        }
        Object.keys(s.stats).forEach((k) => {
            if (typeof s.stats[k] !== 'string') {
                return
            }
            try {
                s.stats[k] = BigInt(s.stats[k])
            } catch {
                // Non-integer string
                return
            }
        })
    })
}
