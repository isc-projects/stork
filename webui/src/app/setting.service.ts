import { Injectable } from '@angular/core'
import { BehaviorSubject } from 'rxjs'

import { SettingsService } from './backend/api/api'

@Injectable({
    providedIn: 'root',
})
export class SettingService {
    private settingsBS = new BehaviorSubject({})
    private settings = {}

    constructor(private settingsApi: SettingsService) {
        this.settingsApi.getSettings().subscribe(
            data => {
                this.settings = data
                this.settingsBS.next(data)
            },
            err => {
                console.info('problem with getting settings', err)
            }
        )
    }

    getSettings() {
        return this.settingsBS.asObservable()
    }
}
