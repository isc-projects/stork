import { TestBed } from '@angular/core/testing'
import { UntypedFormArray, UntypedFormBuilder } from '@angular/forms'
import { DHCPOption } from '../backend/model/dHCPOption'
import { DhcpOptionSetFormService } from './dhcp-option-set-form.service'
import { DhcpOptionFieldFormGroup, DhcpOptionFieldType } from './dhcp-option-field'
import { IPType } from '../iptype'
import { StorkValidators } from '../validators'

describe('DhcpOptionSetFormService', () => {
    let service: DhcpOptionSetFormService
    let formBuilder: UntypedFormBuilder = new UntypedFormBuilder()
    let formArray: UntypedFormArray

    beforeEach(() => {
        TestBed.configureTestingModule({})
        service = TestBed.inject(DhcpOptionSetFormService)

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
        expect((fields.at(2) as DhcpOptionFieldFormGroup).data.fieldType).toBe(DhcpOptionFieldType.Binary)
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

    it('converts specified DHCP options to REST API format', () => {
        // Extract the options from the form and make sure there are
        // three of them.
        const serialized = service.convertFormToOptions(IPType.IPv4, formArray)
        expect(serialized.length).toBe(3)

        expect(serialized[0].hasOwnProperty('alwaysSend')).toBeTrue()
        expect(serialized[0].hasOwnProperty('code')).toBeTrue()
        expect(serialized[0].hasOwnProperty('encapsulate')).toBeTrue()
        expect(serialized[0].hasOwnProperty('fields')).toBeTrue()
        expect(serialized[0].hasOwnProperty('options')).toBeTrue()
        expect(serialized[0].alwaysSend).toBeTrue()
        expect(serialized[0].code).toBe(1024)
        expect(serialized[0].encapsulate.length).toBe(0)
        // It should have 4 option fields of different types.
        expect(serialized[0].fields.length).toBe(4)
        expect(serialized[0].fields[0].fieldType).toBe(DhcpOptionFieldType.IPv6Prefix)
        expect(serialized[0].fields[0].values.length).toBe(2)
        expect(serialized[0].fields[0].values[0]).toBe('3000::')
        expect(serialized[0].fields[0].values[1]).toBe('64')
        expect(serialized[0].fields[1].fieldType).toBe(DhcpOptionFieldType.Psid)
        expect(serialized[0].fields[1].values[0]).toBe('12')
        expect(serialized[0].fields[1].values[1]).toBe('8')
        expect(serialized[0].fields[1].values.length).toBe(2)
        expect(serialized[0].fields[2].fieldType).toBe(DhcpOptionFieldType.Binary)
        expect(serialized[0].fields[2].values.length).toBe(1)
        expect(serialized[0].fields[2].values[0]).toBe('01:02:03')
        expect(serialized[0].fields[3].fieldType).toBe(DhcpOptionFieldType.String)
        expect(serialized[0].fields[3].values.length).toBe(1)
        expect(serialized[0].fields[3].values[0]).toBe('foobar')

        expect(serialized[1].hasOwnProperty('alwaysSend')).toBeTrue()
        expect(serialized[1].hasOwnProperty('code')).toBeTrue()
        expect(serialized[1].hasOwnProperty('encapsulate')).toBeTrue()
        expect(serialized[1].hasOwnProperty('fields')).toBeTrue()
        expect(serialized[1].hasOwnProperty('options')).toBeTrue()
        expect(serialized[1].alwaysSend).toBeFalse()
        expect(serialized[1].code).toBe(2024)
        expect(serialized[1].encapsulate.length).toBe(0)
        expect(serialized[1].fields.length).toBe(2)
        expect(serialized[1].fields[0].values.length).toBe(1)
        expect(serialized[1].fields[0].values[0]).toBe('101')
        expect(serialized[1].fields[1].values.length).toBe(1)
        expect(serialized[1].fields[1].values[0]).toBe('16523')
        expect(serialized[1].hasOwnProperty('options')).toBeTrue()

        expect(serialized[2].hasOwnProperty('alwaysSend')).toBeTrue()
        expect(serialized[2].hasOwnProperty('code')).toBeTrue()
        expect(serialized[2].hasOwnProperty('encapsulate')).toBeTrue()
        expect(serialized[2].hasOwnProperty('fields')).toBeTrue()
        expect(serialized[2].hasOwnProperty('options')).toBeTrue()
        expect(serialized[2].alwaysSend).toBeTrue()
        expect(serialized[2].code).toBe(3087)
        expect(serialized[2].encapsulate).toBe('option-3087')
        expect(serialized[2].fields.length).toBe(0)
        // The option should contain suboptions.
        expect(serialized[2].options.length).toBe(2)
        expect(serialized[2].options[0].hasOwnProperty('code')).toBeTrue()
        expect(serialized[2].options[0].hasOwnProperty('encapsulate')).toBeTrue()
        expect(serialized[2].options[0].hasOwnProperty('fields')).toBeTrue()
        expect(serialized[2].options[0].hasOwnProperty('options')).toBeTrue()
        expect(serialized[2].options[0].code).toBe(1)
        expect(serialized[2].options[0].encapsulate.length).toBe(0)
        expect(serialized[2].options[0].fields.length).toBe(1)
        expect(serialized[2].options[0].fields[0].fieldType).toBe(DhcpOptionFieldType.Uint16)
        expect(serialized[2].options[0].options.length).toBe(0)
        expect(serialized[2].options[1].hasOwnProperty('code')).toBeTrue()
        expect(serialized[2].options[1].hasOwnProperty('encapsulate')).toBeTrue()
        expect(serialized[2].options[1].hasOwnProperty('fields')).toBeTrue()
        expect(serialized[2].options[1].hasOwnProperty('options')).toBeTrue()
        expect(serialized[2].options[1].code).toBe(0)
        expect(serialized[2].options[1].fields.length).toBe(1)
        expect(serialized[2].options[1].fields[0].fieldType).toBe(DhcpOptionFieldType.Uint32)

        expect(serialized[2].options[1].options.length).toBe(1)
        expect(serialized[2].options[1].options[0].code).toBe(5)
        expect(serialized[2].options[1].options[0].encapsulate.length).toBe(0)
        expect(serialized[2].options[1].options[0].fields.length).toBe(1)
        expect(serialized[2].options[1].options[0].fields[0].fieldType).toBe(DhcpOptionFieldType.String)
    })

    it('throws on too much recursion when converting a form', () => {
        // Add an option with four nesting levels. It should throw because
        // we support first and second level suboptions.
        const formArray = formBuilder.array([
            formBuilder.group({
                alwaysSend: formBuilder.control(false),
                optionCode: formBuilder.control(1024),
                optionFields: formBuilder.array([]),
                suboptions: formBuilder.array([
                    formBuilder.group({
                        alwaysSend: formBuilder.control(false),
                        optionCode: formBuilder.control(1),
                        optionFields: formBuilder.array([]),
                        suboptions: formBuilder.array([
                            formBuilder.group({
                                alwaysSend: formBuilder.control(false),
                                optionCode: formBuilder.control(2),
                                optionFields: formBuilder.array([]),
                                suboptions: formBuilder.array([
                                    formBuilder.group({
                                        alwaysSend: formBuilder.control(false),
                                        optionCode: formBuilder.control(3),
                                        optionFields: formBuilder.array([]),
                                        suboptions: formBuilder.array([]),
                                    }),
                                ]),
                            }),
                        ]),
                    }),
                ]),
            }),
        ])
        expect(() => service.convertFormToOptions(IPType.IPv4, formArray)).toThrow()
    })

    it('throws when there is no option code when converting a form', () => {
        const formArray = formBuilder.array([formBuilder.group({})])
        expect(() => service.convertFormToOptions(IPType.IPv4, formArray)).toThrow()
    })

    it('throws when prefix field lacks prefix control', () => {
        const formArray = formBuilder.array([
            formBuilder.group({
                optionCode: formBuilder.control(3087),
                optionFields: formBuilder.array([
                    new DhcpOptionFieldFormGroup(DhcpOptionFieldType.IPv6Prefix, {
                        prefixLength: formBuilder.control(64),
                    }),
                ]),
            }),
        ])
        expect(() => service.convertFormToOptions(IPType.IPv4, formArray)).toThrow()
    })

    it('throws when prefix field lacks prefix control when converting a form', () => {
        const formArray = formBuilder.array([
            formBuilder.group({
                optionCode: formBuilder.control(3087),
                optionFields: formBuilder.array([
                    new DhcpOptionFieldFormGroup(DhcpOptionFieldType.IPv6Prefix, {
                        prefix: formBuilder.control('3000::'),
                    }),
                ]),
            }),
        ])
        expect(() => service.convertFormToOptions(IPType.IPv4, formArray)).toThrow()
    })

    it('throws when psid field lacks psid control when converting a form', () => {
        const formArray = formBuilder.array([
            formBuilder.group({
                optionCode: formBuilder.control(3087),
                optionFields: formBuilder.array([
                    new DhcpOptionFieldFormGroup(DhcpOptionFieldType.Psid, {
                        psidLength: formBuilder.control(8),
                    }),
                ]),
            }),
        ])
        expect(() => service.convertFormToOptions(IPType.IPv4, formArray)).toThrow()
    })

    it('throws when psid field lacks psid length control when converting a form', () => {
        const formArray = formBuilder.array([
            formBuilder.group({
                optionCode: formBuilder.control(3087),
                optionFields: formBuilder.array([
                    new DhcpOptionFieldFormGroup(DhcpOptionFieldType.Psid, {
                        psid: formBuilder.control(100),
                    }),
                ]),
            }),
        ])
        expect(() => service.convertFormToOptions(IPType.IPv4, formArray)).toThrow()
    })

    it('throws when a single value field lacks control when converting a form', () => {
        const formArray = formBuilder.array([
            formBuilder.group({
                optionCode: formBuilder.control(3087),
                optionFields: formBuilder.array([
                    new DhcpOptionFieldFormGroup(DhcpOptionFieldType.Uint8, {
                        psid: formBuilder.control(100),
                    }),
                ]),
            }),
        ])
        expect(() => service.convertFormToOptions(IPType.IPv4, formArray)).toThrow()
    })

    it('converts received DHCP options from REST API format to a form', () => {
        let options: Array<DHCPOption> = [
            {
                alwaysSend: true,
                code: 1024,
                fields: [
                    {
                        fieldType: DhcpOptionFieldType.IPv6Prefix,
                        values: ['3000::', '64'],
                    },
                    {
                        fieldType: DhcpOptionFieldType.Psid,
                        values: ['12', '8'],
                    },
                    {
                        fieldType: DhcpOptionFieldType.Binary,
                        values: ['01:02:03'],
                    },
                    {
                        fieldType: DhcpOptionFieldType.String,
                        values: ['foobar'],
                    },
                ],
                options: [],
            },
            {
                alwaysSend: false,
                code: 2024,
                fields: [
                    {
                        fieldType: DhcpOptionFieldType.Uint8,
                        values: ['101'],
                    },
                    {
                        fieldType: DhcpOptionFieldType.Uint16,
                        values: ['16523'],
                    },
                ],
                options: [],
            },
            {
                alwaysSend: true,
                code: 3087,
                fields: [],
                options: [
                    {
                        code: 1,
                        fields: [
                            {
                                fieldType: DhcpOptionFieldType.Int16,
                                values: ['-1111'],
                            },
                        ],
                        options: [
                            {
                                code: 2,
                                fields: [
                                    {
                                        fieldType: DhcpOptionFieldType.Bool,
                                        values: ['true'],
                                    },
                                ],
                            },
                        ],
                    },
                    {
                        code: 0,
                        fields: [
                            {
                                fieldType: DhcpOptionFieldType.Uint32,
                                values: ['2222'],
                            },
                            {
                                fieldType: DhcpOptionFieldType.Int8,
                                values: ['-127'],
                            },
                            {
                                fieldType: DhcpOptionFieldType.Int32,
                                values: ['-1000'],
                            },
                        ],
                    },
                ],
            },
        ]
        let formArray = service.convertOptionsToForm(IPType.IPv4, options)
        expect(formArray).toBeTruthy()
        expect(formArray.length).toBe(3)

        // Option 1024.
        expect(formArray.at(0).get('alwaysSend')).toBeTruthy()
        expect(formArray.at(0).get('optionCode')).toBeTruthy()
        expect(formArray.at(0).get('optionFields')).toBeTruthy()
        expect(formArray.at(0).get('suboptions')).toBeTruthy()

        expect(formArray.at(0).get('alwaysSend').value).toBeTrue()
        expect(formArray.at(0).get('optionCode').value).toBe(1024)

        // Option 1024 fields.
        expect(formArray.at(0).get('optionFields')).toBeInstanceOf(UntypedFormArray)
        let fields = formArray.at(0).get('optionFields') as UntypedFormArray
        expect(fields.controls.length).toBe(4)

        // Option 1024 field 0.
        expect(fields.at(0)).toBeInstanceOf(DhcpOptionFieldFormGroup)
        expect((fields.at(0) as DhcpOptionFieldFormGroup).data.fieldType).toBe(DhcpOptionFieldType.IPv6Prefix)
        expect(fields.at(0).get('prefix')).toBeTruthy()
        expect(fields.at(0).get('prefixLength')).toBeTruthy()
        expect(fields.at(0).get('prefix').value).toBe('3000::')
        expect(fields.at(0).get('prefixLength').value).toBe('64')

        // Option 1024 field 1.
        expect(fields.at(1)).toBeInstanceOf(DhcpOptionFieldFormGroup)
        expect((fields.at(1) as DhcpOptionFieldFormGroup).data.fieldType).toBe(DhcpOptionFieldType.Psid)
        expect(fields.at(1).get('psid')).toBeTruthy()
        expect(fields.at(1).get('psidLength')).toBeTruthy()
        expect(fields.at(1).get('psid').value).toBe('12')
        expect(fields.at(1).get('psidLength').value).toBe('8')

        // Option 1024 field 2.
        expect(fields.at(2)).toBeInstanceOf(DhcpOptionFieldFormGroup)
        expect((fields.at(2) as DhcpOptionFieldFormGroup).data.fieldType).toBe(DhcpOptionFieldType.Binary)
        expect(fields.at(2).get('control')).toBeTruthy()
        expect(fields.at(2).get('control').value).toBe('01:02:03')

        // Option 1024 field 3.
        expect(fields.at(3)).toBeInstanceOf(DhcpOptionFieldFormGroup)
        expect((fields.at(3) as DhcpOptionFieldFormGroup).data.fieldType).toBe(DhcpOptionFieldType.String)
        expect(fields.at(3).get('control')).toBeTruthy()
        expect(fields.at(3).get('control').value).toBe('foobar')

        // Option 1024 suboptions.
        expect(formArray.at(0).get('suboptions')).toBeInstanceOf(UntypedFormArray)
        expect((formArray.at(0).get('suboptions') as UntypedFormArray).controls.length).toBe(0)

        // Option 2024.
        expect(formArray.at(1).get('alwaysSend')).toBeTruthy()
        expect(formArray.at(1).get('optionCode')).toBeTruthy()
        expect(formArray.at(1).get('optionFields')).toBeTruthy()
        expect(formArray.at(1).get('suboptions')).toBeTruthy()

        expect(formArray.at(1).get('alwaysSend').value).toBeFalse()
        expect(formArray.at(1).get('optionCode').value).toBe(2024)

        // Option 2024 fields.
        expect(formArray.at(1).get('optionFields')).toBeInstanceOf(UntypedFormArray)
        fields = formArray.at(1).get('optionFields') as UntypedFormArray
        expect(fields.controls.length).toBe(2)

        // Option 2024 field 0.
        expect(fields.at(0)).toBeInstanceOf(DhcpOptionFieldFormGroup)
        expect((fields.at(0) as DhcpOptionFieldFormGroup).data.fieldType).toBe(DhcpOptionFieldType.Uint8)
        expect(fields.at(0).get('control')).toBeTruthy()
        expect(fields.at(0).get('control').value).toBe('101')

        // Option 2024 field 1.
        expect(fields.at(1)).toBeInstanceOf(DhcpOptionFieldFormGroup)
        expect((fields.at(1) as DhcpOptionFieldFormGroup).data.fieldType).toBe(DhcpOptionFieldType.Uint16)
        expect(fields.at(1).get('control')).toBeTruthy()
        expect(fields.at(1).get('control').value).toBe('16523')

        // Option 3087.
        expect(formArray.at(2).get('alwaysSend')).toBeTruthy()
        expect(formArray.at(2).get('optionCode')).toBeTruthy()
        expect(formArray.at(2).get('optionFields')).toBeTruthy()
        expect(formArray.at(2).get('suboptions')).toBeTruthy()

        expect(formArray.at(2).get('alwaysSend').value).toBeTrue()
        expect(formArray.at(2).get('optionCode').value).toBe(3087)

        // Option 3087 fields.
        expect(formArray.at(2).get('optionFields')).toBeInstanceOf(UntypedFormArray)
        fields = formArray.at(2).get('optionFields') as UntypedFormArray
        expect(fields.controls.length).toBe(0)

        // Option 3087 suboptions.
        expect(formArray.at(2).get('suboptions')).toBeInstanceOf(UntypedFormArray)
        expect((formArray.at(2).get('suboptions') as UntypedFormArray).controls.length).toBe(2)

        // Option 3087.1.
        expect(formArray.at(2).get('suboptions.0.alwaysSend')).toBeTruthy()
        expect(formArray.at(2).get('suboptions.0.optionCode')).toBeTruthy()
        expect(formArray.at(2).get('suboptions.0.optionFields')).toBeTruthy()
        expect(formArray.at(2).get('suboptions.0.suboptions')).toBeTruthy()

        expect(formArray.at(2).get('suboptions.0.alwaysSend').value).toBeFalse()
        expect(formArray.at(2).get('suboptions.0.optionCode').value).toBe(1)

        // Option 3087.1 field 0.
        fields = formArray.at(2).get('suboptions.0.optionFields') as UntypedFormArray
        expect(fields.controls.length).toBe(1)
        expect(fields.at(0)).toBeInstanceOf(DhcpOptionFieldFormGroup)
        expect((fields.at(0) as DhcpOptionFieldFormGroup).data.fieldType).toBe(DhcpOptionFieldType.Int16)
        expect(fields.at(0).get('control')).toBeTruthy()
        expect(fields.at(0).get('control').value).toBe('-1111')

        // Option 3087.1.2
        expect((formArray.at(2).get('suboptions.0.suboptions') as UntypedFormArray).controls.length).toBe(1)
        expect(formArray.at(2).get('suboptions.0.suboptions.0.alwaysSend')).toBeTruthy()
        expect(formArray.at(2).get('suboptions.0.suboptions.0.optionCode')).toBeTruthy()
        expect(formArray.at(2).get('suboptions.0.suboptions.0.optionFields')).toBeTruthy()
        expect(formArray.at(2).get('suboptions.0.suboptions.0.suboptions')).toBeTruthy()

        expect(formArray.at(2).get('suboptions.0.suboptions.0.alwaysSend').value).toBeFalse()
        expect(formArray.at(2).get('suboptions.0.suboptions.0.optionCode').value).toBe(2)
        expect(
            (formArray.at(2).get('suboptions.0.suboptions.0.optionFields') as UntypedFormArray).controls.length
        ).toBe(1)
        expect((formArray.at(2).get('suboptions.0.suboptions.0.suboptions') as UntypedFormArray).controls.length).toBe(
            0
        )

        // Option 3087.0.
        expect(formArray.at(2).get('suboptions.1.alwaysSend')).toBeTruthy()
        expect(formArray.at(2).get('suboptions.1.optionCode')).toBeTruthy()
        expect(formArray.at(2).get('suboptions.1.optionFields')).toBeTruthy()
        expect(formArray.at(2).get('suboptions.1.suboptions')).toBeTruthy()

        expect(formArray.at(2).get('suboptions.1.alwaysSend').value).toBeFalse()
        expect(formArray.at(2).get('suboptions.1.optionCode').value).toBe(0)

        // Option 3087.0 field 0.
        fields = formArray.at(2).get('suboptions.1.optionFields') as UntypedFormArray
        expect(fields.controls.length).toBe(3)
        expect(fields.at(0)).toBeInstanceOf(DhcpOptionFieldFormGroup)
        expect((fields.at(0) as DhcpOptionFieldFormGroup).data.fieldType).toBe(DhcpOptionFieldType.Uint32)
        expect(fields.at(0).get('control')).toBeTruthy()
        expect(fields.at(0).get('control').value).toBe('2222')

        // Option 3087.0 field 1
        expect(fields.at(1)).toBeInstanceOf(DhcpOptionFieldFormGroup)
        expect((fields.at(1) as DhcpOptionFieldFormGroup).data.fieldType).toBe(DhcpOptionFieldType.Int8)
        expect(fields.at(1).get('control')).toBeTruthy()
        expect(fields.at(1).get('control').value).toBe('-127')

        // Option 3087.0 field 2
        expect(fields.at(2)).toBeInstanceOf(DhcpOptionFieldFormGroup)
        expect((fields.at(2) as DhcpOptionFieldFormGroup).data.fieldType).toBe(DhcpOptionFieldType.Int32)
        expect(fields.at(2).get('control')).toBeTruthy()
        expect(fields.at(2).get('control').value).toBe('-1000')
    })

    it('returns empty array for null options', () => {
        let formArray = service.convertOptionsToForm(IPType.IPv4, null)
        expect(formArray).toBeTruthy()
        expect(formArray.length).toBe(0)
    })

    it('throws on too much recursion when converting options', () => {
        // Add an option with three nesting levels. It should throw because
        // we merely support first and second level suboptions.
        let options: Array<DHCPOption> = [
            {
                code: 1024,
                fields: [],
                options: [
                    {
                        code: 1,
                        fields: [],
                        options: [
                            {
                                code: 2,
                                fields: [],
                                options: [
                                    {
                                        code: 3,
                                        fields: [],
                                    },
                                ],
                            },
                        ],
                    },
                ],
            },
        ]
        expect(() => service.convertOptionsToForm(IPType.IPv4, options)).toThrow()
    })

    it('throws when IPv6 prefix field has only one value', () => {
        let options: Array<DHCPOption> = [
            {
                code: 1024,
                fields: [
                    {
                        fieldType: DhcpOptionFieldType.IPv6Prefix,
                        values: ['3000::'],
                    },
                ],
                options: [],
            },
        ]
        expect(() => service.convertOptionsToForm(IPType.IPv4, options)).toThrow()
    })

    it('throws when IPv6 prefix field has three values', () => {
        let options: Array<DHCPOption> = [
            {
                code: 1024,
                fields: [
                    {
                        fieldType: DhcpOptionFieldType.IPv6Prefix,
                        values: ['3000::', '64', '5'],
                    },
                ],
                options: [],
            },
        ]
        expect(() => service.convertOptionsToForm(IPType.IPv4, options)).toThrow()
    })

    it('throws when PSID field has only one value', () => {
        let options: Array<DHCPOption> = [
            {
                code: 1024,
                fields: [
                    {
                        fieldType: DhcpOptionFieldType.Psid,
                        values: ['12'],
                    },
                ],
                options: [],
            },
        ]
        expect(() => service.convertOptionsToForm(IPType.IPv4, options)).toThrow()
    })

    it('throws when PSID field has three values', () => {
        let options: Array<DHCPOption> = [
            {
                code: 1024,
                fields: [
                    {
                        fieldType: DhcpOptionFieldType.Psid,
                        values: ['12', '8', '5'],
                    },
                ],
                options: [],
            },
        ]
        expect(() => service.convertOptionsToForm(IPType.IPv4, options)).toThrow()
    })

    it('throws when string field has two values', () => {
        let options: Array<DHCPOption> = [
            {
                code: 1024,
                fields: [
                    {
                        fieldType: DhcpOptionFieldType.String,
                        values: ['foo', 'bar'],
                    },
                ],
                options: [],
            },
        ]
        expect(() => service.convertOptionsToForm(IPType.IPv4, options)).toThrow()
    })

    it('creates binary field', () => {
        let formGroup = service.createBinaryField('01:02:03')
        expect(formGroup).toBeTruthy()
        expect(formGroup.data.fieldType).toBe(DhcpOptionFieldType.Binary)
        expect(formGroup.contains('control')).toBeTrue()
        expect(formGroup.get('control').value).toBe('01:02:03')
    })

    it('creates string field', () => {
        let formGroup = service.createStringField('foo')
        expect(formGroup).toBeTruthy()
        expect(formGroup.data.fieldType).toBe(DhcpOptionFieldType.String)
        expect(formGroup.contains('control')).toBeTrue()
        expect(formGroup.get('control').value).toBe('foo')
    })

    it('creates boolean field from string', () => {
        let formGroup = service.createBoolField('true')
        expect(formGroup).toBeTruthy()
        expect(formGroup.data.fieldType).toBe(DhcpOptionFieldType.Bool)
        expect(formGroup.contains('control')).toBeTrue()
        expect(formGroup.get('control').value).toBeTrue()
    })

    it('creates boolean field from boolean', () => {
        let formGroup = service.createBoolField(false)
        expect(formGroup).toBeTruthy()
        expect(formGroup.data.fieldType).toBe(DhcpOptionFieldType.Bool)
        expect(formGroup.contains('control')).toBeTrue()
        expect(formGroup.get('control').value).toBeFalse()
    })

    it('creates uint8 field', () => {
        let formGroup = service.createUint8Field(123)
        expect(formGroup).toBeTruthy()
        expect(formGroup.data.fieldType).toBe(DhcpOptionFieldType.Uint8)
        expect(formGroup.contains('control')).toBeTrue()
        expect(formGroup.get('control').value).toBe(123)
    })

    it('creates uint16 field', () => {
        let formGroup = service.createUint16Field(234)
        expect(formGroup).toBeTruthy()
        expect(formGroup.data.fieldType).toBe(DhcpOptionFieldType.Uint16)
        expect(formGroup.contains('control')).toBeTrue()
        expect(formGroup.get('control').value).toBe(234)
    })

    it('creates uint32 field', () => {
        let formGroup = service.createUint32Field(345)
        expect(formGroup).toBeTruthy()
        expect(formGroup.data.fieldType).toBe(DhcpOptionFieldType.Uint32)
        expect(formGroup.contains('control')).toBeTrue()
        expect(formGroup.get('control').value).toBe(345)
    })

    it('creates ipv4-address field', () => {
        let formGroup = service.createIPv4AddressField('192.0.2.1')
        expect(formGroup).toBeTruthy()
        expect(formGroup.data.fieldType).toBe(DhcpOptionFieldType.IPv4Address)
        expect(formGroup.contains('control')).toBeTrue()
        expect(formGroup.get('control').value).toBe('192.0.2.1')
    })

    it('creates ipv6-address field', () => {
        let formGroup = service.createIPv6AddressField('2001:db8:1::1')
        expect(formGroup).toBeTruthy()
        expect(formGroup.data.fieldType).toBe(DhcpOptionFieldType.IPv6Address)
        expect(formGroup.contains('control')).toBeTrue()
        expect(formGroup.get('control').value).toBe('2001:db8:1::1')
    })

    it('creates ipv6-prefix field', () => {
        let formGroup = service.createIPv6PrefixField('3000::', '64')
        expect(formGroup).toBeTruthy()
        expect(formGroup.data.fieldType).toBe(DhcpOptionFieldType.IPv6Prefix)
        expect(formGroup.contains('prefix')).toBeTrue()
        expect(formGroup.contains('prefixLength')).toBeTrue()
        expect(formGroup.get('prefix').value).toBe('3000::')
        expect(formGroup.get('prefixLength').value).toBe('64')
    })

    it('creates psid field', () => {
        let formGroup = service.createPsidField('12', '8')
        expect(formGroup).toBeTruthy()
        expect(formGroup.data.fieldType).toBe(DhcpOptionFieldType.Psid)
        expect(formGroup.contains('psid')).toBeTrue()
        expect(formGroup.contains('psidLength')).toBeTrue()
        expect(formGroup.get('psid').value).toBe('12')
        expect(formGroup.get('psidLength').value).toBe('8')
    })

    it('creates fqdn field initialized with a full FQDN', () => {
        let formGroup = service.createFqdnField('foo.example.org.')
        expect(formGroup).toBeTruthy()
        expect(formGroup.data.fieldType).toBe(DhcpOptionFieldType.Fqdn)
        expect(formGroup.contains('control')).toBeTrue()
        const control = formGroup.get('control')
        expect(control.value).toBe('foo.example.org.')
        expect(control.hasValidator(StorkValidators.fullFqdn)).toBeTrue()
        expect(control.hasValidator(StorkValidators.partialFqdn)).toBeFalse()
        const toggleButton = formGroup.get('isPartialFqdn')
        expect(toggleButton).toBeTruthy()
        expect(toggleButton.value).toBeFalse()
    })

    it('creates fqdn field initialized with a partial FQDN', () => {
        let formGroup = service.createFqdnField('foo.example.org')
        expect(formGroup).toBeTruthy()
        expect(formGroup.data.fieldType).toBe(DhcpOptionFieldType.Fqdn)
        expect(formGroup.contains('control')).toBeTrue()
        const control = formGroup.get('control')
        expect(control.value).toBe('foo.example.org')
        expect(control.hasValidator(StorkValidators.fullFqdn)).toBeFalse()
        expect(control.hasValidator(StorkValidators.partialFqdn)).toBeTrue()
        const toggleButton = formGroup.get('isPartialFqdn')
        expect(toggleButton).toBeTruthy()
        expect(toggleButton.value).toBeTrue()
    })

    it('creates fqdn field initialized with an empty FQDN', () => {
        let formGroup = service.createFqdnField('')
        expect(formGroup).toBeTruthy()
        expect(formGroup.data.fieldType).toBe(DhcpOptionFieldType.Fqdn)
        expect(formGroup.contains('control')).toBeTrue()
        const control = formGroup.get('control')
        expect(control.value).toBe('')
        expect(control.hasValidator(StorkValidators.fullFqdn)).toBeTrue()
        expect(control.hasValidator(StorkValidators.partialFqdn)).toBeFalse()
        const toggleButton = formGroup.get('isPartialFqdn')
        expect(toggleButton).toBeTruthy()
        expect(toggleButton.value).toBeFalse()
    })

    it('creates fqdn field initialized with an invalid FQDN', () => {
        let formGroup = service.createFqdnField('---')
        expect(formGroup).toBeTruthy()
        expect(formGroup.data.fieldType).toBe(DhcpOptionFieldType.Fqdn)
        expect(formGroup.contains('control')).toBeTrue()
        const control = formGroup.get('control')
        expect(control.value).toBe('---')
        expect(control.hasValidator(StorkValidators.fullFqdn)).toBeTrue()
        expect(control.hasValidator(StorkValidators.partialFqdn)).toBeFalse()
        const toggleButton = formGroup.get('isPartialFqdn')
        expect(toggleButton).toBeTruthy()
        expect(toggleButton.value).toBeFalse()
    })
})
