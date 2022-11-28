import { Component, Input } from '@angular/core'

@Component({
    selector: 'app-dhcp-client-class-set-view',
    templateUrl: './dhcp-client-class-set-view.component.html',
    styleUrls: ['./dhcp-client-class-set-view.component.sass'],
})
export class DhcpClientClassSetViewComponent {
    @Input()
    clientClasses: Array<string> = []

    constructor() {}
}
