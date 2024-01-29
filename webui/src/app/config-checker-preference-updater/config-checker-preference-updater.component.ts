import { Component, Input, OnDestroy, OnInit } from '@angular/core'
import { MessageService } from 'primeng/api'
import { of, Subscription } from 'rxjs'
import { catchError, map } from 'rxjs/operators'
import { ConfigChecker, ConfigCheckerPreference, ConfigCheckerPreferences, ServicesService } from '../backend'
import { getErrorMessage } from '../utils'

/**
 * Smart component to display the config checker metadata and update the config
 * checker preferences.
 * It receives the stream of checker preference changes from the presentational
 * sub-component and processes it. It buffers the changes until the user stops
 * providing them. Next, it analyzes the buffer to detect if any actual change
 * occurred. If yes, it shares the data with the API. The buffer is processed
 * when a specific amount of time passed from the last change.
 */
@Component({
    selector: 'app-config-checker-preference-updater',
    templateUrl: './config-checker-preference-updater.component.html',
    styleUrls: ['./config-checker-preference-updater.component.sass'],
})
export class ConfigCheckerPreferenceUpdaterComponent implements OnInit, OnDestroy {
    /**
     * List of subscriptions created by the component.
     */
    private subscriptions = new Subscription()

    /**
     * The component fetches config checkers related to the provided daemon ID
     * or global list of checkers if the value is null.
     */
    @Input() daemonID: number | null = null

    /**
     * If true, displays only the checker name and state.
     */
    @Input() minimal: boolean = false

    /**
     * List of the checkers passed to sub-component. It will be changing in place.
     */
    checkers: ConfigChecker[] = null

    /**
     * Indicate that the data aren't ready yet.
     */
    loading: boolean = true

    /**
     * Constructs the component.
     * @param servicesApi Used to exchange data with API.
     * @param messageService Used to generate success and error messages
     */
    constructor(
        private servicesApi: ServicesService,
        private messageService: MessageService
    ) {}

    /**
     * Creates two subscriptions.
     * The first fetches the initial data from API.
     * The second processes the config checker preference changes and sends them
     * to API.
     */
    ngOnInit(): void {
        // Fetches config checker metadata from API.
        this.subscriptions.add(
            (this.daemonID == null
                ? this.servicesApi.getGlobalConfigCheckers()
                : this.servicesApi.getDaemonConfigCheckers(this.daemonID)
            )
                .pipe(
                    // Extracts the content.
                    map((data) => data.items),
                    // Handles any connection error. Generates an error message and
                    // returns an empty list.
                    catchError((err) => {
                        this.messageService.add({
                            severity: 'error',
                            summary: 'Cannot get configuration checkers',
                            detail: getErrorMessage(err),
                        })
                        return of([])
                    })
                    // Receives the data.
                )
                .subscribe((data) => {
                    // Sets the checker metadata to the component properties.
                    this.checkers = data
                    this.loading = false
                })
        )
    }

    /**
     * Unsubscribes subscribers.
     */
    ngOnDestroy(): void {
        this.subscriptions.unsubscribe()
    }

    /**
     * Callback called by the sub-component when user changed and submit
     * the checker states.
     * @param preference Config checker preferences provided by a user.
     */
    onChangePreferences(preferenceList: ConfigCheckerPreference[]) {
        this.loading = true

        const preferences: ConfigCheckerPreferences = {
            items: preferenceList,
            total: preferenceList.length,
        }

        const putRequest =
            this.daemonID == null
                ? this.servicesApi.putGlobalConfigCheckerPreferences(preferences)
                : this.servicesApi.putDaemonConfigCheckerPreferences(this.daemonID, preferences)

        putRequest
            .toPromise()
            .then((data) => {
                this.checkers = data.items
                this.messageService.add({
                    severity: 'success',
                    summary: 'Configuration checker preferences updated',
                    detail: 'Updating succeeded',
                })
            })
            .catch((err) => {
                // Restore the checkers
                this.messageService.add({
                    severity: 'error',
                    summary: 'Cannot update configuration checker preferences',
                    detail: getErrorMessage(err),
                })
            })
            .then(() => {
                this.loading = false
            })
    }
}
