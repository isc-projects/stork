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
                for (let a of m.apps) {
                    let versionCheck = this.versionService.checkVersion(a.version, a.type as App)
                    if (versionCheck) {
                        // TODO: severity precedence if more than one app per machine
                        m.versionCheckSeverity = this.mapSeverityToLetter(versionCheck.severity)
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
    isDataOffline() {
        return !this.versionService.isOnlineData()
    }

    /**
     *
     */
    getDataManufactureDate() {
        return this.versionService.getDataManufactureDate()
    }

    private mapSeverityToLetter(s: Severity) {
        switch (s) {
            case 'error':
                return 'a'
            case 'warn':
                return 'b'
            case 'info':
                return 'c'
            case 'success':
                return 'd'
        }
    }

    mapLetterToSeverity(l: string): Severity {
        switch (l) {
            case 'a':
                return 'error'
            case 'b':
                return 'warn'
            case 'c':
                return 'info'
            case 'd':
                return 'success'
        }
    }

    getSubheader(l: string) {
        switch (l) {
            case 'a':
                return 'Stork detected that below machines use software for which security updates are available.'
            case 'b':
                return 'Below machines use software versions that require your attention. Updating is possible.'
            case 'c':
                return 'All good but update is possible.'
            case 'd':
                return 'All good here. Current versions detected.'
        }
    }
}
