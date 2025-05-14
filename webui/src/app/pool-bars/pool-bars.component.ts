import { Component, Input, OnInit } from '@angular/core'
import { DelegatedPrefixPool, Pool } from '../backend'
import { RangedSet, IPv6CidrRange, IPv4, IPv6 } from 'ip-num'

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
     */
    addressPoolsGrouped: [number, Pool[]][] = []

    /**
     * Delegated prefix pools grouped by their IDs.
     */
    pdPoolsGrouped: [number, DelegatedPrefixPool[]][] = []

    /**
     * Splits the pools into groups by their IDs and families and sorts them.
     */
    ngOnInit(): void {
        this.addressPoolsGrouped = this.sortGroups(this.groupById(this.addressPools ?? []), (a, b) => {
            // The IPv4 and IPv6 pools are split into separate arrays.
            const rangeA = RangedSet.fromRangeString(a.pool) as RangedSet<IPv4> & RangedSet<IPv6>
            const rangeB = RangedSet.fromRangeString(b.pool) as RangedSet<IPv4> & RangedSet<IPv6>
            return rangeA.isLessThan(rangeB) ? -1 : rangeA.isGreaterThan(rangeB) ? 1 : 0
        })
        this.pdPoolsGrouped = this.sortGroups(this.groupById(this.pdPools ?? []), (a, b) => {
            // The IPv4 and IPv6 pools are split into separate arrays.
            const prefixA = IPv6CidrRange.fromCidr(a.prefix)
            const prefixB = IPv6CidrRange.fromCidr(b.prefix)
            const firstA = prefixA.getFirst()
            const firstB = prefixB.getFirst()
            let result = firstA.isLessThan(firstB) ? -1 : firstA.isGreaterThan(firstB) ? 1 : 0
            if (result === 0) {
                const maskA = a.prefix.split('/')[1]
                const maskB = b.prefix.split('/')[1]
                result = parseInt(maskA) - parseInt(maskB)
            }
            if (result === 0) {
                result = a.delegatedLength - b.delegatedLength
            }
            if (result === 0) {
                result = a.excludedPrefix?.localeCompare(b.excludedPrefix ?? '') ?? 0
            }
            return result
        })
    }

    /**
     * Groups pools by their IDs. The pools from various families are separated
     * into two arrays.
     * @param pools
     * @returns
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
