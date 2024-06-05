import { AfterViewInit, Component, OnDestroy, OnInit, ViewChild } from '@angular/core'
import { Router, ActivatedRoute, EventType } from '@angular/router'

import { MenuItem, MessageService } from 'primeng/api'

import { DHCPService } from '../backend/api/api'
import { getErrorMessage } from '../utils'
import { concat, EMPTY, of, Subscription } from 'rxjs'
import { catchError, filter, take } from 'rxjs/operators'
import { HostForm } from '../forms/host-form'
import { Host } from '../backend'
import { HostsTableComponent } from '../hosts-table/hosts-table.component'
import { Tab, TabType } from '../tab'

/**
 * This component implements a page which displays hosts along with
 * their DHCP identifiers and IP reservations. The list of hosts is
 * paged and can be filtered by provided URL queryParams or by
 * using form inputs responsible for filtering. The list
 * contains hosts reservations for all subnets and also contain global
 * reservations, i.e. those that are not associated with any particular
 * subnet.
 *
 * This component is also responsible for viewing given host reservation
 * details in tab view, switching between tabs, closing them etc.
 */
@Component({
    selector: 'app-hosts-page',
    templateUrl: './hosts-page.component.html',
    styleUrls: ['./hosts-page.component.sass'],
})
export class HostsPageComponent implements OnInit, OnDestroy, AfterViewInit {
    /**
     * RxJS Subscription holding all subscriptions to Observables, so that they can be all unsubscribed
     * at once onDestroy.
     */
    subscriptions = new Subscription()

    /**
     * Table with hosts component.
     */
    @ViewChild('hostsTableComponent') table: HostsTableComponent

    breadcrumbs = [{ label: 'DHCP' }, { label: 'Host Reservations' }]

    /**
     * Array of tabs with host information.
     *
     * The first tab is always present and displays the hosts list.
     */
    tabs: MenuItem[]

    /**
     * Enumeration for different host tab types displayed by the component.
     */
    HostTabType = TabType

    /**
     * Selected tab index.
     *
     * The first tab has an index of 0.
     */
    activeTabIndex = 0

    /**
     * Holds the information about specific hosts presented in the tabs.
     *
     * The tab holding hosts list is not included in this tab. If only a tab
     * with the hosts list is displayed, this array is empty.
     */
    openedTabs: Tab<HostForm, Host>[] = []

    /**
     * Constructor.
     *
     * @param route activated route used to gather parameters from the URL.
     * @param router router used to navigate between tabs.
     * @param dhcpApi server API used to gather hosts information.
     * @param messageService message service used to display error messages to a user.
     */
    constructor(
        private route: ActivatedRoute,
        private router: Router,
        private dhcpApi: DHCPService,
        private messageService: MessageService
    ) {}

    ngOnDestroy(): void {
        this.subscriptions.unsubscribe()
    }

    /**
     * Component lifecycle hook called upon initialization.
     *
     * It configures initial state of PrimeNG Menu tabs.
     */
    ngOnInit() {
        // Initially, there is only a tab with hosts list.
        this.tabs = [{ label: 'Host Reservations', routerLink: '/dhcp/hosts/all' }]
    }

    /**
     * Component lifecycle hook called after Angular completed the initialization of the
     * component's view.
     *
     * We subscribe to router events to act upon URL and/or queryParams changes.
     * This is done at this step, because we have to be sure that all child components,
     * especially PrimeNG table in HostsTableComponent, are initialized.
     */
    ngAfterViewInit(): void {
        this.subscriptions.add(
            // This component is responsible for routing of multiple
            // components: hosts list, host details, and host forms.
            // We want to preserve the filtering parameters when switching
            // between the tabs. So we need to know both URL and query
            // parameters in the same time.
            //
            // If we register to the `route.queryParamMap` and `route.paramMap`
            // separately or we merge them using the `combineLatest` operator,
            // we may get the situation when the query parameters are updated
            // after the segment parameters. In this case, the filtering
            // parameters are updated twice: first with the new query
            // parameters but with old segment parameters and then with the new
            // query and segment parameters.
            //
            // We need to differently treat the situation when the user
            // switches to detail tab (preserve the filtering parameters and
            // clear the query parameters), when the user back to the list tab
            // (restore the query parameters) and when the user changes the
            // query parameters in URL bar (update the filtering parameters).
            //
            // We need a guarantee that the change of the segment and query
            // parameters are notified in the same time. It is achieved by
            // registering to the `navigation end` event.
            //
            // See: https://stackoverflow.com/a/45765143
            this.router.events
                .pipe(
                    filter((event, idx) => idx === 0 || event.type === EventType.NavigationEnd),
                    catchError((err) => {
                        const msg = getErrorMessage(err)
                        this.messageService.add({
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

                    // Apply to the changes of the host id, e.g. from /dhcp/hosts/all to
                    // /dhcp/hosts/1. Those changes are triggered by switching between the
                    // tabs.

                    // Get host id.
                    const id = paramMap.get('id')
                    if (!id || id === 'all') {
                        // Update the filter only if the target is host list.
                        this.table?.updateFilterFromQueryParameters(queryParamMap)
                        this.switchToTab(0)
                        return
                    }
                    if (id === 'new') {
                        this.openNewHostTab()
                        return
                    }
                    const numericId = parseInt(id, 10)
                    if (!Number.isNaN(numericId)) {
                        // The path has a numeric id indicating that we should
                        // open a tab with selected host information or switch
                        // to this tab if it has been already opened.
                        this.openHostTab(numericId)
                    } else {
                        // In case of failed Id parsing, open list tab.
                        this.switchToTab(0)
                        this.table?.loadDataWithoutFilter()
                    }
                })
        )
    }

    /**
     * Opens existing or new host tab.
     *
     * If the host tab for the given host ID does not exist, a new tab is opened.
     * Otherwise, the existing tab is opened.
     *
     * @param id host ID.
     */
    private openHostTab(id: number) {
        let index = this.openedTabs.findIndex(
            (t) => (t.tabType === TabType.Display || t.tabType === TabType.Edit) && t.tabSubject.id === id
        )
        if (index >= 0) {
            this.switchToTab(index + 1)
            return
        }
        // Check if the host info is already available.
        let hostInfo: any
        if (this.table?.hosts) {
            const filteredHosts = this.table.hosts.filter((host) => host.id === id)
            if (filteredHosts.length > 0) {
                hostInfo = filteredHosts[0]
            }
        }
        // Use the available host info if present (filter operator skips undefined).
        // Otherwise, send the getHost query to the server.
        concat(of(hostInfo).pipe(filter((data) => data)), this.dhcpApi.getHost(id))
            .pipe(take(1))
            .subscribe(
                (data) => {
                    this.openedTabs.push(new Tab(HostForm, TabType.Display, data))
                    this.createMenuItem(this.getHostLabel(data), `/dhcp/hosts/${id}`)
                },
                (err) => {
                    const msg = getErrorMessage(err)
                    this.messageService.add({
                        severity: 'error',
                        summary: 'Cannot get host reservation',
                        detail: 'Error getting host reservation with ID ' + id + ': ' + msg,
                        life: 10000,
                    })
                }
            )
    }

    /**
     * Opens an existing or new host tab for creating new host.
     */
    private openNewHostTab() {
        let index = this.openedTabs.findIndex((t) => t.tabType === TabType.New)
        if (index >= 0) {
            this.switchToTab(index + 1)
            return
        }
        this.openedTabs.push(new Tab(HostForm, TabType.New))
        this.createMenuItem('New Host', '/dhcp/hosts/new')
        return
    }

    /**
     * Closes a tab.
     *
     * This function is called when user closes a selected host tab. If the
     * user a currently selected tab, a previous tab becomes selected.
     *
     * @param event event generated when the tab is closed.
     * @param tabIndex index of the tab to be closed. It must be equal to or
     *        greater than 1.
     */
    closeHostTab(event: Event, tabIndex: number) {
        if (
            this.openedTabs[tabIndex - 1].tabType === TabType.New &&
            this.openedTabs[tabIndex - 1].state.transactionId > 0 &&
            !this.openedTabs[tabIndex - 1].submitted
        ) {
            this.dhcpApi
                .createHostDelete(this.openedTabs[tabIndex - 1].state.transactionId)
                .toPromise()
                .catch((err) => {
                    let msg = err.statusText
                    if (err.error && err.error.message) {
                        msg = err.error.message
                    }
                    this.messageService.add({
                        severity: 'error',
                        summary: 'Failed to delete configuration transaction',
                        detail: 'Failed to delete configuration transaction: ' + msg,
                        life: 10000,
                    })
                })
        } else if (
            this.openedTabs[tabIndex - 1].tabType === TabType.Edit &&
            this.openedTabs[tabIndex - 1].tabSubject.id > 0 &&
            this.openedTabs[tabIndex - 1].state.transactionId > 0 &&
            !this.openedTabs[tabIndex - 1].submitted
        ) {
            this.dhcpApi
                .updateHostDelete(
                    this.openedTabs[tabIndex - 1].tabSubject.id,
                    this.openedTabs[tabIndex - 1].state.transactionId
                )
                .toPromise()
                .catch((err) => {
                    const msg = getErrorMessage(err)
                    this.messageService.add({
                        severity: 'error',
                        summary: 'Failed to delete configuration transaction',
                        detail: 'Failed to delete configuration transaction: ' + msg,
                        life: 10000,
                    })
                })
        }

        // Remove the MenuItem representing the tab.
        this.tabs = [...this.tabs.slice(0, tabIndex), ...this.tabs.slice(tabIndex + 1)]
        // Remove host specific information associated with the tab.
        this.openedTabs.splice(tabIndex - 1, 1)
        if (this.activeTabIndex === tabIndex) {
            // Closing currently selected tab. Switch to previous tab.
            this.switchToTab(tabIndex - 1)
            this.router.navigate([this.tabs[tabIndex - 1].routerLink])
        } else if (this.activeTabIndex > tabIndex) {
            // Sitting on the later tab then the one closed. We don't need
            // to switch, but we have to adjust the active tab index.
            this.activeTabIndex--
        }
        if (event) {
            event.preventDefault()
        }
    }

    /**
     * Selects an existing tab.
     *
     * @param tabIndex index of the tab to be selected.
     */
    private switchToTab(tabIndex: number) {
        if (this.activeTabIndex === tabIndex) {
            return
        }
        this.activeTabIndex = tabIndex
    }

    /**
     * Adds a new tab.
     *
     * @param label tab label.
     * @param routerLink tab router link.
     */
    private createMenuItem(label: string, routerLink: string) {
        this.tabs = [
            ...this.tabs,
            {
                label: label,
                routerLink: routerLink,
            },
        ]
        this.switchToTab(this.tabs.length - 1)
    }

    /**
     * Generates a host tab label.
     *
     * Different host reservation properties may be used to generate the label,
     * depending on their availability:
     * - first reserved IP address,
     * - first reserved delegated prefix,
     * - hostname,
     * - first DHCP identifier,
     * - host reservation ID.
     *
     * @param host host information from which the label should be generated.
     * @returns generated host label.
     */
    getHostLabel(host: Host) {
        if (host.addressReservations && host.addressReservations.length > 0) {
            return host.addressReservations[0].address
        }
        if (host.prefixReservations && host.prefixReservations.length > 0) {
            return host.prefixReservations[0].address
        }
        if (host.hostname && host.hostname.length > 0) {
            return host.hostname
        }
        if (host.hostIdentifiers && host.hostIdentifiers.length > 0) {
            return host.hostIdentifiers[0].idType + '=' + host.hostIdentifiers[0].idHexValue
        }
        return '[' + host.id + ']'
    }

    /**
     * Event handler triggered when a host form tab is being destroyed.
     *
     * The host form component is being destroyed and thus this parent
     * component must save the updated form data in case a user re-opens
     * the form tab.
     *
     * @param event an event holding updated form data.
     */
    onHostFormDestroy(event): void {
        // Find the form matching the form for which the notification has
        // been sent.
        const tab = this.openedTabs.find((t) => t.state && t.state.transactionId === event.transactionId)
        if (tab) {
            // Found the matching form. Update it.
            tab.state = event
        }
    }

    /**
     * Event handler triggered when a host form is submitted.
     *
     * It marks the form as submitted to prevent the component from canceling
     * the transaction. Next, it closes the form tab.
     *
     * @param event an event holding updated form data.
     */
    onHostFormSubmit(event): void {
        // Find the form matching the form for which the notification has
        // been sent.
        const index = this.openedTabs.findIndex((t) => t.state && t.state.transactionId === event.transactionId)
        if (index >= 0) {
            this.openedTabs[index].submitted = true
            this.closeHostTab(null, index + 1)
        }
    }

    /**
     * Event handler triggered when host form editing is canceled.
     *
     * If the event comes from the new host form, the tab is closed. If the
     * event comes from the host update form, the tab is turned into the
     * host view. In both cases, the transaction is deleted in the server.
     *
     * @param hostId host identifier or zero for new host case.
     */
    onHostFormCancel(hostId: number): void {
        // Find the form matching the form for which the notification has
        // been sent.
        const index = this.openedTabs.findIndex(
            (t) => t.tabSubject?.id === hostId || (t.tabType === TabType.New && !hostId)
        )
        if (index >= 0) {
            if (
                hostId &&
                this.openedTabs[index].state?.transactionId &&
                this.openedTabs[index].tabType !== TabType.Display
            ) {
                this.dhcpApi.updateHostDelete(hostId, this.openedTabs[index].state.transactionId).toPromise()
                this.tabs[index + 1].icon = ''
                this.openedTabs[index].setTabType(TabType.Display)
            } else {
                this.closeHostTab(null, index + 1)
            }
        }
    }

    /**
     * Event handler triggered when a user starts editing a host reservation.
     *
     * It replaces the host view with the host edit form in the current tab.
     *
     * @param host an instance carrying host information.
     */
    onHostEditBegin(host: Host): void {
        let index = this.openedTabs.findIndex(
            (t) => (t.tabType === TabType.Display || t.tabType === TabType.Edit) && t.tabSubject.id === host.id
        )
        if (index >= 0) {
            if (this.openedTabs[index].tabType !== TabType.Edit) {
                this.tabs[index + 1].icon = 'pi pi-pencil'
                this.openedTabs[index].setTabType(TabType.Edit)
            }
            this.switchToTab(index + 1)
        }
    }

    /**
     * Event handler triggered when a host was deleted using a delete
     * button on one of the tabs.
     *
     * @param host pointer to the deleted host.
     */
    onHostDelete(host: Host): void {
        // Try to find a suitable tab by host id.
        const index = this.openedTabs.findIndex((t) => t.tabSubject && t.tabSubject.id === host.id)
        if (index >= 0) {
            // Close the tab.
            this.closeHostTab(null, index + 1)
        }
    }
}
