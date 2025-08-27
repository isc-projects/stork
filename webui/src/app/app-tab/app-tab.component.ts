import { Component, Input, Output, EventEmitter } from '@angular/core'

import { forkJoin, lastValueFrom } from 'rxjs'

import { MessageService } from 'primeng/api'

import { ServicesService } from '../backend/api/api'
import { ServerDataService } from '../server-data.service'

import {
    daemonNameToFriendlyName,
    daemonStatusErred,
    daemonStatusIconName,
    daemonStatusIconTooltip,
    getErrorMessage,
} from '../utils'
import { AppTab } from '../apps'
import { Bind9Daemon } from '../backend'

type DaemonInfo = Bind9Daemon & {
    statusErred: boolean
    niceName?: string
    icon?: string
}

@Component({
    selector: 'app-app-tab',
    templateUrl: './app-tab.component.html',
    styleUrls: ['./app-tab.component.sass'],
})
export class AppTabComponent {
    private _appTab: AppTab
    /**
     * Event emitter sending an event to the parent component when the app is
     * refreshed.
     */
    @Output() refreshApp = new EventEmitter<number>()

    /**
     * Event emitter sending an event to the parent component when an app is
     * renamed.
     */
    @Output() renameApp = new EventEmitter<string>()

    /**
     * Information about the daemons.
     */
    daemons: DaemonInfo[] = []

    /**
     * Active tab index used by the tab view.
     */
    // activeTabIndex = 0

    /**
     * Holds a map of existing apps' names and ids.
     *
     * The apps' names are used in rename-app-dialog component to validate
     * the user input.
     */
    existingApps: any = []

    /**
     * Holds a set of existing machines' addresses.
     *
     * The machines' addresses are used in rename-app-dialog component to
     * validate the user input.
     */
    existingMachines: any = []

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

    constructor(
        private servicesApi: ServicesService,
        private serverData: ServerDataService,
        private msgService: MessageService
    ) {}

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
        const daemon = appTab.app.type === 'bind9' ? appTab.app.details.daemon : appTab.app.details.pdnsDaemon
        // Refresh local information about the daemon presented by this component.
        this.daemons = [
            {
                statusErred: this.daemonStatusErred(daemon),
                niceName: daemonNameToFriendlyName(daemon.name),
                icon: daemonStatusIconName(daemon),
                ...daemon,
            },
        ]
    }

    /**
     * Returns information about currently selected app tab.
     */
    get appTab(): AppTab {
        return this._appTab
    }

    /**
     * An action triggered when refresh button is pressed.
     */
    refreshAppState() {
        this.refreshApp.emit(this._appTab.app.id)
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
    private daemonStatusErred(daemon: Bind9Daemon): boolean {
        return daemon.active && daemonStatusErred(daemon)
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
    daemonStatusIconName(daemon: Bind9Daemon) {
        return daemonStatusIconName(daemon)
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
    daemonStatusErrorText(daemon: Bind9Daemon) {
        return daemonStatusIconTooltip(daemon)
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
    handleRenameDialogSubmitted(event) {
        this.appRenameDialogVisible = false
        lastValueFrom(this.servicesApi.renameApp(this.appTab.app.id, { name: event }))
            .then((/* data */) => {
                // Renaming the app was successful.
                this.msgService.add({
                    severity: 'success',
                    summary: 'App renamed',
                    detail: 'App successfully renamed to ' + event,
                })
                // Let's update the app name in the current tab.
                this.appTab.app.name = event
                // Notify the parent component about successfully renaming the app.
                this.renameApp.emit(event)
            })
            .catch((err) => {
                // Renaming the app failed.
                const msg = getErrorMessage(err)
                this.msgService.add({
                    severity: 'error',
                    summary: 'Error renaming app',
                    detail: 'Error renaming app to ' + event + msg,
                    life: 10000,
                })
            })
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
        lastValueFrom(forkJoin([this.serverData.getAppsNames(), this.serverData.getMachinesAddresses()]))
            .then((data) => {
                this.existingApps = data[0]
                this.existingMachines = data[1]
                this.appRenameDialogVisible = true
                this.showRenameDialogClicked = false
            })
            .catch((/* err */) => {
                this.msgService.add({
                    severity: 'error',
                    summary: 'Fetching apps and machines failed',
                    detail: 'Failed to fetch apps and machines list from the server',
                    life: 10000,
                })
                this.showRenameDialogClicked = false
            })
    }
}
