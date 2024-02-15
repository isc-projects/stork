import { Component, OnDestroy, OnInit, ViewChild } from '@angular/core'
import { Router, ActivatedRoute, ParamMap, EventType } from '@angular/router'

import { MenuItem, MessageService } from 'primeng/api'
import { Table } from 'primeng/table'

import { DHCPService } from '../backend/api/api'
import { extractKeyValsAndPrepareQueryParams, getErrorMessage } from '../utils'
import { concat, EMPTY, of, Subject, Subscription } from 'rxjs'
import { catchError, filter, map, take } from 'rxjs/operators'
import { HostForm } from '../forms/host-form'
import { Host, LocalHost } from '../backend'
import { hasDifferentLocalHostData } from '../hosts'
import {
    QueryParamsFilter,
    getBooleanQueryParamsFilterKeys,
    getNumericQueryParamsFilterKeys,
} from './query-params-filter'
import { Location } from '@angular/common'

/**
 * Enumeration for different host tab types displayed by the component.
 */
export enum HostTabType {
    List = 1,
    NewHost,
    EditHost,
    Host,
}

/**
 * A class representing the contents of a tab displayed by the component.
 */
export class HostTab {
    /**
     * Preserves information specified in a host form.
     */
    form: HostForm

    /**
     * Indicates if the form has been submitted.
     */
    submitted = false

    /**
     * Constructor.
     *
     * @param tabType host tab type.
     * @param host host information displayed in the tab.
     */
    constructor(
        public tabType: HostTabType,
        public host?: Host
    ) {
        this._setHostTabType(tabType)
    }

    /**
     * Sets new host tab type and initializes the form accordingly.
     *
     * It is a private function variant that does not check whether the type
     * is already set to the desired value.
     */
    private _setHostTabType(tabType: HostTabType): void {
        switch (tabType) {
            case HostTabType.NewHost:
            case HostTabType.EditHost:
                this.form = new HostForm()
                break
            default:
                this.form = null
                break
        }
        this.submitted = false
        this.tabType = tabType
    }

    /**
     * Sets new host tab type and initializes the form accordingly.
     *
     * It does nothing when the type is already set to the desired value.
     */
    public setHostTabType(tabType: HostTabType): void {
        if (this.tabType === tabType) {
            return
        }
        this._setHostTabType(tabType)
    }
}

/**
 * This component implements a page which displays hosts along with
 * their DHCP identifiers and IP reservations. The list of hosts is
 * paged and can be filtered by a reserved IP address. The list
 * contains host reservations for all subnets and in the future it
 * will also contain global reservations, i.e. those that are not
 * associated with any particular subnet.
 */
@Component({
    selector: 'app-hosts-page',
    templateUrl: './hosts-page.component.html',
    styleUrls: ['./hosts-page.component.sass'],
})
export class HostsPageComponent implements OnInit, OnDestroy {
    private subscriptions = new Subscription()
    @ViewChild('hostsTable') hostsTable: Table

    breadcrumbs = [{ label: 'DHCP' }, { label: 'Host Reservations' }]

    /**
     * Holds all currently displayed host reservations.
     */
    _hosts: Host[]

    /**
     * Holds local hosts of all currently displayed host reservations grouped by app ID.
     * It is indexed by host ID.
     */
    localHostsGroupedByApp: Record<number, LocalHost[][]>

    /**
     * Returns all currently displayed host reservations.
     */
    get hosts(): Host[] {
        return this._hosts
    }

    /**
     * Sets hosts reservations to be displayed.
     * Groups the local hosts by app ID and stores the result in
     * @this.localHostsGroupedByApp.
     */
    set hosts(hosts: Host[]) {
        this._hosts = hosts

        // For each host group the local hosts by app ID.
        this.localHostsGroupedByApp = Object.fromEntries(
            (hosts || []).map((host) => {
                if (!host.localHosts) {
                    return [host.id, []]
                }

                return [
                    host.id,
                    Object.values(
                        // Group the local hosts by app ID.
                        host.localHosts.reduce<Record<number, LocalHost[]>>((accApp, localHost) => {
                            if (!accApp[localHost.appId]) {
                                accApp[localHost.appId] = []
                            }

                            accApp[localHost.appId].push(localHost)

                            return accApp
                        }, {})
                    ),
                ]
            })
        )
    }

    /**
     * Holds the counter of all hosts.
     */
    totalHosts = 0

    /**
     * Indicates if the hosts are being fetched from the server.
     */
    hostsLoading = false

    /**
     * The filter input box content.
     */
    filterText: string = ''

    /**
     * The provided filter.
     * The source property indicates where the filter comes from:
     * - input - the filter is set by the user in the input box,
     * - callback - the filter is set by the child component,
     * - query - the filter is set by the URL query parameters.
     */
    hostFilter$ = new Subject<{ source: 'input' | 'callback' | 'query'; filter: QueryParamsFilter }>()

    /**
     * The recent filter applied to the hosts list. Only filters that pass the
     * validation are used.
     */
    validHostFilter: QueryParamsFilter = {}

    /**
     * Array of tabs with host information.
     *
     * The first tab is always present and displays the hosts list.
     */
    tabs: MenuItem[]

    /**
     * Enumeration for different tab types displayed in this component.
     */
    HostTabType = HostTabType

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
    openedTabs = []

    /**
     * An array of errors in specifying filter text.
     *
     * This array holds errors displayed next to the host filtering input box.
     */
    filterTextFormatErrors: string[] = []

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
        private messageService: MessageService,
        private location: Location
    ) {}

    ngOnDestroy(): void {
        this.hostFilter$.complete()
        this.subscriptions.unsubscribe()
    }

    /**
     * Component lifecycle hook called upon initialization.
     *
     * It configures the component according to the parameters and the query parameters.
     * The id parameter can be set to all or be a numeric host identifier. In the former
     * case, a single tab holding a hosts list is displayed. In the latter case, a tab
     * with host details is automatically opened in addition to the hosts list tab.
     *
     * The query parameters control hosts filtering. If they are specified during the
     * component initialization the hosts list will be filtered when it is first
     * displayed and the filters will be visible in the filtering box. This is useful
     * when a user is directed from other views after clicking on a link and wants to
     * see only selected host reservations.
     *
     * This function also subscribes to changes in the parameters and query parameters
     * which allows for dynamically changing the content, e.g. as a result of selecting
     * one of the tabs or applying hosts list filtering.
     */
    ngOnInit() {
        // Initially, there is only a tab with hosts list.
        this.tabs = [{ label: 'Host Reservations', routerLink: '/dhcp/hosts/all' }]

        // Pipe the valid filter to the hostFilter$ subject.
        this.subscriptions.add(
            this.hostFilter$
                .pipe(
                    // Valid filter has no validation errors.
                    filter((f) => this.validateFilter(f.filter).length === 0),
                    map((f) => f.filter)
                )
                .subscribe((filter) => {
                    // Remember the filter.
                    this.validHostFilter = filter
                    // Update the list of hosts when the filtering parameters change.
                    this.loadHosts()
                })
        )

        // Update the filter representation when the filtering parameters change.
        this.subscriptions.add(
            this.hostFilter$.subscribe((f) => {
                // Update the URL.
                if (f.source != 'query') {
                    this.updateQueryParameters(f.filter)
                }
                // Update the input box.
                if (f.source != 'input') {
                    this.updateFilterText(f.filter)
                }

                this.filterTextFormatErrors = this.validateFilter(f.filter)
            })
        )

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
                        this.updateFilterFromQueryParameters(queryParamMap)
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
                    }
                })
        )
    }

    /**
     * Updates the filter input content based on the provided filter.
     *
     * The numeric and booleans parameters are taken into account. The boolean
     * ones are translated to is:foo or not:foo filtering text.
     */
    private updateFilterText(filter: QueryParamsFilter) {
        const numericKeys = getNumericQueryParamsFilterKeys()
        const booleanKeys = getBooleanQueryParamsFilterKeys()
        const parameters = []

        for (let key of numericKeys) {
            if (filter.hasOwnProperty(key)) {
                parameters.push(` ${key}:${filter[key]}`)
            }
        }

        for (let key of booleanKeys) {
            if (filter.hasOwnProperty(key)) {
                if (filter[key] === true) {
                    parameters.push(`is:${key}`)
                } else if (filter[key] === false) {
                    parameters.push(`not:${key}`)
                } else {
                    parameters.push(`:${filter[key]}`)
                }
            }
        }

        if (filter.text) {
            parameters.push(filter.text)
        }
        this.filterText = parameters.map((p) => p.trim()).join(' ')
    }

    /**
     * Update the URL query parameters based on the provided filter.
     *
     * This function uses the Location provider instead Router or
     * ActivatedRoute to avoid re-rendering the component.
     */
    private updateQueryParameters(filter: QueryParamsFilter) {
        const params = []

        for (let key of Object.keys(filter)) {
            if (filter[key] != null) {
                params.push(`${encodeURIComponent(key)}=${encodeURIComponent(filter[key])}`)
            }
        }

        const baseUrl = this.router.url.split('?')[0]
        this.location.go(baseUrl, params.join('&'))
    }

    /**
     * Updates the filter structure using URL query parameters.
     *
     * This update is triggered when the URL changes.
     * @param params query parameters received from activated route.
     */
    private updateFilterFromQueryParameters(params: ParamMap) {
        const numericKeys = getNumericQueryParamsFilterKeys()
        const booleanKeys = getBooleanQueryParamsFilterKeys()

        const filter: QueryParamsFilter = {}
        filter.text = params.get('text')

        for (let key of numericKeys) {
            // Convert the value to a number. It is NaN if the parameter
            // doesn't exist or it is malformed.
            if (params.has(key)) {
                const value = parseInt(params.get(key))
                filter[key as any] = isNaN(value) ? null : value
            }
        }

        const parseBoolean = (val: string) => (val === 'true' ? true : val === 'false' ? false : null)

        for (let key of booleanKeys) {
            if (params.has(key)) {
                const value = parseBoolean(params.get(key))
                filter[key as any] = value
            }
        }

        this.hostFilter$.next({
            source: 'query',
            filter: filter,
        })
    }

    /**
     * Checks if the provided filter is valid.
     * @param filter A filter to validate
     * @returns List of validation issues. If the list is empty, the filter is
     * valid.
     */
    private validateFilter(filter: QueryParamsFilter): string[] {
        const numericKeys = getNumericQueryParamsFilterKeys()
        const booleanKeys = getBooleanQueryParamsFilterKeys()

        const errors: string[] = []

        for (let key of numericKeys) {
            if (filter.hasOwnProperty(key) && filter[key] == null) {
                errors.push(`Please specify ${key} as a number (e.g., ${key}:2).`)
            }
        }

        for (let key of booleanKeys) {
            if (filter.hasOwnProperty(key) && filter[key] == null) {
                errors.push(`Please specify ${key} as a boolean (e.g., is:${key} or not:${key}).`)
            }
        }

        return errors
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
            (t) => (t.tabType === HostTabType.Host || t.tabType === HostTabType.EditHost) && t.host.id === id
        )
        if (index >= 0) {
            this.switchToTab(index + 1)
            return
        }
        // Check if the host info is already available.
        let hostInfo: any
        if (this.hosts) {
            const filteredHosts = this.hosts.filter((host) => host.id === id)
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
                    this.openedTabs.push(new HostTab(HostTabType.Host, data))
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
        let index = this.openedTabs.findIndex((t) => t.tabType === HostTabType.NewHost)
        if (index >= 0) {
            this.switchToTab(index + 1)
            return
        }
        this.openedTabs.push(new HostTab(HostTabType.NewHost))
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
            this.openedTabs[tabIndex - 1].tabType === HostTabType.NewHost &&
            this.openedTabs[tabIndex - 1].form.transactionId > 0 &&
            !this.openedTabs[tabIndex - 1].submitted
        ) {
            this.dhcpApi
                .createHostDelete(this.openedTabs[tabIndex - 1].form.transactionId)
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
            this.openedTabs[tabIndex - 1].tabType === HostTabType.EditHost &&
            this.openedTabs[tabIndex - 1].host.id > 0 &&
            this.openedTabs[tabIndex - 1].form.transactionId > 0 &&
            !this.openedTabs[tabIndex - 1].submitted
        ) {
            this.dhcpApi
                .updateHostDelete(
                    this.openedTabs[tabIndex - 1].host.id,
                    this.openedTabs[tabIndex - 1].form.transactionId
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
     * Loads hosts from the database into the component.
     *
     * @param event Event object containing an index if the first row, maximum
     * number of rows to be returned and a text for hosts filtering. If it is
     * not specified, the current values are used when available.
     */
    loadHosts(event?) {
        const params = this.validHostFilter
        if (typeof event === 'undefined') {
            event = { first: 0, rows: 10 }
            if (this.hostsTable) {
                // When hosts are filtered, go by default to first page after filtering.
                this.hostsTable.first = 0
                event = this.hostsTable.createLazyLoadMetadata()
            }
        }
        // Indicate that hosts refresh is in progress.
        this.hostsLoading = true
        this.dhcpApi
            .getHosts(
                event.first,
                event.rows,
                params.appId,
                params.subnetId,
                params.keaSubnetId,
                params.text,
                params.global,
                params.conflict
            )
            .toPromise()
            .then((data) => {
                this.hosts = data.items ?? []
                this.totalHosts = data.total ?? 0
            })
            .catch((err) => {
                const msg = getErrorMessage(err)
                this.messageService.add({
                    severity: 'error',
                    summary: 'Cannot get host reservations list',
                    detail: 'Error getting host reservations list: ' + msg,
                    life: 10000,
                })
            })
            .finally(() => {
                this.hostsLoading = false
            })
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
     * Returns the state of the local hosts from the same application/daemon.
     * The state is null if the host reservations are defined only in the
     * configuration file or host database. If they are defined in both places
     * the state is one of the following:
     * - duplicate - reservations have the same boot fields, client classes, and
     *               DHCP options
     * - conflict - reservations are configured differently.
     *
     * @param localHosts local hosts to be checked.
     */
    getLocalHostsState(localHosts: LocalHost[]): 'conflict' | 'duplicate' | null {
        if (localHosts.length <= 1) {
            return null
        }
        if (hasDifferentLocalHostData(localHosts)) {
            return 'conflict'
        } else {
            return 'duplicate'
        }
    }

    /**
     * Filters the list of hosts by text. The text may contain key=val
     * pairs allowing filtering by various keys. Filtering is realized
     * server-side.
     */
    keyUpFilterText(event: Pick<KeyboardEvent, 'key'>) {
        if (this.filterText.length >= 2 || event.key === 'Enter') {
            const filter = extractKeyValsAndPrepareQueryParams<QueryParamsFilter>(
                this.filterText,
                getNumericQueryParamsFilterKeys(),
                getBooleanQueryParamsFilterKeys()
            )

            this.hostFilter$.next({
                source: 'input',
                filter: filter,
            })
        }
    }

    /**
     * Event handler triggered when a host list needs to be filtered.
     */
    onRequestedFiltering(filter: QueryParamsFilter) {
        this.hostFilter$.next({
            source: 'callback',
            filter,
        })
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
        const tab = this.openedTabs.find((t) => t.form && t.form.transactionId === event.transactionId)
        if (tab) {
            // Found the matching form. Update it.
            tab.form = event
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
        const index = this.openedTabs.findIndex((t) => t.form && t.form.transactionId === event.transactionId)
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
            (t) => (t.host && t.host.id === hostId) || (t.tabType === HostTabType.NewHost && !hostId)
        )
        if (index >= 0) {
            if (
                hostId &&
                this.openedTabs[index].form?.transactionId &&
                this.openedTabs[index].tabType !== HostTabType.Host
            ) {
                this.dhcpApi.updateHostDelete(hostId, this.openedTabs[index].form.transactionId).toPromise()
                this.tabs[index + 1].icon = ''
                this.openedTabs[index].setHostTabType(HostTabType.Host)
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
            (t) => (t.tabType === HostTabType.Host || t.tabType === HostTabType.EditHost) && t.host.id === host.id
        )
        if (index >= 0) {
            if (this.openedTabs[index].tabType !== HostTabType.EditHost) {
                this.tabs[index + 1].icon = 'pi pi-pencil'
                this.openedTabs[index].setHostTabType(HostTabType.EditHost)
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
        const index = this.openedTabs.findIndex((t) => t.host && t.host.id === host.id)
        if (index >= 0) {
            // Close the tab.
            this.closeHostTab(null, index + 1)
        }
    }
}
