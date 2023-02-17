import { LocalSubnet, SharedNetwork, Subnet } from './backend'

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

    const stats = subnet.stats as Record<string, number | bigint>

    if ('total-addresses' in stats) {
        return stats['total-addresses']
    } else {
        return stats['total-nas']
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
export function parseSubnetsStatisticValues(subnets: Subnet[] | SharedNetwork[] | LocalSubnet[]): void {
    for (const subnet of subnets) {
        // Parse the nested statistics.
        if ('subnets' in subnet) {
            parseSubnetsStatisticValues(subnet.subnets)
        }
        if ('localSubnets' in subnet) {
            parseSubnetsStatisticValues(subnet.localSubnets)
        }

        // Parse the own statistics.
        if (subnet.stats == null) {
            return
        }

        for (const statName of Object.keys(subnet.stats)) {
            if (typeof subnet.stats[statName] !== 'string') {
                return
            }

            try {
                subnet.stats[statName] = BigInt(subnet.stats[statName])
            } catch {
                // Non-integer string
                return
            }
        }
    }
}
