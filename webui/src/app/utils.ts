import * as moment from 'moment-timezone'
import { IPv6, collapseIPv6Number } from 'ip-num'
import { gt, lt, valid } from 'semver'
import { Bind9Daemon, KeaDaemon } from './backend'

/**
 * Formats the date-like object as local date-time string.
 * @param d Date
 * @returns formatted string on success, otherwise stringified @d
 */
export function datetimeToLocal(d: moment.MomentInput): string | null {
    if (d == null) {
        return null
    }

    try {
        let tz = Intl.DateTimeFormat().resolvedOptions().timeZone
        if (!tz) {
            tz = moment.tz.guess()
        }

        let md = moment(d)

        if (tz) {
            md = md.tz(tz)
            tz = ''
        } else {
            tz = ' UTC'
        }

        return md.format('YYYY-MM-DD HH:mm:ss') + tz
    } catch {
        return d.toString()
    }
}

/**
 * Converts epoch time to local time.
 *
 * @param epochTime epoch time in seconds.
 * @returns Human readable local time.
 */
export function epochToLocal(epochTime) {
    // Date constructor takes epoch time in milliseconds.
    const d = new Date(epochTime * 1000)
    return datetimeToLocal(d)
}

/**
 * Return formatted time duration.
 *
 * @param duration input duration.
 * @param short boolean flag indicating if the duration should be output
 *              using short (if true) or long format (if false).
 * @returns Duration in the format of "Y years M months D days H hours
 *            M minutes S seconds or "Y y M m D d H h M min S sec".
 */
export function durationToString(duration: number, short = false) {
    if (duration > 0) {
        const d = moment.duration(duration, 'seconds')
        let txt = ''
        if (d.years() > 0) {
            txt += ' ' + d.years() + (short ? ' y' : ' years')
        }
        if (d.months() > 0) {
            txt += ' ' + d.months() + (short ? ' m' : ' months')
        }
        if (d.days() > 0) {
            txt += ' ' + d.days() + (short ? ' d' : ' days')
        }
        if (d.hours() > 0) {
            txt += ' ' + d.hours() + (short ? ' h' : ' hours')
        }
        if (d.minutes() > 0) {
            txt += ' ' + d.minutes() + (short ? ' min' : ' minutes')
        }
        if (d.seconds() > 0) {
            txt += ' ' + d.seconds() + (short ? ' s' : ' seconds')
        }

        return txt.trim()
    }
    return ''
}

/**
 * Present count in human readable way ie. big numbers get unit, e.g. 102.3 M
 * instead of 102342543.
 */
export function humanCount(count: string | bigint | number) {
    if (count == null || (typeof count !== 'number' && typeof count !== 'bigint') || Number.isNaN(count)) {
        return count + '' // Casting to string safe for null and undefined
    }

    // Decrease the input number to the safe range for the standard numeric type.
    let exponent = 0
    if (typeof count === 'bigint') {
        while (count > BigInt(Number.MAX_SAFE_INTEGER)) {
            count /= BigInt(1000)
            exponent += 3
        }
    }

    // Convert the count to a standard number.
    count = Number(count)

    const units = ['', 'k', 'M', 'G', 'T', 'P', 'E', 'Z', 'Y']

    // ~~number is the fastest way to truncate mantissa (fractional part).
    while (count >= 1000 && ~~(exponent / 3) < units.length - 1) {
        count /= 1000
        exponent += 3
    }

    const countStr = count.toFixed(exponent >= 3 ? 1 : 0)
    return countStr + units[~~(exponent / 3)]
}

/**
 * Build URL to Grafana dashboard
 */
export function getGrafanaUrl(grafanaBaseUrl: string, name: string, subnet?: string, instance?: string): string {
    if (!grafanaBaseUrl.endsWith('/')) {
        grafanaBaseUrl += '/'
    }

    let url = null
    if (name === 'dhcp4') {
        if (instance) {
            instance += ':9547'
        }
        url = new URL('./d/hRf18FvWz/', grafanaBaseUrl)
    } else if (name === 'dhcp6') {
        if (instance) {
            instance += ':9547'
        }
        url = new URL('./d/AQPHKJUGz/', grafanaBaseUrl)
    } else {
        return ''
    }

    const sp = new URLSearchParams()
    if (subnet) {
        sp.append('var-subnet', subnet)
    }
    if (instance) {
        sp.append('var-instance', instance)
    }
    const spStr = sp.toString()
    let urlStr = url.href
    if (spStr) {
        urlStr += '?' + spStr
    }
    return urlStr
}

/**
 * Builds a tooltip explaining what the link is for.
 * @param subnet an identifier of the subnet
 * @param machine an identifier of the machine the subnet is configured on
 */
export function getGrafanaSubnetTooltip(subnet: number, machine: string) {
    return 'See statistics for subnet ' + subnet + ' on machine ' + machine + ' in Grafana.'
}

/**
 * Extract key:val pairs, is:<flag> and not:<flag> from search text
 * and prepare query params dict. Accepts the keys of the numeric values and
 * boolean flags. The numeric values are converted to numbers, the flags to
 * boolean values. The rest of the text is stored under the 'text' key.
 * @param text search text
 * @param numericKeys keys of the numeric values
 * @param flags keys of the boolean flags
 */
export function extractKeyValsAndPrepareQueryParams<T extends { text?: string }>(
    text: string,
    numericKeys: (keyof T)[],
    flags?: (keyof T)[]
): Partial<{ [key in keyof T]: T[key] }> {
    // find all occurrences key:val in the text
    const re = /(\w+):(\w*)/g
    const matches = []
    let match = re.exec(text)
    while (match !== null) {
        matches.push(match)
        match = re.exec(text)
    }

    // reset query params
    const queryParams: Partial<T> = {}

    // go through all matches and...
    for (const m of matches) {
        let found = false

        // look for keys
        for (const key of numericKeys) {
            if (m[1] !== key) {
                continue
            }
            found = true
            const value = parseInt(m[2])
            queryParams[m[1]] = isNaN(value) ? null : value
            break
        }

        // look for flags
        if (flags && !found) {
            for (const flag of flags) {
                if (m[2] !== flag) {
                    continue
                }
                found = true
                let value: boolean = null
                if (m[1] === 'is') {
                    value = true
                } else if (m[1] === 'not') {
                    value = false
                }
                queryParams[m[2]] = value
            }
        }

        // if found match then remove it from text
        if (found) {
            text = text.replace(m[0], '')
        }
    }

    text = text.trim()
    if (text) {
        queryParams.text = text
    }

    return queryParams
}

/**
 * Returns boolean value indicating if there is an issue with communication
 * with the given daemon
 *
 * @param daemon data structure holding the information about the daemon.
 *
 * @return true if there is a communication problem with the daemon,
 *         false otherwise.
 */
export function daemonStatusErred(daemon) {
    return (
        (daemon.agentCommErrors && daemon.agentCommErrors > 0) ||
        (daemon.caCommErrors && daemon.caCommErrors > 0) ||
        (daemon.daemonCommErrors && daemon.daemonCommErrors > 0) ||
        (daemon.rndcCommErrors && daemon.rndcCommErrors > 0) ||
        (daemon.statsCommErrors && daemon.statsCommErrors > 0)
    )
}

/**
 * Returns the name of the icon to be used to indicate daemon status
 *
 * The icon selected depends on whether the daemon is active or not
 * active and whether there is a communication with the daemon or
 * not.
 *
 * @param daemon data structure holding the information about the daemon.
 *
 * @returns ban icon if the daemon is not active, times icon if the daemon
 *  should be active but the communication with it is broken and
 *  check icon if the communication with the active daemon is ok.
 */
export function daemonStatusIconName(daemon: KeaDaemon) {
    if (!daemon.monitored) {
        return 'pi-ban icon-not-monitored'
    }
    if (!daemon.active) {
        return 'pi-times icon-not-active'
    }
    return 'pi-check icon-ok'
}

/**
 * Returns the color of the icon used to indicate daemon status
 *
 * @param daemon data structure holding the information about the daemon.
 *
 * @returns grey color if the daemon is not active, red if the daemon is
 *          active but there are communication issues, green if the
 *          communication with the active daemon is ok.
 */
export function daemonStatusIconColor(daemon: KeaDaemon) {
    if (!daemon.monitored) {
        return 'var(--gray-400)'
    }
    if (!daemon.active) {
        return 'var(--red-500)'
    }
    return 'var(--green-500)'
}

/**
 * Returns tooltip for the icon presented for the daemon status
 *
 * @param daemon data structure holding the information about the daemon.
 *
 * @returns Tooltip as text. It includes hints about the communication
 *          problems when such problems occur, e.g. it includes the
 *          hint whether the communication is with the agent or daemon.
 */
export function daemonStatusIconTooltip(daemon: KeaDaemon & Bind9Daemon) {
    if (!daemon.monitored) {
        return 'Monitoring of this daemon has been disabled. It can be enabled on the daemon tab on the Kea Apps page.'
    }
    if (daemon.agentCommErrors && daemon.agentCommErrors > 0) {
        return (
            'Communication with the Stork Agent on this machine ' +
            'is broken. The last ' +
            daemon.agentCommErrors +
            ' attempt(s) to ' +
            'communicate with the agent failed. Please make sure ' +
            'that the agent is up on this machine and that the firewall ' +
            'settings permit communication with the agent.'
        )
    }
    if (daemon.caCommErrors && daemon.caCommErrors > 0) {
        return (
            'Communication with the Kea Control Agent on this machine ' +
            'is broken. The Stork Agent appears to be working, but the ' +
            'Kea CA is down or returns errors. The last ' +
            daemon.caCommErrors +
            ' attempt(s) to communicate with the Kea CA failed. Please ' +
            'make sure that Kea CA is up and that the firewall settings permit ' +
            'communication between the Stork Agent and Kea CA running ' +
            'on this machine.'
        )
    }
    if (daemon.daemonCommErrors && daemon.daemonCommErrors > 0) {
        return (
            'Communication with the daemon on this machine ' +
            'is broken. The Stork Agent and Kea Control Agent appear to ' +
            'be working, but the daemon behind Kea CA is not responding or ' +
            'responds with errors. The last ' +
            daemon.daemonCommErrors +
            ' attempt(s) to communicate with the daemon failed. Please ' +
            'make sure that the daemon is up and is reachable from the ' +
            'Kea Control Agent over the control channel (UNIX domain socket).'
        )
    }
    if (daemon.rndcCommErrors && daemon.rndcCommErrors > 0) {
        return (
            'Communication with the BIND 9 daemon over RNDC is broken. The ' +
            'Stork Agent appears to be working, but the BIND 9 daemon appears ' +
            'to be down or is responding with errors. The last ' +
            daemon.rndcCommErrors +
            ' attempt(s) to communicate with the BIND 9 daemon failed.'
        )
    }
    if (daemon.statsCommErrors && daemon.statsCommErrors > 0) {
        return (
            'Communication with the BIND 9 statistics endpoint is broken. The ' +
            'Stork Agent appears to be working, but the BIND 9 statistics channel ' +
            'seems to be unreachable or is responding with errors. The last ' +
            daemon.statsCommErrors +
            ' attempt(s) to communicate with the BIND 9 daemon over ' +
            'the statistics channel failed.'
        )
    }
    return 'Communication with the daemon is OK.'
}

/**
 * Copy text to clipboard.
 *
 * @param textEl instance of the DOM entity the text will be copied from.
 */
export function copyToClipboard(textEl: HTMLInputElement | HTMLTextAreaElement) {
    textEl.select()
    document.execCommand('copy')
    textEl.setSelectionRange(0, 0)
}

/**
 * Clamps number within the inclusive lower and upper bounds.
 *
 * @param value number to clamp
 * @param lower lower clamp bound
 * @param upper upper clamp bound
 *
 * @returns Clamped value to given range
 */
export function clamp(value: number, lower: number, upper: number): number {
    return Math.min(upper, Math.max(value, lower))
}

/**
 * Converts a text into a string of hexadecimal digits.
 *
 * @param s text to convert.
 * @param separator separator to use between groups of digits.
 * @returns converted string.
 */
export function stringToHex(s: string, separator = ':'): string {
    let output = []
    for (let i = 0; i < s.length; i++) {
        const hex = Number(s.charCodeAt(i)).toString(16)
        output.push(hex)
    }
    return output.join(separator)
}

/** General purpose function to extract an error message from object. */
export function getErrorMessage(err: any): string {
    if (err.error && err.error.message) {
        return err.error.message
    }
    if (err.statusText) {
        return err.statusText
    }
    if (err.status) {
        return `status: ${err.status}`
    }
    if (err.message) {
        return err.message
    }
    if (err.cause) {
        return err.cause
    }
    if (err.name) {
        return err.name
    }
    return err.toString()
}

/**
 * Returns the short representation of the excluded prefix.
 * The common octet pairs with the main prefix are replaced by ~.
 *
 * E.g.: for the 'fe80::/64' main prefix and the 'fe80:42::/80' excluded
 * prefix the short form is: '~:42::/80'.
 *
 * It isn't any well-known convention, just a simple idea to limit the
 * length of the bar.
 */
export function formatShortExcludedPrefix(prefix: string, excludedPrefix: string | null): string {
    if (!excludedPrefix) {
        return ''
    }

    // Split the network and length.
    let tokens = prefix.split('/')
    let baseNetwork = tokens[0]
    let [excludedNetwork, excludedLen] = excludedPrefix.split('/')

    let baseNetworkObj = IPv6.fromString(baseNetwork)
    let excludedNetworkObj = IPv6.fromString(excludedNetwork)

    baseNetwork = collapseIPv6Number(baseNetworkObj.toString())
    excludedNetwork = collapseIPv6Number(excludedNetworkObj.toString())

    // Trim the trailing double colon.
    if (baseNetwork.endsWith('::')) {
        baseNetwork = baseNetwork.slice(0, baseNetwork.length - 1)
    }

    // Check if the excluded prefix starts with the base prefix.
    // It should be always true for valid data.
    if (excludedNetwork.startsWith(baseNetwork)) {
        // Replace the common part with ~.
        excludedNetwork = excludedNetwork.slice(baseNetwork.length)
        return `~:${excludedNetwork}/${excludedLen}`
    }

    // Fallback to full excluded prefix.
    return excludedPrefix
}
/**
 * Constructs the base API URL by combining the global base URL from the
 * base HTML tag with the API URL provided in the Angular configuration.
 * It allows to configure the base URL in a single place (index.html) without
 * rebuilding the application.
 */
export function getBaseApiPath(apiUrl: string) {
    // Check if the API path is not relative to root.
    if (apiUrl.includes('://')) {
        // Contains protocol.
        return apiUrl
    }

    const baseElements = document.getElementsByTagName('base')
    if (baseElements.length === 0) {
        return apiUrl
    }

    let baseHref = baseElements[0].href
    if (baseHref.endsWith('/')) {
        baseHref = baseHref.slice(0, -1)
    }
    if (apiUrl.startsWith('/')) {
        apiUrl = apiUrl.slice(1)
    }
    return baseHref + '/' + apiUrl
}

// Mock of the Angular ParamMap class.
export class MockParamMap {
    /** Always returns null. */
    get(/* name: string */): string | null {
        return null
    }

    /** Always returns false. */
    has(/* name: string */): boolean {
        return false
    }
}

/**
 * Converts parameter names from camel case to long names.
 *
 * The words in the long names begin with upper case and are separated with
 * space characters. For example: 'cacheThreshold' becomes 'Cache Threshold'.
 *
 * It also handles several special cases. When the converted name begins with:
 * - pd - it is converted to PD,
 * - ip - it is converted to IP,
 * - underscore character - it is removed.
 *
 * When the name contains:
 * - ddns - it is converted to DDNS,
 * - dhcp - it is converted to DHCP
 *
 * The case of the converted special case strings is ignored.
 *
 * @param key a name to be converted in camel case notation.
 * @returns converted name.
 */
export function uncamelCase(key: string): string {
    let text = key.trim().replace(/_/g, '')
    if (text.length === 0) {
        return key
    }
    text = text.replace(/([A-Z]+)/g, ' $1')
    text = text.replace(/ddns/gi, 'DDNS')
    text = text.replace(/dhcp/gi, 'DHCP')
    text = text.replace(/^pd/gi, 'PD')
    text = text.replace(/^ip/gi, 'IP')
    text = text.charAt(0).toUpperCase() + text.slice(1)
    return text
}

/**
 * Converts a parameter name from JSON notation with hyphens to camel case.
 *
 * It removes hyphens and replaces them with spaces. All words following
 * the hyphens are converted to begin with a capital letter.
 *
 * @param key a name to be converted from JSON notation to camel case.
 * @returns converted name.
 */
export function unhyphen(key: string): string {
    let text = key.trim().replace(/-/g, ' ')
    if (text.length === 0) {
        return key
    }
    let position = 0
    while (position >= 0) {
        position = text.indexOf(' ', position)
        if (position >= 0 && position < text.length - 1) {
            text = text.slice(0, position) + text.charAt(position + 1).toUpperCase() + text.slice(position + 2)
        }
    }
    return text
}

/**
 * Returns severity as text for an index.
 *
 * It is useful in cases when there are several managed servers indexed
 * with numbers. To visually distinguish the servers we sometimes use
 * tags that are colored using severity.
 *
 * @param index an index.
 * @returns `success` for 0, `warning` for 1, `danger` for 2, and 'info'
 * for any other.
 */
export function getSeverityByIndex(index: number): string {
    switch (index) {
        case 0:
            return 'success'
        case 1:
            return 'warning'
        case 2:
            return 'danger'
        default:
            return 'info'
    }
}

/**
 * Returns a string comprising a count and a noun in the plural or
 * singular form, depending on the count.
 *
 * @param count a number of counted items
 * @param noun a noun
 * @param postfix a postfix to be appended to the noun in the plural form
 * @returns formatted noun.
 */
export function formatNoun(count: number, noun, postfix: string): string {
    if (Math.abs(count) != 1) {
        return `${count} ${noun}${postfix}`
    }
    return `${count} ${noun}`
}

/**
 * Deeply copies the object.
 *
 * The copied object must be convertible to JSON.
 *
 * @param obj object to be copied.
 * @returns copied object.
 */
export function deepCopy<T>(obj: T): T {
    return JSON.parse(JSON.stringify(obj))
}

/**
 * Returns friendly daemon name.
 *
 * @param daemonName daemon name from the structures returned by the server.
 * @returns Friendly daemon name.
 */
export function daemonNameToFriendlyName(daemonName: string): string {
    switch (daemonName) {
        case 'dhcp4':
            return 'DHCPv4'
        case 'dhcp6':
            return 'DHCPv6'
        case 'd2':
            return 'DDNS'
        case 'ca':
            return 'CA'
        case 'netconf':
            return 'NETCONF'
        default:
            return daemonName
    }
}

/**
 * Given an array of semantic versions returns the earliest and the latest
 * version from the array.
 *
 * It excludes invalid versions.
 *
 * @param versions semantic versions in no particular order.
 * @returns The earliest and the latest version, or null if there are no valid versions.
 */
export function getVersionRange(versions: string[]): [string, string] | null {
    const validVersions = versions.filter((v) => valid(v))
    if (validVersions.length > 0) {
        const min = validVersions.reduce((prev, curr) => (!prev || lt(curr, prev) ? curr : prev))
        const max = validVersions.reduce((prev, curr) => (!prev || gt(curr, prev) ? curr : prev))
        return [min, max]
    }
    return null
}

/**
 * Deeply compares two objects.
 */
export function deepEqual<T>(a: T, b: T, parents: [any, any][] = []): boolean {
    if (typeof a !== 'object' || typeof b !== 'object' || a == null || b == null) {
        return a === b
    }

    // Check for circular references.
    if (parents.some(([aParent, bParent]) => aParent === a && bParent === b)) {
        return true
    }

    const aKeys = Object.keys(a)
    const bKeys = Object.keys(b)
    if (aKeys.length !== bKeys.length) {
        return false
    }

    for (const key of aKeys) {
        if (!bKeys.includes(key)) {
            return false
        }

        if (!deepEqual(a[key], b[key], parents.concat([[a, b]]))) {
            return false
        }
    }

    return true
}
