import { Component, Input, OnInit, ViewChild } from '@angular/core'
import { UntypedFormControl } from '@angular/forms'
import { v4 as uuidv4 } from 'uuid'
import { OverlayPanel } from 'primeng/overlaypanel'
import { SelectableClientClass } from '../forms/selectable-client-class'

/**
 * A component providing a "chips" input box to specify client classes
 * for a host reservation.
 *
 * The client classes can be typed directly in the input box or they
 * can be selected from a sorted list of classes specified as component's
 * parameter.
 */
@Component({
    selector: 'app-dhcp-client-class-set-form',
    templateUrl: './dhcp-client-class-set-form.component.html',
    styleUrls: ['./dhcp-client-class-set-form.component.sass'],
})
export class DhcpClientClassSetFormComponent implements OnInit {
    /**
     * Reference to the overlay panel holding a list of classes.
     */
    @ViewChild('op')
    classSelectionPanel: OverlayPanel

    /**
     * A form bound to the "chips" input box holding the list of selected
     * class names.
     */
    @Input() classFormControl: UntypedFormControl

    /**
     * Specifies whether the component should show a floating placeholder
     * displaying an advisory information.
     */
    @Input() floatingPlaceholder: boolean = true

    /**
     * Generated input box identifier.
     */
    inputId: string

    /**
     * A sorted list of classes that can be selected in the overlay.
     */
    sortedClientClasses: SelectableClientClass[] = []

    /**
     * An array of the selected class names in the overlay panel.
     */
    selectedClientClasses: string[] = []

    /**
     * Constructor.
     */
    constructor() {}

    /**
     * A component lifecycle hook executed when the component is initialized.
     *
     * It sorts the list of client classes specified as an input.
     */
    ngOnInit(): void {
        this.inputId = uuidv4()
    }

    /**
     * Sorts and sets client classes displayed in the overlay panel.
     *
     * @param clientClasses unordered list of client classes.
     */
    @Input()
    set clientClasses(clientClasses: SelectableClientClass[]) {
        if (!clientClasses) {
            this.sortedClientClasses = []
            return
        }
        this.sortedClientClasses = clientClasses
        this.sortedClientClasses.sort((c1, c2) => {
            return c1.name.localeCompare(c2.name)
        })
    }

    /**
     * Checks if the given class is in the "chips" input box.
     *
     * This function is called to determine whether the checkbox for the given
     * class should be disabled in the overlay panel. It is not possible to
     * remove a class from the input box by deselecting it in the overlay
     * panel.
     *
     * @param clientClass client class name
     * @returns true if the class is present in the input box, false otherwise.
     */
    isUsed(clientClass: string): boolean {
        const value = this.classFormControl.value as Array<string>
        return !!value && value.includes(clientClass)
    }

    /**
     * Inserts a list of client classes selected in the overlay panel.
     *
     * It iterates over the list of selected classes. For each class, it
     * checks if it is already present in the input box. If the class is
     * not present, it is added to the input box. Finally, it clears the
     * list of selected classes and hides the overlay panel.
     */
    mergeSelected(): void {
        let value = (this.classFormControl.value as Array<string>) || []
        for (let selectedClass of this.selectedClientClasses) {
            if (value.indexOf(selectedClass) < 0) {
                value.push(selectedClass)
            }
        }
        this.classFormControl.patchValue(value)
        this.selectedClientClasses = []
        this.classSelectionPanel.hide()
    }

    /**
     * Cancels selecting the classes from the overlay panel.
     *
     * It clears the selected classes list and hides the panel. No classes
     * are inserted into the input box.
     */
    cancelSelected(): void {
        this.selectedClientClasses = []
        this.classSelectionPanel.hide()
    }

    /**
     * Populates the list of selected classes in the overlay panel.
     *
     * It matches the class names present in the input box with the sorted
     * list of classes. It inserts a class into the selected class list when
     * it is present in the input box. It effectively checks these classes on
     * the checkbox list.
     */
    private _fillSelected(): void {
        const value = this.classFormControl.value as Array<string>
        if (!value || value.length === 0) {
            return
        }
        let selectedClasses: string[] = []
        for (let clientClass of value) {
            if (
                this.sortedClientClasses.some((c) => c.name === clientClass) &&
                selectedClasses.indexOf(clientClass) < 0
            ) {
                selectedClasses.push(clientClass)
            }
        }
        this.selectedClientClasses = selectedClasses
    }

    /**
     * Populates selected classes and shows the overlay panel.
     *
     * @param event event resulting from clicking the button to show the
     * overlay panel.
     */
    showClassSelectionPanel(event): void {
        this._fillSelected()
        this.classSelectionPanel.toggle(event)
    }
}
