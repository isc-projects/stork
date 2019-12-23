import { Component, OnInit, Input, Output, EventEmitter } from '@angular/core'

import { MessageService, MenuItem } from 'primeng/api'

@Component({
    selector: 'app-kea-daemons-tabs',
    templateUrl: './kea-app-tab.component.html',
    styleUrls: ['./kea-app-tab.component.sass'],
})
export class KeaAppTabComponent implements OnInit {
    private _appTab: any
    @Output() refreshApp = new EventEmitter<number>()

    tabs: MenuItem[]
    activeTab: MenuItem
    daemons: any[] = []
    daemon: any

    constructor() {}

    ngOnInit() {
        console.info('this.app', this.appTab)
    }

    @Input()
    set appTab(appTab) {
        this._appTab = appTab

        const daemonMap = []
        for (const d of appTab.app.details.daemons) {
            daemonMap[d.name] = d
        }
        const DMAP = [
            ['dhcp4', 'DHCPv4'],
            ['dhcp6', 'DHCPv6'],
            ['d2', 'DDNS'],
            ['ca', 'CA'],
            ['netconf', 'NETCONF'],
        ]
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
        this.daemon = this.daemons[appTab.activeDaemonTabIdx]
        this.tabs = tabs
        this.activeTab = this.tabs[appTab.activeDaemonTabIdx]
    }

    get appTab() {
        return this._appTab
    }

    daemonTabSwitch(item) {
        for (const d of this.daemons) {
            if (d.niceName === item.label) {
                this.daemon = d
                break
            }
        }
    }

    refreshAppState() {
        this.refreshApp.emit(this._appTab.app.id)
    }
}
