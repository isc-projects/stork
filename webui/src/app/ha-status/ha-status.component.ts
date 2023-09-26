import { Component, Input, OnDestroy, OnInit } from '@angular/core'
import { interval, Subscription } from 'rxjs'
import { ServicesService } from '../backend/api/api'
import { KeaStatusHaServers } from '../backend'
import { MessageService } from 'primeng/api'
import { getErrorMessage } from '../utils'

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
    private subscriptions = new Subscription()
    private readonly _haRefreshInterval = 10000
    private readonly _countUpInterval = 1000

    private _appId: number
    private _daemonName: string
    private _receivedStatus: Record<string, KeaStatusHaServers>

    /**
     * Indicates if the data were loaded at least once.
     */
    loadedOnce: boolean = false

    constructor(
        private servicesApi: ServicesService,
        private messageService: MessageService
    ) {}

    ngOnDestroy(): void {
        this.subscriptions.unsubscribe()
    }

    /**
     * Initializes the component.
     *
     * Sets the interval timer to periodically fetch the HA status from
     * the application.
     */
    ngOnInit() {
        this.refreshStatus()

        this.subscriptions.add(
            interval(this._haRefreshInterval).subscribe((x) => {
                this.refreshStatus()
            })
        )
        // Run the live age counters for both local and remote servers.
        // ToDo: Check that it works as expected. Does the timer reset when the component is destroyed?
        this.subscriptions.add(
            interval(this._countUpInterval).subscribe((x) => {
                if (this.hasStatus()) {
                    // Only increase the age counters if they are non-negative.
                    // Negative values indicate that the status age was unknown,
                    // probably because the server was down when attempted to get
                    // its status.
                    if (this.localServer().age >= 0) {
                        this.localServer().age += 1
                    }
                    if (this.remoteServer() && this.remoteServer().age >= 0) {
                        this.remoteServer().age += 1
                    }
                }
            })
        )
    }

    /**
     * Sets Kea application id for which the High Availability state
     * is presented.
     *
     * This should be id of one of the active Kea servers. The HA status
     * of this server and its partner (if it has partner) will be fetched
     * and presented.
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
     * Sets the name of the Kea daemon for which the HA status is displayed.
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
        return !!this._receivedStatus?.[this._daemonName]
    }

    /**
     * Convenience function returning received status of the local server.
     */
    localServer() {
        if (this._receivedStatus[this._daemonName].primaryServer.appId === this.appId) {
            return this._receivedStatus[this._daemonName].primaryServer
        }
        return this._receivedStatus[this._daemonName].secondaryServer
    }

    /**
     * Convenience function returning received status of the remote server.
     */
    remoteServer() {
        if (this._receivedStatus[this._daemonName].primaryServer.appId !== this.appId) {
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
        this.servicesApi
            .getAppServicesStatus(this.appId)
            .toPromise()
            .then((data) => {
                if (data.items) {
                    this._receivedStatus = {}
                    for (const s of data.items) {
                        if (s.status.haServers && s.status.daemon) {
                            this._receivedStatus[s.status.daemon] = s.status.haServers
                        }
                    }
                }
                this.loadedOnce = true
            })
            .catch((err) => {
                this.messageService.add({
                    severity: 'error',
                    summary: `Failed to fetch the HA status for Kea application ID: ${this.appId}`,
                    detail: getErrorMessage(err),
                })
                this._receivedStatus = null
            })
    }

    /**
     * Returns an array of scopes served by the local server.
     *
     * @returns array of strings including local server scopes.
     */
    private localServerScopes(): string[] {
        let scopes: string[] = []
        if (this.hasStatus() && !!this.localServer().scopes) {
            scopes = this.localServer().scopes
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
            scopes = this.remoteServer().scopes
        }
        return scopes
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
    serverFailoverProgress(server): number {
        if (!server) {
            return -1
        }
        let unacked = 0
        if (server.unackedClients > 0) {
            unacked = server.unackedClients
        }
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

        const localFailoverProgress = this.serverFailoverProgress(this.localServer())
        const remoteFailoverProgress = this.serverFailoverProgress(this.remoteServer())

        if (localFailoverProgress >= 0 && remoteFailoverProgress >= 0) {
            return 'Each server failed to see the other and started failover procedure:'
        }

        if (localFailoverProgress >= 0) {
            // Note that the failover for the local server (in case of the local
            // server failure) is conducted by the remote server and vice versa.
            return 'Failover procedure in progress by remote server:'
        }

        if (remoteFailoverProgress >= 0) {
            // Note that the failover for the remote server (in case of the remote
            // server failure) is conducted by the local server and vice versa.
            return 'Failover procedure in progress by local server:'
        }

        // The local server serves no clients, so the remote serves all of them.
        // It may be a hot-standby case or partner-down case.
        if (this.localServerScopes().length === 0 && this.remoteServerScopes().length > 0) {
            return 'The remote server is responding to all DHCP traffic.'
        }

        // The remote server serves no clients, so the local serves all of them.
        // It may be a hot-standby case or partner-down case.
        if (this.remoteServerScopes().length === 0 && this.localServerScopes().length > 0) {
            return 'The local server is responding to all DHCP traffic.'
        }

        // This is the load-balancing case when both servers respond to some
        // DHCP traffic.
        if (this.remoteServerScopes().length > 0 && this.localServerScopes().length > 0) {
            return 'Both servers are responding to DHCP traffic.'
        }

        // If the HA service is being started, the servers synchronize their
        // databases and do not respond to any traffic.
        if (this.remoteServerScopes().length === 0 && this.localServerScopes().length === 0) {
            return 'No servers are responding to DHCP traffic.'
        }
    }
}
