import { Component, Input, OnInit } from '@angular/core'
import { UntypedFormControl, FormsModule, ReactiveFormsModule } from '@angular/forms'
import { v4 as uuidv4 } from 'uuid'
import { SelectableClientClass } from '../forms/selectable-client-class'
import { AutoCompleteCompleteEvent, AutoComplete } from 'primeng/autocomplete'
import { FloatLabel } from 'primeng/floatlabel'
import { NgTemplateOutlet } from '@angular/common'

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
    imports: [AutoComplete, FormsModule, ReactiveFormsModule, FloatLabel, NgTemplateOutlet],
})
export class DhcpClientClassSetFormComponent implements OnInit {
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
     * A list of classes to be displayed as suggested options in PrimeNG AutoComplete input component.
     */
    classesSuggestions: any[] | undefined

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
     * Prepares a list of classes to be displayed as suggested options in PrimeNG AutoComplete input component.
     * @param event AutoComplete event received
     */
    prepareClasses(event: AutoCompleteCompleteEvent) {
        this.classesSuggestions = [
            ...this.sortedClientClasses.filter((c) => c.name.indexOf(event.query) !== -1).map((c) => c.name),
        ]

        const query = event.query.trim()
        if (query && !this.classesSuggestions.includes(query)) {
            this.classesSuggestions = [query, ...this.classesSuggestions]
        }
    }
}
