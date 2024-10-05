import { Injectable } from '@angular/core'
import { minor, coerce, valid, lt, satisfies, gt } from 'semver'
import { AppsVersions, GeneralService } from './backend'
import { delay, distinctUntilChanged, map, mergeMap, shareReplay } from 'rxjs/operators'
import { BehaviorSubject, Observable } from 'rxjs'

/**
 * Interface defining fields for an object which is returned after assessment of software version is done for particular App.
 */
export interface VersionFeedback {
    severity: Severity
    messages: string[]
}

/**
 * Type for all possible ISC apps that have monitored software versions.
 */
export type App = 'kea' | 'bind9' | 'stork'

/**
 * Severity assigned after assessment of software version is done.
 */
export enum Severity {
    error,
    warn,
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
    /**
     * A map for caching returning feedback for queried app and version.
     * The key of the map is the concatenated version and app, e.g. "2.6.1kea" or "1.18.0stork".
     * @private
     */
    private _checkedVersionCache: Map<string, VersionFeedback>

    /**
     * RxJS BehaviorSubject used to trigger current software versions data refresh from the backend.
     * @private
     */
    private _currentDataSubject$ = new BehaviorSubject(undefined)

    /**
     * Stores information how many milliseconds after the data was last fetched from the backend,
     * the data is still considered up-to-date.
     * @private
     */
    private _dataOutdatedThreshold = 24 * 60 * 60 * 1000

    /**
     * Keeps track of Stork server version.
     * @private
     */
    private _storkServerVersion: string = undefined

    /**
     * RxJS Subject to emit next when a machine with severity warning or error was found.
     * @private
     */
    private _warningFound$ = new BehaviorSubject<[boolean, Severity]>([false, Severity.success])

    /**
     * An Observable which emits current software versions data retrieved from the backend.
     * It acts like a cache, because every observer that subscribes to it, receives replayed response
     * from the backend. This is to prevent backend overload with recurring queries.
     * New data from the backend may be fetched using _currentDataSubject$.next().
     */
    currentData$ = this._currentDataSubject$.pipe(
        mergeMap(() => {
            this.dataFetchedTimestamp = new Date()
            console.log('fetching new data with generalService.getIscSwVersions()', this.dataFetchedTimestamp)
            return this.generalService.getIscSwVersions()
        }),
        shareReplay(1)
    )

    /**
     * Stores timestamp when the current software versions data was last fetched.
     */
    dataFetchedTimestamp: Date | undefined

    /**
     * Service constructor.
     * @param generalService service used to query the backend for current software versions data
     */
    constructor(private generalService: GeneralService) {
        this._checkedVersionCache = new Map()
    }

    /**
     * Returns current software versions data Observable.
     */
    getCurrentData(): Observable<AppsVersions> {
        return this.currentData$
    }

    /**
     * Forces retrieval of current software versions data from the backend.
     * Clears the _checkedVersionCache.
     */
    refreshData() {
        // todo: convert to observable to catch errors
        this._checkedVersionCache = new Map()
        this._warningFound$.next([false, Severity.success])
        this._currentDataSubject$.next({})
    }

    /**
     * Returns whether cached data retrieved from the backend is outdated.
     * This is used to regularly query the backend for current software versions data.
     */
    isDataOutdated() {
        return (
            this.dataFetchedTimestamp && Date.now() - this.dataFetchedTimestamp.getTime() < this._dataOutdatedThreshold
        )
    }

    /**
     * Returns an Observable of current manufacture date of the software versions data that was provided by the backend.
     */
    getDataManufactureDate(): Observable<string> {
        return this.currentData$.pipe(map((data) => data.date))
    }

    /**
     * Returns an Observable of the boolean stating whether current software versions data provided by the backend
     * origins from online sources (e.g. ISC GitLab REST api) or from offline data stored in versions.json file.
     */
    isOnlineData(): Observable<boolean> {
        return this.currentData$.pipe(map((data) => !!data.onlineData))
    }

    /**
     * Makes an assessment whether provided app (Kea, Bind9 or Stork Agent) version is up-to-date
     * and returns the feedback information with the severity of the urge to update the software and
     * a message containing details of the assessment.
     * @param version string version that must contain a parsable semver
     * @param app either kea, bind9 or stork
     * @param data input data used to make the assessment
     */
    getSoftwareVersionFeedback(version: string, app: App, data: AppsVersions): VersionFeedback {
        let cacheKey = version + app
        let cachedFeedback = this._checkedVersionCache?.get(cacheKey)
        if (cachedFeedback) {
            console.log('cache used')
            this.checkSeverity(cachedFeedback)
            return cachedFeedback
        }

        console.log('getSoftwareVersionFeedback no cache found')

        let response: VersionFeedback = { severity: Severity.success, messages: [] }
        let sanitizedSemver = this.sanitizeSemver(version)
        let appName = ''
        if (sanitizedSemver) {
            appName = app[0].toUpperCase() + app.slice(1)
            appName += app === 'stork' ? ' agent' : ''
            let isDevelopmentVersion = this.isDevelopmentVersion(sanitizedSemver, app)

            // check security releases first
            let latestSecureVersion = this.getVersion(app, 'latestSecure', data)
            if (latestSecureVersion && lt(sanitizedSemver, latestSecureVersion as string)) {
                response = {
                    severity: Severity.error,
                    messages: [
                        `Security update ${latestSecureVersion} was released for ${appName}. Please update as soon as possible!`,
                    ],
                }

                this._checkedVersionCache.set(cacheKey, response)
                this.checkSeverity(response)
                return response
            }

            // case - stable version
            let currentStableVersionDetails = data?.[app]?.currentStable || null
            let dataDate = data?.date || 'unknown'
            if (isDevelopmentVersion === false) {
                if (!currentStableVersionDetails) {
                    response = {
                        severity: Severity.secondary,
                        messages: [`${appName} ${sanitizedSemver} stable version is not known yet as of ${dataDate}.`],
                    }

                    response = this.getStorkFeedback(app, sanitizedSemver, response)
                    this._checkedVersionCache.set(cacheKey, response)
                    this.checkSeverity(response)
                    return response
                }

                if (Array.isArray(currentStableVersionDetails) && currentStableVersionDetails.length >= 1) {
                    for (let details of currentStableVersionDetails) {
                        if (satisfies(sanitizedSemver, details.range)) {
                            if (lt(sanitizedSemver, details.version)) {
                                response = {
                                    severity: Severity.info,
                                    messages: [
                                        `Stable ${appName} version update (${details.version}) is available (known as of ${dataDate}).`,
                                    ],
                                }
                            } else if (gt(sanitizedSemver, details.version)) {
                                response = {
                                    severity: Severity.secondary,
                                    messages: [
                                        `Current stable ${appName} version (known as of ${dataDate}) is ${details.version}. You are using more recent version ${sanitizedSemver}.`,
                                    ],
                                }
                            } else {
                                response = {
                                    severity: Severity.success,
                                    messages: [
                                        `${sanitizedSemver} is current ${appName} stable version (known as of ${dataDate}).`,
                                    ],
                                }
                            }

                            response = this.getStorkFeedback(app, sanitizedSemver, response)

                            this._checkedVersionCache.set(cacheKey, response)
                            this.checkSeverity(response)
                            return response
                        }
                    }

                    // current version not matching currentStable ranges
                    let stableVersions = data?.[app].sortedStables || null
                    if (Array.isArray(stableVersions) && stableVersions.length > 0) {
                        let versionsText = stableVersions.join(', ')
                        if (lt(sanitizedSemver, stableVersions[0])) {
                            // either semver major or minor are below min(current stable)
                            response = {
                                severity: Severity.warn, // TODO: or info ?
                                // feedback: `${appName} version ${sanitizedSemver} is older than current stable version/s ${versionsText}. Updating to current stable is possible.`,
                                messages: [
                                    `${appName} version ${sanitizedSemver} is older than current stable version/s ${versionsText}.`,
                                ],
                            }
                        } else {
                            // either semver major or minor are bigger than current stable
                            response = {
                                severity: Severity.secondary,
                                messages: [
                                    `${appName} version ${sanitizedSemver} is more recent than current stable version/s ${versionsText} (known as of ${dataDate}).`,
                                ],
                            }
                            // this.feedback = `Current stable ${this.appName} version as of ${this.extendedMetadata.date} is/are ${versionsText}. You are using more recent version ${sanitizedSemver}.`
                        }

                        response = this.getStorkFeedback(app, sanitizedSemver, response)

                        this._checkedVersionCache.set(cacheKey, response)
                        this.checkSeverity(response)
                        return response
                    }
                }

                // wrong json syntax - this shouldn't happen
                throw new Error(
                    'Invalid syntax of the software versions metadata JSON file received from Stork server.'
                )
            }

            // case - development version
            let latestDevVersion = this.getVersion(app, 'latestDev', data)
            if (isDevelopmentVersion === true && latestDevVersion) {
                if (lt(sanitizedSemver, latestDevVersion as string)) {
                    response = {
                        // severity: 'warn',
                        severity: Severity.warn,
                        messages: [
                            `Development ${appName} version update (${latestDevVersion}) is available (known as of ${dataDate}).`,
                        ],
                        // feedback: `You are using ${appName} development version ${sanitizedSemver}. Current development version (known as of ${dataDate}) is ${latestDevVersion}. Please consider updating.`,
                    }
                } else if (gt(sanitizedSemver, latestDevVersion as string)) {
                    response = {
                        severity: Severity.secondary,
                        messages: [
                            `Current development ${appName} version (known as of ${dataDate}) is ${latestDevVersion}. You are using more recent version ${sanitizedSemver}.`,
                        ],
                    }
                } else {
                    response = {
                        severity: Severity.success,
                        messages: [
                            `${sanitizedSemver} is current ${appName} development version (known as of ${dataDate}).`,
                        ],
                    }
                }

                if (currentStableVersionDetails) {
                    response.messages.push(
                        'Please be advised that using development version in production is not recommended.'
                    )
                    response.severity = Severity.warn
                }

                response = this.getStorkFeedback(app, sanitizedSemver, response)

                this._checkedVersionCache.set(cacheKey, response)
                this.checkSeverity(response)
                return response
            }

            throw new Error(`Couldn't asses the software version for ${appName} ${version}!`)
        }

        // fail case
        throw new Error(`Couldn't parse valid semver from given ${version} version!`)
    }

    /**
     * Sanitizes given version string and returns valid semver if it could be parsed.
     * If valid semver couldn't be found, it returns null.
     * @param version version string to look for semver
     */
    sanitizeSemver(version: string): string | null {
        let sanitizedSemver = coerce(version)?.version
        if (sanitizedSemver && valid(sanitizedSemver)) {
            return sanitizedSemver
        }

        return null
    }

    /**
     * Setter of the _storkServerVersion that is tracked by this service.
     * @param version
     */
    setStorkServerVersion(version: string) {
        this._storkServerVersion = version
    }

    /**
     *
     */
    getWarningFound(): Observable<[boolean, Severity]> {
        return this._warningFound$.pipe(
            distinctUntilChanged((prev, curr) => prev[0] === curr[0] && prev[1] <= curr[1]),
            delay(1000)
        )
    }

    /**
     * Returns true if provided app version is a development release.
     * For stable release, false is returned.
     * @param version app version
     * @param app either kea, bind9 or stork
     * @private
     */
    private isDevelopmentVersion(version: string, app: App) {
        // Stork versions are all dev until 2.0.0.
        if (app === 'stork' && lt(version, '2.0.0')) {
            return true
        }
        const minorVersion = minor(version)
        return minorVersion % 2 === 1
    }

    /**
     * Returns software version for given app and type.
     * @param app app for which the version lookup is done; accepted values: 'kea' | 'bind9' | 'stork'
     * @param swType sw version type for which the version lookup is done; accepted values: 'latestSecure' | 'currentStable' | 'latestDev'
     * @param data
     * @return version as either string (in case of latestSecure and latestDev) or array of strings (in case of currentStable)
     */
    private getVersion(app: App, swType: ReleaseType, data: AppsVersions): string | string[] | null {
        return swType === 'currentStable' ? data?.[app]?.sortedStables || null : data?.[app]?.[swType]?.version || null
    }

    /**
     * Checks if Stork Server and Stork Agent versions match.
     * In case of mismatch, given response is modified. Warning severity is set
     * and feedback message is added to existing messages.
     * @param app either Stork, Kea or Bind9 app
     * @param version software version to be checked
     * @param currentResponse current VersionFeedback response
     * @return Modified currentResponse in case of mismatch. In case mismatch was not found, currentResponse returned is not modified.
     */
    private getStorkFeedback(app: App, version: string, currentResponse: VersionFeedback): VersionFeedback {
        if (app === 'stork' && this._storkServerVersion && this._storkServerVersion !== version) {
            let addMsg = `Stork server ${this._storkServerVersion} and Stork agent ${version} versions do not match! Please install matching versions!`
            return {
                severity: Severity.warn,
                messages: [...currentResponse.messages, addMsg],
            }
        }

        return currentResponse
    }

    private checkSeverity(currentResponse: VersionFeedback): void {
        if (currentResponse.severity <= Severity.warn) {
            this._warningFound$.next([true, currentResponse.severity])
        }
    }
}
