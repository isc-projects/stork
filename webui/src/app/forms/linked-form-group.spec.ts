import { UntypedFormBuilder } from '@angular/forms'
import { LinkedFormGroup } from './linked-form-group'

describe('LinkedFormGroup', () => {
    it('should create a form group with additional data', () => {
        const fb = new UntypedFormBuilder()
        const formGroup = new LinkedFormGroup<number>(523, {
            control1: fb.control(''),
            control2: fb.control(''),
        })
        expect(formGroup).toBeTruthy()
        expect(formGroup.data).toBe(523)
        expect(formGroup.contains('control1')).toBeTrue()
        expect(formGroup.contains('control2')).toBeTrue()
    })
})
