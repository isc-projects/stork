import { Component, input, model } from '@angular/core'
import { FormsModule } from '@angular/forms'
import { Checkbox } from 'primeng/checkbox'
import { CheckIcon, TimesIcon } from 'primeng/icons'

/**
 * This component is an HTML input type=checkbox implementation which allows to hold and visualize 3 states:
 * - true (checkbox is filled with a tick icon)
 * - false (checkbox is filled with an x-cross icon)
 * - null (checkbox is empty - not filled).
 */
@Component({
    selector: 'app-tri-state-checkbox',
    standalone: true,
    imports: [FormsModule, Checkbox, CheckIcon, TimesIcon],
    templateUrl: './tri-state-checkbox.component.html',
    styleUrl: './tri-state-checkbox.component.sass',
})
export class TriStateCheckboxComponent {
    /**
     * Holds input value. Also emits the value whenever value changes.
     */
    value = model<boolean | null>(null)

    /**
     * ID of the html input.
     */
    inputID = input<string | undefined>(undefined)

    /**
     * Flag controlling input disabled state. Defaults to false.
     */
    disabled = input<boolean>(false)

    /**
     * Label of the checkbox.
     */
    label = input<string | undefined>(undefined)

    /**
     * Toggles input value in a chain true->false->null->true->...
     * @private
     */
    private toggleValue() {
        if (this.value() === true) {
            this.value.set(false)
        } else if (this.value() === false) {
            this.value.set(null)
        } else {
            this.value.set(true)
        }
    }

    /**
     * Callback called when the checkbox is clicked.
     * @param event
     */
    onClick(event: MouseEvent) {
        this.toggleValue()
        event.preventDefault()
    }

    /**
     * Callback called when the keyboard event happened while the focus was on the checkbox input.
     * @param event
     */
    onKeyDown(event: KeyboardEvent) {
        if (event.key === 'Enter' || event.key === 'Space') {
            this.toggleValue()
            event.preventDefault()
        }
    }
}
