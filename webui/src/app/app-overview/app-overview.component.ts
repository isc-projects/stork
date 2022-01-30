import { Component, Input } from '@angular/core'

/**
 * A component that displays app overview.
 *
 * It comprises the information about the app and machine access points.
 */
@Component({
    selector: 'app-app-overview',
    templateUrl: './app-overview.component.html',
    styleUrls: ['./app-overview.component.sass'],
})
export class AppOverviewComponent {
    /**
     * Pointer to the structure holding the app information.
     */
    @Input() app: any = null
}
