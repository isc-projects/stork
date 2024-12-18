import { TestBed } from '@angular/core/testing'

import {
    KeaGlobalConfigurationForm,
    KeaGlobalParametersForm,
    KeaPoolParametersForm,
    KeaRawConfig,
    KeaSubnetParametersForm,
    OptionsForm,
    SubnetSetFormService,
    UserContextsForm,
    VersionedDaemon,
} from './subnet-set-form.service'
import {
    KeaConfigPoolParameters,
    KeaConfigSubnetDerivedParameters,
    KeaDaemonConfig,
    SharedNetwork,
    Subnet,
} from '../backend'
import { SharedParameterFormGroup } from './shared-parameter-form-group'
import { FormControl, FormGroup, UntypedFormArray, UntypedFormControl } from '@angular/forms'
import { IPType } from '../iptype'

describe('SubnetSetFormService', () => {
    let service: SubnetSetFormService

    beforeEach(() => {
        TestBed.configureTestingModule({})
        service = TestBed.inject(SubnetSetFormService)
    })

    it('should be created', () => {
        expect(service).toBeTruthy()
    })

    it('should create a default options form', () => {
        let fg = service.createDefaultOptionsForm()
        expect(fg).toBeTruthy()
        expect(fg.contains('unlocked'))
        expect(fg.get('unlocked').value).toBeFalse()
        expect(fg.get('unlocked').disabled).toBeTrue()
        expect(fg.contains('data'))
        const data = fg.get('data') as UntypedFormArray
        expect(data).toBeTruthy()
        expect(data.length).toBe(1)
        const data0 = data.get('0') as UntypedFormArray
        expect(data0).toBeTruthy()
        expect(data0.length).toBe(0)
    })

    it('should convert Kea pool parameters to a form group', () => {
        const parameters: KeaConfigPoolParameters[] = [
            {
                clientClass: 'foo',
                requireClientClasses: ['foo', 'bar'],
            },
            {
                clientClass: 'bar',
                requireClientClasses: ['foo', 'bar'],
            },
        ]
        const form = service.convertKeaPoolParametersToForm(parameters)
        let fg = form.get('clientClass') as SharedParameterFormGroup<any>
        expect(fg).toBeTruthy()
        expect(fg.data.type).toBe('string')
        expect(fg.data.min).toBeFalsy()
        expect(fg.data.max).toBeFalsy()
        expect(fg.data.fractionDigits).toBeFalsy()
        expect(fg.data.values).toBeFalsy()
        expect((fg.get('unlocked') as UntypedFormControl)?.value).toBeTrue()
        expect((fg.get('values') as UntypedFormArray)?.controls.length).toBe(2)
        expect((fg.get('values') as UntypedFormArray)?.controls[0].value).toBe('foo')
        expect((fg.get('values') as UntypedFormArray)?.controls[1].value).toBe('bar')

        fg = form.get('requireClientClasses') as SharedParameterFormGroup<any>
        expect(fg).toBeTruthy()
        expect(fg.data.type).toBe('client-classes')
        expect(fg.data.min).toBeFalsy()
        expect(fg.data.max).toBeFalsy()
        expect(fg.data.fractionDigits).toBeFalsy()
        expect(fg.data.values).toBeFalsy()
        expect((fg.get('unlocked') as UntypedFormControl)?.value).toBeFalse()
        expect((fg.get('values') as UntypedFormArray)?.controls.length).toBe(2)
        expect((fg.get('values') as UntypedFormArray)?.controls[0].value).toEqual(['foo', 'bar'])
        expect((fg.get('values') as UntypedFormArray)?.controls[1].value).toEqual(['foo', 'bar'])
    })

    it('should convert address pool data to a form', () => {
        const subnet: Subnet = {
            subnet: '192.0.2.0/24',
            localSubnets: [
                {
                    daemonId: 1,
                    pools: [
                        {
                            pool: '192.0.2.1-192.0.2.10',
                            keaConfigPoolParameters: {
                                clientClass: 'foo',
                                poolID: 50,
                                requireClientClasses: ['foo', 'bar'],
                                options: [
                                    {
                                        alwaysSend: true,
                                        code: 5,
                                        encapsulate: '',
                                        fields: [
                                            {
                                                fieldType: 'ipv4-address',
                                                values: ['192.0.2.1'],
                                            },
                                        ],
                                        options: [],
                                        universe: 4,
                                    },
                                ],
                                optionsHash: '123',
                            },
                        },
                        {
                            pool: '192.0.2.20-192.0.2.30',
                            keaConfigPoolParameters: {
                                clientClass: 'foo',
                                poolID: 50,
                                requireClientClasses: ['foo', 'bar'],
                                options: [
                                    {
                                        alwaysSend: true,
                                        code: 5,
                                        encapsulate: '',
                                        fields: [
                                            {
                                                fieldType: 'ipv4-address',
                                                values: ['192.0.2.2'],
                                            },
                                        ],
                                        options: [],
                                        universe: 4,
                                    },
                                ],
                                optionsHash: '234',
                            },
                        },
                        {
                            pool: '192.0.2.40-192.0.2.50',
                            keaConfigPoolParameters: {
                                clientClass: 'bar',
                                requireClientClasses: ['faz'],
                                options: [
                                    {
                                        alwaysSend: true,
                                        code: 5,
                                        encapsulate: '',
                                        fields: [
                                            {
                                                fieldType: 'ipv4-address',
                                                values: ['192.0.2.3'],
                                            },
                                        ],
                                        options: [],
                                        universe: 4,
                                    },
                                ],
                                optionsHash: '345',
                            },
                        },
                    ],
                },
                {
                    daemonId: 2,
                    pools: [
                        {
                            pool: '192.0.2.1-192.0.2.10',
                            keaConfigPoolParameters: {
                                clientClass: 'foo',
                                poolID: 50,
                                requireClientClasses: ['foo', 'bar'],
                            },
                        },
                        {
                            pool: '192.0.2.20-192.0.2.30',
                            keaConfigPoolParameters: {
                                options: [
                                    {
                                        alwaysSend: true,
                                        code: 5,
                                        encapsulate: '',
                                        fields: [
                                            {
                                                fieldType: 'ipv4-address',
                                                values: ['192.0.2.2'],
                                            },
                                        ],
                                        options: [],
                                        universe: 4,
                                    },
                                ],
                                optionsHash: '234',
                            },
                        },
                    ],
                },
            ],
        }
        const formArray = service.convertAddressPoolsToForm(subnet)
        expect(formArray.length).toBe(3)
        expect(formArray.get('0.range.start')?.value).toBe('192.0.2.1')
        expect(formArray.get('0.range.end')?.value).toBe('192.0.2.10')

        let params = formArray.get('0.parameters') as FormGroup<KeaPoolParametersForm>
        expect(params.get('clientClass.unlocked')?.value).toBeFalse()
        expect((params.get('clientClass.values') as UntypedFormArray).length)?.toBe(2)
        expect(params.get('clientClass.values.0')?.value).toBe('foo')
        expect(params.get('clientClass.values.1')?.value).toBe('foo')
        expect(params.get('requireClientClasses.unlocked')?.value).toBeFalse()
        expect((params.get('requireClientClasses.values') as UntypedFormArray)?.length).toBe(2)
        expect(params.get('requireClientClasses.values.0')?.value).toEqual(['foo', 'bar'])
        expect(params.get('requireClientClasses.values.1')?.value).toEqual(['foo', 'bar'])
        expect(params.get('poolID.unlocked')?.value).toBeFalse()
        expect(params.get('poolID.values.0')?.value).toBe(50)
        expect(params.get('poolID.values.1')?.value).toBe(50)

        let options = formArray.get('0.options.data') as UntypedFormArray
        expect(options.length).toBe(2)
        expect(options.get('0.0.optionFields.0.control')?.value).toBe('192.0.2.1')
        expect((options.get('1') as UntypedFormArray).length).toBe(0)

        let selectedDaemons = formArray.get('0.selectedDaemons') as FormControl<number[]>
        expect(selectedDaemons?.value).toEqual([1, 2])

        expect(formArray.get('1.range.start')?.value).toBe('192.0.2.20')
        expect(formArray.get('1.range.end')?.value).toBe('192.0.2.30')

        params = formArray.get('1.parameters') as FormGroup<KeaPoolParametersForm>
        expect(params.get('clientClass.unlocked')?.value).toBeTrue()
        expect((params.get('clientClass.values') as UntypedFormArray).length)?.toBe(2)
        expect(params.get('clientClass.values.0')?.value).toBe('foo')
        expect(params.get('clientClass.values.1')?.value).toBeFalsy()
        expect(params.get('requireClientClasses.unlocked')?.value).toBeTrue()
        expect((params.get('requireClientClasses.values') as UntypedFormArray)?.length).toBe(2)
        expect(params.get('requireClientClasses.values.0')?.value).toEqual(['foo', 'bar'])
        expect(params.get('requireClientClasses.values.1')?.value).toEqual([])
        expect(params.get('poolID.unlocked')?.value).toBeTrue()
        expect(params.get('poolID.values.0')?.value).toBe(50)
        expect(params.get('poolID.values.1')?.value).toBeFalsy()

        options = formArray.get('1.options.data') as UntypedFormArray
        expect(options.length).toBe(2)
        expect(options.get('0.0.optionFields.0.control')?.value).toBe('192.0.2.2')
        expect(options.get('1.0.optionFields.0.control')?.value).toBe('192.0.2.2')

        selectedDaemons = formArray.get('1.selectedDaemons') as FormControl<number[]>
        expect(selectedDaemons?.value).toEqual([1, 2])

        expect(formArray.get('2.range.start')?.value).toBe('192.0.2.40')
        expect(formArray.get('2.range.end')?.value).toBe('192.0.2.50')

        params = formArray.get('2.parameters') as FormGroup<KeaPoolParametersForm>
        expect(params.get('clientClass.unlocked')?.value).toBeFalse()
        expect((params.get('clientClass.values') as UntypedFormArray).length)?.toBe(1)
        expect(params.get('clientClass.values.0')?.value).toBe('bar')
        expect(params.get('requireClientClasses.unlocked')?.value).toBeFalse()
        expect((params.get('requireClientClasses.values') as UntypedFormArray)?.length).toBe(1)
        expect(params.get('requireClientClasses.values.0')?.value).toEqual(['faz'])
        expect(params.get('poolID.unlocked')?.value).toBeFalse()
        expect(params.get('poolID.values.0')?.value).toBeFalsy()
        expect(params.get('poolID.values.1')?.value).toBeFalsy()

        options = formArray.get('2.options.data') as UntypedFormArray
        expect(options.length).toBe(1)
        expect(options.get('0.0.optionFields.0.control')?.value).toBe('192.0.2.3')

        selectedDaemons = formArray.get('2.selectedDaemons') as FormControl<number[]>
        expect(selectedDaemons?.value).toEqual([1])
    })

    it('should convert form to address pool data', () => {
        const subnet: Subnet = {
            subnet: '192.0.2.0/24',
            localSubnets: [
                {
                    daemonId: 1,
                    pools: [
                        {
                            pool: '192.0.2.1-192.0.2.10',
                            keaConfigPoolParameters: {
                                clientClass: 'foo',
                                poolID: 50,
                                requireClientClasses: ['foo', 'bar'],
                                options: [
                                    {
                                        alwaysSend: true,
                                        code: 5,
                                        encapsulate: '',
                                        fields: [
                                            {
                                                fieldType: 'ipv4-address',
                                                values: ['192.0.2.1'],
                                            },
                                        ],
                                        options: [],
                                        universe: 4,
                                    },
                                ],
                                optionsHash: '123',
                            },
                        },
                        {
                            pool: '192.0.2.20-192.0.2.30',
                            keaConfigPoolParameters: {
                                clientClass: 'foo',
                                poolID: 60,
                                requireClientClasses: ['foo', 'bar'],
                                options: [
                                    {
                                        alwaysSend: true,
                                        code: 5,
                                        encapsulate: '',
                                        fields: [
                                            {
                                                fieldType: 'ipv4-address',
                                                values: ['192.0.2.2'],
                                            },
                                        ],
                                        options: [],
                                        universe: 4,
                                    },
                                ],
                                optionsHash: '234',
                            },
                        },
                        {
                            pool: '192.0.2.40-192.0.2.50',
                            keaConfigPoolParameters: {
                                clientClass: 'bar',
                                poolID: 70,
                                requireClientClasses: ['faz'],
                                options: [
                                    {
                                        alwaysSend: true,
                                        code: 5,
                                        encapsulate: '',
                                        fields: [
                                            {
                                                fieldType: 'ipv4-address',
                                                values: ['192.0.2.3'],
                                            },
                                        ],
                                        options: [],
                                        universe: 4,
                                    },
                                ],
                                optionsHash: '345',
                            },
                        },
                    ],
                },
                {
                    daemonId: 2,
                    pools: [
                        {
                            pool: '192.0.2.1-192.0.2.10',
                            keaConfigPoolParameters: {
                                clientClass: 'foo',
                                poolID: 80,
                                requireClientClasses: ['foo', 'bar'],
                            },
                        },
                        {
                            pool: '192.0.2.20-192.0.2.30',
                            keaConfigPoolParameters: {
                                options: [
                                    {
                                        alwaysSend: true,
                                        code: 5,
                                        encapsulate: '',
                                        fields: [
                                            {
                                                fieldType: 'ipv4-address',
                                                values: ['192.0.2.2'],
                                            },
                                        ],
                                        options: [],
                                        universe: 4,
                                    },
                                ],
                                optionsHash: '234',
                            },
                        },
                    ],
                },
            ],
        }
        const formArray = service.convertAddressPoolsToForm(subnet)

        let pools = service.convertFormToAddressPools([], subnet.localSubnets[0], formArray)
        expect(pools.length).toBe(3)
        expect(pools[0].pool).toBe('192.0.2.1-192.0.2.10')
        expect(pools[0].keaConfigPoolParameters).toBeTruthy()
        expect(pools[0].keaConfigPoolParameters.clientClass).toBe('foo')
        expect(pools[0].keaConfigPoolParameters.poolID).toBe(50)
        expect(pools[0].keaConfigPoolParameters.requireClientClasses).toEqual(['foo', 'bar'])
        expect(pools[0].keaConfigPoolParameters.options?.length).toBe(1)
        expect(pools[0].keaConfigPoolParameters.options[0].alwaysSend).toBeTrue()
        expect(pools[0].keaConfigPoolParameters.options[0].code).toBe(5)
        expect(pools[0].keaConfigPoolParameters.options[0].fields?.length).toBe(1)
        expect(pools[0].keaConfigPoolParameters.options[0].fields[0].fieldType).toBe('ipv4-address')
        expect(pools[0].keaConfigPoolParameters.options[0].fields[0].values?.length).toBe(1)
        expect(pools[0].keaConfigPoolParameters.options[0].fields[0].values[0]).toBe('192.0.2.1')
        expect(pools[1].pool).toBe('192.0.2.20-192.0.2.30')
        expect(pools[1].keaConfigPoolParameters).toBeTruthy()
        expect(pools[1].keaConfigPoolParameters.clientClass).toBe('foo')
        expect(pools[1].keaConfigPoolParameters.poolID).toBe(60)
        expect(pools[1].keaConfigPoolParameters.requireClientClasses).toEqual(['foo', 'bar'])
        expect(pools[1].keaConfigPoolParameters.options?.length).toBe(1)
        expect(pools[1].keaConfigPoolParameters.options[0].alwaysSend).toBeTrue()
        expect(pools[1].keaConfigPoolParameters.options[0].code).toBe(5)
        expect(pools[1].keaConfigPoolParameters.options[0].fields?.length).toBe(1)
        expect(pools[1].keaConfigPoolParameters.options[0].fields[0].fieldType).toBe('ipv4-address')
        expect(pools[1].keaConfigPoolParameters.options[0].fields[0].values?.length).toBe(1)
        expect(pools[1].keaConfigPoolParameters.options[0].fields[0].values[0]).toBe('192.0.2.2')
        expect(pools[2].pool).toBe('192.0.2.40-192.0.2.50')
        expect(pools[2].keaConfigPoolParameters).toBeTruthy()
        expect(pools[2].keaConfigPoolParameters.clientClass).toBe('bar')
        expect(pools[2].keaConfigPoolParameters.poolID).toBe(70)
        expect(pools[2].keaConfigPoolParameters.requireClientClasses).toEqual(['faz'])
        expect(pools[2].keaConfigPoolParameters.options?.length).toBe(1)
        expect(pools[2].keaConfigPoolParameters.options[0].alwaysSend).toBeTrue()
        expect(pools[2].keaConfigPoolParameters.options[0].code).toBe(5)
        expect(pools[2].keaConfigPoolParameters.options[0].fields?.length).toBe(1)
        expect(pools[2].keaConfigPoolParameters.options[0].fields[0].fieldType).toBe('ipv4-address')
        expect(pools[2].keaConfigPoolParameters.options[0].fields[0].values?.length).toBe(1)
        expect(pools[2].keaConfigPoolParameters.options[0].fields[0].values[0]).toBe('192.0.2.3')

        pools = service.convertFormToAddressPools([], subnet.localSubnets[1], formArray)
        expect(pools.length).toBe(2)
        expect(pools[0].pool).toBe('192.0.2.1-192.0.2.10')
        expect(pools[0].keaConfigPoolParameters).toBeTruthy()
        expect(pools[0].keaConfigPoolParameters.clientClass).toBe('foo')
        expect(pools[0].keaConfigPoolParameters.poolID).toBe(80)
        expect(pools[0].keaConfigPoolParameters.requireClientClasses).toEqual(['foo', 'bar'])
        expect(pools[0].keaConfigPoolParameters.options).toEqual([])
        expect(pools[1].pool).toBe('192.0.2.20-192.0.2.30')
        expect(pools[1].keaConfigPoolParameters).toBeTruthy()
        expect(pools[1].keaConfigPoolParameters.clientClass).toBeFalsy()
        expect(pools[1].keaConfigPoolParameters.poolID).toBeFalsy()
        expect(pools[1].keaConfigPoolParameters.requireClientClasses).toEqual([])
        expect(pools[1].keaConfigPoolParameters.options?.length).toBe(1)
        expect(pools[1].keaConfigPoolParameters.options[0].alwaysSend).toBeTrue()
        expect(pools[1].keaConfigPoolParameters.options[0].code).toBe(5)
        expect(pools[1].keaConfigPoolParameters.options[0].fields?.length).toBe(1)
        expect(pools[1].keaConfigPoolParameters.options[0].fields[0].fieldType).toBe('ipv4-address')
        expect(pools[1].keaConfigPoolParameters.options[0].fields[0].values?.length).toBe(1)
        expect(pools[1].keaConfigPoolParameters.options[0].fields[0].values[0]).toBe('192.0.2.2')
    })

    it('should convert prefix pool data to a form', () => {
        const subnet: Subnet = {
            subnet: '2001:db8:1::/64',
            localSubnets: [
                {
                    daemonId: 1,
                    prefixDelegationPools: [
                        {
                            prefix: '3000::/16',
                            delegatedLength: 112,
                            excludedPrefix: '3000::ee00/120',
                            keaConfigPoolParameters: {
                                clientClass: 'foo',
                                poolID: 50,
                                requireClientClasses: ['foo', 'bar'],
                                options: [
                                    {
                                        alwaysSend: true,
                                        code: 23,
                                        encapsulate: '',
                                        fields: [
                                            {
                                                fieldType: 'ipv6-address',
                                                values: ['2001:db8:1::1'],
                                            },
                                        ],
                                        options: [],
                                        universe: 4,
                                    },
                                ],
                                optionsHash: '123',
                            },
                        },
                        {
                            prefix: '3001::/16',
                            delegatedLength: 112,
                            excludedPrefix: null,
                            keaConfigPoolParameters: {
                                clientClass: 'foo',
                                poolID: 60,
                                requireClientClasses: ['foo', 'bar'],
                                options: [
                                    {
                                        alwaysSend: true,
                                        code: 23,
                                        encapsulate: '',
                                        fields: [
                                            {
                                                fieldType: 'ipv6-address',
                                                values: ['2001:db8:1::2'],
                                            },
                                        ],
                                        options: [],
                                        universe: 4,
                                    },
                                ],
                                optionsHash: '234',
                            },
                        },
                        {
                            prefix: '3002::/16',
                            delegatedLength: 112,
                            excludedPrefix: null,
                            keaConfigPoolParameters: {
                                clientClass: 'bar',
                                poolID: 70,
                                requireClientClasses: ['faz'],
                                options: [
                                    {
                                        alwaysSend: true,
                                        code: 23,
                                        encapsulate: '',
                                        fields: [
                                            {
                                                fieldType: 'ipv6-address',
                                                values: ['2001:db8:1::3'],
                                            },
                                        ],
                                        options: [],
                                        universe: 4,
                                    },
                                ],
                                optionsHash: '345',
                            },
                        },
                    ],
                },
                {
                    daemonId: 2,
                    prefixDelegationPools: [
                        {
                            prefix: '3000::/16',
                            delegatedLength: 112,
                            excludedPrefix: '3000::ee00/120',
                            keaConfigPoolParameters: {
                                clientClass: 'foo',
                                poolID: 50,
                                requireClientClasses: ['foo', 'bar'],
                            },
                        },
                        {
                            prefix: '3001::/16',
                            delegatedLength: 112,
                            excludedPrefix: null,
                            keaConfigPoolParameters: {
                                options: [
                                    {
                                        alwaysSend: true,
                                        code: 23,
                                        encapsulate: '',
                                        fields: [
                                            {
                                                fieldType: 'ipv6-address',
                                                values: ['2001:db8:1::2'],
                                            },
                                        ],
                                        options: [],
                                        universe: 4,
                                    },
                                ],
                                optionsHash: '234',
                            },
                        },
                    ],
                },
            ],
        }
        const formArray = service.convertPrefixPoolsToForm(subnet)
        expect(formArray.length).toBe(3)
        expect(formArray.get('0.prefixes.prefix')?.value).toBe('3000::/16')
        expect(formArray.get('0.prefixes.delegatedLength')?.value).toBe(112)
        expect(formArray.get('0.prefixes.excludedPrefix')?.value).toBe('3000::ee00/120')

        let params = formArray.get('0.parameters') as FormGroup<KeaPoolParametersForm>
        expect(params.get('clientClass.unlocked')?.value).toBeFalse()
        expect((params.get('clientClass.values') as UntypedFormArray).length)?.toBe(2)
        expect(params.get('clientClass.values.0')?.value).toBe('foo')
        expect(params.get('clientClass.values.1')?.value).toBe('foo')
        expect(params.get('requireClientClasses.unlocked')?.value).toBeFalse()
        expect((params.get('requireClientClasses.values') as UntypedFormArray)?.length).toBe(2)
        expect(params.get('requireClientClasses.values.0')?.value).toEqual(['foo', 'bar'])
        expect(params.get('requireClientClasses.values.1')?.value).toEqual(['foo', 'bar'])
        expect(params.get('poolID.unlocked')?.value).toBeFalse()
        expect(params.get('poolID.values.0')?.value).toBe(50)
        expect(params.get('poolID.values.1')?.value).toBe(50)

        let options = formArray.get('0.options.data') as UntypedFormArray
        expect(options.length).toBe(2)
        expect(options.get('0.0.optionFields.0.control')?.value).toBe('2001:db8:1::1')
        expect((options.get('1') as UntypedFormArray).length).toBe(0)

        let selectedDaemons = formArray.get('0.selectedDaemons') as FormControl<number[]>
        expect(selectedDaemons?.value).toEqual([1, 2])

        expect(formArray.get('1.prefixes.prefix')?.value).toBe('3001::/16')
        expect(formArray.get('1.prefixes.delegatedLength')?.value).toBe(112)
        expect(formArray.get('1.prefixes.excludedPrefix')?.value).toBeFalsy()

        params = formArray.get('1.parameters') as FormGroup<KeaPoolParametersForm>
        expect(params.get('clientClass.unlocked')?.value).toBeTrue()
        expect((params.get('clientClass.values') as UntypedFormArray).length)?.toBe(2)
        expect(params.get('clientClass.values.0')?.value).toBe('foo')
        expect(params.get('clientClass.values.1')?.value).toBeFalsy()
        expect(params.get('requireClientClasses.unlocked')?.value).toBeTrue()
        expect((params.get('requireClientClasses.values') as UntypedFormArray)?.length).toBe(2)
        expect(params.get('requireClientClasses.values.0')?.value).toEqual(['foo', 'bar'])
        expect(params.get('requireClientClasses.values.1')?.value).toEqual([])
        expect(params.get('poolID.unlocked')?.value).toBeTrue()
        expect(params.get('poolID.values.0')?.value).toBe(60)
        expect(params.get('poolID.values.1')?.value).toBeFalsy()

        options = formArray.get('1.options.data') as UntypedFormArray
        expect(options.length).toBe(2)
        expect(options.get('0.0.optionFields.0.control')?.value).toBe('2001:db8:1::2')
        expect(options.get('1.0.optionFields.0.control')?.value).toBe('2001:db8:1::2')

        selectedDaemons = formArray.get('1.selectedDaemons') as FormControl<number[]>
        expect(selectedDaemons?.value).toEqual([1, 2])

        expect(formArray.get('2.prefixes.prefix')?.value).toBe('3002::/16')
        expect(formArray.get('2.prefixes.delegatedLength')?.value).toBe(112)
        expect(formArray.get('2.prefixes.excludedPrefix')?.value).toBeFalsy()

        params = formArray.get('2.parameters') as FormGroup<KeaPoolParametersForm>
        expect(params.get('clientClass.unlocked')?.value).toBeFalse()
        expect((params.get('clientClass.values') as UntypedFormArray).length)?.toBe(1)
        expect(params.get('clientClass.values.0')?.value).toBe('bar')
        expect(params.get('requireClientClasses.unlocked')?.value).toBeFalse()
        expect((params.get('requireClientClasses.values') as UntypedFormArray)?.length).toBe(1)
        expect(params.get('requireClientClasses.values.0')?.value).toEqual(['faz'])
        expect(params.get('poolID.unlocked')?.value).toBeFalse()
        expect(params.get('poolID.values.0')?.value).toBe(70)

        options = formArray.get('2.options.data') as UntypedFormArray
        expect(options.length).toBe(1)
        expect(options.get('0.0.optionFields.0.control')?.value).toBe('2001:db8:1::3')

        selectedDaemons = formArray.get('2.selectedDaemons') as FormControl<number[]>
        expect(selectedDaemons?.value).toEqual([1])
    })

    it('should convert form to prefix pool data', () => {
        const subnet: Subnet = {
            subnet: '2001:db8:1::/64',
            localSubnets: [
                {
                    daemonId: 1,
                    prefixDelegationPools: [
                        {
                            prefix: '3000::/16',
                            delegatedLength: 112,
                            excludedPrefix: '3000::ee00/120',
                            keaConfigPoolParameters: {
                                clientClass: 'foo',
                                poolID: 50,
                                requireClientClasses: ['foo', 'bar'],
                                options: [
                                    {
                                        alwaysSend: true,
                                        code: 23,
                                        encapsulate: '',
                                        fields: [
                                            {
                                                fieldType: 'ipv6-address',
                                                values: ['2001:db8:1::1'],
                                            },
                                        ],
                                        options: [],
                                        universe: 4,
                                    },
                                ],
                                optionsHash: '123',
                            },
                        },
                        {
                            prefix: '3001::/16',
                            delegatedLength: 112,
                            excludedPrefix: null,
                            keaConfigPoolParameters: {
                                clientClass: 'foo',
                                poolID: 60,
                                requireClientClasses: ['foo', 'bar'],
                                options: [
                                    {
                                        alwaysSend: true,
                                        code: 23,
                                        encapsulate: '',
                                        fields: [
                                            {
                                                fieldType: 'ipv6-address',
                                                values: ['2001:db8:1::2'],
                                            },
                                        ],
                                        options: [],
                                        universe: 4,
                                    },
                                ],
                                optionsHash: '234',
                            },
                        },
                        {
                            prefix: '3002::/16',
                            delegatedLength: 112,
                            excludedPrefix: null,
                            keaConfigPoolParameters: {
                                clientClass: 'bar',
                                poolID: 70,
                                requireClientClasses: ['faz'],
                                options: [
                                    {
                                        alwaysSend: true,
                                        code: 23,
                                        encapsulate: '',
                                        fields: [
                                            {
                                                fieldType: 'ipv6-address',
                                                values: ['2001:db8:1::3'],
                                            },
                                        ],
                                        options: [],
                                        universe: 4,
                                    },
                                ],
                                optionsHash: '345',
                            },
                        },
                    ],
                },
                {
                    daemonId: 2,
                    prefixDelegationPools: [
                        {
                            prefix: '3000::/16',
                            delegatedLength: 112,
                            excludedPrefix: '3000::ee00/120',
                            keaConfigPoolParameters: {
                                clientClass: 'foo',
                                poolID: 50,
                                requireClientClasses: ['foo', 'bar'],
                            },
                        },
                        {
                            prefix: '3001::/16',
                            delegatedLength: 112,
                            excludedPrefix: null,
                            keaConfigPoolParameters: {
                                options: [
                                    {
                                        alwaysSend: true,
                                        code: 23,
                                        encapsulate: '',
                                        fields: [
                                            {
                                                fieldType: 'ipv6-address',
                                                values: ['2001:db8:1::2'],
                                            },
                                        ],
                                        options: [],
                                        universe: 4,
                                    },
                                ],
                                optionsHash: '234',
                            },
                        },
                    ],
                },
            ],
        }
        const formArray = service.convertPrefixPoolsToForm(subnet)

        let pools = service.convertFormToPrefixPools([], subnet.localSubnets[0], formArray)
        expect(pools.length).toBe(3)
        expect(pools[0].prefix).toBe('3000::/16')
        expect(pools[0].delegatedLength).toBe(112)
        expect(pools[0].excludedPrefix).toBe('3000::ee00/120')
        expect(pools[0].keaConfigPoolParameters).toBeTruthy()
        expect(pools[0].keaConfigPoolParameters.clientClass).toBe('foo')
        expect(pools[0].keaConfigPoolParameters.requireClientClasses).toEqual(['foo', 'bar'])
        expect(pools[0].keaConfigPoolParameters.options?.length).toBe(1)
        expect(pools[0].keaConfigPoolParameters.options[0].alwaysSend).toBeTrue()
        expect(pools[0].keaConfigPoolParameters.options[0].code).toBe(23)
        expect(pools[0].keaConfigPoolParameters.options[0].fields?.length).toBe(1)
        expect(pools[0].keaConfigPoolParameters.options[0].fields[0].fieldType).toBe('ipv6-address')
        expect(pools[0].keaConfigPoolParameters.options[0].fields[0].values?.length).toBe(1)
        expect(pools[0].keaConfigPoolParameters.options[0].fields[0].values[0]).toBe('2001:db8:1::1')
        expect(pools[0].keaConfigPoolParameters.poolID).toBe(50)
        expect(pools[1].prefix).toBe('3001::/16')
        expect(pools[1].delegatedLength).toBe(112)
        expect(pools[1].excludedPrefix).toBeFalsy()
        expect(pools[1].keaConfigPoolParameters).toBeTruthy()
        expect(pools[1].keaConfigPoolParameters.clientClass).toBe('foo')
        expect(pools[1].keaConfigPoolParameters.requireClientClasses).toEqual(['foo', 'bar'])
        expect(pools[1].keaConfigPoolParameters.options?.length).toBe(1)
        expect(pools[1].keaConfigPoolParameters.options[0].alwaysSend).toBeTrue()
        expect(pools[1].keaConfigPoolParameters.options[0].code).toBe(23)
        expect(pools[1].keaConfigPoolParameters.options[0].fields?.length).toBe(1)
        expect(pools[1].keaConfigPoolParameters.options[0].fields[0].fieldType).toBe('ipv6-address')
        expect(pools[1].keaConfigPoolParameters.options[0].fields[0].values?.length).toBe(1)
        expect(pools[1].keaConfigPoolParameters.options[0].fields[0].values[0]).toBe('2001:db8:1::2')
        expect(pools[1].keaConfigPoolParameters.poolID).toBe(60)
        expect(pools[2].prefix).toBe('3002::/16')
        expect(pools[2].delegatedLength).toBe(112)
        expect(pools[2].excludedPrefix).toBeFalsy()
        expect(pools[2].keaConfigPoolParameters).toBeTruthy()
        expect(pools[2].keaConfigPoolParameters.clientClass).toBe('bar')
        expect(pools[2].keaConfigPoolParameters.requireClientClasses).toEqual(['faz'])
        expect(pools[2].keaConfigPoolParameters.options?.length).toBe(1)
        expect(pools[2].keaConfigPoolParameters.options[0].alwaysSend).toBeTrue()
        expect(pools[2].keaConfigPoolParameters.options[0].code).toBe(23)
        expect(pools[2].keaConfigPoolParameters.options[0].fields?.length).toBe(1)
        expect(pools[2].keaConfigPoolParameters.options[0].fields[0].fieldType).toBe('ipv6-address')
        expect(pools[2].keaConfigPoolParameters.options[0].fields[0].values?.length).toBe(1)
        expect(pools[2].keaConfigPoolParameters.options[0].fields[0].values[0]).toBe('2001:db8:1::3')
        expect(pools[2].keaConfigPoolParameters.poolID).toBe(70)

        pools = service.convertFormToPrefixPools([], subnet.localSubnets[1], formArray)
        expect(pools.length).toBe(2)
        expect(pools[0].prefix).toBe('3000::/16')
        expect(pools[0].delegatedLength).toBe(112)
        expect(pools[0].excludedPrefix).toBe('3000::ee00/120')
        expect(pools[0].keaConfigPoolParameters).toBeTruthy()
        expect(pools[0].keaConfigPoolParameters.clientClass).toBe('foo')
        expect(pools[0].keaConfigPoolParameters.requireClientClasses).toEqual(['foo', 'bar'])
        expect(pools[0].keaConfigPoolParameters.options).toEqual([])
        expect(pools[0].keaConfigPoolParameters.poolID).toBe(50)
        expect(pools[1].prefix).toBe('3001::/16')
        expect(pools[1].delegatedLength).toBe(112)
        expect(pools[1].excludedPrefix).toBeFalsy()
        expect(pools[1].keaConfigPoolParameters).toBeTruthy()
        expect(pools[1].keaConfigPoolParameters.clientClass).toBeFalsy()
        expect(pools[1].keaConfigPoolParameters.requireClientClasses).toEqual([])
        expect(pools[1].keaConfigPoolParameters.options?.length).toBe(1)
        expect(pools[1].keaConfigPoolParameters.options[0].alwaysSend).toBeTrue()
        expect(pools[1].keaConfigPoolParameters.options[0].code).toBe(23)
        expect(pools[1].keaConfigPoolParameters.options[0].fields?.length).toBe(1)
        expect(pools[1].keaConfigPoolParameters.options[0].fields[0].fieldType).toBe('ipv6-address')
        expect(pools[1].keaConfigPoolParameters.options[0].fields[0].values?.length).toBe(1)
        expect(pools[1].keaConfigPoolParameters.options[0].fields[0].values[0]).toBe('2001:db8:1::2')
        expect(pools[1].keaConfigPoolParameters.poolID).toBeFalsy()
    })

    it('should create a default form for pool parameters', () => {
        let form = service.createDefaultKeaPoolParametersForm()
        expect(Object.keys(form.controls).length).toBe(3)

        for (const key of Object.keys(form.controls)) {
            let control = form.get(key) as SharedParameterFormGroup<any>
            expect(control).toBeTruthy()
            expect(control.controls?.unlocked?.value).toBeFalse()
            expect(control.controls?.values?.value.length).toBe(1)
        }
    })

    it('should create a default form for an address pool', () => {
        const form = service.createDefaultAddressPoolForm('192.0.2.0/24')
        expect(form.get('range.start')?.value).toBe('')
        expect(form.get('range.end')?.value).toBe('')
        const parameters = form.get('parameters') as FormGroup<KeaPoolParametersForm>
        expect(parameters).toBeTruthy()
        expect(Object.keys(parameters.controls).length).toBe(3)
        expect(form.get('options')).toBeTruthy()
    })

    it('should create a default form for a prefix pool', () => {
        const form = service.createDefaultPrefixPoolForm()
        expect(form.get('prefixes.prefix')?.value).toBe('')
        expect(form.get('prefixes.delegatedLength')?.value).toBe(null)
        expect(form.get('prefixes.excludedPrefix')?.value).toBe('')
        const parameters = form.get('parameters') as FormGroup<KeaPoolParametersForm>
        expect(parameters).toBeTruthy()
        expect(Object.keys(parameters.controls).length).toBe(3)
        expect(form.get('options')).toBeTruthy()
    })

    it('should convert Kea IPv4 subnet parameters to a form group', () => {
        let parameters: KeaConfigSubnetDerivedParameters[] = [
            {
                cacheThreshold: 0.25,
                cacheMaxAge: 1000,
                clientClass: 'foo',
                requireClientClasses: ['foo', 'bar'],
                ddnsGeneratedPrefix: 'prefix1',
                ddnsOverrideClientUpdate: true,
                ddnsOverrideNoUpdate: true,
                ddnsQualifyingSuffix: 'suffix1',
                ddnsReplaceClientName: 'always',
                ddnsSendUpdates: true,
                ddnsUpdateOnRenew: true,
                ddnsUseConflictResolution: true,
                fourOverSixInterface: 'eth0',
                fourOverSixInterfaceID: 'foo',
                fourOverSixSubnet: '2001:db8:1::/64',
                hostnameCharSet: '[^A-Za-z0-9.-]',
                hostnameCharReplacement: 'x',
                preferredLifetime: 1000,
                minPreferredLifetime: 1000,
                maxPreferredLifetime: 1000,
                reservationsGlobal: true,
                reservationsInSubnet: true,
                reservationsOutOfPool: true,
                renewTimer: 500,
                rebindTimer: 1000,
                t1Percent: 0.45,
                t2Percent: 0.65,
                calculateTeeTimes: true,
                validLifetime: 1001,
                minValidLifetime: 999,
                maxValidLifetime: 1002,
                allocator: 'flq',
                authoritative: true,
                bootFileName: 'file1',
                _interface: 'eth0',
                interfaceID: 'foo',
                matchClientID: true,
                nextServer: '192.0.2.1',
                pdAllocator: 'flq',
                rapidCommit: true,
                serverHostname: 'foo.example.org.',
                storeExtendedInfo: true,
                relay: {
                    ipAddresses: ['192.0.2.1', '192.0.2.2', '192.0.2.3'],
                },
            },
            {
                cacheThreshold: 0.5,
                cacheMaxAge: 2000,
                clientClass: 'bar',
                requireClientClasses: ['foo', 'bar'],
                ddnsGeneratedPrefix: 'prefix2',
                ddnsOverrideClientUpdate: false,
                ddnsOverrideNoUpdate: false,
                ddnsQualifyingSuffix: 'suffix2',
                ddnsReplaceClientName: 'never',
                ddnsSendUpdates: false,
                ddnsUpdateOnRenew: false,
                ddnsUseConflictResolution: false,
                fourOverSixInterface: 'eth1',
                fourOverSixInterfaceID: 'bar',
                fourOverSixSubnet: '2001:db8:2::/64',
                hostnameCharSet: '[^A-Za-z.-]',
                hostnameCharReplacement: 'y',
                preferredLifetime: 2000,
                minPreferredLifetime: 1999,
                maxPreferredLifetime: 2001,
                reservationsGlobal: false,
                reservationsInSubnet: false,
                reservationsOutOfPool: false,
                renewTimer: 1500,
                rebindTimer: 2500,
                t1Percent: 0.55,
                t2Percent: 0.75,
                calculateTeeTimes: false,
                validLifetime: 2001,
                minValidLifetime: 2001,
                maxValidLifetime: 2001,
                allocator: 'random',
                authoritative: false,
                bootFileName: 'file2',
                _interface: 'eth1',
                interfaceID: 'bar',
                matchClientID: false,
                nextServer: '192.0.2.2',
                pdAllocator: 'random',
                rapidCommit: false,
                serverHostname: 'bar.example.org.',
                storeExtendedInfo: false,
                relay: {
                    ipAddresses: ['192.0.2.1', '192.0.2.2', '192.0.2.3'],
                },
            },
        ]
        let form = service.convertKeaSubnetParametersToForm(IPType.IPv4, null, 'subnet', parameters)
        let fg = form.get('cacheThreshold') as SharedParameterFormGroup<any>
        expect(fg).toBeTruthy()
        expect(fg.data.type).toBe('number')
        expect(fg.data.min).toBe(0)
        expect(fg.data.max).toBe(1)
        expect(fg.data.fractionDigits).toBe(2)
        expect(fg.data.values).toBeFalsy()
        expect((fg.get('unlocked') as UntypedFormControl)?.value).toBeTrue()
        expect((fg.get('values') as UntypedFormArray)?.controls.length).toBe(2)
        expect((fg.get('values') as UntypedFormArray)?.controls[0].value).toBe(0.25)
        expect((fg.get('values') as UntypedFormArray)?.controls[1].value).toBe(0.5)

        fg = form.get('cacheMaxAge') as SharedParameterFormGroup<any>
        expect(fg).toBeTruthy()
        expect(fg.data.type).toBe('number')
        expect(fg.data.min).toBeFalsy()
        expect(fg.data.max).toBeFalsy()
        expect(fg.data.fractionDigits).toBeFalsy()
        expect(fg.data.values).toBeFalsy()
        expect((fg.get('unlocked') as UntypedFormControl)?.value).toBeTrue()
        expect((fg.get('values') as UntypedFormArray)?.controls.length).toBe(2)
        expect((fg.get('values') as UntypedFormArray)?.controls[0].value).toBe(1000)
        expect((fg.get('values') as UntypedFormArray)?.controls[1].value).toBe(2000)

        fg = form.get('clientClass') as SharedParameterFormGroup<any>
        expect(fg).toBeTruthy()
        expect(fg.data.type).toBe('string')
        expect(fg.data.min).toBeFalsy()
        expect(fg.data.max).toBeFalsy()
        expect(fg.data.fractionDigits).toBeFalsy()
        expect(fg.data.values).toBeFalsy()
        expect((fg.get('unlocked') as UntypedFormControl)?.value).toBeTrue()
        expect((fg.get('values') as UntypedFormArray)?.controls.length).toBe(2)
        expect((fg.get('values') as UntypedFormArray)?.controls[0].value).toBe('foo')
        expect((fg.get('values') as UntypedFormArray)?.controls[1].value).toBe('bar')

        fg = form.get('requireClientClasses') as SharedParameterFormGroup<any>
        expect(fg).toBeTruthy()
        expect(fg.data.type).toBe('client-classes')
        expect(fg.data.min).toBeFalsy()
        expect(fg.data.max).toBeFalsy()
        expect(fg.data.fractionDigits).toBeFalsy()
        expect(fg.data.values).toBeFalsy()
        expect((fg.get('unlocked') as UntypedFormControl)?.value).toBeFalse()
        expect((fg.get('values') as UntypedFormArray)?.controls.length).toBe(2)
        expect((fg.get('values') as UntypedFormArray)?.controls[0].value).toEqual(['foo', 'bar'])
        expect((fg.get('values') as UntypedFormArray)?.controls[1].value).toEqual(['foo', 'bar'])

        fg = form.get('ddnsGeneratedPrefix') as SharedParameterFormGroup<any>
        expect(fg).toBeTruthy()
        expect(fg.data.type).toBe('string')
        expect(fg.data.min).toBeFalsy()
        expect(fg.data.max).toBeFalsy()
        expect(fg.data.fractionDigits).toBeFalsy()
        expect(fg.data.values).toBeFalsy()
        expect(fg.data.invalidText).toBe('Please specify a valid prefix.')
        expect((fg.get('unlocked') as UntypedFormControl)?.value).toBeTrue()
        expect((fg.get('values') as UntypedFormArray)?.controls.length).toBe(2)
        expect((fg.get('values') as UntypedFormArray)?.controls[0].value).toBe('prefix1')
        expect((fg.get('values') as UntypedFormArray)?.controls[1].value).toBe('prefix2')

        fg = form.get('ddnsOverrideClientUpdate') as SharedParameterFormGroup<any>
        expect(fg).toBeTruthy()
        expect(fg.data.type).toBe('boolean')
        expect(fg.data.min).toBeFalsy()
        expect(fg.data.max).toBeFalsy()
        expect(fg.data.fractionDigits).toBeFalsy()
        expect(fg.data.values).toBeFalsy()
        expect(fg.data.invalidText).toBeFalsy()
        expect((fg.get('unlocked') as UntypedFormControl)?.value).toBeTrue()
        expect((fg.get('values') as UntypedFormArray)?.controls.length).toBe(2)
        expect((fg.get('values') as UntypedFormArray)?.controls[0].value).toBeTrue()
        expect((fg.get('values') as UntypedFormArray)?.controls[1].value).toBeFalse()

        fg = form.get('ddnsOverrideNoUpdate') as SharedParameterFormGroup<any>
        expect(fg).toBeTruthy()
        expect(fg.data.type).toBe('boolean')
        expect(fg.data.min).toBeFalsy()
        expect(fg.data.max).toBeFalsy()
        expect(fg.data.fractionDigits).toBeFalsy()
        expect(fg.data.values).toBeFalsy()
        expect(fg.data.invalidText).toBeFalsy()
        expect((fg.get('unlocked') as UntypedFormControl)?.value).toBeTrue()
        expect((fg.get('values') as UntypedFormArray)?.controls.length).toBe(2)
        expect((fg.get('values') as UntypedFormArray)?.controls[0].value).toBeTrue()
        expect((fg.get('values') as UntypedFormArray)?.controls[1].value).toBeFalse()

        fg = form.get('ddnsQualifyingSuffix') as SharedParameterFormGroup<any>
        expect(fg).toBeTruthy()
        expect(fg.data.type).toBe('string')
        expect(fg.data.min).toBeFalsy()
        expect(fg.data.max).toBeFalsy()
        expect(fg.data.fractionDigits).toBeFalsy()
        expect(fg.data.values).toBeFalsy()
        expect(fg.data.invalidText).toBe('Please specify a valid suffix.')
        expect((fg.get('unlocked') as UntypedFormControl)?.value).toBeTrue()
        expect((fg.get('values') as UntypedFormArray)?.controls.length).toBe(2)
        expect((fg.get('values') as UntypedFormArray)?.controls[0].value).toBe('suffix1')
        expect((fg.get('values') as UntypedFormArray)?.controls[1].value).toBe('suffix2')

        fg = form.get('ddnsReplaceClientName') as SharedParameterFormGroup<any>
        expect(fg).toBeTruthy()
        expect(fg.data.type).toBe('string')
        expect(fg.data.min).toBeFalsy()
        expect(fg.data.max).toBeFalsy()
        expect(fg.data.fractionDigits).toBeFalsy()
        expect(fg.data.values?.length).toBe(3)
        expect(fg.data.values[0]).toBe('never')
        expect(fg.data.values[1]).toBe('always')
        expect(fg.data.values[2]).toBe('when-not-present')
        expect(fg.data.invalidText).toBeFalsy
        expect((fg.get('unlocked') as UntypedFormControl)?.value).toBeTrue()
        expect((fg.get('values') as UntypedFormArray)?.controls.length).toBe(2)
        expect((fg.get('values') as UntypedFormArray)?.controls[0].value).toBe('always')
        expect((fg.get('values') as UntypedFormArray)?.controls[1].value).toBe('never')

        fg = form.get('ddnsSendUpdates') as SharedParameterFormGroup<any>
        expect(fg).toBeTruthy()
        expect(fg.data.type).toBe('boolean')
        expect(fg.data.min).toBeFalsy()
        expect(fg.data.max).toBeFalsy()
        expect(fg.data.fractionDigits).toBeFalsy()
        expect(fg.data.values).toBeFalsy()
        expect(fg.data.invalidText).toBeFalsy()
        expect((fg.get('unlocked') as UntypedFormControl)?.value).toBeTrue()
        expect((fg.get('values') as UntypedFormArray)?.controls.length).toBe(2)
        expect((fg.get('values') as UntypedFormArray)?.controls[0].value).toBeTrue()
        expect((fg.get('values') as UntypedFormArray)?.controls[1].value).toBeFalse()

        fg = form.get('ddnsUpdateOnRenew') as SharedParameterFormGroup<any>
        expect(fg).toBeTruthy()
        expect(fg.data.type).toBe('boolean')
        expect(fg.data.min).toBeFalsy()
        expect(fg.data.max).toBeFalsy()
        expect(fg.data.fractionDigits).toBeFalsy()
        expect(fg.data.values).toBeFalsy()
        expect(fg.data.invalidText).toBeFalsy()
        expect((fg.get('unlocked') as UntypedFormControl)?.value).toBeTrue()
        expect((fg.get('values') as UntypedFormArray)?.controls.length).toBe(2)
        expect((fg.get('values') as UntypedFormArray)?.controls[0].value).toBeTrue()
        expect((fg.get('values') as UntypedFormArray)?.controls[1].value).toBeFalse()

        fg = form.get('ddnsUseConflictResolution') as SharedParameterFormGroup<any>
        expect(fg).toBeTruthy()
        expect(fg.data.type).toBe('boolean')
        expect(fg.data.min).toBeFalsy()
        expect(fg.data.max).toBeFalsy()
        expect(fg.data.fractionDigits).toBeFalsy()
        expect(fg.data.values).toBeFalsy()
        expect(fg.data.invalidText).toBeFalsy()
        expect((fg.get('unlocked') as UntypedFormControl)?.value).toBeTrue()
        expect((fg.get('values') as UntypedFormArray)?.controls.length).toBe(2)
        expect((fg.get('values') as UntypedFormArray)?.controls[0].value).toBeTrue()
        expect((fg.get('values') as UntypedFormArray)?.controls[1].value).toBeFalse()

        fg = form.get('fourOverSixInterface') as SharedParameterFormGroup<any>
        expect(fg).toBeTruthy()
        expect(fg.data.type).toBe('string')
        expect(fg.data.min).toBeFalsy()
        expect(fg.data.max).toBeFalsy()
        expect(fg.data.fractionDigits).toBeFalsy()
        expect(fg.data.values).toBeFalsy()
        expect(fg.data.invalidText).toBeFalsy()
        expect((fg.get('unlocked') as UntypedFormControl)?.value).toBeTrue()
        expect((fg.get('values') as UntypedFormArray)?.controls.length).toBe(2)
        expect((fg.get('values') as UntypedFormArray)?.controls[0].value).toBe('eth0')
        expect((fg.get('values') as UntypedFormArray)?.controls[1].value).toBe('eth1')

        fg = form.get('fourOverSixInterfaceID') as SharedParameterFormGroup<any>
        expect(fg).toBeTruthy()
        expect(fg.data.type).toBe('string')
        expect(fg.data.min).toBeFalsy()
        expect(fg.data.max).toBeFalsy()
        expect(fg.data.fractionDigits).toBeFalsy()
        expect(fg.data.values).toBeFalsy()
        expect(fg.data.invalidText).toBeFalsy()
        expect((fg.get('unlocked') as UntypedFormControl)?.value).toBeTrue()
        expect((fg.get('values') as UntypedFormArray)?.controls.length).toBe(2)
        expect((fg.get('values') as UntypedFormArray)?.controls[0].value).toBe('foo')
        expect((fg.get('values') as UntypedFormArray)?.controls[1].value).toBe('bar')

        fg = form.get('fourOverSixSubnet') as SharedParameterFormGroup<any>
        expect(fg).toBeTruthy()
        expect(fg.data.type).toBe('string')
        expect(fg.data.min).toBeFalsy()
        expect(fg.data.max).toBeFalsy()
        expect(fg.data.fractionDigits).toBeFalsy()
        expect(fg.data.values).toBeFalsy()
        expect(fg.data.invalidText).toBeFalsy()
        expect((fg.get('unlocked') as UntypedFormControl)?.value).toBeTrue()
        expect((fg.get('values') as UntypedFormArray)?.controls.length).toBe(2)
        expect((fg.get('values') as UntypedFormArray)?.controls[0].value).toBe('2001:db8:1::/64')
        expect((fg.get('values') as UntypedFormArray)?.controls[1].value).toBe('2001:db8:2::/64')

        fg = form.get('hostnameCharSet') as SharedParameterFormGroup<any>
        expect(fg).toBeTruthy()
        expect(fg.data.type).toBe('string')
        expect(fg.data.min).toBeFalsy()
        expect(fg.data.max).toBeFalsy()
        expect(fg.data.fractionDigits).toBeFalsy()
        expect(fg.data.values).toBeFalsy()
        expect(fg.data.invalidText).toBeFalsy()
        expect((fg.get('unlocked') as UntypedFormControl)?.value).toBeTrue()
        expect((fg.get('values') as UntypedFormArray)?.controls.length).toBe(2)
        expect((fg.get('values') as UntypedFormArray)?.controls[0].value).toBe('[^A-Za-z0-9.-]')
        expect((fg.get('values') as UntypedFormArray)?.controls[1].value).toBe('[^A-Za-z.-]')

        fg = form.get('hostnameCharReplacement') as SharedParameterFormGroup<any>
        expect(fg).toBeTruthy()
        expect(fg.data.type).toBe('string')
        expect(fg.data.min).toBeFalsy()
        expect(fg.data.max).toBeFalsy()
        expect(fg.data.fractionDigits).toBeFalsy()
        expect(fg.data.values).toBeFalsy()
        expect(fg.data.invalidText).toBeFalsy()
        expect((fg.get('unlocked') as UntypedFormControl)?.value).toBeTrue()
        expect((fg.get('values') as UntypedFormArray)?.controls.length).toBe(2)
        expect((fg.get('values') as UntypedFormArray)?.controls[0].value).toBe('x')
        expect((fg.get('values') as UntypedFormArray)?.controls[1].value).toBe('y')

        fg = form.get('preferredLifetime') as SharedParameterFormGroup<any>
        expect(fg).toBeFalsy()

        fg = form.get('minPreferredLifetime') as SharedParameterFormGroup<any>
        expect(fg).toBeFalsy()

        fg = form.get('maxPreferredLifetime') as SharedParameterFormGroup<any>
        expect(fg).toBeFalsy()

        fg = form.get('reservationsGlobal') as SharedParameterFormGroup<any>
        expect(fg).toBeTruthy()
        expect(fg.data.type).toBe('boolean')
        expect(fg.data.min).toBeFalsy()
        expect(fg.data.max).toBeFalsy()
        expect(fg.data.fractionDigits).toBeFalsy()
        expect(fg.data.values).toBeFalsy()
        expect(fg.data.invalidText).toBeFalsy()
        expect((fg.get('unlocked') as UntypedFormControl)?.value).toBeTrue()
        expect((fg.get('values') as UntypedFormArray)?.controls.length).toBe(2)
        expect((fg.get('values') as UntypedFormArray)?.controls[0].value).toBeTrue()
        expect((fg.get('values') as UntypedFormArray)?.controls[1].value).toBeFalse()

        fg = form.get('reservationsInSubnet') as SharedParameterFormGroup<any>
        expect(fg).toBeTruthy()
        expect(fg.data.type).toBe('boolean')
        expect(fg.data.min).toBeFalsy()
        expect(fg.data.max).toBeFalsy()
        expect(fg.data.fractionDigits).toBeFalsy()
        expect(fg.data.values).toBeFalsy()
        expect(fg.data.invalidText).toBeFalsy()
        expect((fg.get('unlocked') as UntypedFormControl)?.value).toBeTrue()
        expect((fg.get('values') as UntypedFormArray)?.controls.length).toBe(2)
        expect((fg.get('values') as UntypedFormArray)?.controls[0].value).toBeTrue()
        expect((fg.get('values') as UntypedFormArray)?.controls[1].value).toBeFalse()

        fg = form.get('reservationsOutOfPool') as SharedParameterFormGroup<any>
        expect(fg).toBeTruthy()
        expect(fg.data.type).toBe('boolean')
        expect(fg.data.min).toBeFalsy()
        expect(fg.data.max).toBeFalsy()
        expect(fg.data.fractionDigits).toBeFalsy()
        expect(fg.data.values).toBeFalsy()
        expect(fg.data.invalidText).toBeFalsy()
        expect((fg.get('unlocked') as UntypedFormControl)?.value).toBeTrue()
        expect((fg.get('values') as UntypedFormArray)?.controls.length).toBe(2)
        expect((fg.get('values') as UntypedFormArray)?.controls[0].value).toBeTrue()
        expect((fg.get('values') as UntypedFormArray)?.controls[1].value).toBeFalse()

        fg = form.get('renewTimer') as SharedParameterFormGroup<any>
        expect(fg).toBeTruthy()
        expect(fg.data.type).toBe('number')
        expect(fg.data.min).toBeFalsy()
        expect(fg.data.max).toBeFalsy()
        expect(fg.data.fractionDigits).toBeFalsy()
        expect(fg.data.values).toBeFalsy()
        expect((fg.get('unlocked') as UntypedFormControl)?.value).toBeTrue()
        expect((fg.get('values') as UntypedFormArray)?.controls.length).toBe(2)
        expect((fg.get('values') as UntypedFormArray)?.controls[0].value).toBe(500)
        expect((fg.get('values') as UntypedFormArray)?.controls[1].value).toBe(1500)

        fg = form.get('rebindTimer') as SharedParameterFormGroup<any>
        expect(fg).toBeTruthy()
        expect(fg.data.type).toBe('number')
        expect(fg.data.min).toBeFalsy()
        expect(fg.data.max).toBeFalsy()
        expect(fg.data.fractionDigits).toBeFalsy()
        expect(fg.data.values).toBeFalsy()
        expect((fg.get('unlocked') as UntypedFormControl)?.value).toBeTrue()
        expect((fg.get('values') as UntypedFormArray)?.controls.length).toBe(2)
        expect((fg.get('values') as UntypedFormArray)?.controls[0].value).toBe(1000)
        expect((fg.get('values') as UntypedFormArray)?.controls[1].value).toBe(2500)

        fg = form.get('t1Percent') as SharedParameterFormGroup<any>
        expect(fg).toBeTruthy()
        expect(fg.data.type).toBe('number')
        expect(fg.data.min).toBe(0)
        expect(fg.data.max).toBe(1)
        expect(fg.data.fractionDigits).toBe(2)
        expect(fg.data.values).toBeFalsy()
        expect((fg.get('unlocked') as UntypedFormControl)?.value).toBeTrue()
        expect((fg.get('values') as UntypedFormArray)?.controls.length).toBe(2)
        expect((fg.get('values') as UntypedFormArray)?.controls[0].value).toBe(0.45)
        expect((fg.get('values') as UntypedFormArray)?.controls[1].value).toBe(0.55)

        fg = form.get('t2Percent') as SharedParameterFormGroup<any>
        expect(fg).toBeTruthy()
        expect(fg.data.type).toBe('number')
        expect(fg.data.min).toBe(0)
        expect(fg.data.max).toBe(1)
        expect(fg.data.fractionDigits).toBe(2)
        expect(fg.data.values).toBeFalsy()
        expect((fg.get('unlocked') as UntypedFormControl)?.value).toBeTrue()
        expect((fg.get('values') as UntypedFormArray)?.controls.length).toBe(2)
        expect((fg.get('values') as UntypedFormArray)?.controls[0].value).toBe(0.65)
        expect((fg.get('values') as UntypedFormArray)?.controls[1].value).toBe(0.75)

        fg = form.get('calculateTeeTimes') as SharedParameterFormGroup<any>
        expect(fg).toBeTruthy()
        expect(fg.data.type).toBe('boolean')
        expect(fg.data.min).toBeFalsy()
        expect(fg.data.max).toBeFalsy()
        expect(fg.data.fractionDigits).toBeFalsy()
        expect(fg.data.values).toBeFalsy()
        expect(fg.data.invalidText).toBeFalsy()
        expect((fg.get('unlocked') as UntypedFormControl)?.value).toBeTrue()
        expect((fg.get('values') as UntypedFormArray)?.controls.length).toBe(2)
        expect((fg.get('values') as UntypedFormArray)?.controls[0].value).toBeTrue()
        expect((fg.get('values') as UntypedFormArray)?.controls[1].value).toBeFalse()

        fg = form.get('validLifetime') as SharedParameterFormGroup<any>
        expect(fg).toBeTruthy()
        expect(fg.data.type).toBe('number')
        expect(fg.data.min).toBeFalsy()
        expect(fg.data.max).toBeFalsy()
        expect(fg.data.fractionDigits).toBeFalsy()
        expect(fg.data.values).toBeFalsy()
        expect((fg.get('unlocked') as UntypedFormControl)?.value).toBeTrue()
        expect((fg.get('values') as UntypedFormArray)?.controls.length).toBe(2)
        expect((fg.get('values') as UntypedFormArray)?.controls[0].value).toBe(1001)
        expect((fg.get('values') as UntypedFormArray)?.controls[1].value).toBe(2001)

        fg = form.get('minValidLifetime') as SharedParameterFormGroup<any>
        expect(fg).toBeTruthy()
        expect(fg.data.type).toBe('number')
        expect(fg.data.min).toBeFalsy()
        expect(fg.data.max).toBeFalsy()
        expect(fg.data.fractionDigits).toBeFalsy()
        expect(fg.data.values).toBeFalsy()
        expect((fg.get('unlocked') as UntypedFormControl)?.value).toBeTrue()
        expect((fg.get('values') as UntypedFormArray)?.controls.length).toBe(2)
        expect((fg.get('values') as UntypedFormArray)?.controls[0].value).toBe(999)
        expect((fg.get('values') as UntypedFormArray)?.controls[1].value).toBe(2001)

        fg = form.get('maxValidLifetime') as SharedParameterFormGroup<any>
        expect(fg).toBeTruthy()
        expect(fg.data.type).toBe('number')
        expect(fg.data.min).toBeFalsy()
        expect(fg.data.max).toBeFalsy()
        expect(fg.data.fractionDigits).toBeFalsy()
        expect(fg.data.values).toBeFalsy()
        expect((fg.get('unlocked') as UntypedFormControl)?.value).toBeTrue()
        expect((fg.get('values') as UntypedFormArray)?.controls.length).toBe(2)
        expect((fg.get('values') as UntypedFormArray)?.controls[0].value).toBe(1002)
        expect((fg.get('values') as UntypedFormArray)?.controls[1].value).toBe(2001)

        fg = form.get('allocator') as SharedParameterFormGroup<any>
        expect(fg).toBeTruthy()
        expect(fg.data.type).toBe('string')
        expect(fg.data.min).toBeFalsy()
        expect(fg.data.max).toBeFalsy()
        expect(fg.data.fractionDigits).toBeFalsy()
        expect(fg.data.values?.length).toBe(3)
        expect(fg.data.invalidText).toBeFalsy()
        expect((fg.get('unlocked') as UntypedFormControl)?.value).toBeTrue()
        expect((fg.get('values') as UntypedFormArray)?.controls.length).toBe(2)
        expect((fg.get('values') as UntypedFormArray)?.controls[0].value).toBe('flq')
        expect((fg.get('values') as UntypedFormArray)?.controls[1].value).toBe('random')

        fg = form.get('authoritative') as SharedParameterFormGroup<any>
        expect(fg).toBeTruthy()
        expect(fg.data.type).toBe('boolean')
        expect(fg.data.min).toBeFalsy()
        expect(fg.data.max).toBeFalsy()
        expect(fg.data.fractionDigits).toBeFalsy()
        expect(fg.data.values).toBeFalsy()
        expect(fg.data.invalidText).toBeFalsy()
        expect((fg.get('unlocked') as UntypedFormControl)?.value).toBeTrue()
        expect((fg.get('values') as UntypedFormArray)?.controls.length).toBe(2)
        expect((fg.get('values') as UntypedFormArray)?.controls[0].value).toBeTrue()
        expect((fg.get('values') as UntypedFormArray)?.controls[1].value).toBeFalse()

        fg = form.get('bootFileName') as SharedParameterFormGroup<any>
        expect(fg).toBeTruthy()
        expect(fg.data.type).toBe('string')
        expect(fg.data.min).toBeFalsy()
        expect(fg.data.max).toBeFalsy()
        expect(fg.data.fractionDigits).toBeFalsy()
        expect(fg.data.values).toBeFalsy()
        expect(fg.data.invalidText).toBeFalsy()
        expect((fg.get('unlocked') as UntypedFormControl)?.value).toBeTrue()
        expect((fg.get('values') as UntypedFormArray)?.controls.length).toBe(2)
        expect((fg.get('values') as UntypedFormArray)?.controls[0].value).toBe('file1')
        expect((fg.get('values') as UntypedFormArray)?.controls[1].value).toBe('file2')

        fg = form.get('interface') as SharedParameterFormGroup<any>
        expect(fg).toBeTruthy()
        expect(fg.data.type).toBe('string')
        expect(fg.data.min).toBeFalsy()
        expect(fg.data.max).toBeFalsy()
        expect(fg.data.fractionDigits).toBeFalsy()
        expect(fg.data.values).toBeFalsy()
        expect(fg.data.invalidText).toBeFalsy()
        expect((fg.get('unlocked') as UntypedFormControl)?.value).toBeTrue()
        expect((fg.get('values') as UntypedFormArray)?.controls.length).toBe(2)
        expect((fg.get('values') as UntypedFormArray)?.controls[0].value).toBe('eth0')
        expect((fg.get('values') as UntypedFormArray)?.controls[1].value).toBe('eth1')

        fg = form.get('interfaceID') as SharedParameterFormGroup<any>
        expect(fg).toBeTruthy()
        expect(fg.data.type).toBe('string')
        expect(fg.data.min).toBeFalsy()
        expect(fg.data.max).toBeFalsy()
        expect(fg.data.fractionDigits).toBeFalsy()
        expect(fg.data.values).toBeFalsy()
        expect(fg.data.invalidText).toBeFalsy()
        expect((fg.get('unlocked') as UntypedFormControl)?.value).toBeTrue()
        expect((fg.get('values') as UntypedFormArray)?.controls.length).toBe(2)
        expect((fg.get('values') as UntypedFormArray)?.controls[0].value).toBe('foo')
        expect((fg.get('values') as UntypedFormArray)?.controls[1].value).toBe('bar')

        fg = form.get('matchClientID') as SharedParameterFormGroup<any>
        expect(fg).toBeTruthy()
        expect(fg.data.type).toBe('boolean')
        expect(fg.data.min).toBeFalsy()
        expect(fg.data.max).toBeFalsy()
        expect(fg.data.fractionDigits).toBeFalsy()
        expect(fg.data.values).toBeFalsy()
        expect(fg.data.invalidText).toBeFalsy()
        expect((fg.get('unlocked') as UntypedFormControl)?.value).toBeTrue()
        expect((fg.get('values') as UntypedFormArray)?.controls.length).toBe(2)
        expect((fg.get('values') as UntypedFormArray)?.controls[0].value).toBeTrue()
        expect((fg.get('values') as UntypedFormArray)?.controls[1].value).toBeFalse()

        fg = form.get('nextServer') as SharedParameterFormGroup<any>
        expect(fg).toBeTruthy()
        expect(fg.data.type).toBe('string')
        expect(fg.data.min).toBeFalsy()
        expect(fg.data.max).toBeFalsy()
        expect(fg.data.fractionDigits).toBeFalsy()
        expect(fg.data.values).toBeFalsy()
        expect(fg.data.invalidText).toBe('Please specify an IPv4 address.')
        expect((fg.get('unlocked') as UntypedFormControl)?.value).toBeTrue()
        expect((fg.get('values') as UntypedFormArray)?.controls.length).toBe(2)
        expect((fg.get('values') as UntypedFormArray)?.controls[0].value).toBe('192.0.2.1')
        expect((fg.get('values') as UntypedFormArray)?.controls[1].value).toBe('192.0.2.2')

        fg = form.get('pdAllocator') as SharedParameterFormGroup<any>
        expect(fg).toBeFalsy()

        fg = form.get('rapidCommit') as SharedParameterFormGroup<any>
        expect(fg).toBeFalsy()

        fg = form.get('serverHostname') as SharedParameterFormGroup<any>
        expect(fg).toBeTruthy()
        expect(fg.data.type).toBe('string')
        expect(fg.data.min).toBeFalsy()
        expect(fg.data.max).toBeFalsy()
        expect(fg.data.fractionDigits).toBeFalsy()
        expect(fg.data.values).toBeFalsy()
        expect(fg.data.invalidText).toBe('Please specify a valid hostname.')
        expect((fg.get('unlocked') as UntypedFormControl)?.value).toBeTrue()
        expect((fg.get('values') as UntypedFormArray)?.controls.length).toBe(2)
        expect((fg.get('values') as UntypedFormArray)?.controls[0].value).toBe('foo.example.org.')
        expect((fg.get('values') as UntypedFormArray)?.controls[1].value).toBe('bar.example.org.')

        fg = form.get('storeExtendedInfo') as SharedParameterFormGroup<any>
        expect(fg).toBeTruthy()
        expect(fg.data.type).toBe('boolean')
        expect(fg.data.min).toBeFalsy()
        expect(fg.data.max).toBeFalsy()
        expect(fg.data.fractionDigits).toBeFalsy()
        expect(fg.data.values).toBeFalsy()
        expect(fg.data.invalidText).toBeFalsy()
        expect((fg.get('unlocked') as UntypedFormControl)?.value).toBeTrue()
        expect((fg.get('values') as UntypedFormArray)?.controls.length).toBe(2)
        expect((fg.get('values') as UntypedFormArray)?.controls[0].value).toBeTrue()
        expect((fg.get('values') as UntypedFormArray)?.controls[1].value).toBeFalse()
    })

    it('it should exclude 4o6 parameters from a shared network', () => {
        let parameters: KeaConfigSubnetDerivedParameters[] = [
            {
                clientClass: 'bar',
                fourOverSixInterface: 'eth0',
                fourOverSixInterfaceID: 'foo',
                fourOverSixSubnet: '2001:db8:1::/64',
            },
        ]
        let form = service.convertKeaSubnetParametersToForm(IPType.IPv4, null, 'shared-network', parameters)
        let fg = form.get('clientClass') as SharedParameterFormGroup<any>
        expect(fg).toBeTruthy()
        expect(fg.data.type).toBe('string')
        expect(fg.data.min).toBeFalsy()
        expect(fg.data.max).toBeFalsy()
        expect(fg.data.fractionDigits).toBeFalsy()
        expect(fg.data.values).toBeFalsy()
        expect((fg.get('unlocked') as UntypedFormControl)?.value).toBeFalse()
        expect((fg.get('values') as UntypedFormArray)?.controls.length).toBe(1)
        expect((fg.get('values') as UntypedFormArray)?.controls[0].value).toBe('bar')

        // These should be excluded for a shared network.
        expect(form.contains('fourOverSixInterface')).toBeFalse()
        expect(form.contains('fourOverSixInterfaceID')).toBeFalse()
        expect(form.contains('fourOverSixSubnet')).toBeFalse()
    })

    it('should convert Kea IPv6 subnet parameters to a form group', () => {
        let parameters: KeaConfigSubnetDerivedParameters[] = [
            {
                fourOverSixInterface: 'eth0',
                fourOverSixInterfaceID: 'foo',
                fourOverSixSubnet: '2001:db8:1::/64',
                preferredLifetime: 1000,
                minPreferredLifetime: 1000,
                maxPreferredLifetime: 1000,
                bootFileName: 'file1',
                matchClientID: true,
                nextServer: '192.0.2.1',
                pdAllocator: 'flq',
                rapidCommit: true,
                serverHostname: 'foo.example.org.',
            },
        ]
        let form = service.convertKeaSubnetParametersToForm(IPType.IPv6, null, 'subnet', parameters)
        let fg = form.get('fourOverSixInterface') as SharedParameterFormGroup<any>
        expect(fg).toBeFalsy()

        fg = form.get('fourOverSixInterfaceID') as SharedParameterFormGroup<any>
        expect(fg).toBeFalsy()

        fg = form.get('fourOverSixSubnet') as SharedParameterFormGroup<any>
        expect(fg).toBeFalsy()

        fg = form.get('preferredLifetime') as SharedParameterFormGroup<any>
        expect(fg).toBeTruthy()
        expect(fg.data.type).toBe('number')
        expect(fg.data.min).toBeFalsy()
        expect(fg.data.max).toBeFalsy()
        expect(fg.data.fractionDigits).toBeFalsy()
        expect(fg.data.values).toBeFalsy()
        expect((fg.get('unlocked') as UntypedFormControl)?.value).toBeFalse()
        expect((fg.get('values') as UntypedFormArray)?.controls.length).toBe(1)
        expect((fg.get('values') as UntypedFormArray)?.controls[0].value).toBe(1000)

        fg = form.get('minPreferredLifetime') as SharedParameterFormGroup<any>
        expect(fg).toBeTruthy()
        expect(fg.data.type).toBe('number')
        expect(fg.data.min).toBeFalsy()
        expect(fg.data.max).toBeFalsy()
        expect(fg.data.fractionDigits).toBeFalsy()
        expect(fg.data.values).toBeFalsy()
        expect((fg.get('unlocked') as UntypedFormControl)?.value).toBeFalse()
        expect((fg.get('values') as UntypedFormArray)?.controls.length).toBe(1)
        expect((fg.get('values') as UntypedFormArray)?.controls[0].value).toBe(1000)

        fg = form.get('maxPreferredLifetime') as SharedParameterFormGroup<any>
        expect(fg).toBeTruthy()
        expect(fg.data.type).toBe('number')
        expect(fg.data.min).toBeFalsy()
        expect(fg.data.max).toBeFalsy()
        expect(fg.data.fractionDigits).toBeFalsy()
        expect(fg.data.values).toBeFalsy()
        expect((fg.get('unlocked') as UntypedFormControl)?.value).toBeFalse()
        expect((fg.get('values') as UntypedFormArray)?.controls.length).toBe(1)
        expect((fg.get('values') as UntypedFormArray)?.controls[0].value).toBe(1000)

        fg = form.get('bootFileName') as SharedParameterFormGroup<any>
        expect(fg).toBeFalsy()

        fg = form.get('matchClientID') as SharedParameterFormGroup<any>
        expect(fg).toBeFalsy()

        fg = form.get('nextServer') as SharedParameterFormGroup<any>
        expect(fg).toBeFalsy()

        fg = form.get('pdAllocator') as SharedParameterFormGroup<any>
        expect(fg).toBeTruthy()
        expect(fg.data.type).toBe('string')
        expect(fg.data.min).toBeFalsy()
        expect(fg.data.max).toBeFalsy()
        expect(fg.data.fractionDigits).toBeFalsy()
        expect(fg.data.values?.length).toBe(3)
        expect(fg.data.invalidText).toBeFalsy()
        expect((fg.get('unlocked') as UntypedFormControl)?.value).toBeFalse()
        expect((fg.get('values') as UntypedFormArray)?.controls.length).toBe(1)
        expect((fg.get('values') as UntypedFormArray)?.controls[0].value).toBe('flq')

        fg = form.get('rapidCommit') as SharedParameterFormGroup<any>
        expect(fg).toBeTruthy()
        expect(fg.data.type).toBe('boolean')
        expect(fg.data.min).toBeFalsy()
        expect(fg.data.max).toBeFalsy()
        expect(fg.data.fractionDigits).toBeFalsy()
        expect(fg.data.values).toBeFalsy()
        expect(fg.data.invalidText).toBeFalsy()
        expect((fg.get('unlocked') as UntypedFormControl)?.value).toBeFalse()
        expect((fg.get('values') as UntypedFormArray)?.controls.length).toBe(1)
        expect((fg.get('values') as UntypedFormArray)?.controls[0].value).toBeTrue()
    })

    it('should use a validator for generated prefix', () => {
        let parameters: KeaConfigSubnetDerivedParameters[] = [
            {
                ddnsGeneratedPrefix: '-invalid.prefix',
            },
        ]
        let form = service.convertKeaSubnetParametersToForm(IPType.IPv4, null, 'subnet', parameters)
        let fg = form.get('ddnsGeneratedPrefix') as SharedParameterFormGroup<any>
        expect(fg).toBeTruthy()
        expect(fg.valid).toBeFalse()
    })

    it('should use a validator for qualifying suffix', () => {
        let parameters: KeaConfigSubnetDerivedParameters[] = [
            {
                ddnsQualifyingSuffix: '123',
            },
        ]
        let form = service.convertKeaSubnetParametersToForm(IPType.IPv4, null, 'subnet', parameters)
        let fg = form.get('ddnsQualifyingSuffix') as SharedParameterFormGroup<any>
        expect(fg).toBeTruthy()
        expect(fg.valid).toBeFalse()
    })

    it('should use a validator for 4o6 subnet', () => {
        let parameters: KeaConfigSubnetDerivedParameters[] = [
            {
                fourOverSixSubnet: '2001:db8:1::',
            },
        ]
        let form = service.convertKeaSubnetParametersToForm(IPType.IPv4, null, 'subnet', parameters)
        let fg = form.get('fourOverSixSubnet') as SharedParameterFormGroup<any>
        expect(fg).toBeTruthy()
        expect(fg.valid).toBeFalse()
    })

    it('should use a validator for next server', () => {
        let parameters: KeaConfigSubnetDerivedParameters[] = [
            {
                nextServer: '1.1.2.',
            },
        ]
        let form = service.convertKeaSubnetParametersToForm(IPType.IPv4, null, 'subnet', parameters)
        let fg = form.get('nextServer') as SharedParameterFormGroup<any>
        expect(fg).toBeTruthy()
        expect(fg.valid).toBeFalse()
    })

    it('should use a validator for server hostname', () => {
        let parameters: KeaConfigSubnetDerivedParameters[] = [
            {
                serverHostname: 'abc..foo',
            },
        ]
        let form = service.convertKeaSubnetParametersToForm(IPType.IPv4, null, 'subnet', parameters)
        let fg = form.get('serverHostname') as SharedParameterFormGroup<any>
        expect(fg).toBeTruthy()
        expect(fg.valid).toBeFalse()
    })

    it('should create a default Kea parameters form for an IPv4 subnet', () => {
        let form = service.createDefaultKeaSharedNetworkParametersForm(IPType.IPv4, null)
        expect(Object.keys(form.controls).length).toBe(36)

        for (const key of Object.keys(form.controls)) {
            let control = form.get(key) as SharedParameterFormGroup<any>
            expect(control).toBeTruthy()
            expect(control.controls?.unlocked?.value).toBeFalse()
            expect(control.controls?.values?.value.length).toBe(1)
        }
    })

    it('should create a default Kea parameters for for an IPv6 shared network', () => {
        let form = service.createDefaultKeaSharedNetworkParametersForm(IPType.IPv6, null)
        expect(Object.keys(form.controls).length).toBe(37)

        for (const key of Object.keys(form.controls)) {
            let control = form.get(key) as SharedParameterFormGroup<any>
            expect(control).toBeTruthy()
            expect(control.controls?.unlocked?.value).toBeFalse()
            expect(control.controls?.values?.value.length).toBe(1)
        }
    })

    it('should create a default Kea parameters form for an IPv4 subnet', () => {
        let form = service.createDefaultKeaSubnetParametersForm(IPType.IPv4, null)
        expect(Object.keys(form.controls).length).toBe(39)

        for (const key of Object.keys(form.controls)) {
            let control = form.get(key) as SharedParameterFormGroup<any>
            expect(control).toBeTruthy()
            expect(control.controls?.unlocked?.value).toBeFalse()
            expect(control.controls?.values?.value.length).toBe(1)
        }
    })

    it('should create a default Kea parameters form for an IPv6 subnet', () => {
        let form = service.createDefaultKeaSubnetParametersForm(IPType.IPv6, null)
        expect(Object.keys(form.controls).length).toBe(37)

        for (const key of Object.keys(form.controls)) {
            let control = form.get(key) as SharedParameterFormGroup<any>
            expect(control).toBeTruthy()
            expect(control.controls?.unlocked?.value).toBeFalse()
            expect(control.controls?.values?.value.length).toBe(1)
        }
    })

    it('should create a default subnet form for no particular subnet', () => {
        let form = service.createDefaultSubnetForm(null, [])
        expect(form.get('subnet')?.value).toBeFalsy()
        expect(form.get('subnet')?.disabled).toBeFalse()
        expect(form.get('sharedNetwork')).toBeTruthy()
        expect(form.get('sharedNetwork').value).toBeFalsy()
        expect(form.get('pools')).toBeTruthy()
        expect((form.get('pools') as UntypedFormArray)?.length).toBe(0)
        expect(form.get('prefixPools')).toBeTruthy()
        expect((form.get('prefixPools') as UntypedFormArray)?.length).toBe(0)
        expect(form.get('parameters')).toBeTruthy()
        expect(form.get('options')).toBeTruthy()
        expect(form.get('options.unlocked')).toBeTruthy()
        expect(form.get('options.unlocked').value).toBeFalse()
        expect(form.get('options.data')).toBeTruthy()
        expect((form.get('options.data') as UntypedFormArray)?.length).toBe(1)
        expect(form.get('selectedDaemons')).toBeTruthy()
        expect(form.get('selectedDaemons').value.length).toBe(0)
    })

    it('should create a default subnet form for specific subnet', () => {
        let form = service.createDefaultSubnetForm(null, '192.0.2.0/24')
        expect(form.get('subnet')?.value).toBe('192.0.2.0/24')
        expect(form.get('subnet')?.disabled).toBeTrue()
        expect(form.get('sharedNetwork')).toBeTruthy()
        expect(form.get('sharedNetwork').value).toBeFalsy()
        expect(form.get('pools')).toBeTruthy()
        expect((form.get('pools') as UntypedFormArray)?.length).toBe(0)
        expect(form.get('prefixPools')).toBeTruthy()
        expect((form.get('prefixPools') as UntypedFormArray)?.length).toBe(0)
        expect(form.get('parameters')).toBeTruthy()
        expect(form.get('options')).toBeTruthy()
        expect(form.get('options.unlocked')).toBeTruthy()
        expect(form.get('options.unlocked').value).toBeFalse()
        expect(form.get('options.data')).toBeTruthy()
        expect((form.get('options.data') as UntypedFormArray)?.length).toBe(1)
        expect(form.get('selectedDaemons')).toBeTruthy()
        expect(form.get('selectedDaemons').value.length).toBe(0)
    })

    it('should convert IPv4 subnet data to a form', () => {
        const subnet: Subnet = {
            subnet: '192.0.2.0/24',
            sharedNetworkId: 1,
            localSubnets: [
                {
                    daemonId: 1,
                    pools: [
                        {
                            pool: '192.0.2.1-192.0.2.10',
                            keaConfigPoolParameters: {
                                clientClass: 'foo',
                                requireClientClasses: ['foo', 'bar'],
                            },
                        },
                        {
                            pool: '192.0.2.20-192.0.2.30',
                        },
                    ],
                    keaConfigSubnetParameters: {
                        subnetLevelParameters: {
                            allocator: 'random',
                            options: [
                                {
                                    alwaysSend: true,
                                    code: 5,
                                    encapsulate: '',
                                    fields: [
                                        {
                                            fieldType: 'ipv4-address',
                                            values: ['192.0.2.1'],
                                        },
                                    ],
                                    options: [],
                                    universe: 4,
                                },
                            ],
                            optionsHash: '123',
                        },
                    },
                    userContext: {
                        foo: 'bar',
                        'subnet-name': 'baz',
                    },
                },
            ],
        }
        const formGroup = service.convertSubnetToForm(IPType.IPv4, null, subnet)
        expect(formGroup.get('subnet')?.value).toBe('192.0.2.0/24')
        expect(formGroup.get('subnet')?.disabled).toBeTrue()

        expect(formGroup.get('sharedNetwork')?.value).toBe(1)

        const pools = formGroup.get('pools') as UntypedFormArray
        expect(pools.length).toBe(2)
        expect(pools.get('0.range.start')?.value).toBe('192.0.2.1')
        expect(pools.get('0.range.end')?.value).toBe('192.0.2.10')
        expect(pools.get('1.range.start')?.value).toBe('192.0.2.20')
        expect(pools.get('1.range.end')?.value).toBe('192.0.2.30')

        let poolParameters = pools.get('0.parameters') as FormGroup<KeaPoolParametersForm>
        expect(poolParameters.get('clientClass.unlocked')?.value).toBeFalse()
        expect((poolParameters.get('clientClass.values') as UntypedFormArray).length)?.toBe(1)
        expect(poolParameters.get('clientClass.values.0')?.value).toBe('foo')
        expect(poolParameters.get('requireClientClasses.unlocked')?.value).toBeFalse()
        expect((poolParameters.get('requireClientClasses.values') as UntypedFormArray)?.length).toBe(1)
        expect(poolParameters.get('requireClientClasses.values.0')?.value).toEqual(['foo', 'bar'])

        poolParameters = pools.get('1.parameters') as FormGroup<KeaPoolParametersForm>
        expect(poolParameters.get('clientClass.unlocked')?.value).toBeFalse()
        expect((poolParameters.get('clientClass.values') as UntypedFormArray).length)?.toBe(1)
        expect(poolParameters.get('clientClass.values.0')?.value).toBeFalsy()
        expect(poolParameters.get('requireClientClasses.unlocked')?.value).toBeFalse()
        expect((poolParameters.get('requireClientClasses.values') as UntypedFormArray)?.length).toBe(1)
        expect(poolParameters.get('requireClientClasses.values.0')?.value).toEqual([])

        const options = formGroup.get('options.data') as UntypedFormArray
        expect(options.length).toBe(1)
        expect(options.get('0.0.optionFields.0.control')?.value).toBe('192.0.2.1')

        const parameters = formGroup.get('parameters') as FormGroup<KeaSubnetParametersForm>
        expect(parameters.get('allocator.unlocked')?.value).toBeFalse()
        expect((parameters.get('allocator.values') as UntypedFormArray).length).toBe(1)
        expect(parameters.get('allocator.values.0')?.value).toBe('random')

        const selectedDaemons = formGroup.get('selectedDaemons') as FormControl<number[]>
        expect(selectedDaemons?.value).toEqual([1])
        expect(selectedDaemons?.disabled).toBeTrue()

        const userContextGroup = formGroup.get('userContexts') as FormGroup<UserContextsForm>
        expect(userContextGroup.get('unlocked')?.value).toBeFalse()
        const userContexts = userContextGroup.get('contexts') as UntypedFormArray
        expect(userContexts.length).toBe(1)
        const userContext = userContexts.get('0').value
        expect(userContext['foo']).toBe('bar')
        expect(userContext['subnet-name']).toBe('baz')
        expect(Object.keys(userContext).length).toBe(2)
    })

    it('should only include subnet-level ddns-use-conflict-resolution when all Kea versions are earlier than 2.5.0', () => {
        const subnet: Subnet = {
            subnet: '192.0.2.0/24',
            sharedNetworkId: 1,
            localSubnets: [
                {
                    daemonId: 1,
                    keaConfigSubnetParameters: {
                        subnetLevelParameters: {
                            ddnsUseConflictResolution: true,
                            ddnsConflictResolutionMode: 'check-with-dhcid',
                        },
                    },
                },
            ],
        }
        const formGroup = service.convertSubnetToForm(IPType.IPv4, ['2.0.0', '2.1.2'], subnet)

        const parameters = formGroup.get('parameters') as FormGroup<KeaSubnetParametersForm>
        expect(parameters.get('ddnsUseConflictResolution')).toBeTruthy()
        expect(parameters.get('ddnsConflictResolutionMode')).toBeFalsy()
    })

    it('should only include subnet-level ddns-conflict-resolution-mode when all Kea versions are 2.5.0 or later', () => {
        const subnet: Subnet = {
            subnet: '192.0.2.0/24',
            sharedNetworkId: 1,
            localSubnets: [
                {
                    daemonId: 1,
                    keaConfigSubnetParameters: {
                        subnetLevelParameters: {
                            ddnsUseConflictResolution: true,
                            ddnsConflictResolutionMode: 'check-with-dhcid',
                        },
                    },
                },
            ],
        }
        const formGroup = service.convertSubnetToForm(IPType.IPv4, ['2.5.0', '3.1.2'], subnet)

        const parameters = formGroup.get('parameters') as FormGroup<KeaSubnetParametersForm>
        expect(parameters.get('ddnsConflictResolutionMode')).toBeTruthy()
        expect(parameters.get('ddnsUseConflictResolution')).toBeFalsy()
    })

    it('should include subnet-level ddns-conflict-resolution-mode and ddns-conflict-resolution-mode', () => {
        const subnet: Subnet = {
            subnet: '192.0.2.0/24',
            sharedNetworkId: 1,
            localSubnets: [
                {
                    daemonId: 1,
                    keaConfigSubnetParameters: {
                        subnetLevelParameters: {
                            ddnsUseConflictResolution: true,
                            ddnsConflictResolutionMode: 'check-with-dhcid',
                        },
                    },
                },
            ],
        }
        const formGroup = service.convertSubnetToForm(IPType.IPv4, ['2.2.0', '3.1.2'], subnet)

        const parameters = formGroup.get('parameters') as FormGroup<KeaSubnetParametersForm>
        expect(parameters.get('ddnsConflictResolutionMode')).toBeTruthy()
        expect(parameters.get('ddnsUseConflictResolution')).toBeTruthy()
    })

    it('should convert IPv6 subnet data to a form', () => {
        const subnet: Subnet = {
            subnet: '2001:db8:1::/64',
            localSubnets: [
                {
                    daemonId: 1,
                    pools: [
                        {
                            pool: '2001:db8:1::1-2001:db8:1::10',
                            keaConfigPoolParameters: {
                                clientClass: 'foo',
                                requireClientClasses: ['foo', 'bar'],
                            },
                        },
                        {
                            pool: '2001:db8:1::20-2001:db8:1::30',
                        },
                        {
                            pool: '2001:db8:1::40-2001:db8:1::50',
                        },
                    ],
                    prefixDelegationPools: [
                        {
                            prefix: '3000::/16',
                            delegatedLength: 112,
                            excludedPrefix: '3000::ee00/120',
                            keaConfigPoolParameters: {
                                clientClass: 'baz',
                                requireClientClasses: ['foo'],
                            },
                        },
                    ],
                    keaConfigSubnetParameters: {
                        subnetLevelParameters: {
                            pdAllocator: 'random',
                            options: [
                                {
                                    alwaysSend: true,
                                    code: 23,
                                    encapsulate: '',
                                    fields: [
                                        {
                                            fieldType: 'ipv6-address',
                                            values: ['2001:db8:1::10'],
                                        },
                                    ],
                                    options: [],
                                    universe: 6,
                                },
                            ],
                            optionsHash: '123',
                        },
                    },
                },
                {
                    daemonId: 2,
                    pools: [
                        {
                            pool: '2001:db8:1::1-2001:db8:1::10',
                            keaConfigPoolParameters: {
                                clientClass: 'bar',
                            },
                        },
                        {
                            pool: '2001:db8:1::20-2001:db8:1::30',
                            keaConfigPoolParameters: {
                                requireClientClasses: ['foo'],
                            },
                        },
                    ],
                    prefixDelegationPools: [
                        {
                            prefix: '3000::/16',
                            delegatedLength: 112,
                            excludedPrefix: '3000::ee00/120',
                            keaConfigPoolParameters: {
                                clientClass: 'bar',
                                requireClientClasses: ['foo'],
                            },
                        },
                    ],
                    keaConfigSubnetParameters: {
                        subnetLevelParameters: {
                            pdAllocator: 'flq',
                            options: [
                                {
                                    alwaysSend: true,
                                    code: 23,
                                    encapsulate: '',
                                    fields: [
                                        {
                                            fieldType: 'ipv6-address',
                                            values: ['2001:db8:1::20'],
                                        },
                                    ],
                                    options: [],
                                    universe: 6,
                                },
                            ],
                            optionsHash: '234',
                        },
                    },
                },
            ],
        }
        const formGroup = service.convertSubnetToForm(IPType.IPv6, null, subnet)
        expect(formGroup.get('subnet')?.value).toBe('2001:db8:1::/64')
        expect(formGroup.get('subnet')?.disabled).toBeTrue()

        expect(formGroup.get('sharedNetwork')?.value).toBeFalsy()

        const pools = formGroup.get('pools') as UntypedFormArray
        expect(pools.length).toBe(3)
        expect(pools.get('0.range.start')?.value).toBe('2001:db8:1::1')
        expect(pools.get('0.range.end')?.value).toBe('2001:db8:1::10')
        expect(pools.get('1.range.start')?.value).toBe('2001:db8:1::20')
        expect(pools.get('1.range.end')?.value).toBe('2001:db8:1::30')
        expect(pools.get('2.range.start')?.value).toBe('2001:db8:1::40')
        expect(pools.get('2.range.end')?.value).toBe('2001:db8:1::50')

        let poolParameters = pools.get('0.parameters') as FormGroup<KeaPoolParametersForm>
        expect(poolParameters.get('clientClass.unlocked')?.value).toBeTrue()
        expect((poolParameters.get('clientClass.values') as UntypedFormArray).length)?.toBe(2)
        expect(poolParameters.get('clientClass.values.0')?.value).toBe('foo')
        expect(poolParameters.get('clientClass.values.1')?.value).toBe('bar')
        expect(poolParameters.get('requireClientClasses.unlocked')?.value).toBeTrue()
        expect((poolParameters.get('requireClientClasses.values') as UntypedFormArray)?.length).toBe(2)
        expect(poolParameters.get('requireClientClasses.values.0')?.value).toEqual(['foo', 'bar'])
        expect(poolParameters.get('requireClientClasses.values.1')?.value).toEqual([])

        poolParameters = pools.get('1.parameters') as FormGroup<KeaPoolParametersForm>
        expect(poolParameters.get('clientClass.unlocked')?.value).toBeFalse()
        expect((poolParameters.get('clientClass.values') as UntypedFormArray).length)?.toBe(2)
        expect(poolParameters.get('clientClass.values.0')?.value).toBeFalsy()
        expect(poolParameters.get('clientClass.values.1')?.value).toBeFalsy()
        expect(poolParameters.get('requireClientClasses.unlocked')?.value).toBeTrue()
        expect((poolParameters.get('requireClientClasses.values') as UntypedFormArray)?.length).toBe(2)
        expect(poolParameters.get('requireClientClasses.values.0')?.value).toEqual([])
        expect(poolParameters.get('requireClientClasses.values.1')?.value).toEqual(['foo'])

        poolParameters = pools.get('2.parameters') as FormGroup<KeaPoolParametersForm>
        expect(poolParameters.get('clientClass.unlocked')?.value).toBeFalse()
        expect((poolParameters.get('clientClass.values') as UntypedFormArray).length)?.toBe(1)
        expect(poolParameters.get('clientClass.values.0')?.value).toBeFalsy()
        expect(poolParameters.get('requireClientClasses.unlocked')?.value).toBeFalse()
        expect((poolParameters.get('requireClientClasses.values') as UntypedFormArray)?.length).toBe(1)
        expect(poolParameters.get('requireClientClasses.values.0')?.value).toEqual([])

        const prefixPools = formGroup.get('prefixPools') as UntypedFormArray
        expect(prefixPools.length).toBe(1)
        expect(prefixPools.get('0.prefixes.prefix')?.value).toBe('3000::/16')
        expect(prefixPools.get('0.prefixes.delegatedLength')?.value).toBe(112)
        expect(prefixPools.get('0.prefixes.excludedPrefix')?.value).toBe('3000::ee00/120')

        poolParameters = prefixPools.get('0.parameters') as FormGroup<KeaPoolParametersForm>
        expect(poolParameters.get('clientClass.unlocked')?.value).toBeTrue()
        expect((poolParameters.get('clientClass.values') as UntypedFormArray).length)?.toBe(2)
        expect(poolParameters.get('clientClass.values.0')?.value).toBe('baz')
        expect(poolParameters.get('clientClass.values.1')?.value).toBe('bar')
        expect(poolParameters.get('requireClientClasses.unlocked')?.value).toBeFalse()
        expect((poolParameters.get('requireClientClasses.values') as UntypedFormArray)?.length).toBe(2)
        expect(poolParameters.get('requireClientClasses.values.0')?.value).toEqual(['foo'])
        expect(poolParameters.get('requireClientClasses.values.1')?.value).toEqual(['foo'])

        expect(formGroup.get('options.unlocked')?.value).toBeTrue()

        const options = formGroup.get('options.data') as UntypedFormArray
        expect(options.length).toBe(2)
        expect(options.get('0.0.optionFields.0.control')?.value).toBe('2001:db8:1::10')
        expect(options.get('1.0.optionFields.0.control')?.value).toBe('2001:db8:1::20')

        const parameters = formGroup.get('parameters') as FormGroup<KeaSubnetParametersForm>
        expect(parameters.get('pdAllocator.unlocked')?.value).toBeTrue()
        expect((parameters.get('pdAllocator.values') as UntypedFormArray).length).toBe(2)
        expect(parameters.get('pdAllocator.values.0')?.value).toBe('random')

        const selectedDaemons = formGroup.get('selectedDaemons') as FormControl<number[]>
        expect(selectedDaemons?.value).toEqual([1, 2])
        expect(selectedDaemons?.disabled).toBeFalse()
    })

    it('should convert a subnet with no local subnets to a form', () => {
        const subnet: Subnet = {
            subnet: '192.0.2.0/24',
        }
        const formGroup = service.convertSubnetToForm(IPType.IPv4, null, subnet)
        expect(formGroup.get('subnet')?.value).toBe('192.0.2.0/24')

        const options = formGroup.get('options.data') as UntypedFormArray
        expect(options.length).toBe(0)

        const parameters = formGroup.get('parameters') as FormGroup<KeaSubnetParametersForm>
        expect(parameters.get('allocator.unlocked')?.value).toBeFalse()
        expect((parameters.get('allocator.values') as UntypedFormArray).length).toBe(0)

        const selectedDaemons = formGroup.get('selectedDaemons') as FormControl<number[]>
        expect(selectedDaemons?.value.length).toBe(0)
    })

    it('should convert a form to Kea parameters', () => {
        const daemons = [
            {
                id: 1,
                version: '2.3.4',
            },
            {
                id: 2,
                version: '2.5.0',
            },
        ]
        const params = service.convertFormToKeaSubnetParameters(
            daemons,
            new FormGroup<KeaSubnetParametersForm>({
                cacheThreshold: new SharedParameterFormGroup<number>(
                    {
                        type: 'number',
                        min: 0,
                        max: 1,
                        fractionDigits: 2,
                        versionLowerBound: '2.4.0',
                    },
                    [new FormControl<number>(0.5), new FormControl<number>(0.5)]
                ),
                allocator: new SharedParameterFormGroup<string>(
                    {
                        type: 'string',
                        values: ['iterative', 'random', 'flq'],
                        versionLowerBound: '2.2.0',
                    },
                    [new FormControl<string>('flq'), new FormControl<string>('random')]
                ),
                authoritative: new SharedParameterFormGroup<boolean>(
                    {
                        type: 'boolean',
                        versionUpperBound: '2.5.0',
                    },
                    [new FormControl<boolean>(true), new FormControl<boolean>(false), new FormControl<boolean>(false)]
                ),
                relayAddresses: new SharedParameterFormGroup<string[]>(
                    {
                        type: 'string',
                    },
                    [new FormControl<string[]>(['192.0.2.1', '192.0.2.2']), new FormControl<string[]>(['192.0.2.2'])]
                ),
            })
        )
        expect(params.length).toBe(3)
        expect(params[0].cacheThreshold).toBeFalsy()
        expect(params[0].allocator).toBe('flq')
        expect(params[0].authoritative).toBeTrue()
        expect(params[0].relay).toBeTruthy()
        expect(params[0].relay.ipAddresses).toEqual(['192.0.2.1', '192.0.2.2'])
        expect(params[1].cacheThreshold).toBe(0.5)
        expect(params[1].allocator).toBe('random')
        expect(params[1].authoritative).toBeFalsy()
        expect(params[1].relay).toBeTruthy()
        expect(params[1].relay.ipAddresses).toEqual(['192.0.2.2'])
        expect(params[2].cacheThreshold).toBeFalsy()
        expect(params[2].allocator).toBeFalsy()
        expect(params[2].authoritative).toBeFalse()
        expect(params[2].relay).toBeFalsy()
    })

    it('should convert a form to subnet', () => {
        const subnet0: Subnet = {
            subnet: '192.0.2.0/24',
            sharedNetworkId: 1,
            localSubnets: [
                {
                    daemonId: 1,
                    keaConfigSubnetParameters: {
                        subnetLevelParameters: {
                            options: [
                                {
                                    alwaysSend: true,
                                    code: 5,
                                    encapsulate: '',
                                    fields: [
                                        {
                                            fieldType: 'ipv4-address',
                                            values: ['192.0.2.1'],
                                        },
                                    ],
                                    options: [],
                                    universe: 4,
                                },
                            ],
                            optionsHash: '123',
                        },
                    },
                    userContext: {
                        foo: 'bar',
                    },
                },
                {
                    daemonId: 2,
                    keaConfigSubnetParameters: {
                        subnetLevelParameters: {
                            allocator: 'random',
                        },
                    },
                    userContext: {
                        bar: 'foo',
                    },
                },
            ],
        }
        const formGroup = service.convertSubnetToForm(IPType.IPv4, null, subnet0)

        const daemons: VersionedDaemon[] = [
            {
                id: 1,
                version: '2.2.3',
            },
            {
                id: 2,
                version: null,
            },
        ]
        const subnet1 = service.convertFormToSubnet(daemons, formGroup)

        expect(subnet1.subnet).toBe('192.0.2.0/24')
        expect(subnet1.sharedNetworkId).toBe(1)
        expect(subnet1.localSubnets.length).toBe(2)

        expect(subnet1.localSubnets[0].userContext).toEqual({ foo: 'bar' })
        expect(subnet1.localSubnets[1].userContext).toEqual({ bar: 'foo' })

        expect(subnet1.localSubnets[0].keaConfigSubnetParameters?.subnetLevelParameters?.options?.length).toBe(1)
        expect(subnet1.localSubnets[0].keaConfigSubnetParameters?.subnetLevelParameters?.options[0].code).toBe(5)
        expect(
            subnet1.localSubnets[0].keaConfigSubnetParameters?.subnetLevelParameters?.options[0].alwaysSend
        ).toBeTrue()
        expect(subnet1.localSubnets[0].keaConfigSubnetParameters?.subnetLevelParameters?.options[0].fields.length).toBe(
            1
        )
        expect(
            subnet1.localSubnets[0].keaConfigSubnetParameters?.subnetLevelParameters?.options[0].fields[0].fieldType
        ).toBe('ipv4-address')
        expect(
            subnet1.localSubnets[0].keaConfigSubnetParameters?.subnetLevelParameters?.options[0].fields[0].values.length
        ).toBe(1)
        expect(
            subnet1.localSubnets[0].keaConfigSubnetParameters?.subnetLevelParameters?.options[0].fields[0].values[0]
        ).toBe('192.0.2.1')
        expect(subnet1.localSubnets[0].keaConfigSubnetParameters?.subnetLevelParameters?.options[0].universe).toBe(4)

        expect(subnet1.localSubnets[1].keaConfigSubnetParameters?.subnetLevelParameters?.allocator).toBe('random')
        expect(subnet1.localSubnets[1].keaConfigSubnetParameters?.subnetLevelParameters?.options.length).toBe(0)
    })

    it('should convert a form to subnet when options are locked', () => {
        // It is easier to create a subnet instance and convert it to a
        // form rather than creating the form manually.
        const subnet0: Subnet = {
            subnet: '192.0.2.0/24',
            localSubnets: [
                {
                    daemonId: 1,
                    keaConfigSubnetParameters: {
                        subnetLevelParameters: {
                            options: [
                                {
                                    alwaysSend: true,
                                    code: 5,
                                    encapsulate: '',
                                    fields: [
                                        {
                                            fieldType: 'ipv4-address',
                                            values: ['192.0.2.1'],
                                        },
                                    ],
                                    options: [],
                                    universe: 4,
                                },
                            ],
                            optionsHash: '123',
                        },
                    },
                },
                {
                    daemonId: 2,
                    keaConfigSubnetParameters: {
                        subnetLevelParameters: {
                            allocator: 'random',
                            options: [
                                {
                                    alwaysSend: true,
                                    code: 5,
                                    encapsulate: '',
                                    fields: [
                                        {
                                            fieldType: 'ipv4-address',
                                            values: ['192.0.2.1'],
                                        },
                                    ],
                                    options: [],
                                    universe: 4,
                                },
                            ],
                            optionsHash: '123',
                        },
                    },
                },
            ],
        }
        const formGroup = service.convertSubnetToForm(IPType.IPv4, null, subnet0)

        // Both servers have the same options so the options are locked by default.
        // Let's now modify the second option instance while they are locked. The
        // second conversion should ignore this modification and take the first
        // option for each server.
        formGroup.get('options.data.1.0.optionFields.0.control')?.setValue('10.1.1.1')

        const daemons: VersionedDaemon[] = [
            {
                id: 1,
                version: '2.2.3',
            },
            {
                id: 2,
                version: null,
            },
        ]
        const subnet1 = service.convertFormToSubnet(daemons, formGroup)
        expect(subnet1.localSubnets.length).toBe(2)
        expect(subnet1.localSubnets[1].keaConfigSubnetParameters?.subnetLevelParameters?.options?.length).toBe(1)
        expect(subnet1.localSubnets[1].keaConfigSubnetParameters.subnetLevelParameters.options[0].fields.length).toBe(1)
        expect(
            subnet1.localSubnets[1].keaConfigSubnetParameters.subnetLevelParameters.options[0].fields[0].values.length
        ).toBe(1)
        expect(
            subnet1.localSubnets[1].keaConfigSubnetParameters.subnetLevelParameters.options[0].fields[0].values[0]
        ).toBe('192.0.2.1')
    })

    it('should filter out unsupported parameters during form to subnet conversion', () => {
        // The ddns-use-conflict-resolution was deprecated and replaced with
        // the ddns-conflict-resolution-mode in Kea 2.5.0. Let's create the subnet
        // that carries both of these parameters. The converted subnet should
        // exclude unsupported parameters for specific versions.
        const subnet0: Subnet = {
            subnet: '192.0.2.0/24',
            sharedNetworkId: 1,
            localSubnets: [
                {
                    daemonId: 1,
                    keaConfigSubnetParameters: {
                        subnetLevelParameters: {
                            ddnsUseConflictResolution: true,
                            ddnsConflictResolutionMode: 'check-with-dhcid',
                        },
                    },
                },
                {
                    daemonId: 2,
                    keaConfigSubnetParameters: {
                        subnetLevelParameters: {
                            ddnsUseConflictResolution: true,
                            ddnsConflictResolutionMode: 'check-with-dhcid',
                        },
                    },
                },
            ],
        }
        // Use a wide range of Kea versions in the subnet to form conversion to
        // ensure both parameters are included in the resulting form.
        const formGroup = service.convertSubnetToForm(IPType.IPv4, ['1.0.0', '3.0.0'], subnet0)

        // Specify two daemons. The first one has version 2.3.2 which only supports
        // the ddns-use-conflict-resolution. The latter is 2.5.0 which supports
        // ddns-conflict-resolution-mode.
        const daemons: VersionedDaemon[] = [
            {
                id: 1,
                version: '2.3.2',
            },
            {
                id: 2,
                version: '2.5.0',
            },
        ]
        const subnet1 = service.convertFormToSubnet(daemons, formGroup)

        expect(subnet1.subnet).toBe('192.0.2.0/24')
        expect(subnet1.sharedNetworkId).toBe(1)
        expect(subnet1.localSubnets.length).toBe(2)

        expect(
            subnet1.localSubnets[0].keaConfigSubnetParameters?.subnetLevelParameters?.ddnsUseConflictResolution
        ).toBe(true)
        expect(
            subnet1.localSubnets[0].keaConfigSubnetParameters?.subnetLevelParameters?.ddnsConflictResolutionMode
        ).toBeFalsy()
        expect(
            subnet1.localSubnets[1].keaConfigSubnetParameters?.subnetLevelParameters?.ddnsUseConflictResolution
        ).toBeFalsy()
        expect(
            subnet1.localSubnets[1].keaConfigSubnetParameters?.subnetLevelParameters?.ddnsConflictResolutionMode
        ).toBe('check-with-dhcid')
    })

    it('should change the subnet name in the user context', () => {
        const subnet0: Subnet = {
            subnet: '10.0.0.0/8',
            localSubnets: [
                {
                    daemonId: 1,
                    keaConfigSubnetParameters: { subnetLevelParameters: {} },
                    userContext: {
                        subnetName: 'foo',
                    },
                },
            ],
        }

        // Edit the subnet name.
        const form = service.convertSubnetToForm(IPType.IPv4, null, subnet0)
        form.get('userContexts.names.0')?.setValue('bar')

        const subnet1 = service.convertFormToSubnet([], form)
        expect(subnet1.localSubnets[0].userContext['subnet-name']).toBe('bar')

        // Remove the subnet name.
        form.get('userContexts.names.0')?.setValue('')
        const subnet2 = service.convertFormToSubnet([], form)
        expect(subnet2.localSubnets[0].userContext['subnet-name']).toBeUndefined()
    })

    it('should convert a form to shared network', () => {
        // It is easier to create a shared network instance and convert it to a
        // form rather than creating the form manually.
        const sharedNetwork0: SharedNetwork = {
            name: 'stanza',
            localSharedNetworks: [
                {
                    daemonId: 1,
                    keaConfigSharedNetworkParameters: {
                        sharedNetworkLevelParameters: {
                            options: [
                                {
                                    alwaysSend: true,
                                    code: 5,
                                    encapsulate: '',
                                    fields: [
                                        {
                                            fieldType: 'ipv4-address',
                                            values: ['192.0.2.1'],
                                        },
                                    ],
                                    options: [],
                                    universe: 4,
                                },
                            ],
                            optionsHash: '123',
                        },
                    },
                },
                {
                    daemonId: 2,
                    keaConfigSharedNetworkParameters: {
                        sharedNetworkLevelParameters: {
                            allocator: 'random',
                        },
                    },
                },
            ],
        }
        const formGroup = service.convertSharedNetworkToForm(['1.1.1', '3.3.3'], sharedNetwork0, [])

        const sharedNetwork1 = service.convertFormToSharedNetwork([], IPType.IPv4, formGroup)

        expect(sharedNetwork1.name).toBe('stanza')
        expect(sharedNetwork1.localSharedNetworks.length).toBe(2)

        expect(
            sharedNetwork1.localSharedNetworks[0].keaConfigSharedNetworkParameters?.sharedNetworkLevelParameters
                ?.options?.length
        ).toBe(1)
        expect(
            sharedNetwork1.localSharedNetworks[0].keaConfigSharedNetworkParameters?.sharedNetworkLevelParameters
                ?.options[0].code
        ).toBe(5)
        expect(
            sharedNetwork1.localSharedNetworks[0].keaConfigSharedNetworkParameters?.sharedNetworkLevelParameters
                ?.options[0].alwaysSend
        ).toBeTrue()
        expect(
            sharedNetwork1.localSharedNetworks[0].keaConfigSharedNetworkParameters?.sharedNetworkLevelParameters
                ?.options[0].fields.length
        ).toBe(1)
        expect(
            sharedNetwork1.localSharedNetworks[0].keaConfigSharedNetworkParameters?.sharedNetworkLevelParameters
                ?.options[0].fields[0].fieldType
        ).toBe('ipv4-address')
        expect(
            sharedNetwork1.localSharedNetworks[0].keaConfigSharedNetworkParameters?.sharedNetworkLevelParameters
                ?.options[0].fields[0].values.length
        ).toBe(1)
        expect(
            sharedNetwork1.localSharedNetworks[0].keaConfigSharedNetworkParameters?.sharedNetworkLevelParameters
                ?.options[0].fields[0].values[0]
        ).toBe('192.0.2.1')
        expect(
            sharedNetwork1.localSharedNetworks[0].keaConfigSharedNetworkParameters?.sharedNetworkLevelParameters
                ?.options[0].universe
        ).toBe(4)

        expect(
            sharedNetwork1.localSharedNetworks[1].keaConfigSharedNetworkParameters?.sharedNetworkLevelParameters
                ?.allocator
        ).toBe('random')
        expect(
            sharedNetwork1.localSharedNetworks[1].keaConfigSharedNetworkParameters?.sharedNetworkLevelParameters
                ?.options.length
        ).toBe(0)
    })

    it('should filter out unsupported parameters during form to shared network conversion', () => {
        // The ddns-use-conflict-resolution was deprecated and replaced with
        // the ddns-conflict-resolution-mode in Kea 2.5.0. Let's create the shared network
        // that carries both of these parameters. The converted shared network should
        // exclude unsupported parameters for specific versions.
        const sharedNetwork0: SharedNetwork = {
            name: 'stanza',
            localSharedNetworks: [
                {
                    daemonId: 1,
                    keaConfigSharedNetworkParameters: {
                        sharedNetworkLevelParameters: {
                            ddnsUseConflictResolution: true,
                            ddnsConflictResolutionMode: 'check-with-dhcid',
                        },
                    },
                },
                {
                    daemonId: 2,
                    keaConfigSharedNetworkParameters: {
                        sharedNetworkLevelParameters: {
                            ddnsUseConflictResolution: true,
                            ddnsConflictResolutionMode: 'check-with-dhcid',
                        },
                    },
                },
            ],
        }

        // Use a wide range of Kea versions in the shared network to form conversion to
        // ensure both parameters are included in the resulting form.
        const formGroup = service.convertSharedNetworkToForm(['1.1.1', '3.3.3'], sharedNetwork0, [])

        // Specify two daemons. The first one has version 2.3.2 which only supports
        // the ddns-use-conflict-resolution. The latter is 2.5.0 which supports
        // ddns-conflict-resolution-mode.
        const daemons: VersionedDaemon[] = [
            {
                id: 1,
                version: '2.3.2',
            },
            {
                id: 2,
                version: '2.5.0',
            },
        ]

        const sharedNetwork1 = service.convertFormToSharedNetwork(daemons, IPType.IPv4, formGroup)

        expect(
            sharedNetwork1.localSharedNetworks[0].keaConfigSharedNetworkParameters?.sharedNetworkLevelParameters
                ?.ddnsUseConflictResolution
        ).toBe(true)
        expect(
            sharedNetwork1.localSharedNetworks[0].keaConfigSharedNetworkParameters?.sharedNetworkLevelParameters
                ?.ddnsConflictResolutionMode
        ).toBeFalsy()
        expect(
            sharedNetwork1.localSharedNetworks[1].keaConfigSharedNetworkParameters?.sharedNetworkLevelParameters
                ?.ddnsUseConflictResolution
        ).toBeFalsy()
        expect(
            sharedNetwork1.localSharedNetworks[1].keaConfigSharedNetworkParameters?.sharedNetworkLevelParameters
                ?.ddnsConflictResolutionMode
        ).toBe('check-with-dhcid')
    })

    it('should create a default IPv4 shared network form', () => {
        const fg = service.createDefaultSharedNetworkForm(IPType.IPv4, null, ['foo'])
        expect(fg).toBeTruthy()

        // shared network name
        expect(fg.get('name')).toBeTruthy()
        expect(fg.get('name').value).toBe('')

        // This field is DHCPv4-specific. By checking its existence we ensure that
        // the function distinguishes IPv4 and IPv6 cases.
        expect(fg.get('parameters.bootFileName')).toBeTruthy()

        // options
        expect(fg.get('options.unlocked')).toBeTruthy()
        expect(fg.get('options.unlocked').value).toBeFalse()

        // daemons
        expect(fg.get('selectedDaemons')).toBeTruthy()
        expect(fg.get('selectedDaemons').value).toEqual([])
    })

    it('should create a default IPv6 shared network form', () => {
        const fg = service.createDefaultSharedNetworkForm(IPType.IPv6, null, ['foo'])
        expect(fg).toBeTruthy()

        // shared network name
        expect(fg.get('name')).toBeTruthy()
        expect(fg.get('name').value).toBe('')

        // This field is DHCPv6-specific. By checking its existence we ensure that
        // the function distinguishes IPv4 and IPv6 cases.
        expect(fg.get('parameters.preferredLifetime')).toBeTruthy()

        // options
        expect(fg.get('options.unlocked')).toBeTruthy()
        expect(fg.get('options.unlocked').value).toBeFalse()

        // daemons
        expect(fg.get('selectedDaemons')).toBeTruthy()
        expect(fg.get('selectedDaemons').value).toEqual([])
    })

    it('should convert a form to shared network when options are locked', () => {
        const sharedNetwork0: SharedNetwork = {
            name: 'stanza',
            localSharedNetworks: [
                {
                    daemonId: 1,
                    keaConfigSharedNetworkParameters: {
                        sharedNetworkLevelParameters: {
                            options: [
                                {
                                    alwaysSend: true,
                                    code: 5,
                                    encapsulate: '',
                                    fields: [
                                        {
                                            fieldType: 'ipv4-address',
                                            values: ['192.0.2.1'],
                                        },
                                    ],
                                    options: [],
                                    universe: 4,
                                },
                            ],
                            optionsHash: '123',
                        },
                    },
                },
                {
                    daemonId: 2,
                    keaConfigSharedNetworkParameters: {
                        sharedNetworkLevelParameters: {
                            allocator: 'random',
                            options: [
                                {
                                    alwaysSend: true,
                                    code: 5,
                                    encapsulate: '',
                                    fields: [
                                        {
                                            fieldType: 'ipv4-address',
                                            values: ['192.0.2.1'],
                                        },
                                    ],
                                    options: [],
                                    universe: 4,
                                },
                            ],
                            optionsHash: '123',
                        },
                    },
                },
            ],
        }
        const formGroup = service.convertSharedNetworkToForm(null, sharedNetwork0, [])

        // Both servers have the same options so the options are locked by default.
        // Let's now modify the second option instance while they are locked. The
        // second conversion should ignore this modification and take the first
        // option for each server.
        formGroup.get('options.data.1.0.optionFields.0.control')?.setValue('10.1.1.1')

        const subnet1 = service.convertFormToSharedNetwork(null, IPType.IPv4, formGroup)
        expect(subnet1.localSharedNetworks.length).toBe(2)
        expect(
            subnet1.localSharedNetworks[1].keaConfigSharedNetworkParameters?.sharedNetworkLevelParameters?.options
                ?.length
        ).toBe(1)
        expect(
            subnet1.localSharedNetworks[1].keaConfigSharedNetworkParameters.sharedNetworkLevelParameters.options[0]
                .fields.length
        ).toBe(1)
        expect(
            subnet1.localSharedNetworks[1].keaConfigSharedNetworkParameters.sharedNetworkLevelParameters.options[0]
                .fields[0].values.length
        ).toBe(1)
        expect(
            subnet1.localSharedNetworks[1].keaConfigSharedNetworkParameters.sharedNetworkLevelParameters.options[0]
                .fields[0].values[0]
        ).toBe('192.0.2.1')
    })

    it('should convert DHCPv4 global parameters to form', () => {
        const configs: KeaRawConfig[] = [
            {
                allocator: 'iterative',
                authoritative: false,
                'ddns-conflict-resolution-mode': 'check-with-dhcid',
                'ddns-use-conflict-resolution': true,
                'ddns-generated-prefix': 'myhost',
                'ddns-override-client-update': false,
                'ddns-override-no-update': false,
                'ddns-qualifying-suffix': '',
                'ddns-replace-client-name': 'never',
                'ddns-send-updates': true,
                'ddns-update-on-renew': false,
                'dhcp-ddns': {
                    'enable-updates': false,
                    'max-queue-size': 1024,
                    'ncr-format': 'JSON',
                    'ncr-protocol': 'UDP',
                    'sender-ip': '0.0.0.0',
                    'sender-port': 0,
                    'server-ip': '127.0.0.1',
                    'server-port': 53001,
                },
                'early-global-reservations-lookup': false,
                'echo-client-id': true,
                'expired-leases-processing': {
                    'flush-reclaimed-timer-wait-time': 25,
                    'hold-reclaimed-time': 3600,
                    'max-reclaim-leases': 100,
                    'max-reclaim-time': 250,
                    'reclaim-timer-wait-time': 10,
                    'unwarned-reclaim-cycles': 5,
                },
                'host-reservation-identifiers': ['hw-address', 'duid', 'circuit-id', 'client-id'],
                'reservations-global': false,
                'reservations-in-subnet': true,
                'reservations-lookup-first': false,
                'reservations-out-of-pool': false,
            },
            {
                allocator: 'iterative',
                authoritative: true,
                'ddns-conflict-resolution-mode': 'check-with-dhcid',
                'ddns-use-conflict-resolution': true,
                'ddns-generated-prefix': 'myhost',
                'ddns-override-client-update': false,
                'ddns-override-no-update': false,
                'ddns-qualifying-suffix': 'example.org',
                'ddns-replace-client-name': 'never',
                'ddns-send-updates': true,
                'ddns-update-on-renew': false,
                'dhcp-ddns': {
                    'enable-updates': false,
                    'max-queue-size': 1024,
                    'ncr-format': 'JSON',
                    'ncr-protocol': 'UDP',
                    'sender-ip': '0.0.0.0',
                    'sender-port': 0,
                    'server-ip': '127.0.0.1',
                    'server-port': 53001,
                },
                'early-global-reservations-lookup': false,
                'echo-client-id': true,
                'expired-leases-processing': {
                    'flush-reclaimed-timer-wait-time': 30,
                    'hold-reclaimed-time': 3600,
                    'max-reclaim-leases': 100,
                    'max-reclaim-time': 250,
                    'reclaim-timer-wait-time': 10,
                    'unwarned-reclaim-cycles': 5,
                },
                'host-reservation-identifiers': ['hw-address', 'duid', 'circuit-id'],
                'reservations-global': false,
                'reservations-in-subnet': true,
                'reservations-lookup-first': false,
                'reservations-out-of-pool': false,
            },
        ]
        const formGroup = service.convertKeaGlobalParametersToForm(null, 'Dhcp4', configs)
        expect(formGroup.get('allocator.unlocked')?.value).toBeFalse()
        expect((formGroup.get('allocator.values') as UntypedFormArray).length).toBe(2)
        expect(formGroup.get('allocator.values.0')?.value).toBe('iterative')
        expect(formGroup.get('allocator.values.1')?.value).toBe('iterative')

        expect(formGroup.get('authoritative.unlocked')?.value).toBeTrue()
        expect((formGroup.get('authoritative.values') as UntypedFormArray).length).toBe(2)
        expect(formGroup.get('authoritative.values.0')?.value).toBe(false)
        expect(formGroup.get('authoritative.values.1')?.value).toBe(true)

        expect(formGroup.get('ddnsConflictResolutionMode.unlocked')?.value).toBeFalse()
        expect((formGroup.get('ddnsConflictResolutionMode.values') as UntypedFormArray).length).toBe(2)
        expect(formGroup.get('ddnsConflictResolutionMode.values.0')?.value).toBe('check-with-dhcid')
        expect(formGroup.get('ddnsConflictResolutionMode.values.1')?.value).toBe('check-with-dhcid')

        expect(formGroup.get('ddnsUseConflictResolution.unlocked')?.value).toBeFalse()
        expect((formGroup.get('ddnsUseConflictResolution.values') as UntypedFormArray).length).toBe(2)
        expect(formGroup.get('ddnsUseConflictResolution.values.0')?.value).toBe(true)
        expect(formGroup.get('ddnsUseConflictResolution.values.1')?.value).toBe(true)

        expect(formGroup.get('ddnsGeneratedPrefix.unlocked')?.value).toBeFalse()
        expect((formGroup.get('ddnsGeneratedPrefix.values') as UntypedFormArray).length).toBe(2)
        expect(formGroup.get('ddnsGeneratedPrefix.values.0')?.value).toBe('myhost')
        expect(formGroup.get('ddnsGeneratedPrefix.values.1')?.value).toBe('myhost')

        expect(formGroup.get('ddnsOverrideClientUpdate.unlocked')?.value).toBeFalse()
        expect((formGroup.get('ddnsOverrideClientUpdate.values') as UntypedFormArray).length).toBe(2)
        expect(formGroup.get('ddnsOverrideClientUpdate.values.0')?.value).toBe(false)
        expect(formGroup.get('ddnsOverrideClientUpdate.values.1')?.value).toBe(false)

        expect(formGroup.get('ddnsOverrideNoUpdate.unlocked')?.value).toBeFalse()
        expect((formGroup.get('ddnsOverrideNoUpdate.values') as UntypedFormArray).length).toBe(2)
        expect(formGroup.get('ddnsOverrideNoUpdate.values.0')?.value).toBe(false)
        expect(formGroup.get('ddnsOverrideNoUpdate.values.1')?.value).toBe(false)

        expect(formGroup.get('ddnsQualifyingSuffix.unlocked')?.value).toBeTrue()
        expect((formGroup.get('ddnsQualifyingSuffix.values') as UntypedFormArray).length).toBe(2)
        expect(formGroup.get('ddnsQualifyingSuffix.values.0')?.value).toBe('')
        expect(formGroup.get('ddnsQualifyingSuffix.values.1')?.value).toBe('example.org')

        expect(formGroup.get('ddnsReplaceClientName.unlocked')?.value).toBeFalse()
        expect((formGroup.get('ddnsReplaceClientName.values') as UntypedFormArray).length).toBe(2)
        expect(formGroup.get('ddnsReplaceClientName.values.0')?.value).toBe('never')
        expect(formGroup.get('ddnsReplaceClientName.values.1')?.value).toBe('never')

        expect(formGroup.get('ddnsSendUpdates.unlocked')?.value).toBeFalse()
        expect((formGroup.get('ddnsSendUpdates.values') as UntypedFormArray).length).toBe(2)
        expect(formGroup.get('ddnsSendUpdates.values.0')?.value).toBe(true)
        expect(formGroup.get('ddnsSendUpdates.values.1')?.value).toBe(true)

        expect(formGroup.get('ddnsUpdateOnRenew.unlocked')?.value).toBeFalse()
        expect((formGroup.get('ddnsUpdateOnRenew.values') as UntypedFormArray).length).toBe(2)
        expect(formGroup.get('ddnsUpdateOnRenew.values.0')?.value).toBe(false)
        expect(formGroup.get('ddnsUpdateOnRenew.values.1')?.value).toBe(false)

        expect(formGroup.get('earlyGlobalReservationsLookup.unlocked')?.value).toBeFalse()
        expect((formGroup.get('earlyGlobalReservationsLookup.values') as UntypedFormArray).length).toBe(2)
        expect(formGroup.get('earlyGlobalReservationsLookup.values.0')?.value).toBe(false)
        expect(formGroup.get('earlyGlobalReservationsLookup.values.1')?.value).toBe(false)

        expect(formGroup.get('echoClientId.unlocked')?.value).toBeFalse()
        expect((formGroup.get('echoClientId.values') as UntypedFormArray).length).toBe(2)
        expect(formGroup.get('echoClientId.values.0')?.value).toBe(true)
        expect(formGroup.get('echoClientId.values.1')?.value).toBe(true)

        expect(formGroup.get('expiredFlushReclaimedTimerWaitTime.unlocked')?.value).toBeTrue()
        expect((formGroup.get('expiredFlushReclaimedTimerWaitTime.values') as UntypedFormArray).length).toBe(2)
        expect(formGroup.get('expiredFlushReclaimedTimerWaitTime.values.0')?.value).toBe(25)
        expect(formGroup.get('expiredFlushReclaimedTimerWaitTime.values.1')?.value).toBe(30)

        expect(formGroup.get('expiredHoldReclaimedTime.unlocked')?.value).toBeFalse()
        expect((formGroup.get('expiredHoldReclaimedTime.values') as UntypedFormArray).length).toBe(2)
        expect(formGroup.get('expiredHoldReclaimedTime.values.0')?.value).toBe(3600)
        expect(formGroup.get('expiredHoldReclaimedTime.values.1')?.value).toBe(3600)

        expect(formGroup.get('expiredMaxReclaimLeases.unlocked')?.value).toBeFalse()
        expect((formGroup.get('expiredMaxReclaimLeases.values') as UntypedFormArray).length).toBe(2)
        expect(formGroup.get('expiredMaxReclaimLeases.values.0')?.value).toBe(100)
        expect(formGroup.get('expiredMaxReclaimLeases.values.1')?.value).toBe(100)

        expect(formGroup.get('expiredMaxReclaimTime.unlocked')?.value).toBeFalse()
        expect((formGroup.get('expiredMaxReclaimTime.values') as UntypedFormArray).length).toBe(2)
        expect(formGroup.get('expiredMaxReclaimTime.values.0')?.value).toBe(250)
        expect(formGroup.get('expiredMaxReclaimTime.values.1')?.value).toBe(250)

        expect(formGroup.get('expiredReclaimTimerWaitTime.unlocked')?.value).toBeFalse()
        expect((formGroup.get('expiredReclaimTimerWaitTime.values') as UntypedFormArray).length).toBe(2)
        expect(formGroup.get('expiredReclaimTimerWaitTime.values.0')?.value).toBe(10)
        expect(formGroup.get('expiredReclaimTimerWaitTime.values.1')?.value).toBe(10)

        expect(formGroup.get('expiredUnwarnedReclaimCycles.unlocked')?.value).toBeFalse()
        expect((formGroup.get('expiredUnwarnedReclaimCycles.values') as UntypedFormArray).length).toBe(2)
        expect(formGroup.get('expiredUnwarnedReclaimCycles.values.0')?.value).toBe(5)
        expect(formGroup.get('expiredUnwarnedReclaimCycles.values.1')?.value).toBe(5)

        expect(formGroup.get('hostReservationIdentifiers.unlocked')?.value).toBeTrue()
        expect((formGroup.get('hostReservationIdentifiers.values') as UntypedFormArray).length).toBe(2)
        expect(formGroup.get('hostReservationIdentifiers.values.0')?.value).toEqual([
            'hw-address',
            'duid',
            'circuit-id',
            'client-id',
        ])
        expect(formGroup.get('hostReservationIdentifiers.values.1')?.value).toEqual([
            'hw-address',
            'duid',
            'circuit-id',
        ])

        expect(formGroup.get('reservationsGlobal.unlocked')?.value).toBeFalse()
        expect((formGroup.get('reservationsGlobal.values') as UntypedFormArray).length).toBe(2)
        expect(formGroup.get('reservationsGlobal.values.0')?.value).toBe(false)
        expect(formGroup.get('reservationsGlobal.values.1')?.value).toBe(false)

        expect(formGroup.get('reservationsInSubnet.unlocked')?.value).toBeFalse()
        expect((formGroup.get('reservationsInSubnet.values') as UntypedFormArray).length).toBe(2)
        expect(formGroup.get('reservationsInSubnet.values.0')?.value).toBe(true)
        expect(formGroup.get('reservationsInSubnet.values.1')?.value).toBe(true)

        expect(formGroup.get('reservationsOutOfPool.unlocked')?.value).toBeFalse()
        expect((formGroup.get('reservationsOutOfPool.values') as UntypedFormArray).length).toBe(2)
        expect(formGroup.get('reservationsOutOfPool.values.0')?.value).toBe(false)
        expect(formGroup.get('reservationsOutOfPool.values.1')?.value).toBe(false)

        expect(formGroup.get('pdAllocator')).toBeFalsy()
    })

    it('should only include ddns-use-conflict-resolution when all Kea versions are earlier than 2.5.0', () => {
        const configs: KeaRawConfig[] = [
            {
                'ddns-conflict-resolution-mode': 'check-with-dhcid',
                'ddns-use-conflict-resolution': true,
            },
            {
                'ddns-conflict-resolution-mode': 'check-with-dhcid',
                'ddns-use-conflict-resolution': true,
            },
        ]
        const formGroup = service.convertKeaGlobalParametersToForm(['2.0.1', '2.2.2'], 'Dhcp4', configs)
        expect(formGroup.get('ddnsUseConflictResolution')).toBeTruthy()
        expect(formGroup.get('ddnsConflictResolutionMode')).toBeFalsy()
    })

    it('should only include ddns-conflict-resolution-mode when all Kea versions are 2.5.0 or later', () => {
        const configs: KeaRawConfig[] = [
            {
                'ddns-conflict-resolution-mode': 'check-with-dhcid',
                'ddns-use-conflict-resolution': true,
            },
            {
                'ddns-conflict-resolution-mode': 'check-with-dhcid',
                'ddns-use-conflict-resolution': true,
            },
        ]
        const formGroup = service.convertKeaGlobalParametersToForm(['2.5.0', '3.1.1'], 'Dhcp4', configs)
        expect(formGroup.get('ddnsUseConflictResolution')).toBeFalsy()
        expect(formGroup.get('ddnsConflictResolutionMode')).toBeTruthy()
    })

    it('should include ddns-conflict-resolution-mode and ddns-conflict-resolution-mode', () => {
        const configs: KeaRawConfig[] = [
            {
                'ddns-conflict-resolution-mode': 'check-with-dhcid',
                'ddns-use-conflict-resolution': true,
            },
            {
                'ddns-conflict-resolution-mode': 'check-with-dhcid',
                'ddns-use-conflict-resolution': true,
            },
        ]
        const formGroup = service.convertKeaGlobalParametersToForm(['2.2.0', '3.1.1'], 'Dhcp4', configs)
        expect(formGroup.get('ddnsUseConflictResolution')).toBeTruthy()
        expect(formGroup.get('ddnsConflictResolutionMode')).toBeTruthy()
    })

    it('should convert DHCPv6 global parameters to form', () => {
        const configs: KeaRawConfig[] = [
            {
                allocator: 'iterative',
                'pd-allocator': 'flq',
            },
        ]
        const formGroup = service.convertKeaGlobalParametersToForm(null, 'Dhcp6', configs)
        expect(formGroup.get('allocator.unlocked')?.value).toBeFalse()
        expect((formGroup.get('allocator.values') as UntypedFormArray).length).toBe(1)
        expect(formGroup.get('allocator.values.0')?.value).toBe('iterative')

        expect(formGroup.get('pdAllocator.unlocked')?.value).toBeFalse()
        expect((formGroup.get('pdAllocator.values') as UntypedFormArray).length).toBe(1)
        expect(formGroup.get('pdAllocator.values.0')?.value).toBe('flq')

        expect(formGroup.get('authoritative')).toBeFalsy()
    })

    it('should convert global DHCPv4 configurations to form - single config', () => {
        const configs: KeaDaemonConfig[] = [
            {
                appId: 1,
                appName: 'kea',
                daemonId: 1,
                daemonName: 'dhcp6',
                config: {
                    Dhcp6: {
                        allocator: 'iterative',
                    },
                },
                options: {
                    options: [
                        {
                            alwaysSend: true,
                            code: 42,
                            encapsulate: '',
                            universe: 6,
                        },
                    ],
                },
            },
        ]

        const form = service.convertKeaGlobalConfigurationToForm(null, configs)

        expect(form).toBeDefined()

        const parameters = form.get('parameters') as FormGroup<KeaGlobalParametersForm>
        expect(parameters).toBeDefined()
        expect(parameters.get('allocator.unlocked')?.value).toBeFalse()
        expect(parameters.get('allocator.unlocked')?.value).toBeFalse()
        expect((parameters.get('allocator.values') as UntypedFormArray).length).toBe(1)
        expect(parameters.get('allocator.values.0')?.value).toBe('iterative')

        const options = form.get('options') as FormGroup<OptionsForm>
        expect(options).toBeDefined()
        expect(options.get('unlocked')?.value).toBeFalse()
        expect((options.get('data') as UntypedFormArray).length).toBe(1)
        expect(options.get('data.0.0.optionCode')?.value).toBe(42)
    })

    it('should convert global DHCPv6 configurations to form - many configs', () => {
        const configs: KeaDaemonConfig[] = [
            {
                appId: 1,
                appName: 'kea',
                daemonId: 1,
                daemonName: 'dhcp4',
                config: {
                    Dhcp4: {
                        allocator: 'iterative',
                    },
                },
                options: {
                    options: [
                        {
                            alwaysSend: true,
                            code: 42,
                            encapsulate: '',
                            universe: 4,
                        },
                    ],
                    optionsHash: 'foo',
                },
            },
            {
                appId: 2,
                appName: 'kea',
                daemonId: 2,
                daemonName: 'dhcp4',
                config: {
                    Dhcp4: {
                        allocator: 'random',
                    },
                },
                options: {
                    options: [
                        {
                            alwaysSend: false,
                            code: 24,
                            encapsulate: '',
                            universe: 4,
                        },
                    ],
                    optionsHash: 'true',
                },
            },
        ]

        const form = service.convertKeaGlobalConfigurationToForm(null, configs)

        expect(form).toBeDefined()

        const parameters = form.get('parameters') as FormGroup<KeaGlobalParametersForm>
        expect(parameters).toBeDefined()
        expect(parameters.get('allocator.unlocked')?.value).toBeTrue()
        expect(parameters.get('allocator.unlocked')?.value).toBeTrue()
        expect((parameters.get('allocator.values') as UntypedFormArray).length).toBe(2)
        expect(parameters.get('allocator.values.0')?.value).toBe('iterative')
        expect(parameters.get('allocator.values.1')?.value).toBe('random')

        const options = form.get('options') as FormGroup<OptionsForm>
        expect(options).toBeDefined()
        expect(options.get('unlocked')?.value).toBeTrue()
        expect((options.get('data') as UntypedFormArray).length).toBe(2)
        expect(options.get('data.0.0.optionCode')?.value).toBe(42)
        expect(options.get('data.1.0.optionCode')?.value).toBe(24)
    })

    it('should convert global DHCPv6 configurations to form - daemon mishmash', () => {
        const configs: KeaDaemonConfig[] = [
            {
                appId: 1,
                appName: 'kea',
                daemonId: 1,
                daemonName: 'dhcp4',
                config: {
                    Dhcp4: {
                        allocator: 'iterative',
                    },
                },
            },
            {
                appId: 2,
                appName: 'kea',
                daemonId: 2,
                daemonName: 'dhcp6',
                config: {
                    Dhcp6: {
                        allocator: 'random',
                    },
                },
            },
        ]

        const form = service.convertKeaGlobalConfigurationToForm(null, configs)

        expect(form).toBeNull()
    })

    it('should convert a form to Kea parameters', () => {
        const params = service.convertFormToKeaSubnetParameters(
            [],
            new FormGroup<KeaGlobalParametersForm>({
                cacheThreshold: new SharedParameterFormGroup<number>(
                    {
                        type: 'number',
                        min: 0,
                        max: 1,
                        fractionDigits: 2,
                    },
                    [new FormControl<number>(0.5), new FormControl<number>(0.5)]
                ),
                allocator: new SharedParameterFormGroup<string>(
                    {
                        type: 'string',
                        values: ['iterative', 'random', 'flq'],
                    },
                    [new FormControl<string>('flq'), new FormControl<string>('random')]
                ),
                authoritative: new SharedParameterFormGroup<boolean>(
                    {
                        type: 'boolean',
                    },
                    [new FormControl<boolean>(true), new FormControl<boolean>(false), new FormControl<boolean>(false)]
                ),
            })
        )
        expect(params.length).toBe(3)
        expect(params[0].cacheThreshold).toBe(0.5)
        expect(params[0].allocator).toBe('flq')
        expect(params[0].authoritative).toBeTrue()
        expect(params[1].cacheThreshold).toBe(0.5)
        expect(params[1].allocator).toBe('random')
        expect(params[1].authoritative).toBeFalse()
        expect(params[2].cacheThreshold).toBeFalsy()
        expect(params[2].allocator).toBeFalsy()
        expect(params[2].authoritative).toBeFalse()
    })

    it('should filter out unsupported parameters during form to global parameters conversion', () => {
        // Specify two daemons. The first one has version 2.2.0 which only supports
        // the ddns-use-conflict-resolution. The latter is 3.0.0 which supports
        // ddns-conflict-resolution-mode.
        const daemons: VersionedDaemon[] = [
            {
                id: 1,
                version: '2.2.0',
            },
            {
                id: 2,
                version: '3.0.0',
            },
        ]
        const params = service.convertFormToKeaSubnetParameters(
            daemons,
            new FormGroup<KeaGlobalParametersForm>({
                ddnsUseConflictResolution: new SharedParameterFormGroup<boolean>(
                    {
                        type: 'boolean',
                        versionUpperBound: '2.5.0',
                    },
                    [new FormControl(true), new FormControl(false)]
                ),
                ddnsConflictResolutionMode: new SharedParameterFormGroup<string>(
                    {
                        type: 'string',
                        versionLowerBound: '2.5.0',
                    },
                    [new FormControl<string>('check-with-dhcid'), new FormControl<string>('check-with-dhcid')]
                ),
            })
        )
        console.info(params)
        expect(params.length).toBe(2)
        expect(params[0].ddnsUseConflictResolution).toBe(true)
        expect(params[0].ddnsConflictResolutionMode).toBeFalsy()
    })

    it('should convert a form to the global Kea configuration', () => {
        const form = new FormGroup<KeaGlobalConfigurationForm>({
            parameters: new FormGroup<KeaGlobalParametersForm>({
                cacheThreshold: new SharedParameterFormGroup<number>(
                    {
                        type: 'number',
                        min: 0,
                        max: 1,
                        fractionDigits: 2,
                    },
                    [new FormControl<number>(0.5), new FormControl<number>(0.5)]
                ),
                allocator: new SharedParameterFormGroup<string>(
                    {
                        type: 'string',
                        values: ['iterative', 'random', 'flq'],
                    },
                    [new FormControl<string>('flq'), new FormControl<string>('random')]
                ),
                authoritative: new SharedParameterFormGroup<boolean>(
                    {
                        type: 'boolean',
                    },
                    [new FormControl<boolean>(true), new FormControl<boolean>(false)]
                ),
            }),
            options: new FormGroup<OptionsForm>({
                unlocked: new FormControl<boolean>(true),
                data: new UntypedFormArray([
                    new UntypedFormArray([
                        new FormGroup({
                            alwaysSend: new FormControl(false),
                            optionCode: new FormControl(6),
                            optionFields: new UntypedFormArray([]),
                            suboptions: new UntypedFormArray([]),
                        }),
                    ]),
                    new UntypedFormArray([
                        new FormGroup({
                            alwaysSend: new FormControl(true),
                            optionCode: new FormControl(42),
                            optionFields: new UntypedFormArray([]),
                            suboptions: new UntypedFormArray([]),
                        }),
                    ]),
                ]),
            }),
        })

        const configs = service.convertFormToKeaGlobalParameters(null, form, IPType.IPv4)
        expect(configs).toBeDefined()
        expect(configs.length).toBe(2)

        let config = configs[0]
        expect(config.cacheThreshold).toBe(0.5)
        expect(config.allocator).toBe('flq')
        expect(config.authoritative).toBeTrue()
        expect(config.options).toBeDefined()
        expect(config.options.length).toBe(1)
        expect(config.options[0].alwaysSend).toBeFalse()
        expect(config.options[0].code).toBe(6)

        config = configs[1]
        expect(config.cacheThreshold).toBe(0.5)
        expect(config.allocator).toBe('random')
        expect(config.authoritative).toBeFalse()
        expect(config.options).toBeDefined()
        expect(config.options.length).toBe(1)
        expect(config.options[0].alwaysSend).toBeTrue()
        expect(config.options[0].code).toBe(42)
    })
})
