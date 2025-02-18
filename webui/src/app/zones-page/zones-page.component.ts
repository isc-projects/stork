import { ChangeDetectorRef, Component, OnInit, ViewChild } from '@angular/core'
import { MenuItem, MessageService } from 'primeng/api'
import { Zone, LocalZone, DNSService, ZonesFetchStatus, ZoneInventoryStates, ZoneInventoryState } from '../backend'
import { TabViewCloseEvent } from 'primeng/tabview'
import { lastValueFrom } from 'rxjs'
import { TableLazyLoadEvent } from 'primeng/table'
import { getErrorMessage } from '../utils'

type ZoneInventoryStatus = 'busy' | 'erred' | 'ok' | 'uninitialized' | string

@Component({
    selector: 'app-zones-page',
    templateUrl: './zones-page.component.html',
    styleUrl: './zones-page.component.sass',
})
export class ZonesPageComponent implements OnInit {
    /**
     * Configures the breadcrumbs for the component.
     */
    breadcrumbs: MenuItem[] = [{ label: 'DNS' }, { label: 'Zones' }]

    /**
     * Collection of zones fetched from backend.
     */
    zones: Zone[] = []

    dummyZones: Zone[] = []

    /**
     * Collection of open tabs of the tabView.
     */
    openTabs: Zone[] = []

    /**
     * Unique identifier of a stateful table to use in state storage.
     */
    stateKey: string = 'zones-table-state'

    /**
     * Keeps active tab index.
     */
    activeIdx: number = 0

    expandedRows = {}

    zoneInventoryStates: ZoneInventoryState[] = []

    zoneInventoryTotal: number = 0

    dateTimeFormat = 'YYYY-MM-dd HH:mm:ss'
    zonesLoading: boolean = false
    zonesTotal: number = 0
    inventoryLoading: boolean = false
    putZonesFetchSent: boolean
    inventoryInProgress: boolean = false
    inventoryAppsCompletedCount: number = 0
    inventoryTotalAppsCount: number = 0
    private timeout: any

    @ViewChild('zonesTable') zonesTable

    /**
     *
     * @param cd
     * @param dnsService
     * @param messageService
     */
    constructor(
        private cd: ChangeDetectorRef,
        private dnsService: DNSService,
        private messageService: MessageService
    ) {}

    /**
     *
     */
    ngOnInit(): void {
        const dummyLocalZones: LocalZone[] = []
        for (let i = 0; i < 100; i++) {
            const id = 100 + i * 2 // 100, 102, 104, 106,
            dummyLocalZones.push({
                appId: id,
                appName: `bind9@agent-bind9-${id}`,
                daemonId: 200 + i * 4, // 200, 204, 208, 212, 216,
                view: '_default',
                loadedAt: '2025-02-12T13:03:44.124Z',
                serial: 1000000 + i * 4, // 1000000, 1000004,
                _class: 'IN',
                zoneType: 'primary',
            })
            dummyLocalZones.push({
                appId: id,
                appName: `bind9@agent-bind9-${id}`,
                daemonId: 200 + i * 4, // 200, 204, 208, 212, 216,
                view: '_special',
                loadedAt: '2025-02-12T13:03:44.124Z',
                serial: 1000000 + i * 4 + 1, // 1000001, 1000005,
                _class: 'IN',
                zoneType: 'primary',
            })
            dummyLocalZones.push({
                appId: id,
                appName: `bind9@agent-bind9-${id}`,
                daemonId: 200 + i * 4 + 1, // 201, 205, 209, 213, 217,
                view: '_default',
                loadedAt: '2025-02-12T13:03:44.124Z',
                serial: 1000000 + i * 4 + 2, // 1000002, 1000006,
                _class: 'IN',
                zoneType: 'primary',
            })
            dummyLocalZones.push({
                appId: id + 1,
                appName: `bind9@agent-bind9-${id + 1}`,
                daemonId: 200 + i * 4 + 2, // 202, 206, 210, 214, 218,
                view: '_default',
                loadedAt: '2025-02-12T13:03:44.124Z',
                serial: 1000000 + i * 4 + 3, // 1000003, 1000007,
                _class: 'IN',
                zoneType: 'secondary',
            })
        }

        this.dummyZones = [
            { id: 1, name: 'this.is.example.org', rname: 'org.example.is.this', localZones: dummyLocalZones },
            { id: 2, name: 'this.is.example.com', rname: 'com.example.is.this', localZones: dummyLocalZones },
            { id: 3, name: 'this.example.org', rname: 'org.example.this', localZones: dummyLocalZones },
            { id: 4, name: 'foo.bar.org', rname: 'org.bar.foo', localZones: dummyLocalZones },
            { id: 5, name: 'example.org', rname: 'org.example', localZones: [dummyLocalZones[0]] },
        ]

        this.getZoneInventoryState()
    }

    /**
     * Callback called when tab is closed.
     * @param event
     */
    onTabClose(event: TabViewCloseEvent) {
        this.openTabs.splice(event.index - 1, 1)
        if (event.index <= this.activeIdx) {
            this.activeIdx = 0
        }
        console.log('onTabClose', event, 'openTabs', this.openTabs, 'activeIdx', this.activeIdx)
    }

    /**
     * Opens tab with zone details.
     * @param zone
     */
    openTab(zone: Zone) {
        const openTabsZoneIds = this.openTabs.map((z) => z.id)
        const zoneIdx = openTabsZoneIds.indexOf(zone.id)
        if (zoneIdx >= 0) {
            // tab exists, just switch to it
            this.activeIdx = zoneIdx + 1
        } else {
            this.openTabs = [...this.openTabs, zone]
            this.cd.detectChanges()
            console.log('openTab setting activeIdx to', this.openTabs.length)
            this.activeIdx = this.openTabs.length
        }
    }

    /**
     * Callback called when active index changes.
     * @param indexAfterChange
     */
    onActiveIndexChange(indexAfterChange: number) {
        console.log('onActiveIdxChange', indexAfterChange, 'this.activeIdx', this.activeIdx)
    }

    getZoneInventoryState() {
        this.inventoryLoading = true
        lastValueFrom(this.dnsService.getZoneInventoryStates())
            .then((resp: ZoneInventoryStates | ZonesFetchStatus) => {
                console.log('getZoneInventoryState promise then', resp)
                if (!resp) {
                    this.messageService.add({
                        severity: 'info',
                        summary: 'Zone Inventory empty',
                        detail: 'No state is currently available, presumably because the zones have not been fetched from Stork agents.',
                    })
                } else if ('completedAppsCount' in resp && 'appsCount' in resp) {
                    this.zoneInventoryStates = []
                    this.zoneInventoryTotal = 0
                    this.inventoryInProgress = true
                    this.inventoryAppsCompletedCount = resp.completedAppsCount
                    this.inventoryTotalAppsCount = resp.appsCount
                    console.log(
                        `getZoneInventoryState: zones fetch status ${resp.completedAppsCount} of ${resp.appsCount} fetched`
                    )
                    if (resp.appsCount > resp.completedAppsCount) {
                        this.timeout = setTimeout(() => {
                            this.getZoneInventoryState()
                        }, 10000)
                    }
                } else if ('items' in resp && 'total' in resp) {
                    if (this.inventoryInProgress) {
                        this.inventoryAppsCompletedCount = this.inventoryTotalAppsCount
                        this.messageService.add({
                            severity: 'success',
                            summary: 'Zone Inventory Process done',
                            detail: 'Zone Inventory Process finished successfully!',
                        })
                    }

                    this.inventoryInProgress = false
                    clearTimeout(this.timeout)
                    this.zoneInventoryStates = resp.items ?? []
                    this.zoneInventoryTotal = resp.total ?? 0
                    this.onLazyLoadZones(this.zonesTable?.createLazyLoadMetadata())
                }
            })
            .catch((err) => {
                const msg = getErrorMessage(err)
                this.messageService.add({
                    severity: 'error',
                    summary: 'Error sending request',
                    detail: 'Sending $Check Zone Inventory State$ request failed: ' + msg,
                    life: 10000,
                })
            })
            .finally(() => {
                console.log('getZoneInventoryState promise finally')
                this.inventoryLoading = false
            })
    }

    fetchZones() {
        lastValueFrom(this.dnsService.putZonesFetch())
            .then(() => {
                this.putZonesFetchSent = true
                this.messageService.add({
                    severity: 'success',
                    summary: 'Request sent',
                    detail: 'Sending $Fetch Zones$ request succeeded.',
                })
                // if (this.zoneInventoryTotal === 0) {
                this.getZoneInventoryState()
                // }
            })
            .catch((err) => {
                const msg = getErrorMessage(err)
                this.messageService.add({
                    severity: 'error',
                    summary: 'Error sending request',
                    detail: 'Sending $Fetch Zones$ request failed: ' + msg,
                    life: 10000,
                })
            })
    }

    getSeverity(status: ZoneInventoryStatus) {
        switch (status) {
            case 'ok':
                return 'success'
            case 'busy':
                return 'warning'
            case 'erred':
                return 'danger'
            case 'uninitialized':
                return 'secondary'
            default:
                return 'info'
        }
    }

    getStateErrorMessage(err: string) {
        return `Error when communicating with a zone inventory on an agent: ${err}.`
    }

    getTooltip(status: ZoneInventoryStatus) {
        switch (status) {
            case 'busy':
                return 'Zone inventory on an agent performs a long lasting operation and cannot perform the requested operation at this time.'
            case 'ok':
                return 'Communication with the zone inventory was successful.'
            case 'erred':
                return 'Error when communicating with a zone inventory on an agent.'
            case 'uninitialized':
                return 'Zone inventory was not initialized (neither populated nor loaded).'
            default:
                return null
        }
    }

    onLazyLoadZones(event: TableLazyLoadEvent) {
        this.zonesLoading = true
        this.cd.detectChanges()
        lastValueFrom(this.dnsService.getZones(event?.first ?? 0, event?.rows ?? 10))
            .then((resp) => {
                this.expandedRows = {}
                this.zones = resp?.items ?? []
                this.zonesTotal = resp?.total ?? 0
            })
            .catch((err) => {
                const msg = getErrorMessage(err)
                this.messageService.add({
                    severity: 'error',
                    summary: 'Error retrieving data',
                    detail: 'Retrieving Zones data failed: ' + msg,
                    life: 10000,
                })
            })
            .finally(() => (this.zonesLoading = false))
    }
}
