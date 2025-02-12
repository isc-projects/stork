import { ChangeDetectorRef, Component, OnInit } from '@angular/core'
import { MenuItem } from 'primeng/api'
import { Zone, LocalZone } from '../backend'
import { TabViewCloseEvent } from 'primeng/tabview'

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

    /**
     *
     * @param cd
     */
    constructor(private cd: ChangeDetectorRef) {}

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
                serial: 1000000 + i * 4, // 1000000, 1000004, .08, .12, .16,
            })
            dummyLocalZones.push({
                appId: id,
                appName: `bind9@agent-bind9-${id}`,
                daemonId: 200 + i * 4, // 200, 204, 208, 212, 216,
                view: '_special',
                loadedAt: '2025-02-12T13:03:44.124Z',
                serial: 1000000 + i * 4 + 1, // 1000001, 1000005, .09, .13, .17,
            })
            dummyLocalZones.push({
                appId: id,
                appName: `bind9@agent-bind9-${id}`,
                daemonId: 200 + i * 4 + 1, // 201, 205, 209, 213, 217,
                view: '_default',
                loadedAt: '2025-02-12T13:03:44.124Z',
                serial: 1000000 + i * 4 + 2, // 1000002, 1000006, .10, .14, .18,
            })
            dummyLocalZones.push({
                appId: id + 1,
                appName: `bind9@agent-bind9-${id + 1}`,
                daemonId: 200 + i * 4 + 2, // 202, 206, 210, 214, 218,
                view: '_default',
                loadedAt: '2025-02-12T13:03:44.124Z',
                serial: 1000000 + i * 4 + 3, // 1000003, 1000007, .11, .15, .19,
            })
        }

        this.zones = [
            { id: 1, name: 'this.is.example.org', rname: 'org.example.is.this', localZones: dummyLocalZones },
            { id: 2, name: 'this.is.example.com', rname: 'com.example.is.this', localZones: dummyLocalZones },
            { id: 3, name: 'this.example.org', rname: 'org.example.this', localZones: dummyLocalZones },
            { id: 4, name: 'foo.bar.org', rname: 'org.bar.foo', localZones: dummyLocalZones },
            { id: 5, name: 'example.org', rname: 'org.example', localZones: [dummyLocalZones[0]] },
        ]
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
}
