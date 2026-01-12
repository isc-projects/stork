import { Component, Input } from '@angular/core'
import { ActivatedRoute, RouterLink } from '@angular/router'
import { prerelease, gte } from 'semver'

import { MessageService } from 'primeng/api'

import { ServicesService } from '../backend'

import { durationToString, daemonStatusIconName, daemonStatusIconColor, daemonStatusIconTooltip } from '../utils'
import { KeaDaemon, ModelFile } from '../backend'
import { ManagedAccessDirective } from '../managed-access.directive'
import { NgIf, NgClass, NgFor } from '@angular/common'
import { Button } from 'primeng/button'
import { ToggleSwitch } from 'primeng/toggleswitch'
import { FormsModule } from '@angular/forms'
import { Message } from 'primeng/message'
import { Fieldset } from 'primeng/fieldset'
import { VersionStatusComponent } from '../version-status/version-status.component'
import { Tooltip } from 'primeng/tooltip'
import { TableModule } from 'primeng/table'
import { HelpTipComponent } from '../help-tip/help-tip.component'
import { ConfigReviewPanelComponent } from '../config-review-panel/config-review-panel.component'
import { HaStatusComponent } from '../ha-status/ha-status.component'
import { LocaltimePipe } from '../pipes/localtime.pipe'
import { PlaceholderPipe } from '../pipes/placeholder.pipe'

@Component({
    selector: 'app-kea-daemon',
    templateUrl: './kea-daemon.component.html',
    styleUrls: ['./kea-daemon.component.sass'],
    imports: [
        ManagedAccessDirective,
        NgIf,
        Button,
        ToggleSwitch,
        FormsModule,
        RouterLink,
        Message,
        NgClass,
        Fieldset,
        VersionStatusComponent,
        NgFor,
        Tooltip,
        TableModule,
        HelpTipComponent,
        ConfigReviewPanelComponent,
        HaStatusComponent,
        LocaltimePipe,
        PlaceholderPipe,
    ],
})
export class KeaDaemonComponent {
    @Input() daemon: KeaDaemon

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

    constructor(
        private route: ActivatedRoute,
        private servicesApi: ServicesService,
        private msgService: MessageService
    ) {}

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
     * @return true if there is a communication problem with the daemon,
     *         false otherwise.
     */
    get daemonStatusErred(): boolean {
        return (
            this.daemon.active &&
            (this.daemon.agentCommErrors ?? 0) + (this.daemon.caCommErrors ?? 0) + (this.daemon.daemonCommErrors ?? 0) >
                0
        )
    }

    /**
     * Returns the name of the icon to be used when presenting daemon status
     *
     * The icon selected depends on whether the daemon is active or not
     * active and whether there is a communication with the daemon or
     * not.
     *
     * @returns ban icon if the daemon is not active, times icon if the daemon
     *          should be active but the communication with it is broken and
     *          check icon if the communication with the active daemon is ok.
     */
    get daemonStatusIconName() {
        return daemonStatusIconName(this.daemon)
    }

    /**
     * Returns the color of the icon used when presenting daemon status
     *
     * @returns grey color if the daemon is not active, red if the daemon is
     *          active but there are communication issues, green if the
     *          communication with the active daemon is ok.
     */
    get daemonStatusIconColor() {
        return daemonStatusIconColor(this.daemon)
    }

    /**
     * Returns error text to be displayed when there is a communication issue
     * with a given daemon
     *
     * @returns Error text. It includes hints about the communication
     *          problems when such problems occur, e.g. it includes the
     *          hint whether the communication is with the agent or daemon.
     */
    get daemonStatusErrorText() {
        return daemonStatusIconTooltip(this.daemon)
    }

    /**
     * Changes the monitored state of the given daemon. It sends a request
     * to API.
     */
    changeMonitored() {
        const dmn = { monitored: !this.daemon.monitored }
        this.servicesApi.updateDaemon(this.daemon.id, dmn).subscribe(
            (/* data */) => {
                this.daemon.monitored = dmn.monitored
            },
            (/* err */) => {
                console.warn('Failed to update monitoring flag in daemon')
            }
        )
    }

    /** Returns true if the daemon was never running correctly. */
    get isNeverFetchedDaemon() {
        return this.daemon.reloadedAt == null
    }

    /** Returns true if the daemon is DHCP daemon. */
    get isDhcpDaemon() {
        return this.daemon.name === 'dhcp4' || this.daemon.name === 'dhcp6'
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
     * Returns formatted filename for the file object returned by the server.
     *
     * @param file object containing the file type and file name returned by the
     *             server.
     * @param returns 'default (persistence enabled)' if there is a default file storage
     *                'none (persistence disabled) if there is no file storage,
     *                original file name if it is a non-default file storage.
     */
    filenameFromFile(file: ModelFile) {
        if (!file.filename || file.filename.length === 0) {
            if (file.persist) {
                return 'default (persistence enabled)'
            } else {
                return 'none (persistence disabled)'
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
