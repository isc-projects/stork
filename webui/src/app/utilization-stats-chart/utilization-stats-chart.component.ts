import { Component, Input } from '@angular/core'
import { getStatisticValue } from '../subnets'
import { Subnet } from '../backend/model/subnet'
import { SharedNetwork } from '../backend/model/sharedNetwork'
import { clamp } from '../utils'
import { LocalSubnet } from '../backend'
import {} from 'chart.js'

/**
 * A component displaying a pie chart with address or delegated prefix utilization
 * in a subnet or a shared network.
 *
 * The pie chart shows proportions of free, assigned and declined addresses or
 * free and assigned delegated prefixes. If these statistics are unavailable,
 * the pie chart shows the utilization as percentages of assigned and unassigned,
 * using the addrUtilization or pdUtilization respectively.
 */
@Component({
    selector: 'app-utilization-stats-chart',
    templateUrl: './utilization-stats-chart.component.html',
    styleUrls: ['./utilization-stats-chart.component.sass'],
})
export class UtilizationStatsChartComponent {
    /**
     * Pie chart data initialized during the component initialization.
     */
    data: { labels: string[]; datasets: any[] } = null

    /**
     * Total number of leases fetched from the statistics.
     */
    total: bigint = null

    /**
     * Number of assigned leases fetched from the statistics.
     */
    assigned: bigint = null

    /**
     * Number of declined leases fetched from the statistics.
     */
    declined: bigint = null

    /**
     * Address or delegated prefix utilization fetched from the statistics.
     */
    utilization: number = null

    /**
     * Specifies which type of a chart is currently displayed.
     *
     * There are three different cases this component supports:
     * - when detailed statistics are specified and the statistics are
     *   valid (i.e., numbers are consistent and the number of assigned
     *   leases is greater or equal the number of declined leases);
     * - when detailed statistics are not available but the utilization
     *   percentages are specified;
     * - when the number of assigned addresses is lower than the number of
     *   declined addresses.
     *
     * The last case is rare but it is possible due to the bug #3565 in Kea.
     * In this case, it is impossible to determine whether or not the assigned
     * leases include the declined leases. Thus, it is also impossible to
     * exactly tell how many leases in the subnet or shared network are still
     * free. There is some pool of free leases that can be estimated. Availability
     * of the other leases is uncertain. Thus, this component introduces the
     * metrics of uncertain addresses. These addresses can be free but can be
     * unavailable as well.
     */
    chartCase: 'valid' | 'utilization' | 'invalid' | null = null

    /**
     * Optional chart title displayed at the top.
     */
    @Input() title: string

    /**
     * Lease type for which the statistics should be shown.
     */
    @Input() leaseType: 'na' | 'pd'

    /**
     * An instance of a subnet or a shared network holding statistics.
     *
     * It prepares the data to be displayed in a chart using the statistics
     * conveyed in a subnet, local subnet or a shared network.
     */
    @Input() set network(network: LocalSubnet | Subnet | SharedNetwork) {
        this.total = null
        this.assigned = null
        this.declined = null
        this.data = null
        this.utilization = null

        if (!network) {
            return
        }

        const { totalName, assignedName, declinedName } = this.getStatisticNames(network)

        this.total = getStatisticValue(network, totalName)
        this.assigned = getStatisticValue(network, assignedName)
        // The declined statistics is optional. It is missing for delegated prefixes.
        this.declined = getStatisticValue(network, declinedName) ?? 0n

        // Extract the utilization.
        if (this.isPD && 'pdUtilization' in network) {
            this.utilization = clamp(network['pdUtilization'], 0, 100)
        } else if (!this.isPD && 'addrUtilization' in network) {
            this.utilization = clamp(network['addrUtilization'], 0, 100)
        } else {
            this.utilization = null
        }

        // Start preparing the dataset for a chart. Each chart has at least two types
        // of data (i.e., free leases and assigned leases).
        const documentStyle = getComputedStyle(document.documentElement)

        if (this.hasStats) {
            // The chart cannot handle the big integers. Typically, the statistics fit
            // into the 64-bit integers, so this is not a big deal. Let's try to convert
            // the statistics to 64-bit integers.
            const total64 = this.clampTo64(this.total)
            const assigned64 = this.clampTo64(this.assigned)
            const declined64 = this.clampTo64(this.declined)

            // Validate the clamped values. The total64 will be null if it doesn't fit into 64 bits.
            // The total of 0 also cannot be presented on the chart, so we fallback to the percentages
            // in this case. Also, if the assigned and declined counters are too high to fit into
            // 64-bits or they don't make any sense we'd rather use the percentages.
            if (total64 && assigned64 != null && declined64 != null && total64 >= assigned64 && total64 >= declined64) {
                if (assigned64 >= declined64) {
                    // Per Kea design, the assigned leases should include declined leases as well.
                    // If this is the case, we proceed with a valid scenario in which the
                    // chart contains free, used and possibly declined leases.
                    this.chartCase = 'valid'
                    this.data = {
                        labels: ['Free', 'Used'].concat(!this.isPD ? 'Declined' : []),
                        datasets: [
                            {
                                data: [total64 - assigned64, assigned64 - declined64].concat(
                                    !this.isPD ? declined64 : []
                                ),
                                backgroundColor: [
                                    documentStyle.getPropertyValue('--green-500'),
                                    documentStyle.getPropertyValue('--red-500'),
                                ].concat(!this.isPD ? documentStyle.getPropertyValue('--gray-500') : []),
                                hoverBackgroundColor: [
                                    documentStyle.getPropertyValue('--green-400'),
                                    documentStyle.getPropertyValue('--red-400'),
                                ].concat(!this.isPD ? documentStyle.getPropertyValue('--gray-400') : []),
                            },
                        ],
                    }
                    // Calculate the utilization from the statistics if it is missing.
                    if (!this.hasUtilization) {
                        this.utilization = clamp((assigned64 / total64) * 100, 0, 100)
                    }
                } else {
                    // We're getting into an interesting scenario whereby the number of declined
                    // leases exceeds the number of assigned leases. This shouldn't happen but
                    // it sometimes does when client declines an expired or released (unassigned)
                    // lease. In this case, we use "uncertain" metrics instead of "used" metrics
                    // in the chart. The uncertain addresses are those for which we're unable to
                    // tell whether they are allocated, declined or free.

                    // Estimate the number of uncertain leases by taking the worst case when
                    // none of the assigned addresses include the declined addresses. In this
                    // case, if we add declined and assigned, the remaining addresses to total
                    // can be assumed free. All assigned addresses are marked uncertain because
                    // we don't know whether they are in fact assigned or declined.
                    const uncertain64 = Math.min(total64 - declined64, assigned64)
                    const free64 = total64 - uncertain64 - declined64
                    this.chartCase = 'invalid'
                    this.data = {
                        labels: ['Free', 'Uncertain', 'Declined'],
                        datasets: [
                            {
                                data: [free64, uncertain64, declined64],
                                backgroundColor: [
                                    documentStyle.getPropertyValue('--green-500'),
                                    documentStyle.getPropertyValue('--orange-500'),
                                    documentStyle.getPropertyValue('--surface-700'),
                                ],
                                hoverBackgroundColor: [
                                    documentStyle.getPropertyValue('--green-400'),
                                    documentStyle.getPropertyValue('--orange-400'),
                                    documentStyle.getPropertyValue('--surface-600'),
                                ],
                            },
                        ],
                    }
                }
                return
            }
        }
        // If the stats are invalid or missing, we fallback to the utilization.
        if (this.hasUtilization) {
            this.chartCase = 'utilization'
            this.data = {
                labels: ['% Free', '% Used'],
                datasets: [
                    {
                        data: [100 - this.utilization, this.utilization],
                        backgroundColor: [
                            documentStyle.getPropertyValue('--green-500'),
                            documentStyle.getPropertyValue('--red-500'),
                        ],
                        hoverBackgroundColor: [
                            documentStyle.getPropertyValue('--green-400'),
                            documentStyle.getPropertyValue('--red-400'),
                        ],
                    },
                ],
            }
        }
    }

    /**
     * Convenience function checking if the presented statistics are for the
     * prefix delegation.
     *
     * @return true if the statistics are for the prefix delegation, false otherwise.
     */
    get isPD(): boolean {
        return this.leaseType === 'pd'
    }

    /**
     * Convenience function checking if the provided network has statistics.
     */
    get hasStats(): boolean {
        return this.total != null && this.assigned != null
    }

    /**
     * Convenience function checking if the provided network has utilization.
     */
    get hasUtilization(): boolean {
        return this.utilization != null
    }

    /**
     * It is calculated by subtracting declined from assigned addresses.
     */
    get used(): bigint {
        return this.assigned - this.declined
    }

    /**
     * Returns the number of free leases.
     *
     * It takes into account two cases. In the normal case, the number of free leases
     * is simply a difference between the total and assigned addresses because assigned
     * include declined ones. If, however, the number of declines is greater than the
     * number of assigned we assume a worst case scenario that assigned include no declined
     * addresses. In this case, we also substract the number of declined leases to estimate
     * free leases. Note that this function may return a negative value.
     */
    get free(): bigint {
        let free = this.total - this.assigned
        if (this.assigned < this.declined) {
            free -= this.declined
        }
        return free
    }

    /**
     * Returns the number of leases with uncertain state.
     *
     * This metric is only meaningful when the number of declined addresses
     * exceeds the number of assigned addresses. This normally shouldn't be
     * the case because assigned addresses should include the declined addresses.
     * However, due to the bug #3565 in Kea the number of assigned and declined
     * leases may get inconsistent.  In this case, the proportions between the
     * declined and valid leases in the assigned metrics cannot be determined.
     * Thus, the non-declined leases are marked as having an uncertain state.
     */
    get uncertain(): bigint {
        return this.assigned < this.total - this.declined ? this.assigned : this.total - this.declined
    }

    /**
     * Attempts to convert a number to 64-bits.
     *
     * @param stat statistic to be converted to a 64-bit number.
     * @returns converted number or null if the value is too high.
     */
    private clampTo64(stat: bigint): number | null {
        if (!stat) {
            return 0
        }
        if (typeof stat === 'number') {
            return stat
        }
        if (stat < 2n ** 64n - 1n) {
            return Number(stat)
        }
        return null
    }

    /**
     * Detects the names of the statistics for the given network.
     * IPv4 subnet use "-addresses" suffix, IPv6 subnet use "-nas" suffix.
     * Shared networks use "-nas" suffix for both IPv4 and IPv6 types.
     * IPv6 prefix delegations use "-pds" suffix.
     *
     * @returns The statistics names for the given lease type.
     */
    private getStatisticNames(network: { stats?: object }): {
        totalName: string
        assignedName: string
        declinedName: string
    } {
        if (this.isPD) {
            return {
                totalName: 'total-pds',
                assignedName: 'assigned-pds',
                declinedName: 'declined-pds',
            }
        }

        // The shared networks use the `NA` suffix for both IPv4 and IPv6 types.
        const naNames = {
            totalName: 'total-nas',
            assignedName: 'assigned-nas',
            declinedName: 'declined-nas',
        }

        if (network.stats == null || Object.keys(network.stats).length == 0) {
            // Empty statistics.
            return naNames
        }

        if (Object.keys(network.stats).some((k) => k.endsWith('-nas'))) {
            // NAs statistics.
            return naNames
        }

        return {
            totalName: 'total-addresses',
            assignedName: 'assigned-addresses',
            declinedName: 'declined-addresses',
        }
    }
}
