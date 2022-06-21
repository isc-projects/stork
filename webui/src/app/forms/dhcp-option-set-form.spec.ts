import { FormBuilder } from '@angular/forms'
import { DhcpOptionFieldFormGroup, DhcpOptionFieldType } from './dhcp-option-field'
import { DhcpOptionSetForm } from './dhcp-option-set-form'
import { Universe } from '../universe'

describe('DhcpOptionSetForm', () => {
    let formBuilder: FormBuilder = new FormBuilder()

    it('serializes an option set', () => {
        // Add a form with three options.
        const formArray = formBuilder.array([
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
                ]),
            }),
        ])

        // Extract the options from the form and make sure there are
        // three of them.
        const options = new DhcpOptionSetForm(formArray)
        options.process(Universe.IPv4)
        const serialized = options.getSerializedOptions()
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
        expect(serialized[0].fields[2].fieldType).toBe(DhcpOptionFieldType.HexBytes)
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
        // The option should contain a suboption.
        expect(serialized[2].options.length).toBe(1)
        expect(serialized[2].options[0].hasOwnProperty('code')).toBeTrue()
        expect(serialized[2].options[0].hasOwnProperty('encapsulate')).toBeTrue()
        expect(serialized[2].options[0].hasOwnProperty('fields')).toBeTrue()
        expect(serialized[2].options[0].hasOwnProperty('options')).toBeTrue()
        expect(serialized[2].options[0].code).toBe(1)
        expect(serialized[2].options[0].encapsulate.length).toBe(0)
        expect(serialized[2].options[0].fields.length).toBe(1)
        expect(serialized[2].options[0].fields[0].fieldType).toBe(DhcpOptionFieldType.Uint16)
        expect(serialized[2].options[0].options.length).toBe(0)
    })

    it('throws on too much recursion', () => {
        // Add an option with three nesting levels. It should throw because
        // we merely support first level suboptions.
        const formArray = formBuilder.array([
            formBuilder.group({
                optionCode: formBuilder.control(1024),
                optionFields: formBuilder.array([]),
                suboptions: formBuilder.array([
                    formBuilder.group({
                        optionCode: formBuilder.control(1),
                        optionFields: formBuilder.array([]),
                        suboptions: formBuilder.array([
                            formBuilder.group({
                                optionCode: formBuilder.control(2),
                                optionFields: formBuilder.array([]),
                                suboptions: formBuilder.array([]),
                            }),
                        ]),
                    }),
                ]),
            }),
        ])

        const options = new DhcpOptionSetForm(formArray)
        expect(() => options.process(Universe.IPv4)).toThrow()
    })

    it('throws when there is no option code', () => {
        const formArray = formBuilder.array([formBuilder.group({})])
        const options = new DhcpOptionSetForm(formArray)
        expect(() => options.process(Universe.IPv4)).toThrow()
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
        const options = new DhcpOptionSetForm(formArray)
        expect(() => options.process(Universe.IPv4)).toThrow()
    })

    it('throws when prefix field lacks prefix control', () => {
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
        const options = new DhcpOptionSetForm(formArray)
        expect(() => options.process(Universe.IPv4)).toThrow()
    })

    it('throws when psid field lacks psid control', () => {
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
        const options = new DhcpOptionSetForm(formArray)
        expect(() => options.process(Universe.IPv4)).toThrow()
    })

    it('throws when psid field lacks psid length control', () => {
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
        const options = new DhcpOptionSetForm(formArray)
        expect(() => options.process(Universe.IPv4)).toThrow()
    })

    it('throws when a single value field lacks control', () => {
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
        const options = new DhcpOptionSetForm(formArray)
        expect(() => options.process(Universe.IPv4)).toThrow()
    })

    it('throws when options have not been processed', () => {
        const formArray = formBuilder.array([
            formBuilder.group({
                optionCode: formBuilder.control(3087),
                optionFields: formBuilder.array([
                    new DhcpOptionFieldFormGroup(DhcpOptionFieldType.Uint8, {
                        control: formBuilder.control(100),
                    }),
                ]),
            }),
        ])
        const options = new DhcpOptionSetForm(formArray)
        expect(() => options.getSerializedOptions()).toThrow()
    })
})
