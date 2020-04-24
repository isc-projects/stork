import moment from 'moment-timezone'

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

export function durationToString(duration) {
    if (duration > 0) {
        const d = moment.duration(duration, 'seconds')
        let txt = ''
        if (d.days() > 0) {
            txt += ' ' + d.days() + ' days'
        }
        if (d.hours() > 0) {
            txt += ' ' + d.hours() + ' hours'
        }
        if (d.minutes() > 0) {
            txt += ' ' + d.minutes() + ' minutes'
        }
        if (d.seconds() > 0) {
            txt += ' ' + d.seconds() + ' seconds'
        }

        return txt.trim()
    }
    return ''
}

/**
 * Present count in human reabable way ie. big numbers get unit, e.g. 102 M instead of 102342543.
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
 * Extract key=val pairs from search text and prepare
 * query params dict.
 */
export function extractKeyValsAndPrepareQueryParams(text, keys) {
    // find all occurences key=val in the text
    const re = /(\w+)=(\w*)/g
    const matches = []
    let match = re.exec(text)
    while (match !== null) {
        matches.push(match)
        match = re.exec(text)
    }

    const queryParams = {
        text: null,
    }

    for (const key of keys) {
        queryParams[key] = null
        for (const m of matches) {
            if (m[1] === key) {
                queryParams[key] = m[2]
                text = text.replace(m[0], '')
            }
        }
    }
    text = text.trim()
    if (text) {
        queryParams.text = text
    }

    return queryParams
}
