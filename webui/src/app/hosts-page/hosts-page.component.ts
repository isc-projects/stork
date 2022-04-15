import { Component, OnDestroy, OnInit, ViewChild } from '@angular/core'
import { Router, ActivatedRoute } from '@angular/router'

import { MenuItem, MessageService } from 'primeng/api'
import { Table } from 'primeng/table'

import { DHCPService } from '../backend/api/api'
import { extractKeyValsAndPrepareQueryParams } from '../utils'
import { concat, of, Subscription } from 'rxjs'
import { filter, take } from 'rxjs/operators'

/**
 * Enumeration for different host tab types displayed by the component.
 */
export enum HostTabType {
    List = 1,
    NewHost,
    Host,
}

/**
 * A class representing the contents of a tab displayed by the component.
 */
export class HostTab {
    form: any = {}

    /**
     * Constructor.
     *
     * @param tabType host tab type.
     * @param host host information displayed in the tab.
     */
    constructor(public tabType: HostTabType, public host?: any) {
        this.form = {}
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

    // hosts
    hosts: any[]
    totalHosts = 0

    // filters
    filterText = ''
    queryParams = {
        text: null,
        appId: null,
        global: null,
    }

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

        // If filtering parameters are specified in the query, apply the filtering.
        this.initFilterText()

        // Subscribe to the changes of the filtering parameters.
        this.subscriptions.add(
            this.route.queryParamMap.subscribe(
                (params) => {
                    this.updateQueryParams(params)
                    let event = { first: 0, rows: 10 }
                    if (this.hostsTable) {
                        event = this.hostsTable.createLazyLoadMetadata()
                    }
                    this.loadHosts(event)
                },
                (error) => {
                    // ToDo: Fix silent error catching
                    console.log(error)
                }
            )
        )
        // Apply to the changes of the host id, e.g. from /dhcp/hosts/all to
        // /dhcp/hosts/1. Those changes are triggered by switching between the
        // tabs.
        this.subscriptions.add(
            this.route.paramMap.subscribe(
                (params) => {
                    // Get host id.
                    const id = params.get('id')
                    if (!id || id === 'all') {
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
                },
                (error) => {
                    console.log(error)
                }
            )
        )
    }

    /**
     * Apply filtering according to the query parameters.
     *
     * The following parameters are taken into account:
     * - text
     * - appId
     * - global (translated to is:global or not:global filtering text).
     */
    private initFilterText() {
        const ssParams = this.route.snapshot.queryParamMap
        let text = ''
        if (ssParams.get('text')) {
            text += ' ' + ssParams.get('text')
        }
        if (ssParams.get('appId')) {
            text += ' appId:' + ssParams.get('appId')
        }
        const g = ssParams.get('global')
        if (g === 'true') {
            text += ' is:global'
        } else if (g === 'false') {
            text += ' not:global'
        }
        this.filterText = text.trim()
    }

    /**
     * Updates queryParams structure using query parameters.
     *
     * This update is triggered when user types in the filter box.
     * @param params query parameters received from activated route.
     */
    private updateQueryParams(params) {
        this.queryParams.text = params.get('text')
        this.queryParams.appId = params.get('appId')
        const g = params.get('global')
        if (g === 'true') {
            this.queryParams.global = true
        } else if (g === 'false') {
            this.queryParams.global = false
        } else {
            this.queryParams.global = null
        }
    }

    /**
     * Opens existing or new host tab.
     *
     * If the host tab for the given host ID does not exist, a new tab is opened.
     * Otherwise, the existing tab is opened.
     *
     * @param id host ID.
     */
    private openHostTab(id) {
        let index = this.openedTabs.findIndex((t) => t.tabType === HostTabType.Host && t.host.id === id)
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
                    let msg = err.statusText
                    if (err.error && err.error.message) {
                        msg = err.error.message
                    }
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
    closeHostTab(event, tabIndex) {
        // Remove the MenuItem representing the tab.
        this.tabs.splice(tabIndex, 1)
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
    private switchToTab(tabIndex) {
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
    private createMenuItem(label: string, routerLink: string): any {
        this.tabs.push({
            label: label,
            routerLink: routerLink,
        })
        this.switchToTab(this.tabs.length - 1)
    }

    /**
     * Loads hosts from the database into the component.
     *
     * @param event Event object containing an index if the first row, maximum
     *              number of rows to be returned and a text for hosts filtering.
     */
    loadHosts(event) {
        const params = this.queryParams

        this.dhcpApi
            .getHosts(event.first, event.rows, params.appId, null, params.text, params.global)
            .toPromise()
            .then((data) => {
                this.hosts = data.items
                this.totalHosts = data.total
            })
            .catch((err) => {
                let msg = err.statusText
                if (err.error && err.error.message) {
                    msg = err.error.message
                }
                this.messageService.add({
                    severity: 'error',
                    summary: 'Cannot get host reservations list',
                    detail: 'Error getting host reservations list: ' + msg,
                    life: 10000,
                })
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
    getHostLabel(host) {
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
     * Filters the list of hosts by text. The text may contain key=val
     * pairs allowing filtering by various keys. Filtering is realized
     * server-side.
     */
    keyUpFilterText(event) {
        if (this.filterText.length >= 2 || event.key === 'Enter') {
            const queryParams = extractKeyValsAndPrepareQueryParams(this.filterText, ['appId'], ['global'])
            this.router.navigate(['/dhcp/hosts'], {
                queryParams,
                queryParamsHandling: 'merge',
            })
        }
    }

    /**
     * Returns tooltip explaining where the server has the given host
     * reservation specified, i.e. in the configuration file or a database.
     *
     * @param dataSource data source provided as a string.
     * @returns The tooltip text.
     */
    hostDataSourceTooltip(dataSource): string {
        switch (dataSource) {
            case 'config':
                return "This host is specified in the server's configuration file."
            case 'api':
                return "This host is specified in the server's host database."
            default:
                break
        }
        return ''
    }

    /**
     * Event handler triggered when a host form tab is being closed.
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
}
