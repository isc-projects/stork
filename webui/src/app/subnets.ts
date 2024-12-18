import { AddressRange } from './address-range'
import {
    DelegatedPrefixPool,
    KeaConfigPoolParameters,
    KeaDaemonConfig,
    LocalSubnet,
    Pool,
    SharedNetwork,
    Subnet,
} from './backend'
import { deepEqual } from './utils'

/**
 * Association of the pool with a daemon and daemon-specific pool
 * parameters.
 */
export interface LocalPool {
    daemonId?: number
    appName?: string
    keaConfigPoolParameters?: KeaConfigPoolParameters
}

/**
 * A shared address pool with daemon-specific data sets.
 */
export interface PoolWithLocalPools extends Pool {
    localPools?: LocalPool[]
}

/**
 * A shared delegated prefix pool with daemon-specific data sets.
 */
export interface DelegatedPrefixPoolWithLocalPools extends DelegatedPrefixPool {
    localPools?: LocalPool[]
}

/**
 * Represents a shared network with the lists of unique pools extracted.
 */
export interface SharedNetworkWithUniquePools extends SharedNetwork {
    pools?: Array<Pool>
    prefixDelegationPools?: Array<DelegatedPrefixPool>
}

/**
 * Represents a subnet with the lists of unique pools extracted.
 */
export interface SubnetWithUniquePools extends Subnet {
    pools?: Array<PoolWithLocalPools>
    prefixDelegationPools?: Array<DelegatedPrefixPoolWithLocalPools>
}

/**
 * Get total number of addresses in a subnet.
 * It is taken from DHCPv4 or DHCPv6 stats respectively.
 */
export function getTotalAddresses(subnet: Subnet | SharedNetwork): number | bigint | null {
    // DHCPv4 or DHCPv6 stats
    if (subnet.stats == null) {
        return null
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
        return null
    }

    const stats = subnet.stats as Record<string, number | bigint>
    if ('assigned-addresses' in stats) {
        return subnet.stats['assigned-addresses']
    } else {
        return subnet.stats['assigned-nas']
    }
}

/**
 * Get a BigInt value of statistic with the given name. If the statistic is
 * missing, subnet lacks the statistics, or the value is not numeric, returns
 * null.
 */
export function getStatisticValue(subnet: Subnet | SharedNetwork, name: string): bigint | null {
    if (subnet.stats == null || !subnet.stats.hasOwnProperty(name)) {
        return null
    }
    let value = subnet.stats[name]
    switch (typeof value) {
        case 'bigint':
            return value
        case 'number':
            return BigInt(value)
        default:
            return null
    }
}
/**
 * Parses all string statistics of subnet-like object as big integers.
 * Stork Server returns big counters (e.g., IPv6 statistics) as strings to
 * prevent problems with floating-point number precision. The big integers are
 * not supported by the OpenAPI specification.
 */
export function parseSubnetStatisticValues(subnet: Subnet | SharedNetwork | LocalSubnet): void {
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

/**
 * Helper that converts the statistic values from string to big integer.
 * It is necessary because the statistics can store large numbers and standard
 * JSON parser converts them to double precision number. It causes losing precision.
 */
export function parseSubnetsStatisticValues(subnets: Subnet[] | SharedNetwork[] | LocalSubnet[]): void {
    subnets?.forEach((s: Subnet) => parseSubnetStatisticValues(s))
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
export function extractUniqueSubnetPools(subnets: Subnet[] | Subnet): SubnetWithUniquePools[] {
    let convertedSubnets: SubnetWithUniquePools[] = []
    for (const subnet of Array.isArray(subnets) ? subnets : [subnets]) {
        let pools: Array<PoolWithLocalPools> = []
        let prefixDelegationPools: Array<DelegatedPrefixPoolWithLocalPools> = []
        let convertedSubnet: SubnetWithUniquePools = subnet
        convertedSubnets.push(convertedSubnet)
        if (subnet.localSubnets) {
            for (const ls of subnet.localSubnets) {
                if (ls.pools) {
                    for (const pool of ls.pools) {
                        const lp: LocalPool = {
                            daemonId: ls.daemonId,
                            appName: ls.appName,
                            keaConfigPoolParameters: pool.keaConfigPoolParameters,
                        }
                        const existing = pools.find((p) => p.pool === pool.pool)
                        // Add the pool only if it doesn't exist yet.
                        if (existing) {
                            existing.localPools.push(lp)
                        } else {
                            let p: PoolWithLocalPools = pool
                            p.localPools = [lp]
                            pools.push(p)
                        }
                    }
                }
                if (ls.prefixDelegationPools) {
                    for (const pdPool of ls.prefixDelegationPools) {
                        const lp: LocalPool = {
                            daemonId: ls.daemonId,
                            appName: ls.appName,
                            keaConfigPoolParameters: pdPool.keaConfigPoolParameters,
                        }
                        const existing = prefixDelegationPools.find((p) => {
                            return (
                                p.prefix === pdPool.prefix &&
                                p.delegatedLength === pdPool.delegatedLength &&
                                p.excludedPrefix === pdPool.excludedPrefix
                            )
                        })
                        // Add the pool only if the identical pool doesn't exist yet.
                        if (existing) {
                            existing.localPools.push(lp)
                        } else {
                            let p: DelegatedPrefixPoolWithLocalPools = pdPool
                            p.localPools = [lp]
                            prefixDelegationPools.push(p)
                        }
                    }
                }
            }
        }
        convertedSubnet.pools = pools.sort((a, b) => {
            try {
                const range1 = AddressRange.fromStringRange(a.pool)
                const range2 = AddressRange.fromStringRange(b.pool)
                if (range1.first.isLessThan(range2.first)) {
                    return -1
                } else if (range1.first.isGreaterThan(range2.first)) {
                    return 1
                }
            } catch {
                // Parsing error is very unlikely but we have to handle it.
                return 1
            }
            return 0
        })
        convertedSubnet.prefixDelegationPools = prefixDelegationPools.sort((a, b) => a.prefix.localeCompare(b.prefix))
    }
    return convertedSubnets
}

/**
 * Convenience function checking if the subnet has any address pools.
 *
 * @param subnet subnet instance with local subnet instances.
 * @returns true if the subnet includes at least one address pool.
 */
export function hasAddressPools(subnet: Subnet): boolean {
    if (!subnet.localSubnets || subnet.localSubnets.length === 0) {
        return false
    }
    for (const ls of subnet.localSubnets) {
        if (ls.pools?.length > 0) {
            return true
        }
    }
    return false
}

/**
 * Convenience function checking if the subnet has any delegated prefix pools.
 *
 * @param subnet subnet instance with local subnet instances.
 * @returns true if the subnet includes at least one delegated prefix pool.
 */
export function hasPrefixPools(subnet: Subnet): boolean {
    if (!subnet.localSubnets || subnet.localSubnets.length === 0) {
        return false
    }
    for (const ls of subnet.localSubnets) {
        if (ls.prefixDelegationPools?.length > 0) {
            return true
        }
    }
    return false
}

/**
 * Convenience function checking if the servers using a subnet have different
 * pools defined for it.
 *
 * @param subnet subnet instance with local subnet instances.
 * @returns true if servers using the subnet have different pools defined for
 * it, false otherwise.
 */
export function hasDifferentLocalSubnetPools(subnet: Subnet): boolean {
    if (!subnet.localSubnets || subnet.localSubnets.length <= 1) {
        return false
    }
    for (let i = 1; i < subnet.localSubnets.length; i++) {
        // Check the case when pools in one server are undefined and defined
        // in second server and if lengths are different.
        if (subnet.localSubnets[0].pools?.length !== subnet.localSubnets[i].pools?.length) {
            return true
        }
        if (
            subnet.localSubnets[0].prefixDelegationPools?.length !==
            subnet.localSubnets[i].prefixDelegationPools?.length
        ) {
            return true
        }
        // Check for different address pools.
        if (subnet.localSubnets[i].pools && subnet.localSubnets[0].pools) {
            for (const pool of subnet.localSubnets[i].pools) {
                if (!subnet.localSubnets[0].pools.some((p) => p.pool === pool.pool)) {
                    return true
                }
            }
        }
        // Check for different prefix pools.
        if (subnet.localSubnets[i].prefixDelegationPools && subnet.localSubnets[0].prefixDelegationPools) {
            for (const pool of subnet.localSubnets[i].prefixDelegationPools) {
                if (
                    !subnet.localSubnets[0].prefixDelegationPools.some((p) => {
                        return (
                            p.prefix === pool.prefix &&
                            p.delegatedLength === pool.delegatedLength &&
                            p.excludedPrefix === pool.excludedPrefix
                        )
                    })
                ) {
                    return true
                }
            }
        }
    }
    return false
}

/**
 * Utility function checking if there are differences between
 * DHCP options in a pool for different servers.
 *
 * @param pool pool instance.
 * @returns true if there are differences in DHCP options, false otherwise.
 */
export function hasDifferentLocalPoolOptions(pool: PoolWithLocalPools | DelegatedPrefixPoolWithLocalPools): boolean {
    return (
        !!(pool.localPools?.length > 0) &&
        pool.localPools
            .slice(1)
            .some(
                (lp) =>
                    lp.keaConfigPoolParameters?.optionsHash !== pool.localPools[0].keaConfigPoolParameters?.optionsHash
            )
    )
}

/**
 * Utility function checking if there are differences between
 * DHCP options in the subnet.
 *
 * It checks differences between the option hashes at all configuration
 * inheritance levels (i.e., subnet, shared network and global).
 *
 * @param subnet subnet instance.
 * @returns true if there are differences in DHCP options, false otherwise.
 */
export function hasDifferentLocalSubnetOptions(subnet: Subnet): boolean {
    return (
        !!(subnet.localSubnets?.length > 0) &&
        subnet.localSubnets
            .slice(1)
            .some(
                (ls) =>
                    ls.keaConfigSubnetParameters?.subnetLevelParameters?.optionsHash !==
                        subnet.localSubnets[0].keaConfigSubnetParameters?.subnetLevelParameters?.optionsHash ||
                    ls.keaConfigSubnetParameters?.sharedNetworkLevelParameters?.optionsHash !==
                        subnet.localSubnets[0].keaConfigSubnetParameters?.sharedNetworkLevelParameters?.optionsHash ||
                    ls.keaConfigSubnetParameters?.globalParameters?.optionsHash !==
                        subnet.localSubnets[0].keaConfigSubnetParameters?.globalParameters?.optionsHash
            )
    )
}

/**
 * Utility function checking if there are differences between subnet-level
 * DHCP options in the subnet.
 *
 * @param subnet subnet instance.
 * @returns true if there are differences in DHCP options, false otherwise.
 */
export function hasDifferentSubnetLevelOptions(subnet: Subnet) {
    return (
        !!(subnet.localSubnets?.length > 0) &&
        subnet.localSubnets
            .slice(1)
            .some(
                (ls) =>
                    ls.keaConfigSubnetParameters?.subnetLevelParameters?.optionsHash !==
                    subnet.localSubnets[0].keaConfigSubnetParameters?.subnetLevelParameters?.optionsHash
            )
    )
}

/**
 * Utility function checking if there are differences between global-level
 * DHCP options.
 *
 * @param configs Configs from various daemons.
 * @returns true if there are differences in DHCP options, false otherwise.
 */
export function hasDifferentGlobalLevelOptions(configs: KeaDaemonConfig[]): boolean {
    return (
        configs.length > 0 &&
        configs
            .map((c) => c.options)
            .slice(1)
            .some((opts) => opts?.optionsHash !== configs[0].options?.optionsHash)
    )
}

/**
 * Utility function checking if there are differences between user contexts in
 * subnet.
 *
 * @param subnet subnet instance.
 * @returns true if there are differences in subnet names, false otherwise.
 */
export function hasDifferentSubnetUserContexts(subnet: Subnet): boolean {
    if (!subnet.localSubnets?.length) {
        return false
    }

    return subnet.localSubnets.slice(1).some((ls) => !deepEqual(ls.userContext, subnet.localSubnets[0].userContext))
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
export function extractUniqueSharedNetworkPools(sharedNetworks: SharedNetwork[]): SharedNetworkWithUniquePools[] {
    let convertedSharedNetworks: SharedNetworkWithUniquePools[] = []
    for (const sharedNetwork of sharedNetworks) {
        let pools: Array<Pool> = []
        let prefixDelegationPools: Array<DelegatedPrefixPool> = []
        let convertedSharedNetwork: SharedNetworkWithUniquePools = sharedNetwork
        convertedSharedNetworks.push(convertedSharedNetwork)
        if (!sharedNetwork.subnets) {
            continue
        }
        const convertedSubnets = extractUniqueSubnetPools(convertedSharedNetwork.subnets)
        for (const subnet of convertedSubnets) {
            if (subnet.pools) {
                for (const pool of subnet.pools) {
                    // Add the pool only if it doesn't exist yet.
                    if (!pools.some((p) => p.pool === pool.pool)) {
                        pools.push(pool)
                    }
                }
            }
            if (subnet.prefixDelegationPools) {
                for (const pdPool of subnet.prefixDelegationPools) {
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
            convertedSharedNetwork.pools = pools.sort()
        }
        if (prefixDelegationPools.length) {
            convertedSharedNetwork.prefixDelegationPools = prefixDelegationPools.sort((a, b) =>
                a.prefix.localeCompare(b.prefix)
            )
        }
    }
    return convertedSharedNetworks
}

/**
 * Utility function checking if there are differences between
 * DHCP options in the shared network.
 *
 * It checks differences between the option hashes at shared network and global
 * configuration inheritance levels.
 *
 * @param sharedNetwork shared network instance.
 * @returns true if there are differences in DHCP options, false otherwise.
 */
export function hasDifferentLocalSharedNetworkOptions(sharedNetwork: SharedNetwork): boolean {
    return (
        !!(sharedNetwork.localSharedNetworks?.length > 0) &&
        sharedNetwork.localSharedNetworks
            .slice(1)
            .some(
                (ls) =>
                    ls.keaConfigSharedNetworkParameters?.sharedNetworkLevelParameters?.optionsHash !==
                        sharedNetwork.localSharedNetworks[0].keaConfigSharedNetworkParameters
                            ?.sharedNetworkLevelParameters?.optionsHash ||
                    ls.keaConfigSharedNetworkParameters?.globalParameters?.optionsHash !==
                        sharedNetwork.localSharedNetworks[0].keaConfigSharedNetworkParameters?.globalParameters
                            ?.optionsHash
            )
    )
}

/**
 * Utility function checking if there are differences between shared-network-level
 * DHCP options in the shared network.
 *
 * @param sharedNetwork shared network instance.
 * @returns true if there are differences in DHCP options, false otherwise.
 */
export function hasDifferentSharedNetworkLevelOptions(sharedNetwork: SharedNetwork) {
    return (
        !!(sharedNetwork.localSharedNetworks?.length > 0) &&
        sharedNetwork.localSharedNetworks
            .slice(1)
            .some(
                (ls) =>
                    ls.keaConfigSharedNetworkParameters?.sharedNetworkLevelParameters?.optionsHash !==
                    sharedNetwork.localSharedNetworks[0].keaConfigSharedNetworkParameters?.sharedNetworkLevelParameters
                        ?.optionsHash
            )
    )
}
