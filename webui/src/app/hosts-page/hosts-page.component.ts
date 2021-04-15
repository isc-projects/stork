import { Component, OnInit, ViewChild } from '@angular/core'
import { Router, ActivatedRoute } from '@angular/router'

import { MenuItem, MessageService } from 'primeng/api'
import { Table } from 'primeng/table'

import { DHCPService } from '../backend/api/api'
import { extractKeyValsAndPrepareQueryParams } from '../utils'
import { concat, of } from 'rxjs'
import { filter, take } from 'rxjs/operators'

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
export class HostsPageComponent implements OnInit {
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
     * Pointer to the currently selected tab.
     */
    activeTab: MenuItem

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

    /**
     * Component lifecycle hook called upon initialization.
     *
     * It configures the component according to the parameters and the query parameters.
     * The id parameter can be set to all or be a numeric host identifier. In the former
     * case, a single tab holding a hosts list is displayed. In the latter case, a tab
     * with host details is automatically opened in addition to the hosts list tab.
     *
     * The query parameters control hosts filtering. If they are specified during the
     * component intitialization the hosts list will be filtered when it is first
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
        this.tabs = [{ label: 'Host Reservations', routerLink: '/dhcp/hosts' }]
        this.activeTab = this.tabs[0]

        // If filtering parameters are specified in the query, apply the filtering.
        this.initFilterText()

        // Subscribe to the changes of the filtering parameters.
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
                console.log(error)
            }
        )
        // Apply to the changes of the host id, e.g. from /dhcp/hosts/all to
        // /dhcp/hosts/1. Those changes are triggered by switching between the
        // tabs.
        this.route.paramMap.subscribe(
            (params) => {
                // Get host id.
                const id = params.get('id')
                if (id && id !== 'all') {
                    const numericId = parseInt(id, 10)
                    if (!Number.isNaN(numericId)) {
                        // The path has a numeric id indicating that we should
                        // open a tab with selected host information or switch
                        // to this tab if it has been already opened.
                        this.openHostTab(numericId)
                    }
                } else {
                    // The special id 'all' means: switch to hosts list.
                    this.switchToTab(0)
                }
            },
            (error) => {
                console.log(error)
            }
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
        for (let index = 0; index < this.openedTabs.length; index++) {
            if (this.openedTabs[index].host.id === id) {
                this.switchToTab(index + 1)
                return
            }
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
                    this.tabs.push({
                        label: this.getHostLabel(data),
                        routerLink: '/dhcp/hosts/' + id,
                    })
                    this.openedTabs.push({ host: data })
                    this.switchToTab(this.tabs.length - 1)
                },
                (err) => {
                    let msg = err.statusText
                    if (err.error && err.error.message) {
                        msg = err.error.message
                    }
                    this.messageService.add({
                        severity: 'error',
                        summary: 'Cannot get host reservation',
                        detail: 'Getting host reservation with ID ' + id + ' erred: ' + msg,
                        life: 10000,
                    })
                }
            )
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
        this.activeTab = this.tabs[tabIndex]
        this.activeTabIndex = tabIndex
    }

    /**
     * Loads hosts from the database into the component.
     *
     * @param event Event object containing an index if the first row, maximum
     *              number of rows to be returned and a text for hosts filtering.
     */
    loadHosts(event) {
        const params = this.queryParams

        this.dhcpApi.getHosts(event.first, event.rows, params.appId, null, params.text, params.global).subscribe(
            (data) => {
                this.hosts = data.items
                this.totalHosts = data.total
            },
            (err) => {
                let msg = err.statusText
                if (err.error && err.error.message) {
                    msg = err.error.message
                }
                this.messageService.add({
                    severity: 'error',
                    summary: 'Cannot get host reservations list',
                    detail: 'Getting host reservations list erred: ' + msg,
                    life: 10000,
                })
            }
        )
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
                return 'The server has this host specified in the configuration file.'
            case 'api':
                return 'The server has this host specified in the host database.'
            default:
                break
        }
        return ''
    }
}
