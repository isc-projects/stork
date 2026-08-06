import { Component, input } from '@angular/core'
import { Lease } from '../backend'
import { Fieldset } from 'primeng/fieldset'
import { IdentifierComponent } from '../identifier/identifier.component'
import { JsonTreeRootComponent } from '../json-tree-root/json-tree-root.component'
import { LocaltimePipe } from '../pipes/localtime.pipe'

@Component({
    selector: 'app-lease-detail-box',
    imports: [Fieldset, IdentifierComponent, JsonTreeRootComponent, LocaltimePipe],
    templateUrl: './lease-detail-box.component.html',
    styleUrl: './lease-detail-box.component.sass',
})
export class LeaseDetailBoxComponent {
    /**
     * The lease for which to display details.
     */
    lease = input<Lease>()
}
