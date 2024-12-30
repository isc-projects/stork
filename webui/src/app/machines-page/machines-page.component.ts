import {AfterViewInit, Component, OnDestroy, OnInit, ViewChild} from '@angular/core'
import {ActivatedRoute, EventType, ParamMap, Router} from '@angular/router'

import { MessageService, MenuItem } from 'primeng/api'
import {concat, EMPTY, Observable, Subscription} from 'rxjs'
import { Machine, Settings } from '../backend'

import { ServicesService, SettingsService } from '../backend/api/api'
import { ServerDataService } from '../server-data.service'
import { copyToClipboard, getErrorMessage } from '../utils'
import { Table } from 'primeng/table'
import {catchError, filter} from "rxjs/operators";
import {AuthorizedMachinesTableComponent} from "../authorized-machines-table/authorized-machines-table.component";

interface AppType {
    name: string
    value: string
    id: string
}

@Component({
    selector: 'app-machines-page',
    templateUrl: './machines-page.component.html',
    styleUrls: ['./machines-page.component.sass'],
})
export class MachinesPageComponent implements OnInit, OnDestroy, AfterViewInit {
    private subscriptions = new Subscription()
    breadcrumbs = [{ label: 'Services' }, { label: 'Machines' }]

    // machines table
    machines: Machine[]
    totalMachines: number
    machineMenuItems: MenuItem[]
    machineMenuItemsAuth: MenuItem[]
    machineMenuItemsUnauth: MenuItem[]
    viewSelectionOptions: any[]
    showUnauthorized = false
    serverToken = ''
    selectedMachines: Machine[] = []
    dataLoading: boolean
    stateKey = 'machines-table-session'

    // This counter is used to indicate in UI that there are some
    // unauthorized machines that may require authorization.
    unauthorizedMachinesCount = 0

    // action panel
    appTypes: AppType[]
    selectedAppType: AppType

    // edit machine address
    changeMachineAddressDlgVisible = false
    machineAddress = 'localhost'
    agentPort = ''

    // machine tabs
    activeTabIdx = 0
    tabs: MenuItem[]
    activeItem: MenuItem
    openedMachines: { machine: Machine }[]
    machineTab: { machine: Machine }

    displayAgentInstallationInstruction = false

    // Indicates if the machines registration is administratively disabled.
    registrationDisabled = false

    @ViewChild('authorizedMachinesTableComponent') authorizedMachinesTable: AuthorizedMachinesTableComponent

    constructor(
        private route: ActivatedRoute,
        private router: Router,
        private servicesApi: ServicesService,
        private msgSrv: MessageService,
        private serverData: ServerDataService,
        private settingsService: SettingsService
    ) {}

    ngOnDestroy(): void {
        this.subscriptions.unsubscribe()
    }

    /**
     * Component lifecycle hook called after Angular completed the initialization of the
     * component's view.
     *
     * We subscribe to router events to act upon URL and/or queryParams changes.
     * This is done at this step, because we have to be sure that all child components,
     * especially PrimeNG table in SubnetsTableComponent, are initialized.
     */
    ngAfterViewInit(): void {
        this.router.events
            .pipe(
                filter((event, idx) => idx === 0 || event.type === EventType.NavigationEnd),
                catchError((err) => {
                    const msg = getErrorMessage(err)
                    this.msgSrv.add({
                        severity: 'error',
                        summary: 'Cannot process the URL query',
                        detail: msg,
                        life: 10000,
                    })
                    return EMPTY
                })
            )
            .subscribe(() => {
                const paramMap = this.route.snapshot.paramMap
                const queryParamMap = this.route.snapshot.queryParamMap

                // Apply to the changes of the subnet id, e.g. from /dhcp/subnets/all to
                // /dhcp/subnets/1. Those changes are triggered by switching between the
                // tabs.

                // Get subnet id.
                const id = paramMap.get('id')
                if (!id || id === 'all' || id === 'authorized' || id === 'unauthorized') {
                    // Update the filter only if the target is subnet list.
                    this.showUnauthorized = id === 'unauthorized'
                    this.authorizedMachinesTable?.updateFilterFromQueryParameters(queryParamMap)
                    this.switchToTab(0)
                    return
                }
                const numericId = parseInt(id, 10)
                if (!Number.isNaN(numericId)) {
                    // The path has a numeric id indicating that we should
                    // open a tab with selected subnet information or switch
                    // to this tab if it has been already opened.
                    // this.openTabBySubnetId(numericId)
                    let found = false
                    // if tab for this machine is already opened then switch to it
                    for (let idx = 0; idx < this.openedMachines.length; idx++) {
                        const m = this.openedMachines[idx].machine
                        if (m.id === numericId) {
                            this.switchToTab(idx + 1)
                            found = true
                        }
                    }

                    // if tab is not opened then search for list of machines if the one is present there,
                    // if so then open it in new tab and switch to it
                    if (!found) {
                        for (const m of this.machines) {
                            if (m.id === numericId) {
                                this.addMachineTab(m)
                                this.switchToTab(this.tabs.length - 1)
                                found = true
                                break
                            }
                        }
                    }

                    // if machine is not loaded in list fetch it individually
                    if (!found) {
                        this.servicesApi.getMachine(numericId).subscribe(
                            (data) => {
                                this.addMachineTab(data)
                                this.switchToTab(this.tabs.length - 1)
                            },
                            (err) => {
                                const msg = getErrorMessage(err)
                                this.msgSrv.add({
                                    severity: 'error',
                                    summary: 'Cannot get machine',
                                    detail: 'Failed to get machine with ID ' + numericId + ': ' + msg,
                                    life: 10000,
                                })
                                this.navigateToMachinesList()
                            }
                        )
                    }
                } else {
                    // In case of failed Id parsing, open list tab.
                    this.switchToTab(0)
                    this.authorizedMachinesTable?.loadDataWithoutFilter()
                }
            })
    }

    /** Switches to tab with the given index. */
    switchToTab(index: number) {
        if (this.activeTabIdx === index) {
            return
        }
        this.activeTabIdx = index
        this.activeItem = this.tabs[index]
        if (index > 0) {
            this.machineTab = this.openedMachines[index - 1]
        }
    }

    /** Add a new machine tab. */
    addMachineTab(machine: Machine) {
        this.openedMachines.push({
            machine,
        })
        this.tabs = [
            ...this.tabs,
            {
                label: machine.address,
                id: 'machine-tab' + machine.id,
                routerLink: '/machines/' + machine.id,
            },
        ]
    }

    ngOnInit() {
        this.tabs = [{ label: 'Machines', id: 'all-machines-tab', routerLink: '/machines/authorized' }]

        this.machines = []
        this.appTypes = [
            { name: 'any', value: '', id: 'none-app' },
            { name: 'Bind9', value: 'bind9', id: 'bind-app' },
            { name: 'Kea', value: 'kea', id: 'kea-app' },
        ]
        this.machineMenuItemsAuth = [
            {
                label: 'Refresh',
                id: 'refresh-single-machine',
                icon: 'pi pi-refresh',
            },
            {
                label: 'Dump troubleshooting data',
                id: 'dump-single-machine',
                icon: 'pi pi-download',
                title: 'Download data archive for troubleshooting purposes',
            },
            /* Temporarily disable unauthorization until we find an
               actual use case for it. Also, if we allow unauthorization
               we will have to fix several things, e.g. apps belonging
               to an unauthorized machine will have to disappear.
               For now, a user can simply remove a machine.
            {
                label: 'Unauthorize',
                id: 'unauthorize-single-machine',
                icon: 'pi pi-minus-circle',
            }, */
            {
                label: 'Remove',
                id: 'remove-single-machine',
                icon: 'pi pi-times',
                title: 'Remove machine from Stork Server',
            },
        ]
        this.machineMenuItemsUnauth = [
            {
                label: 'Authorize',
                id: 'authorize-single-machine',
                icon: 'pi pi-check',
            },
            {
                label: 'Remove',
                id: 'remove-single-machine',
                icon: 'pi pi-times',
                title: 'Remove machine from Stork Server',
            },
        ]
        this.machineMenuItems = this.machineMenuItemsAuth

        // Add a select button to switch between authorized and
        // unauthorized machines.
        this.viewSelectionOptions = [
            {
                label: 'Authorized',
                value: false,
            },
            {
                label: 'Unauthorized (0)',
                value: true,
            },
        ]

        this.openedMachines = []

        this.subscriptions.add(
            this.route.paramMap.subscribe((params: ParamMap) => {
                const machineIdStr = params.get('id')
                if (machineIdStr === 'authorized' || machineIdStr === 'unauthorized') {
                    this.showUnauthorized = machineIdStr === 'unauthorized'
                    this.switchToTab(0)
                } else {
                    const machineId = parseInt(machineIdStr, 10)

                    let found = false
                    // if tab for this machine is already opened then switch to it
                    for (let idx = 0; idx < this.openedMachines.length; idx++) {
                        const m = this.openedMachines[idx].machine
                        if (m.id === machineId) {
                            this.switchToTab(idx + 1)
                            found = true
                        }
                    }

                    // if tab is not opened then search for list of machines if the one is present there,
                    // if so then open it in new tab and switch to it
                    if (!found) {
                        for (const m of this.machines) {
                            if (m.id === machineId) {
                                this.addMachineTab(m)
                                this.switchToTab(this.tabs.length - 1)
                                found = true
                                break
                            }
                        }
                    }

                    // if machine is not loaded in list fetch it individually
                    if (!found) {
                        this.servicesApi.getMachine(machineId).subscribe(
                            (data) => {
                                this.addMachineTab(data)
                                this.switchToTab(this.tabs.length - 1)
                            },
                            (err) => {
                                const msg = getErrorMessage(err)
                                this.msgSrv.add({
                                    severity: 'error',
                                    summary: 'Cannot get machine',
                                    detail: 'Failed to get machine with ID ' + machineId + ': ' + msg,
                                    life: 10000,
                                })
                                this.navigateToMachinesList()
                            }
                        )
                    }
                }
            })
        )

        // Settings are needed to check whether or not the machines registration is disabled.
        this.subscriptions.add(
            this.settingsService.getSettings().subscribe({
                next: (settings: Settings) => {
                    this.registrationDisabled = !settings.enableMachineRegistration
                },
                error: (err) => {
                    const msg = getErrorMessage(err)
                    this.msgSrv.add({
                        severity: 'error',
                        summary: 'Cannot get settings',
                        detail: 'Failed to get settings: ' + msg,
                        life: 10000,
                    })
                },
            })
        )

        this.dataLoading = true

        // check current number of unauthorized machines
        this.refreshUnauthorizedMachinesCount()
    }

    /**
     * Refresh count of unauthorized machines.
     *
     * This counter is used to indicate in UI that there are some
     * unauthorized machines that may require authorization.
     */
    refreshUnauthorizedMachinesCount() {
        if (this.showUnauthorized) {
            return
        }
        this.servicesApi.getUnauthorizedMachinesCount().subscribe((count: number) => {
            this.unauthorizedMachinesCount = count
            this.viewSelectionOptions[1].label = 'Unauthorized (' + count + ')'

            // force refresh in UI
            this.viewSelectionOptions = [...this.viewSelectionOptions]
        })
    }

    /**
     * Handler called by the PrimeNG table to load the machine data.
     * @param event Pagination event
     */
    loadMachines(event) {
        this.dataLoading = true
        let text
        if (event.filters?.hasOwnProperty('text')) {
            text = event.filters.text.value
        }

        let app
        if (event.filters?.app?.[0]) {
            app = event.filters.app[0].value
        }

        this.servicesApi.getMachines(event.first, event.rows, text, app, !this.showUnauthorized).subscribe((data) => {
            this.machines = data.items ?? []
            const total = data.total || 0
            this.totalMachines = total
            if (this.showUnauthorized) {
                this.unauthorizedMachinesCount = total
                this.viewSelectionOptions[1].label = 'Unauthorized (' + total + ')'

                // force refresh in UI
                this.viewSelectionOptions = [...this.viewSelectionOptions]
            }
            this.dataLoading = false
        })
        this.refreshUnauthorizedMachinesCount()
    }

    /** Callback called on canceling the edit machine dialog. */
    cancelMachineDialog() {
        this.changeMachineAddressDlgVisible = false
    }

    /** Callback called on key pressed in the edit machine dialog. */
    keyUpMachineDlg(event: KeyboardEvent, machineTab) {
        if (event.key === 'Enter') {
            if (this.changeMachineAddressDlgVisible) {
                this.saveMachine(machineTab)
            }
        }
    }

    /** Callback called on clicking the refresh button. */
    refreshMachinesList(machinesTable: Table) {
        machinesTable.onLazyLoad.emit(machinesTable.createLazyLoadMetadata())
    }

    /**
     * Callback called when toggling between authorized and unauthorized
     * machines.
     */
    onSelectMachinesListChange(machinesTable: Table) {
        this.navigateToMachinesList()
        machinesTable.onLazyLoad.emit(machinesTable.createLazyLoadMetadata())
    }

    /**
     * Callback called on input event emitted by the filter input box.
     *
     * @param table table on which the filtering will apply
     * @param filterText text value of the filter input
     * @param force force filtering for shorter lookup keywords
     */
    inputFilterText(table: Table, filterText: string, force: boolean = false) {
        if (filterText.length >= 3 || (force && filterText != '')) {
            table.filter(filterText, 'text', 'contains')
        } else if (filterText.length == 0) {
            this.clearFilters(table)
        }
    }

    /**
     * Filters the displayed data by application ID.
     */
    filterByApp(machinesTable: Table) {
        machinesTable.filter(this.selectedAppType.value, 'app', 'equals')
    }

    /** Closes a tab with the given index. */
    closeTab(event: PointerEvent, idx: number) {
        this.openedMachines.splice(idx - 1, 1)
        this.tabs = [...this.tabs.slice(0, idx), ...this.tabs.slice(idx + 1)]
        if (this.activeTabIdx === idx) {
            this.switchToTab(idx - 1)
            if (idx - 1 > 0) {
                this.router.navigate(['/machines/' + this.machineTab.machine.id])
            } else {
                this.navigateToMachinesList()
            }
        } else if (this.activeTabIdx > idx) {
            this.activeTabIdx = this.activeTabIdx - 1
        }
        if (event) {
            event.preventDefault()
        }
    }

    /** Fetches new machine state from API. */
    _refreshMachineState(machine: Machine) {
        this.servicesApi.getMachineState(machine.id).subscribe(
            (data) => {
                if (data.error) {
                    this.msgSrv.add({
                        severity: 'error',
                        summary: 'Error getting machine state',
                        detail: 'Error getting state of machine: ' + data.error,
                        life: 10000,
                    })
                } else {
                    this.msgSrv.add({
                        severity: 'success',
                        summary: 'Machine refreshed',
                        detail: 'Refreshing succeeded.',
                    })
                }

                // refresh machine in machines list
                for (let i = 0; i < this.machines.length; i++) {
                    if (this.machines[i].id === data.id) {
                        this.machines[i] = data
                        break
                    }
                }

                // refresh machine in opened tab if present
                for (let i = 0; i < this.openedMachines.length; i++) {
                    if (this.openedMachines[i].machine.id === data.id) {
                        this.openedMachines[i].machine = data
                        break
                    }
                }
            },
            (err) => {
                const msg = getErrorMessage(err)
                this.msgSrv.add({
                    severity: 'error',
                    summary: 'Error getting machine state',
                    detail: 'Error getting state of machine: ' + msg,
                    life: 10000,
                })
            }
        )
    }

    /**
     * Start downloading the dump file.
     */
    downloadDump(machine: Machine) {
        window.location.href = `api/machines/${machine.id}/dump`
    }

    /**
     * Authorize or unauthorize machine.
     *
     * @param machine machine object
     * @param authorized bool, true or false
     * @param machinesTable table where authorized/unauthorized machine belong
     */
    _changeMachineAuthorization(machine: Machine, authorized: boolean, machinesTable: Table) {
        // Block table UI when machine authorization is in progress.
        this.dataLoading = true
        const stateBackup = machine.authorized

        machine.authorized = authorized
        const prefix = authorized ? '' : 'un'
        this.servicesApi.updateMachine(machine.id, machine).subscribe({
            next: (m) => {
                this.msgSrv.add({
                    severity: 'success',
                    summary: `Machine ${prefix}authorized`,
                    detail: `Machine ${m.address} ${prefix}authorization succeeded.`,
                })
                this.refreshMachinesList(machinesTable)
                // Force menu adjustments to take into account that there
                // is new machine and apps available.
                this.serverData.forceReloadAppsStats()
            },
            error: (err) => {
                machine.authorized = stateBackup
                this.dataLoading = false
                const msg = getErrorMessage(err)
                this.msgSrv.add({
                    severity: 'error',
                    summary: `Machine ${prefix}authorization failed`,
                    detail: `Machine ${prefix}authorization attempt failed: ${msg}`,
                    life: 10000,
                })
            },
            complete: () => {
                this.dataLoading = false
            },
        })
    }

    /**
     * Shows menu with actions possible on a given machine. Currently this is
     * authorize/unauthorize or delete. It is called every time the user switches
     * between authorized/unauthorized view.
     *
     * @param event browser event generated when the button is clicked causing
     *        the menu to be toggled
     * @param machineMenu reference to the DOM object that represents the machine menu
     * @param machine reference to a machine
     * @param machinesTable reference to the table with machines
     */
    showMachineMenu(event, machineMenu, machine, machinesTable) {
        if (this.showUnauthorized) {
            this.machineMenuItems = this.machineMenuItemsUnauth
        } else {
            this.machineMenuItems = this.machineMenuItemsAuth
        }

        machineMenu.toggle(event)

        if (this.showUnauthorized) {
            // connect method to authorize machine
            this.machineMenuItems[0].command = () => {
                this._changeMachineAuthorization(machine, true, machinesTable)
            }

            // connect method to delete machine
            this.machineMenuItems[1].command = () => {
                this.deleteMachine(machine.id)
            }
        } else {
            // connect method to refresh machine state
            this.machineMenuItems[0].command = () => {
                this._refreshMachineState(machine)
            }

            // connect method to dump machine configuration
            this.machineMenuItems[1].command = () => {
                this.downloadDump(machine)
            }

            // connect method to authorize machine
            /*this.machineMenuItems[2].command = () => {
                this._changeMachineAuthorization(machine, false, machinesTable)
            }*/

            // connect method to delete machine
            this.machineMenuItems[2].command = () => {
                this.deleteMachine(machine.id)
            }
        }
    }

    /**
     * Delete indicated machine.
     *
     * Additionally app stats will be reloaded and if after deletion
     * there is no more DHCP or DNS apps then the item in the top menu
     * is adjusted.
     *
     * @param machineId ID of machine
     */
    deleteMachine(machineId) {
        this.servicesApi.deleteMachine(machineId).subscribe((/* data */) => {
            // reload apps stats to reflect new state (adjust menu content)
            this.serverData.forceReloadAppsStats()

            // remove from list of machines
            for (let idx = 0; idx < this.machines.length; idx++) {
                const m = this.machines[idx]
                if (m.id === machineId) {
                    this.machines.splice(idx, 1) // TODO: does not work
                    break
                }
            }
            // remove from opened tabs if present
            for (let idx = 0; idx < this.openedMachines.length; idx++) {
                const m = this.openedMachines[idx].machine
                if (m.id === machineId) {
                    this.closeTab(null, idx + 1)
                    break
                }
            }
        })
    }

    /** Sets the edit form-related members using the value of the current machine. */
    editAddress(machineTab) {
        this.machineAddress = machineTab.machine.address
        this.agentPort = machineTab.machine.agentPort.toString() // later string is expected in this.agentPort
        this.changeMachineAddressDlgVisible = true
    }

    /** Alters a given machine in API. */
    saveMachine(machineTab) {
        if (this.machineAddress === machineTab.machine.address && this.agentPort === machineTab.machine.agentPort) {
            machineTab.changeMachineAddressDlgVisible = false
            return
        }
        const m = { address: this.machineAddress, agentPort: parseInt(this.agentPort, 10) }
        this.servicesApi.updateMachine(machineTab.machine.id, m).subscribe(
            (data) => {
                machineTab.machine = data
                this.changeMachineAddressDlgVisible = false
                this.msgSrv.add({
                    severity: 'success',
                    summary: 'Machine address updated',
                    detail: 'Machine address update succeeded.',
                })

                this._refreshMachineState(machineTab.machine)
            },
            (err) => {
                const msg = getErrorMessage(err)
                this.msgSrv.add({
                    severity: 'error',
                    summary: 'Machine address update failed',
                    detail: 'Error updating machine address: ' + msg,
                    life: 10000,
                })
            }
        )
    }

    /**
     * Callback called on machine tab click.
     */
    refreshMachineState(machinesTab) {
        this._refreshMachineState(machinesTab.machine)
    }

    /**
     * Display a dialog with instructions about installing
     * stork agent.
     */
    showAgentInstallationInstruction() {
        this.servicesApi.getMachinesServerToken().subscribe(
            (data) => {
                this.serverToken = data.token
                this.displayAgentInstallationInstruction = true
            },
            (err) => {
                const msg = getErrorMessage(err)
                this.msgSrv.add({
                    severity: 'error',
                    summary: 'Cannot get server token',
                    detail: 'Error getting server token to register machines: ' + msg,
                    life: 10000,
                })
            }
        )
    }

    /**
     * Close the dialog with instructions about installing
     * stork agent.
     */
    closeAgentInstallationInstruction() {
        this.displayAgentInstallationInstruction = false
    }

    /**
     * Send request to stork server to regenerate machines server token.
     */
    regenerateServerToken() {
        this.servicesApi.regenerateMachinesServerToken().subscribe(
            (data) => {
                this.serverToken = data.token
            },
            (err) => {
                const msg = getErrorMessage(err)
                this.msgSrv.add({
                    severity: 'error',
                    summary: 'Cannot regenerate server token',
                    detail: 'Error regenerating server token to register machines: ' + msg,
                    life: 10000,
                })
            }
        )
    }

    /**
     * Return base URL of Stork server website.
     * It is then put into agent installation instructions.
     */
    getBaseUrl() {
        return window.location.origin
    }

    /**
     * Copies selected text to clipboard. See @ref copyToClipboard for details.
     */
    copyToClipboard(textEl) {
        return copyToClipboard(textEl)
    }

    /**
     * Authorizes machines stored in selectedMachines.
     *
     * @param table table where selected machines are to be authorized.
     */
    authorizeSelectedMachines(table: Table) {
        // Calling servicesApi.updateMachine() API sequentially for all selected machines.
        // Max expected count of selected machines is max machines per table page,
        // which currently is 50.
        const updateObservables: Observable<Machine>[] = []
        for (const m of this.selectedMachines) {
            m.authorized = true
            updateObservables.push(this.servicesApi.updateMachine(m.id, m))
        }

        // Use concat to call servicesApi sequentially.
        const authorizations$: Observable<Machine> = concat(...updateObservables)

        // Block table UI when bulk machines authorization is in progress.
        this.dataLoading = true
        authorizations$.subscribe({
            next: (m) => {
                this.msgSrv.add({
                    severity: 'success',
                    summary: 'Machine authorized',
                    detail: `Machine ${m.address} authorization succeeded.`,
                })
            },
            error: (err) => {
                const msg = getErrorMessage(err)
                this.msgSrv.add({
                    severity: 'error',
                    summary: 'Machine authorization failed',
                    detail: 'Machine authorization attempt failed: ' + msg,
                    life: 10000,
                })
                this.dataLoading = false
                this.refreshMachinesList(table)
                // Force menu adjustments to take into account that there
                // is new machine and apps available.
                this.serverData.forceReloadAppsStats()
            },
            complete: () => {
                this.dataLoading = false
                this.refreshMachinesList(table)
                // Force menu adjustments to take into account that there
                // is new machine and apps available.
                this.serverData.forceReloadAppsStats()
            },
        })

        // Clear selection after.
        this.selectedMachines = []

        // Force clear selection in session storage.
        let state = JSON.parse(sessionStorage.getItem(this.stateKey))
        state.selection = []
        sessionStorage.setItem(this.stateKey, JSON.stringify(state))
    }

    /**
     * Callback called when PrimeNG table state is restored.
     *
     * @param state restored table state
     */
    stateRestored(state: any) {
        // Do not restore selection.
        state.selection = []
    }

    /**
     * Clears filtering on given table.
     *
     * @param table table where filtering is to be cleared
     */
    clearFilters(table: Table) {
        table.filter(null, 'text', 'contains')
    }

    /**
     * Navigates to the URL displaying authorized or unauthorized machines.
     *
     * Depending on whether a user requested listing authorized or unauthorized
     * machines the router link is different. This function updates the router
     * link accordingly. It also updates the link of the first tab, so it matches
     * the current router link.
     */
    navigateToMachinesList() {
        const currentLink = this.showUnauthorized ? '/machines/unauthorized' : '/machines/authorized'
        if (this.tabs.length > 0) {
            this.tabs[0].routerLink = currentLink
        }
        this.router.navigate([currentLink])
    }
}
