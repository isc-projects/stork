import { HttpErrorResponse } from '@angular/common/http'
import { Component, OnDestroy, OnInit } from '@angular/core'
import { Router, ActivatedRoute } from '@angular/router'
import { MessageService } from 'primeng/api'
import { Subject, Subscription } from 'rxjs'
import { switchMap } from 'rxjs/operators'
import { ServicesService } from '../backend/api/api'
import { ServerDataService } from '../server-data.service'

/**
 * Component providing a dedicated page for Kea daemon configuration.
 *
 * It fetching all needed data, displaying JSON viewer and providing a few additional functionalities.
 */
@Component({
    selector: 'app-kea-daemon-configuration-page',
    templateUrl: './kea-daemon-configuration-page.component.html',
    styleUrls: ['./kea-daemon-configuration-page.component.sass'],
})
export class KeaDaemonConfigurationPageComponent implements OnInit, OnDestroy {
    breadcrumbs = [
        { label: 'Services' },
        { label: 'Kea Apps', routerLink: '/apps/kea/all' },
        { label: 'App' },
        { label: 'Daemons' },
        { label: 'Daemon' },
        { label: 'Configuration' },
    ]

    // Variables to store values for getters. See specific getter for documentation.
    private _autoExpandNodeCount = 0
    private _configuration = null
    private _daemonId: number = null
    private _downloadFilename = 'data.json'
    private _failedFetch = false

    private changeDaemonId = new Subject<number>()
    private changeAppId = new Subject<number>()
    private subscription = new Subscription()

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
        this.subscription.unsubscribe()
    }

    /**
     * Here the component subscribes for data: friendly names and Kea daemon configuration JSON
     *
     * Handles application ID URL query parameter.
     * When application ID changes then the component displays its friendly name.
     * Properly application ID is optional. It may be omitted or invalid then the name
     * will be invalid or placeholder, but the component displays correct Kea daemon configuration.
     *
     * Handles daemon ID URL query parameter.
     * When daemon ID changes then the component displays its friendly name and fetch Kea daemon
     * configuration JSON. If daemon ID is empty or invalid (non-numeric) then user is redirected to
     * application list.
     *
     * This component subscribes the updates of Kea daemon configuration JSON too.
     */
    ngOnInit(): void {
        // Friendly names of daemons
        const DMAP = {
            dhcp4: 'DHCPv4',
            dhcp6: 'DHCPv6',
            d2: 'DDNS',
            ca: 'CA',
            netconf: 'NETCONF',
        }

        // Update friendly names
        this.subscription.add(
            this.changeAppId.pipe(switchMap((appId) => this.servicesApi.getApp(appId))).subscribe((app) => {
                // Find specific daemon
                const daemons = app.details.daemons.filter((d) => d.id === this._daemonId)
                const daemonName = daemons[0]?.name
                const friendlyName = DMAP[daemonName] ?? daemonName ?? this._daemonId + '' ?? 'Unknown'

                // User-friendly download filename
                this._downloadFilename = `${app.name}_${friendlyName}.json`

                // Breadcrumbs
                this.breadcrumbs[2]['label'] = app.name
                this.breadcrumbs[2]['routerLink'] = `/apps/kea/${app.id}`
                this.breadcrumbs[4]['label'] = friendlyName
                this.breadcrumbs[4]['routerLink'] = `/apps/kea/${app.id}?daemon=${daemonName}`
            })
        )

        // Update Kea daemon configuration
        this.subscription.add(
            this.changeDaemonId
                .pipe(switchMap((daemonId) => this.serverData.getDaemonConfiguration(daemonId)))
                .subscribe((res) => {
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
                        this._configuration = res
                    }
                })
        )

        // Resolve URI parameters
        this.subscription.add(
            this.route.paramMap.subscribe((params) => {
                const appIdStr = params.get('appId')
                const daemonIdStr = params.get('daemonId')

                const appId = parseInt(appIdStr, 10)
                const daemonId = parseInt(daemonIdStr, 10)

                // Daemon ID is required
                if (!Number.isFinite(daemonId)) {
                    this.router.navigate(['/apps/kea/all'])
                }

                this._daemonId = daemonId

                // Fetch app and daemon names
                // It may fail, but it isn't critical (for this view).
                if (Number.isFinite(appId)) {
                    this.changeAppId.next(appId)
                }

                this.changeDaemonId.next(daemonId)
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
     * This function uses a trick with auto expand feature and set it to 0 or max integer value.
     *
     * It works well as expected, but it must have a toggle behavior. It is impossible
     * to collapse viewer which wasn't previously expanded. And vice-versa.
     *
     * To resolve this issue it is needed to explicit manage details-summary tags state
     * in component. But it will complicate viewer solution and not bring many benefits.
     */
    onClickToggleNodes() {
        if (this._autoExpandNodeCount === Number.MAX_SAFE_INTEGER) {
            this._autoExpandNodeCount = 0
        } else {
            this._autoExpandNodeCount = Number.MAX_SAFE_INTEGER
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
    get currentAction() {
        return this._autoExpandNodeCount === 0 ? 'expand' : 'collapse'
    }

    /** Returns 0 for collapse nodes or maximal integer for expand. */
    get autoExpandNodeCount() {
        return this._autoExpandNodeCount
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
