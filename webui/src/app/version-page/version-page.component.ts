import { Component, OnInit } from '@angular/core'
import { App, Severity, VersionDetails, VersionService } from '../version.service'
import { Machine, ServicesService } from '../backend'
import { deepCopy } from '../utils'

/**
 *
 */
@Component({
    selector: 'app-version-page',
    templateUrl: './version-page.component.html',
    styleUrl: './version-page.component.sass',
})
export class VersionPageComponent implements OnInit {
    keaVersions: VersionDetails[]
    bind9Versions: VersionDetails[]
    storkVersions: VersionDetails[]
    machines: Machine[]

    /**
     *
     */
    constructor(
        private versionService: VersionService,
        private servicesApi: ServicesService
    ) {}

    /**
     *
     */
    ngOnInit(): void {
        // prepare kea data
        let keaDetails = deepCopy(this.versionService.getVersionDetails('kea', 'currentStable'))
        this.keaVersions = keaDetails ? (keaDetails as VersionDetails[]) : []
        keaDetails = deepCopy(this.versionService.getVersionDetails('kea', 'latestDev'))
        if (keaDetails) {
            this.keaVersions.push(keaDetails as VersionDetails)
        }

        // prepare bind9 data
        let bindDetails = deepCopy(this.versionService.getVersionDetails('bind9', 'currentStable'))
        this.bind9Versions = bindDetails ? (bindDetails as VersionDetails[]) : []
        bindDetails = deepCopy(this.versionService.getVersionDetails('bind9', 'latestDev'))
        if (bindDetails) {
            this.bind9Versions.push(bindDetails as VersionDetails)
        }

        // prepare stork data
        let storkDetails = deepCopy(this.versionService.getVersionDetails('stork', 'currentStable'))
        this.storkVersions = storkDetails ? (storkDetails as VersionDetails[]) : []
        storkDetails = deepCopy(this.versionService.getVersionDetails('stork', 'latestDev'))
        if (storkDetails) {
            this.storkVersions.push(storkDetails as VersionDetails)
        }

        this.servicesApi.getMachines(0, 100, undefined, undefined, true).subscribe((data) => {
            this.machines = data.items ?? []
            for (let m of this.machines) {
                // TODO: enum?
                m.versionCheckSeverity = 4
                let storkVersionSeverity = this.versionService.checkVersion(m.agentVersion, 'stork')
                m.versionCheckSeverity = Math.min(
                    this.mapSeverityToNumber(storkVersionSeverity.severity),
                    m.versionCheckSeverity
                )
                for (let a of m.apps) {
                    let versionCheck = this.versionService.checkVersion(a.version, a.type as App)
                    if (versionCheck) {
                        m.versionCheckSeverity = Math.min(
                            this.mapSeverityToNumber(versionCheck.severity),
                            m.versionCheckSeverity
                        )
                    }
                }
            }
        })
    }

    /**
     * Configures the breadcrumbs for the component.
     */
    breadcrumbs = [{ label: 'Monitoring' }, { label: 'Software versions' }]

    /**
     * Returns true if version data source is offline json file.
     */
    get isDataOffline() {
        return !this.versionService.isOnlineData()
    }

    /**
     *
     */
    get dataManufactureDate() {
        return this.versionService.getDataManufactureDate()
    }

    /**
     *
     * @param severity
     * @private
     */
    private mapSeverityToNumber(severity: Severity) {
        switch (severity) {
            // case 'error':
            case 'danger':
                return 1
            // case 'warn':
            case 'warning':
                return 2
            case 'info':
                return 3
            case 'success':
            case 'secondary':
                return 4
        }
    }

    /**
     *
     * @param number
     */
    mapNumberToSeverity(number: number): Severity {
        switch (number) {
            case 1:
                // return 'error'
                return 'danger'
            case 2:
                // return 'warn'
                return 'warning'
            case 3:
                return 'info'
            case 4:
                return 'success'
        }
    }

    /**
     *
     * @param number
     */
    getSubheader(number: number) {
        switch (number) {
            case 1:
                return 'Security updates were found for ISC software used on those machines!'
            case 2:
                return 'Those machines use ISC software version that require your attention. Software updates are available.'
            case 3:
                return 'ISC software updates are available for those machines.'
            case 4:
                return `Those machines use up-to-date ISC software (known as of ${this.dataManufactureDate})`
        }
    }
}
