import { AfterViewInit, Component, OnDestroy, OnInit, ViewChild } from '@angular/core'
import { ActivatedRoute, EventType, Router } from '@angular/router'

import { MessageService, MenuItem } from 'primeng/api'
import { BehaviorSubject, concat, EMPTY, lastValueFrom, Observable, Subscription } from 'rxjs'
import { Machine, Settings } from '../backend'

import { ServicesService, SettingsService } from '../backend'
import { ServerDataService } from '../server-data.service'
import { copyToClipboard, getErrorMessage } from '../utils'
import { catchError, filter } from 'rxjs/operators'
import { MachinesTableComponent } from '../machines-table/machines-table.component'
import { Menu } from 'primeng/menu'

@Component({
    selector: 'app-machines-page',
    templateUrl: './machines-page.component.html',
    styleUrls: ['./machines-page.component.sass'],
})
export class MachinesPageComponent implements OnInit, OnDestroy, AfterViewInit {
    private subscriptions = new Subscription()
    breadcrumbs = [{ label: 'Services' }, { label: 'Machines' }]

    machineMenuItems: MenuItem[]
    machineMenuItemsAuth: MenuItem[]
    machineMenuItemsUnauth: MenuItem[]
    viewSelectionOptions: any[]
    serverToken = ''
    dataLoading: boolean

    // This counter is used to indicate in UI that there are some
    // unauthorized machines that may require authorization.
    _unauthorizedMachinesCount = 0

    unauthorizedMachinesCount$ = new BehaviorSubject<number>(this._unauthorizedMachinesCount)

    get unauthorizedMachinesCount(): number {
        return this._unauthorizedMachinesCount
    }

    set unauthorizedMachinesCount(c: number) {
        this._unauthorizedMachinesCount = c
        this.unauthorizedMachinesCount$.next(c)
    }

    // edit machine address
    changeMachineAddressDlgVisible = false
    machineAddress = 'localhost'
    agentPort = ''

    // machine tabs
    activeTabIdx = 0
    tabs: MenuItem[]
    activeItem: MenuItem
    // TODO: is the object type required?
    openedMachines: { machine: Machine }[]
    machineTab: { machine: Machine }

    displayAgentInstallationInstruction = false

    // Indicates if the machines registration is administratively disabled.
    registrationDisabled = false

    get showAuthorized(): boolean {
        return this.table?.validFilter?.authorized ?? null
    }

    @ViewChild('machinesTableComponent') table: MachinesTableComponent

    @ViewChild('machineMenu') machineMenu: Menu

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
        this.unauthorizedMachinesCount$.complete()
    }

    /**
     * Component lifecycle hook called after Angular completed the initialization of the
     * component's view.
     *
     * We subscribe to router events to act upon URL and/or queryParams changes.
     * This is done at this step, because we have to be sure that all child components,
     * especially PrimeNG table in MachinesTableComponent, are initialized.
     */
    ngAfterViewInit(): void {
        this.subscriptions.add(
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

                    // Apply to the changes of the machine id, e.g. from /machines/authorized to
                    // /machines/1. Those changes are triggered by switching between the
                    // tabs.

                    // Get machine id.
                    // this.showUnauthorized = false
                    const id = paramMap.get('id')
                    if (!id || id === 'all') {
                        // Update the filter only if the target is machine list.
                        this.table?.updateFilterFromQueryParameters(queryParamMap)
                        this.switchToTab(0)
                        return
                    }
                    const numericId = parseInt(id, 10)
                    if (!Number.isNaN(numericId)) {
                        // The path has a numeric id indicating that we should
                        // open a tab with selected machine information or switch
                        // to this tab if it has been already opened.

                        // if tab for this machine is already opened then switch to it
                        for (let idx = 0; idx < this.openedMachines.length; idx++) {
                            const m = this.openedMachines[idx].machine
                            if (m.id === numericId) {
                                this.switchToTab(idx + 1)
                                return
                            }
                        }

                        // if tab is not opened then search for list of machines if the one is present there,
                        // if so then open it in new tab and switch to it
                        for (const m of this.table?.dataCollection || []) {
                            if (m.id === numericId) {
                                this.addMachineTab(m)
                                this.switchToTab(this.tabs.length - 1)
                                return
                            }
                        }

                        // if machine is not loaded in list fetch it individually
                        lastValueFrom(this.servicesApi.getMachine(numericId))
                            .then((machine) => {
                                this.addMachineTab(machine)
                                this.switchToTab(this.tabs.length - 1)
                            })
                            .catch((err) => {
                                const msg = getErrorMessage(err)
                                this.msgSrv.add({
                                    severity: 'error',
                                    summary: 'Cannot get machine',
                                    detail: 'Failed to get machine with ID ' + numericId + ': ' + msg,
                                    life: 10000,
                                })
                                this.table?.loadDataWithoutFilter()
                                this.switchToTab(0)
                            })
                    } else {
                        // In case of failed Id parsing, open list tab.
                        this.table?.loadDataWithoutFilter()
                        this.switchToTab(0)
                    }
                })
        )
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
        this.tabs = [{ label: 'Machines', id: 'all-machines-tab', routerLink: '/machines/all' }]

        this.machineMenuItemsAuth = [
            {
                label: 'Refresh machine state information',
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
                value: true,
            },
            {
                label: 'Unauthorized',
                value: false,
                hasBadge: true,
            },
        ]

        this.openedMachines = []

        // Settings are needed to check whether the machines registration is disabled.
        lastValueFrom(this.settingsService.getSettings())
            .then((settings: Settings) => {
                this.registrationDisabled = !settings.enableMachineRegistration
            })
            .catch((err) => {
                const msg = getErrorMessage(err)
                this.msgSrv.add({
                    severity: 'error',
                    summary: 'Cannot get settings',
                    detail: 'Failed to get settings: ' + msg,
                    life: 10000,
                })
            })

        this.dataLoading = true
    }

    /** Callback called on canceling the edit machine dialog. */
    cancelMachineDialog() {
        this.changeMachineAddressDlgVisible = false
    }

    /** Callback called on key pressed in the edit machine dialog. */
    keyUpMachineDlg(event: KeyboardEvent, machineTab: any) {
        if (event.key === 'Enter') {
            if (this.changeMachineAddressDlgVisible) {
                this.saveMachine(machineTab)
            }
        }
    }

    /** Closes a tab with the given index. */
    closeTab(event: MouseEvent, idx: number) {
        this.openedMachines.splice(idx - 1, 1)
        this.tabs = [...this.tabs.slice(0, idx), ...this.tabs.slice(idx + 1)]
        if (this.activeTabIdx === idx) {
            this.switchToTab(idx - 1)
            this.router.navigate(this.tabs[idx - 1].routerLink)
        } else if (this.activeTabIdx > idx) {
            this.activeTabIdx = this.activeTabIdx - 1
        }
        if (event) {
            event.preventDefault()
        }
    }

    /** Fetches new machine state from API. */
    refreshMachineState(machine: Machine) {
        this.dataLoading = true
        lastValueFrom(this.servicesApi.getMachineState(machine.id))
            .then((m: Machine) => {
                if (m.error) {
                    this.msgSrv.add({
                        severity: 'error',
                        summary: 'Error getting machine state',
                        detail: 'Error getting state of machine: ' + m.error,
                        life: 10000,
                    })
                } else {
                    this.msgSrv.add({
                        severity: 'success',
                        summary: 'Machine refreshed',
                        detail: 'Refreshing succeeded.',
                    })
                    // TODO should the code below go here?
                }

                // refresh machine in machines list
                this.table?.refreshMachineState(m)

                // refresh machine in opened tab if present
                const idx = this.openedMachines.map((m) => m.machine.id).indexOf(machine.id)
                if (idx >= 0) {
                    this.openedMachines.splice(idx, 1, { machine: machine })
                }
            })
            .catch((err) => {
                const msg = getErrorMessage(err)
                this.msgSrv.add({
                    severity: 'error',
                    summary: 'Error getting machine state',
                    detail: 'Error getting state of machine: ' + msg,
                    life: 10000,
                })
            })
            .finally(() => (this.dataLoading = false))
    }

    /**
     * Start downloading the dump file.
     */
    downloadDump(machine: Machine) {
        window.location.href = `api/machines/${machine.id}/dump`
    }

    /**
     * Authorize single machine via the updateMachine API.
     * @param machine machine to be authorized
     */
    authorizeMachine(machine: Machine) {
        // Block table UI when machine authorization is in progress.
        this.dataLoading = true
        const stateBackup = machine.authorized

        machine.authorized = true
        lastValueFrom(this.servicesApi.updateMachine(machine.id, machine))
            .then((m) => {
                this.msgSrv.add({
                    severity: 'success',
                    summary: `Machine authorized`,
                    detail: `Machine ${m.address} authorization succeeded.`,
                })
                this.table?.loadDataWithValidFilter()
                // Force menu adjustments to take into account that there
                // is new machine and apps available.
                this.serverData.forceReloadAppsStats()
            })
            .catch((err) => {
                machine.authorized = stateBackup
                const msg = getErrorMessage(err)
                this.msgSrv.add({
                    severity: 'error',
                    summary: `Machine authorization failed`,
                    detail: `Machine authorization attempt failed: ${msg}`,
                    life: 10000,
                })
            })
            .finally(() => (this.dataLoading = false))
    }

    /**
     * Shows menu with actions possible on a given machine. Currently this is
     * authorize/unauthorize or delete. It is called every time the user switches
     * between authorized/unauthorized view.
     *
     * @param event browser event generated when the button is clicked causing
     *        the menu to be toggled
     * @param machine reference to a machine
     */
    showMachineMenu(event: Event, machine: Machine) {
        if (!machine.authorized) {
            this.machineMenuItems = this.machineMenuItemsUnauth
            // connect method to authorize machine
            this.machineMenuItems[0].command = () => {
                this.authorizeMachine(machine)
            }

            // connect method to delete machine
            this.machineMenuItems[1].command = () => {
                this.deleteMachine(machine.id)
            }
        } else {
            this.machineMenuItems = this.machineMenuItemsAuth
            // connect method to refresh machine state
            this.machineMenuItems[0].command = () => {
                this.refreshMachineState(machine)
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

        this.machineMenu.toggle(event)
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
    deleteMachine(machineId: number) {
        this.dataLoading = true
        // TODO: add confirmation dialog?
        lastValueFrom(this.servicesApi.deleteMachine(machineId))
            .then(() => {
                // reload apps stats to reflect new state (adjust menu content)
                this.serverData.forceReloadAppsStats()

                // remove from list of machines
                this.table?.deleteMachine(machineId)

                // remove from opened tabs if present
                const idx = this.openedMachines.map((m) => m.machine.id).indexOf(machineId)
                if (idx >= 0) {
                    this.closeTab(null, idx + 1)
                }
            })
            .catch((err) => {
                const msg = getErrorMessage(err)
                this.msgSrv.add({
                    severity: 'error',
                    summary: 'Deleting machine failed',
                    detail: 'Error deleting machine: ' + msg,
                    life: 10000,
                })
            })
            .finally(() => (this.dataLoading = false))
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
        lastValueFrom(this.servicesApi.updateMachine(machineTab.machine.id, m))
            .then((m: Machine) => {
                machineTab.machine = m
                this.changeMachineAddressDlgVisible = false
                this.msgSrv.add({
                    severity: 'success',
                    summary: 'Machine address updated',
                    detail: 'Machine address update succeeded.',
                })

                this.refreshMachineState(machineTab.machine)
            })
            .catch((err) => {
                const msg = getErrorMessage(err)
                this.msgSrv.add({
                    severity: 'error',
                    summary: 'Machine address update failed',
                    detail: 'Error updating machine address: ' + msg,
                    life: 10000,
                })
            })
    }

    /**
     * Display a dialog with instructions about installing
     * stork agent.
     */
    showAgentInstallationInstruction() {
        lastValueFrom(this.servicesApi.getMachinesServerToken())
            .then((data) => {
                this.serverToken = data.token
                this.displayAgentInstallationInstruction = true
            })
            .catch((err) => {
                const msg = getErrorMessage(err)
                this.msgSrv.add({
                    severity: 'error',
                    summary: 'Cannot get server token',
                    detail: 'Error getting server token to register machines: ' + msg,
                    life: 10000,
                })
            })
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
        lastValueFrom(this.servicesApi.regenerateMachinesServerToken())
            .then((data) => {
                this.serverToken = data.token
            })
            .catch((err) => {
                const msg = getErrorMessage(err)
                this.msgSrv.add({
                    severity: 'error',
                    summary: 'Cannot regenerate server token',
                    detail: 'Error regenerating server token to register machines: ' + msg,
                    life: 10000,
                })
            })
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
     * @param machines
     */
    authorizeSelectedMachines(machines: Machine[]) {
        const unauthorized = machines?.filter((m) => !m.authorized) ?? []
        // Calling servicesApi.updateMachine() API sequentially for all selected machines.
        // Max expected count of selected machines is max machines per table page,
        // which currently is 50.
        const updateObservables: Observable<Machine>[] = []
        for (const m of unauthorized) {
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
                this.table?.loadDataWithValidFilter()
                // this.refreshMachinesList(table)
                // Force menu adjustments to take into account that there
                // is new machine and apps available.
                this.serverData.forceReloadAppsStats()
            },
            complete: () => {
                this.dataLoading = false
                this.table?.loadDataWithValidFilter()
                // this.refreshMachinesList(table)
                // Force menu adjustments to take into account that there
                // is new machine and apps available.
                this.serverData.forceReloadAppsStats()
            },
        })

        // Clear selection after.
        this.table?.clearSelection()
    }

    /**
     * Callback called when the Authorized/Unauthorized machines select button changes after user's click.
     * @param event change event
     */
    onSelectChange(event: any) {
        this.router.navigate(['machines', 'all'], { queryParams: { authorized: event?.value } ?? null })
    }
}
