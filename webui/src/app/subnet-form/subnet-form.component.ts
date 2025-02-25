import { Component, EventEmitter, Input, OnDestroy, OnInit, Output, QueryList, ViewChildren } from '@angular/core'
import { CreateSubnetBeginResponse, DHCPService, Subnet, UpdateSubnetBeginResponse } from '../backend'
import { getErrorMessage, getSeverityByIndex, getVersionRange } from '../utils'
import { MessageService } from 'primeng/api'
import { FormArray, FormGroup, UntypedFormArray, UntypedFormControl } from '@angular/forms'
import {
    AddressPoolForm,
    KeaSubnetParametersForm,
    PrefixPoolForm,
    SubnetSetFormService,
} from '../forms/subnet-set-form.service'
import { createDefaultDhcpOptionFormGroup } from '../forms/dhcp-option-form'
import { IPType } from '../iptype'
import { SubnetFormState } from '../forms/subnet-form'
import { GenericFormService } from '../forms/generic-form.service'
import { DhcpOptionSetFormService } from '../forms/dhcp-option-set-form.service'
import { AddressPoolFormComponent } from '../address-pool-form/address-pool-form.component'
import { SelectableDaemon } from '../forms/selectable-daemon'
import { PrefixPoolFormComponent } from '../prefix-pool-form/prefix-pool-form.component'
import { lastValueFrom } from 'rxjs'

/**
 * A component providing a form for editing and adding a subnet.
 */
@Component({
    selector: 'app-subnet-form',
    templateUrl: './subnet-form.component.html',
    styleUrls: ['./subnet-form.component.sass'],
})
export class SubnetFormComponent implements OnInit, OnDestroy {
    @ViewChildren(AddressPoolFormComponent) addressPoolComponents!: QueryList<AddressPoolFormComponent>

    @ViewChildren(PrefixPoolFormComponent) prefixPoolComponents!: QueryList<PrefixPoolFormComponent>

    /**
     * Form state instance.
     *
     * The instance is shared between the parent and this component.
     * Holding the instance in the parent component allows for restoring
     * the form (after edits) after the component has been (temporarily)
     * destroyed.
     */
    @Input() state: SubnetFormState = null

    /**
     * Subnet identifier.
     *
     * It should be set in cases when the form is used to update an existing
     * subnet. It is not set when the form is used to create new subnet.
     */
    @Input() subnetId: number = 0

    /**
     * An event emitter notifying that the component is destroyed.
     *
     * A parent component receiving this event can remember the current
     * form state.
     */
    @Output() formDestroy = new EventEmitter<SubnetFormState>()

    /**
     * An event emitter notifying that the form has been submitted.
     */
    @Output() formSubmit = new EventEmitter<SubnetFormState>()

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
     * @param subnetSetFormService a service for converting subnet data.
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
            this.state = new SubnetFormState()
        }

        // Check if the form has been already edited and preserved in the
        // parent component. If so, use it. The user will continue making
        // edits.
        if (this.state.preserved) {
            this.state.loaded = true
            return
        }

        // We currently only support updating a subnet. In this case the subnet
        // id must be provided.
        if (this.subnetId) {
            // Send POST to /subnets/{id}/transaction/new.
            this.updateSubnetBegin()
        } else {
            // Send POST to /subnets/transaction/new.
            this.createSubnetBegin()
            this.state.wizard = true
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
        this.state.preserved = true
        this.formDestroy.emit(this.state)
    }

    /**
     * Initializes the subnet form state using the data received from the server.
     *
     * @param response response received from the server holding the subnet data.
     */
    initializeState(response: CreateSubnetBeginResponse | UpdateSubnetBeginResponse) {
        // Success. Clear any existing errors.
        this.state.initError = null

        // The server should return new transaction id and a current list of
        // daemons to select.
        this.state.transactionId = response.id
        this.state.allDaemons = []
        for (let d of response.daemons) {
            this.state.allDaemons.push({
                id: d.id,
                appId: d.app.id,
                appType: d.app.type,
                name: d.name,
                version: d.version,
                label: `${d.app.name}/${d.name}`,
            })
        }
        // Initially, list all daemons.
        this.state.filteredDaemons = this.state.allDaemons
        this.state.allSharedNetworks4 = response.sharedNetworks4 || []
        this.state.allSharedNetworks6 = response.sharedNetworks6 || []
        this.state.clientClasses =
            response.clientClasses?.map((c) => {
                return { name: c }
            }) || []
        // If we update an existing subnet the subnet information should be in the response.
        if (this.subnetId && 'subnet' in response && response.subnet) {
            // Remember the subnet identifier.
            this.state.subnetId = response.subnet.id
            // Get the server names to be displayed next to the configuration parameters.
            this.state.servers = response.subnet.localSubnets.map((ls) => this.state.getDaemonById(ls.daemonId)?.label)
            // Save the subnet information in case we need to revert the form changes.
            // Determine whether it is an IPv6 or IPv4 subnet.
            this.state.dhcpv6 = response.subnet.subnet?.includes(':')
            // Initialize the subnet form controls.
            this.initializeSubnet(response)
        }
        // After the form has been initialized we need to filter out the daemons
        // that can be selected by a user for our subnet.
        this.handleDaemonsChange()
        // Hide the spinner and show the form.
        this.state.loaded = true
    }

    /**
     * Initializes subnet-specific controls in the form.
     *
     * @param subnet subnet data received from the server.
     */
    private initializeSubnet(response: UpdateSubnetBeginResponse): void {
        this.state.group = this.subnetSetFormService.convertSubnetToForm(
            this.state.dhcpv6 ? IPType.IPv6 : IPType.IPv4,
            getVersionRange(response.daemons.map((d) => d.version)),
            response.subnet
        )
    }

    /**
     * Sends a request to the server to begin a new transaction for updating
     * a subnet.
     */
    private createSubnetBegin(): void {
        lastValueFrom(this.dhcpApi.createSubnetBegin())
            .then((data) => {
                this.state.savedSubnetBeginData = data
                this.state.group = this.subnetSetFormService.createDefaultSubnetForm(
                    getVersionRange(data.daemons.map((d) => d.version)),
                    data.subnets || []
                )
                this.initializeState(data)
            })
            .catch((err) => {
                const msg = getErrorMessage(err)
                this.messageService.add({
                    severity: 'error',
                    summary: 'Cannot create new transaction',
                    detail: `Failed to create transaction for subnet creation: ` + msg,
                    life: 10000,
                })
                this.state.loaded = true
                this.state.initError = msg
            })
    }

    /**
     * Sends a request to the server to begin a new transaction for updating
     * a subnet.
     */
    private updateSubnetBegin(): void {
        this.dhcpApi
            .updateSubnetBegin(this.subnetId)
            .toPromise()
            .then((data) => {
                this.state.savedSubnetBeginData = data
                this.initializeState(data)
            })
            .catch((err) => {
                const msg = getErrorMessage(err)
                this.messageService.add({
                    severity: 'error',
                    summary: 'Cannot create new transaction',
                    detail: `Failed to create transaction to update subnet ${this.subnetId}: ` + msg,
                    life: 10000,
                })
                this.state.loaded = true
                this.state.initError = msg
            })
    }

    /**
     * Returns the current subnet prefix.
     */
    get subnet(): string {
        return this.state.group.get('subnet')?.value || ''
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
     * Returns a string representation of a pool range.
     *
     * It is used in the headers of the accordion components showing the pools.
     *
     * @param index a pool index.
     * @returns A string representation of the pool range and a boolean flag
     * indicating if the pool has a range specified.
     */
    getPoolHeader(index: number): [string, boolean] {
        const pools = this.state.group.get('pools') as FormArray<FormGroup<AddressPoolForm>>
        const rangeStart = pools?.at(index)?.get('range.start')?.value
        const rangeEnd = pools?.at(index)?.get('range.end')?.value
        return rangeStart && rangeEnd ? [`${rangeStart}-${rangeEnd}`, true] : ['New Pool', false]
    }

    /**
     * Returns a string representation of a delegated prefix pool prefix.
     *
     * @param index a pool index.
     * @returns A string representation of the pool prefix and a boolean flag
     * indicating if the pool has the prefix specified.
     */
    getPrefixPoolHeader(index: number): [string, boolean] {
        const pools = this.state.group.get('prefixPools') as FormArray<FormGroup<PrefixPoolForm>>
        return pools?.at(index)?.get('prefixes.prefix')?.value
            ? [`${pools.at(index)?.get('prefixes.prefix').value}`, true]
            : ['New Pool', false]
    }

    /**
     * A function called when user specifies subnet prefix and clicks the
     * button to proceed editing the form.
     */
    onSubnetProceed(): void {
        this.state.group.get('subnet').disable()
        this.state.group = this.subnetSetFormService.createDefaultSubnetForm(
            getVersionRange(this.state.savedSubnetBeginData.daemons.map((d) => d.version)),
            this.state.group.get('subnet').value
        )
        this.initializeState(this.state.savedSubnetBeginData)
        this.state.wizard = false
    }

    /**
     * A function called when a user clicked to add a new option form.
     *
     * It creates a new default form group for the option.
     *
     * @param index server index in the {@link servers} array.
     */
    onOptionAdd(index: number): void {
        this.getOptionsData(index).push(createDefaultDhcpOptionFormGroup(this.state.dhcpv6 ? IPType.IPv6 : IPType.IPv4))
    }

    /**
     * Returns a list of selected daemons.
     *
     * @returns A list of selected daemons.
     */
    getSelectedDaemons(): SelectableDaemon[] {
        const selectedDaemons = this.state.group.get('selectedDaemons').value ?? []
        return selectedDaemons.map((sd) => this.state.allDaemons.find((d) => d.id === sd))
    }

    /**
     * Adjusts the form state based on the selected daemons.
     *
     * Servers selection affects the form contents. When none are selected, the
     * default form should be displayed. Otherwise, we should track the configuration
     * values for the respective servers. Removing a server also results in the
     * form update because the parts of the form related to that server must be
     * removed.
     *
     * @param toggledDaemonId optional id of the removed daemon in the controls.
     */
    private handleDaemonsChange(toggledDaemonId?: number): void {
        const toggledDaemonIndex = toggledDaemonId
            ? this.state.filteredDaemons.findIndex((fd) => fd.id === toggledDaemonId)
            : -1
        this.subnetSetFormService.adjustFormForSelectedDaemons(
            this.state.group,
            toggledDaemonIndex,
            this.state.servers.length
        )
        this.addressPoolComponents.forEach((apc) => {
            apc.handleDaemonsChange(toggledDaemonId)
            apc.selectableDaemons = this.getSelectedDaemons()
        })
        this.prefixPoolComponents.forEach((ppc) => {
            ppc.handleDaemonsChange(toggledDaemonId)
            ppc.selectableDaemons = this.getSelectedDaemons()
        })

        // Selecting new daemons may have a large impact on the data already
        // inserted to the form. Update the form state accordingly and see
        // if it is breaking change.
        const selectedDaemons = this.state.group.get('selectedDaemons').value ?? []
        const subnetPrefix = this.state.group.get('subnet').value ?? ''
        if (this.state.updateFormForSelectedDaemons(selectedDaemons, subnetPrefix)) {
            // The breaking change puts us at risk of having irrelevant form contents.
            this.resetOptionsArray()
            this.resetParametersArray()
            return
        }
        // If the number of selected daemons has changed we must update selected servers list.
        this.state.servers = selectedDaemons.map((sd) => this.state.getDaemonById(sd)?.label)
    }

    /**
     * A callback invoked when shared network has been selected or deselected.
     *
     * @param event an event carrying the selected shared network ID.
     */
    onSharedNetworkChange(event): void {
        if (event.value) {
            this.state.group.get('selectedDaemons')?.disable()
        } else {
            this.state.group.get('selectedDaemons')?.enable()
        }
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
     * A callback invoked when new address pool is added.
     *
     * It extends the form to hold the new pool information.
     */
    onAddressPoolAdd(): void {
        const pools = this.state.group.get('pools') as FormArray<FormGroup<AddressPoolForm>>
        pools?.push(this.subnetSetFormService.createDefaultAddressPoolForm(this.subnet))
    }

    /**
     * A callback invoked when an address pool is deleted.
     *
     * Besides deleting the pool it also notifies the user that the pool has
     * been deleted using the message service.
     *
     * @param index pool index.
     */
    onAddressPoolDelete(index: number): void {
        const [poolHeader, specified] = this.getPoolHeader(index)
        const pools = this.state.group.get('pools') as FormArray<FormGroup<AddressPoolForm>>
        if (pools?.length > index) {
            pools.removeAt(index)
            this.messageService.add({
                severity: 'info',
                summary: specified ? `Pool ${poolHeader} deleted from the form` : `Pool deleted from the form`,
                detail: 'You can restore the original pools using the Revert Changes button below.',
                life: 10000,
            })
        }
    }

    /**
     * A callback invoked when new prefix pool is added.
     *
     * It extends the form to hold the new pool information.
     */
    onPrefixPoolAdd(): void {
        const pools = this.state.group.get('prefixPools') as FormArray<FormGroup<PrefixPoolForm>>
        pools?.push(this.subnetSetFormService.createDefaultPrefixPoolForm())
    }

    /**
     * A callback invoked when an address pool is deleted.
     *
     * Besides deleting the pool it also notifies the user that the pool has
     * been deleted using the message service.
     *
     * @param index pool index.
     */
    onPrefixPoolDelete(index: number): void {
        const [poolHeader, specified] = this.getPrefixPoolHeader(index)
        const pools = this.state.group.get('prefixPools') as FormArray<FormGroup<PrefixPoolForm>>
        if (pools?.length > index) {
            pools.removeAt(index)
            this.messageService.add({
                severity: 'info',
                summary: specified ? `Pool ${poolHeader} deleted from the form` : `Pool deleted from the form`,
                detail: 'You can restore the original pools using the Revert Changes button below.',
                life: 10000,
            })
        }
    }

    /**
     * A function called when user clicks the button to revert subnet changes.
     */
    onRevert(): void {
        this.initializeState(this.state.savedSubnetBeginData)
    }

    /**
     * A function called when user clicks the cancel button.
     */
    onCancel(): void {
        this.formCancel.emit(this.subnetId)
    }

    /**
     * A function called when user clicks the retry button after failure to begin
     * a new transaction.
     */
    onRetry(): void {
        if (this.subnetId) {
            this.updateSubnetBegin()
        }
    }

    /**
     * A function called when user clicks the submit button.
     */
    onSubmit(): void {
        let subnet: Subnet
        try {
            const filteredDaemons = this.state.group.get('selectedDaemons').value.map((id) => {
                return {
                    id: id,
                    version: this.state.filteredDaemons?.find((d) => d.id === id)?.version,
                }
            })
            subnet = this.subnetSetFormService.convertFormToSubnet(filteredDaemons, this.state.group)
            if (subnet.sharedNetworkId) {
                subnet.sharedNetwork = this.state.selectableSharedNetworks?.find(
                    (sn) => subnet.sharedNetworkId === sn.id
                )?.name
            }
        } catch (err) {
            this.messageService.add({
                severity: 'error',
                summary: 'Cannot commit the subnet',
                detail: 'Failed to process the subnet form: ' + err,
                life: 10000,
            })
            return
        }
        if (this.subnetId) {
            // Updating an existing subnet.
            const savedBeginData = this.state.savedSubnetBeginData as UpdateSubnetBeginResponse
            const originalSubnet = savedBeginData.subnet
            for (let ls of subnet.localSubnets) {
                const originalLocalSubnet =
                    originalSubnet.localSubnets.find((ols) => ols.daemonId === ls.daemonId) || subnet.localSubnets[0]
                ls.id = originalLocalSubnet?.id
            }
            subnet.id = this.subnetId
            this.dhcpApi
                .updateSubnetSubmit(this.subnetId, this.state.transactionId, subnet)
                .toPromise()
                .then(() => {
                    this.messageService.add({
                        severity: 'success',
                        summary: 'Subnet successfully updated',
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
                        summary: 'Cannot commit subnet updates',
                        detail: 'Failed to update the subnet: ' + msg,
                        life: 10000,
                    })
                })
            return
        }
        // Creating a new subnet.
        this.dhcpApi
            .createSubnetSubmit(this.state.transactionId, subnet)
            .toPromise()
            .then((data) => {
                this.messageService.add({
                    severity: 'success',
                    summary: 'Subnet successfully created',
                })
                // Notify the parent component about successful submission.
                this.state.subnetId = data.subnetId
                this.formSubmit.emit(this.state)
            })
            .catch((err) => {
                let msg = getErrorMessage(err)
                this.messageService.add({
                    severity: 'error',
                    summary: 'Cannot commit subnet',
                    detail: 'Failed to create the subnet: ' + msg,
                    life: 10000,
                })
            })
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
            this.subnetSetFormService.createDefaultKeaSubnetParametersForm(
                this.state.dhcpv6 ? IPType.IPv6 : IPType.IPv4,
                getVersionRange(this.state.savedSubnetBeginData.daemons.map((d) => d.version))
            )
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
