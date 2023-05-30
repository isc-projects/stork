import { Component, Input } from '@angular/core'
import { Subnet } from '../backend'

@Component({
    selector: 'app-subnet-tab',
    templateUrl: './subnet-tab.component.html',
    styleUrls: ['./subnet-tab.component.sass'],
})
export class SubnetTabComponent {
    /**
     * Link to Grafana Dashboard.
     */
    @Input() grafanaUrl: string

    /**
     * Subnet data.
     */
    @Input() subnet: Subnet
}
