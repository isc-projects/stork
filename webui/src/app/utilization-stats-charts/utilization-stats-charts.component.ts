import { Component, Input } from '@angular/core'
import { Subnet } from '../backend'
import { hasAddressPools, hasPrefixPools } from '../subnets'

/**
 * A component displaying pie charts with address and delegated prefix
 * utilizations for a subnet.
 *
 * It displays only total statistics when there is only one server associated
 * with the subnets. If there are more servers, it displays the total statistics
 * and the individual (local) statistics for every server.
 */
@Component({
    selector: 'app-utilization-stats-charts',
    templateUrl: './utilization-stats-charts.component.html',
    styleUrls: ['./utilization-stats-charts.component.sass'],
})
export class UtilizationStatsChartsComponent {
    /**
     * Subnet instance with local subnet instances.
     */
    @Input() subnet: Subnet

    /**
     * Checks if the subnet has IPv6 type.
     *
     * @return true if the subnet has IPv6 type, false otherwise.
     */
    get isIPv6(): boolean {
        return this.subnet.subnet.includes(':')
    }

    /**
     * Checks if there are any address pools defined for the subnet.
     *
     * @return true if subnet includes configured address pools.
     */
    get hasAddressPools(): boolean {
        return hasAddressPools(this.subnet)
    }

    /**
     * Checks if there are any prefix pools defined for the subnet.
     *
     * @return true if subnet includes configured prefix pools.
     */
    get hasPrefixPools(): boolean {
        return hasPrefixPools(this.subnet)
    }
}
