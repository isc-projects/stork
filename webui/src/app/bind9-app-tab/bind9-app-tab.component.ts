import { Component, OnInit, Input, Output, EventEmitter } from '@angular/core'

import { MessageService, MenuItem } from 'primeng/api'

@Component({
    selector: 'app-bind9-app-tab',
    templateUrl: './bind9-app-tab.component.html',
    styleUrls: ['./bind9-app-tab.component.sass'],
})
export class Bind9AppTabComponent implements OnInit {
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
        daemonMap[appTab.app.details.daemon.name] = appTab.app.details.daemon
        const DMAP = [['named', 'named']]
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
