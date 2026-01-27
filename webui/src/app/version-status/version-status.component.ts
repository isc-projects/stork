import { Component, Input, OnDestroy, OnInit } from '@angular/core'
import {
    DaemonName,
    getDaemonAppType,
    isIscDaemon,
    Severity,
    VersionFeedback,
    VersionService,
} from '../version.service'
import { ToastMessageOptions, MessageService } from 'primeng/api'
import { first, Subscription } from 'rxjs'
import { getErrorMessage, getIconBySeverity } from '../utils'
import { map } from 'rxjs/operators'
import { NgIf } from '@angular/common'
import { RouterLink } from '@angular/router'
import { Tooltip } from 'primeng/tooltip'
import { Message } from 'primeng/message'

/**
 * This component displays feedback information about the used version of either Kea, BIND 9, or Stork software.
 * The given version is compared with current known released versions. Feedback contains information
 * about available software updates and how severe the urge to update the software is.
 * The component can be either an inline icon with a tooltip or a block container with feedback visible upfront.
 *
 * The software version check is only done for ISC software. For non-ISC software, e.g. PowerDNS,
 * the component only displays the version without checking if it is up to date.
 */
@Component({
    selector: 'app-version-status',
    templateUrl: './version-status.component.html',
    styleUrl: './version-status.component.sass',
    imports: [NgIf, RouterLink, Tooltip, Message],
})
export class VersionStatusComponent implements OnInit, OnDestroy {
    /**
     * Daemon for which the version check is done.
     * The daemon object should contain its name ('dhcp4', 'dhcp6', 'd2', 'ca', 'netconf' (Kea daemons),
     * 'named' (BIND9), 'pdns' (PowerDNS), or special value 'stork' for the Stork agent).
     * The daemon object should also contain version string.
     * It may also contain numeric daemon ID.
     */
    @Input({ required: true }) daemon!: { id?: number; name?: DaemonName; version?: string }

    /**
     * This flag sets whether the component should display the version as a clickable link to
     * the daemon details tab view. It is relevant only when the inline mode is used.
     * Defaults to true.
     */
    @Input() includeAnchor = true

    /**
     * Version of the software for which the check is done.
     * It is a semver sanitized from provided version string in the input daemon object.
     */
    version: string

    /**
     * This flag sets whether the component has a form of inline icon with the tooltip,
     * or block message.
     * Defaults to true.
     */
    @Input() inline = true

    /**
     * Style class of the component.
     */
    @Input() styleClass: string | undefined

    /**
     * This holds the information how severe the urge to update the software is.
     * It is used to style the icon or the block message.
     */
    severity: Severity

    /**
     * HTML classes added to the icon to apply the style based on severity.
     */
    iconClasses = {}

    /**
     * An array of feedback messages displayed either in the tooltip or in the block message.
     */
    feedbackMessages: string[] = []

    /**
     * Holds PrimeNG Message value for the block message.
     */
    messages: ToastMessageOptions[] | undefined

    /**
     * Full name of the app. This is either 'Kea', 'Bind9' or 'Stork agent'. This is computed based on daemon name.
     */
    appName: string

    /**
     * RxJS Subscription holding all subscriptions to Observables, so that they can be all unsubscribed
     * at once onDestroy.
     * @private
     */
    private _subscriptions = new Subscription()

    /**
     * Class constructor.
     * @param versionService version service used to do software version checking; it returns the feedback about version used
     * @param messageService message service used to display errors
     */
    constructor(
        private versionService: VersionService,
        private messageService: MessageService
    ) {}

    /**
     * Component lifecycle hook called upon initialization. It sets the component's look and the feedback
     * it presents based on the data received from injected service.
     * All the data about current Kea, BIND9 and Stork releases is fetched here from the VersionService.
     * The service gets the data from Stork server. For now, the data source is an offline JSON file,
     * which is valid for the day of Stork release. In the future, an online data source will be the
     * primary one, and the offline will be a fallback option.
     */
    ngOnInit(): void {
        const sanitizedSemver = this.versionService.sanitizeSemver(this.daemon.version)
        if (sanitizedSemver) {
            this.version = sanitizedSemver
        }

        // Mute version checks for non-ISC apps. Version is mandatory. In case it is
        // falsy (undefined, null, empty string), simply return. No feedback will be displayed.
        this.iscApp = isIscDaemon(this.daemon.name)
        if (!this.iscApp || !this.daemon.version) {
            return
        }

        const app = getDaemonAppType(this.daemon.name) as string
        switch (app) {
            case 'bind9':
                this.appName = 'BIND9'
                break
            case 'pdns':
                this.appName = 'PowerDNS'
                break
            case 'stork':
                this.appName = 'Stork agent'
                break
            case 'kea':
                this.appName = 'Kea'
                break
            default:
                this.appName = app?.length ? app[0].toUpperCase() + app.slice(1) : ''
                break
        }

        if (sanitizedSemver) {
            this._subscriptions.add(
                this.versionService
                    .getCurrentData()
                    .pipe(
                        map((data) => {
                            return this.versionService.getSoftwareVersionFeedback(
                                sanitizedSemver,
                                this.daemon.name,
                                data
                            )
                        }),
                        // Use first() operator to unsubscribe after receiving first data.
                        // This is to avoid too many subscriptions for larger Stork deployments.
                        first()
                    )
                    .subscribe({
                        next: (feedback) => {
                            if (feedback) {
                                this.setSeverity(feedback)
                            }
                        },
                        error: (err) => {
                            const msg = getErrorMessage(err)
                            this.messageService.add({
                                severity: 'error',
                                summary: 'Error fetching software version data',
                                detail: 'Error when fetching software version data: ' + msg,
                                life: 10000,
                            })
                        },
                    })
            )
        } else {
            this.messageService.add({
                severity: 'error',
                summary: 'Error parsing software version',
                detail: `Couldn't parse valid semver from given ${this.daemon.version} version!`,
                life: 10000,
            })
        }
    }

    /**
     * Component lifecycle hook called to perform clean-up when destroying the component.
     */
    ngOnDestroy(): void {
        this._subscriptions.unsubscribe()
    }

    /**
     * Holds true if the app is a Kea, BIND9 or Stork agent; false otherwise.
     */
    iscApp: boolean

    /**
     * Sets the severity and the feedback messages. Icon classes are set based on the severity.
     * @param feedback feedback severity and message to be set
     * @private
     */
    private setSeverity(feedback: VersionFeedback) {
        this.severity = feedback.severity
        this.feedbackMessages = feedback.messages ?? []
        const m: ToastMessageOptions = {
            severity: Severity[feedback.severity],
            summary: `${this.appName} ${this.version}`,
            detail: feedback.messages.join('<br><br>'),
        }

        if (feedback.severity === Severity.secondary) {
            m.icon = 'pi-info-circle'
        }

        this.messages = [m]

        switch (feedback.severity) {
            case Severity.success:
                this.iconClasses = { 'text-green-500': true, 'pi-check': true }
                break
            case Severity.warn:
                this.iconClasses = { 'text-orange-400': true, 'pi-exclamation-triangle': true }
                break
            case Severity.error:
                this.iconClasses = { 'text-red-500': true, 'pi-exclamation-circle': false, 'pi-times': true }
                break
            case Severity.info:
            case Severity.secondary:
                this.iconClasses = { 'text-blue-300': true, 'pi-info-circle': true }
                break
        }
    }

    /**
     * Reference to the function so it can be used in the html template.
     * @protected
     */
    protected readonly getIconBySeverity = getIconBySeverity

    /**
     * Reference to the enum so it can be used in the html template.
     * @protected
     */
    protected readonly Severity = Severity
}
