import { Component, Input } from '@angular/core'
import { Subnet } from '../backend'

import { datetimeToLocal } from '../utils'

/**
 * Component that presents subnet as a bar with a sub-bar that shows utilizations in %.
 * It also shows details in a tooltip.
 */
@Component({
    selector: 'app-subnet-bar',
    templateUrl: './subnet-bar.component.html',
    styleUrls: ['./subnet-bar.component.sass'],
})
export class SubnetBarComponent {
    /**
     * Internal subnet instance.
     */
    _subnet: Subnet

    /**
     * Tooltip content.
     */
    tooltip = ''

    /**
     * Sets the subnet. It generates also the tooltip content.
     */
    @Input()
    set subnet(subnet: Subnet) {
        this._subnet = subnet

        if (this._subnet.stats) {
            const stats = this._subnet.stats
            const lines = []
            if (this.addrUtilization > 100 || this.pdUtilization > 100) {
                lines.push('Warning! Utilization is greater than 100%. Data is unreliable.')
                lines.push(
                    'This problem is caused by a Kea limitation - addresses/NAs/PDs in out-of-pool host reservations are reported as assigned but excluded from the total counters.'
                )
                lines.push(
                    'Please manually check that the pool has free addresses and make sure that Kea and Stork are up-to-date.'
                )
                lines.push('')
            }

            if (this._subnet.subnet.includes('.')) {
                // DHCPv4 stats
                lines.push(`Utilization: ${this.addrUtilization}%`)
                lines.push('Total: ' + stats['total-addresses'].toLocaleString('en-US'))
                lines.push('Assigned: ' + stats['assigned-addresses'].toLocaleString('en-US'))
                lines.push('Declined: ' + stats['declined-addresses'].toLocaleString('en-US'))
            } else {
                // DHCPv6 stats
                // IPv6 addresses
                lines.push(`Utilization NAs: ${this.addrUtilization}%`)
                lines.push(`Utilization PDs: ${this.pdUtilization}%`)

                if (stats['total-nas'] !== undefined) {
                    let total = stats['total-nas']
                    if (total === -1) {
                        total = Number.MAX_SAFE_INTEGER
                    }
                    lines.push('Total NAs: ' + total.toLocaleString('en-US'))
                }
                if (stats['assigned-nas'] !== undefined) {
                    lines.push('Assigned NAs: ' + stats['assigned-nas'].toLocaleString('en-US'))
                }
                if (stats['declined-nas'] !== undefined) {
                    lines.push('Declined NAs: ' + stats['declined-nas'].toLocaleString('en-US'))
                }
                // PDs
                if (stats['total-pds'] !== undefined) {
                    let total = stats['total-pds']
                    if (total === -1) {
                        total = Number.MAX_SAFE_INTEGER
                    }
                    lines.push('Total PDs: ' + total.toLocaleString('en-US'))
                }
                if (stats['assigned-pds'] !== undefined) {
                    lines.push('Assigned PDs: ' + stats['assigned-pds'].toLocaleString('en-US'))
                }
            }
            lines.push('Collected at: ' + (datetimeToLocal(this._subnet.statsCollectedAt) || 'never'))
            this.tooltip = lines.join('<br>')
        } else {
            this.tooltip = 'No statistics yet'
        }
    }

    /**
     * Returns the subnet.
     */
    get subnet() {
        return this._subnet
    }

    /**
     * Returns the address utilization. It guaranties that the number will be
     * returned.
     */
    get addrUtilization() {
        return this.subnet.addrUtilization ?? 0
    }

    /**
     * Returns the delegated prefix utilization. It guaranties that the number
     * will be returned.
     */
    get pdUtilization() {
        return this.subnet.pdUtilization ?? 0
    }

    /**
     * Returns true if the subnet is IPv6.
     */
    get isIPv6() {
        return this.subnet.subnet.includes(':')
    }
}
