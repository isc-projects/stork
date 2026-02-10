import { HttpErrorResponse } from '@angular/common/http'
import { Component, OnDestroy, OnInit } from '@angular/core'
import { Router, ActivatedRoute } from '@angular/router'
import { MenuItem, MessageService } from 'primeng/api'
import { Subject, Subscription } from 'rxjs'
import { switchMap } from 'rxjs/operators'
import { ServicesService } from '../backend/api/api'
import { ServerDataService } from '../server-data.service'
import { Daemon, KeaDaemonConfig } from '../backend'
import { BreadcrumbsComponent } from '../breadcrumbs/breadcrumbs.component'
import { Panel } from 'primeng/panel'
import { NgIf } from '@angular/common'
import { Button } from 'primeng/button'
import { JsonTreeRootComponent } from '../json-tree-root/json-tree-root.component'
import { Message } from 'primeng/message'

/**
 * A component providing a dedicated page displaying Kea daemon configuration.
 *
 * It fetches configuration data and displays it in a JSON viewer.
 * The viewer allows for collapsing and expanding selected or all nodes.
 */
@Component({
    selector: 'app-kea-daemon-configuration-page',
    templateUrl: './kea-daemon-configuration-page.component.html',
    styleUrls: ['./kea-daemon-configuration-page.component.sass'],
    imports: [BreadcrumbsComponent, Panel, NgIf, Button, JsonTreeRootComponent, Message],
})
export class KeaDaemonConfigurationPageComponent implements OnInit, OnDestroy {
    breadcrumbs: MenuItem[] = []

    // Variables to store values for getters. See specific getter for documentation.
    private _autoExpand: 'none' | 'all' = 'none'
    private _configuration = null
    private _daemonId: number = null
    private _downloadFilename = 'data.json'
    private _failedFetch = false

    private changeDaemonId = new Subject<number>()
    private subscriptions = new Subscription()

    constructor(
        private route: ActivatedRoute,
        private router: Router,
        private servicesApi: ServicesService,
        private serverData: ServerDataService,
        private msgService: MessageService
    ) {}

    /**
     * Unsubscribe all subscriptions.
     */
    ngOnDestroy(): void {
        this.changeDaemonId.complete()
        this.subscriptions.unsubscribe()
    }

    /**
     * Component lifecycle hook invoked upon the component initialization.
     *
     * It subscribes for necessary data, i.e. friendly names and daemon configuration JSON.
     *
     * The daemon friendly name is fetched for the specified daemon ID query parameter. If the daemon
     * with the specified ID does not exist or the ID is invalid, a placeholder for daemon name
     * is displayed.
     *
     * The daemon ID must be valid and must point to an existing daemon. The function uses
     * it to fetch daemon's friendly name and fetch its configuration. If the daemon ID is
     * invalid, the user is redirected to the daemons list.
     */
    ngOnInit(): void {
        this.breadcrumbs = [
            { label: 'Services' },
            { label: 'Daemons', routerLink: '/daemons/all' },
            { label: 'Daemon' },
            { label: 'Configuration' },
        ]

        // Update Kea daemon configuration
        this.subscriptions.add(
            this.changeDaemonId
                .pipe(switchMap((daemonId) => this.serverData.getDaemonConfiguration(daemonId)))
                .subscribe((res: KeaDaemonConfig) => {
                    if (res instanceof HttpErrorResponse) {
                        this.msgService.add({
                            severity: 'error',
                            summary: 'Fetching daemon configuration failed',
                            detail: res.error?.message ?? res.message,
                            life: 10000,
                        })
                        this._failedFetch = true
                        this._configuration = null
                    } else {
                        this._failedFetch = false
                        this._configuration = res.config || null
                    }
                })
        )

        // Resolve URI parameters
        this.subscriptions.add(
            this.route.paramMap.subscribe((params) => {
                const daemonIdStr = params.get('daemonId')

                const daemonId = parseInt(daemonIdStr, 10)

                // Daemon ID is required
                if (!Number.isFinite(daemonId)) {
                    this.router.navigate(['/daemons/all'])
                }

                this._daemonId = daemonId

                this.changeDaemonId.next(daemonId)
                this._downloadFilename = `daemon_${daemonId}.json`
            })
        )
    }

    /**
     * Handle click event of toggle collapse/expand nodes button.
     *
     * JSON viewer uses HTML built-in details-summary tags. It means that we cannot
     * directly control collapse/expand state. We can only toggle "open" HTML property
     * to indicate that tag should be initially expanded or not.
     *
     * This function implements an auto expand feature and setting the count of the
     * auto expanded nodes to 0 (collapse) or max integer value (expand).
     */
    onClickToggleNodes() {
        if (this._autoExpand === 'none') {
            this._autoExpand = 'all'
        } else {
            this._autoExpand = 'none'
        }
    }

    /** Handle click event on refresh button. */
    onClickRefresh() {
        // Reset configuration instance to display loading indicator.
        this._configuration = null
        this._failedFetch = false
        this.serverData.forceReloadDaemonConfiguration(this.daemonId)
    }

    /** Specifies current toggle/expand button state. */
    get autoExpand() {
        return this._autoExpand
    }

    /** Kea daemon configuration to display */
    get configuration() {
        return this._configuration
    }

    /** Filename of downloaded Kea daemon configuration */
    get downloadFilename() {
        return this._downloadFilename
    }

    /** Kea daemon ID of current configuration. If not set then it is null. */
    get daemonId() {
        return this._daemonId
    }

    /** Indicates that fetch configuration failed. */
    get failedFetch() {
        return this._failedFetch
    }
}
