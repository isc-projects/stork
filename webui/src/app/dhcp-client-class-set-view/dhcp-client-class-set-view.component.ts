import { Component, Input } from '@angular/core'
import { NgFor, NgIf } from '@angular/common'
import { Chip } from 'primeng/chip'

@Component({
    selector: 'app-dhcp-client-class-set-view',
    templateUrl: './dhcp-client-class-set-view.component.html',
    styleUrls: ['./dhcp-client-class-set-view.component.sass'],
    imports: [NgFor, Chip, NgIf],
})
export class DhcpClientClassSetViewComponent {
    @Input()
    clientClasses: Array<string> = []

    constructor() {}
}
