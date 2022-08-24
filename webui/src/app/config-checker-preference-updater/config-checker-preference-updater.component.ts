import { Component, Input, OnDestroy, OnInit } from '@angular/core';
import { MessageService } from 'primeng/api';
import { of, Subject, Subscription } from 'rxjs';
import { buffer, catchError, debounce, debounceTime, map, share } from 'rxjs/operators';
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

    checkers: ConfigChecker[] = []
    preferences = new Subject<ConfigCheckerPreference>()

    constructor(private servicesApi: ServicesService, private messageService: MessageService) {
        
    }

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
            this.checkers = data
        }))

        const preferenceShared = this.preferences.pipe(share())
        this.subscriptions.add(preferenceShared.pipe(
            // It buffers the preferences until no change is provided at a
            // particular time.
            buffer(
                preferenceShared.pipe(debounceTime(5000))
            ),
            // Reduces the preferences to keep only last change for each checker.
            map(preferences => {
                const reducedPreferences = preferences.reduceRight((acc, preference) => {
                    for (let existingPreference of acc) {
                        if (existingPreference.name === preference.name) {
                            return acc
                        }
                    }
                    acc.push(preference)
                    return acc
                }, [] as ConfigCheckerPreference[])

                return {
                    items: reducedPreferences,
                    total: reducedPreferences.length
                } as ConfigCheckerPreferences
            })
        )
        // Send changes to API
        .subscribe(preferences => {
            (this.daemonID == null
                ? this.servicesApi.putGlobalConfigCheckerPreferences(preferences)
                : this.servicesApi.putDaemonConfigCheckerPreferences(this.daemonID, preferences))
                .toPromise()
                .then(data => {
                    this.checkers = data.items
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
}
