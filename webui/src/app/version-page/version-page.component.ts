import { Component, OnDestroy, OnInit } from '@angular/core'
import { AppType, Severity, UpdateNotification, VersionService } from '../version.service'
import { App as BackendApp, AppsVersions, Machine, ServicesService, VersionDetails } from '../backend'
import { deepCopy, getErrorMessage } from '../utils'
import { Observable, of, Subscription, tap } from 'rxjs'
import { catchError, concatMap, map } from 'rxjs/operators'
import { MessageService } from 'primeng/api'

/**
 * This component displays current known released versions of ISC Kea, BIND 9, and Stork.
 * There is also a table with summary of ISC software versions detected by Stork
 * among authorized machines.
 * For now, the source of all versions data used by this component is an offline JSON file,
 * which is valid for the day of Stork release. In the future, an online data source will be the
 * primary one, and the offline will be a fallback option.
 */
@Component({
    selector: 'app-version-page',
    templateUrl: './version-page.component.html',
    styleUrl: './version-page.component.sass',
})
export class VersionPageComponent implements OnInit, OnDestroy {
    /**
     * RxJS Subscription holding all subscriptions to Observables, so that they can be all unsubscribed
     * at once onDestroy.
     * @private
     */
    private _subscriptions = new Subscription()

    /**
     * An array mapping Severity enum to values displayed as groupHeaders texts in
     * the "summary of ISC software versions detected by Stork" table.
     */
    private _groupHeaderMap: string[] = [
        'Some issues were detected for the ISC software running on these machines (security updates available, versions mismatch)!',
        'These machines are running ISC software versions that require your attention. Software updates are available.',
        'ISC software updates are available for these machines.',
        '',
        'These machines are running up-to-date ISC software',
    ]

    /**
     * Configures the breadcrumbs for the component.
     */
    breadcrumbs = [{ label: 'Monitoring' }, { label: 'Software versions' }]

    /**
     * An array of version details of current Kea releases.
     */
    keaVersions: VersionDetails[] = []

    /**
     * An array of version details of current Bind9 releases.
     */
    bind9Versions: VersionDetails[] = []

    /**
     * An array of version details of current Stork releases.
     */
    storkVersions: VersionDetails[] = []

    /**
     * Severity enumeration field used by template.
     * @protected
     */
    protected readonly Severity = Severity

    /**
     * An array mapping Severity enum to values by which rows grouping is done in
     * "summary of ISC software versions detected by Stork" table.
     */
    severityMap: Severity[] = [
        Severity.error,
        Severity.warn,
        Severity.info,
        Severity.success, // SeverityEnum.secondary is mapped to SeverityEnum.success
        Severity.success,
    ]

    /**
     * An array with machine counters per Severity.
     */
    counters = [0, 0, 0, 0, 0]

    /**
     * Keeps information whether "summary of ISC software versions detected by Stork" table's data is loading.
     */
    summaryDataLoading: boolean

    /**
     * Keeps information whether Kea, BIND 9, Stork current versions tables data is loading.
     */
    swVersionsDataLoading: boolean

    /**
     * Keeps information whether there has been error when fetching data from backend.
     */
    errorOccurred = false

    /**
     * Keeps current software versions data received from version service.
     * @private
     */
    private _processedData: AppsVersions

    /**
     * An Observable of current manufacture date of the software versions data that was provided by the version service.
     */
    dataDate$: Observable<string>

    /**
     * An Observable of a boolean provided by the version service that is true if version data source is the offline json file.
     */
    isOfflineData$: Observable<boolean>

    /**
     * An Observable of a boolean provided by the version service that is true when software version alert is active.
     * The alert means that severity warning or higher was detected as part of the ISC software versions assessment.
     * It may be dismissed.
     */
    showAlert$: Observable<boolean>

    /**
     * An Observable of a notification about updates available for Stork server.
     */
    storkServerUpdateAvailable$: Observable<UpdateNotification>

    /**
     * An array of Machines in the "summary of ISC software versions detected by Stork" table.
     */
    machines: Machine[]

    /**
     * Class constructor.
     * @param versionService used to retrieve current software versions data
     * @param servicesApi used to retrieve authorized machines data
     * @param messageService used to display error messages
     */
    constructor(
        private versionService: VersionService,
        private servicesApi: ServicesService,
        private messageService: MessageService
    ) {}

    /**
     * Component lifecycle hook called to perform clean-up when destroying the component.
     */
    ngOnDestroy(): void {
        this._subscriptions.unsubscribe()
    }

    /**
     * Component lifecycle hook called upon initialization.
     * All the data that this component presents in tables, is fetched here from the VersionService.
     * The service gets the data from Stork server. For now, the data source is an offline JSON file,
     * which is valid for the day of Stork release. In the future, an online data source will be the
     * primary one, and the offline will be a fallback option.
     */
    ngOnInit(): void {
        this.dataDate$ = this.versionService.getDataManufactureDate()
        this.isOfflineData$ = this.versionService
            .getDataSource()
            .pipe(map((b) => b === AppsVersions.DataSourceEnum.Offline))
        this.showAlert$ = this.versionService.getVersionAlert().pipe(
            map((a) => {
                return a.detected
            })
        )
        this.storkServerUpdateAvailable$ = this.versionService.getStorkServerUpdateNotification()
        this._subscriptions.add(
            this.versionService
                .getCurrentData()
                .pipe(
                    tap(() => {
                        this.swVersionsDataLoading = true
                    }),
                    // We need to first wait for the data from the getCurrentData() observable.
                    // Whenever new data is emitted by the getCurrentData(), the summary table needs to be refreshed,
                    // and for that getMachinesAppsVersions() api must be called right after.
                    // To keep the order of source and inner observable, let's use concatMap operator.
                    concatMap((data) => {
                        if (data) {
                            this._processedData = data
                            this.keaVersions = deepCopy(data?.kea?.currentStable ?? [])
                            if (
                                data?.kea?.latestDev &&
                                this.versionService.isDevMoreRecentThanStable('kea', this._processedData)
                            ) {
                                this.keaVersions.push(data.kea?.latestDev)
                            }

                            this.bind9Versions = deepCopy(data?.bind9?.currentStable ?? [])
                            if (
                                data.bind9?.latestDev &&
                                data.bind9.latestDev &&
                                this.versionService.isDevMoreRecentThanStable('bind9', this._processedData)
                            ) {
                                this.bind9Versions.push(data?.bind9?.latestDev)
                            }

                            this.storkVersions = deepCopy(data.stork?.currentStable ?? [])
                            if (
                                data.stork?.latestDev &&
                                this.versionService.isDevMoreRecentThanStable('stork', this._processedData)
                            ) {
                                this.storkVersions.push(data.stork?.latestDev)
                            }
                        }

                        this.swVersionsDataLoading = false
                        return this.servicesApi.getMachinesAppsVersions().pipe(
                            tap(() => {
                                this.summaryDataLoading = true
                                this.counters = [0, 0, 0, 0, 0]
                            }),
                            // We don't want to complete the source getCurrentData() observable when error occurs for
                            // the inner getMachinesAppsVersions() observable, so let's catch the inner observable
                            // error here and gracefully return empty machines array.
                            catchError((err) => {
                                const msg = getErrorMessage(err)
                                this.messageService.add({
                                    severity: 'error',
                                    summary: 'Error retrieving software version data',
                                    detail: 'An error occurred when retrieving software version data: ' + msg,
                                    life: 10000,
                                })
                                this.summaryDataLoading = false
                                return of({ items: [] })
                            })
                        )
                    }),
                    map((data) => {
                        data.items?.map((m: Machine & { versionCheckSeverity: Severity }) => {
                            m.versionCheckSeverity = Severity.success
                            m.versionCheckSeverity = Math.min(
                                this.severityMap[
                                    this.versionService.getSoftwareVersionFeedback(
                                        m.agentVersion,
                                        'stork',
                                        this._processedData
                                    )?.severity ?? Severity.success
                                ],
                                m.versionCheckSeverity
                            )
                            m.apps.forEach((a: BackendApp & { mismatchingDaemons: boolean }) => {
                                m.versionCheckSeverity = Math.min(
                                    this.severityMap[
                                        this.versionService.getSoftwareVersionFeedback(
                                            a.version,
                                            a.type as AppType,
                                            this._processedData
                                        )?.severity ?? Severity.success
                                    ],
                                    m.versionCheckSeverity
                                )
                                // daemons version match check
                                if (this.versionService.areKeaDaemonsVersionsMismatching(a)) {
                                    m.versionCheckSeverity = Severity.error
                                    a.mismatchingDaemons = true
                                    this.versionService.detectAlertingSeverity(m.versionCheckSeverity)
                                }
                            })
                            this.counters[m.versionCheckSeverity]++
                            return m
                        })
                        return data
                    })
                )
                .subscribe({
                    next: (data) => {
                        this.machines = data.items ?? []
                        this.summaryDataLoading = false
                    },
                    error: (err) => {
                        const msg = getErrorMessage(err)
                        this.messageService.add({
                            severity: 'error',
                            summary: 'Error retrieving software version data',
                            detail: 'An error occurred when retrieving software version data: ' + msg,
                            life: 10000,
                        })
                        this.machines = []
                        this.swVersionsDataLoading = false
                        this.summaryDataLoading = false
                        this.errorOccurred = true
                    },
                })
        )
    }

    /**
     * Triggers software versions data refresh.
     *
     * New data will be sent from the backend via version service.
     */
    refreshVersions() {
        this.versionService.refreshData()
        this.swVersionsDataLoading = true
        this.summaryDataLoading = true
    }

    /**
     * Contacts with version service to dismiss the Version alert.
     */
    dismissAlert() {
        this.versionService.dismissVersionAlert()
    }

    /**
     * Gets a value displayed as groupHeader texts in
     * the "summary of ISC software versions detected by Stork" table.
     * @param severity severity for which the message is returned
     * @param dataDate data manufacture date
     */
    getGroupHeaderMessage(severity: Severity, dataDate: string) {
        if (severity === Severity.success) {
            return `These machines are running up-to-date ISC software (known as of ${dataDate})`
        }

        return this._groupHeaderMap[severity]
    }

    /**
     * Returns concatenated list of Kea daemons versions for given Kea app.
     * @param a Kea app
     */
    getDaemonsVersions(a: BackendApp): string {
        const daemons: string[] = []
        for (const d of a.details?.daemons ?? []) {
            if (d.name && d.version) {
                daemons.push(`${d.name} ${d.version}`)
            }
        }

        return daemons.join(', ')
    }
}
