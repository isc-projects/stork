import { Component, OnInit } from '@angular/core'
import { ActivatedRoute, ParamMap, Router, NavigationEnd } from '@angular/router'

import { MessageService, MenuItem } from 'primeng/api'

import { ServicesService } from '../backend/api/api'
import { LoadingService } from '../loading.service'
import { ServerDataService } from '../server-data.service'

interface AppType {
    name: string
    value: string
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

    // action panel
    filterText = ''
    appTypes: AppType[]
    selectedAppType: AppType

    // new machine, edit machine address
    newMachineDlgVisible = false
    changeMachineAddressDlgVisible = false
    machineAddress = 'localhost'
    agentPort = ''

    // machine tabs
    activeTabIdx = 0
    tabs: MenuItem[]
    activeItem: MenuItem
    openedMachines: any
    machineTab: any

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
            routerLink: '/machines/' + machine.id,
        })
    }

    ngOnInit() {
        this.tabs = [{ label: 'Machines', id: 'machines-tab-id', routerLink: '/machines/all' }]

        this.machines = []
        this.appTypes = [
            { name: 'any', value: '' },
            { name: 'Bind9', value: 'bind9' },
            { name: 'Kea', value: 'kea' },
        ]
        this.machineMenuItems = [
            {
                label: 'Refresh',
                icon: 'pi pi-refresh',
            },
            {
                label: 'Remove',
                icon: 'pi pi-times',
                title: 'Remove machine from Stork Server',
            },
        ]

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

        this.servicesApi.getMachines(event.first, event.rows, text, app).subscribe((data) => {
            this.machines = data.items
            this.totalMachines = data.total
        })
    }

    showNewMachineDlg() {
        this.newMachineDlgVisible = true
    }

    addNewMachine() {
        if (this.machineAddress.trim() === '') {
            this.msgSrv.add({
                severity: 'error',
                summary: 'Adding new machine erred',
                detail: 'Machine address cannot be empty.',
                life: 10000,
            })
            return
        }

        this.newMachineDlgVisible = false

        let agentPort = this.agentPort.trim()
        if (agentPort === '') {
            agentPort = '8080'
        }
        const m = { address: this.machineAddress, agentPort: parseInt(agentPort, 10) }

        this.loadingService.start('adding new machine')
        this.servicesApi.createMachine(m).subscribe(
            (data) => {
                this.loadingService.stop('adding new machine')
                this.msgSrv.add({
                    severity: 'success',
                    summary: 'New machine added',
                    detail: 'Adding new machine succeeded.',
                })
                this.newMachineDlgVisible = false
                this.addMachineTab(data)

                this.serverData.forceReloadAppsStats()

                this.router.navigate(['/machines/' + data.id])
            },
            (err) => {
                this.loadingService.stop('adding new machine')
                console.info(err)
                let msg = err.statusText
                if (err.error && err.error.message) {
                    msg = err.error.message
                }
                this.msgSrv.add({
                    severity: 'error',
                    summary: 'Adding new machine erred',
                    detail: 'Adding new machine operation erred: ' + msg,
                    life: 10000,
                })
                this.newMachineDlgVisible = false
            }
        )
    }

    cancelMachineDialog() {
        this.newMachineDlgVisible = false
        this.changeMachineAddressDlgVisible = false
    }

    keyUpMachineDlg(event, machineTab) {
        if (event.key === 'Enter') {
            if (this.newMachineDlgVisible) {
                this.addNewMachine()
            } else if (this.changeMachineAddressDlgVisible) {
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

    showMachineMenu(event, machineMenu, machine) {
        machineMenu.toggle(event)

        // connect method to refresh machine state
        this.machineMenuItems[0].command = () => {
            this._refreshMachineState(machine)
        }

        // connect method to delete machine
        this.machineMenuItems[1].command = () => {
            this.servicesApi.deleteMachine(machine.id).subscribe((data) => {
                this.serverData.forceReloadAppsStats()

                // remove from list of machines
                for (let idx = 0; idx < this.machines.length; idx++) {
                    const m = this.machines[idx]
                    if (m.id === machine.id) {
                        this.machines.splice(idx, 1) // TODO: does not work
                        break
                    }
                }
                // remove from opened tabs if present
                for (let idx = 0; idx < this.openedMachines.length; idx++) {
                    const m = this.openedMachines[idx].machine
                    if (m.id === machine.id) {
                        this.closeTab(null, idx + 1)
                        break
                    }
                }
            })
        }
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
}
