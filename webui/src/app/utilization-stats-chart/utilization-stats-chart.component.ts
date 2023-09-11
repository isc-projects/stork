import { Component, Input, OnInit } from '@angular/core'
import { getStatisticValue } from '../subnets'
import { Subnet } from '../backend/model/subnet'
import { SharedNetwork } from '../backend/model/sharedNetwork'
import { clamp } from '../utils'
import { LocalSubnet } from '../backend'

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
export class UtilizationStatsChartComponent implements OnInit {
    /**
     * Optional chart title displayed at the top.
     */
    @Input() title: string

    /**
     * An instance of a subnet or a shared network holding statistics.
     */
    @Input() network: LocalSubnet | Subnet | SharedNetwork

    /**
     * Lease type for which the statistics should be shown.
     */
    @Input() leaseType: 'na' | 'pd'

    /**
     * Pie chart data initialized during the component initialization.
     */
    data: any

    /**
     * Total number of leases fetched from the statistics.
     */
    total: bigint | number

    /**
     * Number of assigned leases fetched from the statistics.
     */
    assigned: bigint | number

    /**
     * Number of used leases.
     *
     * It is calculated by subtracting declined from assigned addresses.
     */
    used: bigint | number

    /**
     * Number of declined leases fetched from the statistics.
     */
    declined: bigint | number

    /**
     * Address or delegated prefix utilization fetched from the statistics.
     */
    utilization: number

    /**
     * A component lifecycle hook invoked on initialization.
     *
     * It prepares the data to be displayed in a chart using the statistics
     * conveyed in a subnet, local subnet or a shared network.
     */
    ngOnInit() {
        if (this.network?.stats) {
            const documentStyle = getComputedStyle(document.documentElement)

            // Fetch the statistics.
            const { totalName, assignedName, declinedName } = this.getStatisticNames()

            this.total = getStatisticValue(this.network, totalName)
            this.assigned = getStatisticValue(this.network, assignedName)
            this.declined = getStatisticValue(this.network, declinedName)
            this.used = this.assigned - this.declined

            if (this.isPD && 'pdUtilization' in this.network) {
                this.utilization = clamp(this.network['pdUtilization'], 0, 100)
            } else if (!this.isPD && 'addrUtilization' in this.network) {
                this.utilization = clamp(this.network['addrUtilization'], 0, 100)
            } else {
                this.utilization = 0
            }

            // Start preparing the dataset for a chart. Each chart has at least two types
            // of data (i.e., free leases and assigned leases).
            let dataset = {
                data: [],
                backgroundColor: [
                    documentStyle.getPropertyValue('--blue-500'),
                    documentStyle.getPropertyValue('--yellow-500'),
                ],
                hoverBackgroundColor: [
                    documentStyle.getPropertyValue('--blue-400'),
                    documentStyle.getPropertyValue('--yellow-400'),
                ],
            }

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
            if (
                !total64 ||
                assigned64 == null ||
                declined64 == null ||
                total64 - assigned64 < 0 ||
                assigned64 - declined64 < 0
            ) {
                dataset.data = [100 - this.utilization, this.utilization]
                this.data = {
                    labels: ['% free', '% used'],
                    datasets: [dataset],
                }
                return
            }

            // The total numbers are correct, so we can present them on the chart.
            dataset.data = [total64 - assigned64, assigned64 - declined64]
            this.data = {
                labels: ['free', 'used'],
                datasets: [dataset],
            }
            // Only addresses can be declined, so we don't include this statistic for
            // prefix delegation.
            if (!this.isPD) {
                this.data.labels.push('declined')
                dataset.data.push(declined64)
                dataset.backgroundColor.push(documentStyle.getPropertyValue('--red-500'))
                dataset.hoverBackgroundColor.push(documentStyle.getPropertyValue('--red-400'))
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
     * Attempts to convert a number to 64-bits.
     *
     * @param stat statistic to be converted to a 64-bit number.
     * @returns converted number or null if the value is too high.
     */
    private clampTo64(stat: bigint | number): number | null {
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
    private getStatisticNames(): { totalName: string; assignedName: string; declinedName: string } {
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

        if (this.network.stats == null || Object.keys(this.network.stats).length == 0) {
            // Empty statistics.
            return naNames
        }

        if (Object.keys(this.network.stats).some((k) => k.endsWith('-nas'))) {
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
