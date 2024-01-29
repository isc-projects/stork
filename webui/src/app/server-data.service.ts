import { HttpErrorResponse } from '@angular/common/http'
import { Injectable } from '@angular/core'
import { Observable, Subject, merge, timer, EMPTY, of } from 'rxjs'
import { switchMap, shareReplay, catchError, filter, map } from 'rxjs/operators'

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
    private reloadDaemonConfiguration: { [daemonId: number]: Subject<number> } = {}

    private _machinesAddresses: Observable<any>
    private _appsNames: Observable<any>
    private _daemonConfigurations: { [daemonId: number]: Observable<any> } = {}

    constructor(
        private auth: AuthService,
        public servicesApi: ServicesService,
        private usersApi: UsersService
    ) {}

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
                switchMap(() => {
                    return this.servicesApi.getAppsStats().pipe(
                        // use subpipe to not complete source due to error
                        catchError(() => EMPTY) // in case of error drop the response (it should not be cached)
                    )
                }),
                shareReplay(1) // cache the response for all subscribers
            )
        }

        return this.appsStats
    }

    /**
     * Force reloading cache for apps stats.
     */
    forceReloadAppsStats() {
        this.reloadAppStats.next()
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
                switchMap(() =>
                    this.usersApi.getGroups().pipe(
                        // use subpipe to not complete source due to error
                        catchError(() => EMPTY) // in case of error drop the response (it should not be cached)
                    )
                ),
                shareReplay(1) // cache the response for all subscribers
            )
        }

        return this.groups
    }

    /**
     * Get name of the system group fetched from the database indicated by group ID.
     *
     * @param groupId Identifier of the group in the database, counted
     *                from 1.
     * @param groupItems List of all groups returned by the server.
     * @returns Group name or unknown string if the group is not found.
     */
    public getGroupName(groupId: number, groupItems: any[]): string {
        // The superadmin group is well known and doesn't require
        // iterating over the list of groups fetched from the server.
        // Especially, if the server didn't respond properly for
        // some reason, we still want to be able to handle the
        // superadmin group.
        if (groupId === 1) {
            return 'superadmin'
        }
        for (const grp of groupItems) {
            if (grp.id === groupId) {
                return grp.name
            }
        }
        return 'unknown'
    }

    /**
     * Returns a set of machines' addresses.
     *
     * This function fetches a list of all machines' ids and addresses and
     * transforms returned data to a set of machines' addresses.
     *
     * @returns Observable holding a list of machines' addresses.
     */
    public getMachinesAddresses(): Observable<Set<string>> {
        this._machinesAddresses = this.servicesApi.getMachinesDirectory().pipe(
            map((data) => {
                const addresses = new Set<string>()
                for (const m of data.items) {
                    addresses.add(m.address)
                }
                return addresses
            })
        )
        return this._machinesAddresses
    }

    /**
     * Returns a set of apps' names.
     *
     * This function fetches a list of all apps' ids and names and
     * transforms returned data to a map with an app name as a key
     * and id as a value.
     *
     * @returns Observable holding a list of apps' names.
     */
    public getAppsNames(): Observable<Map<string, number>> {
        this._appsNames = this.servicesApi.getAppsDirectory().pipe(
            map((data) => {
                const names = new Map<string, number>()
                for (const a of data.items) {
                    names.set(a.name, a.id)
                }
                return names
            })
        )
        return this._appsNames
    }

    /**
     * Get (Kea) daemon configuration from the server and cache it for other subscribers.
     * Cache is refreshed manually or when the user is logged in.
     * @param daemonId Daemon ID
     * @returns Observable of daemon configuration
     */
    public getDaemonConfiguration(daemonId: number): Observable<any | HttpErrorResponse> {
        if (!(daemonId in this._daemonConfigurations)) {
            this.reloadDaemonConfiguration[daemonId] = new Subject<number>()
            this._daemonConfigurations[daemonId] = merge(
                this.reloadDaemonConfiguration[daemonId],
                this.auth.currentUser
            ).pipe(
                filter((x) => x !== null), // filter out trigger which is logout ie user changed to null
                switchMap(() => {
                    return this.servicesApi.getDaemonConfig(daemonId).pipe(
                        // use subpipe to not complete source due to error
                        catchError((err) => of(err)) // in case of error continue with it to prevent broken pipe
                    )
                }),
                shareReplay(1) // cache the response for all subscribers
            )
        }
        return this._daemonConfigurations[daemonId]
    }

    /**
     * Force reloading cache for daemon configuration.
     * @param daemonId Daemon ID
     */
    forceReloadDaemonConfiguration(daemonId: number) {
        if (daemonId in this.reloadDaemonConfiguration) {
            this.reloadDaemonConfiguration[daemonId].next(daemonId)
        }
    }
}
