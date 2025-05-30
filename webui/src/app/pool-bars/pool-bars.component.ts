import { Component, Input, OnInit } from '@angular/core'
import { DelegatedPrefixPool, Pool } from '../backend'
import { RangedSet, IPv6CidrRange, IPv4, IPv6, IPCidrRange } from 'ip-num'

/**
 * A component displaying address pool and delegated prefix pool bars in a
 * single area. The bars are properly aligned and sorted.
 */
@Component({
    selector: 'app-pool-bars',
    templateUrl: './pool-bars.component.html',
    styleUrl: './pool-bars.component.sass',
})
export class PoolBarsComponent implements OnInit {
    /**
     * Address pools to be displayed.
     */
    @Input() addressPools: Pool[] = []

    /**
     * Delegated prefix pools to be displayed.
     */
    @Input() pdPools: DelegatedPrefixPool[] = []

    /**
     * Address pools grouped by their IDs.
     * The pools with the same ID but different families are separated into
     * different arrays.
     */
    addressPoolsGrouped: [number, Pool[]][] = []

    /**
     * Delegated prefix pools grouped by their IDs.
     * The pools with the same ID but different families are separated into
     * different arrays.
     */
    pdPoolsGrouped: [number, DelegatedPrefixPool[]][] = []

    /**
     * Splits the pools into groups by their IDs and families and sorts them.
     */
    ngOnInit(): void {
        this.addressPoolsGrouped = this.sortGroups(
            this.groupById(this.addressPools ?? []),
            this.compareAddressPools.bind(this)
        )
        this.pdPoolsGrouped = this.sortGroups(
            this.groupById(this.pdPools ?? []),
            this.compareDelegatedPrefixPools.bind(this)
        )
    }

    /**
     * Compares two address pools. It is expected that the pools are from the
     * same family (IPv4 or IPv6).
     * The pools are compared by their ranges.
     *
     * @param poolA - The first address pool to compare.
     * @param poolB - The second address pool to compare.
     * @returns A negative number if poolA is less than poolB, a positive number if poolA is greater than poolB, or 0 if they are equal.
     */
    private compareAddressPools(poolA: Pool, poolB: Pool): number {
        const rangeA = RangedSet.fromRangeString(poolA.pool) as RangedSet<IPv4> & RangedSet<IPv6>
        const rangeB = RangedSet.fromRangeString(poolB.pool) as RangedSet<IPv4> & RangedSet<IPv6>
        return rangeA.isLessThan(rangeB) ? -1 : rangeA.isGreaterThan(rangeB) ? 1 : 0
    }

    /**
     * Compares two delegated prefix pools. It is expected that the prefixes are
     * from the same family (IPv4 or IPv6).
     * The pools are compared by their prefixes, delegated lengths, and excluded prefixes.
     *
     * @param poolA - The first delegated prefix pool to compare.
     * @param poolB - The second delegated prefix pool to compare.
     * @returns A negative number if poolA is less than poolB, a positive number if poolA is greater than poolB, or 0 if they are equal.
     */
    private compareDelegatedPrefixPools(poolA: DelegatedPrefixPool, poolB: DelegatedPrefixPool): number {
        let result = this.comparePrefixes(poolA.prefix, poolB.prefix)
        if (result !== 0) {
            // If the prefixes are not equal, return the result.
            return result
        }

        // If equal, compare the delegated lengths.
        result = poolA.delegatedLength - poolB.delegatedLength
        if (result !== 0) {
            // If the delegated lengths are not equal, return the result.
            return result
        }

        // If equal, compare the excluded prefixes. Remember that the
        // excluded prefixes are optional.
        if (!poolA.excludedPrefix && !poolB.excludedPrefix) {
            return 0
        } else if (!poolA.excludedPrefix) {
            return -1
        } else if (!poolB.excludedPrefix) {
            return 1
        } else {
            // Compare the excluded prefixes.
            return this.comparePrefixes(poolA.excludedPrefix, poolB.excludedPrefix)
        }
    }

    /**
     * Compares two prefixes in format "subnet/mask" (e.g. "2001:db8::/32").
     *
     * @param prefixAStr - The first prefix to compare
     * @param prefixBStr - The second prefix to compare.
     * @returns A negative number if prefixAStr is less than prefixBStr, a
     *          positive number if prefixAStr is greater than prefixBStr, or 0
     *          if they are equal.
     */
    private comparePrefixes(prefixAStr: string, prefixBStr: string): number {
        let prefixA: IPv6CidrRange = null
        let prefixB: IPv6CidrRange = null
        try {
            prefixA = IPv6CidrRange.fromCidr(prefixAStr)
        } catch (e) {
            // Prefix is invalid. Set to null to handle it gracefully.
        }
        try {
            prefixB = IPv6CidrRange.fromCidr(prefixBStr)
        } catch (e) {
            // Prefix is invalid. Set to null to handle it gracefully.
        }

        // Check if the prefixes are valid. Valid prefixes take precedence over
        // invalid ones.
        if (prefixA == null && prefixB == null) {
            // Both invalid. Compare as strings.
            return prefixAStr.localeCompare(prefixBStr)
        } else if (prefixA == null) {
            // Valid prefixB, invalid prefixA.
            return 1
        } else if (prefixB == null) {
            // Valid prefixA, invalid prefixB.
            return -1
        }

        const firstA = prefixA.getFirst()
        const firstB = prefixB.getFirst()
        // Compare the first addresses of the prefixes.
        const result = firstA.isLessThan(firstB) ? -1 : firstA.isGreaterThan(firstB) ? 1 : 0
        if (result !== 0) {
            return result
        }
        // If equal, compare the prefix masks.
        const maskA = prefixAStr.split('/')[1]
        const maskB = prefixBStr.split('/')[1]
        return parseInt(maskA) - parseInt(maskB)
    }

    /**
     * Groups pools by their IDs. The pools from various families are separated
     * into two arrays.
     * @param pools The pools to group.
     * @returns An array of tuples containing the pool ID, the pools, and the family (4 or 6).
     */
    private groupById<T extends Pool | DelegatedPrefixPool>(pools: T[]): [number, T[], number][] {
        return (
            Array.from(
                pools
                    .reduce((acc, pool) => {
                        const poolId = pool.keaConfigPoolParameters?.poolID ?? 0
                        const isIPv4 = (pool as Pool).pool?.includes('.')
                        if (!acc.has(poolId)) {
                            acc.set(poolId, [[], []])
                        }
                        acc.get(poolId)[isIPv4 ? 0 : 1].push(pool)
                        return acc
                    }, new Map<number, [T[], T[]]>())
                    .entries()
            )
                // Split the values into two arrays: one for IPv4 and one for IPv6.
                .flatMap(
                    ([id, pools]) =>
                        [
                            [id, pools[0], 4],
                            [id, pools[1], 6],
                        ] as [number, T[], number][]
                )
                // Filter out empty families.
                .filter((group) => group[1].length > 0)
        )
    }

    /**
     * Sorts the pools in groups following these rules:
     * 1. Groups with the single pool take precedence over groups with multiple pools.
     * 2. Groups with the single pool are sorted by the pool properties.
     * 3. Groups with multiple pools are sorted by the pool ID.
     * 4. Pools in the group are sorted by the pool properties.
     * 5. The group contains only pools from the same family.
     * 6. IPv4 groups take precedence over IPv6 groups.
     */
    private sortGroups<T extends Pool | DelegatedPrefixPool>(
        groups: [number, T[], number][],
        compare: (a: T, b: T) => number
    ): [number, T[]][] {
        // Sort pools in each group.
        return groups
            .map((group) => {
                const [id, pools, family] = group
                return [id, pools.sort(compare), family] as [number, T[], number]
            })
            .sort((groupA, groupB) => {
                // Sort groups.
                const [idA, poolsA, familyA] = groupA
                const [idB, poolsB, familyB] = groupB
                // Sort by family first.
                if (familyA !== familyB) {
                    return familyA - familyB
                }
                // One group has a single pool and the other group has multiple pools.
                if (poolsA.length === 1 && poolsB.length > 1) {
                    return -1
                } else if (poolsA.length > 1 && poolsB.length === 1) {
                    return 1
                } else if (poolsA.length === 1 && poolsB.length === 1) {
                    // Both groups have a single pool.
                    const a = poolsA[0]
                    const b = poolsB[0]
                    return compare(a, b)
                } else {
                    // Both groups have multiple pools.
                    return idA - idB
                }
            })
            .map((group) => [group[0], group[1]]) // Remove the family from the result.
    }
}
