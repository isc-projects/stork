import { Component, EventEmitter, Input, OnInit, Output } from '@angular/core'
import {
    DHCPService,
    UpdateKeaDaemonsGlobalParametersBeginResponse,
    UpdateKeaDaemonsGlobalParametersSubmitRequest,
} from '../backend'
import { lastValueFrom } from 'rxjs'
import { getErrorMessage } from '../utils'
import { MessageService } from 'primeng/api'
import {
    KeaGlobalConfigurationForm,
    KeaGlobalParametersForm,
    SubnetSetFormService,
} from '../forms/subnet-set-form.service'
import { FormGroup } from '@angular/forms'

/**
 * A component providing a form for editing global Kea parameters.
 */
@Component({
    selector: 'app-kea-global-configuration-form',
    templateUrl: './kea-global-configuration-form.component.html',
    styleUrl: './kea-global-configuration-form.component.sass',
})
export class KeaGlobalConfigurationFormComponent implements OnInit {
    /**
     * Daemon ID for which configuration should be updated.
     */
    @Input() daemonId: number = 0

    /**
     * An event emitter notifying that the form has been submitted.
     */
    @Output() formSubmit: EventEmitter<void> = new EventEmitter()

    /**
     * An event emitter notifying that form editing has been canceled.
     */
    @Output() formCancel: EventEmitter<void> = new EventEmitter()

    /**
     * Response received from the server upon beginning configuration
     * transaction.
     */
    response: UpdateKeaDaemonsGlobalParametersBeginResponse

    /**
     * A boolean flag set to true when the configuration transaction
     * has been started or an error occurred.
     */
    loaded = false

    /**
     * An error to begin the transaction returned by the server.
     */
    initError: string = null

    /**
     * Form group holding Kea configuration data.
     */
    formGroup: FormGroup<KeaGlobalConfigurationForm>

    /**
     * Constructor.
     *
     * @param dhcpApi a service providing an API to the server.
     * @param messageService a service for displaying error messages to the user.
     * @param subnetSetFormService a service for converting configuration data.
     */
    constructor(
        public dhcpApi: DHCPService,
        public messageService: MessageService,
        public subnetSetFormService: SubnetSetFormService
    ) {}

    /**
     * A component lifecycle hook invoked when the component is initialized.
     */
    ngOnInit(): void {
        this.updateKeaGlobalParametersBegin()
    }

    /**
     * A function called when user clicks the submit button.
     */
    onSubmit(): void {
        const request: UpdateKeaDaemonsGlobalParametersSubmitRequest = {
            configs: this.subnetSetFormService
                .convertFormToKeaGlobalParameters(
                    this.formGroup.get('parameters') as FormGroup<KeaGlobalParametersForm>
                )
                .map((params) => {
                    return {
                        daemonId: this.daemonId,
                        daemonName: this.response.configs.find((config) => config.daemonId === this.daemonId)
                            ?.daemonName,
                        partialConfig: params,
                    }
                }),
        }
        lastValueFrom(this.dhcpApi.updateKeaGlobalParametersSubmit(this.response?.id, request))
            .then(() => {
                this.messageService.add({
                    severity: 'success',
                    summary: 'Kea configuration successfully updated',
                })
                this.formSubmit.emit()
            })
            .catch((err) => {
                const msg = getErrorMessage(err)
                this.messageService.add({
                    severity: 'error',
                    summary: 'Cannot commit configuration updates',
                    detail: 'The transaction updating configuration failed: ' + msg,
                    life: 10000,
                })
            })
    }

    /**
     * A function called when user clicks the cancel button.
     */
    onCancel(): void {
        if (this.response?.id) {
            lastValueFrom(this.dhcpApi.updateKeaGlobalParametersDelete(this.response?.id)).catch((err) => {
                const msg = getErrorMessage(err)
                this.messageService.add({
                    severity: 'error',
                    summary: 'Failed to delete configuration transaction',
                    detail: 'Failed to delete configuration transaction: ' + msg,
                    life: 10000,
                })
            })
        }
        this.formCancel.emit()
    }

    /**
     * A function called when user clicks the retry button after failure to begin
     * a new transaction.
     */
    onRetry(): void {
        this.updateKeaGlobalParametersBegin()
    }

    /**
     * Sends a request to the server to begin a new transaction for updating
     * Kea global parameters.
     */
    private updateKeaGlobalParametersBegin(): void {
        this.loaded = false
        lastValueFrom(
            this.dhcpApi.updateKeaGlobalParametersBegin({
                daemonIds: [this.daemonId],
            })
        )
            .then((data: UpdateKeaDaemonsGlobalParametersBeginResponse) => {
                this.response = data
                this.formGroup = this.subnetSetFormService.convertKeaGlobalConfigurationToForm(this.response.configs)
                this.initError = null
            })
            .catch((err) => {
                const msg = getErrorMessage(err)
                this.messageService.add({
                    severity: 'error',
                    summary: 'Cannot create new transaction',
                    detail: `Failed to create transaction for updating global Kea parameters: ` + msg,
                    life: 10000,
                })
                this.initError = msg
            })
            .finally(() => {
                this.loaded = true
            })
    }
}
