import { Component, OnInit, Input, Output, EventEmitter, OnDestroy } from '@angular/core'
import { AbstractControl, FormBuilder, FormArray, FormGroup } from '@angular/forms'
import { SelectItem } from 'primeng/api'
import { StorkValidators } from '../validators'

/**
 * Holds the state of the form created by the HostFormComponent.
 *
 * The state is shared with the parent component via the event emitter
 * and can be used to re-create the HostFormComponent with the already
 * edited form data. It is particularly useful when the component is
 * destroyed as a result of switching between different tabs.
 */
class HostForm {
    /**
     * A boolean value indicating if the updated form was passed to
     * the parent component when the HostFormComponent was destroyed.
     *
     * When the component is re-created it checks this value to decide
     * whether or not the form should be initialized with default values.
     */
    preserved: boolean = false

    /**
     * A form group comprising all form controls, arrays and other form
     * groups (a parent group for the HostFormComponent form).
     */
    group: FormGroup

    filteredDaemons: any[]

    /**
     * An array of selectable subnets according to the current form data.
     *
     * Suppose a user selected a server in the form. In this case, this
     * array comprises only the subnets served by this server.
     */
    filteredSubnets: any[]

    /**
     * A boolean value indicating if a button for adding next IP reservation
     * should be disabled.
     *
     * The button is disabled when user is adding an IPv4 reservation. The
     * DHCP server accepts at most one IPv4 reservation.
     */
    addIPButtonDisabled: boolean

    /**
     * A boolean value indicating if the dropdown with different IP reservation
     * types should only comprise IPv6 specific ones.
     *
     * This is the case when a user has already added one IPv6 reservation and
     * is adding a second one. The reservation should not mix IPv4 and IPv6
     * addresses and/or prefixes.
     */
    ipv6OnlyTypes: boolean
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
     * A list of daemons that can be selected from the drop down list.
     */
    @Input() daemons: any = [
        {
            id: 1,
            name: 'dhcp4',
            label: 'kea@192.0.2.1/dhcp4',
        },
        {
            id: 2,
            name: 'dhcp6',
            label: 'kea@192.0.2.1/dhcp6',
        },
        {
            id: 3,
            name: 'dhcp4',
            label: 'kea@192.0.2.2/dhcp4',
        },
        {
            id: 4,
            name: 'dhcp6',
            label: 'kea@192.0.2.2/dhcp6',
        },
    ]

    /**
     * A list of subnets that can be selected from the drop down list.
     *
     * An actual drop down list can be shorter depending on the list of
     * selected servers. It displays only the subnets that the selected
     * servers serve.
     */
    @Input() subnets: any = [
        {
            id: 1,
            subnet: '192.0.2.0/24',
            ipv4: true,
            localSubnets: [
                {
                    daemonId: 1,
                },
            ],
        },
        {
            id: 2,
            subnet: '2001:db8:1::/64',
            ipv4: false,
            localSubnets: [
                {
                    daemonId: 2,
                },
                {
                    daemonId: 4,
                },
            ],
        },
    ]

    /**
     * An event emitter notifying about the changes in the form.
     *
     * The event is emitted when the component is destroyed, so the
     * parent component can remember the current form state.
     */
    @Output() onFormChange = new EventEmitter<HostForm>()

    /**
     * Different IP reservation types listed in the drop down.
     */
    ipTypes: SelectItem[] = [
        {
            label: 'IPv4 address',
            value: 'ipv4',
        },
        {
            label: 'IPv6 address',
            value: 'ia_na',
        },
        {
            label: 'IPv6 prefix',
            value: 'ia_pd',
        },
    ]

    /**
     * Different host identifier types listed in the drop down.
     */
    hostIdTypes: SelectItem[] = [
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

    /**
     * Different identifier input formats listed in the drop down.
     */
    public hostIdFormats = [
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
     * Constructor.
     *
     * @param _formBuilder private form builder instance.
     */
    constructor(private _formBuilder: FormBuilder) {}

    /**
     * Component lifecycle hook invoked during initialization.
     *
     * If the provided form instance has been preserved in the parent
     * component this instance is used and the initialization skipped.
     * Otherwise, the form is initialized to defaults.
     */
    ngOnInit() {
        // Initialize the form instance if the parent hasn't supplied one.
        if (!this.form) {
            this.form = new HostForm()
        }
        // Check if the form has been already edited and preserved in the
        // parent component. If so, use it. The user will continue making
        // edits.
        if (this.form.preserved) {
            return
        }
        // New form.
        this.formGroup = this._formBuilder.group({
            globalReservation: [false],
            selectedServers: [],
            selectedSubnet: [],
            hostIdGroup: this._formBuilder.group({
                idType: ['hw-address'],
                idInputHex: ['', StorkValidators.hexIdentifier()],
                idInputText: [''],
                idFormat: ['hex'],
            }),
            ipGroups: this._formBuilder.array([this._createNewIPGroup()]),
            hostname: [''],
        })
        // Initially, list all daemons.
        this.form.filteredDaemons = this.daemons
        // Initially, show all subnets.
        this.form.filteredSubnets = this.subnets
        // By default, we select the IPv4 address option in the drop down.
        // We need to disable adding more IP reservations. It is only allowed
        // in the IPv6 case.
        this.form.addIPButtonDisabled = true
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
    ngOnDestroy() {
        this.form.preserved = true
        this.onFormChange.emit(this.form)
    }

    /**
     * Returns main form group for the component.
     *
     * @returns form group.
     */
    get formGroup(): FormGroup {
        return this.form.group
    }

    /**
     * Sets main form group for the component.
     *
     * @param fg new form group.
     */
    set formGroup(fg: FormGroup) {
        this.form.group = fg
    }

    /**
     * Adds new IP address or delegated prefix input box to the form.
     *
     * By default, the input box is for an IPv4 reservation. However, if
     * there is another box already (only possible in the IPv6 case), the
     * new box uses the type of the last box.
     */
    addIPInput() {
        let ipType = 'ipv4'
        // Check if some IP input boxes have been already added.
        if (this.ipGroups.length > 0) {
            // Some input boxes already exist. Use the last one's type
            // as a default.
            ipType = this.ipGroups.at(this.ipGroups.length - 1).get('ipType').value
        }
        this.ipGroups.push(this._createNewIPGroup(ipType))
        // Refresh the button's disabled flag and the selectable IP types
        // accordingly.
        this.ipTypeChange()
    }

    /**
     * Deletes specified IP address or delegated prefix input box from
     * the form.
     *
     * @param index input box index beginning from 0.
     */
    deleteIPInput(index) {
        ;(this.formGroup.get('ipGroups') as FormArray).removeAt(index)
        this.ipTypeChange()
    }

    /**
     * A function invoked upon selecting an IP reservation type from the
     * drop down or adding or deleting an IP reservation input box.
     *
     * It adjusts the disabled flag for the button adding new IP reservations
     * and a list of selectable IP reservation types.
     */
    ipTypeChange() {
        const ipGroups = this.formGroup.get('ipGroups') as FormArray
        this.form.addIPButtonDisabled = ipGroups.getRawValue().find((group) => group.ipType === 'ipv4')
        this.form.ipv6OnlyTypes =
            ipGroups.length > 1 &&
            ipGroups.getRawValue().find((group) => group.ipType === 'ia_na' || group.ipType === 'ia_pd')
    }

    /**
     * Convenience function returning the form array with IP reservations.
     *
     * @returns form array with IP reservations.
     */
    get ipGroups(): FormArray {
        return this.formGroup.get('ipGroups') as FormArray
    }

    /**
     * Creates new form group for specifying new IP reservation.
     *
     * @param defaultType IP reservation type.
     * @returns form group for specifying new IP reservation.
     */
    private _createNewIPGroup(defaultType = 'ipv4'): FormGroup {
        return this._formBuilder.group({
            ipType: [defaultType],
            inputIPv4: ['', StorkValidators.ipv4()],
            inputNA: ['', StorkValidators.ipv6()],
            inputPD: ['', StorkValidators.ipv6()],
            inputPDLength: ['64'],
        })
    }

    /**
     * A callback invoked when selected servers have changed.
     *
     * Servers selection affects available subnets. If no servers are selected,
     * all subnets are listed for selection. However, if one or more servers
     * are selected only those subnets served by all selected servers are
     * listed. In that case, each listed subnet must be served by all selected
     * servers. If selected servers have no common subnets, no subnets are
     * listed.
     */
    onServersChange() {
        const selectedServers = this.formGroup.get('selectedServers').value
        // We take short path when no servers are selected. Just make all
        // subnets available.
        if (selectedServers.length === 0) {
            this.form.filteredSubnets = this.subnets
            return
        }
        // Filter subnets.
        this.form.filteredSubnets = this.subnets.filter((s) => {
            // We will be filtering by daemonId, so we need to look into
            // the localSubnet.
            return s.localSubnets.some((ls) => {
                return (
                    // At least one daemonId in the subnet should belong to
                    // the array of our selected servers AND each selected
                    // server must be associated with our subnet.
                    selectedServers.includes(ls.daemonId) > 0 &&
                    selectedServers.every((ss) => s.localSubnets.find((ls2) => ls2.daemonId === ss))
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
    }
}
