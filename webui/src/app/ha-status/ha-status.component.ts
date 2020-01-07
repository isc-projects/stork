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
    private _statusErred: Map<string, boolean>
    private _ageCounter: Map<string, number>

    constructor(private servicesApi: ServicesService) {}

    /**
     * Initializes the component.
     *
     * Sets the interval timer to periodically fetch the HA status from
     * the application.
     */
    ngOnInit() {
        this._statusErred = new Map()
        this._ageCounter = new Map()

        this.refreshStatus()

        interval(1000 * 10).subscribe(x => {
            this.refreshStatus()
        })

        interval(1000).subscribe(x => {
            this._ageCounter[this.daemonName] += 1
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
     * Returns age of the remote server HA status.
     *
     * The remote server's status is periodically collected by the local server,
     * i.e. the server to which Stork sends the status-get command.
     *
     * @returns Text indicating how old is the information about the status
     *          of the remote server. It is displayed in seconds when it is
     *          below 1 minute and in minutes otherwise.
     */
    get ageCounter(): string {
        const age = this._ageCounter[this._daemonName]
        if (!age || age === 0) {
            return 'just now'
        }
        if (age < 60) {
            return age + ' seconds ago'
        }
        return Math.round(age / 60) + ' minutes ago'
    }

    /**
     * Checks if the component has fetched the HA status for the current daemon.
     *
     * @returns true if the status has been fetched and is available for display.
     */
    get hasStatus(): boolean {
        return this._receivedStatus && this._receivedStatus[this._daemonName]
    }

    /**
     * Checks if the component has attempted to fetch the HA status and failed.
     *
     * If the status is erred, the error message is displayed indicating an issue
     * with communication.
     *
     * @returns true if there was an error while fetching the HA status.
     */
    get hasErredStatus(): boolean {
        return this._statusErred && this._statusErred[this._daemonName]
    }

    /**
     * Convenience function returning received status for the current daemon.
     */
    get ha() {
        return this._receivedStatus[this._daemonName]
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
                            this._ageCounter[s.status.daemon] = s.status.haServers.remoteServer.age
                            this._statusErred[s.status.daemon] = false
                        }
                    }
                }
                // We were unable to fetch the HA status for this server, thus
                // we mark it as erred.
                if (!this._receivedStatus || !this._receivedStatus[this._daemonName]) {
                    this._statusErred[this._daemonName] = true
                }
            },
            err => {
                console.warn('failed to fetch the HA status for Kea application id ' + this.appId)
                this._statusErred[this._daemonName] = true
                this._receivedStatus = null
            }
        )
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
            this.ha && (this.ha.localServer.state === 'load-balancing' || this.ha.localServer.state === 'hot-standby')
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
            this.ha && (this.ha.remoteServer.state === 'load-balancing' || this.ha.remoteServer.state === 'hot-standby')
        )
    }

    /**
     * Returns a comma separated list of HA scopes served by the local server.
     *
     * This string is printed in the UI in the local server status box.
     *
     * @returns string containing comma separated list of scopes or (none).
     */
    localScopes(): string {
        let scopes: string
        if (this.hasStatus) {
            scopes = this.ha.localServer.scopes.join(', ')
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
    remoteScopes(): string {
        let scopes: string
        if (this.hasStatus) {
            scopes = this.ha.remoteServer.scopes.join(', ')
        }

        return scopes || '(none)'
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
        if (!this.hasStatus) {
            return 'No HA information available!'
        }

        // The local server serves no clients, so the remote serves all of them.
        // It may be a hot-standby case or partner-down case.
        if (this.ha.localServer.scopes.length === 0 && this.ha.remoteServer.scopes.length > 0) {
            return 'The remote server responds to the entire DHCP traffic.'
        }

        // The remote server serves no cliebts, so the local serves all of them.
        // It may be a hot-standby case or partner-down case.
        if (this.ha.remoteServer.scopes.length === 0 && this.ha.localServer.scopes.length > 0) {
            return 'The local server responds to the entire DHCP traffic.'
        }

        // This is the load-balancing case when both servers respond to some
        // DHCP traffic.
        if (this.ha.remoteServer.scopes.length > 0 && this.ha.localServer.scopes.length > 0) {
            return 'Both servers respond to the DHCP traffic.'
        }

        // If the HA service is being started, the servers synchronize their
        // databases and do not respond to any traffic.
        if (this.ha.remoteServer.scopes.length === 0 && this.ha.localServer.scopes.length === 0) {
            return 'No servers respond to the DHCP traffic.'
        }
    }
}
