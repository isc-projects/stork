import { Component, Input, OnInit } from '@angular/core'
import { interval } from 'rxjs'
import { ServicesService } from '../backend/api/api'

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
export class HaStatusComponent implements OnInit {
    private _appId: number
    private _daemonName: string
    private _receivedStatus: Map<string, any>

    public localStatusPanelClass = 'green-colored-panel'
    public remoteStatusPanelClass = 'green-colored-panel'

    constructor(private servicesApi: ServicesService) {}

    /**
     * Initializes the component.
     *
     * Sets the interval timer to periodically fetch the HA status from
     * the application.
     */
    ngOnInit() {
        this.refreshStatus()

        interval(1000 * 10).subscribe(x => {
            this.refreshStatus()
        })
        // Run the live age counters for both local and remote servers.
        interval(1000).subscribe(x => {
            if (this.hasStatus()) {
                this.localServer().age += 1
                this.remoteServer().age += 1
            }
        })
    }

    /**
     * Sets Kea application id for which the High Availability state
     * is presented.
     *
     * This should be id of one of the active Kea servers. Stork will
     * communicate with this server to fetch its HA state and the state
     * of its partner.
     *
     * @param appId The application id.
     */
    @Input()
    set appId(appId) {
        this._appId = appId
    }

    /**
     * Returns Kea application id.
     */
    get appId() {
        return this._appId
    }

    /**
     * Sets the name of the Kea deamon for which the HA status is displayed.
     *
     * @param daemonName One of: dhcp4 or dhcp6.
     */
    @Input()
    set daemonName(daemonName) {
        this._daemonName = daemonName
    }

    /**
     * Checks if the component has fetched the HA status for the current daemon.
     *
     * @returns true if the status has been fetched and is available for display.
     */
    hasStatus(): boolean {
        return this._receivedStatus && this._receivedStatus[this._daemonName]
    }

    /**
     * Convenience function returning received status for the current daemon.
     */
    private haStatus() {
        return this._receivedStatus[this._daemonName]
    }

    /**
     * Convenience function returning received status of the local server.
     */
    localServer() {
        if (this._receivedStatus[this._daemonName].primaryServer.id === this.appId) {
            return this._receivedStatus[this._daemonName].primaryServer
        }
        return this._receivedStatus[this._daemonName].secondaryServer
    }

    /**
     * Convenience function returning received status of the remote server.
     */
    remoteServer() {
        if (this._receivedStatus[this._daemonName].primaryServer.id !== this.appId) {
            return this._receivedStatus[this._daemonName].primaryServer
        }
        return this._receivedStatus[this._daemonName].secondaryServer
    }

    /**
     * Gets the HA status of the Kea server and its partner.
     *
     * This function is invoked periodically to refresh the status.
     */
    private refreshStatus() {
        this.servicesApi.getAppServicesStatus(this.appId).subscribe(
            data => {
                if (data.items) {
                    this._receivedStatus = new Map()
                    for (const s of data.items) {
                        if (s.status.haServers && s.status.daemon) {
                            this._receivedStatus[s.status.daemon] = s.status.haServers
                        }
                    }
                }
                this.refreshPanelColors()
            },
            err => {
                console.warn('failed to fetch the HA status for Kea application id ' + this.appId)
                this._receivedStatus = null
            }
        )
    }

    /**
     * Updates the HA local and remote servers' panel colors.
     *
     * The colors reflect the state of the HA. The green panel color
     * indicates that the servers are in the desired states. The orange
     * color of the panel indicates that some abnormal situation has
     * occurred but it is not severe. For example, one of the servers
     * is down but the orange colored server has taken over serving the
     * DHCP clients. The red colored panel indicates an error which
     * most likely requires Administrator's action. For example, the
     * DHCP server has crashed.
     */
    private refreshPanelColors() {
        switch (this.localServerWarnLevel()) {
            case 'ok':
                this.localStatusPanelClass = 'green-colored-panel'
                break
            case 'warn':
                this.localStatusPanelClass = 'orange-colored-panel'
                break
            default:
                this.localStatusPanelClass = 'red-colored-panel'
                break
        }

        switch (this.remoteServerWarnLevel()) {
            case 'ok':
                this.remoteStatusPanelClass = 'green-colored-panel'
                break
            case 'warn':
                this.remoteStatusPanelClass = 'orange-colored-panel'
                break
            default:
                this.remoteStatusPanelClass = 'red-colored-panel'
                break
        }
    }

    /**
     * Returns the tooltip describing online/offline control status.
     */
    controlStatusTooltip(inTouch): string {
        if (inTouch) {
            return 'Server responds to the commands over the control channel.'
        }
        return 'Server does not respond to the commands over the control channel. It may be down!'
    }

    /**
     * Returns tooltip describing various HA statuses.
     */
    haStateTooltip(state): string {
        switch (state) {
            case 'load-balancing':
            case 'hot-standby':
                return 'Normal operation.'
            case 'partner-down':
                return (
                    'This server now responds to all DHCP queries because it detected ' +
                    'that partner server is not functional!'
                )
            case 'waiting':
                return 'This server is apparently booting up and will try to synchronize its lease database.'
            case 'syncing':
                return 'This saerver is synchronizing its database after failure.'
            case 'ready':
                return 'This server synchronized its lease database and will start normal operation shortly.'
            case 'terminated':
                return 'This server no longer participates in the HA setup because of the too high clock skew.'
            case 'maintained':
                return 'This server is under maintenance.'
            case 'partner-maintained':
                return 'This server responds to all DHCP queries for the partner being in maintenance.'
            case 'unavailable':
                return 'Communication with the server failed. It may have crashed or have been shut down.'
            default:
                return 'Refer to Kea manual for details.'
        }
        return ''
    }

    /**
     * Returns tooltip for last failover time.
     */
    failoverTooltip(name): string {
        return (
            'This is the last time when the ' +
            name +
            ' server went to the partner-down state ' +
            'because its partner was considered offline as a result of unexpected termination ' +
            'or shutdown.'
        )
    }

    /**
     * Checks what icon should be returned for the local server.
     *
     * During normal operation the check icon is displayed. If the server is
     * unavailable the red exclamation mark is shown. In other cases a warning
     * exclamation mark on orange triangle is shown.
     */
    localServerWarnLevel(): string {
        if (this.localStateOk()) {
            return 'ok'
        }
        if (this.localServer().state === 'unavailable' || this.localServer().inTouch === false) {
            return 'error'
        }
        return 'warn'
    }

    /**
     * Checks what icon should be returned for the remote server.
     *
     * During normal operation the check icon is displayed. If the server is
     * unavailable the red exclamation mark is shown. In other cases a warning
     * exclamation mark on orange triangle is shown.
     */
    remoteServerWarnLevel(): string {
        if (this.remoteStateOk()) {
            return 'ok'
        }
        if (this.remoteServer().state === 'unavailable' || this.remoteServer().inTouch === false) {
            return 'error'
        }
        return 'warn'
    }

    /**
     * Checks if the state of the local HA enabled server is good.
     *
     * The desired state is either load-balancing or hot-standby. In other
     * cases it means that the servers are booting up or there is some issue
     * with the partner causing the local server to go to partner-down.
     *
     * @returns true if the server is in the load-balancing or hot-standby
     *          state. This is used in the UI to highlight a potential problem.
     */
    localStateOk(): boolean {
        return (
            this.haStatus() &&
            (this.localServer().state === 'load-balancing' || this.localServer().state === 'hot-standby')
        )
    }

    /**
     * Checks if the state of the remote HA enabled server is good.
     *
     * The desired state is either load-balancing or hot-standby. In other
     * cases it means that the servers are booting up or there is some issue
     * with the partner causing the local server to go to partner-down.
     *
     * @returns true if the server is in the load-balancing or hot-standby
     *          state. This is used in the UI to highlight a potential problem.
     */
    remoteStateOk(): boolean {
        return (
            this.haStatus() &&
            (this.remoteServer().state === 'load-balancing' || this.remoteServer().state === 'hot-standby')
        )
    }

    /**
     * Returns an array of scopes served by the local server.
     *
     * @returns array of strings including local server scopes.
     */
    private localServerScopes(): string[] {
        let scopes: string[] = []
        if (this.hasStatus() && this.localServer().scopes) {
            scopes = this.localServer().scopes.join(', ')
        }
        return scopes
    }

    /**
     * Returns an array of scopes served by the remote server.
     *
     * @returns array of strings including remote server scopes.
     */
    private remoteServerScopes(): string[] {
        let scopes: string[] = []
        if (this.hasStatus() && this.remoteServer().scopes) {
            scopes = this.remoteServer().scopes.join(', ')
        }
        return scopes
    }

    /**
     * Returns a comma separated list of HA scopes served by the local server.
     *
     * This string is printed in the UI in the local server status box.
     *
     * @returns string containing comma separated list of scopes or (none).
     */
    formattedLocalScopes(): string {
        let scopes: string
        if (this.hasStatus() && this.localServer().scopes) {
            scopes = this.localServer().scopes.join(', ')
        }

        return scopes || '(none)'
    }

    /**
     * Returns a comma separated list of HA scopes served by the remote server.
     *
     * This string is printed in the UI in the remote server status box.
     *
     * @returns string containing comma separated list of scopes or (none).
     */
    formattedRemoteScopes(): string {
        let scopes: string
        if (this.hasStatus() && this.remoteServer().scopes) {
            scopes = this.remoteServer().scopes.join(', ')
        }

        return scopes || '(none)'
    }

    /**
     * Returns formatted value of age.
     *
     * The age indicates how long ago the status of one of the servers has
     * been fetched. It is expressed in seconds. This function displays the
     * age in seconds for the age below 1 minute. It displays the age in
     * minutes otherwise. The nagative age value means that the age is not
     * yet determined in which case a hyphen is displayed.
     *
     * @param age in seconds.
     * @returns string containing formatted age.
     */
    formattedAge(age): string {
        if (age && age < 0) {
            return '-'
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
     * @param inTouch boolean value indicating if the communication with the
     *                server was successful or not.
     * @returns the descriptive information whether the server seems to be
     *          online or offline.
     */
    formattedControlStatus(inTouch): string {
        if (inTouch) {
            return 'online'
        }
        return 'offline'
    }

    /**
     * Returns information to be displayed in the note box.
     *
     * The text explains the meaning of scopes served by the local and the
     * remote server. The DHCP clients are grouped into scopes served by the
     * HA partners. Depending on the state, the HA servers become responsible
     * for different scopes. The text displayed in the note box explains which
     * groups of clients are served by which DHCP servers.
     */
    footerInfo(): string {
        if (!this.hasStatus()) {
            return 'No HA information available!'
        }

        // The local server serves no clients, so the remote serves all of them.
        // It may be a hot-standby case or partner-down case.
        if (this.localServerScopes().length === 0 && this.remoteServerScopes().length > 0) {
            return 'The remote server responds to the entire DHCP traffic.'
        }

        // The remote server serves no cliebts, so the local serves all of them.
        // It may be a hot-standby case or partner-down case.
        if (this.remoteServerScopes().length === 0 && this.localServerScopes().length > 0) {
            return 'The local server responds to the entire DHCP traffic.'
        }

        // This is the load-balancing case when both servers respond to some
        // DHCP traffic.
        if (this.remoteServerScopes().length > 0 && this.localServerScopes().length > 0) {
            return 'Both servers respond to the DHCP traffic.'
        }

        // If the HA service is being started, the servers synchronize their
        // databases and do not respond to any traffic.
        if (this.remoteServerScopes().length === 0 && this.localServerScopes().length === 0) {
            return 'No servers respond to the DHCP traffic.'
        }
    }
}
