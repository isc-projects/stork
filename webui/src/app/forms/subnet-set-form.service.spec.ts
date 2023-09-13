import { TestBed } from '@angular/core/testing'

import { SubnetSetFormService } from './subnet-set-form.service'
import { KeaConfigSubnetDerivedParameters } from '../backend'
import { SharedParameterFormGroup } from './shared-parameter-form-group'
import { UntypedFormArray, UntypedFormControl } from '@angular/forms'

describe('SubnetSetFormService', () => {
    let service: SubnetSetFormService

    beforeEach(() => {
        TestBed.configureTestingModule({})
        service = TestBed.inject(SubnetSetFormService)
    })

    it('should be created', () => {
        expect(service).toBeTruthy()
    })

    it('should convert Kea parameters to a form group', () => {
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
        let form = service.convertKeaParametersToForm(parameters)
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

        fg = form.get('minPreferredLifetime') as SharedParameterFormGroup<any>
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

        fg = form.get('maxPreferredLifetime') as SharedParameterFormGroup<any>
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

        fg = form.get('rapidCommit') as SharedParameterFormGroup<any>
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

    it('should use a validator for generated prefix', () => {
        let parameters: KeaConfigSubnetDerivedParameters[] = [
            {
                ddnsGeneratedPrefix: '-invalid.prefix',
            },
        ]
        let form = service.convertKeaParametersToForm(parameters)
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
        let form = service.convertKeaParametersToForm(parameters)
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
        let form = service.convertKeaParametersToForm(parameters)
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
        let form = service.convertKeaParametersToForm(parameters)
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
        let form = service.convertKeaParametersToForm(parameters)
        let fg = form.get('serverHostname') as SharedParameterFormGroup<any>
        expect(fg).toBeTruthy()
        expect(fg.valid).toBeFalse()
    })
})
