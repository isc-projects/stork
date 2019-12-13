import { Component, OnInit } from '@angular/core'
import { ActivatedRoute, ParamMap, Router, NavigationEnd } from '@angular/router'

import { MessageService, MenuItem } from 'primeng/api'

import { ServicesService } from '../backend/api/api'
import { LoadingService } from '../loading.service'

interface ServiceType {
    name: string
    value: string
}

@Component({
    selector: 'app-machines-page',
    templateUrl: './machines-page.component.html',
    styleUrls: ['./machines-page.component.sass'],
})
export class MachinesPageComponent implements OnInit {
    // machines table
    machines: any[]
    totalMachines: number
    machineMenuItems: MenuItem[]

    // action panel
    filterText = ''
    serviceTypes: ServiceType[]
    selectedServiceType: ServiceType

    // new machine
    newMachineDlgVisible = false
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
            address: machine.address,
            agentPort: machine.agentPort,
            activeInplace: false,
        })
        this.tabs.push({
            label: machine.hostname || machine.address,
            routerLink: '/machines/' + machine.id,
        })
    }

    ngOnInit() {
        this.tabs = [{ label: 'Machines', routerLink: '/machines/all' }]

        this.machines = []
        this.serviceTypes = [{ name: 'any', value: '' }, { name: 'BIND', value: 'bind' }, { name: 'Kea', value: 'kea' }]
        this.machineMenuItems = [
            {
                label: 'Refresh',
                icon: 'pi pi-refresh',
            },
            {
                label: 'Delete',
                icon: 'pi pi-times',
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
                        data => {
                            this.addMachineTab(data)
                            this.switchToTab(this.tabs.length - 1)
                        },
                        err => {
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
        console.info(event)
        let text
        if (event.filters.text) {
            text = event.filters.text.value
        }

        let service
        if (event.filters.service) {
            service = event.filters.service.value
        }

        this.servicesApi.getMachines(event.first, event.rows, text, service).subscribe(data => {
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
            data => {
                this.loadingService.stop('adding new machine')
                this.msgSrv.add({
                    severity: 'success',
                    summary: 'New machine added',
                    detail: 'Adding new machine succeeded.',
                })
                this.newMachineDlgVisible = false
                this.addMachineTab(data)
                this.router.navigate(['/machines/' + data.id])
            },
            err => {
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

    cancelNewMachine() {
        this.newMachineDlgVisible = false
    }

    keyDownNewMachine(event) {
        if (event.key === 'Enter') {
            this.addNewMachine()
        }
    }

    refreshMachinesList(machinesTable) {
        machinesTable.onLazyLoad.emit(machinesTable.createLazyLoadMetadata())
    }

    keyDownFilterText(machinesTable, event) {
        if (this.filterText.length >= 3 || event.key === 'Enter') {
            machinesTable.filter(this.filterText, 'text', 'equals')
        }
    }

    filterByService(machinesTable) {
        machinesTable.filter(this.selectedServiceType.value, 'service', 'equals')
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
            data => {
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
            err => {
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
            this.servicesApi.deleteMachine(machine.id).subscribe(data => {
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
        machineTab.activeInplace = true
    }

    saveMachine(machineTab) {
        console.info(machineTab)
        if (
            machineTab.address === machineTab.machine.address &&
            machineTab.agentPort === machineTab.machine.agentPort
        ) {
            machineTab.activeInplace = false
            return
        }
        const m = { address: machineTab.address, agentPort: parseInt(machineTab.agentPort, 10) }
        this.servicesApi.updateMachine(machineTab.machine.id, m).subscribe(
            data => {
                console.info('updated', data)
                machineTab.machine.address = data.address
                machineTab.activeInplace = false
                this.msgSrv.add({
                    severity: 'success',
                    summary: 'Machine address updated',
                    detail: 'Machine address update succeeded.',
                })
            },
            err => {
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

    machineAddressKeyDown(event, machineTab) {
        if (event.key === 'Enter') {
            this.saveMachine(machineTab)
        } else if (event.key === 'Escape') {
            machineTab.activeInplace = false
        }
    }

    refreshMachineState(machinesTab) {
        this._refreshMachineState(machinesTab.machine)
    }
}
