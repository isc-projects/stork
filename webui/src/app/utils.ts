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
        return d.format('YYYY-MM-DD hh:mm:ss') + tz
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
 * Get subnet utilization in % based on stats.
 */
export function getSubnetUtilization(subnet) {
    let utilization = 0.0
    if (!subnet.stats) {
        return utilization
    }
    if (subnet.subnet.includes('.')) {
        // DHCPv4 stats
        utilization = (100 * subnet.stats['assigned-addreses']) / subnet.stats['total-addreses']
    } else {
        // DHCPv6 stats
        let total = subnet.stats['total-nas']
        if (total === -1) {
            total = Number.MAX_SAFE_INTEGER
        }
        utilization = (100 * subnet.stats['assigned-nas']) / total
    }
    return utilization.toFixed(0)
}
