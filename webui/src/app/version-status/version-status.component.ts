import { Component, Input, OnInit } from '@angular/core'
import { valid, minor, lt, satisfies, gt, coerce } from 'semver'
import { VersionService, App } from '../version.service'

/**
 *
 */
@Component({
    selector: 'app-version-status',
    templateUrl: './version-status.component.html',
    styleUrl: './version-status.component.sass',
})
export class VersionStatusComponent implements OnInit {
    /**
     *
     */
    @Input({ required: true }) app: App

    /**
     *
     */
    @Input({ required: true }) version: string

    /**
     *
     */
    @Input() showAppName = false

    @Input() inline = true

    /**
     *
     */
    appName: string

    /**
     *
     */
    isDevelopmentVersion: boolean

    /**
     *
     */
    severity: 'error' | 'warn' | 'success' | 'info'

    /**
     *
     */
    iconClasses = {}

    /**
     *
     */
    feedback: string

    /**
     *
     */
    constructor(private versionService: VersionService) {}

    /**
     *
     */
    ngOnInit(): void {
        this.version = coerce(this.version).version
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

    /**
     *
     * @private
     */
    private checkDevelopmentVersion() {
        if (this.app === 'kea' || this.app === 'bind9') {
            const minorVersion = minor(this.version)
            this.isDevelopmentVersion = minorVersion % 2 === 1
        } else {
            // Stork versions are all dev for now. To be updated with Stork 2.0.0.
            this.isDevelopmentVersion = true
        }
    }

    /**
     *
     * @param severity
     * @param feedback
     * @private
     */
    private setSeverity(severity: typeof this.severity, feedback: string) {
        this.severity = severity
        this.feedback = feedback
        switch (severity) {
            case 'success':
                this.iconClasses = { 'text-green-500': true, 'pi-check': true }
                break
            case 'warn':
                this.iconClasses = { 'text-orange-400': true, 'pi-exclamation-triangle': true }
                break
            case 'error':
                this.iconClasses = { 'text-red-500': true, 'pi-exclamation-circle': false, 'pi-times': true }
                break
            case 'info':
                this.iconClasses = { 'text-blue-300': true, 'pi-info-circle': true }
                break
        }
    }

    /**
     *
     * @private
     */
    private compareVersionsExt() {
        // check security releases first
        let latestSecureVersion = this.versionService.getVersion(this.app, 'latestSecure')
        if (latestSecureVersion && lt(this.version, latestSecureVersion as string)) {
            this.setSeverity(
                'error',
                `Security update ${latestSecureVersion} was released for ${this.appName}. Please update as soon as possible!`
            )
            return
        }

        // case - stable version
        let currentStableVersionDetails = this.versionService.getVersionDetails(this.app, 'currentStable')
        let dataDate = this.versionService.getDataManufactureDate()
        if (this.isDevelopmentVersion === false && currentStableVersionDetails) {
            if (Array.isArray(currentStableVersionDetails) && currentStableVersionDetails.length >= 1) {
                for (let details of currentStableVersionDetails) {
                    if (satisfies(this.version, details.range)) {
                        if (lt(this.version, details.version)) {
                            this.setSeverity(
                                'warn',
                                `Current stable ${this.appName} version (known as of ${dataDate}) is ${details.version}. You are using ${this.version}. Update is recommended.`
                            )
                        } else if (gt(this.version, details.version)) {
                            this.setSeverity(
                                'info',
                                `Current stable ${this.appName} version (known as of ${dataDate}) is ${details.version}. You are using more recent version ${this.version}.`
                            )
                        } else {
                            this.setSeverity(
                                'success',
                                `You have current ${this.appName} stable version (known as of ${dataDate}).`
                            )
                        }
                        return
                    }
                }
                // current version not matching currentStable ranges
                let stableVersions = this.versionService.getStableVersions(this.app)
                if (Array.isArray(stableVersions) && stableVersions.length > 0) {
                    let versionsText = stableVersions.join(', ')
                    if (lt(this.version, stableVersions[0])) {
                        // either semver major or minor are below min(current stable)
                        this.setSeverity(
                            'warn',
                            `Your ${this.appName} version ${this.version} is older than current stable version/s ${versionsText}. Update to current stable is recommended.`
                        )
                    } else {
                        // either semver major or minor are bigger than current stable
                        this.setSeverity(
                            'info',
                            `Your ${this.appName} version ${this.version} is more recent than current stable version/s ${versionsText} (known as of ${dataDate}).`
                        )
                        // this.feedback = `Current stable ${this.appName} version as of ${this.extendedMetadata.date} is/are ${versionsText}. You are using more recent version ${this.version}.`
                    }
                }
            }
            return
        }

        // case - development version
        let latestDevVersion = this.versionService.getVersion(this.app, 'latestDev')
        if (this.isDevelopmentVersion === true && latestDevVersion) {
            if (lt(this.version, latestDevVersion as string)) {
                this.setSeverity(
                    'warn',
                    `You are using ${this.appName} development version ${this.version}. Current development version (known as of ${dataDate}) is ${latestDevVersion}. Please consider updating.`
                )
            } else if (gt(this.version, latestDevVersion as string)) {
                this.setSeverity(
                    'info',
                    `Current development ${this.appName} version (known as of ${dataDate}) is ${latestDevVersion}. You are using more recent version ${this.version}.`
                )
            } else {
                this.setSeverity(
                    'success',
                    `You have current ${this.appName} development version (known as of ${dataDate}).`
                )
            }
            if (currentStableVersionDetails) {
                this.setSeverity(
                    'warn',
                    [
                        this.feedback,
                        `Please be advised that using development version in production is not recommended! Consider using ${this.appName} stable release.`,
                    ].join(' ')
                )
            }
        }
    }
}
