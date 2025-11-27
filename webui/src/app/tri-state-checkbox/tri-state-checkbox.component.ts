import { Component, input, model, signal, viewChild } from '@angular/core'
import { ControlValueAccessor, FormsModule, NG_VALUE_ACCESSOR } from '@angular/forms'
import { Checkbox } from 'primeng/checkbox'
import { CheckIcon, TimesIcon } from 'primeng/icons'

/**
 * This component is an HTML input type=checkbox implementation which allows to hold and visualize 3 states:
 * - true (checkbox is filled with a tick icon)
 * - false (checkbox is filled with an x-cross icon)
 * - null/unset (checkbox is empty - not filled).
 */
@Component({
    selector: 'app-tri-state-checkbox',
    imports: [FormsModule, Checkbox, CheckIcon, TimesIcon],
    templateUrl: './tri-state-checkbox.component.html',
    styleUrl: './tri-state-checkbox.component.sass',
    providers: [
        {
            provide: NG_VALUE_ACCESSOR,
            multi: true,
            useExisting: TriStateCheckboxComponent,
        },
    ],
})
export class TriStateCheckboxComponent implements ControlValueAccessor {
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
    disabled = model<boolean>(false)

    /**
     * Label of the checkbox.
     */
    label = input<string | undefined>(undefined)

    /**
     * Name of the checkbox input.
     */
    name = input<string | undefined>(undefined)

    /**
     * Boolean flag keeping form input touched state.
     */
    touched = false

    /**
     * Checkbox view child.
     */
    checkbox = viewChild(Checkbox)

    /**
     * Internal signal keeping disabled state.
     */
    _disabled = signal<boolean>(false)

    /**
     * Toggles input value in a chain true->false->null->true->...
     * @private
     */
    private toggleValue() {
        if (this.isDisabled()) {
            return
        }

        this.markAsTouched()
        if (this.value() === true) {
            this.value.set(false)
        } else if (this.value() === false) {
            this.value.set(null)
        } else {
            this.value.set(true)
        }

        this.onChange(this.value())
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

    /**
     * Sets input touched state if it wasn't touched before.
     */
    markAsTouched() {
        if (!this.touched) {
            this.onTouched()
            this.touched = true
        }
    }

    /**
     * ControlValueAccessor implementation.
     * Writes input value.
     * @param value boolean or null value
     */
    writeValue(value: boolean | null): void {
        this.value.set(value)
    }

    /**
     * Callback called when input value changes.
     * @param _value changed value
     */
    onChange = (_value: boolean | null): void => {}

    /**
     * ControlValueAccessor implementation.
     * Registers onChange callback function.
     * @param onChange callback function
     */
    registerOnChange(onChange: (value: boolean | null) => void): void {
        this.onChange = onChange
    }

    /**
     * ControlValueAccessor implementation.
     * Callback called when the form input is touched.
     */
    onTouched = (): void => {}

    /**
     * Registers onTouched callback function.
     * @param onTouched callback function
     */
    registerOnTouched(onTouched: () => void): void {
        this.onTouched = onTouched
    }

    /**
     * ControlValueAccessor implementation.
     * Sets input disabled state.
     * @param isDisabled boolean flag
     */
    setDisabledState?(isDisabled: boolean): void {
        this.disabled.set(isDisabled)
    }

    /**
     * Marks the component as disabled.
     */
    markAsDisabled() {
        if (!this._disabled()) {
            this._disabled.set(true)
        }
    }

    /**
     * Checks if the component is disabled. Apart from checking disabled input flag, it also checks internal PrimeNG
     * checkbox component p-disabled class.
     */
    isDisabled(): boolean {
        const isDisabled =
            this.disabled() ||
            this._disabled() ||
            this.checkbox().el.nativeElement.childNodes?.[0]?.classList?.contains('p-disabled')
        if (isDisabled) {
            this.markAsDisabled()
        }

        return isDisabled
    }
}
