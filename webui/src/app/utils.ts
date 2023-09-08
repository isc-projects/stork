import * as moment from 'moment-timezone'
import { IPv6, collapseIPv6Number } from 'ip-num'
import { Bind9Daemon, KeaDaemon } from './backend'
import { Observable, Subject } from 'rxjs'
import { ActivatedRoute, ParamMap, Router } from '@angular/router'

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
    } catch (e) {
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
 * Present count in human readable way ie. big numbers get unit, e.g. 102 M instead of 102342543.
 */
export function humanCount(count: string | bigint | number) {
    if (count == null || (typeof count !== 'number' && typeof count !== 'bigint') || Number.isNaN(count)) {
        return count + '' // Casting to string safe for null and undefined
    }

    const units = ['k', 'M', 'G', 'T', 'P', 'E', 'Z', 'Y']
    let u = -1
    while (count >= 1000 && u < units.length - 1) {
        if (typeof count === 'number') {
            count /= 1000
        } else {
            count /= BigInt(1000)
        }
        ++u
    }

    let countStr = ''
    if (typeof count === 'number') {
        countStr = count.toFixed(u >= 0 ? 1 : 0)
    } else {
        countStr = count.toString()
    }
    return countStr + (u >= 0 ? units[u] : '')
}

/**
 * Build URL to Grafana dashboard
 */
export function getGrafanaUrl(grafanaBaseUrl: string, name: string, subnet?: string, instance?: string): string {
    let url = null
    if (name === 'dhcp4') {
        if (instance) {
            instance += ':9547'
        }
        if (!grafanaBaseUrl.endsWith('/')) {
            grafanaBaseUrl += '/'
        }
        url = new URL('./d/hRf18FvWz/', grafanaBaseUrl)
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
 * and prepare query params dict. Expected keys are passed as keys
 * and expected flags are passed as flags.
 */
export function extractKeyValsAndPrepareQueryParams<T extends { text: string }>(text: string, keys: (keyof T)[], flags: (keyof T)[]): Partial<{ [key in keyof T]: string}> {
    // find all occurrences key:val in the text
    const re = /(\w+):(\w*)/g
    const matches = []
    let match = re.exec(text)
    while (match !== null) {
        matches.push(match)
        match = re.exec(text)
    }

    // reset query params
    const queryParams: Partial<{ [key in keyof T]: string}> = {}
    queryParams.text = null

    for (const key of keys) {
        queryParams[key] = null
    }
    if (flags) {
        for (const flag of flags) {
            queryParams[flag] = null
        }
    }

    // go through all matches and...
    for (const m of matches) {
        let found = false

        // look for keys
        for (const key of keys) {
            if (m[1] === key) {
                queryParams[key] = m[2]
                found = true
                break
            }
        }

        // look for flags
        if (flags && !found) {
            for (const flag of flags) {
                if (m[2] !== flag) {
                    continue
                }
                if (m[1] === 'is') {
                    queryParams[m[2]] = 'true'
                    found = true
                } else if (m[1] === 'not') {
                    queryParams[m[2]] = 'false'
                    found = true
                }
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
        return 'grey'
    }
    if (!daemon.active) {
        return '#f11'
    }
    return '#00a800'
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
export function copyToClipboard(textEl: HTMLInputElement) {
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
    let [baseNetwork, _] = prefix.split('/')
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
