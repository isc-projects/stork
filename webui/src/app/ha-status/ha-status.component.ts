import { Component, Input, OnDestroy, OnInit } from '@angular/core'
import { interval, lastValueFrom, Subscription } from 'rxjs'
import { ServicesService } from '../backend/api/api'
import { KeaHAServerStatus, ServiceStatus } from '../backend'
import { MessageService } from 'primeng/api'
import { datetimeToLocal, getErrorMessage } from '../utils'

/**
 * An interface representing HA table cell data.
 *
 * The HA relationships are presented in a table. This interface
 * describes the contents of each data cell in that table.
 */
interface RelationshipNodeCell {
    iconType?: string
    appId?: number
    appName?: string
    value?: string | number
    progress?: number
}

/**
 * An interface representing a table row with relationship name.
 *
 * The HA relationships are presented in a table. This interface
 * describes the contents of a row with the relationship name.
 */
interface RelationshipTableRow {
    name: string
    styleClass?: string
    cells: RelationshipNodeCell[]
}

/**
 * An interface representing relationship data row.
 */
interface RelationshipTableDataRow {
    relationship: RelationshipTableRow
    title: string
    styleClass?: string
    cells?: RelationshipNodeCell[]
}

/**
 * A callback type used internally in the component.
 *
 * I sets the HA table cell contents.
 */
type RelationshipNodeCellFunction = (serverStatus: KeaHAServerStatus) => RelationshipNodeCell

/**
 * Component presenting live status of High Availability in Kea.
 *
 * The presented status is periodically refreshed.
 */
@Component({
    selector: 'app-ha-status',
    templateUrl: './ha-status.component.html',
    styleUrls: ['./ha-status.component.sass'],
})
export class HaStatusComponent implements OnInit, OnDestroy {
    /**
     * A periodic timer subscription counting down until next data refresh.
     */
    private timerSubscription = new Subscription()

    /**
     * A data refresh timer interval in seconds.
     */
    private readonly refreshInterval = 10

    /**
     * ID of the Kea application for which the High Availability state
     * is presented.
     *
     * This should be id of one of the active Kea servers. The HA status
     * of this server and its partner (if it has any) is fetched and presented.
     */
    @Input() appId: number

    /**
     * A name of the Kea daemon for which the HA status is displayed.
     */
    @Input() daemonName: string

    /**
     * Holds displayable HA status.
     *
     * The received HA services status is converted to a tree table form and
     * displayed. This status is updated every time a new request is made to
     * the Stork server.
     */
    status: Array<RelationshipTableDataRow> = []

    /**
     * Indicates if the status table should contain a column for partner status.
     *
     * In the passive-backup configuration there is only one active server, so
     * the partner's state is not presented. This flag is set to true when the
     * partner status should be presented.
     */
    hasPartnerColumn = false

    /**
     * Countdown presenting the time remaining to a next data refresh.
     */
    refreshCountdown = this.refreshInterval

    /**
     * Indicates if the data were loaded at least once.
     */
    loadedOnce: boolean = false

    constructor(
        private servicesApi: ServicesService,
        private messageService: MessageService
    ) {}

    /**
     * A lifecycle hook invoked when the component is initialized.
     *
     * It fetches the current status of the HA services and sets a periodic
     * refresh timer to refresh this data.
     */
    ngOnInit() {
        this.refreshStatus()
    }

    /**
     * A lifecycle hook invoked when the component is destroyed.
     *
     * It removes the periodic timer subscription.
     */
    ngOnDestroy(): void {
        this.timerSubscription.unsubscribe()
    }

    /**
     * Checks if the component has fetched the HA status for the current daemon.
     *
     * @returns true if the status has been fetched and is available for display.
     */
    get hasStatus(): boolean {
        return this.status?.length > 0
    }

    /**
     * Convenience function returning received status of the local server.
     *
     * @param relationship a relationship contains the status of the primary
     * and/or secondary/standby server.
     * @returns Local server status in the relationship or null if there is
     * no local server status.
     */
    getLocalServerStatus(relationship: ServiceStatus): KeaHAServerStatus | null {
        return relationship.status?.haServers?.primaryServer.appId === this.appId
            ? relationship.status?.haServers.primaryServer
            : relationship.status?.haServers.secondaryServer
    }

    /**
     * Convenience function returning received status of the remote server.
     *
     * @param relationship a relationship contains the status of the primary
     * and/or secondary/standby server.
     * @returns Remote server status in the relationship or null if there is
     * no remote server status.
     */
    getRemoteServerStatus(relationship: ServiceStatus): KeaHAServerStatus | null {
        return relationship.status?.haServers?.primaryServer.appId !== this.appId
            ? relationship.status?.haServers.primaryServer
            : relationship.status?.haServers.secondaryServer
    }

    /**
     * Convenience function generating a single HA table data row.
     *
     * @param servers statuses of the servers in the relationship.
     * @param relationship relationship associated with this data row.
     * @param title row title (i.e., the text in the first column).
     * @param fn callback function generating each subsequent cell.
     * @returns A table row holding generated data.
     */
    private makeTableDataRow(
        servers: KeaHAServerStatus[],
        relationship: RelationshipTableRow,
        title: string,
        fn: RelationshipNodeCellFunction
    ): RelationshipTableDataRow {
        return {
            relationship: relationship,
            title: title,
            cells: servers.map((s) => {
                return fn(s)
            }),
        }
    }

    /**
     * Gets the HA status of the Kea server and its partner.
     *
     * This function is invoked periodically to refresh the status.
     */
    private refreshStatus() {
        // Get the services status for the app from the server.
        lastValueFrom(this.servicesApi.getAppServicesStatus(this.appId))
            .then((data) => {
                if (data.items) {
                    let status: RelationshipTableDataRow[] = []
                    data.items
                        // Exclude the non-matching daemons and the services that have no
                        // local server state. It is ok if they don't have the remote
                        // server state because it could be the passive-backup config.
                        .filter((relationship) => {
                            return (
                                relationship.status?.daemon === this.daemonName &&
                                this.getLocalServerStatus(relationship)
                            )
                        })
                        .forEach((relationship, index) => {
                            // Local server status must exist.
                            const servers = [this.getLocalServerStatus(relationship)]
                            // Remote server status is optional.
                            let remoteStatus = this.getRemoteServerStatus(relationship)
                            if (remoteStatus) {
                                servers.push(remoteStatus)
                            }
                            // Create a row holding a relationship name and server data. Subsequent rows
                            // are associated with this structure which groups the data pertaining to the
                            // relationship together.
                            let relationshipRow: RelationshipTableRow = {
                                styleClass: 'relationship-pane',
                                name: `Relationship #${index + 1}`,
                                cells: servers.map<RelationshipNodeCell>((s, index) => {
                                    let cell: RelationshipNodeCell = {
                                        value: s.role,
                                    }
                                    // It only makes sense to add an app link if it is a remote
                                    // server. The local server is currently displayed.
                                    if (index > 0) {
                                        cell.appId = s.appId
                                        cell.appName = `Kea@${s.controlAddress}`
                                    }
                                    return cell
                                }),
                            }
                            // Create an array of data rows associated with the current relationship.
                            let relationshipDataRows: RelationshipTableDataRow[] = []

                            // Begin with the rows that are always present regardless
                            // of the HA mode.
                            relationshipDataRows = [
                                this.makeTableDataRow(servers, relationshipRow, 'Control status', (s) => {
                                    return {
                                        iconType: s.inTouch ? 'ok' : 'error',
                                        value: this.formatControlStatus(s),
                                    }
                                }),
                                this.makeTableDataRow(servers, relationshipRow, 'State', (s) => {
                                    return {
                                        iconType: this.getStateIconType(s),
                                        value: this.formatState(s),
                                    }
                                }),
                                this.makeTableDataRow(servers, relationshipRow, 'Scopes', (s) => {
                                    return { value: this.formatScopes(s) }
                                }),
                                this.makeTableDataRow(servers, relationshipRow, 'Status time', (s) => {
                                    return { value: datetimeToLocal(s.statusTime) }
                                }),
                                this.makeTableDataRow(servers, relationshipRow, 'Status age', (s) => {
                                    return { value: this.formatAge(s.age) }
                                }),
                            ]

                            // Other rows are only displayed if this is a hot-standby or
                            // a load-balancing configuration.
                            if (servers.length > 1) {
                                // The heartbeat status goes first before anything else.
                                relationshipDataRows.unshift(
                                    this.makeTableDataRow(servers, relationshipRow, 'Heartbeat status', (s) => {
                                        return {
                                            iconType: !s.commInterrupted ? 'ok' : 'error',
                                            value: this.formatHeartbeatStatus(s),
                                        }
                                    })
                                )
                                // Other rows are appended at the end.
                                relationshipDataRows = relationshipDataRows.concat([
                                    this.makeTableDataRow(servers, relationshipRow, 'Last in partner-down', (s) => {
                                        return { value: datetimeToLocal(s.failoverTime) || 'never' }
                                    }),
                                    this.makeTableDataRow(servers, relationshipRow, 'Unacked clients', (s) => {
                                        return { value: this.formatUnackedClients(s) }
                                    }),
                                    this.makeTableDataRow(servers, relationshipRow, 'Connecting clients', (s) => {
                                        return { value: this.formatFailoverNumber(s, s.connectingClients) }
                                    }),
                                    this.makeTableDataRow(servers, relationshipRow, 'Analyzed packets', (s) => {
                                        return { value: this.formatFailoverNumber(s, s.analyzedPackets) }
                                    }),
                                    this.makeTableDataRow(servers, relationshipRow, 'Failover progress', (s) => {
                                        let progress = this.calculateServerFailoverProgress(s)
                                        return progress >= 0 ? { progress: progress } : { value: 'n/a' }
                                    }),
                                    {
                                        relationship: relationshipRow,
                                        title: 'Summary',
                                        cells: [
                                            { value: this.createSummary(servers[0], servers[1]) },
                                            { value: this.createSummary(servers[1], servers[0]) },
                                        ],
                                    },
                                ])
                            }
                            // Check if we need to show any icons in the relationship top-level row.
                            relationshipRow.cells.forEach((r, index) => {
                                r.iconType = relationshipDataRows.find((ch) => {
                                    return ch.cells?.[index].iconType && ch.cells?.[index].iconType !== 'ok'
                                })?.cells[index].iconType
                            })
                            status = status.concat(relationshipDataRows)
                        })
                    // Check if we need to add the partner's column. This is the case in the
                    // hot-standby and load-balancing case.
                    this.hasPartnerColumn = status.some((r) => r.cells?.length > 1)
                    this.status = status
                }
                this.loadedOnce = true
            })
            .catch((err) => {
                this.messageService.add({
                    severity: 'error',
                    summary: `Failed to fetch the HA status for Kea application ID: ${this.appId}`,
                    detail: getErrorMessage(err),
                })
                this.status = []
            })
            .finally(() => {
                this.setCountdownTimer()
            })
    }

    /**
     * Sets the countdown timer to next data refresh.
     */
    setCountdownTimer(): void {
        // Start the countdown to the next refresh with the 1s interval, so the user
        // can observe the countdown.
        this.refreshCountdown = this.refreshInterval
        this.timerSubscription = interval(1000).subscribe(() => {
            this.refreshCountdown -= 1
            if (this.refreshCountdown <= 0) {
                this.timerSubscription.unsubscribe()
                this.refreshStatus()
            }
        })
    }

    /**
     * Returns formatted heartbeat status for the server.
     *
     * This information is only available if the extended status format
     * is supported (Kea 1.7.8 and later).
     *
     * @returns 'unknown' if extended format is not supported for this server,
     *          'ok' if the server is responding to the heartbeats,
     *          'failed' otherwise.
     */
    formatHeartbeatStatus(serverStatus: KeaHAServerStatus): string {
        if (serverStatus.commInterrupted < 0) {
            return 'unknown'
        } else if (serverStatus.commInterrupted > 0) {
            return 'failed'
        }
        return 'ok'
    }

    /**
     * Returns formatted status of the control channel.
     *
     * Depending on the value of the boolean parameter specified, this function
     * returns the word "online" or "offline" to indicate the status of the
     * communication with one of the servers.
     *
     * @returns A descriptive information whether the server seems to be
     *          online or offline. It may be also unknown if the returned
     *          status does not contain the flag.
     */
    formatControlStatus(serverStatus: KeaHAServerStatus): string {
        if (serverStatus.inTouch == null) {
            return 'unknown'
        }
        if (serverStatus.inTouch) {
            return 'online'
        }
        return 'offline'
    }

    /**
     * Returns formatted HA state of the server.
     *
     * This function checks if the HA status of the server is initialized
     * and returns this status if it is initialized. Otherwise it returns
     * "fetching..." to indicate that the status will appear once it is
     * fetched (pending).
     *
     * @returns A state or the text "fetching..." if it is not available yet.
     */
    formatState(serverStatus: KeaHAServerStatus): string {
        if (!serverStatus.state) {
            return 'fetching...'
        }
        return serverStatus.state
    }

    /**
     * Returns a comma-separated list of HA scopes served by the server.
     *
     * @returns string containing a comma-separated list of scopes, the word
     *          none, or none (standby server).
     */
    formatScopes(serverStatus: KeaHAServerStatus): string {
        let scopes: string
        if (serverStatus.scopes) {
            scopes = serverStatus.scopes.join(', ')
        }
        if (!scopes && serverStatus.state === 'hot-standby' && serverStatus.role === 'standby') {
            scopes = 'none (standby server)'
        }
        return scopes || 'none'
    }

    /**
     * Returns formatted value of an age.
     *
     * The age indicates how long ago the status of one of the servers has
     * been fetched. It is expressed in seconds. This function displays the
     * age in seconds for the age below 1 minute. It displays the age in
     * minutes otherwise. The negative age value means that the age is not
     * yet determined in which case 'n/a' is displayed.
     *
     * @param age in seconds.
     * @returns String containing formatted age.
     */
    formatAge(age: number | null): string {
        if (age && age < 0) {
            return 'n/a'
        }
        if (!age) {
            return 'just now'
        }
        if (age < 60) {
            return age + ' seconds ago'
        }
        return Math.round(age / 60) + ' minutes ago'
    }

    /**
     * Returns formatted number of unacked clients for the server which fails
     * to respond to the heartbeats.
     *
     * The partner server starts to monitor the DHCP traffic directed to the
     * partner when the heartbeat has been failing with this server longer than
     * the configured period of time. The partner monitors the traffic by checking
     * the value of the 'secs' field (DHCPv4) or 'elapsed time' option (DHCPv6).
     * If these values exceed the configured threshold the client sending the
     * packet is considered unacked. If the number of unacked clients exceeds
     * the configured threshold for the number of unacked clients, the surviving
     * server enters the partner-down state.
     *
     * The string returned by this function includes the number of unacked
     * clients and the configured threshold for this number, e.g. 3 of 5,
     * which indicates that 3 out of 5 clients have been unacked so far.
     *
     * @return A string containing the number of unacked clients by the server
     *         and the maximum number of unacked clients before the server
     *         transitions to the partner-down state.
     */
    formatUnackedClients(serverStatus: KeaHAServerStatus): string {
        let s = 'n/a'
        let unacked = 0
        let all = 0
        // Monitor unacked clients only if we're in the communication interrupted
        // state, i.e. heartbeat communication has been failing for a certain
        // period of time.
        if (serverStatus.commInterrupted > 0) {
            if (serverStatus.unackedClients != null) {
                unacked = serverStatus.unackedClients
            }
            all = unacked
            if (serverStatus.unackedClientsLeft != null) {
                all += serverStatus.unackedClientsLeft
            }
        }
        // If both unacked and unacked left values are 0 there is nothing
        // to print. It looks that we don't monitor unacked clients for
        // this server.
        if (all > 0) {
            s = unacked + ' of ' + (all + 1)
        }
        return s
    }

    /**
     * Returns a formatted failover related number.
     *
     * This function is used to format the number of connecting clients or
     * analyzed packets. If the communication is not interrupted the returned
     * value is n/a. Otherwise, the number is returned.
     *
     * @returns A given value or n/a string.
     */
    formatFailoverNumber(serverStatus: KeaHAServerStatus, n: number): string | number {
        if (!serverStatus.unackedClients && !serverStatus.unackedClientsLeft) {
            return 'n/a'
        }
        return Math.max(0, n)
    }

    /**
     * Returns the value of the failover progress bar for the selected server.
     *
     * The failover progress is calculated as the number of unacked clients
     * divided by the number of max-unacked-clients+1 for that server and
     * expressed in percentage.
     *
     * @param server Data structure holding the status of the local or remote
     *               server.
     * @return Progress value or -1 if there is no failover in progress.
     */
    calculateServerFailoverProgress(server: KeaHAServerStatus): number {
        let unacked = server.unackedClients || 0
        let all = unacked
        if (server.unackedClientsLeft > 0) {
            all += server.unackedClientsLeft
        }
        if (all === 0) {
            return -1
        }
        return Math.floor((100 * unacked) / (all + 1))
    }

    /**
     * Returns a summary text for the server.
     *
     * @param serverStatus status of the server for which the summary is returned
     * @param otherServerStatus status of the partner.
     * @returns Summary text displayed in table for a server.
     */
    createSummary(serverStatus: KeaHAServerStatus, otherServerStatus: KeaHAServerStatus): string {
        if (this.calculateServerFailoverProgress(serverStatus) >= 0) {
            return 'Server has started the failover procedure.'
        }
        const scopes = serverStatus?.scopes || []
        if (scopes.length === 0) {
            return 'Server is responding to no DHCP traffic.'
        }
        const otherScopes = otherServerStatus?.scopes || []
        if (otherScopes.length === 0) {
            return 'Server is responding to all DHCP traffic.'
        }
        return 'Server is responding to DHCP traffic.'
    }

    /**
     * Returns an icon type suitable for a server state.
     *
     * Depending on the value returned by this function different icons are
     * displayed next to the server state.
     *
     * @returns ok when server status has been fetched and indicates normal
     *          operation, e.g. load balancing; warn when server status
     *          has been fetched and indicates other state; pending if the
     *          server status hasn't been determined yet.
     */
    getStateIconType(serverStatus: KeaHAServerStatus): 'ok' | 'warn' | 'pending' {
        if (!serverStatus.state) {
            return 'pending'
        }
        if (
            serverStatus.state === 'load-balancing' ||
            serverStatus.state === 'hot-standby' ||
            serverStatus.state === 'passive-backup'
        ) {
            return 'ok'
        }
        return 'warn'
    }
}
