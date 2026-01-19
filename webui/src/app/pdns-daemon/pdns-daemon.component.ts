import { Component, Input } from '@angular/core'
import { PdnsDaemon } from '../backend'
import { PlaceholderPipe } from '../pipes/placeholder.pipe'
import { DurationPipe } from '../pipes/duration.pipe'
import { EventsPanelComponent } from '../events-panel/events-panel.component'
import { AccessPointsComponent } from '../access-points/access-points.component'

@Component({
    selector: 'app-pdns-daemon',
    templateUrl: './pdns-daemon.component.html',
    styleUrl: './pdns-daemon.component.sass',
    imports: [PlaceholderPipe, DurationPipe, EventsPanelComponent, AccessPointsComponent],
})
export class PdnsDaemonComponent {
    /**
     * PowerDNS daemon information.
     */
    @Input() daemon: PdnsDaemon
}
