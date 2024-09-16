import { Injectable } from '@angular/core'
import { App, AppVersionMetadata, VersionDetails } from './version-status/version-status.component'
import { minor, major, sort } from 'semver'
import { deepCopy } from './utils'

type SwRelease = 'latestSecure' | 'currentStable' | 'latestDev'

@Injectable({
    providedIn: 'root',
})
export class VersionService {
    dataManufactureDate: string

    private _processedData: {}

    private _stableVersion: {}

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

    constructor() {
        this.dataManufactureDate = '2024-09-01'
        this.processData()
    }

    /**
     * Returns software version for given app and type.
     * @param app app for which the version lookup is done; accepted values: 'kea' | 'bind9' | 'stork'
     * @param swType sw version type for which the version lookup is done; accepted values: 'latestSecure' | 'currentStable' | 'latestDev'
     */
    getVersion(app: App, swType: SwRelease): string | string[] | null {
        switch (swType) {
            case 'latestSecure':
            case 'latestDev':
                return this.versionMetadata[app]?.[swType]?.version || null
            case 'currentStable':
                return (
                    this.versionMetadata[app]?.[swType]?.map((swDetails: VersionDetails) => swDetails.version) || null
                )
        }
    }

    /**
     * Returns the date (as string) when the versions data was manufactured.
     */
    getDataManufactureDate(): string {
        // This will have to be updated in case of "online concept"
        return this.versionMetadata.date as string
    }

    private processData() {
        let newData = deepCopy(this.versionMetadata)
        Object.keys(newData).forEach((k) => {
            if (newData[k] !== 'date') {
                Object.keys(newData[k]).forEach((innerK) => {
                    switch (innerK) {
                        case 'latestSecure':
                            newData[k][innerK].status = 'Security release'
                            newData[k][innerK].major = major(newData[k][innerK].version)
                            newData[k][innerK].minor = minor(newData[k][innerK].version)
                            break
                        case 'latestDev':
                            newData[k][innerK].status = 'Security release'
                            newData[k][innerK].major = major(newData[k][innerK].version)
                            newData[k][innerK].minor = minor(newData[k][innerK].version)
                            break
                        case 'currentStable':
                            for (let e of newData[k][innerK]) {
                                e.status = 'Current Stable'
                                e.major = major(e.version)
                                e.minor = minor(e.version)
                            }
                            let versionsText = newData[k][innerK].map((ver: VersionDetails) => ver.version)
                            this._stableVersion[k] = sort(versionsText)
                            break
                    }
                })
            }
        })
        this._processedData = newData
    }
}
