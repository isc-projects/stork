import { Component, Input } from '@angular/core'
import { PdnsDaemon } from '../backend'
import { PlaceholderPipe } from '../pipes/placeholder.pipe'
import { DurationPipe } from '../pipes/duration.pipe'

@Component({
    selector: 'app-pdns-daemon',
    templateUrl: './pdns-daemon.component.html',
    styleUrl: './pdns-daemon.component.sass',
    imports: [PlaceholderPipe, DurationPipe],
})
export class PdnsDaemonComponent {
    /**
     * PowerDNS daemon information.
     */
    @Input() daemon: PdnsDaemon
}
