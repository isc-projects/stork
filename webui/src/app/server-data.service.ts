import { Injectable } from '@angular/core'
import { Observable, Subject, merge, timer } from 'rxjs'
import { filter, switchMap, shareReplay, takeUntil } from 'rxjs/operators'

import { ServicesService } from './backend/api/api'
import { AppsStats } from './backend/model/appsStats'

/**
 * Service for providing and caching data from the server.
 */
@Injectable({
    providedIn: 'root',
})
export class ServerDataService {
    private appsStats: Observable<AppsStats>
    private reload = new Subject<void>()
    private valid = false

    constructor(private servicesApi: ServicesService) {}

    /**
     * Get apps stats from the server and cache it for other subscribers.
     * Cache is refreshed after 30 minutes.
     */
    getAppsStats() {
        if (!this.appsStats) {
            const refreshInterval = 1000 * 60 * 30 // 30 mins
            const refreshTimer = timer(0, refreshInterval)

            // For each timer tick and and for each reload
            // make an http request to fetch new data
            this.appsStats = merge(refreshTimer, this.reload).pipe(
                switchMap((_) => this.servicesApi.getAppsStats()),
                shareReplay(1) // cache the response for all subscribers
            )
        }

        if (!this.valid) {
            this.forceReloadAppsStats()
            this.valid = true
        }

        return this.appsStats
    }

    /**
     * Force reloading cache for apps stats.
     */
    forceReloadAppsStats() {
        this.reload.next()
    }

    /**
     * Invalidates cached responses forcing to reload on next subscription
     */
    invalidate() {
        this.valid = false
    }
}
