import {
    AsyncValidatorFn,
    UntypedFormArray,
    UntypedFormBuilder,
    UntypedFormControl,
    UntypedFormGroup,
    ValidationErrors,
    ValidatorFn,
} from '@angular/forms'
import { Observable } from 'rxjs'
import { FormProcessor } from './form-processor'

describe('FormProcessor', () => {
    let processor: FormProcessor = new FormProcessor()
    let formBuilder: UntypedFormBuilder = new UntypedFormBuilder()

    it('copies a complex form control with multiple nesting levels', () => {
        let validator1: ValidatorFn = (): ValidationErrors | null => {
            return null
        }
        let validator2: AsyncValidatorFn = ():
            | Promise<ValidationErrors | null>
            | Observable<ValidationErrors | null> => {
            return new Promise(null)
        }
        let formArray = formBuilder.array([
            formBuilder.group({
                foo: formBuilder.control('abc', validator1, validator2),
                bar: formBuilder.group(
                    {
                        baz: formBuilder.array(
                            [formBuilder.control('ccc'), formBuilder.control('xyz')],
                            validator1,
                            validator2
                        ),
                        zab: formBuilder.control(['foo', 'bar']),
                    },
                    { validators: validator1, asyncValidators: validator2 }
                ),
            }),
            formBuilder.control('aaa'),
        ])
        formArray.get('0.foo').markAsTouched()

        let clonedArray = processor.cloneControl(formArray)

        expect(clonedArray).toBeTruthy()
        expect(clonedArray.length).toBe(2)

        expect(clonedArray.at(0)).toBeInstanceOf(UntypedFormGroup)
        expect(clonedArray.at(1)).toBeInstanceOf(UntypedFormControl)

        expect(clonedArray.at(0).get('foo')).toBeTruthy()
        expect(clonedArray.at(0).get('foo')).toBeInstanceOf(UntypedFormControl)
        expect(clonedArray.at(0).get('foo').value).toBe('abc')
        expect(clonedArray.at(0).get('foo').hasValidator(validator1)).toBeTrue()
        expect(clonedArray.at(0).get('foo').hasAsyncValidator(validator2)).toBeTrue()
        expect(clonedArray.at(0).get('foo').touched).toBeTrue()

        expect(clonedArray.at(0).get('bar')).toBeTruthy()
        expect(clonedArray.at(0).get('bar')).toBeInstanceOf(UntypedFormGroup)
        expect(clonedArray.at(0).get('bar').hasValidator(validator1)).toBeTrue()
        expect(clonedArray.at(0).get('bar').hasAsyncValidator(validator2)).toBeTrue()

        expect(clonedArray.at(0).get('bar.baz')).toBeTruthy()
        expect(clonedArray.at(0).get('bar.baz')).toBeInstanceOf(UntypedFormArray)
        expect(clonedArray.at(0).get('bar.baz').hasValidator(validator1)).toBeTrue()
        expect(clonedArray.at(0).get('bar.baz').hasAsyncValidator(validator2)).toBeTrue()

        let baz = clonedArray.at(0).get('bar.baz') as UntypedFormArray
        expect(baz.length).toBe(2)
        expect(baz.at(0)).toBeInstanceOf(UntypedFormControl)
        expect(baz.at(0).value).toBe('ccc')
        expect(baz.at(1)).toBeInstanceOf(UntypedFormControl)
        expect(baz.at(1).value).toBe('xyz')

        let zab = clonedArray.at(0).get('bar.zab') as UntypedFormControl
        expect(zab.value).toEqual(formArray.get('0.bar.zab').value)
        // Ensure that the array value has been deeply copied.
        expect(zab.value).not.toBe(formArray.get('0.bar.zab').value)

        expect(clonedArray.at(1).value).toBe('aaa')
    })
})
