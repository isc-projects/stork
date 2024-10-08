import { Component, OnDestroy, OnInit } from '@angular/core'
import { App, Severity, VersionService } from '../version.service'
import { AppsVersions, Machine, ServicesService, VersionDetails } from '../backend'
import { deepCopy, getErrorMessage } from '../utils'
import { Observable, of, Subscription, tap } from 'rxjs'
import { catchError, concatMap, map } from 'rxjs/operators'
import { MessageService } from 'primeng/api'

/**
 * This component displays current known released versions of ISC Kea, Bind9 and Stork.
 * There is also a table with summary of ISC software versions detected by Stork
 * among authorized machines.
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
        'Security updates were found for ISC software used on those machines!',
        'Those machines use ISC software version that require your attention. Software updates are available.',
        'ISC software updates are available for those machines.',
        '',
        `Those machines use up-to-date ISC software`,
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
     * Keeps information whether Kea, Bind9, Stork current versions tables data is loading.
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
     */
    ngOnInit(): void {
        this.dataDate$ = this.versionService.getDataManufactureDate()
        this.isOfflineData$ = this.versionService.isOnlineData().pipe(map((b) => !b))
        this._subscriptions.add(
            this.versionService
                .getCurrentData()
                .pipe(
                    tap(() => {
                        this.swVersionsDataLoading = true
                    }),
                    concatMap((data) => {
                        if (data) {
                            this._processedData = data
                            this.keaVersions = deepCopy(data?.kea?.currentStable ?? [])
                            if (data?.kea?.latestDev) {
                                this.keaVersions.push(data.kea?.latestDev)
                            }

                            this.bind9Versions = deepCopy(data?.bind9?.currentStable ?? [])
                            if (data.bind9?.latestDev) {
                                this.bind9Versions.push(data?.bind9?.latestDev)
                            }

                            this.storkVersions = deepCopy(data.stork?.currentStable ?? [])
                            if (data.stork?.latestDev) {
                                this.storkVersions.push(data.stork?.latestDev)
                            }
                        }

                        this.swVersionsDataLoading = false
                        return this.servicesApi.getMachinesAppsVersions().pipe(
                            tap(() => {
                                this.summaryDataLoading = true
                                this.counters = [0, 0, 0, 0, 0]
                            }),
                            catchError((err) => {
                                let msg = getErrorMessage(err)
                                this.messageService.add({
                                    severity: 'error',
                                    summary: 'Error retrieving software versions data',
                                    detail: 'Error occurred when retrieving software versions data: ' + msg,
                                    life: 10000,
                                })
                                this.summaryDataLoading = false
                                return of({ items: [] })
                            })
                        )
                    }),
                    map((data) => {
                        data.items.map((m: Machine & { versionCheckSeverity: Severity }) => {
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
                            m.apps.forEach((a) => {
                                m.versionCheckSeverity = Math.min(
                                    this.severityMap[
                                        this.versionService.getSoftwareVersionFeedback(
                                            a.version,
                                            a.type as App,
                                            this._processedData
                                        )?.severity ?? Severity.success
                                    ],
                                    m.versionCheckSeverity
                                )
                            })

                            // TODO: daemons version match check - done on backend?
                            this.counters[m.versionCheckSeverity]++
                            return m
                        })
                        return data
                    })
                )
                .subscribe({
                    next: (data) => {
                        this.machines = data.items
                        this.summaryDataLoading = false
                    },
                    error: (err) => {
                        let msg = getErrorMessage(err)
                        this.messageService.add({
                            severity: 'error',
                            summary: 'Error retrieving software versions data',
                            detail: 'Error occurred when retrieving software versions data: ' + msg,
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
     * Triggers software versions data refresh - new data will be sent from the backend via version service.
     */
    refreshVersions() {
        this.versionService.refreshData()
    }

    /**
     * Gets a value displayed as groupHeader texts in
     * the "summary of ISC software versions detected by Stork" table.
     * @param severity severity for which the message is returned
     * @param dataDate data manufacture date
     */
    getGroupHeaderMessage(severity: Severity, dataDate: string) {
        if (severity === Severity.success) {
            return `Those machines use up-to-date ISC software (known as of ${dataDate})`
        }

        return this._groupHeaderMap[severity]
    }
}
