import { Component, EventEmitter, Input, OnDestroy, OnInit, Output } from '@angular/core'
import { DHCPService, SharedNetwork, UpdateSharedNetworkBeginResponse } from '../backend'
import { GenericFormService } from '../forms/generic-form.service'
import { MessageService } from 'primeng/api'
import { DhcpOptionSetFormService } from '../forms/dhcp-option-set-form.service'
import { getErrorMessage, getSeverityByIndex } from '../utils'
import { createDefaultDhcpOptionFormGroup } from '../forms/dhcp-option-form'
import { FormGroup, UntypedFormArray, UntypedFormControl } from '@angular/forms'
import { SharedNetworkFormState } from '../forms/shared-network-form'
import { KeaSubnetParametersForm, SubnetSetFormService } from '../forms/subnet-set-form.service'
import { lastValueFrom } from 'rxjs'

@Component({
    selector: 'app-shared-network-form',
    templateUrl: './shared-network-form.component.html',
    styleUrl: './shared-network-form.component.sass',
})
export class SharedNetworkFormComponent implements OnInit, OnDestroy {
    /**
     * Form state instance.
     *
     * The instance is shared between the parent and this component.
     * Holding the instance in the parent component allows for restoring
     * the form (after edits) after the component has been (temporarily)
     * destroyed.
     */
    @Input() state: SharedNetworkFormState = null

    /**
     * Shared network identifier.
     *
     * It should be set in cases when the form is used to update an existing
     * shared network. It is not set when the form is used to create new
     * shared network.
     */
    @Input() sharedNetworkId: number = 0

    /**
     * An event emitter notifying that the component is destroyed.
     *
     * A parent component receiving this event can remember the current
     * form state.
     */
    @Output() formDestroy = new EventEmitter<SharedNetworkFormState>()

    /**
     * An event emitter notifying that the form has been submitted.
     */
    @Output() formSubmit = new EventEmitter<SharedNetworkFormState>()

    /**
     * An event emitter notifying that form editing has been canceled.
     */
    @Output() formCancel = new EventEmitter<number>()

    /**
     * Constructor.
     *
     * @param dhcpApi a service providing an API to the server.
     * @param genericFormService a generic form conversion service.
     * @param messageService a service for displaying error messages to the user.
     * @param optionsFormService a service for converting DHCP options.
     * @param sharedNetworkSetFormService a service for converting shared network data.
     */
    constructor(
        public dhcpApi: DHCPService,
        public genericFormService: GenericFormService,
        public messageService: MessageService,
        public optionsFormService: DhcpOptionSetFormService,
        public subnetSetFormService: SubnetSetFormService
    ) {}

    /**
     * A component lifecycle hook invoked when the component is initialized.
     *
     * It creates a form state if the state is not provided by the parent.
     * It holds all the necessary information about the form, including user
     * selections, data received from the server before the edits etc. It
     * can be sometimes cached by the parent component and used to re-create
     * the form.
     */
    ngOnInit() {
        // If the state was cached by the parent there is no need to create it.
        // It happens when a user switches between the subnet tabs and the
        // component is temporarily destroyed.
        if (!this.state) {
            this.state = new SharedNetworkFormState()
        }

        if (this.state.loaded) {
            return
        }

        // We currently only support updating a shared network In this case the
        // id must be provided.
        if (this.sharedNetworkId) {
            // Send POST to /shared-betworks/{id}/transaction/new.
            this.updateSharedNetworkBegin()
        }
    }

    /**
     * Component lifecycle hook invoked when the component is destroyed.
     *
     * It emits an event to the parent to cause the parent to preserve
     * the form instance. This instance can be later used to continue making
     * the edits when the component is re-created. It also sets the
     * preserved flag to indicate that the form was recovered, and thus
     * skip initialization in the next ngOnInit function invocation.
     */
    ngOnDestroy(): void {
        this.state.markLoaded()
        this.formDestroy.emit(this.state)
    }

    /**
     * Initializes the shared network form state using the data received from the server.
     *
     * @param response response received from the server holding the shared network data.
     */
    initializeState(response: UpdateSharedNetworkBeginResponse) {
        this.state.initStateFromServerResponse(response)

        // If we update an existing shared network the shared network information should be
        // in the response.
        if (this.sharedNetworkId && 'sharedNetwork' in response && response.sharedNetwork) {
            // Initialize the shared network form controls.
            this.state.group = this.subnetSetFormService.convertSharedNetworkToForm(
                response.sharedNetwork,
                this.state.existingSharedNetworkNames
            )
        }
        // After the form has been initialized we need to filter out the daemons
        // that can be selected by a user for our shared network.
        this.handleDaemonsChange()

        // Hide the spinner and show the form.
        this.state.markLoaded()
    }

    /**
     * Sends a request to the server to begin a new transaction for updating
     * a shared network.
     */
    private updateSharedNetworkBegin(): void {
        lastValueFrom(this.dhcpApi.updateSharedNetworkBegin(this.sharedNetworkId))
            .then((data) => {
                this.state.savedSharedNetworkBeginData = data
                this.initializeState(data)
            })
            .catch((err) => {
                const msg = getErrorMessage(err)
                this.messageService.add({
                    severity: 'error',
                    summary: 'Cannot create new transaction',
                    detail: `Failed to create transaction for updating shared network ${this.sharedNetworkId}: ` + msg,
                    life: 10000,
                })
                this.state.setInitError(msg)
            })
    }

    /**
     * Returns severity of a tag associating a form control with a server.
     *
     * @param index server index in the {@link servers} array.
     * @returns `success` for the first server, `warning` for the second
     * server, `danger` for the third server, and 'info' for any other
     * server.
     */
    getServerTagSeverity(index: number): string {
        return getSeverityByIndex(index)
    }

    /**
     * A function called when a user clicked to add a new option form.
     *
     * It creates a new default form group for the option.
     *
     * @param index server index in the {@link servers} array.
     */
    onOptionAdd(index: number): void {
        this.getOptionsData(index).push(createDefaultDhcpOptionFormGroup(this.state.ipType))
    }

    /**
     * Adjusts the form state based on the selected daemons.
     *
     * Servers selection affecs the form contents. When none are selected, the
     * default form should be displayed. Otherwise, we should track the configuration
     * values for the respective servers. Removing a server also results in the
     * form update because the parts of the form related to that server must be
     * removed.
     *
     * @param toggledDaemonId optional id of the removed daemon in the controls.
     */
    private handleDaemonsChange(toggledDaemonId?: number): void {
        this.subnetSetFormService.adjustFormForSelectedDaemons(
            this.state.group,
            this.state.getFilteredDaemonIndex(toggledDaemonId),
            this.state.servers.length
        )

        // Selecting new daemons may have a large impact on the data already
        // inserted to the form. Update the form state accordingly and see
        // if it is breaking change.
        const selectedDaemons = this.state.group.get('selectedDaemons').value ?? []
        if (this.state.updateFormForSelectedDaemons(selectedDaemons)) {
            // The breaking change puts us at risk of having irrelevant form contents.
            this.resetOptionsArray()
            this.resetParametersArray()
            return
        }
        // If the number of selected daemons has changed we must update selected servers list.
        this.state.updateServers(selectedDaemons)
    }

    /**
     * A callback invoked when selected DHCP servers have changed.
     *
     * Adjusts the form state based on the selected daemons.
     */
    onDaemonsChange(event): void {
        this.handleDaemonsChange(event.itemValue)
    }

    /**
     * A function called when user clicks the button to revert shared network changes.
     */
    onRevert(): void {
        this.initializeState(this.state.savedSharedNetworkBeginData)
    }

    /**
     * A function called when user clicks the cancel button.
     */
    onCancel(): void {
        this.formCancel.emit(this.sharedNetworkId)
    }

    /**
     * A function called when user clicks the retry button after failure to begin
     * a new transaction.
     */
    onRetry(): void {
        if (this.sharedNetworkId) {
            this.updateSharedNetworkBegin()
        }
    }

    /**
     * A function called when user clicks the submit button.
     */
    onSubmit(): void {
        let sharedNetwork: SharedNetwork

        try {
            sharedNetwork = this.subnetSetFormService.convertFormToSharedNetwork(this.state.ipType, this.state.group)
            sharedNetwork.subnets = this.state.savedSharedNetworkBeginData?.sharedNetwork?.subnets
        } catch (err) {
            this.messageService.add({
                severity: 'error',
                summary: 'Cannot commit the sharde network',
                detail: 'Processing the shared network form failed: ' + err,
                life: 10000,
            })
            return
        }

        if (this.sharedNetworkId) {
            // Updating an existing shared network.
            sharedNetwork.id = this.sharedNetworkId
            lastValueFrom(
                this.dhcpApi.updateSharedNetworkSubmit(this.sharedNetworkId, this.state.transactionId, sharedNetwork)
            )
                .then(() => {
                    this.messageService.add({
                        severity: 'success',
                        summary: 'Shared network successfully updated',
                    })
                    // Notify the parent component about successful submission.
                    this.formSubmit.emit(this.state)
                })
                .catch((err) => {
                    let msg = err.statusText
                    if (err.error && err.error.message) {
                        msg = err.error.message
                    }
                    this.messageService.add({
                        severity: 'error',
                        summary: 'Cannot commit shared network updates',
                        detail: 'The transaction updating the shared network failed: ' + msg,
                        life: 10000,
                    })
                })
            return
        }
    }

    /**
     * Resets the part of the form comprising assorted DHCP parameters.
     *
     * It removes all existing controls and re-creates the default one.
     */
    private resetParametersArray() {
        let parameters = this.state.group.get('parameters') as FormGroup<KeaSubnetParametersForm>
        if (!parameters) {
            return
        }

        for (let key of Object.keys(parameters.controls)) {
            let unlocked = parameters.get(key).get('unlocked') as UntypedFormControl
            unlocked?.setValue(false)
            let values = parameters.get(key).get('values') as UntypedFormArray
            if (values?.length > 0) {
                values.controls.splice(0)
            }
        }
        this.state.group.setControl(
            'parameters',
            this.subnetSetFormService.createDefaultKeaSubnetParametersForm(this.state.ipType)
        )
    }

    /**
     * Resets the part of the form comprising DHCP options.
     *
     * It removes all existing option sets and re-creates the default one.
     */
    private resetOptionsArray() {
        this.state.group.get('options.unlocked').setValue(false)
        this.getOptionsData().clear()
        this.getOptionsData().push(new UntypedFormArray([]))
    }

    /**
     * Returns options data for all servers or for a specified server.
     *
     * @param index optional index of the server.
     * @returns An array of options data for all servers or for a single server.
     */
    private getOptionsData(index?: number): UntypedFormArray {
        return index === undefined
            ? (this.state.group.get('options.data') as UntypedFormArray)
            : (this.getOptionsData().at(index) as UntypedFormArray)
    }
}
