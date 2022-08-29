import { Component, Input, OnDestroy, OnInit } from '@angular/core'
import { MessageService } from 'primeng/api'
import { of, Subject, Subscription } from 'rxjs'
import { buffer, catchError, debounce, debounceTime, filter, map, share } from 'rxjs/operators'
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
     * The preferences changes aren't immediately pushed to a server to avoid
     * generating too many requests. Instead of it, the component waits a given
     * number of milliseconds after the last change.
     */
    @Input() waitMilliseconds: number = 5000

    /**
     * If true, displays only the checker name and state.
     */
    @Input() minimal: boolean = false

    /**
     * List of the checkers passed to sub-component. It will be changing in place.
     */
    checkers: ConfigChecker[] = null
    /**
     * The deep copy of the list passed to the sub-component.
     * It's used to detect if any modifications were provided by a user.
     */
    originalCheckers: ConfigChecker[] = []

    /**
     * Collects the preference changes received from sub-component.
     */
    preferences = new Subject<ConfigCheckerPreference>()

    /**
     * Constructs the component.
     * @param servicesApi Used to exchange data with API.
     * @param messageService Used to generate success and error messages
     */
    constructor(private servicesApi: ServicesService, private messageService: MessageService) {}

    /**
     * Creates two subscriptions.
     * The first fetches the initial data from API.
     * The second processes the config checker preference changes and send them
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
                    this._setCheckers(data)
                })
        )

        // Create multicast observable.
        const preferenceShared = this.preferences.pipe(share())
        this.subscriptions.add(
            preferenceShared
                .pipe(
                    // It buffers the preferences until no change is provided at a
                    // particular time.
                    buffer(preferenceShared.pipe(debounceTime(this.waitMilliseconds))),
                    // Reduces the preferences to keep only last change for each checker.
                    map((preferences) => {
                        let reducedPreferences = preferences
                            .reduceRight((acc, preference) => {
                                for (let existingPreference of acc) {
                                    if (existingPreference.name === preference.name) {
                                        return acc
                                    }
                                }
                                acc.push(preference)
                                return acc
                            }, [] as ConfigCheckerPreference[])
                            // Filter out non-changed preferences.
                            .filter((preference) => {
                                for (let checker of this.originalCheckers) {
                                    if (preference.name == checker.name) {
                                        return preference.state !== checker.state
                                    }
                                }
                                return true
                            })

                        // Creates the preferences object.
                        return {
                            items: reducedPreferences,
                            total: reducedPreferences.length,
                        } as ConfigCheckerPreferences
                    }),
                    // Filter out empty objects.
                    filter((preferences) => preferences.total !== 0)
                )
                // Send changes to API
                .subscribe((preferences) => {
                    (this.daemonID == null
                        ? this.servicesApi.putGlobalConfigCheckerPreferences(preferences)
                        : this.servicesApi.putDaemonConfigCheckerPreferences(this.daemonID, preferences)
                    )
                        .toPromise()
                        .then((data) => {
                            this._setCheckers(data.items)
                            this.messageService.add({
                                severity: 'success',
                                summary: 'Configuration checker preferences updated',
                                detail: 'Updating succeeded',
                            })
                        })
                        .catch((err) => {
                            // Restore the checkers
                            this._setCheckers(this.originalCheckers)
                            this.messageService.add({
                                severity: 'error',
                                summary: 'Cannot update configuration checker preferences',
                                detail: getErrorMessage(err),
                            })
                        })
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
     * Callback called by the sub-component when user changed the checker state.
     * It passes the provided checker preference to the observable stream.
     * @param preference Config checker preference provided by a user.
     */
    onChangePreference(preference: ConfigCheckerPreference) {
        this.preferences.next(preference)
    }

    /**
     * Helper function that deep copies the checkers and assign the checkers
     * and copies to the properties.
     * @param checkers Config checkers received from API.
     */
    private _setCheckers(checkers: ConfigChecker[]) {
        this.checkers = checkers
        // Make a deep copy using the old-school way.
        this.originalCheckers = JSON.parse(JSON.stringify(checkers))
    }
}
