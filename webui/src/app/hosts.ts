import { Host } from './backend'

/**
 * Utility function checking if the are differences between
 * configurable local host data in the host.
 *
 * The local host holds the data specific to the DHCP servers
 * owning the host. The host can include different DHCP options
 * and client classes for different DHCP servers.
 *
 * @param host host instance.
 * @returns true if there are differences in DHCP options or
 * client classes between the local hosts, false otherwise.
 */
export function hasDifferentLocalHostData(host: Host): boolean {
    return host.localHosts?.length > 0 &&
        (host.localHosts.slice(1).some((lh) => lh.optionsHash !== host.localHosts[0].optionsHash) ||
            host.localHosts
                .slice(1)
                .some((lh) => JSON.stringify(lh.clientClasses) !== JSON.stringify(host.localHosts[0].clientClasses)))
        ? true
        : false
}
