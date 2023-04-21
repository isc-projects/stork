import { DelegatedPrefix, LocalSubnet, SharedNetwork, Subnet } from './backend'

/**
 * Represents a subnet with the lists of unique pools extracted.
 */
export class SubnetWithUniquePools implements Subnet {
    id?: number
    subnet?: string
    sharedNetwork?: string
    clientClass?: string
    addrUtilization?: number
    pdUtilization?: number
    stats?: object
    statsCollectedAt?: string
    localSubnets?: Array<LocalSubnet>
    pools?: Array<string> = []
    prefixDelegationPools?: Array<DelegatedPrefix> = []
}

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

/**
 * Converts the list of subnets into the subnets with extracted unique pools.
 *
 * The address and delegated prefix pools are carried in the objects associating
 * them with the respective DHCP servers. The servers with the same subnets often
 * have the same pools configured (e.g. in the high availability case). This function
 * detects pools eliminating the repeated ones. The returned list of subnets contains
 * the lists of unique pools found on both servers.
 *
 * @param subnets a list of subnets received from the Stork server.
 * @returns a list of converted subnets with the list of unique pools attached.
 */
export function extractUniqueSubnetPools(subnets: Subnet[]): SubnetWithUniquePools[] {
    let convertedSubnets: SubnetWithUniquePools[] = []
    for (const subnet of subnets) {
        let pools: Array<string> = []
        let prefixDelegationPools: Array<DelegatedPrefix> = []
        let convertedSubnet: SubnetWithUniquePools = subnet
        convertedSubnets.push(convertedSubnet)
        if (!subnet.localSubnets) {
            continue
        }
        for (const ls of subnet.localSubnets) {
            if (ls.pools) {
                for (const pool of ls.pools) {
                    // Add the pool only if it doesn't exist yet.
                    if (!pools.includes(pool)) {
                        pools.push(pool)
                    }
                }
            }
            if (ls.prefixDelegationPools) {
                for (const pdPool of ls.prefixDelegationPools) {
                    // Add the pool only if the identical pool doesn't exist yet.
                    if (
                        !prefixDelegationPools.some(
                            (p) =>
                                p.prefix === pdPool.prefix &&
                                p.delegatedLength === pdPool.delegatedLength &&
                                p.excludedPrefix === pdPool.excludedPrefix
                        )
                    ) {
                        prefixDelegationPools.push(pdPool)
                    }
                }
            }
        }
        if (pools.length) {
            convertedSubnet.pools = pools.sort()
        }
        if (prefixDelegationPools.length) {
            convertedSubnet.prefixDelegationPools = prefixDelegationPools.sort((a, b) =>
                a.prefix.localeCompare(b.prefix)
            )
        }
    }
    return convertedSubnets
}
