import { Component, OnInit, Input, Output, EventEmitter } from '@angular/core'

import moment from 'moment-timezone'

import { MessageService, MenuItem } from 'primeng/api'

import { durationToString } from '../utils'

@Component({
    selector: 'app-bind9-app-tab',
    templateUrl: './bind9-app-tab.component.html',
    styleUrls: ['./bind9-app-tab.component.sass'],
})
export class Bind9AppTabComponent implements OnInit {
    private _appTab: any
    @Output() refreshApp = new EventEmitter<number>()

    daemons: any[] = []

    constructor() {}

    ngOnInit() {}

    @Input()
    set appTab(appTab) {
        this._appTab = appTab

        const daemonMap = []
        daemonMap[appTab.app.details.daemon.name] = appTab.app.details.daemon
        const DMAP = [['named', 'named']]
        const daemons = []
        for (const dm of DMAP) {
            if (daemonMap[dm[0]] !== undefined) {
                daemonMap[dm[0]].niceName = dm[1]
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

    /**
     * Get cache utilization based on stats.
     * A percentage is returned as floored int.
     */
    getCacheUtilization(daemon) {
        let utilization = 0.0
        if (!daemon.cacheHitRatio) {
            return utilization
        }
        utilization = 100 * daemon.cacheHitRatio
        return Math.floor(utilization)
    }
}
