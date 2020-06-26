import { Injectable } from '@angular/core'
import { Observable, Subject, merge, timer, EMPTY } from 'rxjs'
import { switchMap, shareReplay, catchError, filter } from 'rxjs/operators'

import { AuthService } from './auth.service'
import { ServicesService, UsersService } from './backend/api/api'
import { AppsStats } from './backend/model/appsStats'
import { Groups } from './backend/model/groups'

/**
 * Service for providing and caching data from the server.
 */
@Injectable({
    providedIn: 'root',
})
export class ServerDataService {
    private appsStats: Observable<AppsStats>
    private groups: Observable<Groups>
    private reloadAppStats = new Subject<void>()

    constructor(private auth: AuthService, private servicesApi: ServicesService, private usersApi: UsersService) {}

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
            this.appsStats = merge(refreshTimer, this.reloadAppStats, this.auth.currentUser).pipe(
                filter((x) => x !== null), // filter out trigger which is logout ie user changed to null
                switchMap((_) =>
                    this.servicesApi.getAppsStats().pipe(
                        // use subpipe to not complete source due to error
                        catchError((err) => EMPTY) // in case of error drop the response (it should not be cached)
                    )
                ),
                shareReplay(1) // cache the response for all subscribers
            )
        }

        return this.appsStats
    }

    /**
     * Get system groups from the server and cache it for other subscribers.
     *
     * Cache is refreshed upon user login.
     */
    getGroups() {
        if (!this.groups) {
            this.groups = this.auth.currentUser.pipe(
                filter((x) => x !== null), // filter out trigger which is logout ie user changed to null
                switchMap((_) =>
                    this.usersApi.getGroups().pipe(
                        // use subpipe to not complete source due to error
                        catchError((err) => EMPTY) // in case of error drop the response (it should not be cached)
                    )
                ),
                shareReplay(1) // cache the response for all subscribers
            )
        }

        return this.groups
    }

    /**
     * Returns name of the system group fetched from the database.
     *
     * @param groupId Identifier of the group in the database, counted
     *                from 1.
     * @param groupItems List of all groups returned by the server.
     * @returns Group name or unknown string if the group is not found.
     */
    public groupName(groupId: number, groupItems: any[]): string {
        // The superadmin group is well known and doesn't require
        // iterating over the list of groups fetched from the server.
        // Especially, if the server didn't respond properly for
        // some reason, we still want to be able to handle the
        // superadmin group.
        if (groupId === 1) {
            return 'superadmin'
        }
        const groupIdx = groupId
        if (groupItems.hasOwnProperty(groupIdx)) {
            return this.groups[groupIdx].name
        }
        return 'unknown'
    }

    /**
     * Force reloading cache for apps stats.
     */
    forceReloadAppsStats() {
        this.reloadAppStats.next()
    }
}
