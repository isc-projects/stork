import { Component, OnInit } from '@angular/core'
import { FormBuilder, FormControl, FormGroup, NgForm, Validators } from '@angular/forms'

import { MessageService } from 'primeng/api'

import { SettingsService } from '../backend/api/api'

@Component({
    selector: 'app-settings-page',
    templateUrl: './settings-page.component.html',
    styleUrls: ['./settings-page.component.sass'],
})
export class SettingsPageComponent implements OnInit {
    public settingsForm: FormGroup

    constructor(private fb: FormBuilder, private settingsApi: SettingsService, private msgSrv: MessageService) {
        this.settingsForm = this.fb.group({
            bind9_stats_puller_interval: [''],
            kea_stats_puller_interval: [''],
            kea_hosts_puller_interval: [''],
            grafana_url: [''],
            prometheus_url: [''],
        })
    }

    ngOnInit() {
        this.settingsApi.getSettings().subscribe(
            data => {
                this.settingsForm.patchValue(data)
            },
            err => {
                let msg = err.statusText
                if (err.error && err.error.message) {
                    msg = err.error.message
                }
                this.msgSrv.add({
                    severity: 'error',
                    summary: 'Cannot get settings',
                    detail: 'Getting settings erred: ' + msg,
                    life: 10000,
                })
            }
        )
    }

    saveSettings() {
        const settings = this.settingsForm.getRawValue()

        this.settingsApi.updateSettings(settings).subscribe(
            data => {
                this.msgSrv.add({
                    severity: 'success',
                    summary: 'Settings updated',
                    detail: 'Updating settings succeeded.',
                })
            },
            err => {
                let msg = err.statusText
                if (err.error && err.error.message) {
                    msg = err.error.message
                }
                this.msgSrv.add({
                    severity: 'error',
                    summary: 'Cannot get settings',
                    detail: 'Getting settings erred: ' + msg,
                    life: 10000,
                })
            }
        )
    }
}
