import {Component, OnDestroy, OnInit, ViewChild} from '@angular/core'
import {PrefilteredTable} from "../table";
import {Machine, ServicesService} from "../backend";
import {Table, TableLazyLoadEvent} from "primeng/table";
import {ActivatedRoute} from "@angular/router";
import {MenuItem, MessageService} from "primeng/api";
import {Location} from "@angular/common";
import {lastValueFrom} from "rxjs";
import {getErrorMessage} from "../utils";
import {MachinesFilter} from "../authorized-machines-table/authorized-machines-table.component";

@Component({
    selector: 'app-unauthorized-machines-table',
    templateUrl: './unauthorized-machines-table.component.html',
    styleUrl: './unauthorized-machines-table.component.sass',
})
export class UnauthorizedMachinesTableComponent  extends PrefilteredTable<MachinesFilter, Machine>
    implements OnInit, OnDestroy
{
    /**
     * Array of all numeric keys that are supported when filtering machines via URL queryParams.
     * Note that it doesn't have to contain machines prefilterKey, which is 'appId'.
     * prefilterKey by default is considered as a primary queryParam filter key.
     */
    queryParamNumericKeys: (keyof MachinesFilter)[] = []

    /**
     * Array of all boolean keys that are supported when filtering machines via URL queryParams.
     * Currently, no boolean key is supported in queryParams filtering.
     */
    queryParamBooleanKeys: (keyof MachinesFilter)[] = []

    /**
     * Array of all numeric keys that can be used to filter machines.
     */
    filterNumericKeys: (keyof MachinesFilter)[] = ['appId']

    /**
     * Array of all boolean keys that can be used to filter machines.
     */
    filterBooleanKeys: (keyof MachinesFilter)[] = []

    /**
     * Prefix of the stateKey. Will be used to evaluate stateKey.
     */
    stateKeyPrefix: string = 'unauthorized-machines-table-session'

    /**
     * queryParam keyword of the filter by appId.
     */
    prefilterKey: keyof MachinesFilter = 'appId'

    /**
     * Array of FilterValidators that will be used for validation of filters, which values are limited
     * only to known values.
     */
    filterValidators = []

    /**
     * PrimeNG table instance.
     */
    @ViewChild('machinesTable') table: Table

    machines: Machine[] = []

    constructor(
        private route: ActivatedRoute,
        private servicesApi: ServicesService,
        private messageService: MessageService,
        private location: Location
    ) {
        super(route, location)
    }

    /**
     * Component lifecycle hook called to perform clean-up when destroying the component.
     */
    ngOnDestroy(): void {
        super.onDestroy()
    }

    /**
     * Component lifecycle hook called upon initialization.
     */
    ngOnInit(): void {
        super.onInit()
    }
    loadData(event: TableLazyLoadEvent): void {
        // Indicate that machines refresh is in progress.
        this.dataLoading = true
        // The goal is to send to backend something as simple as:
        // this.someApi.getMachines(JSON.stringify(event))
        lastValueFrom(
            this.servicesApi.getMachines(
                event.first,
                event.rows,
                this.getTableFilterValue('text', event.filters),
                this.prefilterValue ?? this.getTableFilterValue('appId', event.filters),
                false
            )
        )
            .then((data) => {
                this.machines = data.items ?? []
                this.totalRecords = data.total ?? 0
            })
            .catch((err) => {
                const msg = getErrorMessage(err)
                this.messageService.add({
                    severity: 'error',
                    summary: 'Cannot get machine list',
                    detail: 'Error getting machine list: ' + msg,
                    life: 10000,
                })
            })
            .finally(() => {
                this.dataLoading = false
            })
    }
    openedMachines: { machine: Machine }[]
    _refreshMachineState(machine: Machine) {
        this.servicesApi.getMachineState(machine.id).subscribe(
            (data) => {
                if (data.error) {
                    this.messageService.add({
                        severity: 'error',
                        summary: 'Error getting machine state',
                        detail: 'Error getting state of machine: ' + data.error,
                        life: 10000,
                    })
                } else {
                    this.messageService.add({
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
                this.messageService.add({
                    severity: 'error',
                    summary: 'Error getting machine state',
                    detail: 'Error getting state of machine: ' + msg,
                    life: 10000,
                })
            }
        )
    }

    machineMenuItems: MenuItem[]
    machineMenuItemsAuth: MenuItem[]

    showMachineMenu(event, machineMenu, machine, machinesTable) {
        // if (this.showUnauthorized) {
        //   this.machineMenuItems = this.machineMenuItemsUnauth
        // } else
        {
            this.machineMenuItems = this.machineMenuItemsAuth
        }

        machineMenu.toggle(event)

        // if (this.showUnauthorized) {
        //   // connect method to authorize machine
        //   this.machineMenuItems[0].command = () => {
        //     this._changeMachineAuthorization(machine, true, machinesTable)
        //   }
        //
        //   // connect method to delete machine
        //   this.machineMenuItems[1].command = () => {
        //     this.deleteMachine(machine.id)
        //   }
        // } else
        {
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
     * Start downloading the dump file.
     */
    downloadDump(machine: Machine) {
        window.location.href = `api/machines/${machine.id}/dump`
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
            // this.serverData.forceReloadAppsStats()

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
                    // this.closeTab(null, idx + 1)
                    break
                }
            }
        })
    }
}
