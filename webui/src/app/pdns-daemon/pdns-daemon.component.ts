import { Component, Input } from '@angular/core'
import { PdnsDaemon } from '../backend'

@Component({
    selector: 'app-pdns-daemon',
    templateUrl: './pdns-daemon.component.html',
    styleUrl: './pdns-daemon.component.sass',
})
export class PdnsDaemonComponent {
    /**
     * PowerDNS daemon information.
     */
    @Input() daemon: PdnsDaemon
}
