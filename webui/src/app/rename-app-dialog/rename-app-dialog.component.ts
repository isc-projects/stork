import { Component, OnInit, Input, OnChanges, SimpleChanges, Output, EventEmitter } from '@angular/core'

/**
 * Component providing a dialog box to rename an app.
 *
 * This component provides a dialog box with a single input to rename
 * an app. The component validates the new app name during typing.
 * It verifies that the app name is unique and it doesn't reference
 * a non-existing machine address. For example, if the app name
 * "kea@machine3" references "machine3", the machine with the address
 * "machine3" must exist in the system. The list of existing apps and
 * machines is provided as input to the component. The "Rename" button
 * is enabled only when the name is valid. The "Cancel" button is always
 * enabled.
 */
@Component({
    selector: 'app-rename-app-dialog',
    templateUrl: './rename-app-dialog.component.html',
    styleUrls: ['./rename-app-dialog.component.sass'],
})
export class RenameAppDialogComponent implements OnInit, OnChanges {
    /**
     * App id of the app being renamed.
     */
    @Input() appId: number

    /**
     * Current app name in the input box.
     */
    @Input() appName = ''

    /**
     * A map holding apps' names as keys and ids as values.
     */
    @Input() existingApps = new Map<string, number>()

    /**
     * A set holding machines' addresses.
     */
    @Input() existingMachines = new Set<string>()

    /**
     * Indicates if the dialog box is visible.
     */
    @Input() visible = false

    /**
     * Events emitter triggered when rename button is pressed.
     */
    @Output() submitted: EventEmitter<string> = new EventEmitter()

    /**
     * Events emitter triggered when dialog box is hidden.
     */
    @Output() hidden = new EventEmitter()

    /**
     * Holds app name as it was before editing.
     */
    private _originalAppName = ''

    /**
     * Boolean flag indicating if the current name is valid.
     */
    invalid = false

    /**
     * Holds the current error displayed for an invalid name.
     */
    errorText: string

    /**
     * No-op constructor.
     */
    constructor() {}

    /**
     * Lifecycle hook initializing the component.
     *
     * It saves the original app name.
     */
    ngOnInit(): void {
        this._originalAppName = this.appName
    }

    /**
     * Lifecycle hook triggered when values bound to the component change.
     *
     * This hook reacts to the change of an app id. If it changes, it indicates
     * that the component is now used to edit another app's name, and therefore
     * the new original name must be remembered.
     *
     * @param changes collection of old and new values.
     */
    ngOnChanges(changes: SimpleChanges) {
        if (changes.hasOwnProperty('appId') && changes.appId.currentValue !== changes.appId.previousValue) {
            this._originalAppName = this.appName
        }
    }

    /**
     * Event handler triggered during editing the app name.
     *
     * Special cases of pressing an Enter or Escape buttons are handled
     * by this function. The former causes the form submission. The
     * latter cancels the dialog box.
     *
     * @param event triggered key-up event holding pressed key name.
     */
    handleKeyUp(event) {
        switch (event.key) {
            case 'Enter': {
                this.save()
                break
            }
            case 'Escape': {
                this.cancel()
                break
            }
            default: {
                this.validateName()
            }
        }
    }

    /**
     * Event handler triggered when dialog box gets hidden.
     */
    handleOnHide() {
        this.hidden.emit()
    }

    /**
     * Cancel the dialog box.
     *
     * The dialog box is closed, original app name is restored and the
     * errors are cleared. This function does not emit any events. An
     * event informing about closing the dialog box is emitted when the
     * onHide event is triggered.
     */
    cancel() {
        this.visible = false
        this.appName = this._originalAppName
        this.clearError()
    }

    /**
     * Save new app name.
     *
     * The dialog box is closed and an event indicating that the rename
     * button was pressed is triggered. The event holds the new app name.
     */
    save() {
        this.visible = false
        this.submitted.emit(this.appName.trim())
    }

    /**
     * Validate current app name.
     *
     * This function sets "invalid" and "errorText" class members. The
     * app name is considered invalid if the current app name duplicates
     * existing app's name, references a non-existing machine or only
     * consists of the whitespaces.
     */
    private validateName() {
        // Setting existing apps is optional. If they are not set, skip
        // the checks.
        if (this.existingApps.size !== 0) {
            // Check if there is an app with this name already but with
            // a different app id.
            const existingAppId = this.existingApps.get(this.appName)
            if (existingAppId && existingAppId !== this.appId) {
                this.signalError('An app with this name already exists.')
                return
            }
        }
        const regexpMatch = this.appName.match(/^\s*([^@]*)(@+)([^@%]*)(%\S+)*\s*$/)
        if (regexpMatch) {
            // If there is the @ character, the machine name becomes mandatory.
            if (regexpMatch[3].trim().length === 0) {
                this.signalError('The @ character must be followed by a machine address or name.')
                return
            }
            // Setting existing machines is optional. if they are not set,
            // skip the checks.
            if (this.existingMachines.size !== 0 && regexpMatch[2].length === 1 && regexpMatch[3].length > 0) {
                // Check if the name references an existing machine.
                if (!this.existingMachines.has(regexpMatch[3])) {
                    // The referenced machine does not exist. Raise an error.
                    this.signalError('Machine ' + regexpMatch[3] + ' does not exist.')
                    return
                }
            }
            // The app name before @ character(s) must not be empty.
            if (regexpMatch[1].length === 0) {
                this.signalError('An app name preceding the @ character must not be empty.')
                return
            }
        }
        // Ensure that the app name does not consist of whitespaces only.
        if (this.appName.trim().length === 0) {
            this.signalError('An app name must not be empty.')
            return
        }

        // The app name looks correct.
        this.clearError()
    }

    /**
     * Mark the app name invalid.
     *
     * @param errorText error to be displayed in the dialog box.
     */
    private signalError(errorText) {
        this.invalid = true
        this.errorText = errorText
    }

    /**
     * Mark the app name valid.
     */
    private clearError() {
        this.invalid = false
        this.errorText = ''
    }
}
