import { Component, Input } from '@angular/core'
import { Button } from '@openng/optimus-ui/button'
import { RouterLink } from '@angular/router'

/**
 * A component that displays the control buttons for a BIND 9 daemon.
 */
@Component({
    selector: 'app-pdns-daemon-controls',
    imports: [Button, RouterLink],
    templateUrl: './pdns-daemon-controls.component.html',
    styleUrl: './pdns-daemon-controls.component.sass',
})
export class PdnsDaemonControlsComponent {
    /**
     * The ID of the daemon whose controls are being displayed.
     */
    @Input() daemonId: number
}
