import { ValidatorFn, Validators } from '@angular/forms'

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
     * A validator checking if an input is a valid IPv4 address.
     *
     * @returns validator function.
     */
    static ipv4(): ValidatorFn {
        return Validators.pattern('^(?:[0-9]{1,3}.){3}[0-9]{1,3}$')
    }

    /**
     * A validator checking if an input is a valid IPv6 address or a prefix
     * without the length.
     *
     * @returns validator function.
     */
    static ipv6(): ValidatorFn {
        return Validators.pattern(
            '([0-9a-fA-F]{1,4}:){7,7}[0-9a-fA-F]{1,4}|([0-9a-fA-F]{1,4}:){1,7}:|([0-9a-fA-F]{1,4}:){1,6}:[0-9a-fA-F]{1,4}|([0-9a-fA-F]{1,4}:){1,5}(:[0-9a-fA-F]{1,4}){1,2}|([0-9a-fA-F]{1,4}:){1,4}(:[0-9a-fA-F]{1,4}){1,3}|([0-9a-fA-F]{1,4}:){1,3}(:[0-9a-fA-F]{1,4}){1,4}|([0-9a-fA-F]{1,4}:){1,2}(:[0-9a-fA-F]{1,4}){1,5}|[0-9a-fA-F]{1,4}:((:[0-9a-fA-F]{1,4}){1,6})|:((:[0-9a-fA-F]{1,4}){1,7}|:)|fe80:(:[0-9a-fA-F]{0,4}){0,4}%[0-9a-zA-Z]{1,}|::(ffff(:0{1,4}){0,1}:){0,1}((25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9]).){3,3}(25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9])|([0-9a-fA-F]{1,4}:){1,4}:((25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9]).){3,3}(25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9])'
        )
    }
}
