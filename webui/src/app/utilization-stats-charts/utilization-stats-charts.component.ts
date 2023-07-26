import { Component, Input } from '@angular/core'
import { LocalSubnet, SharedNetwork, Subnet } from '../backend'
import { hasAddressPools, hasPrefixPools } from '../subnets'
import { IPType } from '../iptype'

/**
 * A component displaying pie charts with address and delegated prefix
 * utilizations for a subnet or shared network.
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
     * Subnet or shared network instance.
     */
    @Input() network: Subnet & SharedNetwork

    /**
     * Checks if the subnet or shared network has IPv6 type.
     *
     * @return true if the subnet or shared network has IPv6 type, false otherwise.
     */
    get isIPv6(): boolean {
        return this.network.universe == IPType.IPv6 || this.network.subnet?.includes(':')
    }

    /**
     * Checks if there are any address pools defined for the subnet or shared network.
     *
     * @return true if subnet or shared network includes configured address pools.
     */
    get hasAddressPools(): boolean {
        return this.network.subnets?.some((s) => hasAddressPools(s)) || hasAddressPools(this.network)
    }

    /**
     * Checks if there are any prefix pools defined for the subnet or shared network.
     *
     * @return true if subnet or shared network includes configured prefix pools.
     */
    get hasPrefixPools(): boolean {
        return this.network.subnets?.some((s) => hasPrefixPools(s)) || hasPrefixPools(this.network)
    }

    /**
     * Returns an array of local subnet instances if the instance is a subnet.
     *
     * @returns local subnet instances or an empty array if the instance is a
     * shared network.
     */
    get localSubnets(): LocalSubnet[] {
        return this.network.localSubnets || []
    }
}
