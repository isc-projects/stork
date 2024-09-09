import { AfterViewInit, Component, OnDestroy, OnInit, ViewChild } from '@angular/core'
import { Router, ActivatedRoute, EventType } from '@angular/router'

import { DHCPService } from '../backend/api/api'
import { getErrorMessage } from '../utils'
import { parseSubnetsStatisticValues, extractUniqueSubnetPools, parseSubnetStatisticValues } from '../subnets'
import { SettingService } from '../setting.service'
import { Subscription, lastValueFrom, EMPTY } from 'rxjs'
import { catchError, filter, map } from 'rxjs/operators'
import { Settings, Subnet } from '../backend'
import { MenuItem, MessageService } from 'primeng/api'
import { SubnetFormState } from '../forms/subnet-form'
import { Tab, TabType } from '../tab'
import { SubnetsTableComponent } from '../subnets-table/subnets-table.component'

/**
 * Component for presenting DHCP subnets.
 */
@Component({
    selector: 'app-subnets-page',
    templateUrl: './subnets-page.component.html',
    styleUrls: ['./subnets-page.component.sass'],
})
export class SubnetsPageComponent implements OnInit, OnDestroy, AfterViewInit {
    /**
     * Enumeration for different subnet tab types displayed by the component.
     */
    SubnetTabType = TabType

    private subscriptions = new Subscription()
    breadcrumbs = [{ label: 'DHCP' }, { label: 'Subnets' }]

    /**
     * Table with subnets component.
     */
    @ViewChild('subnetsTableComponent') table: SubnetsTableComponent

    // Tab menu

    /**
     * Array of tab menu items with subnet information.
     *
     * The first tab is always present and displays the subnet list.
     *
     * Note: we cannot use the URL with no segment for the list tab. It causes
     * the first tab to be always marked active. The Tab Menu has a built-in
     * feature to highlight items based on the current route. It seems that it
     * matches by a prefix instead of an exact value (the "/foo/bar" URL
     * matches the menu item with the "/foo" URL).
     */
    tabs: MenuItem[] = [{ label: 'Subnets', routerLink: '/dhcp/subnets/all' }]

    /**
     * Selected tab menu index.
     *
     * The first tab has an index of 0.
     */
    activeTabIndex = 0

    /**
     * Indicates if the component is loading data from the server.
     */
    loading: boolean = false

    /**
     * Holds the information about specific subnets presented in the tabs.
     *
     * The entry corresponding to subnets list is not related to any specific
     * subnet. Its ID is 0.
     */
    openedTabs: Tab<SubnetFormState, Subnet>[] = [new Tab(SubnetFormState, TabType.List, { id: 0 })]

    grafanaUrl: string

    constructor(
        private route: ActivatedRoute,
        private router: Router,
        private dhcpApi: DHCPService,
        private settingSvc: SettingService,
        private messageService: MessageService
    ) {}

    ngOnDestroy(): void {
        this.subscriptions.unsubscribe()
    }

    ngOnInit() {
        // ToDo: Silent error catching
        this.subscriptions.add(
            this.settingSvc.getSettings().subscribe(
                (data: Settings) => {
                    this.grafanaUrl = data?.grafanaUrl
                },
                (error) => {
                    this.messageService.add({
                        severity: 'error',
                        summary: 'Cannot fetch server settings',
                        detail: getErrorMessage(error),
                    })
                }
            )
        )
    }

    /**
     * Component lifecycle hook called after Angular completed the initialization of the
     * component's view.
     *
     * We subscribe to router events to act upon URL and/or queryParams changes.
     * This is done at this step, because we have to be sure that all child components,
     * especially PrimeNG table in SubnetsTableComponent, are initialized.
     */
    ngAfterViewInit(): void {
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

                // Apply to the changes of the subnet id, e.g. from /dhcp/subnets/all to
                // /dhcp/subnets/1. Those changes are triggered by switching between the
                // tabs.

                // Get subnet id.
                const id = paramMap.get('id')
                if (!id || id === 'all') {
                    // Update the filter only if the target is subnet list.
                    this.table?.updateFilterFromQueryParameters(queryParamMap)
                    this.switchToTab(0)
                    return
                }
                if (id === 'new') {
                    this.openNewSubnetTab()
                    return
                }
                const numericId = parseInt(id, 10)
                if (!Number.isNaN(numericId)) {
                    // The path has a numeric id indicating that we should
                    // open a tab with selected subnet information or switch
                    // to this tab if it has been already opened.
                    this.openTabBySubnetId(numericId)
                } else {
                    // In case of failed Id parsing, open list tab.
                    this.switchToTab(0)
                    this.table?.loadDataWithoutFilter()
                }
            })
    }

    /**
     * Opens an existing or new subnet tab for creating a subnet.
     */
    openNewSubnetTab() {
        let index = this.openedTabs.findIndex((t) => t.tabType === TabType.New)
        if (index >= 0) {
            this.switchToTab(index)
            return
        }
        this.appendNewTab()
        this.switchToTab(this.openedTabs.length - 1)
    }

    /**
     * Open a subnet tab.
     *
     * If the tab already exists, switch to it without fetching the data.
     * Otherwise, fetch the subnet information from the API and create a
     * new tab.
     *
     * @param subnetId Subnet ID or a NaN for subnet list.
     */
    openTabBySubnetId(subnetId: number) {
        const tabIndex = this.openedTabs.map((t) => t.tabSubject?.id).indexOf(subnetId)
        if (tabIndex >= 0) {
            this.switchToTab(tabIndex)
            return
        }
        this.createTab(subnetId).then(() => {
            this.switchToTab(this.openedTabs.length - 1)
        })
    }

    /**
     * Close a menu tab by index.
     *
     * @param index Tab index.
     * @param event Event triggered upon tab closing.
     */
    closeTabByIndex(index: number, event?: Event) {
        if (index == 0) {
            return
        }

        if (
            this.openedTabs[index].tabType === TabType.Edit &&
            this.openedTabs[index].tabSubject?.id > 0 &&
            this.openedTabs[index].state?.transactionId > 0 &&
            !this.openedTabs[index].submitted
        ) {
            this.dhcpApi
                .updateSubnetDelete(this.openedTabs[index].tabSubject.id, this.openedTabs[index].state.transactionId)
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
            this.openedTabs[index].tabType === TabType.New &&
            this.openedTabs[index].state?.transactionId > 0 &&
            !this.openedTabs[index].submitted
        ) {
            this.dhcpApi
                .createSubnetDelete(this.openedTabs[index].state.transactionId)
                .toPromise()
                .catch((err) => {
                    let msg = getErrorMessage(err)
                    this.messageService.add({
                        severity: 'error',
                        summary: 'Failed to delete configuration transaction',
                        detail: 'Failed to delete configuration transaction: ' + msg,
                        life: 10000,
                    })
                })
        }

        this.openedTabs.splice(index, 1)
        this.tabs = [...this.tabs.slice(0, index), ...this.tabs.slice(index + 1)]

        if (this.activeTabIndex === index) {
            // Closing currently selected tab. Switch to previous tab.
            this.switchToTab(index - 1)
            this.router.navigate([this.tabs[index - 1].routerLink])
        } else if (this.activeTabIndex > index) {
            // Sitting on the later tab then the one closed. We don't need
            // to switch, but we have to adjust the active tab index.
            this.activeTabIndex--
        }

        if (event) {
            event.preventDefault()
        }
    }

    /**
     * Create a new tab for a given subnet ID.
     *
     * It fetches the subnet information from the API.
     *
     * @param subnetId Subnet Id.
     */
    private createTab(subnetId: number): Promise<void> {
        this.loading = true
        return (
            lastValueFrom(
                // Fetch data from API.
                this.dhcpApi.getSubnet(subnetId).pipe(
                    map((subnet) => {
                        if (subnet) {
                            parseSubnetStatisticValues(subnet)
                        }
                        return subnet
                    })
                )
            )
                // Execute and use.
                .then((data) => {
                    this.appendTab(data)
                })
                .catch((error) => {
                    const msg = getErrorMessage(error)
                    this.messageService.add({
                        severity: 'error',
                        summary: 'Cannot get subnet',
                        detail: `Error getting subnet with ID ${subnetId}: ${msg}`,
                        life: 10000,
                    })
                })
                .finally(() => {
                    this.loading = false
                })
        )
    }

    /**
     * Appends a tab for creating a new subnet.
     */
    private appendNewTab() {
        this.openedTabs.push(new Tab(SubnetFormState, TabType.New))
        this.tabs = [
            ...this.tabs,
            {
                label: 'New Subnet',
                icon: 'pi pi-pencil',
                routerLink: `/dhcp/subnets/new`,
            },
        ]
    }

    /**
     * Append a new tab to the list of tabs.
     *
     * @param subnet Subnet data.
     */
    private appendTab(subnet: Subnet) {
        this.openedTabs.push(new Tab(SubnetFormState, TabType.Display, subnet))
        this.tabs = [
            ...this.tabs,
            {
                label: subnet.subnet,
                routerLink: `/dhcp/subnets/${subnet.id}`,
            },
        ]
    }

    /**
     * Switch to tab identified by an index.
     *
     * @param index Tab index.
     */
    private switchToTab(index: number) {
        if (this.activeTabIndex == index) {
            return
        }
        this.activeTabIndex = index
    }

    /**
     * Event handler triggered when a user starts editing a subnet.
     *
     * It replaces the subnet view with the subnet edit form in the current tab.
     *
     * @param subnet an instance carrying subnet information.
     */
    onSubnetEditBegin(subnet): void {
        let index = this.openedTabs.findIndex(
            (t) => (t.tabType === TabType.Display || t.tabType === TabType.Edit) && t.tabSubject.id === subnet.id
        )
        if (index >= 0) {
            if (this.openedTabs[index].tabType !== TabType.Edit) {
                this.tabs[index].icon = 'pi pi-pencil'
                this.openedTabs[index].setTabType(TabType.Edit)
            }
            this.switchToTab(index)
        }
    }

    /**
     * Event handler triggered when user saves the edited subnet.
     *
     * @param event an event holding updated form data.
     */
    onSubnetFormSubmit(event): void {
        // Find the form matching the form for which the notification has
        // been sent.
        const index = this.openedTabs.findIndex((t) => t.state && t.state.transactionId === event.transactionId)
        if (index >= 0) {
            this.dhcpApi
                .getSubnet(event.subnetId)
                .pipe(
                    map((subnet) => {
                        if (subnet) {
                            parseSubnetsStatisticValues([subnet])
                            subnet = extractUniqueSubnetPools([subnet])[0]
                        }
                        return subnet
                    })
                )
                .toPromise()
                .then((subnet) => {
                    this.openedTabs[index].tabSubject = subnet
                    const existingIndex = this.table?.dataCollection?.findIndex((s) => s.id === subnet.id)
                    if (existingIndex >= 0) {
                        this.table.dataCollection[existingIndex] = subnet
                    }
                })
                .catch((error) => {
                    this.messageService.add({
                        severity: 'error',
                        summary: 'Cannot load updated subnet',
                        detail: getErrorMessage(error),
                    })
                })
                .finally(() => {
                    this.tabs[index].icon = ''
                    this.tabs[index].label = this.openedTabs[index].tabSubject.subnet
                    this.tabs[index].routerLink = `/dhcp/subnets/${event.subnetId}`
                    this.openedTabs[index].setTabType(TabType.Display)
                    this.router.navigate([this.tabs[index].routerLink])
                })
        }
    }

    /**
     * Event handler triggered when subnet form editing is canceled.
     *
     * If the event comes from the new subnet form, the tab is closed. If the
     * event comes from the subnet update form, the tab is turned into the
     * subnet view. In both cases, the transaction is deleted in the server.
     *
     * @param subnetId subnet identifier or null for new subnet case.
     */
    onSubnetFormCancel(subnetId?: number): void {
        const index = this.openedTabs.findIndex(
            (t) => t.tabSubject?.id === subnetId || (t.tabType === TabType.New && !subnetId)
        )
        if (index >= 0) {
            if (
                subnetId &&
                this.openedTabs[index].state?.transactionId &&
                this.openedTabs[index].tabType === TabType.Edit
            ) {
                this.dhcpApi
                    .updateSubnetDelete(subnetId, this.openedTabs[index].state.transactionId)
                    .toPromise()
                    .catch((err) => {
                        let msg = getErrorMessage(err)
                        this.messageService.add({
                            severity: 'error',
                            summary: 'Failed to delete configuration transaction',
                            detail: 'Failed to delete configuration transaction: ' + msg,
                            life: 10000,
                        })
                    })
                this.tabs[index].icon = ''
                this.openedTabs[index].setTabType(TabType.Display)
            } else {
                this.closeTabByIndex(index)
            }
        }
    }

    /**
     * Event handler triggered when a subnet form tab is being destroyed.
     *
     * The subnet form component is being destroyed and thus this parent
     * component must save the updated form data in case a user re-opens
     * the form tab.
     *
     * @param event an event holding updated form data.
     */
    onSubnetFormDestroy(event): void {
        const tab = this.openedTabs.find((t) => t.state?.transactionId === event.transactionId)
        if (tab) {
            // Found the matching form. Update it.
            tab.state = event
        }
    }

    /**
     * Event handler triggered when a subnet was deleted using a delete
     * button on one of the tabs.
     *
     * @param subnet pointer to the deleted subnet.
     */
    onSubnetDelete(subnet: Subnet): void {
        // Try to find a suitable tab by subnet id.
        const index = this.openedTabs.findIndex((t) => t.tabSubject && t.tabSubject.id === subnet.id)
        if (index >= 0) {
            // Close the tab.
            this.closeTabByIndex(index)
        }
    }
}
