import { Component, OnDestroy, OnInit } from '@angular/core'
import { ActivatedRoute, ParamMap, Router } from '@angular/router'

import { MessageService, MenuItem } from 'primeng/api'
import { Subscription } from 'rxjs'
import { Machine } from '../backend'

import { ServicesService } from '../backend/api/api'
import { ServerDataService } from '../server-data.service'
import { copyToClipboard, getErrorMessage } from '../utils'
import { Table } from 'primeng/table'

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
export class MachinesPageComponent implements OnInit, OnDestroy {
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

    constructor(
        private route: ActivatedRoute,
        private router: Router,
        private servicesApi: ServicesService,
        private msgSrv: MessageService,
        private serverData: ServerDataService
    ) {}

    ngOnDestroy(): void {
        this.subscriptions.unsubscribe()
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
        this.tabs = [...this.tabs, {
            label: machine.address,
            id: 'machine-tab' + machine.id,
            routerLink: '/machines/' + machine.id,
        }]
    }

    ngOnInit() {
        this.tabs = [{ label: 'Machines', id: 'all-machines-tab', routerLink: '/machines/all' }]

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
                if (machineIdStr === 'all') {
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
                                this.router.navigate(['/machines/all'])
                            }
                        )
                    }
                }
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
        this.servicesApi.getMachines(0, 1, null, null, false).subscribe((data) => {
            const total = data.total || 0
            this.unauthorizedMachinesCount = total
            this.viewSelectionOptions[1].label = 'Unauthorized (' + total + ')'

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
        if (event.filters?.text) {
            text = event.filters.text.value
        }

        let app
        if (event.filters?.app) {
            app = event.filters.app.value
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
     * Callback called on input event emitted by the filter input box.
     *
     * @param table table on which the filtering will apply
     * @param filterTxt text value of the filter input
     */
    inputFilterText(table: Table, filterTxt?: string) {
        if (filterTxt.length >= 3) {
            table.filter(filterTxt, 'text', 'contains')
        } else if (filterTxt.length == 0) {
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
                this.router.navigate(['/machines/all'])
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
     */
    _changeMachineAuthorization(machine, authorized, machinesTable) {
        machine.authorized = authorized
        const txt = 'Machine ' + (authorized ? '' : 'un')
        this.servicesApi.updateMachine(machine.id, machine).subscribe(
            (/* data */) => {
                this.msgSrv.add({
                    severity: 'success',
                    summary: txt + 'authorized',
                    detail: txt + 'authorization succeeded.',
                })
                this.refreshMachinesList(machinesTable)
                // Force menu adjustments to take into account that there
                // is new machine and apps available.
                this.serverData.forceReloadAppsStats()
            },
            (err) => {
                const msg = getErrorMessage(err)
                this.msgSrv.add({
                    severity: 'error',
                    summary: txt + 'authorization failed',
                    detail: txt + 'authorization attempt failed: ' + msg,
                    life: 10000,
                })
            }
        )
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
    authorizeSelectedMachines(table) {
        // Calling _changeMachineAuthorization sequentially for all selected machines.
        // Max expected count of selected machines is max machines per table page,
        // which currently is 50.
        for (const m of this.selectedMachines) {
            this._changeMachineAuthorization(m, true, table)
        }
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
}
