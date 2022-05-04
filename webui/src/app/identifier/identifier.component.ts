import { Component, Input, OnInit } from '@angular/core'

/**
 * A component displaying a DHCP identifier in hex and optionally
 * a string format.
 *
 * Identifiers are used to associate host reservations and leases
 * with clients. Typical lease identifiers are MAC address and
 * client-id. Host reservations have more identifier types:
 * e.g., circuit-id, flex-id. All identifiers can be represented
 * using strings of hexadecimal digits. Some identifiers (typically
 * circuit-id and flex-id) can also be represented as ASCII text.
 * This component expects an identifier formatted using the string
 * of hexadecimal digits as an input value. It detects whether or
 * not it can convert it to a textual format. In this case, it
 * provides a button next to the identifier to toggle between the
 * hex and text formats.
 *
 * If a label is specified besides the identifier, the label and
 * the identifier are displayed in the following format:
 *
 *  <label>=(<identifier>)
 *
 * If the label is not specified only the identifier without
 * parens is displayed.
 *
 */
@Component({
    selector: 'app-identifier',
    templateUrl: './identifier.component.html',
    styleUrls: ['./identifier.component.sass'],
})
export class IdentifierComponent implements OnInit {
    /**
     * Identifier in the hex format.
     *
     * It can contain colons and spaces. It must have an even number
     * of hexadecimal digits.
     */
    @Input() hexValue = ''

    /**
     * Optional identifier label, e.g., hw-address.
     */
    @Input() label = ''

    /**
     * Optional link.
     *
     * If the value is specified, the identifier becomes a link to
     * the specified target rather than a span.
     */
    @Input() link = ''

    /**
     * Specifies if by default the identifier should be displayed as
     * a string of hexadecimal digits, even if it is convertible to
     * a textual format.
     *
     * Identifiers like hw-address should typically be displayed
     * in hex format rather than textual. Setting this value to
     * true does not preclude on-demand conversion to the textual form
     * using the button if the identifier is convertible.
     */
    @Input() defaultHexFormat = false

    /**
     * Class used for styling the component view.
     */
    @Input() styleClass = ''

    /**
     * Holds the identifier in the hex format.
     */
    hexId: string = null

    /**
     * Holds the identifier in the textual format if available.
     */
    textId: string = null

    /**
     * Boolean value indicating if the currently displayed value is
     * displayed in the hex format.
     *
     * This value is bound to a toggle button to track its current state.
     */
    hexFormat = true

    /**
     * A hook invoked during the component initialization.
     *
     * It attempts to parse the input hexValue. If the value is empty or
     * it is not a valid string of hexadecimal digits the component displays
     * an error string in place of the identifier. If the hex string is valid
     * the component displays the converted (text) value unless the
     * defaultHexFormat is on.
     */
    ngOnInit() {
        // Attempt to parse the specified identifier.
        const parsedValue = this._parse(this.hexValue)
        // If there was an error parsing the input value or the input
        // value is not convertible to text, let's use the output from
        // parsing and assign it to hexId. If the hexId becomes null,
        // an error will be displayed instead of the identifier.
        if (parsedValue === null || parsedValue === this.hexValue) {
            this.hexId = parsedValue
            return
        }
        // Set the identifiers in hex and text formats.
        this.hexId = this.hexValue
        this.textId = parsedValue
        // Typically, the text format is the default but it can be overridden
        // by the caller, e.g., for MAC addresses.
        this.hexFormat = this.defaultHexFormat
    }

    /**
     * Parse an identifier specified as a string of space or colon separated
     * hexadecimal digits into a textual form.
     *
     * There must be an even number of digits in the string.
     *
     * @param value input string holding an identifier in the hex format.
     * @return null if the specified identifier is invalid or empty;
     * an input string when the identifier is not convertible to a textual
     * format; otherwise, an identifier converted to a textual format.
     */
    private _parse(value: string): string | null {
        const inputValue = value.replace(/\:|\s/g, '')
        if (inputValue.length === 0 || inputValue.length % 2 !== 0) {
            return null
        }
        let outputValue = ''
        for (let n = 0; n < inputValue.length; n += 2) {
            const charCode = parseInt(inputValue.substr(n, 2), 16)
            if (isNaN(charCode)) {
                return null
            }
            if (charCode < 32 || charCode > 126) {
                return value
            }
            outputValue += String.fromCharCode(charCode)
        }
        return outputValue
    }

    /**
     * Conditionally wraps the specified string value with label and parens.
     *
     * If a label is specified, it returns the label and the specified text
     * value in the following format: <label>=(<value>). Otherwise, it returns
     * the value (i.e., identifier or an error text) without parens.
     *
     * @return Optional label and value that can be an identifier or an error
     * text.
     */
    condWrap(value: string): string {
        let text = value
        if (this.label != '') {
            text = `${this.label}=(${text})`
        }
        return text
    }
}
