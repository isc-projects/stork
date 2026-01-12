import { Component, Input, OnDestroy, OnInit } from '@angular/core'
import { DaemonName, isKeaDaemon, Severity, VersionFeedback, VersionService } from '../version.service'
import { ToastMessageOptions, MessageService } from 'primeng/api'
import { first, Subscription } from 'rxjs'
import { daemonNameToFriendlyName, getErrorMessage, getIconBySeverity } from '../utils'
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
     * Daemon name for which the version check is done.
     * Accepts daemon names like 'dhcp4', 'dhcp6', 'd2', 'ca', 'netconf' (Kea daemons),
     * 'named' (BIND9), 'pdns' (PowerDNS), or special value 'stork' for the Stork agent.
     */
    @Input({ required: true }) daemonName!: DaemonName

    /**
     * Version of the software for which the check is done. This must contain parsable semver.
     */
    @Input({ required: true }) version: string

    /**
     * For inline component version, this flag enables showing the daemon name with its version on the left side
     * of the icon with the tooltip.
     * Defaults to false.
     */
    @Input() showDaemonName = false

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
     * Friendly display name for the daemon. Computed using daemonNameToFriendlyName.
     * For 'stork' daemon, it returns 'Stork agent'.
     * @private
     */
    private _daemonDisplayName: string

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
        // Set display name using daemonNameToFriendlyName, with special case for stork
        if (this.daemonName === 'stork') {
            this._daemonDisplayName = 'Stork agent'
        } else {
            this._daemonDisplayName = daemonNameToFriendlyName(this.daemonName)
        }
        // Mute version checks for non-ISC apps. Version is mandatory. In case it is
        // false (undefined, null, empty string), simply return. No feedback will be displayed.
        if (!this.iscApp || !this.version) {
            return
        }

        const sanitizedSemver = this.versionService.sanitizeSemver(this.version)
        if (sanitizedSemver) {
            this.version = sanitizedSemver
            this._subscriptions.add(
                this.versionService
                    .getCurrentData()
                    .pipe(
                        map((data) => {
                            return this.versionService.getSoftwareVersionFeedback(
                                sanitizedSemver,
                                this.daemonName,
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
     * Getter of the friendly display name for the daemon.
     */
    get daemonDisplayName() {
        return this._daemonDisplayName
    }

    /**
     * Returns true if the app is a Kea, BIND9 or Stork agent.
     */
    get iscApp(): boolean {
        return isKeaDaemon(this.daemonName) || ['named', 'stork'].includes(this.daemonName)
    }

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
            summary: `${this.daemonDisplayName} ${this.version}`,
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
