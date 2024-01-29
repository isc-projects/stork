import { Injectable } from '@angular/core'
import { AbstractControl, UntypedFormGroup, UntypedFormArray } from '@angular/forms'
import { FormProcessor } from './form-processor'

/**
 * A service providing generic form manipulation functions.
 */
@Injectable({
    providedIn: 'root',
})
export class GenericFormService extends FormProcessor {
    /**
     * Constructor.
     *
     * Creates form builder instance.
     */
    constructor() {
        super()
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
    public setValuesFromFormGroup(formGroup: UntypedFormGroup, values: Object): void {
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
