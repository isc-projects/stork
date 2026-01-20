import { Component, Input } from '@angular/core'
import { Bind9Daemon, Bind9DaemonView } from '../backend'
import { VersionStatusComponent } from '../version-status/version-status.component'
import { RouterLink } from '@angular/router'
import { NgFor, NgIf } from '@angular/common'
import { Tooltip } from 'primeng/tooltip'
import { LocaltimePipe } from '../pipes/localtime.pipe'
import { PlaceholderPipe } from '../pipes/placeholder.pipe'
import { DurationPipe } from '../pipes/duration.pipe'
import { AccessPointsComponent } from '../access-points/access-points.component'
import { EventsPanelComponent } from '../events-panel/events-panel.component'

/**
 * Component for displaying information about a BIND9 daemon.
 */
@Component({
    selector: 'app-bind9-daemon',
    templateUrl: './bind9-daemon.component.html',
    styleUrl: './bind9-daemon.component.sass',
    imports: [
        VersionStatusComponent,
        RouterLink,
        NgFor,
        NgIf,
        Tooltip,
        LocaltimePipe,
        PlaceholderPipe,
        DurationPipe,
        AccessPointsComponent,
        EventsPanelComponent,
    ],
})
export class Bind9DaemonComponent {
    /**
     * BIND9 daemon information.
     */
    @Input() daemon: Bind9Daemon

    /**
     * Get cache effectiveness based on stats for a BIND9 view.
     *
     * @param view is a data structure holding the information about the BIND9 view.
     * @return A percentage is returned as floored int.
     */
    getQueryUtilization(view: Bind9DaemonView) {
        let utilization = 0.0
        if (!view.queryHitRatio) {
            return utilization
        }
        utilization = 100 * view.queryHitRatio
        return Math.floor(utilization)
    }
}
