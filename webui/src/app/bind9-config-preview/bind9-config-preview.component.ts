import { Component, EventEmitter, Input, Output, ViewChild } from '@angular/core'
import { CommonModule } from '@angular/common'
import { Bind9FormattedConfig } from '../backend'
import { CheckboxChangeEvent, CheckboxModule } from 'primeng/checkbox'
import { FormsModule } from '@angular/forms'
import { ProgressSpinnerModule } from 'primeng/progressspinner'
import { DialogModule } from 'primeng/dialog'
import { Bind9ConfigViewFeederComponent } from '../bind9-config-view-feeder/bind9-config-view-feeder.component'
import { ButtonModule } from 'primeng/button'
import { TooltipModule } from 'primeng/tooltip'

/**
 * A component that displays BIND 9 configuration file in a dialog.
 * It contains a checkbox to toggle between displaying partial and
 * full configuration. When this checkbox is clicked, the component
 * sends a new request to the backend to get the suitable configuration.
 */
@Component({
    selector: 'app-bind9-config-preview',
    standalone: true,
    imports: [
        Bind9ConfigViewFeederComponent,
        ButtonModule,
        CommonModule,
        CheckboxModule,
        DialogModule,
        FormsModule,
        ProgressSpinnerModule,
        TooltipModule,
    ],
    templateUrl: './bind9-config-preview.component.html',
    styleUrl: './bind9-config-preview.component.sass',
})
export class Bind9ConfigPreviewComponent {
    /**
     * A reference to the child component that sends requests to the
     * server to get the configuration according to the state of the
     * checkbox.
     */
    @ViewChild(Bind9ConfigViewFeederComponent) bind9ConfigViewFeeder: Bind9ConfigViewFeederComponent

    /**
     * The ID of the daemon whose configuration is being displayed.
     */
    @Input({ required: true }) daemonId: number

    /**
     * The type of the file to be displayed. The server uses this
     * selection to determine which file contents to return.
     */
    @Input({ required: true }) fileType: 'config' | 'rndc-key'

    /**
     * Indicates whether or not the dialog is visible.
     */
    @Input() visible = false

    /**
     * An event emitter that emits the new value of the visible property.
     */
    @Output() visibleChange = new EventEmitter<boolean>()

    /**
     * The configuration to be displayed.
     */
    config: Bind9FormattedConfig | null = null

    /**
     * Indicates if the full configuration should be displayed.
     */
    showFullConfig: boolean = false

    /**
     * Updates the configuration to be displayed.
     *
     * @param config is the new configuration to be displayed.
     */
    handleConfigChange(config: Bind9FormattedConfig): void {
        this.config = config
    }

    /**
     * An event handler invoked when the checkbox is clicked.
     *
     * It toggles between displaying the full and partial configuration.
     * It calls the child component to fetch the configuration according
     * to the new state of the checkbox.
     *
     * @param event is an event object containing the new boolean value of
     * the checkbox.
     */
    handleFullConfigToggle(event: CheckboxChangeEvent): void {
        this.showFullConfig = event.checked as boolean
        this.bind9ConfigViewFeeder.updateConfig(this.showFullConfig)
    }

    /**
     * Refreshes the configuration from the server on demand.
     */
    handleConfigRefresh(): void {
        this.bind9ConfigViewFeeder.updateConfig(this.showFullConfig)
    }

    /**
     * Sets the visibility of the dialog.
     *
     * It emits an event with the new value of the visible property.
     *
     * @param visible is the new value of the visible property.
     */
    handleVisibleChange(visible: boolean): void {
        this.visible = visible
        this.visibleChange.emit(visible)
        if (!visible) {
            this.bind9ConfigViewFeeder.cancelUpdateConfig()
        }
    }
}
