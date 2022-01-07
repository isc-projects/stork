import { Component, Input, OnInit } from '@angular/core'

/**
 * A component displaying a DHCP identifier in hex and optionally
 * a string format.
 *
 * Identifiers are used to associate host reservations and leases
 * with clients.   Typical lease identifiers are MAC address and
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
     * It can contain colons and spaces. It must have even number
     * of hexadecimal digits.
     */
    @Input() hexValue = ''

    /**
     * Optional identifier label, e.g. hw-address.
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
     * The identifiers like hw-address should typically be displayed
     * in the hex format rather than textual. Setting this value to
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
    private _hexId = ''

    /**
     * Holds the identifier in the textual format if available.
     */
    private _textId = ''

    /**
     * Holds the currently displayed identifier.
     *
     * It is set to one of the values: _hexId or _textId.
     */
    private _currentId = ''

    /**
     * Boolean value indicating if the identifier is convertible to
     * a textual form.
     */
    convertible = false

    /**
     * Boolean value indicating if the currently displayed value is
     * displayed in the hex format.
     *
     * This value is bound to a toggle button to track its current state.
     */
    hexFormat = false

    /**
     * No-op constructor.
     */
    constructor() {}

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
        if (parsedValue === '') {
            // The identifier was invalid or empty. Display an error message.
            // It is unlikely because the server should return properly
            // formatted values.
            this._currentId = 'unrecognized hex string'
        } else {
            // If the identifier was parsed correctly we can use it as the
            // valid identifier in hex format.
            this._hexId = this.hexValue
            // The input value is returned when the identifier is valid but
            // it is not convertible to a textual format.
            if (parsedValue === this.hexValue) {
                // In that case switch to the hex format and do not assign
                // the textual value.
                this.hexFormat = true
                this._currentId = this._hexId
            } else {
                // The identifier is convertible to a textual format. Mark the
                // value as convertible to enable the toggle button.
                this._textId = parsedValue
                this.convertible = true
                console.info(this.defaultHexFormat)
                if (this.defaultHexFormat) {
                    // Use the hex format by default when externally requested
                    // by a caller.
                    this.hexFormat = true
                    this._currentId = this._hexId
                } else {
                    // Caller did not request the hex format, so show the textual
                    // format by default.
                    this._currentId = this._textId
                }
            }
        }
    }

    /**
     * Parse an identifier specified as a string of space or colon separated
     * hexadecimal digits into a textual form.
     *
     * There must be an even number of digits in the string.
     *
     * @param value input string holding an identifier in the hex format.
     * @return An empty string when the specified identifier is invalid or
     * empty; an input string when the identifier is not convertible to a
     * textual format; otherwise, an identifier converted to a textual format.
     */
    private _parse(value: string): string {
        const inputValue = value.replace(/\:|\s/g, '')
        if (inputValue.length === 0 || inputValue.length % 2 !== 0) {
            return ''
        }
        let outputValue = ''
        for (let n = 0; n < inputValue.length; n += 2) {
            const charCode = parseInt(inputValue.substr(n, 2), 16)
            if (isNaN(charCode)) {
                return ''
            }
            if (charCode < 32 || charCode > 126) {
                outputValue = value
                break
            }
            outputValue += String.fromCharCode(charCode)
        }
        return outputValue
    }

    /**
     * Returns a text displayed by the component.
     *
     * If a label was specified, it returns the label and an identifier in
     * the following format: <label>=(<identifier>). Otherwise, it returns
     * the identifier without parens.
     *
     * @return Optional label and an identifier displayed by the component.
     */
    get displayedText(): string {
        let text = ''
        if (this.label != '') {
            text = this.label + '=('
        }
        text += this._currentId
        if (this.label != '') {
            text += ')'
        }
        return text
    }

    /**
     * Toggles between the identifier formats.
     *
     * @param e event emitted as a result of pressing the toggle button.
     */
    toggle(e) {
        this._currentId = e.checked ? this._hexId : this._textId
    }
}
