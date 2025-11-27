import { Component, Input } from '@angular/core'
import { Pool } from '../backend'
import { UtilizationBarComponent } from '../utilization-bar/utilization-bar.component'

/**
 * A component displaying an address pool.
 */
@Component({
    selector: 'app-address-pool-bar',
    templateUrl: './address-pool-bar.component.html',
    styleUrls: ['./address-pool-bar.component.sass'],
    imports: [UtilizationBarComponent],
})
export class AddressPoolBarComponent {
    /**
     * Address pool.
     */
    @Input() pool: Pool
}
