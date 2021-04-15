import { Component, Input, OnInit } from '@angular/core'

/**
 * Component presenting reservation details for a selected host.
 */
@Component({
    selector: 'app-host-tab',
    templateUrl: './host-tab.component.html',
    styleUrls: ['./host-tab.component.sass'],
})
export class HostTabComponent implements OnInit {
    /**
     * Input structure containing host information to be displayed.
     */
    @Input() host: any

    constructor() {}

    ngOnInit(): void {}
}
