import { FormArray } from '@angular/forms'
import { createDefaultDhcpOptionFormGroup } from './dhcp-option-form'

describe('DhcpOptionForm', () => {
    it('should create a default option form group', () => {
        const fg = createDefaultDhcpOptionFormGroup()
        expect(fg.contains('optionCode')).toBeTrue()
        expect(fg.contains('optionFields')).toBeTrue()
        expect(fg.contains('suboptions')).toBeTrue()

        expect(fg.get('optionCode').value).toBe(null)
        expect((fg.get('optionFields') as FormArray).length).toBe(0)
        expect((fg.get('suboptions') as FormArray).length).toBe(0)
    })
})
