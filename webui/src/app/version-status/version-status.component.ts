import { Component, Input, OnDestroy, OnInit } from '@angular/core'
import { AppType, Severity, VersionFeedback, VersionService } from '../version.service'
import { Message, MessageService } from 'primeng/api'
import { first, Subscription } from 'rxjs'
import { getErrorMessage } from '../utils'
import { map } from 'rxjs/operators'

/**
 * This component displays feedback information about the used version of either Kea, Bind9, or Stork software.
 * The given version is compared with current known released versions. Feedback contains information
 * about available software updates and how severe the urge to update the software is.
 * The component can be either an inline icon with a tooltip or a block container with feedback visible upfront.
 */
@Component({
    selector: 'app-version-status',
    templateUrl: './version-status.component.html',
    styleUrl: './version-status.component.sass',
})
export class VersionStatusComponent implements OnInit, OnDestroy {
    /**
     * Type of software for which the version check is done.
     */
    @Input({ required: true }) app!: AppType

    /**
     * Version of the software for which the check is done. This must contain parsable semver.
     */
    @Input({ required: true }) version: string

    /**
     * For inline component version, this flag enables showing the app name with its version on the left side
     * of the icon with the tooltip.
     * Defaults to false.
     */
    @Input() showAppName = false

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
    messages: Message[] | undefined

    /**
     * Full name of the app. This is either 'Kea', 'Bind9' or 'Stork agent'. This is computed based on app field.
     * @private
     */
    private _appName: string

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
     * Component lifecycle hook called upon initialization.
     */
    ngOnInit(): void {
        this._appName = this.app[0].toUpperCase() + this.app.slice(1)
        this._appName += this.app === 'stork' ? ' agent' : ''
        let sanitizedSemver = this.versionService.sanitizeSemver(this.version)
        if (sanitizedSemver) {
            this.version = sanitizedSemver
            this._subscriptions.add(
                this.versionService
                    .getCurrentData()
                    .pipe(
                        map((data) => {
                            return this.versionService.getSoftwareVersionFeedback(sanitizedSemver, this.app, data)
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
                            let msg = getErrorMessage(err)
                            this.messageService.add({
                                severity: 'error',
                                summary: 'Error fetching software versions data',
                                detail: 'Error when fetching software versions data: ' + msg,
                                life: 10000,
                            })
                        },
                    })
            )
        } else {
            this.messageService.add({
                severity: 'error',
                summary: 'Error parsing software version',
                detail: `Couldn't parse valid semver from given ${this.version} version!`,
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
     * Getter of the full name of the app. This is either 'Kea', 'Bind9' or 'Stork agent'. This is computed based on the app field.
     */
    get appName() {
        return this._appName
    }

    /**
     * Sets the severity and the feedback messages. Icon classes are set based on the severity.
     * @param feedback feedback severity and message to be set
     * @private
     */
    private setSeverity(feedback: VersionFeedback) {
        this.severity = feedback.severity
        this.feedbackMessages = feedback.messages ?? []
        let m: Message = {
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
}
