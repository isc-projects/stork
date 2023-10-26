import { AbstractControl, FormGroup, ValidationErrors, ValidatorFn, Validators } from '@angular/forms'
import { IPv4, IPv4CidrRange, IPv6, IPv6CidrRange, Validator } from 'ip-num'
import { AddressRangeForm } from './forms/subnet-set-form.service'
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
     * @returns validator function.
     */
    static ipv6Prefix(): ValidatorFn {
        return (control: AbstractControl): ValidationErrors | null => {
            if (control.value === null || typeof control.value !== 'string' || control.value.length === 0) {
                return null
            }
            let ipv6 = control.value
            if (!ipv6.includes('/') || !Validator.isValidIPv6CidrNotation(ipv6)[0]) {
                return { ipv6: `${ipv6} is not a valid IPv6 prefix.` }
            }
            return null
        }
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
            if (Validator.isValidIPv4String(split[0])[0]) {
                try {
                    const range = IPv4CidrRange.fromCidr(subnet)
                    const ipv4 = IPv4.fromDecimalDottedString(control.value)
                    if (range.getFirst().isGreaterThan(ipv4) || range.getLast().isLessThan(ipv4)) {
                        return { ipInSubnet: `${control.value} does not belong to subnet ${subnet}.` }
                    }
                } catch (err) {
                    return { ipInSubnet: `${control.value} is not a valid IPv4 address.` }
                }
                return null
            } else if (Validator.isValidIPv6String(split[0])[0]) {
                try {
                    const range = IPv6CidrRange.fromCidr(subnet)
                    const ipv6 = IPv6.fromString(control.value)
                    if (range.getFirst().isGreaterThan(ipv6) || range.getLast().isLessThan(ipv6)) {
                        return { ipInSubnet: `${control.value} does not belong to subnet ${subnet}.` }
                    }
                } catch (err) {
                    return { ipInSubnet: `${control.value} is not a valid IPv6 address.` }
                }
            } else {
                return { ipInSubnet: `${subnet} is not a valid subnet prefix.` }
            }
            return null
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
        } catch (err) {
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
        if (fg.get('start').hasError('addressBounds')) {
            delete fg.get('start').errors['addressBounds']
            fg.get('start').updateValueAndValidity()
        }
        if (fg.get('end').hasError('addressBounds')) {
            delete fg.get('end').errors['addressBounds']
            fg.get('end').updateValueAndValidity()
        }
        return null
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
