import { Component, EventEmitter, Input, OnInit, Output } from '@angular/core'
import {
    AbstractControl,
    AbstractControlOptions,
    AsyncValidatorFn,
    FormArray,
    FormBuilder,
    FormControl,
    FormGroup,
    Validators,
    ValidatorFn,
} from '@angular/forms'
import { v4 as uuidv4 } from 'uuid'
import { MenuItem } from 'primeng/api'
import { LinkedFormGroup } from '../forms/linked-form-group'
import { DhcpOptionField, DhcpOptionFieldFormGroup, DhcpOptionFieldType } from '../forms/dhcp-option-field'
import { DhcpOptionsService } from '../dhcp-options.service'
import { createDefaultDhcpOptionFormGroup } from '../forms/dhcp-option-form'
import { IPType } from '../iptype'
import { StorkValidators } from '../validators'

/**
 * An interface to a DHCP option description.
 *
 * It is used to define a list of standard DHCP options.
 */
interface DHCPOptionListItem {
    label: string
    value: number
}

/**
 * A component providing a form to edit DHCP option information.
 *
 * It provides controls to select a DHCP option from the predefined options
 * or type the option code if it is not predefined. A user can interactively
 * add option fields of different types by selecting them from the dropdown
 * list.
 *
 * If a user adds a sub-option, a new instance of the DhcpOptionSetForm
 * component is created. This instance can hold multiple instances of the
 * DhcpOptionForm component, one for each sub-option.
 *
 * This component uses a custom DhcpOptionFieldFormGroup class (instead of
 * the FormGroup) to associate option fields with their types. It is
 * important for correct interpretation of the data specified by the user.
 */
@Component({
    selector: 'app-dhcp-option-form',
    templateUrl: './dhcp-option-form.component.html',
    styleUrls: ['./dhcp-option-form.component.sass'],
})
export class DhcpOptionFormComponent implements OnInit {
    /**
     * Sets the options universe: DHCPv4 or DHCPv6.
     */
    @Input() v6 = false

    /**
     * An empty form group instance created by the parent component.
     */
    @Input() formGroup: FormGroup

    /**
     * A form group index within the array of option form groups.
     *
     * Suppose the parent component maintains an array of form groups,
     * each form group for configuring one option. This number holds
     * a position of this form group within this array. It is important
     * in exchanging the events with the parent to indicate which form
     * group the event pertains to.
     */
    @Input() formIndex: number

    /**
     * Nesting level of this component.
     *
     * It is set to 0 for top-level options. It is set to 1 for a
     * sub-option belonging to a top-level option, etc.
     */
    @Input() nestLevel = 0

    /**
     * An event emitted when an option should be deleted.
     *
     * The parent component should react to this event by removing the
     * form group from the form array. The event contains the index of
     * the form group to remove.
     */
    @Output() optionDelete = new EventEmitter<number>()

    /**
     * An enum indicating an option field type accessible from the HTML template.
     */
    FieldType = DhcpOptionFieldType

    /**
     * Holds a list of selectable option field types.
     */
    fieldTypes: MenuItem[] = []

    /**
     * Holds a reference to the last executed command for adding a new option field.
     */
    lastFieldCommand: () => void

    /**
     * Holds a last added field type.
     */
    lastFieldType: string = ''

    /**
     * A unique id of the option selection dropdown or input box.
     */
    codeInputId: string

    /**
     * A unique id of the Always Send checkbox.
     */
    alwaysSendCheckboxId: string

    /**
     * Constructor.
     *
     * @param _formBuilder a form builder instance used in this component.
     * @param optionsService a service exposing a list of standard DHCP options
     *                       to configure.
     */
    constructor(private _formBuilder: FormBuilder, public optionsService: DhcpOptionsService) {}

    /**
     * A component lifecycle hook called on component initialization.
     *
     * It initializes the list of selectable option fields and associates
     * their selection with appropriate handler functions.
     */
    ngOnInit(): void {
        this.lastFieldType = 'hex-bytes'
        this.lastFieldCommand = this.addHexBytesField
        this.codeInputId = uuidv4()
        this.alwaysSendCheckboxId = uuidv4()
        this.fieldTypes = [
            {
                label: 'hex-bytes',
                id: this.FieldType.HexBytes,
                command: () => {
                    this.lastFieldType = 'hex-bytes'
                    this.lastFieldCommand = this.addHexBytesField
                    this.addHexBytesField()
                },
            },
            {
                label: 'string',
                id: this.FieldType.String,
                command: () => {
                    this.lastFieldType = 'string'
                    this.lastFieldCommand = this.addStringField
                    this.addStringField()
                },
            },
            {
                label: 'bool',
                id: this.FieldType.Bool,
                command: () => {
                    this.lastFieldType = 'bool'
                    this.lastFieldCommand = this.addBoolField
                    this.addBoolField()
                },
            },
            {
                label: 'uint8',
                id: this.FieldType.Uint8,
                command: () => {
                    this.lastFieldType = 'uint8'
                    this.lastFieldCommand = this.addUint8Field
                    this.addUint8Field()
                },
            },
            {
                label: 'uint16',
                id: this.FieldType.Uint16,
                command: () => {
                    this.lastFieldType = 'uint16'
                    this.lastFieldCommand = this.addUint16Field
                    this.addUint16Field()
                },
            },
            {
                label: 'uint32',
                id: this.FieldType.Uint32,
                command: () => {
                    this.lastFieldType = 'uint32'
                    this.lastFieldCommand = this.addUint32Field
                    this.addUint32Field()
                },
            },
            {
                label: 'ipv4-address',
                id: this.FieldType.IPv4Address,
                command: () => {
                    this.lastFieldType = 'ipv4-address'
                    this.lastFieldCommand = this.addIPv4AddressField
                    this.addIPv4AddressField()
                },
            },
            {
                label: 'ipv6-address',
                id: this.FieldType.IPv6Address,
                command: () => {
                    this.lastFieldType = 'ipv6-address'
                    this.lastFieldCommand = this.addIPv6AddressField
                    this.addIPv6AddressField()
                },
            },
            {
                label: 'ipv6-prefix',
                id: this.FieldType.IPv6Prefix,
                command: () => {
                    this.lastFieldType = 'ipv6-prefix'
                    this.lastFieldCommand = this.addIPv6PrefixField
                    this.addIPv6PrefixField()
                },
            },
            {
                label: 'psid',
                id: this.FieldType.Psid,
                command: () => {
                    this.lastFieldType = 'psid'
                    this.lastFieldCommand = this.addPsidField
                    this.addPsidField()
                },
            },
            {
                label: 'fqdn',
                id: this.FieldType.Fqdn,
                command: () => {
                    this.lastFieldType = 'fqdn'
                    this.lastFieldCommand = this.addFqdnField
                    this.addFqdnField()
                },
            },
        ]

        // Currently we support only one nesting level. It is not possible
        // to add a second-level sub-option.
        if (this.topLevel) {
            this.fieldTypes.push({
                label: 'suboption',
                id: this.FieldType.Suboption,
                command: () => {
                    this.lastFieldCommand = this.addSuboption
                    this.addSuboption()
                },
            })
        }
    }

    /**
     * Convenience function returning a form array with option fields.
     */
    get optionFields(): FormArray {
        return this.formGroup.get('optionFields') as FormArray
    }

    /**
     * Convenience function returning a form array with sub-options.
     */
    get suboptions(): FormArray {
        return this.formGroup.get('suboptions') as FormArray
    }

    /**
     * Convenience function checking if it is a top-level option.
     *
     * @returns true if this is a top level option.
     */
    get topLevel(): boolean {
        return this.nestLevel === 0
    }

    /**
     * Adds a single control to the array of the option fields.
     *
     * It is called for option fields which come with a single control,
     * e.g. a string option fields.
     *
     * @param fieldType DHCP option field type.
     * @param control a control associated with the option field.
     */
    private _addSimpleField(fieldType: DhcpOptionFieldType, control: FormControl): void {
        this.optionFields.push(new DhcpOptionFieldFormGroup(fieldType, { control: control }))
    }

    /**
     * Adds multiple controls to the array of the option fields.
     *
     * it is called for the option fields which require multiple controls,
     * e.g. delegated prefix option field requires an input for the prefix
     * and another input for the prefix length.
     *
     * @param fieldType DHCP option field type.
     * @param controls the controls associated with the option field.
     */
    private _addComplexField(fieldType: DhcpOptionFieldType, controls: { [key: string]: AbstractControl }): void {
        this.optionFields.push(new DhcpOptionFieldFormGroup(fieldType, controls))
    }

    /**
     * Adds a control for option field specified in hex-bytes format.
     */
    addHexBytesField(): void {
        this._addSimpleField(this.FieldType.HexBytes, this._formBuilder.control('', StorkValidators.hexIdentifier()))
    }

    /**
     * Adds a control for option field specified as a string.
     */
    addStringField(): void {
        this._addSimpleField(this.FieldType.String, this._formBuilder.control('', Validators.required))
    }

    /**
     * Adds a control for option field specified as a boolean.
     */
    addBoolField(): void {
        this._addSimpleField(this.FieldType.Bool, this._formBuilder.control(''))
    }

    /**
     * Adds a control for option field specified as uint8.
     */
    addUint8Field(): void {
        this._addSimpleField(this.FieldType.Uint8, this._formBuilder.control(null, Validators.required))
    }

    /**
     * Adds a control for option field specified as uint16.
     */
    addUint16Field(): void {
        this._addSimpleField(this.FieldType.Uint16, this._formBuilder.control(null, Validators.required))
    }

    /**
     * Adds a control for option field specified as uint32.
     */
    addUint32Field(): void {
        this._addSimpleField(this.FieldType.Uint32, this._formBuilder.control(null, Validators.required))
    }

    /**
     * Adds a control for option field containing an IPv4 address.
     */
    addIPv4AddressField(): void {
        this._addSimpleField(this.FieldType.IPv4Address, this._formBuilder.control('', StorkValidators.ipv4()))
    }

    /**
     * Adds a control for option field containing an IPv6 address.
     */
    addIPv6AddressField(): void {
        this._addSimpleField(this.FieldType.IPv6Address, this._formBuilder.control('', StorkValidators.ipv6()))
    }

    /**
     * Adds controls for option field containing an IPv6 prefix.
     */
    addIPv6PrefixField(): void {
        this._addComplexField(this.FieldType.IPv6Prefix, {
            prefix: this._formBuilder.control('', StorkValidators.ipv6()),
            prefixLength: this._formBuilder.control('64', Validators.required),
        })
    }

    /**
     * Adds controls for option field containing a PSID.
     */
    addPsidField(): void {
        this._addComplexField(this.FieldType.Psid, {
            psid: this._formBuilder.control(null, Validators.required),
            psidLength: this._formBuilder.control(null, Validators.required),
        })
    }

    /**
     * Adds a control for option field containing an FQDN.
     */
    addFqdnField(): void {
        this._addSimpleField(
            this.FieldType.Fqdn,
            this._formBuilder.control('', [Validators.required, StorkValidators.fullFqdn])
        )
    }

    /**
     * Initializes a new sub-option in the current option.
     */
    addSuboption(): void {
        this.suboptions.push(createDefaultDhcpOptionFormGroup(this.v6 ? IPType.IPv6 : IPType.IPv4))
    }

    /**
     * Notifies the parent component to delete current option from the form.
     */
    deleteOption(): void {
        this.optionDelete.emit(this.formIndex)
    }

    /**
     * Removes option field from the current option.
     *
     * @param index index of an option to remove.
     */
    deleteField(index: number): void {
        this.optionFields.removeAt(index)
    }

    /**
     * Toggles between fully qualified and partial name.
     *
     * @param event an event indicating whether the partial FQDN was selected.
     * @param index option field index.
     */
    togglePartialFqdn(event, index: number) {
        if (event.checked) {
            // Selected partial FQDN.
            this.optionFields.at(index).get('control').setValidators([Validators.required, StorkValidators.partialFqdn])
        } else {
            // Selected non-partial FQDN.
            this.optionFields.at(index).get('control').setValidators([Validators.required, StorkValidators.fullFqdn])
        }
        this.optionFields.at(index).get('control').updateValueAndValidity()
    }
}
