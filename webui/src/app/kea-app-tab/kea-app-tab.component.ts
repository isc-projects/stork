import { Component, OnInit, Input, Output, EventEmitter, OnDestroy } from '@angular/core'
import { ActivatedRoute } from '@angular/router'
import { forkJoin, Observable, Subscription } from 'rxjs'
import { prerelease, gte } from 'semver'

import { MessageService } from 'primeng/api'

import { AppTab } from '../apps'
import { ServicesService } from '../backend/api/api'
import { ServerDataService } from '../server-data.service'

import {
    durationToString,
    daemonStatusErred,
    daemonStatusIconName,
    daemonStatusIconColor,
    daemonStatusIconTooltip,
    getErrorMessage,
} from '../utils'
import { KeaDaemon, ModelFile } from '../backend'

@Component({
    selector: 'app-kea-app-tab',
    templateUrl: './kea-app-tab.component.html',
    styleUrls: ['./kea-app-tab.component.sass'],
})
export class KeaAppTabComponent implements OnInit, OnDestroy {
    private subscriptions = new Subscription()
    private _appTab: AppTab
    @Output() refreshApp = new EventEmitter<number>()
    @Input() refreshedAppTab: Observable<AppTab>

    daemons: KeaDaemon[] = []

    activeTabIndex = 0

    /**
     * Holds a map of existing apps' names and ids.
     *
     * The apps' names are used in rename-app-dialog component to validate
     * the user input.
     */
    existingApps = new Map<string, number>()

    /**
     * Holds a set of existing machines' addresses.
     *
     * The machines' addresses are used in rename-app-dialog component to
     * validate the user input.
     */
    existingMachines = new Set<string>()

    /**
     * Controls whether the rename-app-dialog is visible or not.
     */
    appRenameDialogVisible = false

    /**
     * Indicates if a pencil icon was clicked.
     *
     * As a result of clicking this icon a dialog box is shown to
     * rename an app. Loading the dialog box may take a while before
     * the information about available apps and machines is loaded.
     * In the meantime, a spinner is shown, indicating that the dialog
     * box is loading.
     */
    showRenameDialogClicked = false

    /**
     * Holds Kea documentation anchors indexed by hook libraries base names.
     * Anchors are valid for Kea versions < 2.4.0.
     * For more recent Kea versions, there is no need to track the anchors,
     * since new anchor type was provided in documentation, which can be generated
     * automatically: std-ischooklib-hook_lib_base_name.so
     *
     * Makes lookup as efficient as O(log(n)) instead of the O(n)
     * that would result from an alternative switch-case statement.
     */
    anchorsByHook = {
        'libdhcp_bootp.so': 'bootp-support-for-bootp-clients',
        'libdhcp_cb_cmds.so': 'cb-cmds-configuration-backend-commands',
        'libdhcp_class_cmds.so': 'class-cmds-class-commands',
        'libdhcp_ddns_tuning.so': 'ddns-tuning-ddns-tuning',
        'libdhcp_flex_id.so': 'flex-id-flexible-identifier-for-host-reservations',
        'libdhcp_flex_option.so': 'flex-option-flexible-option-actions-for-option-value-settings',
        'libddns_gss_tsig.so': 'gss-tsig-sign-dns-updates-with-gss-tsig',
        'libdhcp_ha.so': 'ha-high-availability-outage-resilience-for-kea-servers',
        'libdhcp_host_cache.so': 'host-cache-host-cache-reservations-for-improved-performance',
        'libdhcp_host_cmds.so': 'host-cmds-host-commands',
        'libdhcp_lease_cmds.so': 'lease-cmds-lease-commands-for-easier-lease-management',
        'libdhcp_lease_query.so': 'lease-query-leasequery-support',
        'libdhcp_legal_log.so': 'legal-log-forensic-logging',
        'libdhcp_limits.so': 'limits-limits-to-manage-lease-allocation-and-packet-processing',
        'libdhcp_mysql_cb.so': 'mysql-cb-configuration-backend-for-mysql',
        'libdhcp_pgsql_cb.so': 'pgsql-cb-configuration-backend-for-postgresql',
        'libdhcp_radius.so': 'radius-radius-server-support',
        'libca_rbac.so': 'rbac-role-based-access-control',
        'libdhcp_run_script.so': 'run-script-run-script-support-for-external-hook-scripts',
        'libdhcp_stat_cmds.so': 'stat-cmds-statistics-commands-for-supplemental-lease-statistics',
        'libdhcp_subnet_cmds.so': 'subnet-cmds-subnet-commands-to-manage-subnets-and-shared-networks',
        'libdhcp_user_chk.so': 'user-chk-user-check',
    }

    /**
     * Event emitter sending an event to the parent component when an app is
     * renamed.
     */
    @Output() renameApp = new EventEmitter<string>()

    constructor(
        private route: ActivatedRoute,
        private servicesApi: ServicesService,
        private serverData: ServerDataService,
        private msgService: MessageService
    ) {}

    ngOnDestroy(): void {
        this.subscriptions.unsubscribe()
    }

    /**
     * Subscribes to the updates of the information about daemons
     *
     * The information about the daemons may be updated as a result of
     * pressing the refresh button in the app tab. In such case, this
     * component emits an event to which the parent component reacts
     * and updates the daemons. When the daemons are updated, it
     * notifies this component via the subscription mechanism.
     */
    ngOnInit() {
        this.subscriptions.add(
            this.refreshedAppTab.subscribe((data) => {
                if (data) {
                    this.initDaemons(data.app.details.daemons)
                }
            })
        )
    }

    /**
     * Selects new application tab
     *
     * As a result, the local information about the daemons is updated.
     *
     * @param appTab pointer to the new app tab data structure.
     */
    @Input()
    set appTab(appTab: AppTab) {
        this._appTab = appTab
        // Refresh local information about the daemons presented by this
        // component.
        this.initDaemons(appTab.app.details.daemons)
    }

    /**
     * Returns information about currently selected app tab.
     */
    get appTab(): AppTab {
        return this._appTab
    }

    /**
     * Initializes information about the daemons according to the information
     * carried in the provided parameter.
     *
     * As a result of invoking this function, the view of the component will be
     * updated.
     *
     * @param appTabDaemons information about the daemons stored in the app tab
     *                      data structure.
     */
    private initDaemons(appTabDaemons: KeaDaemon[]) {
        const activeDaemonTabName = this.route.snapshot.queryParamMap.get('daemon')
        const daemonMap = []
        for (const d of appTabDaemons) {
            daemonMap[d.name] = d
        }
        const DMAP = [
            ['dhcp4', 'DHCPv4'],
            ['dhcp6', 'DHCPv6'],
            ['d2', 'DDNS'],
            ['ca', 'CA'],
            ['netconf', 'NETCONF'],
        ]
        const daemons = []
        let idx = 0
        for (const dm of DMAP) {
            if (daemonMap[dm[0]] !== undefined) {
                daemonMap[dm[0]].niceName = dm[1]
                daemonMap[dm[0]].subnets = []
                daemonMap[dm[0]].totalSubnets = 0
                daemonMap[dm[0]].statusErred = this.daemonStatusErred(daemonMap[dm[0]])
                daemons.push(daemonMap[dm[0]])

                if (dm[0] === activeDaemonTabName) {
                    this.activeTabIndex = idx
                }
                idx += 1
            }
        }
        this.daemons = daemons
    }

    /**
     * An action triggered when refresh button is pressed.
     */
    refreshAppState() {
        this.refreshApp.emit(this._appTab.app.id)
    }

    /**
     * Converts duration to pretty string.
     *
     * @param duration duration value to be converted.
     *
     * @returns duration as text
     */
    showDuration(duration: number) {
        return durationToString(duration)
    }

    /**
     * Returns boolean value indicating if there is an issue with communication
     * with the active daemon.
     *
     * @param daemon data structure holding the information about the daemon.
     *
     * @return true if there is a communication problem with the daemon,
     *         false otherwise.
     */
    private daemonStatusErred(daemon: KeaDaemon): boolean {
        if (daemon.active && daemonStatusErred(daemon)) {
            return true
        }
        return false
    }

    /**
     * Returns the name of the icon to be used when presenting daemon status
     *
     * The icon selected depends on whether the daemon is active or not
     * active and whether there is a communication with the daemon or
     * not.
     *
     * @param daemon data structure holding the information about the daemon.
     *
     * @returns ban icon if the daemon is not active, times icon if the daemon
     *          should be active but the communication with it is broken and
     *          check icon if the communication with the active daemon is ok.
     */
    daemonStatusIconName(daemon: KeaDaemon) {
        return daemonStatusIconName(daemon)
    }

    /**
     * Returns the color of the icon used when presenting daemon status
     *
     * @param daemon data structure holding the information about the daemon.
     *
     * @returns grey color if the daemon is not active, red if the daemon is
     *          active but there are communication issues, green if the
     *          communication with the active daemon is ok.
     */
    daemonStatusIconColor(daemon: KeaDaemon) {
        return daemonStatusIconColor(daemon)
    }

    /**
     * Returns error text to be displayed when there is a communication issue
     * with a given daemon
     *
     * @param daemon data structure holding the information about the daemon.
     *
     * @returns Error text. It includes hints about the communication
     *          problems when such problems occur, e.g. it includes the
     *          hint whether the communication is with the agent or daemon.
     */
    daemonStatusErrorText(daemon: KeaDaemon) {
        return daemonStatusIconTooltip(daemon)
    }

    /**
     * Changes the monitored state of the given daemon. It sends a request
     * to API.
     */
    changeMonitored(daemon: KeaDaemon) {
        const dmn = { monitored: !daemon.monitored }
        this.servicesApi.updateDaemon(daemon.id, dmn).subscribe(
            (/* data */) => {
                daemon.monitored = dmn.monitored
            },
            (/* err */) => {
                console.warn('Failed to update monitoring flag in daemon')
            }
        )
    }

    /** Returns true if the daemon was never running correctly. */
    isNeverFetchedDaemon(daemon: KeaDaemon) {
        return daemon.reloadedAt == null
    }

    /** Returns true if the daemon is DHCP daemon. */
    isDhcpDaemon(daemon: KeaDaemon) {
        return daemon.name === 'dhcp4' || daemon.name === 'dhcp6'
    }

    /**
     * Checks if the specified log target can be viewed
     *
     * Only the logs that are stored in the file can be viewed in Stork. The
     * logs output to stdout, stderr or syslog can't be viewed in Stork.
     *
     * @param target log target output location
     * @returns true if the log target can be viewed, false otherwise.
     */
    logTargetViewable(target: string): boolean {
        return target !== 'stdout' && target !== 'stderr' && !target.startsWith('syslog')
    }

    /**
     * Reacts to submitting a new app name from the dialog.
     *
     * This function is called when a user presses the rename button in
     * the app-rename-app-dialog component. It attempts to submit the new
     * name to the server.
     *
     * If the app is successfully renamed, the app name is refreshed in
     * the app tab view. Additionally, the success message is displayed
     * in the message service.
     *
     * @param event holds new app name.
     */
    handleRenameDialogSubmitted(name: string) {
        this.servicesApi.renameApp(this.appTab.app.id, { name: name }).subscribe(
            (/* data */) => {
                // Renaming the app was successful.
                this.msgService.add({
                    severity: 'success',
                    summary: 'App renamed',
                    detail: 'App successfully renamed to ' + name,
                })
                // Let's update the app name in the current tab.
                this.appTab.app.name = name
                // Notify the parent component about successfully renaming the app.
                this.renameApp.emit(name)
            },
            (err) => {
                // Renaming the app failed.
                const msg = getErrorMessage(err)
                this.msgService.add({
                    severity: 'error',
                    summary: 'Error renaming app',
                    detail: 'Error renaming app to ' + name + ': ' + msg,
                    life: 10000,
                })
            }
        )
    }

    /**
     * Reacts to hiding a dialog box for renaming an app.
     *
     * This function is called when a dialog box for renaming an app is
     * closed. It is triggered both in the case when the form is submitted
     * or cancelled.
     */
    handleRenameDialogHidden() {
        this.appRenameDialogVisible = false
    }

    /**
     * Shows a dialog for renaming an app.
     *
     * The dialog box component requires a set of machines' addresses
     * and a map of existing apps' names to validate the new app name.
     * Therefore, this function attempts to load the machines' addresses
     * and apps' names prior to displaying the dialog. If it fails, the
     * dialog box is not displayed.
     */
    showRenameAppDialog() {
        this.showRenameDialogClicked = true
        forkJoin([this.serverData.getAppsNames(), this.serverData.getMachinesAddresses()]).subscribe(
            (data) => {
                this.existingApps = data[0]
                this.existingMachines = data[1]
                this.appRenameDialogVisible = true
                this.showRenameDialogClicked = false
            },
            (/* err */) => {
                this.msgService.add({
                    severity: 'error',
                    summary: 'Failed to fetch apps and machines',
                    detail: 'Failed to fetch app and machine list from the server',
                    life: 10000,
                })
                this.showRenameDialogClicked = false
            }
        )
    }

    /**
     * Returns formatted filename for the file object returned by the server.
     *
     * @param file object containing the file type and file name returned by the
     *             server.
     * @param returns 'none' if file name is blank and it is not a lease file,
     *                'none (lease persistence disabled) if it is a lease file,
     *                original file name if it is not blank.
     */
    filenameFromFile(file: ModelFile) {
        if (!file.filename || file.filename.length === 0) {
            if (file.filetype === 'Lease file') {
                return 'none (lease persistence disabled)'
            } else {
                return 'none'
            }
        }
        return file.filename
    }

    /**
     * Returns formatted database name from type returned by the server.
     *
     * @param databaseType database type.
     * @returns 'MySQL', 'PostgreSQL', 'Cassandra' or 'Unknown'.
     */
    databaseNameFromType(databaseType: 'memfile' | 'mysql' | 'postgresql' | 'cql') {
        switch (databaseType) {
            case 'memfile':
                return 'Memfile'
            case 'mysql':
                return 'MySQL'
            case 'postgresql':
                return 'PostgreSQL'
            case 'cql':
                return 'Cassandra'
            default:
                break
        }
        return 'Unknown'
    }

    /**
     * Returns the base name of a path.
     *
     * @param path path to take the base name out of
     *
     * @returns base name
     */
    basename(path: string) {
        return path.split('/').pop()
    }

    /**
     * Returns an anchor used in the Kea documentation specific to the given hook library.
     *
     * @param hookLibrary basename of the hook library
     * @param keaVersion Kea version retrieved from the daemon
     *
     * @returns anchor or null if the hook library is not recognized
     */
    docAnchorFromHookLibrary(hookLibrary: string, keaVersion: string): string | null {
        const isNewVer = gte(keaVersion || '1.0.0', '2.4.0') // Kea versions >= 2.4 are considered new, where new anchors were introduced in ARM docs.
        if (!hookLibrary || !keaVersion || !(isNewVer || this.anchorsByHook[hookLibrary])) {
            // Return null:
            // - if hook name is not provided
            // - or if Kea version is not provided
            // - or if it is older Kea version (< 2.4) and there is no lookup value in the anchorsByHook for given hook.
            // For Kea version >= 2.4 lookup is not needed because new 'std-ischooklib-hook_name.so' anchors were
            // introduced in ARM.
            return null
        }
        const isPreRel = prerelease(keaVersion) != null // Will not be null for e.g. '2.5.4-git', will be null for '2.5.4'.
        const version = isPreRel ? 'latest' : 'kea-' + keaVersion
        const anchorId = isNewVer ? 'std-ischooklib-' + hookLibrary : this.anchorsByHook[hookLibrary]

        return version + '/arm/hooks.html#' + anchorId
    }

    /**
     * Copies provided hook path to the clipboard using Clipboard API.
     *
     * @param hookPath path to the hook library
     */
    copyHookPathToClipboard(hookPath: string): void {
        navigator.clipboard.writeText(hookPath).then(
            () => {
                // Copy hook path to the clipboard success.
                this.msgService.add({
                    severity: 'success',
                    summary: 'Hook path copied',
                    detail: 'Hook path ' + hookPath + ' was copied to the clipboard.',
                })
            },
            () => {
                // Copy hook path to the clipboard fail.
                this.msgService.add({
                    severity: 'error',
                    summary: 'Copy to clipboard failed',
                    detail: 'Hook path ' + hookPath + ' failed to copy to the clipboard.',
                    life: 10000,
                })
            }
        )
    }
}
