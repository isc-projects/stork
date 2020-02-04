import { Component, OnInit, Input, Output, EventEmitter } from '@angular/core'

import moment from 'moment-timezone'

import { MessageService, MenuItem } from 'primeng/api'

import { DHCPService } from '../backend/api/api'

import { durationToString } from '../utils'

@Component({
    selector: 'app-kea-app-tab',
    templateUrl: './kea-app-tab.component.html',
    styleUrls: ['./kea-app-tab.component.sass'],
})
export class KeaAppTabComponent implements OnInit {
    private _appTab: any
    @Output() refreshApp = new EventEmitter<number>()

    daemons: any[] = []

    constructor(private dhcpApi: DHCPService) {}

    ngOnInit() {}

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
        for (const dm of DMAP) {
            if (daemonMap[dm[0]] !== undefined) {
                daemonMap[dm[0]].niceName = dm[1]
                daemonMap[dm[0]].subnets = []
                daemonMap[dm[0]].totalSubnets = 0
                daemons.push(daemonMap[dm[0]])
            }
        }
        this.daemons = daemons
    }

    get appTab() {
        return this._appTab
    }

    refreshAppState() {
        this.refreshApp.emit(this._appTab.app.id)
    }

    showDuration(duration) {
        return durationToString(duration)
    }

    loadSubnets(daemon, event) {
        let dhcpVer = 4
        if (daemon.name === 'dhcp6') {
            dhcpVer = 6
        }
        this.dhcpApi.getSubnets(event.first, event.rows, this._appTab.app.id, dhcpVer).subscribe(data => {
            daemon.subnets = data.items
            daemon.totalSubnets = data.total
        })
    }
}
