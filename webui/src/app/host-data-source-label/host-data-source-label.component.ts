import { Component, Input } from '@angular/core'

/**
 * This component is used to display the label of the host data source.
 */
@Component({
    selector: 'app-host-data-source-label',
    templateUrl: './host-data-source-label.component.html',
    styleUrls: ['./host-data-source-label.component.sass'],
})
export class HostDataSourceLabelComponent {
    /**
     * The host data source to display.
     */
    @Input()
    dataSource: string
}
