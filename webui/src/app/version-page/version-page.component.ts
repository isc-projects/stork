import { Component, OnDestroy, OnInit } from '@angular/core'
import { App, Severity, VersionService } from '../version.service'
import { AppsVersions, Machine, ServicesService, VersionDetails } from '../backend'
import { deepCopy, getErrorMessage } from '../utils'
import { of, Subscription, tap } from 'rxjs'
import { catchError, concatMap } from 'rxjs/operators'
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
     * Configures the breadcrumbs for the component.
     */
    breadcrumbs = [{ label: 'Monitoring' }, { label: 'Software versions' }]

    /**
     * Keeps true if version data source is the offline json file.
     */
    isDataOffline: boolean

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
     * Current manufacture date of the software versions data that was provided by the version service.
     */
    dataDate: string = 'unknown'

    /**
     * An array mapping Severity enum to values displayed as groupheaders texts in
     * the "summary of ISC software versions detected by Stork" table.
     */
    subheaderMap: string[] = []

    /**
     * Keeps information whether "summary of ISC software versions detected by Stork" table's data is loading.
     */
    summaryDataLoading: boolean

    /**
     * Keeps information whether Kea, Bind9, Stork current versions tables data is loading.
     */
    swVersionsDataLoading: boolean

    /**
     * Keeps current software versions data received from version service.
     * @private
     */
    private _processedData: AppsVersions

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
        this.summaryDataLoading = true
        this.swVersionsDataLoading = true
        this._subscriptions.add(
            this.versionService.getDataManufactureDate().subscribe({
                next: (date) => {
                    this.dataDate = date
                    this.subheaderMap = [
                        'Security updates were found for ISC software used on those machines!',
                        'Those machines use ISC software version that require your attention. Software updates are available.',
                        'ISC software updates are available for those machines.',
                        '',
                        `Those machines use up-to-date ISC software (known as of ${this.dataDate})`,
                    ]
                },
                error: (err) => {
                    console.error('err1', err)
                    let msg = getErrorMessage(err)
                    this.messageService.add({
                        severity: 'error',
                        summary: 'Error retrieving software versions data',
                        detail: 'Error occurred when retrieving software versions data: ' + msg,
                        life: 10000,
                    })
                },
                complete: () => console.log('complete1'),
            })
        )
        this._subscriptions.add(
            this.versionService.isOnlineData().subscribe({
                next: (isOnline) => (this.isDataOffline = !isOnline),
                error: (err) => {
                    console.error('err2', err)
                    let msg = getErrorMessage(err)
                    this.messageService.add({
                        severity: 'error',
                        summary: 'Error retrieving software versions data',
                        detail: 'Error occurred when retrieving software versions data: ' + msg,
                        life: 10000,
                    })
                },
                complete: () => console.log('complete2'),
            })
        )
        this._subscriptions.add(
            this.versionService
                .getCurrentData()
                .pipe(
                    concatMap((data) => {
                        this._processedData = data
                        this.keaVersions = deepCopy(data?.kea?.currentStable ?? [])
                        if (data?.kea?.latestDev) {
                            this.keaVersions.push(data.kea?.latestDev)
                        }

                        this.bind9Versions = deepCopy(data.bind9?.currentStable ?? [])
                        if (data.bind9?.latestDev) {
                            this.bind9Versions.push(data.bind9?.latestDev)
                        }

                        this.storkVersions = deepCopy(data.stork?.currentStable ?? [])
                        if (data.stork?.latestDev) {
                            this.storkVersions.push(data.stork?.latestDev)
                        }

                        this.swVersionsDataLoading = false
                        this.counters = [0, 0, 0, 0, 0]
                        return this.servicesApi.getMachinesAppsVersions().pipe(
                            // tap(() => {
                            //     throw new Error('err from tap')
                            // }),
                            catchError((err) => {
                                console.error('err3', err)
                                let msg = getErrorMessage(err)
                                this.messageService.add({
                                    severity: 'error',
                                    summary: 'Error retrieving software versions data',
                                    detail: 'Error occurred when retrieving software versions data: ' + msg,
                                    life: 10000,
                                })
                                this.swVersionsDataLoading = false
                                return of({ items: [] })
                            })
                        )
                    })
                )
                .subscribe({
                    next: (data) => {
                        data.items.map((machine) => {
                            let m = machine as Machine & { versionCheckSeverity: Severity }
                            m.versionCheckSeverity = Severity.success
                            try {
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
                            } catch (err) {
                                console.error('err4', err)
                                let msg = getErrorMessage(err)
                                this.messageService.add({
                                    severity: 'error',
                                    summary: 'Error retrieving software versions data',
                                    detail: 'Error occurred when retrieving software versions data: ' + msg,
                                    life: 10000,
                                })
                            }

                            // TODO: daemons version match check

                            this.counters[m.versionCheckSeverity]++
                            return m
                        })
                        this.machines = data.items
                        this.summaryDataLoading = false
                    },
                    error: (err) => {
                        console.error('err5', err)
                        let msg = getErrorMessage(err)
                        this.messageService.add({
                            severity: 'error',
                            summary: 'Error retrieving software versions data',
                            detail: 'Error occurred when retrieving software versions data: ' + msg,
                            life: 10000,
                        })
                        this.swVersionsDataLoading = false
                        this.summaryDataLoading = false
                    },
                    complete: () => console.log('complete3'),
                })
        )
    }

    /**
     * Triggers software versions data refresh - new data will be sent from the backend via version service.
     */
    refreshVersions() {
        this.swVersionsDataLoading = true
        this.summaryDataLoading = true
        this.versionService.refreshData()
    }
}
