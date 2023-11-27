import { Component, Input } from '@angular/core'
import { FormControl } from '@angular/forms'

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
     * A form bound to the "chips" input box holding the list of selected
     * class names.
     */
    @Input() classFormControl: FormControl<T>
}
