import { LocalHost } from './backend'

/**
 * Utility function checking if there are differences between
 * configurable local host data.
 *
 * The local host holds the data specific to the DHCP servers
 * owning the host. The host can include different DHCP options,
 * client classes and boot fields for different DHCP servers.
 *
 * @param localHosts local host instances.
 * @returns true if there are differences in DHCP options,
 * client classes or boot fields between the local hosts,
 * false otherwise.
 */
export function hasDifferentLocalHostData(localHosts: LocalHost[]): boolean {
    return (
        hasDifferentLocalHostOptions(localHosts) ||
        hasDifferentLocalHostClientClasses(localHosts) ||
        hasDifferentLocalHostBootFields(localHosts) ||
        hasDifferentLocalHostHostname(localHosts) ||
        hasDifferentLocalHostIPReservations(localHosts)
    )
}

/**
 * Utility function checking if there are differences between
 * DHCP options in the local hosts.
 *
 * @param localHosts local host instances.
 * @returns true if there are differences in DHCP options, false
 * otherwise.
 */
export function hasDifferentLocalHostOptions(localHosts: LocalHost[]): boolean {
    if (localHosts == null || localHosts.length <= 1) {
        return false
    }
    return localHosts.slice(1).some((lh) => lh.optionsHash !== localHosts[0].optionsHash)
}

/**
 * Utility function checking if there are differences between
 * client classes in the local hosts.
 *
 * @param localHosts local host instances.
 * @returns true if there are differences in client classes, false
 * otherwise.
 */
export function hasDifferentLocalHostClientClasses(localHosts: LocalHost[]): boolean {
    if (localHosts == null || localHosts.length <= 1) {
        return false
    }

    return localHosts
        .slice(1)
        .some((lh) => JSON.stringify(lh.clientClasses?.sort()) !== JSON.stringify(localHosts[0].clientClasses?.sort()))
}

/**
 * Utility function checking if there are differences between boot
 * fields, i.e. next server, server hostname or boot file name.
 *
 * @param localHosts local host instances.
 * @returns true if there are differences in boot fields, false otherwise.
 */
export function hasDifferentLocalHostBootFields(localHosts: LocalHost[]): boolean {
    if (localHosts == null || localHosts.length <= 1) {
        return false
    }

    const reference = localHosts[0]
    return localHosts
        .slice(1)
        .some(
            (lh) =>
                lh.nextServer !== reference.nextServer ||
                lh.serverHostname !== reference.serverHostname ||
                lh.bootFileName !== reference.bootFileName
        )
}

/**
 * Utility function checking if there are differences between IP reservations
 * in the local hosts.
 *
 * @param localHosts local host instances.
 * @returns true if there are differences in IP reservations, false
 * otherwise.
 */
export function hasDifferentLocalHostIPReservations(localHosts: LocalHost[]): boolean {
    if (localHosts == null || localHosts.length <= 1) {
        return false
    }

    const getSortedIPAddresses = (lh: LocalHost) => (lh.ipReservations ?? []).map((r) => r.address).sort()
    const referenceAddresses = getSortedIPAddresses(localHosts[0])
    return localHosts
        .slice(1)
        .some(
            (lh) =>
                (lh.ipReservations?.length ?? 0) != referenceAddresses.length ||
                getSortedIPAddresses(lh).some((a, i) => a !== referenceAddresses[i])
        )
}

/**
 * Utility function checking if there are differences between hostnames
 * in the local hosts.
 *
 * @param localHosts local host instances.
 * @returns true if there are differences in hostnames, false otherwise.
 */
export function hasDifferentLocalHostHostname(localHosts: LocalHost[]): boolean {
    if (localHosts == null || localHosts.length <= 1) {
        return false
    }

    return localHosts.slice(1).some((lh) => lh.hostname !== localHosts[0].hostname)
}
