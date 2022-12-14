import { Component, Input } from '@angular/core'

// Expected and supported value types for the below component.
type ValueType = number | string | bigint | null

/**
 * Display a given value in human-readable form using metric prefixes. It
 * generates a tooltip with an exact value visible on hover.
 */
@Component({
    selector: 'app-human-count',
    templateUrl: './human-count.component.html',
    styleUrls: ['./human-count.component.sass'],
})
export class HumanCountComponent {
    /**
     * Stores the value.
     */
    private _value: ValueType

    /**
     * Setter for a value. It accepts any kind of value. The strings are
     * converted to numbers (if possible).
     */
    @Input() set value(value: ValueType) {
        if (typeof value === 'string') {
            try {
                value = BigInt(value)
            } catch {
                // Invalid conversion. Keep it as is.
            }
        }

        this._value = value
    }

    /**
     * Returns a value.
     */
    get value(): ValueType {
        return this._value
    }

    /**
     * Indicates if the value is set.
     */
    get hasValue(): boolean {
        return this.value != null
    }

    /**
     * Indicates if the value is set but it isn't a valid number.
     */
    get hasInvalidValue(): boolean {
        if (!this.hasValue) {
            return false
        }

        const type = typeof this._value
        if (type === 'number') {
            return isNaN(this._value as number)
        }
        return type !== 'bigint'
    }

    /**
     * Indicates if the value is a valid number.
     */
    get hasValidValue(): boolean {
        return this.hasValue && !this.hasInvalidValue
    }
}
