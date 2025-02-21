import { Component, OnInit } from '@angular/core'
import { FormControl, FormGroup, FormBuilder, Validators } from '@angular/forms'

import { MessageService } from 'primeng/api'

import { SettingsService } from '../backend'
import { getErrorMessage } from '../utils'

/**
 * An interface specifying the form controls for the server settings.
 */
interface SettingsForm {
    appsStatePullerInterval: FormControl<number>
    bind9StatsPullerInterval: FormControl<number>
    keaHostsPullerInterval: FormControl<number>
    keaStatsPullerInterval: FormControl<number>
    keaStatusPullerInterval: FormControl<number>
    grafanaUrl: FormControl<string>
    enableMachineRegistration: FormControl<boolean>
    enableOnlineSoftwareVersions: FormControl<boolean>
}

/**
 * An interface holding information required to render a single
 * form control.
 */
interface SettingsItem {
    title: string
    formControlName: string
    help: string
}

/**
 * A component providing a form to specify server settings.
 */
@Component({
    selector: 'app-settings-page',
    templateUrl: './settings-page.component.html',
    styleUrls: ['./settings-page.component.sass'],
})
export class SettingsPageComponent implements OnInit {
    /**
     * A path specified in the breadcrumbs.
     */
    breadcrumbs = [{ label: 'Configuration' }, { label: 'Settings' }]

    /**
     * A list of interval settings to specify in the form.
     *
     * A numeric input form control is created for each setting in this
     * array. The value is validated with the required and min validators.
     * The expected value must be non-negative.
     */
    intervalSettings: SettingsItem[] = [
        {
            title: 'Apps State Puller Interval',
            formControlName: 'appsStatePullerInterval',
            help: 'This puller refreshes the state of the apps running on the monitored machines.',
        },
        {
            title: 'BIND 9 Statistics Puller Interval',
            formControlName: 'bind9StatsPullerInterval',
            help: 'This puller refreshes statistics from the BIND 9 servers.',
        },
        {
            title: 'Kea Hosts Puller Interval',
            formControlName: 'keaHostsPullerInterval',
            help: 'This puller fetches the host reservations from the Kea servers.',
        },
        {
            title: 'Kea Statistics Puller Interval',
            formControlName: 'keaStatsPullerInterval',
            help: 'This puller refreshes statistics from the Kea servers',
        },
        {
            title: 'Kea Status Puller Interval',
            formControlName: 'keaStatusPullerInterval',
            help: 'This puller fetches the high-availability status from the Kea servers.',
        },
    ]

    /**
     * A list of URL settings to specify in the form.
     *
     * A URL input form control is created for each setting in this array.
     */
    urlSettings: SettingsItem[] = [
        {
            title: 'URL to Grafana',
            formControlName: 'grafanaUrl',
            help: 'Specifies a Grafana instance URL. It is optional.',
        },
    ]

    /**
     * A form holding the settings.
     */
    settingsForm: FormGroup<SettingsForm>

    /**
     * A union defining form state.
     *
     * It controls what is rendered.
     */
    formState: 'loading' | 'fail' | 'success'

    /**
     * Constructor.
     *
     * @param fb form builder instance.
     * @param settingsApi a service for communicating with the server.
     * @param msgSrv a message service.
     */
    constructor(
        private fb: FormBuilder,
        private settingsApi: SettingsService,
        private msgSrv: MessageService
    ) {
        this.settingsForm = this.fb.group({
            appsStatePullerInterval: [0, [Validators.required, Validators.min(0)]],
            bind9StatsPullerInterval: [0, [Validators.required, Validators.min(0)]],
            keaHostsPullerInterval: [0, [Validators.required, Validators.min(0)]],
            keaStatsPullerInterval: [0, [Validators.required, Validators.min(0)]],
            keaStatusPullerInterval: [0, [Validators.required, Validators.min(0)]],
            grafanaUrl: [''],
            enableMachineRegistration: [false],
            enableOnlineSoftwareVersions: [false],
        })
    }

    /**
     * Gathers the current settings from the server and initializes them
     * in the form.
     */
    private getSettings(): void {
        this.formState = 'loading'
        this.settingsApi.getSettings().subscribe({
            next: (data) => {
                this.settingsForm.patchValue(data)
                this.formState = 'success'
            },
            error: (err) => {
                this.formState = 'fail'
                const msg = getErrorMessage(err)
                this.msgSrv.add({
                    severity: 'error',
                    summary: 'Cannot get settings',
                    detail: 'Error getting settings: ' + msg,
                    life: 10000,
                })
            },
        })
    }

    /**
     * A component lifecycle hook invoked upon the component initialization.
     *
     * It gathers the current settings from the server and initializes them
     * in the form.
     */
    ngOnInit() {
        this.getSettings()
    }

    /**
     * Retries gathering the settings after failure.
     */
    retry(): void {
        this.getSettings()
    }

    /**
     * Saves the current values of the settings in the backend.
     */
    saveSettings(): void {
        if (!this.settingsForm.valid) {
            return
        }
        const settings = this.settingsForm.getRawValue()

        this.settingsApi.updateSettings(settings).subscribe({
            next: (/* data */) => {
                this.msgSrv.add({
                    severity: 'success',
                    summary: 'Settings updated',
                    detail: 'Updating settings succeeded.',
                })
            },
            error: (err) => {
                const msg = getErrorMessage(err)
                this.msgSrv.add({
                    severity: 'error',
                    summary: 'Cannot save settings',
                    detail: 'Error saving settings: ' + msg,
                    life: 10000,
                })
            },
        })
    }

    /**
     * Indicates if the given form field has assigned error with the
     * specific name.
     *
     * @param name control name.
     * @param errType error type name.
     * @returns A boolean value indicating if the control has the error.
     */
    hasError(name: string, errType: string): boolean {
        return !!this.settingsForm.get(name)?.hasError(errType)
    }
}
