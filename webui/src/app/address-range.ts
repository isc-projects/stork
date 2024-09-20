import { AbstractIPNum, IPv4, IPv6, collapseIPv6Number } from 'ip-num'
import { isIPv4 } from 'ip-num'

/**
 * IP address range.
 *
 * It is used to parse and represent address ranges specified as strings in
 * the address pool declarations.
 */
export class AddressRange {
    /**
     * Parsed first address in the range.
     */
    readonly first: AbstractIPNum

    /**
     * Parsed last address in the range.
     */
    readonly last: AbstractIPNum

    /**
     * Instantiates an address range specified as <address range start>-<address-range-end>.
     *
     * @param ipRangeFirst first address in the range specified as string.
     * @param ipRangeLast last address in the range specified as string.
     * @throws Error when the address range boundaries are invalid or do not belong
     * to the same family.
     */
    constructor(ipRangeFirst: string, ipRangeLast: string) {
        let first: AbstractIPNum
        let last: AbstractIPNum
        try {
            // Parse both ends of the address range as IP addresses.
            first = ipRangeFirst.includes('.') ? new IPv4(ipRangeFirst.trim()) : new IPv6(ipRangeFirst.trim())
            last = ipRangeLast.includes('.') ? new IPv4(ipRangeLast.trim()) : new IPv6(ipRangeLast.trim())
        } catch {
            throw Error(`invalid IP addresses in the ${ipRangeFirst}-${ipRangeLast} range`)
        }
        // They should both have the same family.
        if (isIPv4(first) != isIPv4(last)) {
            throw Error(`inconsistent IP address family in the ${ipRangeFirst}-${ipRangeLast} range`)
        }
        if (first.isGreaterThan(last)) {
            throw Error(`first address in the range must not be greater than the last address`)
        }
        this.first = first
        this.last = last
    }

    /**
     * Convenience function creating an address range from string.
     *
     * @param ipRange an address range specified as string.
     * @returns Address range instance.
     * @throws Error when the address range boundaries are invalid or do not belong
     * to the same family.
     */
    static fromStringRange(ipRange: string): AddressRange {
        // The address range should be in the form of 192.0.2.1-192.0.2.10.
        const splitRange = ipRange.split('-')
        if (splitRange.length != 2) {
            throw Error(`${ipRange} is not a valid address range`)
        }
        return new AddressRange(splitRange[0], splitRange[1])
    }

    /**
     * Convenience function creating an address range from first and last address.
     *
     * @param ipRangeFirst first address in the range as string.
     * @param ipRangeLast last address in the range as string.
     * @returns Address range instance.
     * @throws Error when the address range boundaries are invalid or do not belong
     * to the same family.
     */
    static fromStringBounds(ipRangeFirst: string, ipRangeLast: string): AddressRange {
        return new AddressRange(ipRangeFirst, ipRangeLast)
    }

    /**
     * Converts an IP address to a string.
     *
     * IPv6 addresses are collapsed into an abbreviated form.
     *
     * @param ip an IP address belonging to the address range.
     * @returns An IP address as a string.
     */
    private convertToString(ip: AbstractIPNum): string {
        return isIPv4(ip) ? (ip as IPv4).toString() : collapseIPv6Number((ip as IPv6).toString())
    }

    /**
     * Returns the first address in the range as string.
     *
     * @returns Address as a string.
     */
    getFirst(): string {
        return this.convertToString(this.first)
    }

    /**
     * Returns the last address in the range as string.
     *
     * @returns Address as a string.
     */
    getLast(): string {
        return this.convertToString(this.last)
    }
}
