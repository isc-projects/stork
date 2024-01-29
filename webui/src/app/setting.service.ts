import { Injectable } from '@angular/core'
import { BehaviorSubject } from 'rxjs'

import { AuthService } from './auth.service'
import { SettingsService } from './backend/api/api'

@Injectable({
    providedIn: 'root',
})
export class SettingService {
    private settingsBS = new BehaviorSubject({})

    constructor(
        private auth: AuthService,
        private settingsApi: SettingsService
    ) {
        // Only get the settings when the user is logged in.
        this.auth.currentUser.subscribe(() => {
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
