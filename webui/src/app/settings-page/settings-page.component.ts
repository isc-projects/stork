import { Component, OnInit } from '@angular/core'
import { UntypedFormBuilder, UntypedFormGroup, Validators } from '@angular/forms'

import { MessageService } from 'primeng/api'

import { SettingsService } from '../backend/api/api'
import { getErrorMessage } from '../utils'

@Component({
    selector: 'app-settings-page',
    templateUrl: './settings-page.component.html',
    styleUrls: ['./settings-page.component.sass'],
})
export class SettingsPageComponent implements OnInit {
    breadcrumbs = [{ label: 'Configuration' }, { label: 'Settings' }]

    public settingsForm: UntypedFormGroup

    constructor(
        private fb: UntypedFormBuilder,
        private settingsApi: SettingsService,
        private msgSrv: MessageService
    ) {
        this.settingsForm = this.fb.group({
            bind9_stats_puller_interval: ['', [Validators.required, Validators.min(0)]],
            grafana_url: [''],
            kea_hosts_puller_interval: ['', [Validators.required, Validators.min(0)]],
            kea_stats_puller_interval: ['', [Validators.required, Validators.min(0)]],
            kea_status_puller_interval: ['', [Validators.required, Validators.min(0)]],
            prometheus_url: [''],
        })
    }

    ngOnInit() {
        this.settingsApi.getSettings().subscribe(
            (data) => {
                const numericSettings = [
                    'bind9_stats_puller_interval',
                    'kea_hosts_puller_interval',
                    'kea_stats_puller_interval',
                    'kea_status_puller_interval',
                ]
                const stringSettings = ['grafana_url', 'prometheus_url']

                for (const s of numericSettings) {
                    if (data[s] === undefined) {
                        data[s] = 0
                    }
                }
                for (const s of stringSettings) {
                    if (data[s] === undefined) {
                        data[s] = ''
                    }
                }

                this.settingsForm.patchValue(data)
            },
            (err) => {
                const msg = getErrorMessage(err)
                this.msgSrv.add({
                    severity: 'error',
                    summary: 'Cannot get settings',
                    detail: 'Error getting settings: ' + msg,
                    life: 10000,
                })
            }
        )
    }

    /**
     * Saves the current values of the settings on the backend side.
     */
    saveSettings() {
        if (!this.settingsForm.valid) {
            return
        }
        const settings = this.settingsForm.getRawValue()

        this.settingsApi.updateSettings(settings).subscribe(
            (data) => {
                this.msgSrv.add({
                    severity: 'success',
                    summary: 'Settings updated',
                    detail: 'Updating settings succeeded.',
                })
            },
            (err) => {
                const msg = getErrorMessage(err)
                this.msgSrv.add({
                    severity: 'error',
                    summary: 'Cannot get settings',
                    detail: 'Error getting settings: ' + msg,
                    life: 10000,
                })
            }
        )
    }

    /**
     * Indicates if the given form field has assigned error with the
     * specific name.
     */
    hasError(name: string, errType: string) {
        const setting = this.settingsForm.get(name)
        if (setting.errors && setting.errors[errType]) {
            return true
        }
        return false
    }
}
