import { TestBed } from '@angular/core/testing'
import { UntypedFormBuilder, UntypedFormArray } from '@angular/forms'
import { DhcpOptionFieldFormGroup, DhcpOptionFieldType } from './dhcp-option-field'

import { GenericFormService } from './generic-form.service'

describe('GenericFormService', () => {
    let service: GenericFormService
    let formBuilder: UntypedFormBuilder = new UntypedFormBuilder()
    let formArray: UntypedFormArray

    beforeEach(() => {
        TestBed.configureTestingModule({})
        service = TestBed.inject(GenericFormService)
        formArray = formBuilder.array([
            formBuilder.group({
                alwaysSend: formBuilder.control(true),
                optionCode: formBuilder.control(1024),
                optionFields: formBuilder.array([
                    new DhcpOptionFieldFormGroup(DhcpOptionFieldType.IPv6Prefix, {
                        prefix: formBuilder.control('3000::'),
                        prefixLength: formBuilder.control(64),
                    }),
                    new DhcpOptionFieldFormGroup(DhcpOptionFieldType.Psid, {
                        psid: formBuilder.control(12),
                        psidLength: formBuilder.control(8),
                    }),
                    new DhcpOptionFieldFormGroup(DhcpOptionFieldType.HexBytes, {
                        control: formBuilder.control('01:02:03'),
                    }),
                    new DhcpOptionFieldFormGroup(DhcpOptionFieldType.String, {
                        control: formBuilder.control('foobar'),
                    }),
                ]),
            }),
            formBuilder.group({
                alwaysSend: formBuilder.control(false),
                optionCode: formBuilder.control(2024),
                optionFields: formBuilder.array([
                    new DhcpOptionFieldFormGroup(DhcpOptionFieldType.Uint8, {
                        control: formBuilder.control(101),
                    }),
                    new DhcpOptionFieldFormGroup(DhcpOptionFieldType.Uint16, {
                        control: formBuilder.control(16523),
                    }),
                ]),
            }),
            formBuilder.group({
                alwaysSend: formBuilder.control(true),
                optionCode: formBuilder.control(3087),
                suboptions: formBuilder.array([
                    formBuilder.group({
                        alwaysSend: formBuilder.control(false),
                        optionCode: formBuilder.control(1),
                        optionFields: formBuilder.array([
                            new DhcpOptionFieldFormGroup(DhcpOptionFieldType.Uint16, {
                                control: formBuilder.control(1111),
                            }),
                        ]),
                    }),
                    formBuilder.group({
                        alwaysSend: formBuilder.control(false),
                        optionCode: formBuilder.control(0),
                        optionFields: formBuilder.array([
                            new DhcpOptionFieldFormGroup(DhcpOptionFieldType.Uint32, {
                                control: formBuilder.control(2222),
                            }),
                        ]),
                        suboptions: formBuilder.array([
                            formBuilder.group({
                                alwaysSend: formBuilder.control(false),
                                optionCode: formBuilder.control(5),
                                optionFields: formBuilder.array([
                                    new DhcpOptionFieldFormGroup(DhcpOptionFieldType.String, {
                                        control: formBuilder.control('foo'),
                                    }),
                                ]),
                            }),
                        ]),
                    }),
                ]),
            }),
        ])
    })

    it('should be created', () => {
        expect(service).toBeTruthy()
    })

    it('copies a complex form control with multiple nesting levels', () => {
        let clonedArray = service.cloneControl(formArray)

        expect(clonedArray).toBeTruthy()
        expect(clonedArray.length).toBe(3)

        // Option 1024.
        expect(clonedArray.at(0).get('alwaysSend')).toBeTruthy()
        expect(clonedArray.at(0).get('optionCode')).toBeTruthy()
        expect(clonedArray.at(0).get('optionFields')).toBeTruthy()

        expect(clonedArray.at(0).get('alwaysSend').value).toBeTrue()
        expect(clonedArray.at(0).get('optionCode').value).toBe(1024)

        // Option 1024 fields.
        expect(clonedArray.at(0).get('optionFields')).toBeInstanceOf(UntypedFormArray)
        let fields = clonedArray.at(0).get('optionFields') as UntypedFormArray
        expect(fields.controls.length).toBe(4)

        // Option 1024 field 0.
        expect(fields.at(0)).toBeInstanceOf(DhcpOptionFieldFormGroup)
        expect((fields.at(0) as DhcpOptionFieldFormGroup).data.fieldType).toBe(DhcpOptionFieldType.IPv6Prefix)
        expect(fields.at(0).get('prefix')).toBeTruthy()
        expect(fields.at(0).get('prefixLength')).toBeTruthy()
        expect(fields.at(0).get('prefix').value).toBe('3000::')
        expect(fields.at(0).get('prefixLength').value).toBe(64)

        // Option 1024 field 1.
        expect(fields.at(1)).toBeInstanceOf(DhcpOptionFieldFormGroup)
        expect((fields.at(1) as DhcpOptionFieldFormGroup).data.fieldType).toBe(DhcpOptionFieldType.Psid)
        expect(fields.at(1).get('psid')).toBeTruthy()
        expect(fields.at(1).get('psidLength')).toBeTruthy()
        expect(fields.at(1).get('psid').value).toBe(12)
        expect(fields.at(1).get('psidLength').value).toBe(8)

        // Option 1024 field 2.
        expect(fields.at(2)).toBeInstanceOf(DhcpOptionFieldFormGroup)
        expect((fields.at(2) as DhcpOptionFieldFormGroup).data.fieldType).toBe(DhcpOptionFieldType.HexBytes)
        expect(fields.at(2).get('control')).toBeTruthy()
        expect(fields.at(2).get('control').value).toBe('01:02:03')

        // Option 1024 field 3.
        expect(fields.at(3)).toBeInstanceOf(DhcpOptionFieldFormGroup)
        expect((fields.at(3) as DhcpOptionFieldFormGroup).data.fieldType).toBe(DhcpOptionFieldType.String)
        expect(fields.at(3).get('control')).toBeTruthy()
        expect(fields.at(3).get('control').value).toBe('foobar')

        // Option 2024.
        expect(clonedArray.at(1).get('alwaysSend')).toBeTruthy()
        expect(clonedArray.at(1).get('optionCode')).toBeTruthy()
        expect(clonedArray.at(1).get('optionFields')).toBeTruthy()

        expect(clonedArray.at(1).get('alwaysSend').value).toBeFalse()
        expect(clonedArray.at(1).get('optionCode').value).toBe(2024)

        // Option 2024 fields.
        expect(clonedArray.at(1).get('optionFields')).toBeInstanceOf(UntypedFormArray)
        fields = clonedArray.at(1).get('optionFields') as UntypedFormArray
        expect(fields.controls.length).toBe(2)

        // Option 2024 field 0.
        expect(fields.at(0)).toBeInstanceOf(DhcpOptionFieldFormGroup)
        expect((fields.at(0) as DhcpOptionFieldFormGroup).data.fieldType).toBe(DhcpOptionFieldType.Uint8)
        expect(fields.at(0).get('control')).toBeTruthy()
        expect(fields.at(0).get('control').value).toBe(101)

        // Option 2024 field 1.
        expect(fields.at(1)).toBeInstanceOf(DhcpOptionFieldFormGroup)
        expect((fields.at(1) as DhcpOptionFieldFormGroup).data.fieldType).toBe(DhcpOptionFieldType.Uint16)
        expect(fields.at(1).get('control')).toBeTruthy()
        expect(fields.at(1).get('control').value).toBe(16523)

        // Option 3087.
        expect(clonedArray.at(2).get('alwaysSend')).toBeTruthy()
        expect(clonedArray.at(2).get('optionCode')).toBeTruthy()
        expect(clonedArray.at(2).get('suboptions')).toBeTruthy()

        expect(clonedArray.at(2).get('alwaysSend').value).toBeTrue()
        expect(clonedArray.at(2).get('optionCode').value).toBe(3087)

        // Option 3087 suboptions.
        expect(clonedArray.at(2).get('suboptions')).toBeInstanceOf(UntypedFormArray)
        expect((clonedArray.at(2).get('suboptions') as UntypedFormArray).controls.length).toBe(2)

        // Option 3087.1.
        expect(clonedArray.at(2).get('suboptions.0.alwaysSend')).toBeTruthy()
        expect(clonedArray.at(2).get('suboptions.0.optionCode')).toBeTruthy()
        expect(clonedArray.at(2).get('suboptions.0.optionFields')).toBeTruthy()

        // Option 3087.1 field 0.
        fields = clonedArray.at(2).get('suboptions.0.optionFields') as UntypedFormArray
        expect(fields.controls.length).toBe(1)
        expect(fields.at(0)).toBeInstanceOf(DhcpOptionFieldFormGroup)
        expect((fields.at(0) as DhcpOptionFieldFormGroup).data.fieldType).toBe(DhcpOptionFieldType.Uint16)
        expect(fields.at(0).get('control')).toBeTruthy()
        expect(fields.at(0).get('control').value).toBe(1111)

        // Option 3087.0.
        expect(clonedArray.at(2).get('suboptions.1.alwaysSend')).toBeTruthy()
        expect(clonedArray.at(2).get('suboptions.1.optionCode')).toBeTruthy()
        expect(clonedArray.at(2).get('suboptions.1.optionFields')).toBeTruthy()

        expect(clonedArray.at(2).get('suboptions.1.alwaysSend').value).toBeFalse()
        expect(clonedArray.at(2).get('suboptions.1.optionCode').value).toBe(0)

        // Option 3087.0 field 0.
        fields = clonedArray.at(2).get('suboptions.1.optionFields') as UntypedFormArray
        expect(fields.controls.length).toBe(1)
        expect(fields.at(0)).toBeInstanceOf(DhcpOptionFieldFormGroup)
        expect((fields.at(0) as DhcpOptionFieldFormGroup).data.fieldType).toBe(DhcpOptionFieldType.Uint32)
        expect(fields.at(0).get('control')).toBeTruthy()
        expect(fields.at(0).get('control').value).toBe(2222)
    })

    it('should set existing form group values', () => {
        let formGroup = formBuilder.group({
            nextServer: ['192.0.2.1'],
            serverHostname: ['myhost'],
            bootFileName: ['/tmp/boot1'],
        })
        let source = {
            nextServer: '192.1.1.1',
            serverHostname: 'another',
            accessPoint: '',
        }
        service.setFormGroupValues(formGroup, source)
        expect(formGroup.get('nextServer')).toBeTruthy()
        expect(formGroup.get('serverHostname')).toBeTruthy()
        expect(formGroup.get('bootFileName')).toBeTruthy()
        expect(formGroup.get('accessPoint')).toBeFalsy()
        expect(formGroup.get('nextServer').value).toBe('192.1.1.1')
        expect(formGroup.get('serverHostname').value).toBe('another')
        expect(formGroup.get('bootFileName').value).toBeFalsy()
    })

    it('should set values from form group', () => {
        let formGroup = formBuilder.group({
            nextServer: ['192.0.2.1'],
            serverHostname: ['myhost'],
            bootFileName: ['/tmp/boot1'],
        })
        let values: any = {}
        service.setValuesFromFormGroup(formGroup, values)
        expect(values.hasOwnProperty('nextServer'))
        expect(values.hasOwnProperty('serverHostname'))
        expect(values.hasOwnProperty('bootFileName'))
        expect(values['nextServer']).toEqual('192.0.2.1')
        expect(values['serverHostname']).toEqual('myhost')
        expect(values['bootFileName']).toEqual('/tmp/boot1')
    })

    it('should properly set form array controls', () => {
        let formArray = formBuilder.array([])
        let control1 = formBuilder.control([])
        let control2 = formBuilder.control([])

        service.setArrayControl(0, formArray, control1)
        expect(formArray.length).toBe(1)
        expect(formArray.at(0)).toBe(control1)

        service.setArrayControl(0, formArray, control2)
        expect(formArray.length).toBe(1)
        expect(formArray.at(0)).toBe(control2)

        service.setArrayControl(10, formArray, control1)
        expect(formArray.length).toBe(2)
        expect(formArray.at(0)).toBe(control2)
        expect(formArray.at(1)).toBe(control1)

        service.setArrayControl(2, formArray, control2)
        expect(formArray.length).toBe(3)
        expect(formArray.at(0)).toBe(control2)
        expect(formArray.at(1)).toBe(control1)
        expect(formArray.at(2)).toBe(control2)
    })
})
