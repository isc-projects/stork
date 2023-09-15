import { TestBed } from '@angular/core/testing'

import { KeaSubnetParametersForm, SubnetForm, SubnetSetFormService } from './subnet-set-form.service'
import { KeaConfigSubnetDerivedParameters, Subnet } from '../backend'
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
                t1Percent: 45,
                t2Percent: 65,
                calculateTeeTimes: true,
                validLifetime: 1000,
                minValidLifetime: 1000,
                maxValidLifetime: 1000,
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
            },
            {
                cacheThreshold: 0.5,
                cacheMaxAge: 2000,
                clientClass: 'bar',
                requireClientClasses: ['foo'],
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
                minPreferredLifetime: 2000,
                maxPreferredLifetime: 2000,
                reservationsGlobal: false,
                reservationsInSubnet: false,
                reservationsOutOfPool: false,
                renewTimer: 1500,
                rebindTimer: 2500,
                t1Percent: 55,
                t2Percent: 75,
                calculateTeeTimes: false,
                validLifetime: 2000,
                minValidLifetime: 2000,
                maxValidLifetime: 2000,
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
            },
        ]
        let form = service.convertKeaParametersToForm(IPType.IPv4, parameters)
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
        expect((fg.get('unlocked') as UntypedFormControl)?.value).toBeTrue()
        expect((fg.get('values') as UntypedFormArray)?.controls.length).toBe(2)
        expect((fg.get('values') as UntypedFormArray)?.controls[0].value).toEqual(['foo', 'bar'])
        expect((fg.get('values') as UntypedFormArray)?.controls[1].value).toEqual(['foo'])

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
        expect(fg.data.max).toBe(100)
        expect(fg.data.fractionDigits).toBeFalsy()
        expect(fg.data.values).toBeFalsy()
        expect((fg.get('unlocked') as UntypedFormControl)?.value).toBeTrue()
        expect((fg.get('values') as UntypedFormArray)?.controls.length).toBe(2)
        expect((fg.get('values') as UntypedFormArray)?.controls[0].value).toBe(45)
        expect((fg.get('values') as UntypedFormArray)?.controls[1].value).toBe(55)

        fg = form.get('t2Percent') as SharedParameterFormGroup<any>
        expect(fg).toBeTruthy()
        expect(fg.data.type).toBe('number')
        expect(fg.data.min).toBe(0)
        expect(fg.data.max).toBe(100)
        expect(fg.data.fractionDigits).toBeFalsy()
        expect(fg.data.values).toBeFalsy()
        expect((fg.get('unlocked') as UntypedFormControl)?.value).toBeTrue()
        expect((fg.get('values') as UntypedFormArray)?.controls.length).toBe(2)
        expect((fg.get('values') as UntypedFormArray)?.controls[0].value).toBe(65)
        expect((fg.get('values') as UntypedFormArray)?.controls[1].value).toBe(75)

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
        expect((fg.get('values') as UntypedFormArray)?.controls[0].value).toBe(1000)
        expect((fg.get('values') as UntypedFormArray)?.controls[1].value).toBe(2000)

        fg = form.get('minValidLifetime') as SharedParameterFormGroup<any>
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

        fg = form.get('maxValidLifetime') as SharedParameterFormGroup<any>
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
        let form = service.convertKeaParametersToForm(IPType.IPv6, parameters)
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
        let form = service.convertKeaParametersToForm(IPType.IPv4, parameters)
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
        let form = service.convertKeaParametersToForm(IPType.IPv4, parameters)
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
        let form = service.convertKeaParametersToForm(IPType.IPv4, parameters)
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
        let form = service.convertKeaParametersToForm(IPType.IPv4, parameters)
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
        let form = service.convertKeaParametersToForm(IPType.IPv4, parameters)
        let fg = form.get('serverHostname') as SharedParameterFormGroup<any>
        expect(fg).toBeTruthy()
        expect(fg.valid).toBeFalse()
    })

    it('should create a default form for an IPv4 subnet', () => {
        let form = service.createDefaultKeaParametersForm(IPType.IPv4)
        expect(Object.keys(form.controls).length).toBe(37)

        for (const key of Object.keys(form.controls)) {
            let control = form.get(key) as SharedParameterFormGroup<any>
            expect(control).toBeTruthy()
            expect(control.controls?.unlocked?.value).toBeFalse()
            expect(control.controls?.values?.value.length).toBe(1)
        }
    })

    it('should create a default form for an IPv6 subnet', () => {
        let form = service.createDefaultKeaParametersForm(IPType.IPv6)
        expect(Object.keys(form.controls).length).toBe(35)

        for (const key of Object.keys(form.controls)) {
            let control = form.get(key) as SharedParameterFormGroup<any>
            expect(control).toBeTruthy()
            expect(control.controls?.unlocked?.value).toBeFalse()
            expect(control.controls?.values?.value.length).toBe(1)
        }
    })

    it('should convert IPv4 subnet data to a form', () => {
        const subnet: Subnet = {
            subnet: '192.0.2.0/24',
            localSubnets: [
                {
                    daemonId: 1,
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
        const formGroup = service.convertSubnetToForm(IPType.IPv4, subnet)
        expect(formGroup.get('subnet')?.value).toBe('192.0.2.0/24')
        expect(formGroup.get('subnet')?.disabled).toBeTrue()

        const options = formGroup.get('options.data') as UntypedFormArray
        expect(options.length).toBe(1)
        expect(options.get('0.0.optionFields.0.control')?.value).toBe('192.0.2.1')

        const parameters = formGroup.get('parameters') as FormGroup<KeaSubnetParametersForm>
        expect(parameters.get('allocator.unlocked')?.value).toBeFalse()
        expect((parameters.get('allocator.values') as UntypedFormArray).length).toBe(1)
        expect(parameters.get('allocator.values.0')?.value).toBe('random')

        const selectedDaemons = formGroup.get('selectedDaemons') as FormControl<number[]>
        expect(selectedDaemons?.value).toEqual([1])
    })

    it('should convert IPv6 subnet data to a form', () => {
        const subnet: Subnet = {
            subnet: '2001:db8:1::/64',
            localSubnets: [
                {
                    daemonId: 1,
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
        const formGroup = service.convertSubnetToForm(IPType.IPv6, subnet)
        expect(formGroup.get('subnet')?.value).toBe('2001:db8:1::/64')
        expect(formGroup.get('subnet')?.disabled).toBeTrue()

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
    })

    it('should convert a subnet with no local subnets to a form', () => {
        const subnet: Subnet = {
            subnet: '192.0.2.0/24',
        }
        const formGroup = service.convertSubnetToForm(IPType.IPv4, subnet)
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
        const params = service.convertFormToKeaParameters(
            new FormGroup<KeaSubnetParametersForm>({
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

    it('should convert a form to subnet', () => {
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
                        },
                    },
                },
            ],
        }
        const formGroup = service.convertSubnetToForm(IPType.IPv4, subnet0)

        const subnet1 = service.convertFormToSubnet(formGroup)

        expect(subnet1.subnet).toBe('192.0.2.0/24')
        expect(subnet1.localSubnets.length).toBe(2)

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
        const formGroup = service.convertSubnetToForm(IPType.IPv4, subnet0)

        // Both servers have the same options so the options are locked by default.
        // Let's now modify the second option instance while they are locked. The
        // second conversion should ignore this modification and take the first
        // option for each server.
        formGroup.get('options.data.1.0.optionFields.0.control')?.setValue('10.1.1.1')

        const subnet1 = service.convertFormToSubnet(formGroup)
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
})
