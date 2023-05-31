import { Component, Input } from '@angular/core'

/**
 * A component displaying an address pool.
 */
@Component({
    selector: 'app-address-pool-bar',
    templateUrl: './address-pool-bar.component.html',
    styleUrls: ['./address-pool-bar.component.sass'],
})
export class AddressPoolBarComponent {
    /**
     * Address pool as text.
     */
    @Input() pool: string
}
