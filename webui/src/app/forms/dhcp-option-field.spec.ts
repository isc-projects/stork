import { UntypedFormBuilder } from '@angular/forms'
import { DhcpOptionFieldFormGroup, DhcpOptionFieldType } from './dhcp-option-field'

describe('DhcpOptionField', () => {
    it('should create a form group for dhcp option field', () => {
        const fb = new UntypedFormBuilder()
        const formGroup = new DhcpOptionFieldFormGroup(DhcpOptionFieldType.String, {
            control1: fb.control(''),
            control2: fb.control(''),
        })
        expect(formGroup).toBeTruthy()
        expect(formGroup.data).toBeTruthy()
        expect(formGroup.data.fieldType).toBe(DhcpOptionFieldType.String)
        expect(formGroup.data.getInputId(0).length).toBeGreaterThan(0)
        expect(formGroup.data.getInputId(1).length).toBeGreaterThan(0)
        expect(formGroup.contains('control1')).toBeTrue()
        expect(formGroup.contains('control2')).toBeTrue()
    })
})
