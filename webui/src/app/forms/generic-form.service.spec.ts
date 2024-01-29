import { TestBed } from '@angular/core/testing'
import { UntypedFormBuilder } from '@angular/forms'
import { DhcpOptionFieldFormGroup, DhcpOptionFieldType } from './dhcp-option-field'

import { GenericFormService } from './generic-form.service'

describe('GenericFormService', () => {
    let service: GenericFormService
    let formBuilder: UntypedFormBuilder = new UntypedFormBuilder()

    beforeEach(() => {
        TestBed.configureTestingModule({})
        service = TestBed.inject(GenericFormService)
        formBuilder.array([
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
                    new DhcpOptionFieldFormGroup(DhcpOptionFieldType.Binary, {
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
