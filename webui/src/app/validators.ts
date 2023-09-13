import { AbstractControl, ValidationErrors, ValidatorFn, Validators } from '@angular/forms'
import { Validator } from 'ip-num'

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
     * A validator checking if an input is a valid partial or full FQDN.
     *
     * Inspired by: https://stackoverflow.com/questions/11809631/fully-qualified-domain-name-validation
     *
     * @param control form control instance holding the validated value.
     * @returns validator function.
     */
    static fqdn(control: AbstractControl): ValidationErrors | null {
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
