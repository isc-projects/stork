import { UntypedFormArray } from '@angular/forms'
import { createDefaultDhcpOptionFormGroup } from './dhcp-option-form'
import { IPType } from '../iptype'

describe('DhcpOptionForm', () => {
    it('should create a default option form group', () => {
        const fg = createDefaultDhcpOptionFormGroup(IPType.IPv4)
        expect(fg.contains('alwaysSend')).toBeTrue()
        expect(fg.contains('optionCode')).toBeTrue()
        expect(fg.contains('optionFields')).toBeTrue()
        expect(fg.contains('suboptions')).toBeTrue()

        expect(fg.get('alwaysSend').value).toBeFalse()
        expect(fg.get('optionCode').value).toBe(null)
        expect((fg.get('optionFields') as UntypedFormArray).length).toBe(0)
        expect((fg.get('suboptions') as UntypedFormArray).length).toBe(0)
    })
})
