import { Component, OnInit, Input, Output, EventEmitter } from '@angular/core'

import * as moment from 'moment-timezone'

import { MessageService, MenuItem } from 'primeng/api'

import {
    durationToString,
    daemonStatusErred,
    daemonStatusIconName,
    daemonStatusIconColor,
    daemonStatusIconTooltip,
} from '../utils'

@Component({
    selector: 'app-bind9-app-tab',
    templateUrl: './bind9-app-tab.component.html',
    styleUrls: ['./bind9-app-tab.component.sass'],
})
export class Bind9AppTabComponent implements OnInit {
    private _appTab: any
    @Output() refreshApp = new EventEmitter<number>()
    @Input() refreshedAppTab: any

    daemons: any[] = []

    appRenameDialogVisible = false

    constructor() {}

    /**
     * Subscribes to the updates of the information about daemons
     *
     * The information about the daemons may be updated as a result of
     * pressing the refresh button in the app tab. In such case, this
     * component emits an event to which the parent component reacts
     * and updates the daemons. When the daemons are updated, it
     * notifies this compoment via the subscription mechanism.
     */
    ngOnInit() {
        this.refreshedAppTab.subscribe((data) => {
            if (data) {
                this.initDaemon(data.app.details.daemon)
            }
        })
    }

    /**
     * Selects new application tab
     *
     * As a result, the local information about the daemons is updated.
     *
     * @param appTab pointer to the new app tab data structure.
     */
    @Input()
    set appTab(appTab) {
        this._appTab = appTab
        // Refresh local information about the daemon presented by this component.
        this.initDaemon(appTab.app.details.daemon)
    }

    /**
     * Returns information about currently selected app tab.
     */
    get appTab() {
        return this._appTab
    }

    /**
     * Initializes information about the daemon according to the information
     * carried in the provided parameter.
     *
     * As a result of invoking this function, the view of the component will be
     * updated.
     *
     * @param appTabDaemons information about the daemon stored in the app tab
     *                      data structure.
     */
    private initDaemon(appTabDaemon) {
        const daemonMap = []
        daemonMap[appTabDaemon.name] = appTabDaemon
        const DMAP = [['named', 'named']]
        const daemons = []
        for (const dm of DMAP) {
            if (daemonMap[dm[0]] !== undefined) {
                daemonMap[dm[0]].niceName = dm[1]
                daemonMap[dm[0]].statusErred = this.daemonStatusErred(daemonMap[dm[0]])
                daemons.push(daemonMap[dm[0]])
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
    showDuration(duration) {
        return durationToString(duration)
    }

    /**
     * Get cache effectiveness based on stats.
     * A percentage is returned as floored int.
     */
    getQueryUtilization(daemon) {
        let utilization = 0.0
        if (!daemon.queryHitRatio) {
            return utilization
        }
        utilization = 100 * daemon.queryHitRatio
        return Math.floor(utilization)
    }

    /**
     * Returns boolean value indicating if there is an issue with communication
     * with the given daemon
     *
     * @param daemon data structure holding the information about the daemon.
     *
     * @return true if there is a communication problem with the daemon,
     *         false otherwise.
     */
    private daemonStatusErred(daemon): boolean {
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
     *          should be active but the communication with it is borken and
     *          check icon if the communication with the active daemon is ok.
     */
    daemonStatusIconName(daemon) {
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
    daemonStatusIconColor(daemon) {
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
    daemonStatusErrorText(daemon) {
        return daemonStatusIconTooltip(daemon)
    }

    /**
     * Reacts to submitting a new app name from the dialog.
     *
     * This function is called when a user presses the rename button in
     * the app-rename-app-dialog component. It attempts to submit the new
     * name to the server.
     */
    handleRenameDialogSubmitted() {
        this.appRenameDialogVisible = false
    }

    /**
     * Reacts to cancelling renaming an app.
     *
     * This function is called when a user presses the cancel button in
     * the app-rename-app-dialog component. It marks the dialog hidden.
     */
    handleRenameDialogCancelled() {
        this.appRenameDialogVisible = false
    }

    /**
     * Shows a dialog for renaming an app.
     */
    renameApp() {
        this.appRenameDialogVisible = true
    }
}
