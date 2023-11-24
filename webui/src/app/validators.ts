import { AbstractControl, FormArray, FormGroup, ValidationErrors, ValidatorFn, Validators } from '@angular/forms'
import { IPv4, IPv4CidrRange, IPv6, IPv6CidrRange, Validator } from 'ip-num'
import { AddressPoolForm, AddressRangeForm, PrefixForm, PrefixPoolForm } from './forms/subnet-set-form.service'
import { AddressRange } from './address-range'

/**
 * A class with various static form validation functions.
 *
 * It comprises form validators potentially useful in many Stork components.
 * Where possible, it uses generic Angular built-in validators.
 */
export class StorkValidators {
    /**
     * A validator checking if the identifier is a string of hexadecimal
     * digits with a colon or dash used as a separator.
     *
     * @returns validator function.
     */
    static hexIdentifier(): ValidatorFn {
        return Validators.pattern('^([0-9A-Fa-f]{2}[:-]{0,1})+([0-9A-Fa-f]{2})')
    }

    /**
     * A validator checking if an identifier consisting of a string of hexadecimal
     * digits and, optionally, colons, spaces and dashes has valid length.
     *
     * It checks that the number of hexadecimal digits does not exceed the specified
     * value and ignores colons, spaces and dashes.
     *
     * @param maxLength maximum number of digits.
     * @returns validator function or null.
     */
    static hexIdentifierLength(maxLength: number): ValidatorFn {
        return (control: AbstractControl): ValidationErrors | null => {
            // If it is not a string we leave the validation to other validators.
            if (control.value === null || typeof control.value !== 'string' || control.value.length === 0) {
                return null
            }
            let s = control.value
            s = s.replace(/\s|:|-/gi, '')
            if (s.length > maxLength) {
                return { maxlength: `The number of hexadecimal digits exceeds the maximum value of ${maxLength}.` }
            }
            return null
        }
    }

    /**
     * A validator checking if an input is a valid IPv4 address.
     *
     * @returns validator function.
     */
    static ipv4(): ValidatorFn {
        return (control: AbstractControl): ValidationErrors | null => {
            if (control.value === null || typeof control.value !== 'string' || control.value.length === 0) {
                return null
            }
            let ipv4 = control.value
            if (!Validator.isValidIPv4String(ipv4)[0]) {
                return { ipv4: `${ipv4} is not a valid IPv4 address.` }
            }
            return null
        }
    }

    /**
     * A validator checking if an input is a valid IPv6 address or a prefix
     * without the length.
     *
     * @returns validator function.
     */
    static ipv6(): ValidatorFn {
        return (control: AbstractControl): ValidationErrors | null => {
            if (control.value === null || typeof control.value !== 'string' || control.value.length === 0) {
                return null
            }
            let ipv6 = control.value
            if (!Validator.isValidIPv6String(ipv6)[0]) {
                return { ipv6: `${ipv6} is not a valid IPv6 address.` }
            }
            return null
        }
    }

    /**
     * A validator checking if an input is a valid IPv6 prefix (including length).
     *
     * @returns validation errors if the prefix is invalid or null otherwise.
     */
    static ipv6Prefix(control: AbstractControl): ValidationErrors | null {
        if (control.value === null || typeof control.value !== 'string' || control.value.length === 0) {
            return null
        }
        let ipv6 = control.value
        if (!ipv6.includes('/') || !Validator.isValidIPv6CidrNotation(ipv6)[0]) {
            return { ipv6: `${ipv6} is not a valid IPv6 prefix.` }
        }
        return null
    }

    /**
     * A validator checking if the excluded prefix is valid.
     *
     * Follows sanity checks in the RFC6603, section 4.2. It skips
     * the validation if the prefix or excluded prefix are not in
     * the CIDR format or are not specified. Validity of the CIDR
     * format should be performed by other validators.
     *
     * @param prefix a prefix holding the excluded prefix.
     * @returns validation errors if the excluded prefix is invalid
     * or null otherwise.
     */
    static ipv6ExcludedPrefix(control: AbstractControl): ValidationErrors | null {
        const fg = control as FormGroup<PrefixForm>
        if (!fg) {
            return { ipv6ExcludedPrefix: 'Invalid form group type.' }
        }
        try {
            const prefix = fg.get('prefix').value as string
            const excludedPrefix = fg.get('excludedPrefix').value as string
            const prefixCidr = IPv6CidrRange.fromCidr(prefix)
            const excludedPrefixCidr = IPv6CidrRange.fromCidr(excludedPrefix)
            // The excluded prefix must be smaller than the prefix.
            if (prefixCidr.getSize() <= excludedPrefixCidr.getSize()) {
                return {
                    ipv6ExcludedPrefix: `${control.value} excluded prefix is length must be greater than the ${prefix} prefix length.`,
                }
            }
            // See RFC6603, section 4.2.
            if (
                prefixCidr.getFirst().getValue() >> (BigInt(128) - prefixCidr.getPrefix().getValue()) !=
                excludedPrefixCidr.getFirst().getValue() >> (BigInt(128) - prefixCidr.getPrefix().getValue())
            ) {
                return {
                    ipv6ExcludedPrefix: `${excludedPrefix} excluded prefix is not within the ${prefix} prefix.`,
                }
            }
        } catch (_) {
            return null
        }
        return null
    }

    /**
     * A validator checking if a specified IP address is in the subnet.
     *
     * @param subnet a subnet string in the CIDR format.
     * @returns Validator function.
     */
    static ipInSubnet(subnet: string): ValidatorFn {
        return (control: AbstractControl): ValidationErrors | null => {
            if (control.value === null || typeof control.value !== 'string' || control.value.length === 0) {
                return { ipInSubnet: `Please specify an IP address belonging to ${subnet}.` }
            }
            const split = subnet.split('/')
            if (split.length != 2) {
                return { ipInSubnet: `${subnet} is not a valid subnet prefix.` }
            }
            const prefix = split[0]
            if (Validator.isValidIPv4String(prefix)[0]) {
                try {
                    const range = IPv4CidrRange.fromCidr(subnet)
                    const ipv4 = IPv4.fromDecimalDottedString(control.value)
                    if (range.getFirst().isGreaterThan(ipv4) || range.getLast().isLessThan(ipv4)) {
                        return { ipInSubnet: `${control.value} does not belong to subnet ${subnet}.` }
                    }
                } catch (_) {
                    return { ipInSubnet: `${control.value} is not a valid IPv4 address.` }
                }
                return null
            } else if (Validator.isValidIPv6String(prefix)[0]) {
                try {
                    const range = IPv6CidrRange.fromCidr(subnet)
                    const ipv6 = IPv6.fromString(control.value)
                    if (range.getFirst().isGreaterThan(ipv6) || range.getLast().isLessThan(ipv6)) {
                        return { ipInSubnet: `${control.value} does not belong to subnet ${subnet}.` }
                    }
                } catch (_) {
                    return { ipInSubnet: `${control.value} is not a valid IPv6 address.` }
                }
            } else {
                return { ipInSubnet: `${subnet} is not a valid subnet prefix.` }
            }
            return null
        }
    }

    /**
     * Selectively removes a control error.
     *
     * @param control form control for which the error should be removed.
     * @param errorKey error key/name.
     */
    private static clearControlError(control: AbstractControl, errorKey: string): void {
        if (control.hasError(errorKey)) {
            delete control.errors[errorKey]
            control.updateValueAndValidity()
        }
    }

    /**
     * A validator checking if an address range boundaries are correct.
     *
     * The start address must be lower or equal the end address.
     *
     * @returns validation errors or null if the range boundaries are valid.
     */
    static ipRangeBounds(control: AbstractControl): ValidationErrors | null {
        const fg = control as FormGroup<AddressRangeForm>
        if (!fg) {
            return { addressBounds: `Invalid form group type.` }
        }
        try {
            const start = fg.get('start')?.value as string
            const end = fg.get('end')?.value as string
            if (start && end) {
                AddressRange.fromStringBounds(start, end)
            }
        } catch (_) {
            if (fg.get('start').valid) {
                fg.get('start').setErrors({ addressBounds: true })
                fg.get('end').markAsDirty()
            }
            if (fg.get('end').valid) {
                fg.get('end').setErrors({ addressBounds: true })
                fg.get('start').markAsDirty()
            }
            return {
                addressBounds:
                    'Invalid address pool boundaries. Make sure that the first address is equal or lower than the last address.',
            }
        }
        StorkValidators.clearControlError(fg.get('start'), 'addressBounds')
        StorkValidators.clearControlError(fg.get('end'), 'addressBounds')
        return null
    }

    /**
     * A validator checking if the are overlaps between address ranges.
     *
     * It sets errors for each overlapping range in the array.
     *
     * @param control a form array holding address ranges.
     * @returns validation errors or null if the ranges do not overlap.
     */
    static ipRangeOverlaps(control: AbstractControl): ValidationErrors | null {
        const fa = control as FormArray<FormGroup<AddressPoolForm>>
        if (!fa) {
            return { ipRangeOverlaps: 'Invalid form array type.' }
        }
        // Go over the current pools and collect the address ranges sorted.
        // Each element of the sorted list contains an address range and a
        // pointer to the form group holding the range. The ranges are sorted
        // by the lower range boundaries. Invalid ranges are removed by the
        // filter function.
        interface RangeData {
            addressRange: AddressRange
            control: AbstractControl
            failedOnThisPass: boolean
        }
        const ranges: RangeData[] = fa.controls
            .map((ctl) => {
                try {
                    return {
                        addressRange: AddressRange.fromStringBounds(
                            ctl.get('range.start')?.value,
                            ctl.get('range.end')?.value
                        ),
                        control: ctl.get('range'),
                        failedOnThisPass: false,
                    }
                } catch (_) {
                    StorkValidators.clearControlError(ctl.get('range'), 'ipRangeOverlaps')
                    return null
                }
            })
            .filter((range) => range)
            .sort((range1, range2) => {
                if (range1.addressRange.first.isLessThan(range2.addressRange.first)) {
                    return -1
                } else if (range1.addressRange.first.isGreaterThan(range2.addressRange.first)) {
                    return 1
                }
                return 0
            })
        let result: ValidationErrors | null = null
        // If there is only one range there is no overlap.
        if (ranges.length === 1) {
            StorkValidators.clearControlError(ranges[0].control, 'ipRangeOverlaps')
        } else {
            // Now that we have the ranges sorted by the lower boundaries we have to make sure
            // that the upper boundaries of each range are lower than the start of the next range.
            for (let i = 0; i < ranges.length - 1; i++) {
                if (ranges[i].addressRange.last.isGreaterThanOrEquals(ranges[i + 1].addressRange.first)) {
                    result = {
                        ipRangeOverlaps: `Address range ${ranges[i].addressRange.getFirst()}-${ranges[
                            i
                        ].addressRange.getLast()} overlaps with ${ranges[i + 1].addressRange.getFirst()}-${ranges[
                            i + 1
                        ].addressRange.getLast()}.`,
                    }
                    // The two ranges overlap. Set the error in the respective form groups.
                    ranges.slice(i, i + 2).forEach((range) => {
                        range.control.setErrors(result)
                        range.failedOnThisPass = true
                    })
                } else {
                    // The ranges do not overlap. Clear the errors.
                    ranges.slice(i, i + 2).forEach((range) => {
                        if (!range.failedOnThisPass) {
                            StorkValidators.clearControlError(range.control, 'ipRangeOverlaps')
                        }
                    })
                }
            }
        }
        return result
    }

    /**
     * A validator checking if there are overlaps between prefixes.
     *
     * It sets errors for each overlapping prefix in the array.
     *
     * @param control a form array holding prefixes.
     * @returns validation errors or null if the prefixes do not overlap.
     */
    static ipv6PrefixOverlaps(control: AbstractControl): ValidationErrors | null {
        const fa = control as FormArray<FormGroup<PrefixPoolForm>>
        if (!fa) {
            return { ipv6PrefixOverlaps: 'Invalid form array type.' }
        }
        // Go over the current pools and collect the address ranges sorted.
        // Each element of the sorted list contains an address range and a
        // pointer to the form group holding the range. Invalid ranges are
        // removed by the
        // filter function.
        interface PrefixData {
            original: string
            prefix: IPv6CidrRange
            control: AbstractControl
            failedOnThisPass: boolean
        }
        const prefixes: PrefixData[] = fa.controls
            .map((ctl) => {
                try {
                    return {
                        original: ctl.get('prefixes.prefix')?.value as string,
                        prefix: IPv6CidrRange.fromCidr(ctl.get('prefixes.prefix')?.value as string),
                        control: ctl.get('prefixes'),
                        failedOnThisPass: false,
                    }
                } catch (_) {
                    StorkValidators.clearControlError(ctl.get('prefixes'), 'ipv6PrefixOverlaps')
                    return null
                }
            })
            .filter((prefix) => prefix)
            .sort((prefix1, prefix2) => {
                if (prefix1.prefix.getFirst().isLessThan(prefix2.prefix.getFirst())) {
                    return -1
                } else if (prefix1.prefix.getFirst().isGreaterThan(prefix2.prefix.getFirst())) {
                    return 1
                }
                return 0
            })

        let result: ValidationErrors | null = null
        // If there is only one prefix, there is no overlap.
        if (prefixes.length === 1) {
            StorkValidators.clearControlError(prefixes[0].control, 'ipv6PrefixOverlaps')
        } else {
            for (let i = 0; i < prefixes.length - 1; i++) {
                if (prefixes[i].prefix.getLast().isGreaterThanOrEquals(prefixes[i + 1].prefix.getFirst())) {
                    result = {
                        ipv6PrefixOverlaps: `Prefix ${prefixes[i].original} overlaps with with ${
                            prefixes[i + 1].original
                        }.`,
                    }
                    // The two prefixes overlap. Set the error in the respective form groups.
                    prefixes.slice(i, i + 2).forEach((prefix) => {
                        prefix.control.setErrors(result)
                        prefix.failedOnThisPass = true
                    })
                } else {
                    // The prefixes do not overlap. Clear the errors.
                    prefixes.slice(i, i + 2).forEach((prefix) => {
                        if (!prefix.failedOnThisPass) {
                            StorkValidators.clearControlError(prefix.control, 'ipv6PrefixOverlaps')
                        }
                    })
                }
            }
        }
        return result
    }

    /**
     * A validator checking if a delegated prefix is smaller than the prefix.
     *
     * @param control form group instance holding prefix data.
     * @returns validation errors when the delegated length is lower or equal
     * the prefix length, null otherwise.
     */
    static ipv6PrefixDelegatedLength(control: AbstractControl): ValidationErrors | null {
        const fg = control as FormGroup<PrefixPoolForm>
        if (!fg) {
            return { addressBounds: `Invalid form group type.` }
        }
        let result: ValidationErrors | null = null
        try {
            const prefix = fg.get('prefix').value as string
            const delegatedLength = fg.get('delegatedLength').value as number
            const prefixCidr = IPv6CidrRange.fromCidr(prefix)
            if (delegatedLength && prefixCidr.getPrefix().getValue() >= delegatedLength) {
                result = {
                    ipv6PrefixDelegatedLength: `Delegated prefix length must be greater than the ${prefix} prefix length.`,
                }
            }
        } catch (_) {
            return null
        }
        return result
    }

    /**
     * A validator checking if a delegated prefix is larger than the excluded prefix.
     *
     * @param control form group instance holding prefix data.
     * @returns validation errors when the delegated length is greater or equal
     * the excluded prefix length, null otherwise.
     */
    static ipv6ExcludedPrefixDelegatedLength(control: AbstractControl): ValidationErrors | null {
        const fg = control as FormGroup<PrefixPoolForm>
        if (!fg) {
            return { addressBounds: `Invalid form group type.` }
        }
        let result: ValidationErrors | null = null
        try {
            const prefix = fg.get('excludedPrefix').value as string
            const delegatedLength = fg.get('delegatedLength').value as number
            const prefixCidr = IPv6CidrRange.fromCidr(prefix)
            if (delegatedLength && prefixCidr.getPrefix().getValue() <= delegatedLength) {
                result = {
                    ipv6ExcludedPrefixDelegatedLength: `Delegated prefix length must be lower than the ${prefix} excluded prefix length.`,
                }
            }
        } catch (_) {
            // This validator does not check invalid prefixes.
            return null
        }
        return result
    }

    /**
     * A validator checking if an input is a valid partial or full FQDN.
     *
     * Inspired by: https://stackoverflow.com/questions/11809631/fully-qualified-domain-name-validation
     *
     * @param control form control instance holding the validated value.
     * @returns validator function.
     */
    static fqdn(control: AbstractControl): ValidationErrors | null {
        if (control.value === null || typeof control.value !== 'string' || control.value.length === 0) {
            return null
        }
        if (StorkValidators.partialFqdn(control) && StorkValidators.fullFqdn(control)) {
            return { fqdn: `${control.value} is not a valid FQDN` }
        }
        return null
    }

    /**
     * A validator checking if an input is a valid partial FQDN.
     *
     * Inspired by: https://stackoverflow.com/questions/11809631/fully-qualified-domain-name-validation
     *
     * @param control form control instance holding the validated value.
     * @returns validation errors or null if the FQDN is valid.
     */
    static partialFqdn(control: AbstractControl): ValidationErrors | null {
        if (
            Validators.pattern(
                '(?=^.{4,253}$)(^((?!-)[a-zA-Z0-9-]{0,62}[a-zA-Z0-9].)*((?!-)[a-zA-Z0-9-]{0,62}[a-zA-Z0-9])$)'
            )(control)
        ) {
            return { 'partial-fqdn': `${control.value} is not a valid partial FQDN` }
        }
        return null
    }

    /**
     * A validator checking if an input is a valid full FQDN.
     *
     * Inspired by: https://stackoverflow.com/questions/11809631/fully-qualified-domain-name-validation
     *
     * @param control form control instance holding the validated value.
     * @returns validation errors or null if the FQDN is valid.
     */
    static fullFqdn(control: AbstractControl): ValidationErrors | null {
        if (
            Validators.pattern('(?=^.{4,253}$)(^((?!-)[a-zA-Z0-9-]{0,62}[a-zA-Z0-9]\\.)+[a-zA-Z]{2,63}\\.{1}$)')(
                control
            )
        ) {
            return { 'full-fqdn': `${control.value} is not a valid full FQDN` }
        }
        return null
    }
}
