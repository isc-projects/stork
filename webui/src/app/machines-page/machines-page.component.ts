import { AfterViewInit, ChangeDetectorRef, Component, OnDestroy, OnInit, ViewChild } from '@angular/core'
import { ActivatedRoute, EventType, Router } from '@angular/router'

import { MessageService, MenuItem, ConfirmationService } from 'primeng/api'
import { BehaviorSubject, concat, EMPTY, lastValueFrom, Observable, Subscription } from 'rxjs'
import { Machine, Settings } from '../backend'

import { ServicesService, SettingsService } from '../backend'
import { ServerDataService } from '../server-data.service'
import { copyToClipboard, deepCopy, getErrorMessage } from '../utils'
import { catchError, filter } from 'rxjs/operators'
import { MachinesTableComponent } from '../machines-table/machines-table.component'
import { Menu } from 'primeng/menu'
import { SelectButtonChangeEvent } from 'primeng/selectbutton'

/**
 * This component implements a page which displays authorized
 * and unauthorized machines. The list of machines is
 * paged and can be filtered by provided URL queryParams or by
 * using form inputs responsible for filtering.
 *
 * This component is also responsible for viewing given machine
 * details in tab view, switching between tabs, closing them etc.
 */
@Component({
    selector: 'app-machines-page',
    templateUrl: './machines-page.component.html',
    styleUrls: ['./machines-page.component.sass'],
})
export class MachinesPageComponent implements OnInit, OnDestroy, AfterViewInit {
    /**
     * RxJS Subscription holding all subscriptions to Observables, so that they can be all unsubscribed
     * at once onDestroy.
     * @private
     */
    private subscriptions = new Subscription()

    /**
     * View breadcrumbs menu items.
     */
    breadcrumbs = [{ label: 'Services' }, { label: 'Machines' }]

    /**
     * Machine popup menu items.
     */
    machineMenuItems: MenuItem[]

    /**
     * Authorized machine popup menu items.
     */
    machineMenuItemsAuthorized: MenuItem[] = [
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
            title: 'Remove machine from Stork server',
        },
    ]

    /**
     * Unauthorized machine menu items.
     */
    machineMenuItemsUnauthorized: MenuItem[] = [
        {
            label: 'Authorize',
            id: 'authorize-single-machine',
            icon: 'pi pi-check',
        },
        {
            label: 'Remove',
            id: 'remove-single-machine',
            icon: 'pi pi-times',
            title: 'Remove machine from Stork server',
        },
    ]

    /**
     * Options for SelectButton component used to switch between Authorized/Unauthorized Machines view.
     */
    selectButtonOptions = [
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

    /**
     * Server token used for machine registration.
     */
    serverToken = ''

    /**
     * This counter is used to indicate in UI that there are some
     * unauthorized machines that may require authorization.
     * @private
     */
    private _unauthorizedMachinesCount = 0

    /**
     * Getter of the _unauthorizedMachinesCount property.
     */
    get unauthorizedMachinesCount(): number {
        return this._unauthorizedMachinesCount
    }

    /**
     * Setter of the _unauthorizedMachinesCount property.
     * Also triggers emitting next value by unauthorizedMachinesCount$ RxJS Subject.
     * @param c count to be set
     */
    set unauthorizedMachinesCount(c: number) {
        this._unauthorizedMachinesCount = c
        this.unauthorizedMachinesCount$.next(c)
    }

    /**
     * RxJS subject used to keep up-to-date count of Unauthorized machines.
     */
    unauthorizedMachinesCount$ = new BehaviorSubject<number>(this._unauthorizedMachinesCount)

    /**
     * Boolean flag keeping state whether the Change Machine address Dialog is visible or not.
     */
    changeMachineAddressDialogVisible = false

    /**
     * Machine's address.
     */
    machineAddress = 'localhost'

    /**
     * Machine's agent port.
     */
    agentPort = ''

    /**
     * Index of active tab in the TabMenu.
     */
    activeTabIdx: number = 0

    /**
     * TabMenu menu items.
     */
    tabs: MenuItem[]

    /**
     * Keeps state of open TabMenu tabs.
     */
    openedMachines: Machine[]

    /**
     * Reference to currently open machine tab.
     */
    get machineTab(): Machine | null {
        return this.openedMachines[this.activeTabIdx - 1] ?? null
    }

    /**
     * Boolean flag keeping state whether the Agent Installation Instructions Dialog is visible or not.
     */
    displayAgentInstallationInstruction = false

    /**
     * Indicates if the machines registration is administratively disabled.
     */
    registrationDisabled = false

    /**
     * Getter returning true if only Authorized machines are to be displayed,
     * false if only Unauthorized machines are to be displayed,
     * or null if both Authorized and Unauthorized machines are to be displayed.
     */
    get showAuthorized(): boolean | null {
        return this.machinesTable?.validFilter?.authorized ?? null
    }

    /**
     * Machines table component.
     */
    @ViewChild('machinesTable') machinesTable: MachinesTableComponent

    /**
     * Machines popup menu component.
     */
    @ViewChild('machineMenu') machineMenu: Menu

    /**
     * Component's constructor.
     * @param route activated route used to gather parameters from the URL.
     * @param router router used to navigate between tabs.
     * @param servicesApi services API to do all CRUD machine related operations
     * @param msgSrv Message service used to display feedback messages in UI.
     * @param serverData Server Data service used to reload Apps stats whenever machines registration state changes.
     * @param settingsService Settings service used to retrieve global settings.
     * @param confirmationService Confirmation used to handle confirmation dialogs.
     * @param cd Change detection used to manually detect changes to avoid error NG0100: ExpressionChangedAfterItHasBeenCheckedError
     */
    constructor(
        private route: ActivatedRoute,
        private router: Router,
        private servicesApi: ServicesService,
        private msgSrv: MessageService,
        private serverData: ServerDataService,
        private settingsService: SettingsService,
        private confirmationService: ConfirmationService,
        private cd: ChangeDetectorRef
    ) {}

    /**
     * Component lifecycle hook called to perform clean-up when destroying the component.
     */
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

                    // Apply to the changes of the machine id, e.g. from /machines/all to
                    // /machines/1. Those changes are triggered by switching between the
                    // tabs.

                    // Get machine id.
                    const id = paramMap.get('id')
                    if (!id || id === 'all') {
                        // Update the filter only if the target is machine list.
                        this.machinesTable?.updateFilterFromQueryParameters(queryParamMap)
                        this.cd.detectChanges()
                        this.switchToTab(0)
                        return
                    }
                    const numericId = parseInt(id, 10)
                    if (!Number.isNaN(numericId)) {
                        // The path has a numeric id indicating that we should
                        // open a tab with selected machine information or switch
                        // to this tab if it has been already opened.
                        for (let idx = 0; idx < this.openedMachines.length; idx++) {
                            const m = this.openedMachines[idx]
                            if (m.id === numericId) {
                                this.switchToTab(idx + 1)
                                return
                            }
                        }

                        // if tab is not opened then search for list of machines if the one is present there,
                        // if so then open it in new tab and switch to it
                        for (const m of this.machinesTable?.dataCollection || []) {
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
                                this.machinesTable?.loadDataWithoutFilter()
                                this.switchToTab(0)
                            })
                    } else {
                        // In case of failed Id parsing, open list tab.
                        this.machinesTable?.loadDataWithoutFilter()
                        this.switchToTab(0)
                    }
                })
        )
    }

    /**
     * Switches to tab with the given index.
     * @param index tab index to switch to
     */
    switchToTab(index: number) {
        if (this.activeTabIdx === index) {
            return
        }

        this.activeTabIdx = index
    }

    /**
     * Adds new tab with machine details.
     * @param machine machine for which the tab is added
     */
    addMachineTab(machine: Machine) {
        this.openedMachines.push(machine)
        this.tabs = [
            ...this.tabs,
            {
                label: machine.address,
                id: 'machine-tab' + machine.id,
                routerLink: '/machines/' + machine.id,
                machineId: machine.id,
            },
        ]
    }

    /**
     * Component lifecycle hook called upon initialization.
     * It configures initial state of PrimeNG Menu tabs and fetches global settings.
     */
    ngOnInit() {
        this.tabs = [{ label: 'Machines', id: 'all-machines-tab', routerLink: '/machines/all' }]

        this.machineMenuItems = this.machineMenuItemsAuthorized
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
    }

    /**
     * Callback called on key up in the edit machine dialog.
     * @param event keyboard event
     * @param machine machine under edit in the dialog
     */
    onEditMachineDialogKeyUp(event: KeyboardEvent, machine: Machine) {
        if (!machine) {
            this.showFeedbackForNullMachine()
            return
        }

        if (event.key === 'Enter') {
            if (this.changeMachineAddressDialogVisible) {
                this.saveMachine(machine)
            }
        }
    }

    /**
     * Closes a tab with the given index.
     * @param event mouse click event
     * @param idx index of the tab to be closed
     */
    closeTab(event: MouseEvent, idx: number) {
        this.openedMachines.splice(idx - 1, 1)
        this.tabs = [...this.tabs.slice(0, idx), ...this.tabs.slice(idx + 1)]
        if (this.activeTabIdx === idx) {
            // Closing the tab which is currently active.
            this.switchToTab(idx - 1)
            this.router.navigate([this.tabs[this.activeTabIdx].routerLink])
        } else if (this.activeTabIdx > idx) {
            // Closing the tab which is to the left from currently active tab.
            this.switchToTab(this.activeTabIdx - 1)
            this.router.navigate([this.tabs[this.activeTabIdx].routerLink])
        }

        event?.preventDefault()
    }

    /**
     * Fetches new machine state from API.
     * @param machine machine to be refreshed
     */
    refreshMachineState(machine: Machine) {
        if (!machine) {
            this.showFeedbackForNullMachine()
            return
        }

        this.machinesTable?.setDataLoading(true)
        lastValueFrom(this.servicesApi.getMachineState(machine.id))
            .then((m: Machine) => {
                if (m.error) {
                    this.msgSrv.add({
                        severity: 'error',
                        summary: 'Error getting machine state from Stork agent',
                        detail:
                            'The machine state was retrieved from the Stork server, but the server had problems communicating with the Stork agent on the machine: ' +
                            m.error,
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
                this.machinesTable?.refreshMachineState(m)

                // refresh machine in opened tab if present
                const idx = this.openedMachines.map((m) => m.id).indexOf(machine.id)
                if (idx >= 0) {
                    this.openedMachines.splice(idx, 1, m)
                }
            })
            .catch((err) => {
                const msg = getErrorMessage(err)
                this.msgSrv.add({
                    severity: 'error',
                    summary: 'Error getting machine state from Stork server',
                    detail: 'Error getting state of machine: ' + msg,
                    life: 10000,
                })
            })
            .finally(() => this.machinesTable?.setDataLoading(false))
    }

    /**
     * Start downloading the dump file.
     * @param machine machine for which the download is expected
     */
    downloadDump(machine: Machine) {
        if (!machine) {
            this.showFeedbackForNullMachine()
            return
        }

        window.location.href = `api/machines/${machine.id}/dump`
    }

    /**
     * Authorize single machine via the updateMachine API.
     * @param machine machine to be authorized
     */
    authorizeMachine(machine: Machine) {
        // Block table UI when machine authorization is in progress.
        this.machinesTable?.setDataLoading(true)
        const stateBackup = machine.authorized

        machine.authorized = true
        lastValueFrom(this.servicesApi.updateMachine(machine.id, machine))
            .then((m) => {
                this.msgSrv.add({
                    severity: 'success',
                    summary: `Machine authorized`,
                    detail: `Machine ${m.address} authorization succeeded.`,
                })
                this.machinesTable?.loadDataWithValidFilter()
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
            .finally(() => this.machinesTable?.setDataLoading(false))
    }

    /**
     * Shows popup menu with actions possible on a given machine.
     * There are different actions for authorized and unauthorized machine.
     *
     * @param event browser event generated when the button is clicked causing
     *        the menu to be toggled
     * @param machine reference to a machine
     */
    onMachineMenuDisplay(event: Event, machine: Machine) {
        if (!machine.authorized) {
            this.machineMenuItems = this.machineMenuItemsUnauthorized
            // connect method to authorize machine
            this.machineMenuItems[0].command = () => {
                this.authorizeMachine(machine)
            }

            // connect method to delete machine
            this.machineMenuItems[1].command = () => {
                this.deleteMachine(machine.id)
            }
        } else {
            this.machineMenuItems = this.machineMenuItemsAuthorized
            // connect method to refresh machine state
            this.machineMenuItems[0].command = () => {
                this.refreshMachineState(machine)
            }

            // connect method to dump machine configuration
            this.machineMenuItems[1].command = () => {
                this.downloadDump(machine)
            }

            // connect method to unauthorize machine
            /*this.machineMenuItems[2].command = () => {
                this.unauthorizeMachine(machine)
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
        this.confirmationService.confirm({
            message: `Are you sure you want to delete the machine with ID ${machineId}?`,
            header: 'Confirm',
            icon: 'pi pi-exclamation-triangle',
            accept: () => {
                this.machinesTable?.setDataLoading(true)
                lastValueFrom(this.servicesApi.deleteMachine(machineId))
                    .then(() => {
                        // reload apps stats to reflect new state (adjust menu content)
                        this.serverData.forceReloadAppsStats()

                        // remove from list of machines
                        this.machinesTable?.deleteMachine(machineId)

                        // remove from opened tabs if present
                        const idx = this.openedMachines.map((m) => m.id).indexOf(machineId)
                        if (idx >= 0) {
                            this.closeTab(null, idx + 1)
                        }
                        this.msgSrv.add({
                            severity: 'success',
                            summary: 'Machine deleted',
                            detail: 'Deletion succeeded.',
                        })
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
                    .finally(() => this.machinesTable?.setDataLoading(false))
            },
        })
    }

    /**
     * Sets the edit form-related members using the value of the current machine.
     * @param machine machine which is edited
     */
    editAddress(machine: Machine) {
        if (!machine) {
            this.showFeedbackForNullMachine()
            return
        }

        this.machineAddress = machine.address
        this.agentPort = machine.agentPort.toString() // later string is expected in this.agentPort
        this.changeMachineAddressDialogVisible = true
    }

    /**
     * Saves edited machine address and agent port of the machine using updateMachine API.
     * @param machine machine to be saved
     */
    saveMachine(machine: Machine) {
        if (!machine) {
            this.showFeedbackForNullMachine()
            return
        }

        if (this.machineAddress === machine.address && this.agentPort === machine.agentPort.toString()) {
            this.changeMachineAddressDialogVisible = false
            this.msgSrv.add({
                severity: 'success',
                summary: 'Machine address does not require updating',
                detail: 'Machine address was not changed.',
            })
            return
        }
        const m = {
            address: this.machineAddress,
            agentPort: parseInt(this.agentPort, 10),
            authorized: !!machine.authorized,
        }
        lastValueFrom(this.servicesApi.updateMachine(machine.id, m))
            .then((m: Machine) => {
                this.changeMachineAddressDialogVisible = false
                this.msgSrv.add({
                    severity: 'success',
                    summary: 'Machine address updated',
                    detail: 'Machine address update succeeded.',
                })
                machine = m

                // refresh machine in machines list
                this.refreshMachineState(machine)

                // refresh opened tab title
                const tabMenuItem = this.tabs.find((tab) => tab.machineId === machine.id)
                if (tabMenuItem) {
                    tabMenuItem.label = machine.address
                }
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
    copyToClipboard(textEl: HTMLInputElement | HTMLTextAreaElement) {
        return copyToClipboard(textEl)
    }

    /**
     * Authorizes machines stored in selectedMachines.
     *
     * @param machines array of Machines to be authorized
     */
    onAuthorizeSelectedMachines(machines: Machine[]) {
        // Filter out machines that are already authorized.
        const unauthorized = deepCopy(machines?.filter((m) => !m.authorized)) ?? []

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
        this.machinesTable?.setDataLoading(true)
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
                this.machinesTable?.setDataLoading(false)
                this.machinesTable?.loadDataWithValidFilter()
                // Force menu adjustments to take into account that there
                // is new machine and apps available.
                this.serverData.forceReloadAppsStats()
            },
            complete: () => {
                this.machinesTable?.setDataLoading(false)
                this.machinesTable?.loadDataWithValidFilter()
                // Force menu adjustments to take into account that there
                // is new machine and apps available.
                this.serverData.forceReloadAppsStats()
            },
        })

        // Clear selection after.
        this.machinesTable?.clearSelection()
    }

    /**
     * Callback called when the Authorized/Unauthorized machines select button changes after user's click.
     * @param event change event
     */
    onSelectButtonChange(event: SelectButtonChangeEvent) {
        this.router.navigate(['machines', 'all'], { queryParams: { authorized: event?.value } ?? null })
    }

    /**
     * This helper method displays feedback to the user that other method was called for not existing machine.
     * @private
     */
    private showFeedbackForNullMachine() {
        this.msgSrv.add({
            severity: 'error',
            summary: 'Machine not found',
            detail: 'Machine was not found',
            life: 10000,
        })
    }
}
