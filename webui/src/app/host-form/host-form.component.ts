import { Component, OnInit, Input, Output, EventEmitter, OnDestroy } from '@angular/core'
import {
    AbstractControl,
    UntypedFormBuilder,
    UntypedFormArray,
    UntypedFormGroup,
    ValidatorFn,
    Validators,
    ValidationErrors,
    UntypedFormControl,
} from '@angular/forms'
import { MessageService, SelectItem } from 'primeng/api'
import { map } from 'rxjs/operators'
import { collapseIPv6Number, isIPv4, IPv4, IPv6, Validator } from 'ip-num'
import { StorkValidators } from '../validators'
import { DHCPService } from '../backend/api/api'
import { CreateHostBeginResponse } from '../backend/model/createHostBeginResponse'
import { DHCPOption } from '../backend/model/dHCPOption'
import { Host } from '../backend/model/host'
import { IPReservation } from '../backend/model/iPReservation'
import { LocalHost } from '../backend/model/localHost'
import { UpdateHostBeginResponse } from '../backend/model/updateHostBeginResponse'
import { Subnet } from '../backend/model/subnet'
import { HostForm } from '../forms/host-form'
import { createDefaultDhcpOptionFormGroup } from '../forms/dhcp-option-form'
import { DhcpOptionSetFormService } from '../forms/dhcp-option-set-form.service'
import { SelectableDaemon } from '../forms/selectable-daemon'
import { IPType } from '../iptype'
import { getErrorMessage, stringToHex } from '../utils'
import { SelectableClientClass } from '../forms/selectable-client-class'
import { hasDifferentLocalHostData } from '../hosts'
import { GenericFormService } from '../forms/generic-form.service'

/**
 * A form validator checking if a subnet has been selected for
 * non-global reservation.
 *
 * @param group top-level component form group.
 * @returns validation errors when no subnet has been selected for
 *          a non-global reservation.
 */
function subnetRequiredValidator(group: UntypedFormGroup): ValidationErrors | null {
    if (!group.get('globalReservation').value && !group.get('selectedSubnet').value) {
        // It is not a global reservation and no subnet has been selected.
        const errs = {
            err: 'subnet is required for non-global reservations',
        }
        // Highlight the dropdown.
        if (group.get('selectedSubnet').touched && group.get('selectedSubnet').dirty) {
            group.get('selectedSubnet').setErrors(errs)
        }
        return errs
    }
    // Clear errors because everything seems fine.
    group.get('selectedSubnet').setErrors(null)
    return null
}

/**
 * A form validator checking if a selected DHCP identifier has been specified
 * and does not exceed the maximum length.
 *
 * @param group a selected form group belonging to the ipGroups array.
 * @returns validation errors when no value has been specified in the
 *          idInputHex input box (when selected type is hex) or idInputText
 *          (when selected type is text). It also returns validation errors
 *          when hw-address exceeds 40 hexadecimal digits or 20 characters
 *          or when other identifiers exceed 256 hexadecimal digits or 128
 *          characters.
 */
function identifierValidator(group: UntypedFormGroup): ValidationErrors | null {
    const idType = group.get('idType')
    const idInputHex = group.get('idInputHex')
    const idInputText = group.get('idInputText')
    let valErrors: ValidationErrors = null
    switch (group.get('idFormat').value) {
        case 'hex':
            // User selected hex format. Clear validation errors pertaining
            // to the idInputText and set errors for idInputHex if the
            // required value is not specified.
            idInputText.setErrors(null)
            valErrors =
                Validators.required(idInputHex) ||
                StorkValidators.hexIdentifierLength(idType.value === 'hw-address' ? 40 : 256)(idInputHex)
            if (idInputHex.valid) {
                idInputHex.setErrors(valErrors)
            }
            return valErrors
        case 'text':
            // User selected text format.
            idInputHex.setErrors(null)
            valErrors =
                Validators.required(idInputText) ||
                Validators.maxLength(idType.value === 'hw-address' ? 20 : 128)(idInputText)
            if (idInputText.valid) {
                idInputText.setErrors(valErrors)
            }
            return valErrors
    }
    return null
}

/**
 * A form validator checking if the specified IP address is within a
 * selected subnet range.
 *
 * It skips the validation if the IP address is not specified, if the
 * specified address is invalid, subnet hasn't been selected or the
 * reservation is global.
 *
 * @param ipType specified if an IPv4 or IPv6 address is validated.
 * @param hostForm a host form state.
 * @returns validator function that returns validation errors when a
 *          subnet is selected and the specified IPv4 or IPv6 address
 *          is not in this subnet range.
 */
function addressInSubnetValidator(ipType: IPType, hostForm: HostForm): ValidatorFn {
    return (control: AbstractControl): ValidationErrors | null => {
        // The value must be specified, must be a correct IP address, the
        // reservation must not be global and the subnet must be specified.
        if (
            control.value === null ||
            typeof control.value !== 'string' ||
            control.value.length === 0 ||
            !hostForm ||
            (hostForm.group && hostForm.group.get('globalReservation').value) ||
            !hostForm.filteredSubnets ||
            (ipType === IPType.IPv4 && !Validator.isValidIPv4String(control.value)[0]) ||
            (ipType === IPType.IPv6 && !Validator.isValidIPv6String(control.value)[0])
        ) {
            return null
        }
        // Convert the address to an IPv4 or IPv6 object.
        let ipAddress: IPv4 | IPv6
        ipAddress = ipType === IPType.IPv4 ? IPv4.fromString(control.value) : IPv6.fromString(control.value)
        if (!ipAddress) {
            return null
        }
        // Find the selected subnet range.
        const subnetRange = hostForm.getSelectedSubnetRange()
        if (!subnetRange) {
            return null
        }
        // Make sure the address is within the subnet boundaries.
        if (ipAddress.isLessThan(subnetRange[1].getFirst()) || ipAddress.isGreaterThan(subnetRange[1].getLast())) {
            return { 'ip-subnet-range': `IP address is not in the subnet ${subnetRange[0]} range.` }
        }
        return null
    }
}

/**
 * Converted data from the server's response to the createHostAdd or
 * createHostUpdate call.
 *
 * The received data is processed by this component to create a list of
 * selectable daemons. The list of selectable daemons comprises labels
 * of the servers that the user sees in the multi-select dropdown. The
 * object implementing this interface may optionally contain a host
 * instance, that is returned only in response to the createHostUpdate
 * call.
 */
export interface MappedHostBeginData {
    id: number
    subnets: Array<Subnet>
    daemons: Array<SelectableDaemon>
    clientClasses: Array<SelectableClientClass>
    host?: Host
}

/**
 * A component providing a form for editing and adding new host
 * reservation.
 */
@Component({
    selector: 'app-host-form',
    templateUrl: './host-form.component.html',
    styleUrls: ['./host-form.component.sass'],
})
export class HostFormComponent implements OnInit, OnDestroy {
    /**
     * Form state instance.
     *
     * The instance is shared between the parent and this component.
     * Holding the instance in the parent component allows for restoring
     * the form (after edits) after the component has been (temporarily)
     * destroyed.
     */
    @Input() form: HostForm = null

    /**
     * Host identifier.
     *
     * It should be set in cases when the form is used to update an existing
     * host reservation. It is not set when the form is used to create new
     * host reservation
     */
    @Input() hostId: number = 0

    /**
     * An event emitter notifying that the component is destroyed.
     *
     * A parent component receiving this event can remember the current
     * form state.
     */
    @Output() formDestroy = new EventEmitter<HostForm>()

    /**
     * An event emitter notifying that the form has been submitted.
     */
    @Output() formSubmit = new EventEmitter<HostForm>()

    /**
     * An event emitter notifying that form editing has been canceled.
     */
    @Output() formCancel = new EventEmitter<number>()

    /**
     * Different IP reservation types listed in the drop down.
     */
    ipTypes: SelectItem[] = []

    /**
     * Different host identifier types listed in the drop down.
     */
    hostIdTypes: SelectItem[] = []

    /**
     * Different identifier input formats listed in the drop down.
     */
    hostIdFormats = [
        {
            label: 'hex',
            value: 'hex',
        },
        {
            label: 'text',
            value: 'text',
        },
    ]

    /**
     * Default placeholder displayed in the IPv4 reservation input box.
     */
    static defaultIPv4Placeholder = '?.?.?.?'

    /**
     * Default placeholder displayed in the IPv6 reservation input box.
     */
    static defaultIPv6Placeholder = 'e.g. 2001:db8:1::'

    /**
     * Current placeholder displayed in the IPv4 reservation input box.
     */
    ipv4Placeholder = HostFormComponent.defaultIPv4Placeholder

    /**
     * Current placeholder displayed in the IPv6 reservation input box.
     */
    ipv6Placeholder = HostFormComponent.defaultIPv6Placeholder

    /**
     * Holds the received server's response to the updateHostBegin call.
     *
     * It is required to revert host reservation edits.
     */
    savedUpdateHostBeginData: MappedHostBeginData

    /**
     * Constructor.
     *
     * @param _formBuilder private form builder instance.
     * @param _dhcpApi REST API server service.
     * @param _optionSetFormService service providing functions to convert the
     * host reservation information between the form and REST API formats.
     * @param _messageService service displaying error and success messages.
     */
    constructor(
        private _formBuilder: UntypedFormBuilder,
        private _dhcpApi: DHCPService,
        private _genericFormService: GenericFormService,
        private _optionSetFormService: DhcpOptionSetFormService,
        private _messageService: MessageService
    ) {}

    /**
     * Component lifecycle hook invoked during initialization.
     *
     * If the provided form instance has been preserved in the parent
     * component this instance is used and the initialization skipped.
     * Otherwise, the form is initialized to defaults.
     */
    ngOnInit(): void {
        // Initialize the form instance if the parent hasn't supplied one.
        if (!this.form) {
            this.form = new HostForm()
        }
        // Initialize the options in the drop down lists.
        this._updateHostIdTypes()
        this._updateIPTypes()

        // Check if the form has been already edited and preserved in the
        // parent component. If so, use it. The user will continue making
        // edits.
        if (this.form.preserved) {
            return
        }

        // New form.
        this._createDefaultFormGroup()

        // Begin transaction.
        if (this.hostId) {
            // Send POST to /hosts/{id}/transaction/new.
            this._updateHostBegin()
        } else {
            // Send POST to /hosts/new/transaction/new.
            this._createHostBegin()
        }
    }

    /**
     * Creates a default form group.
     *
     * It is used during the component initialization and when the current
     * changes are reverted on user's request.
     */
    private _createDefaultFormGroup(): void {
        this.formGroup = this._formBuilder.group(
            {
                globalReservation: [false],
                splitFormMode: [false],
                selectedDaemons: ['', Validators.required],
                selectedSubnet: [null],
                hostIdGroup: this._formBuilder.group(
                    {
                        idType: [this.hostIdTypes[0].label],
                        idInputHex: ['', StorkValidators.hexIdentifier()],
                        idInputText: [''],
                        idFormat: ['hex'],
                    },
                    {
                        validators: [identifierValidator],
                    }
                ),
                ipGroups: this._formBuilder.array([this._createNewIPGroup()]),
                hostname: ['', StorkValidators.fqdn],
                clientClasses: this._formBuilder.array([this._formBuilder.control(null)]),
                bootFields: this._formBuilder.array([this._createDefaultBootFieldsFormGroup()]),
                // The outer array holds different option sets for different servers.
                // The inner array holds the actual option sets. If the split-mode
                // is disabled, there is only one outer array.
                options: this._formBuilder.array([this._formBuilder.array([])]),
            },
            {
                validators: [subnetRequiredValidator],
            }
        )
    }

    /**
     * Creates a default form group for boot fields.
     *
     * The boot fields include next server, server hostname and the
     * boot file name controls.
     *
     * @returns created form group.
     */
    private _createDefaultBootFieldsFormGroup(): UntypedFormGroup {
        let formGroup = this._formBuilder.group({
            nextServer: ['', StorkValidators.ipv4()],
            serverHostname: ['', StorkValidators.fqdn],
            bootFileName: [''],
        })
        return formGroup
    }

    /**
     * Sends a request to the server to begin a new transaction for adding
     * new host reservation.
     *
     * If the call is successful, the form components are initialized with the
     * returned data, e.g. a list of available servers, subnets etc.
     * If an error occurs, the error text is remembered and displayed along
     * with the retry button.
     */
    private _createHostBegin(): void {
        this._dhcpApi
            .createHostBegin()
            .pipe(
                map((data) => {
                    // We have to mangle the returned information and store them
                    // in the format usable by the component.
                    return this._mapHostBeginData(data)
                })
            )
            .toPromise()
            .then((data) => {
                this._initializeForm(data)
            })
            .catch((err) => {
                const msg = getErrorMessage(err)
                this._messageService.add({
                    severity: 'error',
                    summary: 'Cannot create new transaction',
                    detail: 'Failed to create transaction for adding new host: ' + msg,
                    life: 10000,
                })
                this.form.initError = msg
            })
    }

    /**
     * Sends a request to the server to begin a new transaction for updating
     * a host reservation.
     *
     * If the call is successful, the form components initialized wih the
     * returned data, i.e., a list of available servers, subnets, host reservation
     * information. If an error occurs, the error text is remembered and displayed
     * along with the retry button.
     */
    private _updateHostBegin(): void {
        this._dhcpApi
            .updateHostBegin(this.hostId)
            .pipe(
                map((data) => {
                    // We have to mangle the returned information and store them
                    // in the format usable by the component.
                    return this._mapHostBeginData(data)
                })
            )
            .toPromise()
            .then((data) => {
                this._initializeForm(data)
            })
            .catch((err) => {
                const msg = getErrorMessage(err)
                this._messageService.add({
                    severity: 'error',
                    summary: 'Cannot create new transaction',
                    detail: `Failed to create transaction for updating host ${this.hostId}: ` + msg,
                    life: 10000,
                })
                this.form.initError = msg
            })
    }

    /**
     * Processes and converts received data when new transaction is begun.
     *
     * For each daemon, it generates a user friendly label by concatenating
     * app name and daemon name. The list of friendly names is displayed in
     * the dropdown where a user selects servers. Other data is returned with
     * no change.
     *
     * @param data a response received as a result of beginning a transaction
     *             to create a new host or to update an existing host.
     * @returns processed data that includes friendly daemon names.
     */
    private _mapHostBeginData(data: CreateHostBeginResponse | UpdateHostBeginResponse): MappedHostBeginData {
        const daemons: Array<SelectableDaemon> = []
        for (const d of data.daemons) {
            const daemon = {
                id: d.id,
                appId: d.app.id,
                appType: 'kea',
                name: d.name,
                version: d.name,
                label: `${d.app.name}/${d.name}`,
            }
            daemons.push(daemon)
        }
        const clientClasses: Array<SelectableClientClass> = []
        for (const c of data.clientClasses) {
            clientClasses.push({
                name: c,
            })
        }
        const mappedData: MappedHostBeginData = {
            id: data.id,
            subnets: data.subnets,
            daemons: daemons,
            clientClasses: clientClasses,
        }
        if ('host' in data) {
            mappedData.host = data.host
        }
        return mappedData
    }

    /**
     * Initializes the from with the data received when the transaction is begun.
     *
     * It sets transaction ID and a list of available daemons and subnets. If the
     * received data comprises host information, the form controls pertaining to
     * the host information are also filled.
     *
     * @param data a response received as a result of beginning a transaction
     *             to create a new host or to update an existing host.
     */
    private _initializeForm(data: MappedHostBeginData): void {
        // Success. Clear any existing errors.
        this.form.initError = null
        // The server should return new transaction id and a current list of
        // daemons and subnets to select.
        this.form.transactionId = data.id
        this.form.allDaemons = data.daemons
        this.form.allSubnets = data.subnets
        // Initially, list all daemons.
        this.form.filteredDaemons = this.form.allDaemons
        // Initially, show all subnets.
        this.form.filteredSubnets = this.form.allSubnets
        // Client classes.
        this.form.clientClasses = data.clientClasses
        // Initialize host-specific controls if the host information is available.
        if (this.hostId && 'host' in data && data.host) {
            this.savedUpdateHostBeginData = data
            this._initializeHost(data.host)
        }
    }

    /**
     * Initializes host reservation specific controls in the form.
     *
     * @param host host information received from the server and to be updated.
     */
    private _initializeHost(host: Host): void {
        const selectedDaemons: number[] = []
        const localHosts = (host.localHosts || []).filter((lh) => lh.dataSource === 'api')
        for (let lh of localHosts) {
            selectedDaemons.push(lh.daemonId)
        }
        this.formGroup.get('selectedDaemons').setValue(selectedDaemons)
        this._handleDaemonsChange()

        if (!host.subnetId) {
            this.formGroup.get('globalReservation').setValue(true)
        }
        if (host.subnetId) {
            this.formGroup.get('selectedSubnet').setValue(host.subnetId)
        }
        if (host.hostIdentifiers?.length > 0) {
            this.formGroup.get('hostIdGroup.idType').setValue(host.hostIdentifiers[0].idType)
            this.formGroup.get('hostIdGroup.idFormat').setValue('hex')
            this.formGroup.get('hostIdGroup.idInputHex').setValue(host.hostIdentifiers[0].idHexValue)
        }
        if (host.addressReservations?.length > 0 && (this.form.dhcpv4 || this.form.dhcpv6)) {
            for (let i = 0; i < host.addressReservations.length; i++) {
                if (this.ipGroups.length <= i) {
                    this.addIPInput()
                }
                if (this.form.dhcpv4) {
                    this.ipGroups.at(i).get('inputIPv4').setValue(host.addressReservations[i].address)
                } else {
                    this.ipGroups.at(i).get('inputNA').setValue(host.addressReservations[i].address)
                }
            }
        }
        if (host.prefixReservations?.length > 0) {
            for (let i = 0; i < host.prefixReservations.length; i++) {
                if (this.ipGroups.length <= i) {
                    this.addIPInput()
                }
                let pdSplit = host.prefixReservations[i].address.split('/', 2)
                if (pdSplit.length == 2) {
                    let pdLen = parseInt(pdSplit[1], 10)
                    if (!isNaN(pdLen) && pdLen <= 128) {
                        this.ipGroups.at(i).get('inputPDLen').setValue(pdLen)
                        this.ipGroups.at(i).get('inputPD').setValue(pdSplit[0])
                    }
                }
            }
        }
        if (host.hostname) {
            this.formGroup.get('hostname').setValue(host.hostname)
        }

        // Split form mode is only set when there are multiple servers associated
        // with the edited host and at least one of the servers has different
        // set of DHCP options, client classes or boot fields.
        const splitFormMode = hasDifferentLocalHostData(localHosts)
        this.formGroup.get('splitFormMode').setValue(splitFormMode)

        for (let i = 0; i < (splitFormMode ? localHosts.length : 1); i++) {
            // Options.
            this._genericFormService.setArrayControl(
                i,
                this.optionsArray,
                this._optionSetFormService.convertOptionsToForm(
                    this.form.dhcpv4 ? IPType.IPv4 : IPType.IPv6,
                    localHosts[i].options
                )
            )
            // Client classes.
            const clientClasses = localHosts[i].clientClasses ? [...localHosts[i].clientClasses] : []
            this._genericFormService.setArrayControl(
                i,
                this.clientClassesArray,
                this._formBuilder.control(clientClasses)
            )
            // Boot fields.
            let bootFields = this._createDefaultBootFieldsFormGroup()
            this._genericFormService.setFormGroupValues(bootFields, localHosts[i])
            this._genericFormService.setArrayControl(i, this.bootFieldsArray, bootFields)
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
        this.form.preserved = true
        this.formDestroy.emit(this.form)
    }

    /**
     * Returns main form group for the component.
     *
     * @returns form group.
     */
    get formGroup(): UntypedFormGroup {
        return this.form.group
    }

    /**
     * Sets main form group for the component.
     *
     * @param fg new form group.
     */
    set formGroup(fg: UntypedFormGroup) {
        this.form.group = fg
    }

    /**
     * Updates presented list of selectable host ID types.
     *
     * The list depends on whether we have selected a DHCPv6 server,
     * DHCPv4 server or no servers.
     */
    private _updateHostIdTypes(): void {
        if (this.form.dhcpv6) {
            // DHCPv6 server supports fewer identifier types.
            this.hostIdTypes = [
                {
                    label: 'hw-address',
                    value: 'hw-address',
                },
                {
                    label: 'duid',
                    value: 'duid',
                },
                {
                    label: 'flex-id',
                    value: 'flex-id',
                },
            ]
            return
        }
        this.hostIdTypes = [
            {
                label: 'hw-address',
                value: 'hw-address',
            },
            {
                label: 'client-id',
                value: 'client-id',
            },
            {
                label: 'circuit-id',
                value: 'circuit-id',
            },
            {
                label: 'duid',
                value: 'duid',
            },
            {
                label: 'flex-id',
                value: 'flex-id',
            },
        ]
    }

    /**
     * Adds new IP address or delegated prefix input box to the form.
     *
     * By default, the input box is for an IPv4 reservation. However, if
     * there is another box already (only possible in the IPv6 case), the
     * new box uses the type of the last box.
     */
    addIPInput(): void {
        let ipType = this.form.dhcpv6 ? 'ia_na' : 'ipv4'
        // Check if some IP input boxes have been already added.
        if (this.ipGroups.length > 0) {
            // Some input boxes already exist. Use the last one's type
            // as a default.
            ipType = this.ipGroups.at(this.ipGroups.length - 1).get('ipType').value
        }
        this.ipGroups.push(this._createNewIPGroup(ipType))
    }

    /**
     * Deletes specified IP address or delegated prefix input box from
     * the form.
     *
     * @param index input box index beginning from 0.
     */
    deleteIPInput(index): void {
        ;(this.formGroup.get('ipGroups') as UntypedFormArray).removeAt(index)
    }

    /**
     * Updates presented list of selectable IP reservation types.
     *
     * The list depends on whether we have selected a DHCPv6 server,
     * DHCPv4 server or no servers.
     */
    private _updateIPTypes(): void {
        if (this.form.dhcpv6) {
            this.ipTypes = [
                {
                    label: 'IPv6 address',
                    value: 'ia_na',
                },
                {
                    label: 'IPv6 prefix',
                    value: 'ia_pd',
                },
            ]
            return
        }
        this.ipTypes = [
            {
                label: 'IPv4 address',
                value: 'ipv4',
            },
        ]
    }

    /**
     * Convenience function returning the form array with IP reservations.
     *
     * @returns form array with IP reservations.
     */
    get ipGroups(): UntypedFormArray {
        return this.formGroup.get('ipGroups') as UntypedFormArray
    }

    /**
     * Creates new form group for specifying new IP reservation.
     *
     * @param defaultType IP reservation type.
     * @returns form group for specifying new IP reservation.
     */
    private _createNewIPGroup(defaultType = 'ipv4'): UntypedFormGroup {
        return this._formBuilder.group({
            ipType: [defaultType],
            inputIPv4: [
                '',
                Validators.compose([StorkValidators.ipv4(), addressInSubnetValidator(IPType.IPv4, this.form)]),
            ],
            inputNA: [
                '',
                Validators.compose([StorkValidators.ipv6(), addressInSubnetValidator(IPType.IPv6, this.form)]),
            ],
            inputPD: ['', StorkValidators.ipv6()],
            inputPDLength: ['64', Validators.required],
        })
    }

    /**
     * Clears specified IP reservations.
     *
     * It is used in cases when user switches between different server types,
     * e.g. previously selected a DHCPv4 server and now switched to DHCPv6
     * server. In that case, the specified information is no longer valid.
     */
    private _resetIPGroups(): void {
        // Nothing to do if there are no IP reservations specified.
        if (this.ipGroups.length > 0) {
            this.ipGroups.clear()
            this.ipGroups.push(this._createNewIPGroup(this.form.dhcpv6 ? 'ia_na' : 'ipv4'))
        }
    }

    /**
     * Convenience function returning the form array with client class sets.
     *
     * Each item in the array comprises a form control with an array of
     * client classes as strings for one of the servers (when split edit
     * mode enabled) or for all servers (when split edit mode disabled.)
     *
     * @returns Form array comprising client classes for different servers.
     */
    get clientClassesArray(): UntypedFormArray {
        return this.formGroup.get('clientClasses') as UntypedFormArray
    }

    /**
     * Convenience function returning form control with client classes.
     *
     * @param index index of the control to return.
     * @returns form control with client classes.
     */
    getClientClassSetControl(index: number): UntypedFormControl {
        return this.clientClassesArray.at(index) as UntypedFormControl
    }

    /**
     * Resets the part of the form comprising client classes.
     *
     * It removes all existing client class form controls and re-creates
     * the default one.
     */
    private _resetClientClassesArray() {
        this.clientClassesArray.clear()
        this.clientClassesArray.push(this._formBuilder.control(null))
    }

    /**
     * Convenience function returning the form array with boot field sets.
     *
     * Each item in the array comprises a form group with three controls
     * (i.e., next server, server hostname and boot file name).
     */
    get bootFieldsArray(): UntypedFormArray {
        return this.formGroup.get('bootFields') as UntypedFormArray
    }

    /**
     * Convenience function returning form group with boot field controls.
     * for a single server or all servers.
     *
     * If the split form mode is false, it always returns the first (common)
     * group.
     *
     * @param index index of the form group to return.
     * @returns form group with boot fields.
     */
    getBootFieldsGroup(index: number): UntypedFormGroup {
        return this.bootFieldsArray.at(this.formGroup.get('splitFormMode').value ? index : 0) as UntypedFormGroup
    }

    /**
     * Resets the part of the form comprising boot fields.
     *
     * It removes all existing boot field form groups and re-creates
     * the default one.
     */
    private _resetBootFieldsArray() {
        this.bootFieldsArray.clear()
        this.bootFieldsArray.push(this._createDefaultBootFieldsFormGroup())
    }

    /**
     * Convenience function returning the form array with DHCP option sets.
     *
     * Each item in the array comprises another form array representing a
     * DHCP option set for one of the servers (when split edit mode enabled)
     * or for all servers (when split edit mode disabled).
     *
     * @returns Form array comprising option sets for different servers.
     */
    get optionsArray(): UntypedFormArray {
        return this.formGroup.get('options') as UntypedFormArray
    }

    /**
     * Convenience function returning the form array with DHCP options.
     *
     * @param index index of the form array to return.
     * @returns form array with DHCP options.
     */
    getOptionSetArray(index: number): UntypedFormArray {
        return this.optionsArray.at(index) as UntypedFormArray
    }

    /**
     * Resets the part of the form comprising DHCP options.
     *
     * It removes all existing option sets and re-creates the default one.
     */
    private _resetOptionsArray() {
        this.optionsArray.clear()
        this.optionsArray.push(this._formBuilder.array([]))
    }

    /**
     * A callback invoked when user toggles the split mode button.
     *
     * When the user turns on the split mode editing mode, this function
     * ensures that each selected DHCP server is associated with its own
     * options form. When the split mode is off, this function leaves only
     * one form, common for all servers.
     */
    onSplitModeChange(): void {
        const splitFormMode = this.formGroup.get('splitFormMode').value
        if (splitFormMode) {
            const selectedDaemons = this.formGroup.get('selectedDaemons').value
            const itemsToAdd = selectedDaemons.length - this.optionsArray.length
            if (selectedDaemons.length >= this.optionsArray.length) {
                for (let i = 0; i < itemsToAdd; i++) {
                    this.bootFieldsArray.push(this._genericFormService.cloneControl(this.bootFieldsArray.at(0)))
                    this.clientClassesArray.push(this._genericFormService.cloneControl(this.clientClassesArray.at(0)))
                    this.optionsArray.push(this._optionSetFormService.cloneControl(this.optionsArray.at(0)))
                }
            }
        } else {
            for (let i = this.optionsArray.length; i >= 1; i--) {
                this.bootFieldsArray.removeAt(i)
                this.clientClassesArray.removeAt(i)
                this.optionsArray.removeAt(i)
            }
        }
    }

    /**
     * Convenience function returning the control with selected daemons.
     *
     * returns form control with selected daemon IDs.
     */
    get selectedDaemons(): AbstractControl {
        return this.formGroup.get('selectedDaemons')
    }

    /**
     * Adjusts the form state based on the selected daemons.
     *
     * Servers selection affects available subnets. If no servers are selected,
     * all subnets are listed for selection. However, if one or more servers
     * are selected only those subnets served by all selected servers are
     * listed. In that case, each listed subnet must be served by all selected
     * servers. If selected servers have no common subnets, no subnets are
     * listed.
     */
    private _handleDaemonsChange(): void {
        // Capture the servers selected by the user.
        const selectedDaemons = this.formGroup.get('selectedDaemons').value

        // Selecting new daemons may have a large impact on the data already
        // inserted to the form. Update the form state accordingly and see
        // if it is breaking change.
        if (this.form.updateFormForSelectedDaemons(this.formGroup.get('selectedDaemons').value)) {
            // The breaking change puts us at risk of having server specific data
            // that no longer matches the servers selection. Let's reset the data.
            this._resetBootFieldsArray()
            this._resetOptionsArray()
            this._resetClientClassesArray()
        }

        // Selectable host identifier types depend on the selected server types.
        this._updateHostIdTypes()

        if (
            this.ipGroups.length === 0 ||
            (this.form.dhcpv4 && this.ipGroups.getRawValue().some((g) => g.ipType !== 'ipv4')) ||
            (this.form.dhcpv6 && this.ipGroups.getRawValue().some((g) => g.ipType === 'ipv4'))
        ) {
            // The current IP reservation edits no longer match the selected server types.
            // Let's reset current IP reservations and let the user start over.
            this._resetIPGroups()
            this._updateIPTypes()
        }

        // We take short path when no servers are selected. Just make all
        // subnets available.
        if (selectedDaemons.length === 0) {
            this.form.filteredSubnets = this.form.allSubnets
            return
        }
        // Filter subnets.
        this.form.filteredSubnets = this.form.allSubnets.filter((s) => {
            // We will be filtering by daemonId, so we need to look into
            // the localSubnet.
            return s.localSubnets.some((ls) => {
                return (
                    // At least one daemonId in the subnet should belong to
                    // the array of our selected servers AND each selected
                    // server must be associated with our subnet.
                    selectedDaemons.includes(ls.daemonId) > 0 &&
                    selectedDaemons.every((ss) => s.localSubnets.find((ls2) => ls2.daemonId === ss))
                )
            })
        })
        // Changing the list of selectable subnets may affect previous
        // subnet selection. If previously selected subnet is still in
        // the filtered list we can keep this selection. Otherwise, we
        // have to reset the subnet selection.
        if (!this.form.filteredSubnets.find((fs) => fs.id === this.formGroup.get('selectedSubnet').value)) {
            this.formGroup.get('selectedSubnet').patchValue(null)
        }

        const splitFormMode = this.formGroup.get('splitFormMode').value
        if (splitFormMode) {
            let bootFieldSets: UntypedFormGroup[] = []
            let clientClassSets: UntypedFormControl[] = []
            let optionSets: UntypedFormArray[] = []
            for (let i = 0; i < selectedDaemons.length; i++) {
                bootFieldSets.push(this._createDefaultBootFieldsFormGroup())
                clientClassSets.push(this._formBuilder.control(null))
                optionSets.push(this._formBuilder.array([]))
            }
            this.formGroup.setControl('bootFields', this._formBuilder.array(bootFieldSets))
            this.formGroup.setControl('clientClasses', this._formBuilder.array(clientClassSets))
            this.formGroup.setControl('options', this._formBuilder.array(optionSets))
        }
    }

    /**
     * A callback invoked when selected DHCP servers have changed.
     *
     * Adjusts the form state based on the selected daemons.
     */
    onDaemonsChange(): void {
        this._handleDaemonsChange()
    }

    /**
     * A callback called when a subnet has been selected or de-selected.
     *
     * It iterates over the specified IP addresses and checks if they belong
     * to the new subnet boundaries. It also updates the placeholders of the
     * respective input boxes. The placeholders contain IP addresses suitable
     * for the selected subnet.
     */
    onSelectedSubnetChange(): void {
        for (let i = 0; i < this.ipGroups.length; i++) {
            this.ipGroups.at(i).get('inputIPv4').updateValueAndValidity()
            this.ipGroups.at(i).get('inputNA').updateValueAndValidity()
        }
        const range = this.form.getSelectedSubnetRange()
        if (range) {
            let first = range[1].getFirst()
            if (isIPv4(first)) {
                this.ipv4Placeholder = `in range of ${first.toString()} - ${range[1].getLast()}`
            } else {
                this.ipv6Placeholder = collapseIPv6Number(first.toString())
            }
        } else {
            this.ipv4Placeholder = HostFormComponent.defaultIPv4Placeholder
            this.ipv6Placeholder = HostFormComponent.defaultIPv6Placeholder
        }
    }

    /**
     * A callback called when new host identifier type has been selected.
     *
     * It updates the validity of the input fields in which the identifier is
     * specified.
     */
    onSelectedIdentifierChange(): void {
        if (this.formGroup.get('hostIdGroup.idFormat').value === 'hex') {
            this.formGroup.get('hostIdGroup.idInputHex').updateValueAndValidity()
        } else {
            this.formGroup.get('hostIdGroup.idInputText').updateValueAndValidity()
        }
    }

    /**
     * A function called when a user clicked to add a new option form.
     *
     * It creates a new default form group for the option.
     */
    onOptionAdd(index: number): void {
        this.getOptionSetArray(index).push(
            createDefaultDhcpOptionFormGroup(this.form.dhcpv6 ? IPType.IPv6 : IPType.IPv4)
        )
    }

    /**
     * A function called when a user attempts to submit the new host reservation
     * or update an existing one.
     *
     * It collects the data from the form and sends the request to commit the
     * current transaction (hosts/new/transaction/{id}/submit or
     * hosts/{id}/transaction/{id}/submit).
     */
    onSubmit(): void {
        // Check if it is global reservation or subnet-level reservation.
        const selectedSubnet = this.formGroup.get('globalReservation').value
            ? 0
            : this.formGroup.get('selectedSubnet').value

        // Client classes.
        let clientClasses: Array<Array<string>> = []
        for (let ctrl of this.clientClassesArray.controls) {
            let clientClass: Array<string> = []
            if (Array.isArray(ctrl.value)) {
                for (let c of ctrl.value) {
                    clientClass.push(c)
                }
            }
            clientClasses.push(clientClass)
            if (!this.formGroup.get('splitFormMode').value) {
                break
            }
        }

        // DHCP options.
        let options: Array<Array<DHCPOption>> = []
        for (let arr of this.optionsArray.controls) {
            try {
                options.push(
                    this._optionSetFormService.convertFormToOptions(
                        this.form.dhcpv4 ? IPType.IPv4 : IPType.IPv6,
                        arr as UntypedFormArray
                    )
                )
                // There should be only one option set when the split mode is disabled.
                if (!this.formGroup.get('splitFormMode').value) {
                    break
                }
            } catch (err) {
                this._messageService.add({
                    severity: 'error',
                    summary: 'Cannot commit new host',
                    detail: 'Failed to process specified DHCP options: ' + err,
                    life: 10000,
                })
                return
            }
        }

        // Create associations with the daemons.
        let localHosts: LocalHost[] = []
        const selectedDaemons = this.formGroup.get('selectedDaemons').value
        for (let i = 0; i < selectedDaemons.length; i++) {
            localHosts.push({
                daemonId: selectedDaemons[i],
                dataSource: 'api',
                clientClasses: this.formGroup.get('splitFormMode').value ? clientClasses[i] : clientClasses[0],
                options: this.formGroup.get('splitFormMode').value ? options[i] : options[0],
            })
            // Boot fields.
            this._genericFormService.setValuesFromFormGroup(this.getBootFieldsGroup(i), localHosts[i])
        }

        // Use hex value or convert text value to hex.
        const idHexValue =
            this.formGroup.get('hostIdGroup.idFormat').value === 'hex'
                ? this.formGroup.get('hostIdGroup.idInputHex').value.trim()
                : stringToHex(this.formGroup.get('hostIdGroup.idInputText').value.trim())

        let addressReservations: IPReservation[] = []
        let prefixReservations: IPReservation[] = []
        for (let i = 0; i < this.ipGroups.length; i++) {
            const group = this.ipGroups.at(i)
            switch (group.get('ipType').value) {
                case 'ipv4':
                    const inputIPv4 = group.get('inputIPv4').value.trim()
                    if (inputIPv4.length > 0) {
                        addressReservations.push({
                            address: `${inputIPv4}/32`,
                        })
                    }
                    break
                case 'ia_na':
                    const inputNA = group.get('inputNA').value.trim()
                    if (inputNA.length > 0) {
                        addressReservations.push({
                            address: `${inputNA}/128`,
                        })
                    }
                    break
                case 'ia_pd':
                    const inputPD = group.get('inputPD').value.trim()
                    if (inputPD.length > 0) {
                        prefixReservations.push({
                            address: `${inputPD}/${group.get('inputPDLength').value}`,
                        })
                    }
                    break
            }
        }

        // Create host.
        let host: Host = {
            subnetId: selectedSubnet,
            hostIdentifiers: [
                {
                    idType: this.formGroup.get('hostIdGroup.idType').value,
                    idHexValue: idHexValue,
                },
            ],
            addressReservations: addressReservations,
            prefixReservations: prefixReservations,
            hostname: this.formGroup.get('hostname').value.trim(),
            localHosts: localHosts,
        }

        // Update the existing host.
        if (this.hostId) {
            host.id = this.hostId
            this._dhcpApi
                .updateHostSubmit(this.hostId, this.form.transactionId, host)
                .toPromise()
                .then(() => {
                    this._messageService.add({
                        severity: 'success',
                        summary: 'Host reservation successfully updated',
                    })
                    // Notify the parent component about successful submission.
                    this.formSubmit.emit(this.form)
                })
                .catch((err) => {
                    let msg = err.statusText
                    if (err.error && err.error.message) {
                        msg = err.error.message
                    }
                    this._messageService.add({
                        severity: 'error',
                        summary: 'Cannot commit host updates',
                        detail: 'The transaction to update the host failed: ' + msg,
                        life: 10000,
                    })
                })
            return
        }
        // Submit new host.
        this._dhcpApi
            .createHostSubmit(this.form.transactionId, host)
            .toPromise()
            .then(() => {
                this._messageService.add({
                    severity: 'success',
                    summary: 'Host reservation successfully added',
                })
                // Notify the parent component about successful submission.
                this.formSubmit.emit(this.form)
            })
            .catch((err) => {
                const msg = getErrorMessage(err)
                this._messageService.add({
                    severity: 'error',
                    summary: 'Cannot commit new host',
                    detail: 'The transaction to add a new host failed: ' + msg,
                    life: 10000,
                })
            })
    }

    /**
     * A function called when user clicks the retry button after failure to begin
     * a new transaction.
     */
    onRetry(): void {
        if (this.hostId) {
            this._updateHostBegin()
        } else {
            this._createHostBegin()
        }
    }

    /**
     * A function called when user clicks the button to revert host edit changes.
     */
    onRevert(): void {
        this._createDefaultFormGroup()
        this._initializeForm(this.savedUpdateHostBeginData)
    }

    /**
     * A function called when user clicks cancel button.
     *
     * It causes the parent component to close the form.
     */
    onCancel(): void {
        this.formCancel.emit(this.hostId)
    }
}
