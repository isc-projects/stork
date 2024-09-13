import { Component, OnInit } from '@angular/core'
import { App, AppVersionMetadata, VersionDetails } from '../version-status/version-status.component'
import { minor, major } from 'semver'

/**
 *
 */
@Component({
    selector: 'app-version-page',
    templateUrl: './version-page.component.html',
    styleUrl: './version-page.component.sass',
})
export class VersionPageComponent implements OnInit {
    keaVersions: VersionDetails[] = []
    bind9Versions: VersionDetails[] = []
    storkVersions: VersionDetails[] = []

    /**
     *
     */
    ngOnInit(): void {
        // prepare kea data
        for (let s of (this.extendedMetadata.kea as AppVersionMetadata).currentStable) {
            let record = s
            record.status = 'Current Stable'
            record.major = major(s.version)
            record.minor = minor(s.version)
            this.keaVersions.push(record)
        }

        let devRecord = (this.extendedMetadata.kea as AppVersionMetadata).latestDev
        devRecord.status = 'Development'
        devRecord.major = major(devRecord.version)
        devRecord.minor = minor(devRecord.version)
        this.keaVersions.push(devRecord)

        // prepare bind9 data
        for (let s of (this.extendedMetadata.bind9 as AppVersionMetadata).currentStable) {
            let record = s
            record.status = 'Current Stable'
            record.major = major(s.version)
            record.minor = minor(s.version)
            this.bind9Versions.push(record)
        }

        devRecord = (this.extendedMetadata.bind9 as AppVersionMetadata).latestDev
        devRecord.status = 'Development'
        devRecord.major = major(devRecord.version)
        devRecord.minor = minor(devRecord.version)
        this.bind9Versions.push(devRecord)

        // prepare stork data
        if ((this.extendedMetadata.stork as AppVersionMetadata).currentStable) {
            for (let s of (this.extendedMetadata.stork as AppVersionMetadata).currentStable) {
                let record = s
                record.status = 'Current Stable'
                record.major = major(s.version)
                record.minor = minor(s.version)
                this.storkVersions.push(record)
            }
        }

        devRecord = (this.extendedMetadata.stork as AppVersionMetadata).latestDev
        devRecord.status = 'Development'
        devRecord.major = major(devRecord.version)
        devRecord.minor = minor(devRecord.version)
        this.storkVersions.push(devRecord)
    }

    /**
     *
     */
    extendedMetadata: { [a in App | 'date']: AppVersionMetadata | string } = {
        date: '2024-09-01',
        kea: {
            currentStable: [
                {
                    version: '2.6.1',
                    releaseDate: '2024-07-31',
                    eolDate: '2026-07-01',
                },
                {
                    version: '2.4.1',
                    releaseDate: '2023-11-29',
                    eolDate: '2025-07-01',
                },
            ],
            latestDev: {
                version: '2.7.2',
                releaseDate: '2024-08-28',
            },
        },
        stork: {
            latestDev: {
                version: '1.18.0',
                releaseDate: '2024-08-07',
            },
            latestSecure: {
                version: '1.15.1',
                releaseDate: '2024-03-27',
            },
        },
        bind9: {
            currentStable: [
                {
                    version: '9.18.29',
                    releaseDate: '2024-08-21',
                    eolDate: '2026-07-01',
                    ESV: 'true',
                },
                {
                    version: '9.20.1',
                    releaseDate: '2024-08-28',
                    eolDate: '2028-07-01',
                },
            ],
            latestDev: {
                version: '9.21.0',
                releaseDate: '2024-08-28',
            },
        },
    }

    /**
     * Configures the breadcrumbs for the component.
     */
    breadcrumbs = [{ label: 'Monitoring' }, { label: 'Software versions' }]
}
