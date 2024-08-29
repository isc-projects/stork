import { Component, EventEmitter, Input, Output } from '@angular/core'
import { NamedCascadedParameters } from '../cascaded-parameters-board/cascaded-parameters-board.component'

/**
 * A component displaying global Kea configuration.
 *
 * It displays an edit button to start editing the configuration.
 */
@Component({
    selector: 'app-kea-global-configuration-view',
    templateUrl: './kea-global-configuration-view.component.html',
    styleUrl: './kea-global-configuration-view.component.sass',
})
export class KeaGlobalConfigurationViewComponent {
    /**
     * Holds fetched configuration.
     */
    @Input() dhcpParameters: Array<NamedCascadedParameters<Object>> = []

    /**
     * An event emitter notifying a parent that user has clicked the
     * Edit button to modify the global parameters.
     */
    @Output() editBegin = new EventEmitter<void>()

    /**
     * A list of parameters not presented in this view but fetched from
     * the server in the configuration.
     */
    excludedParameters: Array<string> = [
        'clientClasses',
        'configControl',
        'hostsDatabases',
        'hooksLibraries',
        'loggers',
        'optionData',
        'optionDef',
        'optionsHash',
        'reservations',
        'subnet4',
        'subnet6',
        'sharedNetworks',
    ]

    /**
     * A callback invoked when the edit button was clicked.
     *
     * It emits an event to the parent, so the parent can open a form.
     */
    onEditBegin(): void {
        this.editBegin.emit()
    }
}
