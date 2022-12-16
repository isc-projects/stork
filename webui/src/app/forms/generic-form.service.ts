import { Injectable } from '@angular/core'
import {
    UntypedFormBuilder,
    AbstractControl,
    UntypedFormGroup,
    UntypedFormArray,
    UntypedFormControl,
} from '@angular/forms'
import { DhcpOptionFieldFormGroup } from './dhcp-option-field'

/**
 * A service providing generic form manipulation functions.
 */
@Injectable({
    providedIn: 'root',
})
export class GenericFormService {
    /**
     * Form builder instance used by the service to create the reactive forms.
     */
    _formBuilder: UntypedFormBuilder

    /**
     * Constructor.
     *
     * Creates form builder instance.
     */
    constructor() {
        this._formBuilder = new UntypedFormBuilder()
    }

    /**
     * Performs deep copy of the form array holding DHCP options or its fragment.
     *
     * I copies all controls, including DhcpOptionFieldFormGroup, with their
     * validators. Controls belonging to forms or arrays are copied recursively.
     *
     * This function implementation is derived from the following article:
     * https://newbedev.com/deep-copy-of-angular-reactive-form
     *
     * @param control top-level control to be copied.
     * @returns copied control instance.
     */
    public cloneControl<T extends AbstractControl>(control: T): T {
        let newControl: T

        if (control instanceof DhcpOptionFieldFormGroup) {
            const formGroup = new DhcpOptionFieldFormGroup(
                (control as DhcpOptionFieldFormGroup).data.fieldType,
                {},
                control.validator,
                control.asyncValidator
            )

            const controls = control.controls

            Object.keys(controls).forEach((key) => {
                formGroup.addControl(key, this.cloneControl(controls[key]))
            })

            newControl = formGroup as any
        } else if (control instanceof UntypedFormGroup) {
            const formGroup = new UntypedFormGroup({}, control.validator, control.asyncValidator)
            const controls = control.controls

            Object.keys(controls).forEach((key) => {
                formGroup.addControl(key, this.cloneControl(controls[key]))
            })

            newControl = formGroup as any
        } else if (control instanceof UntypedFormArray) {
            const formArray = new UntypedFormArray([], control.validator, control.asyncValidator)

            control.controls.forEach((formControl) => formArray.push(this.cloneControl(formControl)))

            newControl = formArray as any
        } else if (control instanceof UntypedFormControl) {
            newControl = new UntypedFormControl(control.value, control.validator, control.asyncValidator) as any
        } else {
            throw new Error('Error: unexpected control value')
        }

        if (control.disabled) {
            newControl.disable({ emitEvent: false })
        }

        return newControl
    }

    /**
     * Convenience function mapping field values from the specified object to the
     * form group values.
     *
     * For each field in a form group it tries to find a corresponding value in the
     * object. If the value exists, it is set as the form group value.
     *
     * @param formGroup form group whose values are set.
     * @param values an object holding fields with basic type values to be copied
     * to the form group.
     */
    public setFormGroupValues(formGroup: UntypedFormGroup, values: Object): void {
        for (let key of Object.keys(formGroup.controls)) {
            if (values.hasOwnProperty(key) && values[key]) {
                formGroup.get(key).setValue(values[key])
            } else {
                formGroup.get(key).setValue(null)
            }
        }
    }

    /**
     * Convenience function copying values from the form group to the provided
     * object.
     *
     * The copied values are stored in the destination under the same keys as
     * in the form group.
     *
     * @param formGroup form group holding the user specified data.
     * @param values object to which the values are copied.
     */
    public setValuesFromFormGroup<T>(formGroup: UntypedFormGroup, values: T): void {
        for (let key of Object.keys(formGroup.controls)) {
            if (formGroup.get(key).value) {
                values[key] = formGroup.get(key).value
            }
        }
    }

    /**
     * Convenience function setting control in the form group array, at the
     * specific index.
     *
     * If the index is out of bounds, the control is appended to the array.
     *
     * @param index form array index.
     * @param formArray form array where the control should be set.
     * @param control control to be set or appended to the array.
     */
    public setArrayControl(index: number, formArray: UntypedFormArray, control: AbstractControl): void {
        if (formArray.length > index) {
            formArray.setControl(index, control)
            return
        }
        formArray.push(control)
    }
}
