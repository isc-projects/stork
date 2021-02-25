import { Component, OnInit } from '@angular/core'
import { ActivatedRoute, ParamMap, Router, NavigationEnd } from '@angular/router'

import { MessageService, MenuItem } from 'primeng/api'

import { ServicesService } from '../backend/api/api'
import { LoadingService } from '../loading.service'
import { ServerDataService } from '../server-data.service'
import { copyToClipboard } from '../utils'

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
export class MachinesPageComponent implements OnInit {
    breadcrumbs = [{ label: 'Services' }, { label: 'Machines' }]

    // machines table
    machines: any[]
    totalMachines: number
    machineMenuItems: MenuItem[]
    machineMenuItemsAuth: MenuItem[]
    machineMenuItemsUnauth: MenuItem[]
    showUnauthorized = false
    serverToken = ''

    // This counter is used to indicate in UI that there are some
    // unauthorized machines that may require authorization.
    unauthorizedMachinesCount = 0

    // action panel
    filterText = ''
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
    openedMachines: any
    machineTab: any

    displayAgentInstallationInstruction = false

    constructor(
        private route: ActivatedRoute,
        private router: Router,
        private servicesApi: ServicesService,
        private msgSrv: MessageService,
        private serverData: ServerDataService,
        private loadingService: LoadingService
    ) {}

    switchToTab(index) {
        if (this.activeTabIdx === index) {
            return
        }
        this.activeTabIdx = index
        this.activeItem = this.tabs[index]
        if (index > 0) {
            this.machineTab = this.openedMachines[index - 1]
        }
    }

    addMachineTab(machine) {
        this.openedMachines.push({
            machine,
        })
        this.tabs.push({
            label: machine.address,
            id: 'machine-tab' + machine.id,
            routerLink: '/machines/' + machine.id,
        })
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
            /* Temporarily disable unauthorization until we find an
               actual use case for it. Also, if we allow unauthorization
               we will have to fix several things, e.g. apps belonging
               to an unathorized machine will have to disappear.
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

        this.openedMachines = []

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
                            let msg = err.statusText
                            if (err.error && err.error.message) {
                                msg = err.error.message
                            }
                            this.msgSrv.add({
                                severity: 'error',
                                summary: 'Cannot get machine',
                                detail: 'Getting machine with ID ' + machineId + ' erred: ' + msg,
                                life: 10000,
                            })
                            this.router.navigate(['/machines/all'])
                        }
                    )
                }
            }
        })

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
            this.unauthorizedMachinesCount = data.total
        })
    }

    loadMachines(event) {
        let text
        if (event.filters.text) {
            text = event.filters.text.value
        }

        let app
        if (event.filters.app) {
            app = event.filters.app.value
        }

        this.servicesApi.getMachines(event.first, event.rows, text, app, !this.showUnauthorized).subscribe((data) => {
            this.machines = data.items
            this.totalMachines = data.total
        })
        this.refreshUnauthorizedMachinesCount()
    }

    cancelMachineDialog() {
        this.changeMachineAddressDlgVisible = false
    }

    keyUpMachineDlg(event, machineTab) {
        if (event.key === 'Enter') {
            if (this.changeMachineAddressDlgVisible) {
                this.saveMachine(machineTab)
            }
        }
    }

    refreshMachinesList(machinesTable) {
        machinesTable.onLazyLoad.emit(machinesTable.createLazyLoadMetadata())
    }

    keyUpFilterText(machinesTable, event) {
        if (this.filterText.length >= 3 || event.key === 'Enter') {
            machinesTable.filter(this.filterText, 'text', 'equals')
        }
    }

    filterByApp(machinesTable) {
        machinesTable.filter(this.selectedAppType.value, 'app', 'equals')
    }

    closeTab(event, idx) {
        this.openedMachines.splice(idx - 1, 1)
        this.tabs.splice(idx, 1)
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

    _refreshMachineState(machine) {
        this.servicesApi.getMachineState(machine.id).subscribe(
            (data) => {
                if (data.error) {
                    this.msgSrv.add({
                        severity: 'error',
                        summary: 'Getting machine state erred',
                        detail: 'Getting state of machine erred: ' + data.error,
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
                for (const m of this.machines) {
                    if (m.id === data.id) {
                        Object.assign(m, data)
                        if (data.error === undefined) {
                            m.error = ''
                        }
                        break
                    }
                }

                // refresh machine in opened tab if present
                for (const m of this.openedMachines) {
                    if (m.machine.id === data.id) {
                        Object.assign(m.machine, data)
                        if (data.error === undefined) {
                            m.machine.error = ''
                        }
                        break
                    }
                }
            },
            (err) => {
                let msg = err.statusText
                if (err.error && err.error.message) {
                    msg = err.error.message
                }
                this.msgSrv.add({
                    severity: 'error',
                    summary: 'Getting machine state erred',
                    detail: 'Getting state of machine erred: ' + msg,
                    life: 10000,
                })
            }
        )
    }

    /**
     * Authorize or unauthorize machine.
     *
     * @param machine machine object
     * @param authorized bool, true or false
     */
    _changeMachineAuthorization(machine, authorized, machinesTable) {
        machine.authorized = authorized
        const txt = 'Machine ' + (authorized ? 'de' : '') + 'authorized'
        this.servicesApi.updateMachine(machine.id, machine).subscribe(
            (data) => {
                this.msgSrv.add({
                    severity: 'success',
                    summary: txt,
                    detail: 'Update of the machine authorization status succeeded.',
                })
                this.refreshMachinesList(machinesTable)
            },
            (err) => {
                let msg = err.statusText
                if (err.error && err.error.message) {
                    msg = err.error.message
                }
                this.msgSrv.add({
                    severity: 'error',
                    summary: txt + ' attempt failed',
                    detail: 'Update of the machine authorization status failed: ' + msg,
                    life: 10000,
                })
            }
        )
    }

    /**
     * Shows menu with actions possible on a given machine. Currently this is
     * authorize/deauthorize or delete. It is called every time the user switches
     * between authorized/unauthorized view.
     *
     * @param event tbd
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

            // connect method to authorize machine
            this.machineMenuItems[1].command = () => {
                this._changeMachineAuthorization(machine, false, machinesTable)
            }

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
        this.servicesApi.deleteMachine(machineId).subscribe((data) => {
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

    editAddress(machineTab) {
        this.machineAddress = machineTab.machine.address
        this.agentPort = machineTab.machine.agentPort.toString() // later string is expected in this.agentPort
        this.changeMachineAddressDlgVisible = true
    }

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
                let msg = err.statusText
                if (err.error && err.error.message) {
                    msg = err.error.message
                }
                this.msgSrv.add({
                    severity: 'error',
                    summary: 'Machine address update failed',
                    detail: 'Updating machine address erred: ' + msg,
                    life: 10000,
                })
            }
        )
    }

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
            },
            (err) => {
                let msg = err.statusText
                if (err.error && err.error.message) {
                    msg = err.error.message
                }
                this.msgSrv.add({
                    severity: 'error',
                    summary: 'Cannot get server token',
                    detail: 'Getting server token for registering machines erred: ' + msg,
                    life: 10000,
                })
            }
        )
        this.displayAgentInstallationInstruction = true
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
                let msg = err.statusText
                if (err.error && err.error.message) {
                    msg = err.error.message
                }
                this.msgSrv.add({
                    severity: 'error',
                    summary: 'Cannot regenerate server token',
                    detail: 'Regenerating server token for registering machines erred: ' + msg,
                    life: 10000,
                })
            }
        )
    }

    /**
     * Return base URL of stork server website.
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
}
