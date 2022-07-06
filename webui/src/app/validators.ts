import { AbstractControl, ValidationErrors, ValidatorFn, Validators } from '@angular/forms'

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
            if (control.value === null || typeof control.value !== 'string') {
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
        // See: https://github.com/ip-num/ip-num/blob/master/src/Validator.ts
        return Validators.pattern(
            '^(0?[0-9]?[0-9]|1[0-9][0-9]|2[0-4][0-9]|25[0-5])\\.(0?[0-9]?[0-9]|1[0-9][0-9]|2[0-4][0-9]|25[0-5])\\.(0?[0-9]?[0-9]|1[0-9][0-9]|2[0-4][0-9]|25[0-5])\\.(0?[0-9]?[0-9]|1[0-9][0-9]|2[0-4][0-9]|25[0-5])$'
        )
    }

    /**
     * A validator checking if an input is a valid IPv6 address or a prefix
     * without the length.
     *
     * @returns validator function.
     */
    static ipv6(): ValidatorFn {
        // See: https://github.com/ip-num/ip-num/blob/master/src/Validator.ts
        return Validators.pattern(
            '^s*((([0-9A-Fa-f]{1,4}:){7}([0-9A-Fa-f]{1,4}|:))|(([0-9A-Fa-f]{1,4}:){6}(:[0-9A-Fa-f]{1,4}|((25[0-5]|2[0-4]d|1dd|[1-9]?d)(.(25[0-5]|2[0-4]d|1dd|[1-9]?d)){3})|:))|(([0-9A-Fa-f]{1,4}:){5}(((:[0-9A-Fa-f]{1,4}){1,2})|:((25[0-5]|2[0-4]d|1dd|[1-9]?d)(.(25[0-5]|2[0-4]d|1dd|[1-9]?d)){3})|:))|(([0-9A-Fa-f]{1,4}:){4}(((:[0-9A-Fa-f]{1,4}){1,3})|((:[0-9A-Fa-f]{1,4})?:((25[0-5]|2[0-4]d|1dd|[1-9]?d)(.(25[0-5]|2[0-4]d|1dd|[1-9]?d)){3}))|:))|(([0-9A-Fa-f]{1,4}:){3}(((:[0-9A-Fa-f]{1,4}){1,4})|((:[0-9A-Fa-f]{1,4}){0,2}:((25[0-5]|2[0-4]d|1dd|[1-9]?d)(.(25[0-5]|2[0-4]d|1dd|[1-9]?d)){3}))|:))|(([0-9A-Fa-f]{1,4}:){2}(((:[0-9A-Fa-f]{1,4}){1,5})|((:[0-9A-Fa-f]{1,4}){0,3}:((25[0-5]|2[0-4]d|1dd|[1-9]?d)(.(25[0-5]|2[0-4]d|1dd|[1-9]?d)){3}))|:))|(([0-9A-Fa-f]{1,4}:){1}(((:[0-9A-Fa-f]{1,4}){1,6})|((:[0-9A-Fa-f]{1,4}){0,4}:((25[0-5]|2[0-4]d|1dd|[1-9]?d)(.(25[0-5]|2[0-4]d|1dd|[1-9]?d)){3}))|:))|(:(((:[0-9A-Fa-f]{1,4}){1,7})|((:[0-9A-Fa-f]{1,4}){0,5}:((25[0-5]|2[0-4]d|1dd|[1-9]?d)(.(25[0-5]|2[0-4]d|1dd|[1-9]?d)){3}))|:)))(%.+)?s*$'
        )
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
