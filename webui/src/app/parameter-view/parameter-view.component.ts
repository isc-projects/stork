import { Component, Input } from '@angular/core'

/**
 * A component displaying a parameter in the cascaded parameters board.
 *
 * If a parameter has a basic type it is displayed as is. If it is an
 * object or an array, its values are displayed as a list.
 */
@Component({
    selector: 'app-parameter-view',
    templateUrl: './parameter-view.component.html',
    styleUrl: './parameter-view.component.sass',
})
export class ParameterViewComponent {
    /**
     * Input parameter to be displayed.
     */
    @Input() parameter: string | number | boolean | Array<any> | Object | null

    /**
     * Checks if the specified parameter is an object.
     */
    get isParameterObject(): boolean {
        return !Array.isArray(this.parameter) && typeof this.parameter === 'object'
    }

    /**
     * Checks if object parameter has any values.
     *
     * It is used to determine if the placeholder should be displayed
     * instead of the key/value pairs.
     */
    get parameterObjectHasValues(): boolean {
        return this.parameter && this.isParameterObject && Object.keys(this.parameterAsRecord).length > 0
    }

    /**
     * Casts the parameter to a record.
     */
    get parameterAsRecord(): Record<string, any> {
        return this.parameter as Record<string, any>
    }

    /**
     * Checks if the parameter is an array.
     */
    get isParameterArray(): boolean {
        return Array.isArray(this.parameter)
    }

    /**
     * Casts the parameter to an array.
     */
    get parameterAsArray(): Array<any> {
        return this.parameter as Array<any>
    }

    /**
     * Checks if the parameter has a basic type.
     */
    get isParameterBasicType(): boolean {
        return !this.isParameterObject && !this.isParameterArray
    }

    /**
     * Casts the parameter to any.
     */
    get parameterAsAny(): any {
        return this.parameter
    }
}
