import { Component, OnInit, Input, Output, EventEmitter } from '@angular/core'

import { MessageService, MenuItem } from 'primeng/api'

@Component({
    selector: 'app-kea-daemons-tabs',
    templateUrl: './kea-service-tab.component.html',
    styleUrls: ['./kea-service-tab.component.sass'],
})
export class KeaServiceTabComponent implements OnInit {
    private _serviceTab: any
    @Output() refreshService = new EventEmitter<number>()

    tabs: MenuItem[]
    activeTab: MenuItem
    daemons: any[] = []
    daemon: any

    constructor() {}

    ngOnInit() {
        console.info('this.service', this.serviceTab)
    }

    @Input()
    set serviceTab(serviceTab) {
        this._serviceTab = serviceTab

        const daemonMap = []
        for (const d of serviceTab.service.details.daemons) {
            daemonMap[d.name] = d
        }
        const DMAP = [['dhcp4', 'DHCPv4'], ['dhcp6', 'DHCPv6'], ['d2', 'DDNS'], ['ca', 'CA'], ['netconf', 'NETCONF']]
        const daemons = []
        const tabs = []
        for (const dm of DMAP) {
            if (daemonMap[dm[0]] !== undefined) {
                daemonMap[dm[0]].niceName = dm[1]
                daemons.push(daemonMap[dm[0]])

                tabs.push({
                    label: dm[1],
                    command: event => {
                        this.daemonTabSwitch(event.item)
                    },
                })
            }
        }
        this.daemons = daemons
        this.daemon = this.daemons[serviceTab.activeDaemonTabIdx]
        this.tabs = tabs
        this.activeTab = this.tabs[serviceTab.activeDaemonTabIdx]
    }

    get serviceTab() {
        return this._serviceTab
    }

    daemonTabSwitch(item) {
        for (const d of this.daemons) {
            if (d.niceName === item.label) {
                this.daemon = d
                break
            }
        }
    }

    refreshServiceState() {
        this.refreshService.emit(this._serviceTab.service.id)
    }
}
