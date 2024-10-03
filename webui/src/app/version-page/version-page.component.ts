import { Component, OnDestroy, OnInit } from '@angular/core'
import { App, Severity, VersionService } from '../version.service'
import { AppsVersions, Machine, ServicesService, VersionDetails } from '../backend'
import { deepCopy } from '../utils'
import { Subscription } from 'rxjs'

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
        Severity.danger,
        Severity.warning,
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
     */
    constructor(
        private versionService: VersionService,
        private servicesApi: ServicesService
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
            this.versionService.getDataManufactureDate().subscribe((date) => {
                this.dataDate = date
                this.subheaderMap = [
                    'Security updates were found for ISC software used on those machines!',
                    'Those machines use ISC software version that require your attention. Software updates are available.',
                    'ISC software updates are available for those machines.',
                    '',
                    `Those machines use up-to-date ISC software (known as of ${this.dataDate})`,
                ]
            })
        )
        this._subscriptions.add(
            this.versionService.isOnlineData().subscribe((isOnline) => (this.isDataOffline = !isOnline))
        )
        this._subscriptions.add(
            this.versionService.getCurrentData().subscribe((data) => {
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
                // todo: forkJoin or other operator instead inner subscribe?
                this.servicesApi.getMachinesAppsVersions().subscribe({
                    next: (data) => {
                        data.items.map((m) => {
                            m.versionCheckSeverity = Severity.success
                            // TODO: daemons version match check
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
                                // shelve begin
                                if (a.type === 'kea') {
                                    a.version = this.keaVers[this.kI++ % this.keaVers.length]
                                }
                                // shelve end

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
                            this.counters[m.versionCheckSeverity]++
                            return m
                        })
                        this.machines = data.items
                        this.summaryDataLoading = false
                    },
                })
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
