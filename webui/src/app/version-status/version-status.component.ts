import { Component, Input, OnInit } from '@angular/core'
import { valid, minor, lt } from 'semver'

interface VersionMetadata {
    latestStable?: string
    latestDev?: string
    latestSecure?: string
}

type App = 'kea' | 'bind' | 'stork'

@Component({
    selector: 'app-version-status',
    templateUrl: './version-status.component.html',
    styleUrl: './version-status.component.sass',
})
export class VersionStatusComponent implements OnInit {
    @Input({ required: true }) app: App

    @Input({ required: true }) version: string

    isDevelopmentVersion: boolean

    severity: 'error' | 'warning' | 'success'

    iconClasses = {}

    feedback: string

    // hardcode for now
    versionsMetadata: { [a in App]: VersionMetadata } = {
        kea: { latestStable: '2.6.1', latestDev: '2.7.2' },
        stork: { latestDev: '1.18.0', latestSecure: '1.15.1' },
        bind: { latestStable: '9.20.1', latestDev: '9.21.0' },
    }

    ngOnInit(): void {
        if (valid(this.version)) {
            this.checkDevelopmentVersion()
            this.compareVersions()
        } else {
            // TODO: graceful error logging
            console.error(`Provided semver ${this.version} is not valid!`)
        }
    }

    private checkDevelopmentVersion() {
        if (this.app === 'kea' || this.app === 'bind') {
            const minorVersion = minor(this.version)
            this.isDevelopmentVersion = minorVersion % 2 === 1
        }
    }

    private compareVersions() {
        // check security releases first
        if (
            this.versionsMetadata[this.app]?.hasOwnProperty('latestSecure') &&
            lt(this.version, this.versionsMetadata[this.app].latestSecure)
        ) {
            this.severity = 'error'
            this.feedback = `Security update ${this.versionsMetadata[this.app].latestSecure} was released. Please update as soon as possible.`
            this.iconClasses = { 'text-red-500': true, 'pi-exclamation-circle': true }
            return
        }

        // case - stable version
        if (
            this.isDevelopmentVersion === false &&
            this.versionsMetadata[this.app]?.hasOwnProperty('latestStable') &&
            lt(this.version, this.versionsMetadata[this.app].latestStable)
        ) {
            this.severity = 'warning'
            this.feedback = `Latest stable version is ${this.versionsMetadata[this.app].latestStable}. You are using ${this.version}. Update is recommended.`
            this.iconClasses = { 'text-orange-400': true, 'pi-exclamation-triangle': true }
            return
        }

        // case - development version
        if (this.isDevelopmentVersion === true) {
            if (
                this.versionsMetadata[this.app]?.hasOwnProperty('latestDev') &&
                lt(this.version, this.versionsMetadata[this.app].latestDev)
            ) {
                this.feedback = `You are using development release ${this.version}. Latest development release is ${this.versionsMetadata[this.app].latestDev}. Please consider updating.`
            }
            this.severity = 'warning'
            this.iconClasses = { 'text-orange-400': true, 'pi-exclamation-triangle': true }
            this.feedback = [
                this.feedback,
                'Please be advised that using dev release in production is not recommended! Consider updating to latest stable.',
            ].join(' ')
        }
    }
}
