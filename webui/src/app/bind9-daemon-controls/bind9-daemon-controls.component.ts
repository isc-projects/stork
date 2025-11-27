import { Component, Input } from '@angular/core'
import { ButtonModule } from 'primeng/button'
import { DialogModule } from 'primeng/dialog'
import { Bind9ConfigPreviewComponent } from '../bind9-config-preview/bind9-config-preview.component'
import { CommonModule } from '@angular/common'

/**
 * A component that displays the control buttons for a BIND 9 daemon.
 */
@Component({
    selector: 'app-bind9-daemon-controls',
    imports: [ButtonModule, Bind9ConfigPreviewComponent, CommonModule, DialogModule],
    templateUrl: './bind9-daemon-controls.component.html',
    styleUrl: './bind9-daemon-controls.component.sass',
})
export class Bind9DaemonControlsComponent {
    /**
     * The ID of the daemon whose controls are being displayed.
     */
    @Input() daemonId: number

    /**
     * Holds the state of the dialogs (shown or hidden).
     */
    dialogVisible: Record<'config' | 'rndc-key', boolean> = {
        config: false,
        'rndc-key': false,
    }

    /**
     * Activates or deactivates the dialog of the selected type.
     *
     * @param type is the type of the dialog to activate or deactivate.
     * @param visible indicates if the dialog should be shown or hidden.
     */
    setDialogVisibility(type: 'config' | 'rndc-key', visible: boolean): void {
        this.dialogVisible[type] = visible
    }
}
