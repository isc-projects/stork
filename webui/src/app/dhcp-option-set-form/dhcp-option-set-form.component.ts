import { Component, EventEmitter, Input, Output } from '@angular/core'
import { UntypedFormArray, UntypedFormBuilder } from '@angular/forms'

/**
 * A component aggregating multiple forms for editing DHCP option information.
 *
 * It maintains a form array for DHCP options. For each array item it displays
 * the DhcpOptionFormComponent. It also exposes a button to add more DHCP
 * options.
 */
@Component({
    selector: 'app-dhcp-option-set-form',
    templateUrl: './dhcp-option-set-form.component.html',
    styleUrls: ['./dhcp-option-set-form.component.sass'],
})
export class DhcpOptionSetFormComponent {
    /**
     * Sets the options universe: DHCPv4 or DHCPv6.
     */
    @Input() v6 = false

    /**
     * An array holding form groups for each embedded DHCP option form.
     */
    @Input() formArray: UntypedFormArray

    /**
     * Nesting level of edited options in this component.
     * It is set to 0 for top-level options. It is set to 1 for
     * sub-options belonging to the top-level options, etc.
     */
    @Input() nestLevel = 0

    /**
     * Option space the options belong to.
     *
     * It is used to find the definitions of the selected options.
     */
    @Input() optionSpace = null

    /**
     * An event emitter sending an event when user adds a new option.
     */
    @Output() optionAdd = new EventEmitter<void>()

    /**
     * Constructor.
     *
     * @param _formBuilder a form builder instance used in this component.
     */
    constructor(private _formBuilder: UntypedFormBuilder) {}

    /**
     * Notifies a parent component to create new option form group.
     */
    notifyOptionAdded(): void {
        this.optionAdd.emit()
    }

    /**
     * Deletes an option from the array on child component's request.
     *
     * @param index index of an option to be removed.
     */
    onOptionDelete(index: number): void {
        // Removing an array element means removing an instance of the
        // app-dhcp-option-form. It comprises the components that mark
        // the form as 'touched' during destroy. Thus, we mark the array
        // touched on our own to work around the problem of "expression
        // has changed after it was checked". Marking it as touched
        // guarantees that the touched state won't change during the component
        // redraw. We have no other control over it because it stems from
        // the primeng implementation.
        this.formArray.markAsTouched()
        this.formArray.removeAt(index)
    }
}
