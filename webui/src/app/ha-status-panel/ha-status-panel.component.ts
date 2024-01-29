import { Component, Input } from '@angular/core'

/**
 * Enumeration indicating the server's HA state.
 *
 * It is used to distinguish between the situations when the server is
 * performing normal operation (ok), when the server is in a state which
 * may require administrator's attention (not ok), or when the state hasn't
 * been fetched yet (pending).
 */
enum HAStateKind {
    Ok,
    NotOk,
    Pending,
}

/**
 * Component presenting live status of High Availability in Kea for
 * a single server.
 *
 * This component is embedded within the HaStatusComponent to present
 * the status of the individual servers.
 */
@Component({
    selector: 'app-ha-status-panel',
    templateUrl: './ha-status-panel.component.html',
    styleUrls: ['./ha-status-panel.component.sass'],
})
export class HaStatusPanelComponent {
    public StateKind = HAStateKind

    /**
     * Holds status fetched for this server from the backend.
     */
    private _serverStatus

    /**
     * Holds the style class of the panel view.
     *
     * The code may dynamically switch to a different class depending
     * on the server status. Switching to a different class causes the
     * panel to change its color to highlight warnings and errors.
     */
    public statusPanelClass = 'green-colored-panel'

    /**
     * Panel title set by the parent component.
     */
    @Input()
    public panelTitle: string

    /**
     * Server name set by the parent component.
     */
    @Input()
    public serverName: string

    /**
     * Indicates if monitoring one or two active servers.
     *
     * Two active servers are monitored in load-balancing and hot-standby
     * modes. A single server is monitored in the passive-backup mode. The
     * panel view is adjusted if this is single server case, e.g. the
     * last failover time is not presented.
     */
    @Input()
    public singleActiveServer = false

    /**
     * Indicates if the link to the app should be presented in the panel title.
     *
     * The link is presented for the remote servers. It is not presented for
     * the local servers. The parent component sets this flag.
     */
    @Input()
    public showServerLink = false

    /**
     * No-op constructor.
     */
    constructor() {}

    /**
     * Sets new status information for the server.
     *
     * The panel colors are refreshed according to the new status
     * information to highlight errors, warnings or normal operation.
     *
     * @param serverStatus New server status.
     */
    @Input()
    set serverStatus(serverStatus) {
        this._serverStatus = serverStatus
        this.refreshPanelColors()
    }

    /**
     * Returns status information fetched for the server.
     */
    get serverStatus() {
        return this._serverStatus
    }

    /**
     * Checks if the parent component has fetched the HA status for the server.
     *
     * @returns true if the status has been fetched and is available for display.
     */
    private hasStatus(): boolean {
        return this._serverStatus
    }

    /**
     * Returns the help tip describing online/offline control status.
     */
    controlStatusHelptip(): string {
        if (!this.hasStatus()) {
            return ''
        }
        if (this.serverStatus.inTouch) {
            return 'Server responds to commands over the control channel.'
        }
        return 'Server does not respond to commands over the control channel. It may be down!'
    }

    /**
     * Returns help tip describing various HA states.
     */
    haStateHelptip(): string {
        if (!this.hasStatus()) {
            return ''
        }
        switch (this.serverStatus.state) {
            case 'load-balancing':
            case 'hot-standby':
                return 'Normal operation.'
            case 'partner-down':
                return (
                    'This server is now responding to all DHCP queries because it detected ' +
                    'that its partner server is not functional!'
                )
            case 'passive-backup':
                return (
                    'The server has no active partner, unlike in load-balancing or hot-standby ' +
                    'mode. This server may be configured to send lease updates to the ' +
                    'backup servers, but there is no automatic failover triggered in case ' +
                    'of failure.'
                )
            case 'waiting':
                return 'This server is booting up and will try to synchronize its lease database.'
            case 'syncing':
                return 'This server is synchronizing its database after a failure.'
            case 'ready':
                return 'This server synchronized its lease database and will start normal operation shortly.'
            case 'terminated':
                return 'This server no longer participates in the HA setup because of too-high clock skew.'
            case 'maintained':
                return 'This server is under maintenance.'
            case 'partner-maintained':
                return 'This server is responding to all DHCP queries while its partner is in maintenance.'
            case 'unavailable':
                return 'Communication with the server failed. It may have crashed or have been shut down.'
            default:
                return 'Refer to the Kea ARM for details.'
        }
        return ''
    }

    /**
     * Returns help tip for last failover time.
     */
    failoverHelptip(): string {
        return (
            'This is the last time when the ' +
            this.serverName +
            ' server went to the partner-down state ' +
            'because its partner was considered offline as a result of unexpected termination ' +
            'or shutdown.'
        )
    }

    /**
     * Returns help tip for served HA scopes.
     */
    scopesHelptip(): string {
        return (
            'This is a list of HA scopes currently being served by this ' +
            'server. If the server is responding to the DHCP queries as a ' +
            'primary or secondary in the load-balancing mode or as a ' +
            'primary in the hot-standby mode, it is typically a single scope shown. ' +
            'There may be two scopes shown if a load-balancing server is currently ' +
            'serving all DHCP clients when its partner is down. There may be no scopes ' +
            'shown when it is a standby server in the hot-standby mode, because such ' +
            'a server is not responding to any DHCP queries, but passively receiving ' +
            'lease updates from the primary. The standby server will start serving ' +
            'the primary server scope in the event of primary failure.'
        )
    }

    /**
     * Returns help tip for status time.
     */
    statusTimeHelptip(): string {
        return (
            'This is the time when the ' +
            this.serverName +
            ' server reported its state for the last time. ' +
            'This is not necessarily the time when the state information ' +
            'was refreshed in the UI. The presented state information is ' +
            'typically delayed by 10 to 30 seconds because it is cached by the Kea ' +
            'servers and the Stork backend. Caching minimizes the performance ' +
            'impact on the DHCP servers reporting their states over the control ' +
            'channel.'
        )
    }

    /**
     * Returns help tip for status age.
     *
     * The age indicates how long ago the given server reported its status.
     */
    collectedHelptip(): string {
        return (
            'This is the duration between the "Status Time" and now, i.e. this indicates ' +
            'how long ago the ' +
            this.serverName +
            ' server reported its state. A long duration ' +
            'indicates that there is a communication problem with the server. The ' +
            'typical duration is between 10 and 30 seconds.'
        )
    }

    /**
     * Returns help tip for the heartbeat status.
     */
    heartbeatStatusHelptip(): string {
        if (!this.serverStatus.commInterrupted || this.serverStatus.commInterrupted < 0) {
            return 'The status of the heartbeat communication with the ' + this.serverName + ' server is unknown.'
        } else if (this.serverStatus.commInterrupted > 0) {
            return (
                'Heartbeat communication with the ' +
                this.serverName +
                ' server ' +
                ' is interrupted. It means that the server failed to ' +
                ' respond to ha-heartbeat commands longer than the configured ' +
                ' value of max-response-delay.'
            )
        }
        return 'The server responds to ha-heartbeat commands sent by the ' + ' partner.'
    }

    /**
     * Returns help tip for the number of unacked clients.
     */
    unackedClientsHelptip(): string {
        return (
            'This is the number of clients considered unacked by the partner. ' +
            'This value is only set when the partner has lost heartbeat communication ' +
            'with this server and has started the failover procedure, by monitoring ' +
            'whether the server is responding to DHCP traffic. The unacked ' +
            'number indicates clients that have been trying to get leases from this server ' +
            'longer than the time specified by the max-ack-delay configuration ' +
            'parameter.'
        )
    }

    /**
     * Returns help tip for the number of connecting clients counted by
     * the partner server when the heartbeat communication between them is
     * interrupted.
     */
    connectingClientsHelptip(): string {
        return (
            'This is the total number of clients trying to get new leases ' +
            'from the server with which the partner server is unable to ' +
            'communicate via ha-heartbeat. It includes both unacked clients ' +
            'and the clients for which the secs field or elapsed time option is ' +
            'below the max-ack-delay.'
        )
    }

    /**
     * Returns help tip for the number of packets directed to the server when
     * the heartbeat communication to this server gets interrupted.
     */
    analyzedPacketsHelptip(): string {
        return (
            'This is the total number of packets directed to the server ' +
            'with which the partner is unable to communicate via ha-heartbeat. ' +
            'This may include several packets from the same client which ' +
            'tried to resend a DHCPDISCOVER or Solicit after the server failed to ' +
            'respond to previous queries.'
        )
    }

    /**
     * Updates the panel colors according to the status fetched.
     *
     * The colors reflect the state of the HA. The green panel color
     * indicates that the server is in a desired state. The orange
     * color of the panel indicates that some abnormal situation has
     * occurred but it is not severe. For example, one of the servers
     * is down but the orange colored server has taken over serving the
     * DHCP clients. The red colored panel indicates an error which
     * most likely requires Administrator's action. For example, the
     * DHCP server has crashed.
     */
    private refreshPanelColors() {
        switch (this.serverWarnLevel()) {
            case 'ok':
                this.statusPanelClass = 'green-colored-panel'
                break
            case 'warn':
                this.statusPanelClass = 'orange-colored-panel'
                break
            default:
                this.statusPanelClass = 'red-colored-panel'
                break
        }
    }

    /**
     * Checks if the extended HA information is available for the given
     * Kea server.
     *
     * The extended information is returned since Kea 1.7.8 release. It
     * includes information about the failover progress, i.e. how many
     * clients have been trying to get the lease since the heartbeat
     * failure, how many clients failed to get the lease (unacked clients),
     * how many packets have been analyzed by the partner server etc.
     *
     * @returns true if the extended server status is supported, false
     *          otherwise.
     */
    extendedFormatSupported(): boolean {
        // Negative value of the commInterrupted is explicitly indicating
        // that the extended format is not supported. A zero or positive
        // value indicates it is supported.
        return this.hasStatus() && (!this.serverStatus.commInterrupted || this.serverStatus.commInterrupted >= 0)
    }

    /**
     * Checks what icon should be returned for the server.
     *
     * During normal operation the check icon is displayed. If the server is
     * unavailable the red exclamation mark is shown. In other cases a warning
     * exclamation mark on orange triangle is shown.
     */
    serverWarnLevel(): string {
        if (this.stateKind() === HAStateKind.Ok) {
            return 'ok'
        }
        if (this.serverStatus.state === 'unavailable' || this.serverStatus.inTouch === false) {
            return 'error'
        }
        return 'warn'
    }

    /**
     * Checks kind of a state based on server status information.
     *
     * Depending on the value returned by this function different icons are
     * displayed next to the server state.
     *
     * @returns ok when server status has been fetched and indicates normal
     *          operation, e.g. load balancing; not ok when server status
     *          has been fetched and indicates other state; pending if the
     *          server status hasn't been determined yet.
     */
    stateKind(): HAStateKind {
        if (!this.hasStatus || !this.serverStatus.state || this.serverStatus.state === '') {
            return HAStateKind.Pending
        }
        if (
            this.serverStatus.state === 'load-balancing' ||
            this.serverStatus.state === 'hot-standby' ||
            this.serverStatus.state === 'passive-backup'
        ) {
            return HAStateKind.Ok
        }
        return HAStateKind.NotOk
    }

    /**
     * Returns a comma-separated list of HA scopes served by the server.
     *
     * This string is printed in the UI in the local server status box.
     *
     * @returns string containing a comma-separated list of scopes, the word
     *          none, or none (standby server).
     */
    formattedLocalScopes(): string {
        let scopes: string
        if (this.hasStatus()) {
            if (this.serverStatus.scopes) {
                scopes = this.serverStatus.scopes.join(', ')
            }
            if (!scopes && this.serverStatus.state === 'hot-standby' && this.serverStatus.role === 'standby') {
                scopes = 'none (standby server)'
            }
        }

        return scopes || 'none'
    }

    /**
     * Returns formatted value of age.
     *
     * The age indicates how long ago the status of one of the servers has
     * been fetched. It is expressed in seconds. This function displays the
     * age in seconds for the age below 1 minute. It displays the age in
     * minutes otherwise. The negative age value means that the age is not
     * yet determined in which case 'n/a' is displayed.
     *
     * @param age in seconds.
     * @returns string containing formatted age.
     */
    formattedAge(age): string {
        if (age && age < 0) {
            return 'n/a'
        }
        if (!age || age === 0) {
            return 'just now'
        }
        if (age < 60) {
            return age + ' seconds ago'
        }
        return Math.round(age / 60) + ' minutes ago'
    }

    /**
     * Returns formatted status of the control channel.
     *
     * Depending on the value of the boolean parameter specified, this function
     * returns the word "online" or "offline" to indicate the status of the
     * communication with one of the servers.
     *
     * @returns the descriptive information whether the server seems to be
     *          online or offline.
     */
    formattedControlStatus(): string {
        if (!this.hasStatus() || this.serverStatus.inTouch === null) {
            return 'unknown'
        }
        if (this.serverStatus.inTouch) {
            return 'online'
        }
        return 'offline'
    }

    /**
     * Returns formatted HA state of the server.
     *
     * This function checks if the HA status of the server is initialized
     * and returns this status if it is initialized. Otherwise it returns
     * the "not fetched yet" to indicate that the status will appear once
     * it is fetched (pending).
     *
     * @returns a server status or the text "fetching..." if it is not
     * available yet.
     */
    formattedState(): string {
        if (!this.hasStatus() || !this.serverStatus.state || this.serverStatus.state === '') {
            return 'fetching...'
        }
        return this.serverStatus.state
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
    formattedHeartbeatStatus(): string {
        if (this.serverStatus.commInterrupted < 0) {
            return 'unknown'
        } else if (this.serverStatus.commInterrupted > 0) {
            return 'failed'
        }
        return 'ok'
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
    formattedUnackedClients(): string {
        let s = 'n/a'
        let unacked = 0
        let all = 0
        // Monitor unacked clients only if we're in the communication interrupted
        // state, i.e. heartbeat communication has been failing for a certain
        // period of time.
        if (this.hasStatus && this.serverStatus.commInterrupted > 0) {
            if (this.serverStatus.unackedClients != null) {
                unacked = this.serverStatus.unackedClients
            }
            all = unacked
            if (this.serverStatus.unackedClientsLeft != null) {
                all += this.serverStatus.unackedClientsLeft
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
     * Returns formatted failover related number.
     *
     * This function is used to format the number of connecting clients or
     * analyzed packets. If the communication is not interrupted the returned
     * value is n/a. Otherwise, the number is returned.
     *
     * @returns A given value or n/a string.
     */
    formattedFailoverNumber(n): any {
        if (!this.serverStatus.unackedClients && !this.serverStatus.unackedClientsLeft) {
            return 'n/a'
        }
        if (n && n > 0) {
            return n
        }
        return 0
    }
}
