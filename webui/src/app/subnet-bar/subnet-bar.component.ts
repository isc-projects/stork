import { Component, Input } from '@angular/core'
import { Subnet } from '../backend'

/**
 * Component that presents subnet as a bar with a sub-bar that shows utilizations in %.
 * It also shows details in a tooltip.
 */
@Component({
    selector: 'app-subnet-bar',
    standalone: false,
    templateUrl: './subnet-bar.component.html',
    styleUrls: ['./subnet-bar.component.sass'],
})
export class SubnetBarComponent {
    /**
     * Sets the subnet.
     */
    @Input()
    subnet: Subnet

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
