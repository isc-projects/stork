import {
    AbstractControl,
    AbstractControlOptions,
    AsyncValidatorFn,
    FormArray,
    FormControl,
    ValidatorFn,
} from '@angular/forms'
import { LinkedFormGroup } from './linked-form-group'

/**
 * Type of the form group holding a shared parameter.
 */
interface SharedParameterForm {
    /**
     * A control indicating if the parameter is unlocked for editing
     * different values for different servers.
     */
    unlocked: FormControl

    /**
     * Controls for the parameter values for different servers.
     */
    values: FormArray
}

/**
 * Returns an inner array type or its own type.
 *
 * This is specifically useful in {@link EditableParameterSpec} when the
 * generic type is an array. In that case, we want to extract the inner
 * array type and use this type for the possible input values.
 */
type Unarray<T> = T extends Array<infer U> ? U : T

/**
 * A shared parameter descriptor in the form group.
 *
 * @typeParam type of the parameter values.
 */
interface EditableParameterSpec<T> {
    /**
     * Parameter type used to select appropriate control type.
     */
    type: string

    /**
     * A set of values valid for the parameter.
     */
    values?: Unarray<T>[]

    /**
     * Indicates if the parameter carries an array of values.
     */
    isArray?: boolean

    /**
     * Minimum allowed number value.
     */
    min?: number

    /**
     * Maximum allowed number value.
     */
    max?: number

    /**
     * A number of fractional digits for the value.
     */
    fractionDigits?: number

    /**
     * An error text to be displayed when the specified value in the
     * form is invalid.
     */
    invalidText?: string

    /**
     * Indicates if the value is mandatory.
     */
    required?: boolean

    /**
     * A value to be set when the control in the form is cleared.
     */
    clearValue?: T

    /**
     * An earliest daemon version supporting this parameter.
     *
     * The parameter is not configurable for the Kea versions earlier than
     * indicated by this field.
     */
    versionLowerBound?: string

    /**
     * The first daemon version that no longer supports this parameter.
     *
     * The parameter is not configurable for the Kea versions equal or later
     * than indicated by this field.
     */
    versionUpperBound?: string
}

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
export class SharedParameterFormGroup<
    TDataType,
    TControl extends { [K in keyof TControl]: AbstractControl<any> } = any,
> extends LinkedFormGroup<EditableParameterSpec<TDataType>, SharedParameterForm> {
    /**
     * Constructor.
     *
     * @param data custom data.
     * @param controls form controls belonging to the form group.
     * @param validatorOrOpts validators or control options.
     * @param asyncValidator asynchronous validators.
     */
    constructor(
        data: EditableParameterSpec<TDataType>,
        controls: FormControl<TDataType>[],
        validatorOrOpts?: ValidatorFn | AbstractControlOptions | ValidatorFn[],
        asyncValidator?: AsyncValidatorFn | AsyncValidatorFn[]
    ) {
        let fgControls = {
            unlocked: new FormControl({
                value: !!(
                    controls?.length > 1 &&
                    controls.some((c) => JSON.stringify(c.value) != JSON.stringify(controls[0].value))
                ),
                disabled: controls.length <= 1,
            }),
            values: new FormArray(controls),
        }
        super(data, fgControls, validatorOrOpts, asyncValidator)
        this.data = data
    }
}
