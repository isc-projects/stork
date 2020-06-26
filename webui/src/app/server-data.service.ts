import { Injectable } from '@angular/core'
import { Observable, Subject, merge, timer, EMPTY } from 'rxjs'
import { switchMap, shareReplay, catchError, filter } from 'rxjs/operators'

import { AuthService } from './auth.service'
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

    constructor(private auth: AuthService, private servicesApi: ServicesService) {}

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
            this.appsStats = merge(refreshTimer, this.reload, this.auth.currentUser).pipe(
                filter((x) => x !== null), // filter out trigger which is logout ie user changed to null
                switchMap((_) =>
                    this.servicesApi.getAppsStats().pipe(
                        // use subpipe to not complete source due to error
                        catchError((err) => EMPTY) // in case of error drop the response at all (it should not be cached)
                    )
                ),
                shareReplay(1) // cache the response for all subscribers
            )
        }

        return this.appsStats
    }

    /**
     * Force reloading cache for apps stats.
     */
    forceReloadAppsStats() {
        this.reload.next()
    }
}
