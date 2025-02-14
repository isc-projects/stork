import { Component, EventEmitter, Input, Output } from '@angular/core'

import { ConfirmationService, MessageService } from 'primeng/api'
import { Host, Lease, LeasesSearchErredApp, LocalHost } from '../backend'

import { DHCPService } from '../backend/api/api'
import {
    hasDifferentLocalHostBootFields,
    hasDifferentLocalHostClientClasses,
    hasDifferentLocalHostData,
    hasDifferentLocalHostHostname,
    hasDifferentLocalHostIPReservations,
    hasDifferentLocalHostOptions,
} from '../hosts'
import { durationToString, epochToLocal, getErrorMessage } from '../utils'

enum HostReservationUsage {
    Conflicted = 1,
    Declined,
    Expired,
    Used,
}

/**
 * The internal lease info structure.
 */
interface LeaseInfo {
    leases: Lease[]
    usage?: HostReservationUsage
    culprit?: Lease
}

/**
 * Component presenting reservation details for a selected host.
 *
 * It is embedded in the apps-page and used in cases when a user
 * selects one or more host reservations. If multiple host tabs are
 * opened, a single instance of this component is used to present
 * information associated with those tabs. Selecting a different
 * tab causes the change of the host input property. This also
 * triggers a REST API call to fetch leases for the reserved
 * addresses and prefixes, if they haven't been fetched yet for
 * the given host. The lease information is cached for all opened
 * tabs. A user willing to refresh the cached lease information must
 * click the refresh button on the selected tab. The lease information
 * is used to present whether the reserved IP address or prefix is in
 * use and whether the lease is assigned to a client which does not
 * have a reservation for it (conflict).
 */
@Component({
    selector: 'app-host-tab',
    templateUrl: './host-tab.component.html',
    styleUrls: ['./host-tab.component.sass'],
})
export class HostTabComponent {
    /**
     * An event emitter notifying a parent that user has clicked the
     * Edit button to modify the host reservation.
     */
    @Output() hostEditBegin = new EventEmitter<any>()

    /**
     * An event emitter notifying a parent that user has clicked the
     * Delete button to delete the host reservation.
     */
    @Output() hostDelete = new EventEmitter<any>()

    Usage = HostReservationUsage

    /**
     * Structure containing host information currently displayed.
     */
    currentHost: Host

    /**
     * Local hosts of the @currentHost grouped by differential daemons.
     * If all the daemons have the same set of options, the nested array will
     * contain a single element.
     */
    localHostsGroups: {
        dhcpOptions: LocalHost[][]
        bootFields: LocalHost[][]
        clientClasses: LocalHost[][]
        appID: LocalHost[][]
        hostname: LocalHost[][]
        ipReservations: LocalHost[][]
    } = { bootFields: [], dhcpOptions: [], clientClasses: [], appID: [], hostname: [], ipReservations: [] }

    /**
     * Indicates if the boot fields panel should be displayed.
     */
    displayBootFields: boolean

    /**
     * A map caching leases for various hosts.
     *
     * Thanks to this caching, it is possible to switch between the
     * host tabs and avoid fetching lease information for a current
     * host whenever a different tab is selected.
     */
    private _leasesForHosts = new Map<number, Map<string, LeaseInfo>>()

    /**
     * Leases fetched for currently selected tab (host).
     */
    currentLeases: Map<string, LeaseInfo>

    /**
     * A map of booleans indicating for which hosts leases search
     * is in progress.
     */
    private _leasesSearchStatus = new Map<number, boolean>()

    /**
     * List of Kea apps which returned an error during leases search.
     */
    erredApps: LeasesSearchErredApp[] = []

    hostDeleted = false

    /**
     * Component constructor.
     *
     * @param msgService service displaying error messages upon a communication
     *                   error with the server.
     * @param dhcpApi service used to communicate with the server over REST API.
     */
    constructor(
        private dhcpApi: DHCPService,
        private confirmService: ConfirmationService,
        private msgService: MessageService
    ) {}

    /**
     * Returns information about currently selected host.
     */
    @Input()
    get host() {
        return this.currentHost
    }

    /**
     * Sets a host to be displayed by the component.
     *
     * This setter is called when the user selects one of the host tabs.
     * If leases for this host have not been already gathered, the function
     * queries the server for the leases corresponding to the host. Otherwise,
     * cached lease information is displayed.
     *
     * @param host host information.
     */
    set host(host) {
        // Make the new host current.
        this.currentHost = host
        this.localHostsGroups = {
            bootFields: [],
            dhcpOptions: [],
            clientClasses: [],
            appID: [],
            hostname: [],
            ipReservations: [],
        }
        // The host is null if the tab with a list of hosts is selected.
        if (!this.currentHost) {
            return
        }
        // Check if we already have lease information for this host.
        const leasesForHost = this._leasesForHosts.get(host.id)
        if (leasesForHost) {
            // We have the lease information already. Let's use it.
            this.currentLeases = leasesForHost
        } else {
            // We don't have lease information for this host. Need to
            // fetch it from Kea servers via Stork server.
            this._fetchLeases(host.id)
        }
        this.displayBootFields = !!this.currentHost.localHosts?.some(
            (lh) => lh.nextServer || lh.serverHostname || lh.bootFileName
        )

        // Group local hosts by the app ID.
        const localHostsByAppID = Object.values(
            (host.localHosts ?? [])
                // Group by app ID.
                .reduce<Record<number, LocalHost[]>>((acc, lh) => {
                    if (!acc[lh.appId]) {
                        // Create an array for the app ID if it doesn't exist yet.
                        acc[lh.appId] = []
                    }
                    // Add the local host to the array.
                    acc[lh.appId].push(lh)
                    // Return the accumulator.
                    return acc
                }, {})
        )

        // Group local hosts by the boot fields equality.
        const localHostsByBootFields: LocalHost[][] = []
        if (hasDifferentLocalHostBootFields(this.host.localHosts)) {
            for (let localHosts of localHostsByAppID) {
                if (hasDifferentLocalHostBootFields(localHosts)) {
                    localHostsByBootFields.push(...localHosts.map((lh) => [lh]))
                } else {
                    localHostsByBootFields.push(localHosts)
                }
            }
        } else {
            localHostsByBootFields.push(this.host.localHosts)
        }

        // Group local hosts by the DHCP options equality.
        const localHostsByDhcpOptions: LocalHost[][] = []
        if (hasDifferentLocalHostOptions(this.host.localHosts)) {
            for (let localHosts of localHostsByAppID) {
                if (hasDifferentLocalHostOptions(localHosts)) {
                    localHostsByDhcpOptions.push(...localHosts.map((lh) => [lh]))
                } else {
                    localHostsByDhcpOptions.push(localHosts)
                }
            }
        } else {
            localHostsByDhcpOptions.push(this.host.localHosts)
        }

        // Group local hosts by the client classes equality.
        const localHostsByClientClasses: LocalHost[][] = []
        if (hasDifferentLocalHostClientClasses(this.host.localHosts)) {
            for (let localHosts of localHostsByAppID) {
                if (hasDifferentLocalHostClientClasses(localHosts)) {
                    localHostsByClientClasses.push(...localHosts.map((lh) => [lh]))
                } else {
                    localHostsByClientClasses.push(localHosts)
                }
            }
        } else {
            localHostsByClientClasses.push(this.host.localHosts)
        }

        // Group local hosts by the IP reservations equality.
        const localHostsByIPReservations: LocalHost[][] = []
        if (hasDifferentLocalHostIPReservations(this.host.localHosts)) {
            for (let localHosts of localHostsByAppID) {
                if (hasDifferentLocalHostIPReservations(localHosts)) {
                    localHostsByIPReservations.push(...localHosts.map((lh) => [lh]))
                } else {
                    localHostsByIPReservations.push(localHosts)
                }
            }
        } else {
            localHostsByIPReservations.push(this.host.localHosts)
        }

        // Group local hosts by the hostname equality.
        const localHostsByHostname: LocalHost[][] = []
        if (hasDifferentLocalHostHostname(this.host.localHosts)) {
            for (let localHosts of localHostsByAppID) {
                if (hasDifferentLocalHostHostname(localHosts)) {
                    localHostsByHostname.push(...localHosts.map((lh) => [lh]))
                } else {
                    localHostsByHostname.push(localHosts)
                }
            }
        } else {
            localHostsByHostname.push(this.host.localHosts)
        }

        this.localHostsGroups = {
            bootFields: localHostsByBootFields,
            dhcpOptions: localHostsByDhcpOptions,
            clientClasses: localHostsByClientClasses,
            appID: localHostsByAppID,
            hostname: localHostsByHostname,
            ipReservations: localHostsByIPReservations,
        }
    }

    /**
     * Returns boolean value indicating if the leases are being searched
     * for the currently displayed host.
     *
     * @returns true if leases are being searched for the currently displayed
     *          host, false otherwise.
     */
    get leasesSearchInProgress() {
        return !!this._leasesSearchStatus.get(this.host.id)
    }

    /**
     * Fetches leases for the given host from the Stork server.
     *
     * @param hostId host identifier.
     */
    private _fetchLeases(hostId) {
        // Do not search again if the search is already in progress.
        if (this._leasesSearchStatus.get(hostId)) {
            return
        }
        // Indicate that the search is already in progress for that host.
        this._leasesSearchStatus.set(hostId, true)
        this.erredApps = []
        this.dhcpApi.getLeases(undefined, hostId).subscribe(
            (data) => {
                // Finished searching the leases.
                this._leasesSearchStatus.set(hostId, false)
                // Collect the lease information and store it in the cache.
                const leases = new Map<string, LeaseInfo>()
                if (data.items) {
                    for (const lease of data.items) {
                        this._mergeLease(leases, data.conflicts, lease)
                    }
                }
                this.erredApps = data.erredApps
                this._leasesForHosts.set(hostId, leases)
                this.currentLeases = leases
            },
            (err) => {
                // Finished searching the leases.
                this._leasesSearchStatus.set(hostId, false)
                const msg = getErrorMessage(err)
                this.msgService.add({
                    severity: 'error',
                    summary: 'Error searching leases for the host',
                    detail: 'Error searching by host ID ' + hostId + ': ' + msg,
                    life: 10000,
                })
            }
        )
    }

    /**
     * Merges lease information into the cache.
     *
     * The cache contains leases gathered for various IP addresses and/or
     * delegated prefixes for a host. There may be multiple leases for a
     * single reserved IP address or delegated prefix.
     *
     * @param leases leases cache for the host.
     * @param conflicts array of conflicting lease ids.
     * @param newLease a lease to be merged to the cache.
     */
    private _mergeLease(leases: Map<string, LeaseInfo>, conflicts, newLease: Lease) {
        // Check if the lease is in conflict with the host reservation.
        if (conflicts) {
            for (const conflictId of conflicts) {
                if (newLease.id === conflictId) {
                    newLease['conflict'] = true
                }
            }
        }
        let reservedLeaseInfo = leases.get(newLease.ipAddress)
        if (reservedLeaseInfo) {
            // There is already some lease cached for this IP address.
            reservedLeaseInfo.leases.push(newLease)
        } else {
            // There is no lease cached for this IP address yet.
            reservedLeaseInfo = { leases: [newLease] }
            let leaseKey = newLease.ipAddress
            if (newLease.prefixLength && newLease.prefixLength !== 0 && newLease.prefixLength !== 128) {
                leaseKey += '/' + newLease.prefixLength
            }
            leases.set(leaseKey, reservedLeaseInfo)
        }
        let newUsage = this.Usage.Used
        if (newLease['conflict']) {
            newUsage = this.Usage.Conflicted
        } else {
            // Depending on the lease state, we should adjust the usage information
            // displayed next to the IP reservation.
            switch (newLease['state']) {
                case 0:
                    newUsage = this.Usage.Used
                    break
                case 1:
                    newUsage = this.Usage.Declined
                    break
                case 2:
                    newUsage = this.Usage.Expired
                    break
                default:
                    break
            }
        }
        // If the usage hasn't been set yet, or the new usage overrides the current
        // usage, update the usage information. The usage values in the enum are ordered
        // by importance: conflicted, declined, expired, used. Even if one of the leases
        // is used, but the other one is conflicted, we mark the usage as conflicted.
        if (!reservedLeaseInfo['usage'] || newUsage < reservedLeaseInfo['usage']) {
            reservedLeaseInfo['usage'] = newUsage
            reservedLeaseInfo['culprit'] = newLease
        }
    }

    /**
     * Returns lease usage text for an enum value.
     *
     * @param usage usage enum indicating if the lease is used, declined, expired
     *        or conflicted.
     * @returns usage text displayed next to the IP address or delegated prefix.
     */
    getLeaseUsageText(usage) {
        switch (usage) {
            case this.Usage.Used:
                return 'in use'
            case this.Usage.Declined:
                return 'declined'
            case this.Usage.Expired:
                return 'expired'
            case this.Usage.Conflicted:
                return 'in conflict'
            default:
                break
        }
        return 'unused'
    }

    /**
     * Returns information displayed in the expanded row for an IP reservation.
     *
     * When user clicks on a chevron icon next to the reserved IP address, a row
     * is expanded displaying the lease state summary for the given IP address.
     * The summary is generated by this function.
     *
     * @param leaseInfo lease information for the specified IP address or
     *        delegated prefix.
     * @returns lease state summary text.
     */
    getLeaseSummary(leaseInfo: LeaseInfo) {
        let summary = 'Lease information unavailable.'
        if (!leaseInfo.culprit) {
            return summary
        }
        const m = leaseInfo.leases.length > 1
        switch (leaseInfo.usage) {
            case this.Usage.Used:
                // All leases are assigned to the client who has a reservation
                // for it. Simply say how many leases are assigned and when they
                // expire.
                summary =
                    'Found ' +
                    leaseInfo.leases.length +
                    ' assigned lease' +
                    (m ? 's' : '') +
                    ' with the' +
                    (m ? ' latest' : '') +
                    ' expiration time at ' +
                    epochToLocal(this._getLatestLeaseExpirationTime(leaseInfo)) +
                    '.'
                return summary
            case this.Usage.Expired:
                const expirationTime = leaseInfo.culprit.cltt + leaseInfo.culprit.validLifetime
                const expirationDuration = durationToString(new Date().getTime() / 1000 - expirationTime, true)
                summary =
                    'Found ' +
                    leaseInfo.leases.length +
                    ' lease' +
                    (m ? 's' : '') +
                    ' for this reservation' +
                    (m ? '. This includes a lease' : '') +
                    ' that expired at ' +
                    epochToLocal(expirationTime)

                if (expirationDuration) {
                    summary += ' ' + '(' + expirationDuration + ' ago).'
                }
                return summary
            case this.Usage.Declined:
                // Found leases for our client but at least one of them is declined.
                summary =
                    'Found ' +
                    leaseInfo.leases.length +
                    ' lease' +
                    (m ? 's' : '') +
                    ' for this reservation' +
                    (m ? '. This includes a declined lease with' : ' which is declined and has an') +
                    ' expiration time at ' +
                    epochToLocal(leaseInfo.culprit.cltt + leaseInfo.culprit.validLifetime) +
                    '.'
                return summary
            case this.Usage.Conflicted:
                // Found lease assignments to other clients than the one which
                // has a reservation.
                let identifier = ''
                if (leaseInfo.culprit.hwAddress) {
                    identifier = 'MAC address=' + leaseInfo.culprit.hwAddress
                } else if (leaseInfo.culprit.duid) {
                    identifier = 'DUID=' + leaseInfo.culprit.duid
                } else if (leaseInfo.culprit.clientId) {
                    identifier = 'client-id=' + leaseInfo.culprit.clientId
                }
                summary =
                    'Found a lease with an expiration time at ' +
                    epochToLocal(leaseInfo.culprit.cltt + leaseInfo.culprit.validLifetime) +
                    ' assigned to the client with ' +
                    identifier +
                    ', for which it was not reserved.'
                return summary
            default:
                break
        }
        return summary
    }

    /**
     * Returns the latest expiration time from the leases held in the
     * cache for the particular IP address or delegated prefix.
     *
     * @param leaseInfo lease information for a reserved IP address or
     *        delegated prefix.
     *
     * @returns expiration time relative to the epoch or 0 if no lease
     *          is present.
     */
    private _getLatestLeaseExpirationTime(leaseInfo) {
        let latestExpirationTime = 0
        for (const lease of leaseInfo.leases) {
            const expirationTime = lease.cltt + lease.validLifetime
            if (expirationTime > latestExpirationTime) {
                latestExpirationTime = expirationTime
            }
        }
        return latestExpirationTime
    }

    /**
     * Starts leases refresh for a current host.
     */
    refreshLeases() {
        this._fetchLeases(this.host.id)
    }

    /**
     * Event handler called when user begins host editing.
     *
     * It emits an event to the parent component to notify that host is
     * is now edited.
     */
    onHostEditBegin(): void {
        this.hostEditBegin.emit(this.host)
    }

    /*
     * Displays a dialog to confirm host deletion.
     */
    confirmDeleteHost() {
        this.confirmService.confirm({
            message: 'Are you sure that you want to permanently delete this host reservation?',
            header: 'Delete Host',
            icon: 'pi pi-exclamation-triangle',
            accept: () => {
                this.deleteHost()
            },
        })
    }

    /**
     * Sends a request to the server to delete the host reservation.
     */
    deleteHost() {
        // Disable the button for deleting the host to prevent pressing the
        // button multiple times and sending multiple requests.
        this.hostDeleted = true
        this.dhcpApi
            .deleteHost(this.host.id)
            .toPromise()
            .then(() => {
                // Re-enable the delete button.
                this.hostDeleted = false
                this.msgService.add({
                    severity: 'success',
                    summary: 'Host reservation successfully deleted',
                })
                // Notify the parent that the host was deleted and the tab can be closed.
                this.hostDelete.emit(this.host)
            })
            .catch((err) => {
                // Re-enable the delete button.
                this.hostDeleted = false
                // Issues with deleting the host.
                const msg = getErrorMessage(err)
                this.msgService.add({
                    severity: 'error',
                    summary: 'Cannot delete the host',
                    detail: 'Failed to delete the host: ' + msg,
                    life: 10000,
                })
            })
    }

    /**
     * Checks if there is at least one local host from the host database.
     */
    hasAnyLocalHostFromDatabase() {
        return !!this.host.localHosts?.some((lh) => lh.dataSource === 'api')
    }

    /**
     * Checks if there is at least one local host from the configuration.
     */
    hasAnyLocalHostFromConfig() {
        return !!this.host.localHosts?.some((lh) => lh.dataSource === 'config')
    }

    /**
     * Checks if provided DHCP servers owning the reservation have different data.
     *
     * @returns true, if provided DHCP servers have different data.
     */
    daemonsHaveDifferentHostData(localHosts: LocalHost[]): boolean {
        return hasDifferentLocalHostData(localHosts)
    }
}
