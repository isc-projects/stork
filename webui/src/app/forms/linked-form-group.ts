import { AbstractControl, AbstractControlOptions, AsyncValidatorFn, FormGroup, ValidatorFn } from '@angular/forms'

/**
 * Extends the FormGroup with custom data of selected type.
 *
 * The FormGroup class is not well suited for the forms with changing
 * set of controls that can't be determined upfront. In that case, it
 * is useful to hold additional information with the form group that,
 * for example, indicates the type of the data, an identifier of the
 * input box, etc. This class derives from the FormGroup (behaves like
 * the FormGroup) and holds such additional custom information.
 *
 * Even though the FormGroup is marked final, deriving from it should be
 * safe in this particular case. The derived class does not call any
 * protected methods and is independent of the base class's API.
 */
export class LinkedFormGroup<
    LinkedData,
    TControl extends { [K in keyof TControl]: AbstractControl<any> } = any,
> extends FormGroup<TControl> {
    /**
     * Custom data associated with the form group.
     */
    data: LinkedData

    /**
     * Constructor.
     *
     * @param data custom data.
     * @param controls form controls belonging to the form group.
     * @param validatorOrOpts validators or control options.
     * @param asyncValidator asynchronous validators.
     */
    constructor(
        data: LinkedData,
        controls: TControl,
        validatorOrOpts?: ValidatorFn | AbstractControlOptions | ValidatorFn[],
        asyncValidator?: AsyncValidatorFn | AsyncValidatorFn[]
    ) {
        super(controls, validatorOrOpts, asyncValidator)
        this.data = data
    }
}
