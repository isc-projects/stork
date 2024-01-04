import {
    AbstractControl,
    UntypedFormArray,
    UntypedFormBuilder,
    UntypedFormControl,
    UntypedFormGroup,
} from '@angular/forms'

/**
 * Base class for services used to convert data from and to rective forms.
 */
export class FormProcessor {
    /**
     * Form builder instance used to create the reactive forms.
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
     * Performs deep copy of the control, form group or form array.
     *
     * I copies all controls with their validators. Controls belonging
     * to form groups or arrays are copied recursively.
     *
     * This function implementation is derived from the following article:
     * https://newbedev.com/deep-copy-of-angular-reactive-form
     *
     * @param control top-level control to be copied.
     * @returns copied control instance.
     */
    public cloneControl<T extends AbstractControl>(control: T): T {
        let newControl: T

        if (control instanceof UntypedFormGroup) {
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
            // If the value is an array we need to perform a deep copy explicitly.
            let clonedValue = Array.isArray(control.value) ? [...control.value] : control.value
            newControl = new UntypedFormControl(clonedValue, control.validator, control.asyncValidator) as any
        } else {
            throw new Error('Error: unexpected control value')
        }

        if (control.disabled) {
            newControl.disable({ emitEvent: false })
        }
        if (control.touched) {
            newControl.markAsTouched()
        }

        return newControl
    }
}
