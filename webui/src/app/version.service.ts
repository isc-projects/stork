import { Injectable } from '@angular/core'
import { minor, major, sort } from 'semver'
import { deepCopy } from './utils'

type SwRelease = 'latestSecure' | 'currentStable' | 'latestDev'

/**
 *
 */
export interface VersionDetails {
    version: string
    releaseDate: string
    eolDate?: string
    ESV?: string
    status?: string
    major?: number
    minor?: number
    range?: string
}

/**
 *
 */
export interface AppVersionMetadata {
    currentStable?: VersionDetails[]
    latestDev: VersionDetails
    latestSecure?: VersionDetails
}

/**
 *
 */
export type App = 'kea' | 'bind9' | 'stork'

@Injectable({
    providedIn: 'root',
})
export class VersionService {
    dataManufactureDate: string

    private _processedData: { [a in App | 'date']: AppVersionMetadata | string }

    private readonly _stableVersion: { [a in App]: string[] }

    // static for now; to be provided from server
    versionMetadata: { [a in App | 'date']: AppVersionMetadata | string } = {
        date: '2024-09-01',
        kea: {
            currentStable: [
                {
                    version: '2.6.1',
                    releaseDate: '2024-07-31',
                    eolDate: '2026-07-01',
                },
                {
                    version: '2.4.1',
                    releaseDate: '2023-11-29',
                    eolDate: '2025-07-01',
                },
            ],
            latestDev: {
                version: '2.7.2',
                releaseDate: '2024-08-28',
            },
        },
        stork: {
            latestDev: {
                version: '1.18.0',
                releaseDate: '2024-08-07',
            },
            latestSecure: {
                version: '1.15.1',
                releaseDate: '2024-03-27',
            },
        },
        bind9: {
            currentStable: [
                {
                    version: '9.18.29',
                    releaseDate: '2024-08-21',
                    eolDate: '2026-07-01',
                    ESV: 'true',
                },
                {
                    version: '9.20.1',
                    releaseDate: '2024-08-28',
                    eolDate: '2028-07-01',
                },
            ],
            latestDev: {
                version: '9.21.0',
                releaseDate: '2024-08-28',
            },
        },
    }

    private _onlineData: boolean

    constructor() {
        this._stableVersion = { kea: [], bind9: [], stork: [] }
        this.dataManufactureDate = '2024-09-01'
        this.processData()
        // For now force to false.
        this._onlineData = false
    }

    /**
     * Returns software version for given app and type.
     * @param app app for which the version lookup is done; accepted values: 'kea' | 'bind9' | 'stork'
     * @param swType sw version type for which the version lookup is done; accepted values: 'latestSecure' | 'currentStable' | 'latestDev'
     * @return version as either string (in case of latestSecure and latestDev) or array of strings (in case of currentStable)
     */
    getVersion(app: App, swType: SwRelease): string | string[] | null {
        return swType === 'currentStable'
            ? this._stableVersion[app] || null
            : this._processedData[app]?.[swType]?.version || null
    }

    /**
     * Returns software version details for given app and type.
     * @param app app for which the version lookup is done; accepted values: 'kea' | 'bind9' | 'stork'
     * @param swType sw version type for which the version lookup is done; accepted values: 'latestSecure' | 'currentStable' | 'latestDev'
     * @return version details as either single VersionDetails (in case of latestSecure and latestDev) or array of VersionDetails (in case of currentStable)
     */
    getVersionDetails(app: App, swType: SwRelease): VersionDetails | VersionDetails[] | null {
        return this._processedData[app]?.[swType] || null
    }

    /**
     * Returns sorted current stable semver versions as an array of strings for given app.
     * @param app either kea, bind9 or stork app
     */
    getStableVersions(app: App): string[] | null {
        return this._stableVersion[app] || null
    }

    /**
     * Returns the date (as string) when the versions data was manufactured.
     */
    getDataManufactureDate(): string {
        // This will have to be updated in case of "online concept"
        return this.versionMetadata.date as string
    }

    /**
     *
     */
    isOnlineData(): boolean {
        return this._onlineData
    }

    private processData() {
        let newData = deepCopy(this.versionMetadata)
        Object.keys(newData).forEach((app) => {
            if (newData[app] !== 'date') {
                Object.keys(newData[app]).forEach((swType) => {
                    switch (swType) {
                        case 'latestSecure':
                            newData[app][swType].status = 'Security release'
                            newData[app][swType].major = major(newData[app][swType].version)
                            newData[app][swType].minor = minor(newData[app][swType].version)
                            break
                        case 'latestDev':
                            newData[app][swType].status = 'Development'
                            newData[app][swType].major = major(newData[app][swType].version)
                            newData[app][swType].minor = minor(newData[app][swType].version)
                            break
                        case 'currentStable':
                            for (let e of newData[app][swType]) {
                                e.status = 'Current Stable'
                                e.major = major(e.version)
                                e.minor = minor(e.version)
                                e.range = `${e.major}.${e.minor}.x`
                            }

                            let versionsText = newData[app][swType].map((ver: VersionDetails) => ver.version)
                            this._stableVersion[app] = sort(versionsText)
                            break
                    }
                })
            }
        })
        this._processedData = newData
    }
}
