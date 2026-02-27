import { Injectable, inject } from '@angular/core'
import { BehaviorSubject } from 'rxjs'

import { AuthService } from './auth.service'
import { SettingsService } from './backend/api/api'

@Injectable({
    providedIn: 'root',
})
export class SettingService {
    private auth = inject(AuthService)
    private settingsApi = inject(SettingsService)

    private settingsBS = new BehaviorSubject({})

    constructor() {
        // Only get the settings when the user is logged in.
        this.auth.currentUser$.subscribe(() => {
            if (this.auth.currentUserValue) {
                this.settingsApi.getSettings().subscribe(
                    (data) => {
                        this.settingsBS.next(data)
                    },
                    (err) => {
                        console.info('Problem getting settings', err)
                    }
                )
            }
        })
    }

    /** Returns the server settings as observable. */
    getSettings() {
        return this.settingsBS.asObservable()
    }
}
