import { Component, Input, OnInit } from '@angular/core'
import { valid, minor, lt } from 'semver'

interface VersionMetadata {
    latestStable?: string
    latestDev?: string
    latestSecure?: string
}

interface VersionDetail {
    version: string
    releaseDate: string
    eolDate?: string
    ESV?: string
}

interface AppVersionMetadata {
    currentStable?: VersionDetail | VersionDetail[]
    latestDev: VersionDetail
    latestSecure?: VersionDetail
}

type App = 'kea' | 'bind9' | 'stork'

@Component({
    selector: 'app-version-status',
    templateUrl: './version-status.component.html',
    styleUrl: './version-status.component.sass',
})
export class VersionStatusComponent implements OnInit {
    @Input({ required: true }) app: App

    @Input({ required: true }) version: string

    @Input() showAppName = false

    appName: string

    isDevelopmentVersion: boolean

    severity: 'error' | 'warning' | 'success'

    iconClasses = {}

    feedback: string

    // hardcode for now
    versionsMetadata: { [a in App | 'date']: VersionMetadata | string } = {
        kea: { latestStable: '2.6.1', latestDev: '2.7.2' },
        stork: { latestDev: '1.18.0', latestSecure: '1.15.1' },
        bind9: { latestStable: '9.20.1', latestDev: '9.21.0' },
        date: '2024-09-10',
    }

    test: {[a in App | 'date']: AppVersionMetadata | string} = {
        date: '2024-09-10',
        kea: {
            currentStable: [
                {
                    version: '2.6.1',
                    releaseDate: '2024-07-31',
                    eolDate: '2026-07-01'
                },
                {
                    version: '2.4.1',
                    releaseDate: '2023-11-29',
                    eolDate: '2025-07-01'
                },
            ],
            latestDev: {
                version: '2.7.2',
                releaseDate: '2024-08-28'
            }
        },
        stork: {
            latestDev: {
                version: '1.18.0',
                releaseDate: '2024-08-07'
            },
            latestSecure: {
                version: '1.15.1',
                releaseDate: '2024-03-27'
            }
        },
        bind9: {
            currentStable: [
                {
                    version: '9.18.29',
                    releaseDate: '2024-08-21',
                    eolDate: '2026-07-01',
                    ESV: 'true'
                },
                {
                    version: '9.20.1',
                    releaseDate: '2024-08-28',
                    eolDate: '2028-07-01'
                },
            ],
            latestDev: {
                version: '9.21.0',
                releaseDate: '2024-08-28'
            }
        }
    }

    ngOnInit(): void {
        if (valid(this.version)) {
            this.appName = this.app[0].toUpperCase() + this.app.slice(1)
            this.appName += this.app === 'stork' ? ' agent' : ''
            this.checkDevelopmentVersion()
            this.compareVersionsExt()
        } else {
            // TODO: graceful error logging
            console.error(`Provided semver ${this.version} is not valid!`)
        }
    }

    private checkDevelopmentVersion() {
        if (this.app === 'kea' || this.app === 'bind9') {
            const minorVersion = minor(this.version)
            this.isDevelopmentVersion = minorVersion % 2 === 1
        } else {
            // Stork versions are all dev for now. To be updated with Stork 2.0.0.
            this.isDevelopmentVersion = true
        }
    }

    private compareVersions() {
        // check security releases first
        if (
            this.versionsMetadata[this.app]?.hasOwnProperty('latestSecure') &&
            lt(this.version, (this.versionsMetadata[this.app] as VersionMetadata).latestSecure)
        ) {
            this.severity = 'error'
            this.feedback = `Security update ${(this.versionsMetadata[this.app] as VersionMetadata).latestSecure} was released for ${this.appName}. Please update as soon as possible.`
            this.iconClasses = { 'text-red-500': true, 'pi-exclamation-circle': true }
            return
        }

        // case - stable version
        if (this.isDevelopmentVersion === false && this.versionsMetadata[this.app]?.hasOwnProperty('latestStable')) {
            if (lt(this.version, (this.versionsMetadata[this.app] as VersionMetadata).latestStable)) {
                this.severity = 'warning'
                this.feedback = `Latest stable ${this.appName} version is ${(this.versionsMetadata[this.app] as VersionMetadata).latestStable}. You are using ${this.version}. Update is recommended.`
                this.iconClasses = { 'text-orange-400': true, 'pi-exclamation-triangle': true }
            } else {
                this.severity = 'success'
                this.feedback = `You have the latest ${this.appName} stable version. This information is based on data from ${this.versionsMetadata.date}.`
                this.iconClasses = { 'text-green-500': true, 'pi-check': true }
            }
            return
        }

        // case - development version
        if (this.isDevelopmentVersion === true && this.versionsMetadata[this.app]?.hasOwnProperty('latestDev')) {
            if (lt(this.version, (this.versionsMetadata[this.app] as VersionMetadata).latestDev)) {
                this.feedback = `You are using ${this.appName} development version ${this.version}. Latest development version is ${(this.versionsMetadata[this.app] as VersionMetadata).latestDev}. Please consider updating.`
                this.severity = 'warning'
                this.iconClasses = { 'text-orange-400': true, 'pi-exclamation-triangle': true }
            } else {
                this.severity = 'success'
                this.feedback = `You have the latest ${this.appName} development version. This information is based on data from ${this.versionsMetadata.date}.`
                this.iconClasses = { 'text-green-500': true, 'pi-check': true }
            }
            if (this.versionsMetadata[this.app]?.hasOwnProperty('latestStable')) {
                this.severity = 'warning'
                this.iconClasses = {
                    'text-green-500': false,
                    'pi-check': false,
                    'text-orange-400': true,
                    'pi-exclamation-triangle': true,
                }
                this.feedback = [
                    this.feedback,
                    `Please be advised that using development version in production is not recommended! Consider using ${this.appName} stable release.`,
                ].join(' ')
            }
        }
    }

    private compareVersionsExt() {
        // check security releases first
        if (
            this.test[this.app]?.hasOwnProperty('latestSecure') &&
            lt(this.version, (this.test[this.app] as AppVersionMetadata).latestSecure.version)
        ) {
            this.severity = 'error'
            this.feedback = `Security update ${(this.test[this.app] as AppVersionMetadata).latestSecure.version} was released for ${this.appName}. Please update as soon as possible.`
            this.iconClasses = { 'text-red-500': true, 'pi-exclamation-circle': true }
            return
        }

        // case - stable version
        if (this.isDevelopmentVersion === false && this.test[this.app]?.hasOwnProperty('currentStable')) {
            if (Array.isArray((this.test[this.app] as AppVersionMetadata).currentStable)) {
                // todo: array of metadata handling
                this.severity = 'warning'
                this.feedback = `TBD`
                this.iconClasses = { 'text-primary-500': true, 'pi-search': true }
            } else {
                if (lt(this.version, ((this.test[this.app] as AppVersionMetadata).currentStable as VersionDetail).version)) {
                    this.severity = 'warning'
                    this.feedback = `Latest stable ${this.appName} version is ${((this.test[this.app] as AppVersionMetadata).currentStable as VersionDetail).version}. You are using ${this.version}. Update is recommended.`
                    this.iconClasses = { 'text-orange-400': true, 'pi-exclamation-triangle': true }
                } else {
                    this.severity = 'success'
                    this.feedback = `You have the latest ${this.appName} stable version. This information is based on data from ${this.test.date}.`
                    this.iconClasses = { 'text-green-500': true, 'pi-check': true }
                }
            }
            return
        }

        // case - development version
        if (this.isDevelopmentVersion === true && this.test[this.app]?.hasOwnProperty('latestDev')) {
            if (lt(this.version, (this.test[this.app] as AppVersionMetadata).latestDev.version)) {
                this.feedback = `You are using ${this.appName} development version ${this.version}. Latest development version is ${(this.test[this.app] as AppVersionMetadata).latestDev.version}. Please consider updating.`
                this.severity = 'warning'
                this.iconClasses = { 'text-orange-400': true, 'pi-exclamation-triangle': true }
            } else {
                this.severity = 'success'
                this.feedback = `You have the latest ${this.appName} development version. This information is based on data from ${this.test.date}.`
                this.iconClasses = { 'text-green-500': true, 'pi-check': true }
            }
            if (this.test[this.app]?.hasOwnProperty('currentStable')) {
                this.severity = 'warning'
                this.iconClasses = {
                    'text-green-500': false,
                    'pi-check': false,
                    'text-orange-400': true,
                    'pi-exclamation-triangle': true,
                }
                this.feedback = [
                    this.feedback,
                    `Please be advised that using development version in production is not recommended! Consider using ${this.appName} stable release.`,
                ].join(' ')
            }
        }
    }
}
