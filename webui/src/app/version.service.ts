import { Injectable } from '@angular/core'
import { minor, coerce, valid, lt, satisfies, gt, minSatisfying } from 'semver'
import { App, AppsVersions, GeneralService } from './backend'
import { distinctUntilChanged, map, mergeMap, shareReplay } from 'rxjs/operators'
import { BehaviorSubject, Observable, tap } from 'rxjs'

/**
 * Interface defining fields for an object which is returned after
 * assessment of software version is done for particular App.
 */
export interface VersionFeedback {
    severity: Severity
    messages: string[]
    update?: string
}

/**
 * Interface defining software version alert.
 * Whether user should be notified about ('detected' flag),
 * and if so, what is the severity.
 */
export interface VersionAlert {
    detected: boolean
    severity: Severity
}

/**
 * Interface defining notification about software update.
 */
export interface UpdateNotification {
    available: boolean
    feedback: VersionFeedback
}

/**
 * Type for all possible ISC apps that have monitored software versions.
 */
export type AppType = 'kea' | 'bind9' | 'stork'

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
    private _dataOutdatedThreshold = 24 * 60 * 60 * 1000 // consider data out-of-date after 24 hours

    /**
     * Keeps track of Stork server version.
     * @private
     */
    private _storkServerVersion: string = undefined

    /**
     * RxJS Subject to emit next when a machine with severity warning or error was found.
     * @private
     */
    private _versionAlert$ = new BehaviorSubject<VersionAlert>({ detected: false, severity: Severity.success })

    /**
     * RxJS Subject that emits next value when a notification for the Stork server update changes.
     * @private
     */
    private _serverUpdateNotification$ = new BehaviorSubject<UpdateNotification>({
        available: false,
        feedback: { severity: Severity.success, messages: [] },
    })

    /**
     * An Observable which emits current software versions data retrieved from the backend.
     * It acts like a cache, because every observer that subscribes to it, receives replayed response
     * from the backend. This is to prevent backend overload with recurring queries.
     * New data from the backend may be fetched using _currentDataSubject$.next().
     */
    currentData$ = this._currentDataSubject$.pipe(
        mergeMap(() => {
            this.dataFetchedTimestamp = new Date()
            return this.generalService.getSoftwareVersions().pipe(tap((d) => this.checkStorkServerUpdates(d)))
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
     * @return AppsVersions RxJS Observable
     */
    getCurrentData(): Observable<AppsVersions> {
        if (this.isDataOutdated()) {
            this.refreshData()
        }

        return this.currentData$
    }

    /**
     * Forces retrieval of current software versions data from the backend.
     * Clears the _checkedVersionCache and disables previous _versionAlert$.
     */
    refreshData(): void {
        this._checkedVersionCache = new Map()
        this._versionAlert$.next({ detected: false, severity: Severity.success })
        this._currentDataSubject$.next({})
    }

    /**
     * Returns whether cached data retrieved from the backend is outdated.
     * This is used to regularly query the backend for current software versions data.
     * @return true if data is outdated; false otherwise
     */
    isDataOutdated(): boolean {
        return (
            this.dataFetchedTimestamp && Date.now() - this.dataFetchedTimestamp.getTime() > this._dataOutdatedThreshold
        )
    }

    /**
     * Returns an Observable of current manufacture date of the software versions data that was provided by the backend.
     * @return data manufacture date as string RxJS Observable
     */
    getDataManufactureDate(): Observable<string> {
        return this.currentData$.pipe(map((data) => data.date))
    }

    /**
     * Returns an Observable of the versions data source stating whether current data provided by the backend
     * origins from online sources (e.g. ISC GitLab REST api) or from offline data stored in versions.json file.
     * @return DataSourceEnum Observable
     */
    getDataSource(): Observable<AppsVersions.DataSourceEnum> {
        return this.currentData$.pipe(map((data) => data.dataSource))
    }

    /**
     * Makes an assessment whether provided app (Kea, Bind9 or Stork Agent) version is up-to-date
     * and returns the feedback information with the severity of the urge to update the software and
     * a message containing details of the assessment.
     * @param version string version that must contain a parsable semver
     * @param app either kea, bind9 or stork
     * @param data input data used to make the assessment
     * @return assessment result as a VersionFeedback object; it contains severity and messages to be displayed to the user
     * @throws Error when the assessment fails for any reason
     */
    getSoftwareVersionFeedback(version: string, app: AppType, data: AppsVersions): VersionFeedback {
        const cacheKey = version + app
        const cachedFeedback = this._checkedVersionCache?.get(cacheKey)
        if (cachedFeedback) {
            this.detectAlertingSeverity(cachedFeedback.severity)
            return cachedFeedback
        }

        let response: VersionFeedback = { severity: Severity.success, messages: [] }
        const sanitizedSemver = this.sanitizeSemver(version)
        let appName = ''
        if (sanitizedSemver) {
            appName = app === 'bind9' ? app.toUpperCase() : app[0].toUpperCase() + app.slice(1)
            appName += app === 'stork' ? ' agent' : ''
            const isDevelopmentVersion = this.isDevelopmentVersion(sanitizedSemver, app)

            // check security releases first
            const latestSecureVersionDetails = data?.[app]?.latestSecure || null
            if (
                latestSecureVersionDetails &&
                Array.isArray(latestSecureVersionDetails) &&
                latestSecureVersionDetails.length >= 1
            ) {
                let isSecure = false
                const secureDevVersions = []
                for (const details of latestSecureVersionDetails) {
                    if (this.isDevelopmentVersion(details.version, app)) {
                        secureDevVersions.push(details.version)
                    }

                    if (satisfies(sanitizedSemver, details.range)) {
                        if (lt(sanitizedSemver, details.version)) {
                            response = {
                                severity: Severity.error,
                                messages: [
                                    `Security update ${details.version} was released for ${appName}. Please update as soon as possible!`,
                                ],
                                update: details.version,
                            }

                            response = this.getStorkFeedback(app, sanitizedSemver, response)
                            return this.setCacheAndReturnResponse(cacheKey, response)
                        }

                        // matching range was found and detected semver >= security release, so break the for loop
                        isSecure = true
                        break
                    }
                }

                if (!isSecure && isDevelopmentVersion && secureDevVersions.length >= 1) {
                    const minDevSecure = minSatisfying(secureDevVersions, '*')
                    if (lt(sanitizedSemver, minDevSecure)) {
                        response = {
                            severity: Severity.error,
                            messages: [
                                `Security update ${minDevSecure} was released for ${appName}. Please update as soon as possible!`,
                            ],
                            update: minDevSecure,
                        }

                        response = this.getStorkFeedback(app, sanitizedSemver, response)
                        return this.setCacheAndReturnResponse(cacheKey, response)
                    }
                }
            }

            const currentStableVersionDetails = data?.[app]?.currentStable || null
            const currentStableMetadataAvailable =
                Array.isArray(currentStableVersionDetails) && currentStableVersionDetails.length > 0
            const sortedCurrentStableVersions = this.getVersion(app, 'currentStable', data)
            const sortedCurrentStablesAvailable =
                Array.isArray(sortedCurrentStableVersions) && sortedCurrentStableVersions.length > 0
            const dataDate = data?.date || 'unknown'

            // case - stable version
            if (!isDevelopmentVersion) {
                if (!currentStableVersionDetails) {
                    response = {
                        severity: Severity.secondary,
                        messages: [
                            `As of ${dataDate}, the ${appName} ${sanitizedSemver} stable version is not known yet.`,
                        ],
                    }

                    response = this.getStorkFeedback(app, sanitizedSemver, response)
                    return this.setCacheAndReturnResponse(cacheKey, response)
                }

                if (currentStableMetadataAvailable) {
                    for (const details of currentStableVersionDetails) {
                        if (satisfies(sanitizedSemver, details.range)) {
                            if (lt(sanitizedSemver, details.version)) {
                                response = {
                                    severity: Severity.info,
                                    messages: [
                                        `Stable ${appName} version update (${details.version}) is available (known as of ${dataDate}).`,
                                    ],
                                    update: details.version,
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
                            return this.setCacheAndReturnResponse(cacheKey, response)
                        }
                    }

                    // current version not matching currentStable ranges
                    if (sortedCurrentStablesAvailable) {
                        const versionsText = sortedCurrentStableVersions.join(', ')
                        if (lt(sanitizedSemver, sortedCurrentStableVersions[0])) {
                            // either semver major or minor are below min(current stable)
                            response = {
                                severity: Severity.warn,
                                messages: [
                                    `${appName} version ${sanitizedSemver} is older than current stable version/s ${versionsText}.`,
                                ],
                                update: sortedCurrentStableVersions[0],
                            }
                        } else {
                            // either semver major or minor are bigger than current stable
                            response = {
                                severity: Severity.secondary,
                                messages: [
                                    `${appName} version ${sanitizedSemver} is more recent than current stable version/s ${versionsText} (known as of ${dataDate}).`,
                                ],
                            }
                        }

                        response = this.getStorkFeedback(app, sanitizedSemver, response)
                        return this.setCacheAndReturnResponse(cacheKey, response)
                    }
                }

                // wrong json syntax - this shouldn't happen
                throw new Error(
                    'Invalid syntax of the software versions metadata JSON file received from Stork server.'
                )
            }

            // case - development version
            const latestDevVersion = this.getVersion(app, 'latestDev', data)
            if (isDevelopmentVersion) {
                if (latestDevVersion) {
                    if (lt(sanitizedSemver, latestDevVersion as string)) {
                        response = {
                            severity: Severity.warn,
                            messages: [
                                `Development ${appName} version update (${latestDevVersion}) is available (known as of ${dataDate}).`,
                            ],
                            update: latestDevVersion as string,
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

                    if (currentStableMetadataAvailable) {
                        response.messages.push(
                            'Please be advised that using development version in production is not recommended.'
                        )
                    }

                    response = this.getStorkFeedback(app, sanitizedSemver, response)
                    return this.setCacheAndReturnResponse(cacheKey, response)
                } else if (currentStableMetadataAvailable && sortedCurrentStablesAvailable) {
                    // There is no metadata for development release, but there is metadata for stable releases.
                    // This is very uncommon case, but it is possible.
                    response = {
                        severity: Severity.secondary,
                        messages: [
                            `${appName} has current stable version/s ${sortedCurrentStableVersions.join(', ')} available (known as of ${dataDate}).`,
                        ],
                    }

                    response.messages.push(
                        'Please be advised that using development version in production is not recommended.'
                    )

                    response = this.getStorkFeedback(app, sanitizedSemver, response)
                    return this.setCacheAndReturnResponse(cacheKey, response)
                }
            }

            throw new Error(`Couldn't asses the software version for ${appName} ${version}!`)
        }

        // fail case
        throw new Error(`Couldn't parse valid semver from given ${version} version!`)
    }

    /**
     * Returns true when the latest development release version is more recent than
     * the latest stable version or when there are no stable releases; false otherwise.
     * @param app either stork, kea or bind9 app
     * @param data versions data used to determine returned value
     */
    isDevMoreRecentThanStable(app: AppType, data: AppsVersions): boolean {
        const stables = this.getVersion(app, 'currentStable', data)
        if (!stables || stables?.length < 1) {
            return true
        }

        const devVersion = this.getVersion(app, 'latestDev', data) as string
        if (!devVersion) {
            return false
        }

        const lastStable = stables[stables.length - 1]
        return gt(devVersion, lastStable)
    }

    /**
     * Sanitizes given version string and returns valid semver if it could be parsed.
     * If valid semver couldn't be found, it returns null.
     * @param version version string to look for semver
     * @return sanitized semver or null in case semver was not parsed
     */
    sanitizeSemver(version: string): string | null {
        const sanitizedSemver = coerce(version)?.version
        if (sanitizedSemver && valid(sanitizedSemver)) {
            return sanitizedSemver
        }

        return null
    }

    /**
     * Setter of the _storkServerVersion that is tracked by this service.
     * @param version
     */
    setStorkServerVersion(version: string): void {
        this._storkServerVersion = version
    }

    /**
     * Returns an observable of VersionAlert.
     * The observable will emit next alert only if:
     * 'VersionAlert.detected' of the _versionAlert$ subject changes
     * or the _versionAlert$ subject reports higher severity than was reported before.
     * @return VersionAlert RxJS Observable
     */
    getVersionAlert(): Observable<VersionAlert> {
        return this._versionAlert$.pipe(
            distinctUntilChanged((prev, curr) => prev.detected === curr.detected && prev.severity <= curr.severity)
        )
    }

    /**
     * Returns an observable of UpdateNotification.
     * The observable will emit next notification when the notification
     * for Stork server update changes. The UpdateNotification contains
     * information:
     * available - whether there is an update available or not
     * feedback - VersionFeedback object holding update Severity, feedback messages, and update software version available
     */
    getStorkServerUpdateNotification(): Observable<UpdateNotification> {
        return this._serverUpdateNotification$
    }

    /**
     * Dismisses the _versionAlert$ by setting 'detected' flag to false and completing the RxJS subject.
     */
    dismissVersionAlert(): void {
        this._versionAlert$.next({ detected: false, severity: Severity.success })
        this._versionAlert$.complete()
    }

    /**
     * Checks whether all daemons for provided Kea app have the exact same version.
     * @param app Kea app to be checked
     * @return true if any daemon version mismatch is found; falsy (may also return undefined) otherwise
     * (in case all Kea daemons have the same version or when provided app wasn't the Kea app, or it couldn't be determined)
     */
    areKeaDaemonsVersionsMismatching(app: App): boolean {
        if (app?.type === 'kea') {
            const daemons = app.details?.daemons?.filter((daemon) => daemon.version)
            return daemons?.slice(1)?.some((daemon) => daemon.version !== daemons?.[0]?.version)
        }

        return false
    }

    /**
     * Returns true if provided app version is a development release.
     * For stable release, false is returned.
     * @param version app version
     * @param app either kea, bind9 or stork
     * @return true if provided app version is a development release; false otherwise
     * @private
     */
    private isDevelopmentVersion(version: string, app: AppType): boolean {
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
     * @return version as either string (in case of latestDev) or array of strings (in case of currentStable or latestSecure)
     * @private
     */
    private getVersion(app: AppType, swType: ReleaseType, data: AppsVersions): string | string[] | null {
        return swType === 'currentStable'
            ? data?.[app]?.sortedStableVersions || null
            : swType === 'latestDev'
              ? data?.[app]?.[swType]?.version || null
              : data?.[app]?.latestSecure?.map((details) => details.version) || null
    }

    /**
     * Checks if Stork Server and Stork Agent versions match.
     * In case of mismatch, given response is modified. Warning severity is set
     * and feedback message is added to existing messages.
     * @param app either Stork, Kea or Bind9 app
     * @param version software version to be checked
     * @param currentResponse current VersionFeedback response
     * @return Modified currentResponse in case of mismatch. In case mismatch was not found, currentResponse returned is not modified.
     * @private
     */
    private getStorkFeedback(app: AppType, version: string, currentResponse: VersionFeedback): VersionFeedback {
        if (app === 'stork' && this._storkServerVersion && this._storkServerVersion !== version) {
            const addMsg = `Stork server ${this._storkServerVersion} and Stork agent ${version} versions do not match! Please install matching versions!`
            return {
                severity: Severity.warn,
                messages: [...currentResponse.messages, addMsg],
                update: currentResponse.update ?? '',
            }
        }

        return currentResponse
    }

    /**
     * Checks given severity level and if it serious enough, it triggers the version alert.
     * @param severity current version severity
     */
    detectAlertingSeverity(severity: Severity): void {
        if (severity <= Severity.warn) {
            this._versionAlert$.next({ detected: true, severity: severity })
        }
    }

    /**
     * Helper function calling repeatable code:
     * 1. sets _checkedVersionCache for given cacheKey
     * 2. calls detectHigherSeverity(response) for given response
     * 3. returns the response
     * @param cacheKey _checkedVersionCache map key
     * @param response VersionFeedback response
     * @private
     */
    private setCacheAndReturnResponse(cacheKey: string, response: VersionFeedback): VersionFeedback {
        this._checkedVersionCache.set(cacheKey, response)
        this.detectAlertingSeverity(response.severity)
        return response
    }

    /**
     * Checks if there is update available for Stork server.
     * @param data AppsVersions data used to perform the check
     * @private
     */
    private checkStorkServerUpdates(data: AppsVersions): void {
        if (!this.sanitizeSemver(this._storkServerVersion)) {
            return
        }

        const serverFeedback = this.getSoftwareVersionFeedback(this._storkServerVersion, 'stork', data)
        const updateType = serverFeedback.severity === Severity.error ? 'security update' : 'update'
        this._serverUpdateNotification$.next({
            available: !!serverFeedback.update,
            feedback: {
                severity: serverFeedback.severity,
                update: serverFeedback.update || '',
                messages: [
                    !!serverFeedback.update
                        ? `Stork server ${updateType} is available (${serverFeedback.update}).`
                        : `Stork server is up-to-date.`,
                ],
            },
        })
        this.detectAlertingSeverity(serverFeedback?.severity ?? Severity.success)
    }
}
