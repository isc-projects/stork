import { Component, OnInit } from '@angular/core'
import { VersionDetails, VersionService } from '../version.service'

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

    /**
     *
     */
    constructor(private versionService: VersionService) {}

    /**
     *
     */
    ngOnInit(): void {
        // prepare kea data
        let keaDetails = this.versionService.getVersionDetails('kea', 'currentStable')
        this.keaVersions = keaDetails ? (keaDetails as VersionDetails[]) : []
        keaDetails = this.versionService.getVersionDetails('kea', 'latestDev')
        if (keaDetails) {
            this.keaVersions.push(keaDetails as VersionDetails)
        }

        // prepare bind9 data
        let bindDetails = this.versionService.getVersionDetails('bind9', 'currentStable')
        this.bind9Versions = bindDetails ? (bindDetails as VersionDetails[]) : []
        bindDetails = this.versionService.getVersionDetails('bind9', 'latestDev')
        if (bindDetails) {
            this.bind9Versions.push(bindDetails as VersionDetails)
        }

        // prepare stork data
        let storkDetails = this.versionService.getVersionDetails('stork', 'currentStable')
        this.storkVersions = storkDetails ? (storkDetails as VersionDetails[]) : []
        storkDetails = this.versionService.getVersionDetails('stork', 'latestDev')
        if (storkDetails) {
            this.storkVersions.push(storkDetails as VersionDetails)
        }
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
