import { FormArray, FormControl, FormGroup, UntypedFormBuilder } from '@angular/forms'
import { StorkValidators } from './validators'
import {
    AddressPoolForm,
    AddressRangeForm,
    KeaPoolParametersForm,
    PrefixForm,
    PrefixPoolForm,
} from './forms/subnet-set-form.service'
import { SharedParameterFormGroup } from './forms/shared-parameter-form-group'

describe('StorkValidators', () => {
    let formBuilder: UntypedFormBuilder = new UntypedFormBuilder()

    it('validates hex identifier', () => {
        // Doesn't contain hexadecimal digits.
        expect(StorkValidators.hexIdentifier()(formBuilder.control('value'))).toBeTruthy()
        // Good identifier with colons as separator.
        expect(StorkValidators.hexIdentifier()(formBuilder.control('01:02:03'))).toBeFalsy()
        // Good identifier with dashes as a separator.
        expect(StorkValidators.hexIdentifier()(formBuilder.control('ca-fe-03'))).toBeFalsy()
        // Spaces as separator are supported.
        expect(StorkValidators.hexIdentifier()(formBuilder.control('ca fe 03'))).toBeFalsy()
        // Empty string is fine for this validator.
        expect(StorkValidators.hexIdentifier()(formBuilder.control(''))).toBeFalsy()
    })

    it('validates hex identifier length', () => {
        expect(StorkValidators.hexIdentifierLength(6)(formBuilder.control('01:02:03'))).toBeFalsy()
        expect(StorkValidators.hexIdentifierLength(8)(formBuilder.control('112233'))).toBeFalsy()
        expect(StorkValidators.hexIdentifierLength(10)(formBuilder.control('ac-de-aa'))).toBeFalsy()
        expect(StorkValidators.hexIdentifierLength(10)(formBuilder.control('ac de aa'))).toBeFalsy()

        expect(StorkValidators.hexIdentifierLength(4)(formBuilder.control('ab-cd-ef'))).toBeTruthy()
        expect(StorkValidators.hexIdentifierLength(2)(formBuilder.control('ee:02:90'))).toBeTruthy()
        expect(StorkValidators.hexIdentifierLength(6)(formBuilder.control('01020389'))).toBeTruthy()
        expect(StorkValidators.hexIdentifierLength(4)(formBuilder.control('ab cd ef'))).toBeTruthy()
    })

    it('validates IPv4 address', () => {
        // Partial address is not valid.
        expect(StorkValidators.ipv4()(formBuilder.control('10.0.'))).toBeTruthy()
        // Prefix is not valid.
        expect(StorkValidators.ipv4()(formBuilder.control('10.0.0.0/24'))).toBeTruthy()
        // Too many dots.
        expect(StorkValidators.ipv4()(formBuilder.control('192.0..2.1'))).toBeTruthy()
        // Dot after address.
        expect(StorkValidators.ipv4()(formBuilder.control('192.0.2.1.'))).toBeTruthy()
        // IPv6 address is not valid.
        expect(StorkValidators.ipv4()(formBuilder.control('2001:db8:1::1'))).toBeTruthy()
        // Must use dots to separate the IP address bytes.
        expect(StorkValidators.ipv4()(formBuilder.control('192x0x2x1'))).toBeTruthy()
        // Too high numbers.
        expect(StorkValidators.ipv4()(formBuilder.control('999.999.999.999'))).toBeTruthy()
        // Valid address.
        expect(StorkValidators.ipv4()(formBuilder.control('192.0.2.1'))).toBeFalsy()
    })

    it('validates IPv4 addresses array', () => {
        // Partial address is not valid.
        expect(StorkValidators.ipv4()(formBuilder.control(['10.0.', '10.1.1.1']))).toBeTruthy()
        // Prefix is not valid.
        expect(StorkValidators.ipv4()(formBuilder.control(['8.8.8.8', '10.0.0.0/24']))).toBeTruthy()
        // IPv6 address is not valid.
        expect(StorkValidators.ipv4()(formBuilder.control(['2001:db8:1::1', '8.8.8.8']))).toBeTruthy()
        // Valid addresses.
        expect(StorkValidators.ipv4()(formBuilder.control(['192.0.2.1', '192.0.2.2']))).toBeFalsy()
    })

    it('validates IPv6 address', () => {
        // Partial address is not valid.
        expect(StorkValidators.ipv6()(formBuilder.control('2001'))).toBeTruthy()
        // Dots are not valid.
        expect(StorkValidators.ipv6()(formBuilder.control('3000..'))).toBeTruthy()
        // No colons at the end.
        expect(StorkValidators.ipv6()(formBuilder.control('3001:123'))).toBeTruthy()
        // IPv4 address is not valid.
        expect(StorkValidators.ipv6()(formBuilder.control('192.0.2.1'))).toBeTruthy()
        // Valid address.
        expect(StorkValidators.ipv6()(formBuilder.control('3000:1:2::3'))).toBeFalsy()
    })

    it('validates IPv6 addresses array', () => {
        // Partial address is not valid.
        expect(StorkValidators.ipv6()(formBuilder.control(['2001', '2001:db8:1::']))).toBeTruthy()
        // Dots are not valid.
        expect(StorkValidators.ipv6()(formBuilder.control(['2001:db8:1::1', '3000..']))).toBeTruthy()
        // IPv4 address is not valid.
        expect(StorkValidators.ipv6()(formBuilder.control(['192.0.2.1', '2001:db8:1::1']))).toBeTruthy()
        // Valid addresses.
        expect(StorkValidators.ipv6()(formBuilder.control(['3000:1:2::3', '2001:db8:1::1']))).toBeFalsy()
    })

    it('validates IPv6 prefix', () => {
        // Partial prefix is not valid.
        expect(StorkValidators.ipv6Prefix(formBuilder.control('2001::'))).toBeTruthy()
        // Dots are not valid.
        expect(StorkValidators.ipv6Prefix(formBuilder.control('3000../64'))).toBeTruthy()
        // No colons at the end.
        expect(StorkValidators.ipv6Prefix(formBuilder.control('3001:123/64'))).toBeTruthy()
        // IPv4 prefix is not valid.
        expect(StorkValidators.ipv6Prefix(formBuilder.control('192.0.2.0/24'))).toBeTruthy()
        // Prefix length must match the number of meaningful bits.
        expect(StorkValidators.ipv6Prefix(formBuilder.control('2001:db8:1::/18'))).toBeTruthy()
        // Valid prefix.
        expect(StorkValidators.ipv6Prefix(formBuilder.control('3000:1:2::/64'))).toBeFalsy()
    })

    it('validates IP prefix', () => {
        // Partial prefix is not valid.
        expect(StorkValidators.ipPrefix(formBuilder.control('2001::'))).toBeTruthy()
        expect(StorkValidators.ipPrefix(formBuilder.control('192.'))).toBeTruthy()
        // Dots are not valid.
        expect(StorkValidators.ipPrefix(formBuilder.control('3000../64'))).toBeTruthy()
        // Colons are not valid.
        expect(StorkValidators.ipPrefix(formBuilder.control('192:0:2:0/64'))).toBeTruthy()
        // No colons at the end.
        expect(StorkValidators.ipPrefix(formBuilder.control('3001:123/64'))).toBeTruthy()
        // Incorrect IPv4 prefix length
        expect(StorkValidators.ipPrefix(formBuilder.control('192.0.2.0/36'))).toBeTruthy()
        // Incorrect IPv6 prefix length
        expect(StorkValidators.ipPrefix(formBuilder.control('2001:db8:1::/129'))).toBeTruthy()
        // The prefix must match the prefix length (mask).
        expect(StorkValidators.ipPrefix(formBuilder.control('192.110.111.0/23'))).toBeTruthy()
        expect(StorkValidators.ipPrefix(formBuilder.control('2001:db8:1::/16'))).toBeTruthy()
        // IPv4 prefix is valid.
        expect(StorkValidators.ipPrefix(formBuilder.control('192.0.2.0/24'))).toBeFalsy()
        // IPv6 prefix is valid too.
        expect(StorkValidators.ipPrefix(formBuilder.control('3000:1:2::/64'))).toBeFalsy()
    })

    it('validates if an address is in the IPv4 subnet', () => {
        // Skip validation when controls have no meaningful value.
        expect(StorkValidators.ipInSubnet('192.0.2.0/24')(formBuilder.control(null))).toEqual({
            ipInSubnet: 'Please specify an IP address belonging to 192.0.2.0/24.',
        })
        expect(StorkValidators.ipInSubnet('192.0.2.0/24')(formBuilder.control(65))).toEqual({
            ipInSubnet: 'Please specify an IP address belonging to 192.0.2.0/24.',
        })
        expect(StorkValidators.ipInSubnet('192.0.2.0/24')(formBuilder.control(''))).toEqual({
            ipInSubnet: 'Please specify an IP address belonging to 192.0.2.0/24.',
        })
        // Valid address belongs to the subnet.
        expect(StorkValidators.ipInSubnet('192.0.2.0/24')(formBuilder.control('192.0.2.100'))).toBeFalsy()
        // Outside of a subnet.
        expect(StorkValidators.ipInSubnet('192.0.2.0/24')(formBuilder.control('192.0.3.100'))).toEqual({
            ipInSubnet: '192.0.3.100 does not belong to subnet 192.0.2.0/24.',
        })
        // Wrong family.
        expect(StorkValidators.ipInSubnet('192.0.2.0/24')(formBuilder.control('2001:db8:1::10'))).toEqual({
            ipInSubnet: '2001:db8:1::10 is not a valid IPv4 address.',
        })
        // Invalid subnet.
        expect(StorkValidators.ipInSubnet('192.0.2.0')(formBuilder.control('192.0.2.1'))).toEqual({
            ipInSubnet: '192.0.2.0 is not a valid subnet prefix.',
        })
        expect(StorkValidators.ipInSubnet('/24')(formBuilder.control('192.0.2.1'))).toEqual({
            ipInSubnet: '/24 is not a valid subnet prefix.',
        })
    })

    it('validates if an address is in the IPv6 subnet', () => {
        // Skip validation when controls have no meaningful value.
        expect(StorkValidators.ipInSubnet('2001:db8:1::/64')(formBuilder.control(null))).toEqual({
            ipInSubnet: 'Please specify an IP address belonging to 2001:db8:1::/64.',
        })
        expect(StorkValidators.ipInSubnet('2001:db8:1::/64')(formBuilder.control(65))).toEqual({
            ipInSubnet: 'Please specify an IP address belonging to 2001:db8:1::/64.',
        })
        expect(StorkValidators.ipInSubnet('2001:db8:1::/64')(formBuilder.control(''))).toEqual({
            ipInSubnet: 'Please specify an IP address belonging to 2001:db8:1::/64.',
        })
        // Valid address belongs to the subnet.
        expect(StorkValidators.ipInSubnet('2001:db8:1::/64')(formBuilder.control('2001:db8:1::3:ff00'))).toBeFalsy()
        // Outside of a subnet.
        expect(StorkValidators.ipInSubnet('2001:db8:1::/64')(formBuilder.control('2001:db8:2::3:ff00'))).toEqual({
            ipInSubnet: '2001:db8:2::3:ff00 does not belong to subnet 2001:db8:1::/64.',
        })
        // Wrong family.
        expect(StorkValidators.ipInSubnet('2001:db8:1::/64')(formBuilder.control('192.0.2.1'))).toEqual({
            ipInSubnet: '192.0.2.1 is not a valid IPv6 address.',
        })
        // Invalid subnet.
        expect(StorkValidators.ipInSubnet('/')(formBuilder.control('192.0.2.1'))).toEqual({
            ipInSubnet: '/ is not a valid subnet prefix.',
        })
    })

    it('validates IPv4 address range', () => {
        let fg = formBuilder.group({
            start: formBuilder.control('192.0.2.1'),
            end: formBuilder.control('192.0.2.5'),
        })
        expect(StorkValidators.ipRangeBounds(fg)).toBeFalsy()
        expect(fg.valid).toBeTrue()

        fg = formBuilder.group({
            start: formBuilder.control('192.0.2.5'),
            end: formBuilder.control('192.0.2.5'),
        })
        expect(StorkValidators.ipRangeBounds(fg)).toBeFalsy()
        expect(fg.valid).toBeTrue()

        fg = formBuilder.group({
            start: formBuilder.control('192.0.2.5'),
            end: formBuilder.control('192.0.2.1'),
        })
        expect(StorkValidators.ipRangeBounds(fg)).toEqual({
            addressBounds:
                'Invalid address pool boundaries. Make sure that the first address is equal or lower than the last address.',
        })
        expect(fg.get('start').invalid).toBeTrue()
        expect(fg.get('end').invalid).toBeTrue()

        fg = formBuilder.group({
            start: formBuilder.control('192.0.2.1'),
            end: formBuilder.control('192.0.2.'),
        })
        expect(StorkValidators.ipRangeBounds(fg)).toEqual({
            addressBounds:
                'Invalid address pool boundaries. Make sure that the first address is equal or lower than the last address.',
        })
        expect(fg.get('start').invalid).toBeTrue()
        expect(fg.get('end').invalid).toBeTrue()
    })

    it('validates IPv6 address range', () => {
        let fg = formBuilder.group({
            start: formBuilder.control('2001:db8:1::'),
            end: formBuilder.control('2001:db8:2::ffff'),
        })
        expect(StorkValidators.ipRangeBounds(fg)).toBeFalsy()
        expect(fg.valid).toBeTrue()

        fg = formBuilder.group({
            start: formBuilder.control('2001:db8:1::'),
            end: formBuilder.control('2001:db8:1::'),
        })
        expect(StorkValidators.ipRangeBounds(fg)).toBeFalsy()
        expect(fg.valid).toBeTrue()

        fg = formBuilder.group({
            start: formBuilder.control('2001:db8:2::ffff'),
            end: formBuilder.control('2001:db8:1::'),
        })
        expect(StorkValidators.ipRangeBounds(fg)).toEqual({
            addressBounds:
                'Invalid address pool boundaries. Make sure that the first address is equal or lower than the last address.',
        })
        expect(fg.get('start').invalid).toBeTrue()
        expect(fg.get('end').invalid).toBeTrue()

        fg = formBuilder.group({
            start: formBuilder.control('2001:db8:1::'),
            end: formBuilder.control('2001:db8:x::'),
        })
        expect(StorkValidators.ipRangeBounds(fg)).toEqual({
            addressBounds:
                'Invalid address pool boundaries. Make sure that the first address is equal or lower than the last address.',
        })
        expect(fg.get('start').invalid).toBeTrue()
        expect(fg.get('end').invalid).toBeTrue()
    })

    it('detects overlaps in the IPv4 address ranges', () => {
        let fa = new FormArray([
            new FormGroup<AddressPoolForm>({
                range: new FormGroup<AddressRangeForm>({
                    start: new FormControl('192.0.2.50'),
                    end: new FormControl('192.0.2.60'),
                }),
                parameters: new FormGroup<KeaPoolParametersForm>({}),
                options: new FormGroup({
                    unlocked: new FormControl(false),
                    data: new FormArray([]),
                }),
                selectedDaemons: new FormControl([]),
            }),
            new FormGroup<AddressPoolForm>({
                range: new FormGroup<AddressRangeForm>({
                    start: new FormControl('192.0.2.5'),
                    end: new FormControl('192.0.2.15'),
                }),
                parameters: new FormGroup<KeaPoolParametersForm>({}),
                options: new FormGroup({
                    unlocked: new FormControl(false),
                    data: new FormArray([]),
                }),
                selectedDaemons: new FormControl([]),
            }),
            new FormGroup<AddressPoolForm>({
                range: new FormGroup<AddressRangeForm>({
                    start: new FormControl('192.0.2.49'),
                    end: new FormControl('192.0.2.51'),
                }),
                parameters: new FormGroup<KeaPoolParametersForm>({}),
                options: new FormGroup({
                    unlocked: new FormControl(false),
                    data: new FormArray([]),
                }),
                selectedDaemons: new FormControl([]),
            }),
            new FormGroup<AddressPoolForm>({
                range: new FormGroup<AddressRangeForm>({
                    start: new FormControl('192.0.2.100'),
                    end: new FormControl('192.0.2.115'),
                }),
                parameters: new FormGroup<KeaPoolParametersForm>({}),
                options: new FormGroup({
                    unlocked: new FormControl(false),
                    data: new FormArray([]),
                }),
                selectedDaemons: new FormControl([]),
            }),
            new FormGroup<AddressPoolForm>({
                range: new FormGroup<AddressRangeForm>({
                    start: new FormControl('192.0.2.88'),
                    end: new FormControl('192.0.2.100'),
                }),
                parameters: new FormGroup<KeaPoolParametersForm>({}),
                options: new FormGroup({
                    unlocked: new FormControl(false),
                    data: new FormArray([]),
                }),
                selectedDaemons: new FormControl([]),
            }),
        ])
        expect(StorkValidators.ipRangeOverlaps(fa)).toBeTruthy()

        // Range 0 overlaps with range 2.
        expect(fa.at(0).invalid).toBeTrue()
        // Range 1 does not overlap.
        expect(fa.at(1).invalid).toBeFalse()
        // Range 2 overlaps with range 0.
        expect(fa.at(2).invalid).toBeTrue()
        // Range 3 overlaps with range 4.
        expect(fa.at(3).invalid).toBeTrue()
        // Range 4 overlaps with range 3.
        expect(fa.at(4).invalid).toBeTrue()

        // Correct the ranges.
        fa.get('2.range.end')?.setValue('192.0.2.49')
        fa.get('3.range.start')?.setValue('192.0.2.101')
        expect(StorkValidators.ipRangeOverlaps(fa)).toBeFalsy()
        expect(fa.invalid).toBeFalse()
    })

    it('clears detected overlaps for a single IPv6 range', () => {
        let fa = new FormArray([
            new FormGroup<AddressPoolForm>({
                range: new FormGroup<AddressRangeForm>({
                    start: new FormControl('2001:db8:1::1'),
                    end: new FormControl('2001:db8:1::100'),
                }),
                parameters: new FormGroup<KeaPoolParametersForm>({}),
                options: new FormGroup({
                    unlocked: new FormControl(false),
                    data: new FormArray([]),
                }),
                selectedDaemons: new FormControl([]),
            }),
            new FormGroup<AddressPoolForm>({
                range: new FormGroup<AddressRangeForm>({
                    start: new FormControl('2001:db8:1::1'),
                    end: new FormControl('2001:db8:1::1'),
                }),
                parameters: new FormGroup<KeaPoolParametersForm>({}),
                options: new FormGroup({
                    unlocked: new FormControl(false),
                    data: new FormArray([]),
                }),
                selectedDaemons: new FormControl([]),
            }),
        ])
        expect(StorkValidators.ipRangeOverlaps(fa)).toBeTruthy()

        expect(fa.at(0).invalid).toBeTrue()
        expect(fa.at(1).invalid).toBeTrue()

        // Remove the second prefix.
        fa.removeAt(1)
        expect(StorkValidators.ipRangeOverlaps(fa)).toBeFalsy()
        expect(fa.invalid).toBeFalse()
    })

    it('clears detected overlaps when IPv6 range gets invalid', () => {
        let fa = new FormArray([
            new FormGroup<AddressPoolForm>({
                range: new FormGroup<AddressRangeForm>({
                    start: new FormControl('2001:db8:1::1'),
                    end: new FormControl('2001:db8:1::100'),
                }),
                parameters: new FormGroup<KeaPoolParametersForm>({}),
                options: new FormGroup({
                    unlocked: new FormControl(false),
                    data: new FormArray([]),
                }),
                selectedDaemons: new FormControl([]),
            }),
            new FormGroup<AddressPoolForm>({
                range: new FormGroup<AddressRangeForm>({
                    start: new FormControl('2001:db8:1::1'),
                    end: new FormControl('2001:db8:1::1'),
                }),
                parameters: new FormGroup<KeaPoolParametersForm>({}),
                options: new FormGroup({
                    unlocked: new FormControl(false),
                    data: new FormArray([]),
                }),
                selectedDaemons: new FormControl([]),
            }),
        ])
        expect(StorkValidators.ipRangeOverlaps(fa)).toBeTruthy()

        expect(fa.at(0).invalid).toBeTrue()
        expect(fa.at(1).invalid).toBeTrue()

        // Invalidate one of the ranges.
        fa.get('0.range.start').setValue('invalid')
        expect(StorkValidators.ipRangeOverlaps(fa)).toBeFalsy()
        expect(fa.invalid).toBeFalse()
    })

    it('detects overlaps in the IPv6 address ranges', () => {
        let fa = new FormArray([
            new FormGroup<AddressPoolForm>({
                range: new FormGroup<AddressRangeForm>({
                    start: new FormControl('2001:db8:1::1'),
                    end: new FormControl('2001:db8:1::100'),
                }),
                parameters: new FormGroup<KeaPoolParametersForm>({}),
                options: new FormGroup({
                    unlocked: new FormControl(false),
                    data: new FormArray([]),
                }),
                selectedDaemons: new FormControl([]),
            }),
            new FormGroup<AddressPoolForm>({
                range: new FormGroup<AddressRangeForm>({
                    start: new FormControl('2001:db8:1::1'),
                    end: new FormControl('2001:db8:1::1'),
                }),
                parameters: new FormGroup<KeaPoolParametersForm>({}),
                options: new FormGroup({
                    unlocked: new FormControl(false),
                    data: new FormArray([]),
                }),
                selectedDaemons: new FormControl([]),
            }),
            new FormGroup<AddressPoolForm>({
                range: new FormGroup<AddressRangeForm>({
                    start: new FormControl('2001:db8:100::cafe'),
                    end: new FormControl('2001:db8:300::cafe'),
                }),
                parameters: new FormGroup<KeaPoolParametersForm>({}),
                options: new FormGroup({
                    unlocked: new FormControl(false),
                    data: new FormArray([]),
                }),
                selectedDaemons: new FormControl([]),
            }),
            new FormGroup<AddressPoolForm>({
                range: new FormGroup<AddressRangeForm>({
                    start: new FormControl('2001:db8:99::'),
                    end: new FormControl('2001:db8:100::ffff'),
                }),
                parameters: new FormGroup<KeaPoolParametersForm>({}),
                options: new FormGroup({
                    unlocked: new FormControl(false),
                    data: new FormArray([]),
                }),
                selectedDaemons: new FormControl([]),
            }),
            new FormGroup<AddressPoolForm>({
                range: new FormGroup<AddressRangeForm>({
                    start: new FormControl('2001:db8::'),
                    end: new FormControl('2001:db8::ffff'),
                }),
                parameters: new FormGroup<KeaPoolParametersForm>({}),
                options: new FormGroup({
                    unlocked: new FormControl(false),
                    data: new FormArray([]),
                }),
                selectedDaemons: new FormControl([]),
            }),
        ])
        expect(StorkValidators.ipRangeOverlaps(fa)).toBeTruthy()

        // Range 0 overlaps with range 1.
        expect(fa.at(0).invalid).toBeTrue()
        // Range 1 overlaps with range 0.
        expect(fa.at(1).invalid).toBeTrue()
        // Range 2 overlaps with range 3.
        expect(fa.at(2).invalid).toBeTrue()
        // Range 3 overlaps with range 2.
        expect(fa.at(3).invalid).toBeTrue()
        // Range 4 does not overlap.
        expect(fa.at(4).invalid).toBeFalse()

        // Correct the ranges.
        fa.get('0.range.start')?.setValue('2001:db8:1::2')
        fa.get('2.range.end')?.setValue('2001:db8:100::aaaa')
        expect(StorkValidators.ipRangeOverlaps(fa)).toBeFalsy()
        expect(fa.invalid).toBeFalse()
    })

    it('clears detected overlaps for a single IPv6 prefix', () => {
        let fa = new FormArray([
            new FormGroup<PrefixPoolForm>({
                prefixes: new FormGroup<PrefixForm>({
                    prefix: new FormControl('2001:db8:1::/64'),
                    delegatedLength: new FormControl(null),
                    excludedPrefix: new FormControl(null),
                }),
                parameters: new FormGroup<KeaPoolParametersForm>({}),
                options: new FormGroup({
                    unlocked: new FormControl(false),
                    data: new FormArray([]),
                }),
                selectedDaemons: new FormControl([]),
            }),
            new FormGroup<PrefixPoolForm>({
                prefixes: new FormGroup<PrefixForm>({
                    prefix: new FormControl('2001:db8:1::ff00/120'),
                    delegatedLength: new FormControl(null),
                    excludedPrefix: new FormControl(null),
                }),
                parameters: new FormGroup<KeaPoolParametersForm>({}),
                options: new FormGroup({
                    unlocked: new FormControl(false),
                    data: new FormArray([]),
                }),
                selectedDaemons: new FormControl([]),
            }),
        ])
        expect(StorkValidators.ipv6PrefixOverlaps(fa)).toBeTruthy()

        expect(fa.at(0).invalid).toBeTrue()
        expect(fa.at(1).invalid).toBeTrue()

        // Remove the second prefix.
        fa.removeAt(1)
        expect(StorkValidators.ipv6PrefixOverlaps(fa)).toBeFalsy()
        expect(fa.invalid).toBeFalse()
    })

    it('clears detected overlaps when an IPv6 prefix gets invalid', () => {
        let fa = new FormArray([
            new FormGroup<PrefixPoolForm>({
                prefixes: new FormGroup<PrefixForm>({
                    prefix: new FormControl('2001:db8:1::/64'),
                    delegatedLength: new FormControl(null),
                    excludedPrefix: new FormControl(null),
                }),
                parameters: new FormGroup<KeaPoolParametersForm>({}),
                options: new FormGroup({
                    unlocked: new FormControl(false),
                    data: new FormArray([]),
                }),
                selectedDaemons: new FormControl([]),
            }),
            new FormGroup<PrefixPoolForm>({
                prefixes: new FormGroup<PrefixForm>({
                    prefix: new FormControl('2001:db8:1::ff00/120'),
                    delegatedLength: new FormControl(null),
                    excludedPrefix: new FormControl(null),
                }),
                parameters: new FormGroup<KeaPoolParametersForm>({}),
                options: new FormGroup({
                    unlocked: new FormControl(false),
                    data: new FormArray([]),
                }),
                selectedDaemons: new FormControl([]),
            }),
        ])
        expect(StorkValidators.ipv6PrefixOverlaps(fa)).toBeTruthy()

        expect(fa.at(0).invalid).toBeTrue()
        expect(fa.at(1).invalid).toBeTrue()

        // Invalidate the prefix value.
        fa.get('0.prefixes.prefix').setValue('invalid')
        expect(StorkValidators.ipv6PrefixOverlaps(fa)).toBeFalsy()
        expect(fa.invalid).toBeFalse()
    })

    it('detects overlaps between the IPv6 prefixes', () => {
        let fa = new FormArray([
            new FormGroup<PrefixPoolForm>({
                prefixes: new FormGroup<PrefixForm>({
                    prefix: new FormControl('2001:db8:1::/64'),
                    delegatedLength: new FormControl(null),
                    excludedPrefix: new FormControl(null),
                }),
                parameters: new FormGroup<KeaPoolParametersForm>({}),
                options: new FormGroup({
                    unlocked: new FormControl(false),
                    data: new FormArray([]),
                }),
                selectedDaemons: new FormControl([]),
            }),
            new FormGroup<PrefixPoolForm>({
                prefixes: new FormGroup<PrefixForm>({
                    prefix: new FormControl('2001:db8:1::ff00/120'),
                    delegatedLength: new FormControl(null),
                    excludedPrefix: new FormControl(null),
                }),
                parameters: new FormGroup<KeaPoolParametersForm>({}),
                options: new FormGroup({
                    unlocked: new FormControl(false),
                    data: new FormArray([]),
                }),
                selectedDaemons: new FormControl([]),
            }),
            new FormGroup<PrefixPoolForm>({
                prefixes: new FormGroup<PrefixForm>({
                    prefix: new FormControl('3000::/48'),
                    delegatedLength: new FormControl(null),
                    excludedPrefix: new FormControl(null),
                }),
                parameters: new FormGroup<KeaPoolParametersForm>({}),
                options: new FormGroup({
                    unlocked: new FormControl(false),
                    data: new FormArray([]),
                }),
                selectedDaemons: new FormControl([]),
            }),
            new FormGroup<PrefixPoolForm>({
                prefixes: new FormGroup<PrefixForm>({
                    prefix: new FormControl('3000::/64'),
                    delegatedLength: new FormControl(null),
                    excludedPrefix: new FormControl(null),
                }),
                parameters: new FormGroup<KeaPoolParametersForm>({}),
                options: new FormGroup({
                    unlocked: new FormControl(false),
                    data: new FormArray([]),
                }),
                selectedDaemons: new FormControl([]),
            }),
            new FormGroup<PrefixPoolForm>({
                prefixes: new FormGroup<PrefixForm>({
                    prefix: new FormControl('2001:db8:2::/64'),
                    delegatedLength: new FormControl(null),
                    excludedPrefix: new FormControl(null),
                }),
                parameters: new FormGroup<KeaPoolParametersForm>({}),
                options: new FormGroup({
                    unlocked: new FormControl(false),
                    data: new FormArray([]),
                }),
                selectedDaemons: new FormControl([]),
            }),
        ])
        expect(StorkValidators.ipv6PrefixOverlaps(fa)).toBeTruthy()

        // Range 0 overlaps with range 1.
        expect(fa.at(0).invalid).toBeTrue()
        // Range 1 overlaps with range 0.
        expect(fa.at(1).invalid).toBeTrue()
        // Range 2 overlaps with range 3.
        expect(fa.at(2).invalid).toBeTrue()
        // Range 3 overlaps with range 2.
        expect(fa.at(3).invalid).toBeTrue()
        // Range 4 does not overlap.
        expect(fa.at(4).invalid).toBeFalse()

        // Correct the prefixes.
        fa.get('0.prefixes.prefix')?.setValue('2001:db8:1::ee00/120')
        fa.get('2.prefixes.prefix')?.setValue('3001::/48')
        expect(StorkValidators.ipv6PrefixOverlaps(fa)).toBeFalsy()
        expect(fa.invalid).toBeFalse()
    })

    it('validates excluded prefix being in the prefix', () => {
        // Valid excluded prefix.
        let fg = formBuilder.group({
            prefix: formBuilder.control('2001:db8:dead:beef::/56'),
            delegatedLength: formBuilder.control(57),
            excludedPrefix: formBuilder.control('2001:db8:dead:beef::01/60'),
        })
        expect(StorkValidators.ipv6ExcludedPrefix(fg)).toBeFalsy()
        // This validator ignores invalid values.
        fg = formBuilder.group({
            prefix: formBuilder.control('invalid'),
            delegatedLength: formBuilder.control(57),
            excludedPrefix: formBuilder.control('2001:db8:dead:beef::01/60'),
        })
        fg = formBuilder.group({
            prefix: formBuilder.control('2001:db8:dead:beef::/56'),
            delegatedLength: formBuilder.control(57),
            excludedPrefix: formBuilder.control('invalid'),
        })
        expect(StorkValidators.ipv6ExcludedPrefix(fg)).toBeFalsy()
        //  Non-matching prefixes.
        fg = formBuilder.group({
            prefix: formBuilder.control('2001:db8:dead:cafe::/56'),
            delegatedLength: formBuilder.control(57),
            excludedPrefix: formBuilder.control('2001:db8:dead:beef::01/64'),
        })
        // Excluded prefix must be smaller
        expect(StorkValidators.ipv6ExcludedPrefix(fg)).toBeTruthy()
        fg = formBuilder.group({
            prefix: formBuilder.control('2001:db8:dead:cafe::/56'),
            delegatedLength: formBuilder.control(57),
            excludedPrefix: formBuilder.control('2001:db8:dead:beef::01/56'),
        })
        const validationErrors = StorkValidators.ipv6ExcludedPrefix(fg)
        expect(validationErrors).toBeTruthy()
        expect(validationErrors.ipv6ExcludedPrefix).toBe(
            '2001:db8:dead:beef::01/56 excluded prefix length must be greater than the 2001:db8:dead:cafe::/56 prefix length.'
        )
    })

    it('validates delegated prefix length for prefix length', () => {
        // Valid delegated prefix length.
        let fg = formBuilder.group({
            prefix: formBuilder.control('2001:db8:dead:beef::/56'),
            delegatedLength: formBuilder.control(57),
        })
        // Equal prefix length and delegated length is fine too.
        expect(StorkValidators.ipv6PrefixDelegatedLength(fg)).toBeFalsy()
        fg = formBuilder.group({
            prefix: formBuilder.control('2001:db8:dead:beef::/56'),
            delegatedLength: formBuilder.control(56),
        })
        expect(StorkValidators.ipv6PrefixDelegatedLength(fg)).toBeFalsy()
        // Invalid prefix is not validated here.
        fg = formBuilder.group({
            prefix: formBuilder.control('invalid'),
            delegatedLength: formBuilder.control(57),
        })
        expect(StorkValidators.ipv6PrefixDelegatedLength(fg)).toBeFalsy()
        // Invalid delegated length is not validated here.
        fg = formBuilder.group({
            prefix: formBuilder.control('3000::/16'),
            delegatedLength: formBuilder.control(null),
        })
        expect(StorkValidators.ipv6PrefixDelegatedLength(fg)).toBeFalsy()
        // Delegated prefix length must be greater.
        fg = formBuilder.group({
            prefix: formBuilder.control('2001:db8:dead:beef::/64'),
            delegatedLength: formBuilder.control(48),
        })
        const validationErrors = StorkValidators.ipv6PrefixDelegatedLength(fg)
        expect(validationErrors).toBeTruthy()
        expect(validationErrors.ipv6PrefixDelegatedLength).toBe(
            'Delegated prefix length must be greater or equal the 2001:db8:dead:beef::/64 prefix length.'
        )
    })

    it('validates delegated prefix length for excluded prefix length', () => {
        // Valid delegated prefix length.
        let fg = formBuilder.group({
            prefix: formBuilder.control('2001:db8:dead:beef::/56'),
            delegatedLength: formBuilder.control(64),
            excludedPrefix: '2001:db8:dead:beef::0:0:0/80',
        })
        expect(StorkValidators.ipv6ExcludedPrefixDelegatedLength(fg)).toBeFalsy()
        // Invalid prefix is not validated here.
        fg = formBuilder.group({
            prefix: formBuilder.control('2001:db8:dead:beef::/56'),
            delegatedLength: formBuilder.control(64),
            excludedPrefix: 'invalid',
        })
        expect(StorkValidators.ipv6ExcludedPrefixDelegatedLength(fg)).toBeFalsy()
        // Invalid delegated length is not validated here.
        fg = formBuilder.group({
            prefix: formBuilder.control('3000::/16'),
            delegatedLength: formBuilder.control(null),
            excludedPrefix: '2001:db8:dead:beef::0:0:0/80',
        })
        expect(StorkValidators.ipv6ExcludedPrefixDelegatedLength(fg)).toBeFalsy()
        // Delegated prefix length must be lower.
        fg = formBuilder.group({
            prefix: formBuilder.control('2001:db8:dead:beef::/56'),
            delegatedLength: formBuilder.control(80),
            excludedPrefix: '2001:db8:dead:beef::0:0:0/80',
        })
        let validationErrors = StorkValidators.ipv6ExcludedPrefixDelegatedLength(fg)
        expect(validationErrors).toBeTruthy()
        expect(validationErrors.ipv6ExcludedPrefixDelegatedLength).toBe(
            'Delegated prefix length must be lower than the 2001:db8:dead:beef::0:0:0/80 excluded prefix length.'
        )
        fg = formBuilder.group({
            prefix: formBuilder.control('2001:db8:dead:beef::/64'),
            delegatedLength: formBuilder.control(96),
            excludedPrefix: '2001:db8:dead:beef::0:0:0/80',
        })
        validationErrors = StorkValidators.ipv6ExcludedPrefixDelegatedLength(fg)
        expect(validationErrors).toBeTruthy()
        expect(validationErrors.ipv6ExcludedPrefixDelegatedLength).toBe(
            'Delegated prefix length must be lower than the 2001:db8:dead:beef::0:0:0/80 excluded prefix length.'
        )
    })

    it('detects overlaps in the address pool identifiers', () => {
        let fa = new FormArray([
            new FormGroup<AddressPoolForm>({
                range: new FormGroup<AddressRangeForm>({
                    start: new FormControl('192.0.2.50'),
                    end: new FormControl('192.0.2.60'),
                }),
                parameters: new FormGroup<KeaPoolParametersForm>({
                    poolID: new SharedParameterFormGroup(null, [new FormControl<number>(50)]),
                }),
                options: new FormGroup({
                    unlocked: new FormControl(false),
                    data: new FormArray([]),
                }),
                selectedDaemons: new FormControl([]),
            }),
            new FormGroup<AddressPoolForm>({
                range: new FormGroup<AddressRangeForm>({
                    start: new FormControl('192.0.2.61'),
                    end: new FormControl('192.0.2.70'),
                }),
                parameters: new FormGroup<KeaPoolParametersForm>({
                    poolID: new SharedParameterFormGroup(null, [
                        new FormControl<number>(50),
                        new FormControl<number>(60),
                    ]),
                }),
                options: new FormGroup({
                    unlocked: new FormControl(false),
                    data: new FormArray([]),
                }),
                selectedDaemons: new FormControl([]),
            }),
            new FormGroup<AddressPoolForm>({
                range: new FormGroup<AddressRangeForm>({
                    start: new FormControl('192.0.2.71'),
                    end: new FormControl('192.0.2.80'),
                }),
                parameters: new FormGroup<KeaPoolParametersForm>({
                    poolID: new SharedParameterFormGroup(null, [
                        new FormControl<number>(70),
                        new FormControl<number>(60),
                    ]),
                }),
                options: new FormGroup({
                    unlocked: new FormControl(false),
                    data: new FormArray([]),
                }),
                selectedDaemons: new FormControl([]),
            }),
            new FormGroup<AddressPoolForm>({
                range: new FormGroup<AddressRangeForm>({
                    start: new FormControl('192.0.2.81'),
                    end: new FormControl('192.0.2.90'),
                }),
                parameters: new FormGroup<KeaPoolParametersForm>({
                    poolID: new SharedParameterFormGroup(null, [
                        new FormControl<number>(80),
                        new FormControl<number>(80),
                    ]),
                }),
                options: new FormGroup({
                    unlocked: new FormControl(false),
                    data: new FormArray([]),
                }),
                selectedDaemons: new FormControl([]),
            }),
            new FormGroup<AddressPoolForm>({
                range: new FormGroup<AddressRangeForm>({
                    start: new FormControl('192.0.2.91'),
                    end: new FormControl('192.0.2.100'),
                }),
                parameters: new FormGroup<KeaPoolParametersForm>({}),
                options: new FormGroup({
                    unlocked: new FormControl(false),
                    data: new FormArray([]),
                }),
                selectedDaemons: new FormControl([]),
            }),
        ])
        expect(StorkValidators.poolIDOverlaps(fa)).toBeTruthy()

        expect(fa.at(0).invalid).toBeTrue()
        expect(fa.at(1).invalid).toBeTrue()
        expect(fa.at(2).invalid).toBeTrue()
        expect(fa.at(3).invalid).toBeFalse()
        expect(fa.at(4).invalid).toBeFalse()

        // Correct the pool-ids.
        fa.get('1.parameters.poolID.values.0')?.setValue(60)
        fa.get('2.parameters.poolID.values.1')?.setValue(70)
        expect(StorkValidators.poolIDOverlaps(fa)).toBeFalsy()
        expect(fa.invalid).toBeFalse()
    })

    it('detects overlaps in the prefix pool identifiers', () => {
        let fa = new FormArray([
            new FormGroup<PrefixPoolForm>({
                prefixes: new FormGroup<PrefixForm>({
                    prefix: new FormControl('2001:db8:1::/64'),
                    delegatedLength: new FormControl(null),
                    excludedPrefix: new FormControl(null),
                }),
                parameters: new FormGroup<KeaPoolParametersForm>({
                    poolID: new SharedParameterFormGroup(null, [
                        new FormControl<number>(80),
                        new FormControl<number>(90),
                    ]),
                }),
                options: new FormGroup({
                    unlocked: new FormControl(false),
                    data: new FormArray([]),
                }),
                selectedDaemons: new FormControl([]),
            }),
            new FormGroup<PrefixPoolForm>({
                prefixes: new FormGroup<PrefixForm>({
                    prefix: new FormControl('2001:db8:1::ff00/120'),
                    delegatedLength: new FormControl(null),
                    excludedPrefix: new FormControl(null),
                }),
                parameters: new FormGroup<KeaPoolParametersForm>({
                    poolID: new SharedParameterFormGroup(null, [
                        new FormControl<number>(90),
                        new FormControl<number>(90),
                    ]),
                }),
                options: new FormGroup({
                    unlocked: new FormControl(false),
                    data: new FormArray([]),
                }),
                selectedDaemons: new FormControl([]),
            }),
        ])
        expect(StorkValidators.poolIDOverlaps(fa)).toBeTruthy()

        expect(fa.at(0).invalid).toBeTrue()
        expect(fa.at(1).invalid).toBeTrue()

        // Correct the overlapping identifier.
        fa.get('1.parameters.poolID.values.0')?.setValue(100)
        expect(StorkValidators.poolIDOverlaps(fa)).toBeFalsy()
        expect(fa.invalid).toBeFalse()
    })

    it('validates full fqdn', () => {
        expect(StorkValidators.fullFqdn(formBuilder.control('a..bc.'))).toBeTruthy()
        expect(StorkValidators.fullFqdn(formBuilder.control('a.b.'))).toBeTruthy()
        expect(StorkValidators.fullFqdn(formBuilder.control('test-.xyz.'))).toBeTruthy()
        expect(StorkValidators.fullFqdn(formBuilder.control('-test.xyz.'))).toBeTruthy()
        expect(StorkValidators.fullFqdn(formBuilder.control('.test.xyz.'))).toBeTruthy()
        expect(StorkValidators.fullFqdn(formBuilder.control('test'))).toBeTruthy()
        expect(StorkValidators.fullFqdn(formBuilder.control('a.bc'))).toBeTruthy()

        expect(StorkValidators.fullFqdn(formBuilder.control('test--abc.ec-a-b.xyz.'))).toBeFalsy()
        expect(StorkValidators.fullFqdn(formBuilder.control('test.abc.xyz.'))).toBeFalsy()
        expect(StorkValidators.fullFqdn(formBuilder.control('a123.xyz.'))).toBeFalsy()
    })

    it('validates partial fqdn', () => {
        expect(StorkValidators.partialFqdn(formBuilder.control('a..bc'))).toBeTruthy()
        expect(StorkValidators.partialFqdn(formBuilder.control('test-.xyz'))).toBeTruthy()
        expect(StorkValidators.partialFqdn(formBuilder.control('-test.xyz'))).toBeTruthy()
        expect(StorkValidators.partialFqdn(formBuilder.control('test.xyz.'))).toBeTruthy()
        expect(StorkValidators.partialFqdn(formBuilder.control('.test.xyz'))).toBeTruthy()

        expect(StorkValidators.partialFqdn(formBuilder.control('a.bc'))).toBeFalsy()
        expect(StorkValidators.partialFqdn(formBuilder.control('test--abc.x-yz'))).toBeFalsy()
        expect(StorkValidators.partialFqdn(formBuilder.control('test.abc.xyz'))).toBeFalsy()
        expect(StorkValidators.partialFqdn(formBuilder.control('test'))).toBeFalsy()
    })

    it('validates fqdn', () => {
        // Invalid FQDN should cause an error.
        expect(StorkValidators.fqdn(formBuilder.control('test.'))).toBeTruthy()

        // A full or partial FQDN should be fine.
        expect(StorkValidators.fqdn(formBuilder.control('test--abc.ec-a-b.xyz.'))).toBeFalsy()
        expect(StorkValidators.fqdn(formBuilder.control('test'))).toBeFalsy()
    })

    it('validates that a prefix is not in the list', () => {
        const subnets = ['192.0.2.0/24', '192.0.3.0/24', '192.0.4.0/24', '2001:db8:1::/64', '2001:db8:2::/64']
        expect(StorkValidators.prefixInList(subnets)(formBuilder.control('192.0.2.0/24'))).toBeTruthy()
        expect(StorkValidators.prefixInList(subnets)(formBuilder.control('192.0.02.0/24'))).toBeTruthy()
        expect(StorkValidators.prefixInList(subnets)(formBuilder.control('2001:db8:1:0:0::/64'))).toBeTruthy()
        expect(StorkValidators.prefixInList(subnets)(formBuilder.control('2001:db8:2::/64'))).toBeTruthy()
        expect(StorkValidators.prefixInList(subnets)(formBuilder.control(''))).toBeFalsy()
        expect(StorkValidators.prefixInList(subnets)(formBuilder.control('192'))).toBeFalsy()
        expect(StorkValidators.prefixInList(subnets)(formBuilder.control('192.0.5.0/24'))).toBeFalsy()
        expect(StorkValidators.prefixInList(subnets)(formBuilder.control('2001:db8:3::/72'))).toBeFalsy()
    })

    it('validates that a value is not in the list', () => {
        const values = ['foo', 'bar', 'baz']
        expect(StorkValidators.valueInList(values)(formBuilder.control('foo'))).toBeTruthy()
        expect(StorkValidators.valueInList(values)(formBuilder.control('baz'))).toBeTruthy()
        expect(StorkValidators.valueInList(values)(formBuilder.control(''))).toBeFalsy()
        expect(StorkValidators.valueInList(values)(formBuilder.control(null))).toBeFalsy()
        expect(StorkValidators.valueInList(values)(formBuilder.control('wall'))).toBeFalsy()
    })
})
