import { Component, Input, OnInit } from '@angular/core'
import { FormGroup, UntypedFormArray, UntypedFormControl } from '@angular/forms'
import { v4 as uuidv4 } from 'uuid'
import { AddressPoolForm, KeaPoolParametersForm, SubnetSetFormService } from '../forms/subnet-set-form.service'
import { SelectableDaemon } from '../forms/selectable-daemon'
import { createDefaultDhcpOptionFormGroup } from '../forms/dhcp-option-form'
import { IPType } from '../iptype'
import { getSeverityByIndex } from '../utils'

/**
 * A component providing a form for editing and adding an address pool.
 */
@Component({
    selector: 'app-address-pool-form',
    templateUrl: './address-pool-form.component.html',
    styleUrls: ['./address-pool-form.component.sass'],
})
export class AddressPoolFormComponent implements OnInit {
    /**
     * Subnet prefix.
     */
    @Input() subnet: string

    /**
     * An array of daemons that can be associated with a pool.
     */
    @Input() selectableDaemons: SelectableDaemon[]

    /**
     * Form group holding address pool data.
     */
    @Input() formGroup: FormGroup<AddressPoolForm>

    /**
     * An array of server names associated with the address pool.
     */
    servers: string[] = []

    /**
     * UUIDS used as unique element identifiers.
     */
    uuids = {
        poolStart: uuidv4(),
        poolEnd: uuidv4(),
        selectedDaemons: uuidv4(),
    }

    /**
     * Constructor.
     *
     * @param subnetSetFormService a service providing form conversion functions.
     */
    constructor(private subnetSetFormService: SubnetSetFormService) {}

    /**
     * A component lifecycle hook invoked when the component is initialized.
     *
     * It initializes the server names using the set of selected daemons in the form.
     */
    ngOnInit(): void {
        const selectedDaemons = this.formGroup.get('selectedDaemons').value ?? []
        if (selectedDaemons.length > 0) {
            this.servers = selectedDaemons.map(
                (sd) => this.selectableDaemons.find((d) => d.id === sd)?.label ?? 'unknown'
            )
        }
    }

    /**
     * Checks if the pool belongs to an IPv6 subnet.
     *
     * @returns true if the subnet belongs to an IPv6 subnet, false otherwise.
     */
    get v6(): boolean {
        return this.subnet.includes(':')
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
    handleDaemonsChange(toggledDaemonId?: number): void {
        const toggledDaemonIndex = toggledDaemonId
            ? this.selectableDaemons.findIndex((fd) => fd.id === toggledDaemonId)
            : -1
        // Selecting new daemons may have a large impact on the data already
        // inserted to the form. Update the form state accordingly and see
        // if it is breaking change.
        const selectedDaemons = this.formGroup.get('selectedDaemons').value ?? []
        if (selectedDaemons.length === 0) {
            // The breaking change puts us at risk of having irrelevant form contents.
            this.resetOptionsArray()
            this.resetParametersArray()
        } else {
            this.subnetSetFormService.adjustFormForSelectedDaemons(
                this.formGroup,
                toggledDaemonIndex,
                this.servers.length
            )
        }
        // If the number of selected daemons has changed, we must update the selected servers list.
        this.servers = selectedDaemons.map((sd) => this.selectableDaemons.find((d) => d.id === sd)?.label ?? 'unknown')
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
     * Resets the part of the form comprising assorted DHCP parameters.
     *
     * It removes all existing controls and re-creates the default one.
     */
    private resetParametersArray() {
        let parameters = this.formGroup.get('parameters') as FormGroup<KeaPoolParametersForm>
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
        this.formGroup.setControl('parameters', this.subnetSetFormService.createDefaultKeaPoolParametersForm())
    }

    /**
     * Resets the part of the form comprising DHCP options.
     *
     * It removes all existing option sets and re-creates the default one.
     */
    private resetOptionsArray() {
        this.formGroup.get('options.unlocked').setValue(false)
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
        return index == null
            ? (this.formGroup.get('options.data') as UntypedFormArray)
            : (this.getOptionsData().at(index) as UntypedFormArray)
    }

    /**
     * A function called when a user clicked to add a new option form.
     *
     * It creates a new default form group for the option.
     *
     * @param index server index in the {@link servers} array.
     */
    onOptionAdd(index: number): void {
        this.getOptionsData(index).push(createDefaultDhcpOptionFormGroup(this.v6 ? IPType.IPv6 : IPType.IPv4))
    }
}
