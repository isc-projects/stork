import { Component, OnInit } from '@angular/core'
import { ActivatedRoute, ParamMap, Router, NavigationEnd } from '@angular/router'

import { MessageService } from 'primeng/api'

import { ServicesService } from '../backend/api/api'

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

    // action panel
    filterText = ''
    serviceTypes: ServiceType[]
    selectedServiceType: ServiceType

    // new machine
    newMachineDlgVisible = false
    machineAddress = 'agent-kea:8080'

    // machine tabs
    activeTabIdx = 0
    individualMachines: any[]

    constructor(
        private route: ActivatedRoute,
        private router: Router,
        private servicesApi: ServicesService,
        private msgSrv: MessageService
    ) {}

    switchToTab(index) {
        // TODO: this is hack but I cannot find other way to activate newly added tab
        setTimeout(() => {
            this.activeTabIdx = index
        }, 100)
    }

    ngOnInit() {
        this.machines = []
        this.serviceTypes = [{ name: 'any', value: '' }, { name: 'BIND', value: 'bind' }, { name: 'Kea', value: 'kea' }]
        this.individualMachines = []

        this.route.paramMap.subscribe((params: ParamMap) => {
            const machineId = params.get('id')
            if (machineId) {
                let found = false
                // if tab for this machine is already opened then switch to it
                this.individualMachines.every((m, mIdx) => {
                    if (m.hostname === machineId) {
                        console.info('found opened machine', mIdx)
                        this.switchToTab(Number(mIdx) + 1)
                        found = true
                        return false
                    }
                    return true
                })

                // if tab is not opened then search for list of machines if the one is present there,
                // if so then open it in new tab and switch to it
                if (!found) {
                    for (const m of this.machines) {
                        if (m.hostname === machineId) {
                            console.info('found machine in the list, opening it')
                            this.individualMachines.push(m)
                            this.switchToTab(this.individualMachines.length)
                            found = true
                            break
                        }
                    }
                }

                // if machine is not loaded in list fetch it individually
                if (!found) {
                    console.info('fetching machine')
                    // TODO: needed proper ID of machine from DB
                }
            }
        })
    }

    loadMachines(event) {
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

    tabChange(event) {
        if (event.index === 0) {
            this.router.navigate(['/machines/'])
        } else {
            const m = this.individualMachines[event.index - 1]
            this.router.navigate(['/machines/' + m.hostname])
        }
    }

    showNewMachineDlg() {
        this.newMachineDlgVisible = true
    }

    addNewMachine(machinesTable) {
        this.newMachineDlgVisible = false

        const m = { address: this.machineAddress }

        this.servicesApi.createMachine(m).subscribe(
            data => {
                console.info(data)
                this.msgSrv.add({
                    severity: 'success',
                    summary: 'New machine added',
                    detail: 'Adding new machine succeeded.',
                })
                this.newMachineDlgVisible = false
                this.individualMachines.push(data)
                this.switchToTab(this.individualMachines.length)
                this.refresh(machinesTable)
            },
            err => {
                console.info(err)
                let msg = err.statusText
                if (err.error && err.error.detail) {
                    msg = err.error.detail
                }
                this.msgSrv.add({
                    severity: 'error',
                    summary: 'Adding new machine erred',
                    detail: 'Adding new machine operation erred: ' + msg,
                    sticky: true,
                })
                this.newMachineDlgVisible = false
            }
        )
    }

    cancelNewMachine() {
        this.newMachineDlgVisible = false
    }

    keyDownNewMachine(machinesTable, event) {
        if (event.key === 'Enter') {
            this.addNewMachine(machinesTable)
        }
    }

    refresh(machinesTable) {
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

    openMachineTab(machine) {
        console.info(machine)
    }
}
