import { Component, Input } from '@angular/core'
import { FormControl } from '@angular/forms'
import { AutoCompleteCompleteEvent } from 'primeng/autocomplete'

/**
 * A component providing a form control for specifying an array of values.
 *
 * @tparam T Type of the values to specify.
 */
@Component({
    selector: 'app-array-value-set-form',
    templateUrl: './array-value-set-form.component.html',
    styleUrls: ['./array-value-set-form.component.sass'],
})
export class ArrayValueSetFormComponent<T> {
    /**
     * A form bound to the "AutoComplete" input box holding the list of selected
     * class names.
     */
    @Input({ required: true }) classFormControl: FormControl<T>

    /**
     * An array of suggested options in the AutoComplete component.
     */
    suggestions: string[] = []

    /**
     * Prepares a list of suggested options to be displayed in PrimeNG AutoComplete input component.
     * @param event AutoComplete event received
     */
    prepareSuggestions(event: AutoCompleteCompleteEvent) {
        const query = event.query.trim()
        if (!query) {
            // Do not let empty strings.
            this.suggestions = []
            return
        }

        const suggestions = []
        this.suggestions = [query, ...suggestions]
    }
}
