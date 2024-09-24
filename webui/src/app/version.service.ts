import { Injectable } from '@angular/core'
import { minor, major, sort, coerce, valid, lt, satisfies, gt } from 'semver'
import { deepCopy } from './utils'
import { AppsVersions, GeneralService } from './backend'
import { filter, tap } from 'rxjs/operators'

/**
 * Interface defining fields for an object which describes either Stork, Kea or Bind9 software release.
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
 * Interface defining fields for an object which describes all possible types for either Stork, Kea or Bind9 software release.
 */
export interface AppVersionMetadata {
    currentStable?: VersionDetails[]
    latestDev: VersionDetails
    latestSecure?: VersionDetails
}

/**
 * Interface defining fields for an object which is returned after assessment of software version is done for particular App.
 */
export interface VersionFeedback {
    severity: Severity
    feedback: string
}

/**
 * Type for all possible ISC apps that have monitored software versions.
 */
export type App = 'kea' | 'bind9' | 'stork'

/**
 * Severity assigned after assessment of software version is done.
 */
export enum Severity {
    danger,
    warning,
    info,
    secondary,
    success,
}

/**
 * Type for different sorts of released software.
 */
type ReleaseType = 'latestSecure' | 'currentStable' | 'latestDev'

/**
 * Service providing current ISC Kea, Bind9 and Stork software versions.
 * Current data is fetched from Stork server.
 * The service also provides utilities to assess whether used ISC software is up to date.
 */
@Injectable({
    providedIn: 'root',
})
export class VersionService {
    dataManufactureDate: string

    private _checkedVersionCache: Map<string, VersionFeedback>

    private _processedData: { [a in App | 'date']: AppVersionMetadata | string }

    private _stableVersion: { [a in App]: string[] }

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

    asyncMetadata: AppsVersions | undefined

    dataFetchedTimestamp: Date | undefined

    fetchData() {
        this.generalService
            .getIscSwVersions()
            .pipe(
                tap(() => console.log('trying getIscSwVersions')),
                filter(() => !this.asyncMetadata || (this.dataFetchedTimestamp && this.isDataOld()))
            )
            .subscribe((data) => {
                console.log('went through filter and resp rxed')
                this.asyncMetadata = data
                this.dataFetchedTimestamp = new Date()
                this._stableVersion = { kea: [], bind9: [], stork: [] }
                this.dataManufactureDate = data['date']
                this._checkedVersionCache = new Map()
                this._onlineData = data['onlineData'] ?? false
                this.processData()
            })
    }

    isDataOld() {
        let now = new Date()
        return this.dataFetchedTimestamp && now.getTime() - this.dataFetchedTimestamp.getTime() < 10000
    }

    private _onlineData: boolean

    constructor(private generalService: GeneralService) {
        this.fetchData()
        // this._stableVersion = { kea: [], bind9: [], stork: [] }
        // this.dataManufactureDate = '2024-09-01'
        // this.processData()
        // this._checkedVersionCache = new Map()
        // // For now force to false.
        // this._onlineData = false
    }

    /**
     * Returns software version for given app and type.
     * @param app app for which the version lookup is done; accepted values: 'kea' | 'bind9' | 'stork'
     * @param swType sw version type for which the version lookup is done; accepted values: 'latestSecure' | 'currentStable' | 'latestDev'
     * @return version as either string (in case of latestSecure and latestDev) or array of strings (in case of currentStable)
     */
    private getVersion(app: App, swType: ReleaseType): string | string[] | null {
        return swType === 'currentStable'
            ? this._stableVersion?.[app] || null
            : this._processedData?.[app]?.[swType]?.version || null
    }

    /**
     * Returns software version details for given app and type.
     * @param app app for which the version lookup is done; accepted values: 'kea' | 'bind9' | 'stork'
     * @param swType sw version type for which the version lookup is done; accepted values: 'latestSecure' | 'currentStable' | 'latestDev'
     * @return version details as either single VersionDetails (in case of latestSecure and latestDev) or array of VersionDetails (in case of currentStable)
     */
    getVersionDetails(app: App, swType: ReleaseType): VersionDetails | VersionDetails[] | null {
        return this._processedData?.[app]?.[swType] || null
    }

    /**
     * Returns sorted current stable semver versions as an array of strings for given app.
     * @param app either kea, bind9 or stork app
     */
    private getStableVersions(app: App): string[] | null {
        return this._stableVersion?.[app] || null
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
        // let newData = deepCopy(this.versionMetadata)
        let newData = deepCopy(this.asyncMetadata)
        Object.keys(newData).forEach((key) => {
            if (key === 'kea' || key === 'bind9' || key === 'stork') {
                Object.keys(newData[key]).forEach((swType) => {
                    if (newData[key][swType]) {
                        switch (swType) {
                            case 'latestSecure':
                                newData[key][swType].status = 'Security release'
                                newData[key][swType].major = major(newData[key][swType].version)
                                newData[key][swType].minor = minor(newData[key][swType].version)
                                break
                            case 'latestDev':
                                newData[key][swType].status = 'Development'
                                newData[key][swType].major = major(newData[key][swType].version)
                                newData[key][swType].minor = minor(newData[key][swType].version)
                                break
                            case 'currentStable':
                                for (let e of newData[key][swType]) {
                                    e.status = 'Current Stable'
                                    e.major = major(e.version)
                                    e.minor = minor(e.version)
                                    e.range = `${e.major}.${e.minor}.x`
                                }

                                let versionsText = newData[key][swType].map((ver: VersionDetails) => ver.version)
                                this._stableVersion[key] = sort(versionsText)
                                break
                        }
                    }
                })
            }
        })
        this._processedData = newData
    }

    /**
     *
     * @param version
     * @param app
     */
    checkVersion(version: string, app: App): VersionFeedback | null {
        let cachedFeedback = this._checkedVersionCache?.get(version + app)
        if (cachedFeedback) {
            console.log('cache used')
            return cachedFeedback
        }

        let response: VersionFeedback = { severity: Severity.info, feedback: '' }
        let sanitizedSemver = coerce(version).version
        let appName = ''
        if (valid(sanitizedSemver)) {
            appName = app[0].toUpperCase() + app.slice(1)
            appName += app === 'stork' ? ' agent' : ''
            let isDevelopmentVersion = this.isDevelopmentVersion(sanitizedSemver, app)

            // check security releases first
            let latestSecureVersion = this.getVersion(app, 'latestSecure')
            if (latestSecureVersion && lt(sanitizedSemver, latestSecureVersion as string)) {
                response = {
                    // severity: 'error',
                    severity: Severity.danger,
                    feedback: `Security update ${latestSecureVersion} was released for ${appName}. Please update as soon as possible!`,
                }

                this._checkedVersionCache.set(version + app, response)
                return response
            }

            // case - stable version
            let currentStableVersionDetails = this.getVersionDetails(app, 'currentStable')
            let dataDate = this.getDataManufactureDate()
            if (isDevelopmentVersion === false && currentStableVersionDetails) {
                if (Array.isArray(currentStableVersionDetails) && currentStableVersionDetails.length >= 1) {
                    for (let details of currentStableVersionDetails) {
                        if (satisfies(sanitizedSemver, details.range)) {
                            if (lt(sanitizedSemver, details.version)) {
                                response = {
                                    // severity: 'warn',
                                    severity: Severity.info,
                                    feedback: `Stable ${appName} version update (${details.version}) is available (known as of ${dataDate}).`,
                                }
                            } else if (gt(sanitizedSemver, details.version)) {
                                response = {
                                    severity: Severity.secondary,
                                    feedback: `Current stable ${appName} version (known as of ${dataDate}) is ${details.version}. You are using more recent version ${sanitizedSemver}.`,
                                }
                            } else {
                                response = {
                                    severity: Severity.success,
                                    feedback: `${sanitizedSemver} is current ${appName} stable version (known as of ${dataDate}).`,
                                }
                            }

                            this._checkedVersionCache.set(version + app, response)
                            return response
                        }
                    }

                    // current version not matching currentStable ranges
                    let stableVersions = this.getStableVersions(app)
                    if (Array.isArray(stableVersions) && stableVersions.length > 0) {
                        let versionsText = stableVersions.join(', ')
                        if (lt(sanitizedSemver, stableVersions[0])) {
                            // either semver major or minor are below min(current stable)
                            response = {
                                severity: Severity.warning, // TODO: or info ?
                                // feedback: `${appName} version ${sanitizedSemver} is older than current stable version/s ${versionsText}. Updating to current stable is possible.`,
                                feedback: `${appName} version ${sanitizedSemver} is older than current stable version/s ${versionsText}.`,
                            }
                        } else {
                            // either semver major or minor are bigger than current stable
                            response = {
                                severity: Severity.secondary,
                                feedback: `${appName} version ${sanitizedSemver} is more recent than current stable version/s ${versionsText} (known as of ${dataDate}).`,
                            }
                            // this.feedback = `Current stable ${this.appName} version as of ${this.extendedMetadata.date} is/are ${versionsText}. You are using more recent version ${sanitizedSemver}.`
                        }

                        this._checkedVersionCache.set(version + app, response)
                        return response
                    }
                }

                // wrong json syntax - this shouldn't happen
                return null
            }

            // case - development version
            let latestDevVersion = this.getVersion(app, 'latestDev')
            if (isDevelopmentVersion === true && latestDevVersion) {
                if (lt(sanitizedSemver, latestDevVersion as string)) {
                    response = {
                        // severity: 'warn',
                        severity: Severity.warning,
                        feedback: `Development ${appName} version update (${latestDevVersion}) is available (known as of ${dataDate}).`,
                        // feedback: `You are using ${appName} development version ${sanitizedSemver}. Current development version (known as of ${dataDate}) is ${latestDevVersion}. Please consider updating.`,
                    }
                } else if (gt(sanitizedSemver, latestDevVersion as string)) {
                    response = {
                        severity: Severity.secondary,
                        feedback: `Current development ${appName} version (known as of ${dataDate}) is ${latestDevVersion}. You are using more recent version ${sanitizedSemver}.`,
                    }
                } else {
                    response = {
                        severity: Severity.success,
                        feedback: `${sanitizedSemver} is current ${appName} development version (known as of ${dataDate}).`,
                    }
                }

                if (currentStableVersionDetails) {
                    let extFeedback = [
                        response.feedback,
                        // `Please be advised that using development version in production is not recommended! Consider using ${appName} stable release.`,
                        `Please be advised that using development version in production is not recommended.`,
                    ].join(' ')
                    response = {
                        // severity: 'warn',
                        severity: Severity.warning,
                        feedback: extFeedback,
                    }
                }

                this._checkedVersionCache.set(version + app, response)
                return response
            }
        }

        // fail case
        return null
    }

    /**
     *
     * @param version
     * @param app
     * @private
     */
    private isDevelopmentVersion(version: string, app: App) {
        if (app === 'kea' || app === 'bind9') {
            const minorVersion = minor(version)
            return minorVersion % 2 === 1
        }
        // Stork versions are all dev for now. To be updated with Stork 2.0.0.
        return true
    }
}
