import { Component, Input, OnDestroy, OnInit } from '@angular/core';
import { MessageService } from 'primeng/api';
import { of, Subject, Subscription } from 'rxjs';
import { buffer, catchError, debounce, debounceTime, filter, map, share } from 'rxjs/operators';
import { ConfigChecker, ConfigCheckerPreference, ConfigCheckerPreferences, ServicesService } from '../backend';
import { getErrorMessage } from '../utils';

@Component({
    selector: 'app-config-checker-preference-updater',
    templateUrl: './config-checker-preference-updater.component.html',
    styleUrls: ['./config-checker-preference-updater.component.sass']
})
export class ConfigCheckerPreferenceUpdaterComponent implements OnInit, OnDestroy {
    private subscriptions = new Subscription()

    @Input() daemonID: number | null = null
    @Input() waitMilliseconds: number = 5000

    checkers: ConfigChecker[] = []
    originalCheckers: ConfigChecker[] = []
    preferences = new Subject<ConfigCheckerPreference>()

    constructor(private servicesApi: ServicesService, private messageService: MessageService) { }

    ngOnInit(): void {
        this.subscriptions.add((this.daemonID == null
            ? this.servicesApi.getGlobalConfigCheckers()
            : this.servicesApi.getDaemonConfigCheckers(this.daemonID)
        ).pipe(
            map(data => data.items),
            catchError(err => {
                this.messageService.add({
                    severity: "error",
                    summary: "Cannot get configuration checkers",
                    detail: getErrorMessage(err)
                })
                return of([])
            })
        ).subscribe((data) => {
            this._setCheckers(data)
        }))

        const preferenceShared = this.preferences.pipe(share())
        this.subscriptions.add(preferenceShared.pipe(
            // It buffers the preferences until no change is provided at a
            // particular time.
            buffer(
                preferenceShared.pipe(debounceTime(this.waitMilliseconds))
            ),
            // Reduces the preferences to keep only last change for each checker.
            map(preferences => {
                let reducedPreferences = preferences.reduceRight((acc, preference) => {
                    for (let existingPreference of acc) {
                        if (existingPreference.name === preference.name) {
                            return acc
                        }
                    }
                    acc.push(preference)
                    return acc
                }, [] as ConfigCheckerPreference[])
                // Filter out not changed preferences.
                .filter(preference => {
                    for (let checker of this.originalCheckers) {
                        if (preference.name == checker.name) {
                            return preference.state !== checker.state
                        }
                    }
                    return true
                })

                return {
                    items: reducedPreferences,
                    total: reducedPreferences.length
                } as ConfigCheckerPreferences
            }),
            filter(preferences => preferences.total !== 0)
        )
        // Send changes to API
        .subscribe(preferences => {
            (this.daemonID == null
                ? this.servicesApi.putGlobalConfigCheckerPreferences(preferences)
                : this.servicesApi.putDaemonConfigCheckerPreferences(this.daemonID, preferences))
                .toPromise()
                .then(data => {
                    this._setCheckers(data.items)
                    this.messageService.add({
                        severity: "success",
                        summary: "Configuration checker preferences updated",
                        detail: "Updating succeeded"
                    })
                })
                .catch(err => {
                    this.messageService.add({
                        severity: "error",
                        summary: "Cannot update configuration checker preferences",
                        detail: getErrorMessage(err)
                    })
                })
        }))
    }

    ngOnDestroy(): void {
        this.subscriptions.unsubscribe()
    }

    onChangePreference(preference: ConfigCheckerPreference) {
        this.preferences.next(preference)
    }

    private _setCheckers(checkers: ConfigChecker[]) {
        this.checkers = checkers
        // Make a deep copy
        this.originalCheckers = JSON.parse(JSON.stringify(checkers))
    }
}
