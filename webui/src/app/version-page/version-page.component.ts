import { Component, OnDestroy, OnInit } from '@angular/core'
import { App, Severity, VersionDetails, VersionService } from '../version.service'
import { AppsVersions, Machine, ServicesService } from '../backend'
import { deepCopy } from '../utils'
import { Subscription } from 'rxjs'

/**
 *
 */
@Component({
    selector: 'app-version-page',
    templateUrl: './version-page.component.html',
    styleUrl: './version-page.component.sass',
})
export class VersionPageComponent implements OnInit, OnDestroy {
    private _subscriptions = new Subscription()
    /**
     * Returns true if version data source is offline json file.
     */
    isDataOffline: boolean
    // () {
    //     return !this.versionService.isOnlineData()
    // }
    keaVersions: VersionDetails[] = []
    bind9Versions: VersionDetails[] = []
    storkVersions: VersionDetails[] = []
    protected readonly Severity = Severity
    severityMap: Severity[] = [
        Severity.danger,
        Severity.warning,
        Severity.info,
        Severity.success, // SeverityEnum.secondary is mapped to SeverityEnum.success
        Severity.success,
    ]
    counters = [0, 0, 0, 0, 0]
    dataDate: string = 'unknown'
    subheaderMap: string[] = []
    summaryDataLoading: boolean
    swVersionsDataLoading: boolean

    private processedData: { processedData: AppsVersions; stableVersions: { [a in App]: string[] } }

    machines: Machine[]

    /**
     *
     */
    constructor(
        private versionService: VersionService,
        private servicesApi: ServicesService
    ) {}

    ngOnDestroy(): void {
        this._subscriptions.unsubscribe()
    }

    /**
     *
     */
    ngOnInit(): void {
        this.summaryDataLoading = true
        this.swVersionsDataLoading = true
        this._subscriptions.add(
            this.versionService.getDataManufactureDateAsync().subscribe((date) => {
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
            this.versionService.getProcessedData().subscribe((data) => {
                this.processedData = data

                this.keaVersions = deepCopy(data.processedData?.kea?.currentStable ?? [])
                if (data.processedData?.kea?.latestDev) {
                    this.keaVersions.push(data.processedData?.kea?.latestDev)
                }

                this.bind9Versions = deepCopy(data.processedData?.bind9?.currentStable ?? [])
                if (data.processedData?.bind9?.latestDev) {
                    this.bind9Versions.push(data.processedData?.bind9?.latestDev)
                }

                this.storkVersions = deepCopy(data.processedData?.stork?.currentStable ?? [])
                if (data.processedData?.stork?.latestDev) {
                    this.storkVersions.push(data.processedData?.stork?.latestDev)
                }

                this.swVersionsDataLoading = false

                this.counters = [0, 0, 0, 0, 0]
                this.servicesApi.getMachinesAppsVersions().subscribe({
                    next: (data) => {
                        data.items.map((m) => {
                            m.versionCheckSeverity = Severity.success
                            // TODO: daemons version match check
                            m.versionCheckSeverity = Math.min(
                                this.severityMap[
                                    this.versionService.checkVersionSync(m.agentVersion, 'stork')?.severity ??
                                        Severity.success
                                ],
                                m.versionCheckSeverity
                            )
                            m.apps.forEach((a) => {
                                m.versionCheckSeverity = Math.min(
                                    this.severityMap[
                                        this.versionService.checkVersionSync(a.version, a.type as App)?.severity ??
                                            Severity.success
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

        // this.servicesApi.getMachines(0, 100, undefined, undefined, true)
        // this.servicesApi.getMachinesAppsVersions().pipe(
        //     map((d)=>d.items),
        //     mergeMap((items)=>forkJoin(
        //         this.versionService.checkVersion(items[0].agentVersion, 'stork'),
        //         this.versionService.checkVersion(items[0].apps[0].version, items[0].apps[0].type as App),
        //         (sever1, sever2) => {
        //             items[0].versionCheckSeverity = Math.min(sever1.severity, sever2.severity)
        //             return items
        //         }
        //     ))
        //
        //
        // )
        //
        //     .subscribe({
        //     next: (data) => {
        //         this.machines = data ?? []
        //         // for (let m of this.machines) {
        //         //     m.agentVersion = this.storkVers[this.sI++ % this.storkVers.length]
        //         //
        //         //     m.versionCheckSeverity = Severity.success
        //         //     let storkCheck = this.versionService.checkVersion(m.agentVersion, 'stork')
        //         //     // TODO: daemons version match check
        //         //     if (storkCheck) {
        //         //         m.versionCheckSeverity = Math.min(this.severityMap[storkCheck.severity], m.versionCheckSeverity)
        //         //     }
        //         //
        //         //     for (let a of m.apps) {
        //         //         if (a.type === 'kea') {
        //         //             a.version = this.keaVers[this.kI++ % this.keaVers.length]
        //         //             let dV = undefined
        //         //             let dIdx = 0
        //         //             for (let d of a.details.daemons) {
        //         //                 if (dIdx > 0 && dV !== d.version) {
        //         //                     console.error('Kea daemons versions mismatch!')
        //         //                 }
        //         //
        //         //                 dV = d.version
        //         //                 console.log('kea daemon', dIdx, d.version)
        //         //                 dIdx++
        //         //             }
        //         //         }
        //         //         let versionCheck = this.versionService.checkVersion(a.version, a.type as App)
        //         //         if (versionCheck) {
        //         //             m.versionCheckSeverity = Math.min(
        //         //                 this.severityMap[versionCheck.severity],
        //         //                 m.versionCheckSeverity
        //         //             )
        //         //         }
        //         //     }
        //         // }
        //         this.dataLoading = false
        //     },
        //     complete: () => {
        //         console.log('getMachinesAppsVersions complete')
        //     },
        // })
    }

    /**
     * Configures the breadcrumbs for the component.
     */
    breadcrumbs = [{ label: 'Monitoring' }, { label: 'Software versions' }]

    refreshVersions() {
        this.swVersionsDataLoading = true
        this.summaryDataLoading = true
        this.versionService.refreshData()
    }
}
