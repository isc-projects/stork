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
     * Bytes of the hex identifier.
     */
    hexBytes: number[] = []

    /**
     * Normalized hex identifier.
     */
    _hexValue: string = ''

    /**
     * Identifier in the hex format.
     *
     * It can contain colons and spaces. It must have an even number
     * of hexadecimal digits.
     */
    get hexValue(): string {
        return this._hexValue
    }

    @Input() set hexValue(value: string) {
        this._hexValue = this.normalizeHexString(value)
        this.hexBytes = this.splitIntoBytes(this._hexValue)
    }

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
        // Typically, the text format is the default but it can be overridden
        // by the caller, e.g., for MAC addresses.
        this.hexFormat = this.defaultHexFormat
    }

    /**
     * Indicates if the hex identifier is empty.
     */
    get isEmpty(): boolean {
        return this.hexBytes.length === 0
    }

    /**
     * Returns a list of bytes in the hex identifier.
     */
    private splitIntoBytes(normalizedHexValue: string): number[] {
        if (normalizedHexValue.length === 0) {
            return []
        }

        const output: number[] = []
        for (const byteStr of normalizedHexValue.split(':')) {
            const charCode = parseInt(byteStr, 16)
            output.push(charCode)
        }
        return output
    }

    /**
     * Normalizes the hex identifier. Replace spaces with colons or add them
     * if they are missing.
     */
    private normalizeHexString(hexValue: string): string {
        hexValue = hexValue.replace(/\:|\s/g, '')
        const bytes = []
        for (let n = 0; n < hexValue.length; n += 2) {
            bytes.push(hexValue.slice(n, n + 2))
        }
        return bytes.join(':')
    }
}
