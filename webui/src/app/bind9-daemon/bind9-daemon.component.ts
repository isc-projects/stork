import { Component, Input } from '@angular/core'
import { Bind9Daemon, Bind9DaemonView, DNSZoneType } from '../backend'

/**
 * Component for displaying information about a BIND9 daemon.
 */
@Component({
    selector: 'app-bind9-daemon',
    templateUrl: './bind9-daemon.component.html',
    styleUrl: './bind9-daemon.component.sass',
})
export class Bind9DaemonComponent {
    /**
     * ID of the parent application.
     */
    @Input() appId: number

    /**
     * BIND9 daemon information.
     */
    @Input() daemon: Bind9Daemon

    /**
     * All zone types except builtin type.
     */
    configuredZoneTypes: string[] = Object.values(DNSZoneType).filter((t) => t !== DNSZoneType.Builtin)

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
