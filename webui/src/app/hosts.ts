import { Host } from './backend'

/**
 * Utility function checking if there are differences between
 * configurable local host data in the host.
 *
 * The local host holds the data specific to the DHCP servers
 * owning the host. The host can include different DHCP options,
 * client classes and boot fields for different DHCP servers.
 *
 * @param host host instance.
 * @returns true if there are differences in DHCP options,
 * client classes or boot fields between the local hosts,
 * false otherwise.
 */
export function hasDifferentLocalHostData(host: Host): boolean {
    return (
        hasDifferentLocalHostOptions(host) ||
        hasDifferentLocalHostClientClasses(host) ||
        hasDifferentLocalHostBootFields(host)
    )
}

/**
 * Utility function checking if there are differences between
 * DHCP options in the host.
 *
 * @param host host instance.
 * @returns true if there are differences in DHCP options, false
 * otherwise.
 */
export function hasDifferentLocalHostOptions(host: Host): boolean {
    return (
        !!(host.localHosts?.length > 0) &&
        host.localHosts.slice(1).some((lh) => lh.optionsHash !== host.localHosts[0].optionsHash)
    )
}

/**
 * Utility function checking if there are differences between
 * client classes in the host.
 *
 * @param host host instance.
 * @returns true if there are differences in client classes, false
 * otherwise.
 */
export function hasDifferentLocalHostClientClasses(host: Host): boolean {
    return (
        !!(host.localHosts?.length > 0) &&
        host.localHosts
            .slice(1)
            .some(
                (lh) =>
                    JSON.stringify(lh.clientClasses?.sort()) !==
                    JSON.stringify(host.localHosts[0].clientClasses?.sort())
            )
    )
}

/**
 * Utility function checking if there are differences between boot
 * fields, i.e. next server, server hostname or boot file name.
 *
 * @param host host instance.
 * @returns true if there are differences in boot fields, false otherwise.
 */
export function hasDifferentLocalHostBootFields(host: Host): boolean {
    return (
        !!(host.localHosts?.length > 0) &&
        host.localHosts
            .slice(1)
            .some(
                (lh) =>
                    lh.nextServer !== host.localHosts[0].nextServer ||
                    lh.serverHostname !== host.localHosts[0].serverHostname ||
                    lh.bootFileName !== host.localHosts[0].bootFileName
            )
    )
}
