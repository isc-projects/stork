import * as moment from 'moment-timezone'

export function datetimeToLocal(d) {
    try {
        let tz = Intl.DateTimeFormat().resolvedOptions().timeZone
        if (!tz) {
            tz = moment.tz.guess()
        }
        if (tz) {
            d = moment(d).tz(tz)
            tz = ''
        } else {
            d = moment(d)
            tz = ' UTC'
        }

        // If year is < 2 it means that the date is not set.
        // In Go if date is zeroed then it is 0001.01.01.
        if (d.year() < 2) {
            return ''
        }
        return d.format('YYYY-MM-DD HH:mm:ss') + tz
    } catch (e) {
        return d
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
 * @returns Duration in the format of "D days H hours M minutes S seconds"
 *          or "D d H h M min S sec".
 */
export function durationToString(duration, short = false) {
    if (duration > 0) {
        const d = moment.duration(duration, 'seconds')
        let txt = ''
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
export function humanCount(count) {
    const units = ['k', 'M', 'G', 'T', 'P', 'E', 'Z', 'Y']
    let u = -1
    do {
        count /= 1000
        ++u
    } while (Math.abs(count) >= 1000 && u < units.length - 1)
    return count.toFixed(1) + ' ' + units[u]
}

/**
 * Build URL to Grafana dashboard
 */
export function getGrafanaUrl(grafanaBaseUrl, name, subnet, instance) {
    let url = null
    if (name === 'dhcp4') {
        if (instance) {
            instance += ':9547'
        }
        console.info('grafanaBaseUrl', grafanaBaseUrl)
        const b = grafanaBaseUrl.replace(/\/+$/, '')
        console.info(b)
        url = new URL('/d/hRf18FvWz/', b)
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
export function getGrafanaSubnetTooltip(subnet, machine) {
    return 'See statistics for subnet ' + subnet + ' on machine ' + machine + ' in Grafana.'
}

/**
 * Extract key:val pairs, is:<flag> and not:<flag> from search text
 * and prepare query params dict. Expected keys are passed as keys
 * and expected flags are passed as flags.
 */
export function extractKeyValsAndPrepareQueryParams(text, keys, flags) {
    // find all occurrences key=val in the text
    const re = /(\w+):(\w*)/g
    const matches = []
    let match = re.exec(text)
    while (match !== null) {
        matches.push(match)
        match = re.exec(text)
    }

    // reset query params
    const queryParams = {
        text: null,
    }
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
                    queryParams[m[2]] = true
                    found = true
                } else if (m[1] === 'not') {
                    queryParams[m[2]] = false
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
 *  should be active but the communication with it is borken and
 *  check icon if the communication with the active daemon is ok.
 */
export function daemonStatusIconName(daemon) {
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
export function daemonStatusIconColor(daemon) {
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
export function daemonStatusIconTooltip(daemon) {
    if (!daemon.monitored) {
        return 'Monitoring of this daemon has been disabled. You can enable it on the daemon tab on the Kea app page.'
    }
    if (daemon.agentCommErrors && daemon.agentCommErrors > 0) {
        return (
            'Communication with the Stork Agent on this machine ' +
            'is broken. Last ' +
            daemon.agentCommErrors +
            ' attempt(s) to ' +
            'communicate with the agent failed. Please make sure ' +
            'that the agent is up on this machine and that the firewall ' +
            'settings permit to communicate with the agent.'
        )
    }
    if (daemon.caCommErrors && daemon.caCommErrors > 0) {
        return (
            'Communication with the Kea Control Agent on this machine ' +
            'is broken. The Stork Agent appears to work fine but the ' +
            'Kea CA is down or returns errors. Last ' +
            daemon.caCommErrors +
            ' attempt(s) to communicate with Kea CA failed. Please ' +
            'make sure that Kea CA is up and the firewall settings permit ' +
            'for the communication between the Stork Agent and Kea CA running ' +
            'on that machine.'
        )
    }
    if (daemon.daemonCommErrors && daemon.daemonCommErrors > 0) {
        return (
            'Communication with the daemon on this machine ' +
            'is broken. The Stork Agent and Kea Control Agent appear to ' +
            'work fine, but the daemon behind Kea CA does not respond or ' +
            'responds with errors. Last ' +
            daemon.daemonCommErrors +
            ' attempt(s) to communicate with the daemon failed. Please ' +
            'make sure that the daemon is up and is reachable from the ' +
            'Kea Control Agent over the control channel (unix domain socket).'
        )
    }
    if (daemon.rndcCommErrors && daemon.rndcCommErrors > 0) {
        return (
            'Communication with the BIND9 daemon over RNDC is broken. The ' +
            'Stork Agent appears to work fine, but the BIND9 daemon may ' +
            'be down or responds with errors. Last ' +
            daemon.rndcCommErrors +
            ' attempt(s) to communicate with BIND9 daemon failed.'
        )
    }
    if (daemon.statsCommErrors && daemon.statsCommErrors > 0) {
        return (
            'Communication with BIND9 statistics endpoint is broken. The ' +
            'Stork Agent appears to work fine, but the BIND9 statistics channel ' +
            'seems to be unreachable or responds with errors. Last ' +
            daemon.statsCommErrors +
            ' attempt(s) to communicate with the BIND9 daemon over ' +
            'the statistics channel failed.'
        )
    }
    return 'Communication with the daemon is ok.'
}

/**
 * Copy text to clipboard.
 *
 * @param textEl instance of the DOM entity the text will be copied from.
 */
export function copyToClipboard(textEl) {
    textEl.select()
    document.execCommand('copy')
    textEl.setSelectionRange(0, 0)
}
