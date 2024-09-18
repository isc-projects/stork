import { Component, OnInit } from '@angular/core'
import { VersionDetails, VersionService } from '../version.service'
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
}
