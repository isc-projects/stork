import { Component, EventEmitter, Input, OnInit, Output } from '@angular/core'
import { UntypedFormArray, UntypedFormBuilder, UntypedFormGroup, Validators } from '@angular/forms'
import { v4 as uuidv4 } from 'uuid'
import { MenuItem } from 'primeng/api'
import { DhcpOptionFieldFormGroup, DhcpOptionFieldType } from '../forms/dhcp-option-field'
import { DhcpOptionsService } from '../dhcp-options.service'
import { DhcpOptionSetFormService } from '../forms/dhcp-option-set-form.service'
import { createDefaultDhcpOptionFormGroup } from '../forms/dhcp-option-form'
import { IPType } from '../iptype'
import { StorkValidators } from '../validators'
import { DhcpOptionDef } from '../dhcp-option-def'

/**
 * A signature to a function adding a field to the form.
 */
type AddFieldFn = () => void

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
    @Input() formGroup: UntypedFormGroup

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
     * Option space the option belongs to.
     *
     * It is used to find a definition of a selected option.
     */
    @Input() optionSpace = null

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
     * Option definition of a currently selected option.
     *
     * It is null if the option definition doesn't exist for the selected
     * option.
     */
    optionDef: DhcpOptionDef

    /**
     * Constructor.
     *
     * @param _formBuilder a form builder instance used in this component.
     * @param _optionSetFormService a service providing functions to convert
     * options from and to reactive forms.
     * @param optionsService a service exposing a list of standard DHCP options
     * to configure.
     */
    constructor(
        private _formBuilder: UntypedFormBuilder,
        private _optionSetFormService: DhcpOptionSetFormService,
        public optionsService: DhcpOptionsService
    ) {}

    /**
     * Returns a function to be invoked when selected option field type is
     * added to the form.
     *
     * It is invoked in two situations: when user adds an option field with
     * a button or when the default option form is opened using the option
     * definition.
     *
     * @param fieldName option field type name as string.
     * @returns pointer to a function to be invoked to add the option field
     * to the form.
     */
    private _getFieldCommand(fieldName: string): AddFieldFn {
        switch (fieldName) {
            case DhcpOptionFieldType.Binary:
                return () => {
                    this.lastFieldType = DhcpOptionFieldType.Binary
                    this.lastFieldCommand = this.addBinaryField
                    this.addBinaryField()
                }
            case DhcpOptionFieldType.String:
                return () => {
                    this.lastFieldType = DhcpOptionFieldType.String
                    this.lastFieldCommand = this.addStringField
                    this.addStringField()
                }
            case DhcpOptionFieldType.Bool:
                return () => {
                    this.lastFieldType = DhcpOptionFieldType.Bool
                    this.lastFieldCommand = this.addBoolField
                    this.addBoolField()
                }
            case DhcpOptionFieldType.Uint8:
                return () => {
                    this.lastFieldType = DhcpOptionFieldType.Uint8
                    this.lastFieldCommand = this.addUint8Field
                    this.addUint8Field()
                }
            case DhcpOptionFieldType.Uint16:
                return () => {
                    this.lastFieldType = DhcpOptionFieldType.Uint16
                    this.lastFieldCommand = this.addUint16Field
                    this.addUint16Field()
                }
            case DhcpOptionFieldType.Uint32:
                return () => {
                    this.lastFieldType = DhcpOptionFieldType.Uint32
                    this.lastFieldCommand = this.addUint32Field
                    this.addUint32Field()
                }
            case DhcpOptionFieldType.Int8:
                return () => {
                    this.lastFieldType = DhcpOptionFieldType.Int8
                    this.lastFieldCommand = this.addInt8Field
                    this.addInt8Field()
                }
            case DhcpOptionFieldType.Int16:
                return () => {
                    this.lastFieldType = DhcpOptionFieldType.Int16
                    this.lastFieldCommand = this.addInt16Field
                    this.addInt16Field()
                }
            case DhcpOptionFieldType.Int32:
                return () => {
                    this.lastFieldType = DhcpOptionFieldType.Int32
                    this.lastFieldCommand = this.addInt32Field
                    this.addInt32Field()
                }
            case DhcpOptionFieldType.IPv4Address:
                return () => {
                    this.lastFieldType = DhcpOptionFieldType.IPv4Address
                    this.lastFieldCommand = this.addIPv4AddressField
                    this.addIPv4AddressField()
                }
            case DhcpOptionFieldType.IPv6Address:
                return () => {
                    this.lastFieldType = DhcpOptionFieldType.IPv6Address
                    this.lastFieldCommand = this.addIPv6AddressField
                    this.addIPv6AddressField()
                }
            case DhcpOptionFieldType.IPv6Prefix:
                return () => {
                    this.lastFieldType = DhcpOptionFieldType.IPv6Prefix
                    this.lastFieldCommand = this.addIPv6PrefixField
                    this.addIPv6PrefixField()
                }
            case DhcpOptionFieldType.Psid:
                return () => {
                    this.lastFieldType = DhcpOptionFieldType.Psid
                    this.lastFieldCommand = this.addPsidField
                    this.addPsidField()
                }
            case DhcpOptionFieldType.Fqdn:
                return () => {
                    this.lastFieldType = DhcpOptionFieldType.Fqdn
                    this.lastFieldCommand = this.addFqdnField
                    this.addFqdnField()
                }
            default:
                return () => {}
        }
    }

    /**
     * A component lifecycle hook called on component initialization.
     *
     * It initializes the list of selectable option fields and associates
     * their selection with appropriate handler functions.
     */
    ngOnInit(): void {
        this.lastFieldType = DhcpOptionFieldType.Binary
        this.lastFieldCommand = this.addBinaryField
        this.codeInputId = uuidv4()
        this.alwaysSendCheckboxId = uuidv4()
        this.fieldTypes = [
            {
                label: DhcpOptionFieldType.Binary,
                id: this.FieldType.Binary,
                command: this._getFieldCommand(DhcpOptionFieldType.Binary),
            },
            {
                label: DhcpOptionFieldType.String,
                id: this.FieldType.String,
                command: this._getFieldCommand(DhcpOptionFieldType.String),
            },
            {
                label: DhcpOptionFieldType.Bool,
                id: this.FieldType.Bool,
                command: this._getFieldCommand(DhcpOptionFieldType.Bool),
            },
            {
                label: DhcpOptionFieldType.Uint8,
                id: this.FieldType.Uint8,
                command: this._getFieldCommand(DhcpOptionFieldType.Uint8),
            },
            {
                label: DhcpOptionFieldType.Uint16,
                id: this.FieldType.Uint16,
                command: this._getFieldCommand(DhcpOptionFieldType.Uint16),
            },
            {
                label: DhcpOptionFieldType.Uint32,
                id: this.FieldType.Uint32,
                command: this._getFieldCommand(DhcpOptionFieldType.Uint32),
            },
            {
                label: DhcpOptionFieldType.Int8,
                id: this.FieldType.Int8,
                command: this._getFieldCommand(DhcpOptionFieldType.Int8),
            },
            {
                label: DhcpOptionFieldType.Int16,
                id: this.FieldType.Int16,
                command: this._getFieldCommand(DhcpOptionFieldType.Int16),
            },
            {
                label: DhcpOptionFieldType.Int32,
                id: this.FieldType.Int32,
                command: this._getFieldCommand(DhcpOptionFieldType.Int32),
            },
            {
                label: DhcpOptionFieldType.IPv4Address,
                id: this.FieldType.IPv4Address,
                command: this._getFieldCommand(DhcpOptionFieldType.IPv4Address),
            },
            {
                label: DhcpOptionFieldType.IPv6Address,
                id: this.FieldType.IPv6Address,
                command: this._getFieldCommand(DhcpOptionFieldType.IPv6Address),
            },
            {
                label: DhcpOptionFieldType.IPv6Prefix,
                id: this.FieldType.IPv6Prefix,
                command: this._getFieldCommand(DhcpOptionFieldType.IPv6Prefix),
            },
            {
                label: DhcpOptionFieldType.Psid,
                id: this.FieldType.Psid,
                command: this._getFieldCommand(DhcpOptionFieldType.Psid),
            },
            {
                label: DhcpOptionFieldType.Fqdn,
                id: this.FieldType.Fqdn,
                command: this._getFieldCommand(DhcpOptionFieldType.Fqdn),
            },
        ]

        // We support only two nesting levels.
        if (this.nestLevel <= 1) {
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
    get optionFields(): UntypedFormArray {
        return this.formGroup.get('optionFields') as UntypedFormArray
    }

    /**
     * Convenience function returning a form array with sub-options.
     */
    get suboptions(): UntypedFormArray {
        return this.formGroup.get('suboptions') as UntypedFormArray
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
     * Adds a single control the array of the option fields.
     *
     * @param optionField an option field form group.
     */
    private _addField(optionField: DhcpOptionFieldFormGroup) {
        this.optionFields.push(optionField)
    }

    /**
     * Adds a control for option field specified in binary format.
     */
    addBinaryField(): void {
        this._addField(this._optionSetFormService.createBinaryField())
    }

    /**
     * Adds a control for option field specified as a string.
     */
    addStringField(): void {
        this._addField(this._optionSetFormService.createStringField())
    }

    /**
     * Adds a control for option field specified as a boolean.
     */
    addBoolField(): void {
        this._addField(this._optionSetFormService.createBoolField())
    }

    /**
     * Adds a control for option field specified as uint8.
     */
    addUint8Field(): void {
        this._addField(this._optionSetFormService.createUint8Field())
    }

    /**
     * Adds a control for option field specified as uint16.
     */
    addUint16Field(): void {
        this._addField(this._optionSetFormService.createUint16Field())
    }

    /**
     * Adds a control for option field specified as uint32.
     */
    addUint32Field(): void {
        this._addField(this._optionSetFormService.createUint32Field())
    }

    /**
     * Adds a control for option field specified as int8.
     */
    addInt8Field(): void {
        this._addField(this._optionSetFormService.createInt8Field())
    }

    /**
     * Adds a control for option field specified as int16.
     */
    addInt16Field(): void {
        this._addField(this._optionSetFormService.createInt16Field())
    }

    /**
     * Adds a control for option field specified as int32.
     */
    addInt32Field(): void {
        this._addField(this._optionSetFormService.createInt32Field())
    }

    /**
     * Adds a control for option field containing an IPv4 address.
     */
    addIPv4AddressField(): void {
        this._addField(this._optionSetFormService.createIPv4AddressField())
    }

    /**
     * Adds a control for option field containing an IPv6 address.
     */
    addIPv6AddressField(): void {
        this._addField(this._optionSetFormService.createIPv6AddressField())
    }

    /**
     * Adds controls for option field containing an IPv6 prefix.
     */
    addIPv6PrefixField(): void {
        this._addField(this._optionSetFormService.createIPv6PrefixField())
    }

    /**
     * Adds controls for option field containing a PSID.
     */
    addPsidField(): void {
        this._addField(this._optionSetFormService.createPsidField())
    }

    /**
     * Adds a control for option field containing an FQDN.
     */
    addFqdnField(): void {
        this._addField(this._optionSetFormService.createFqdnField())
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

    /**
     * Returns an array of option codes encapsulated by the currently
     * selected option.
     *
     * @returns An array of option codes or an empty array if the option
     * definition doesn't exist.
     */
    getStandardDhcpOptionDefCodes(): Array<number> {
        if (!this.optionDef) {
            return []
        }
        return this.v6
            ? this.optionsService.findStandardDhcpv6OptionDefsBySpace(this.optionDef.encapsulate).map((def) => def.code)
            : this.optionsService.findStandardDhcpv4OptionDefsBySpace(this.optionDef.encapsulate).map((def) => def.code)
    }

    /**
     * A handler called when user selects or types a new option code.
     *
     * The handler clears the existing option fields and suboptions.
     * It locates a corresponding option definition and adds suitable
     * fields to the form based on it. If the option definition is not
     * found, it adds no option fields to the form.
     *
     * @param event an event triggered on option code selection.
     */
    onOptionCodeChange(event) {
        this.optionFields.clear()
        this.suboptions.clear()
        let optionCode = event.value
        this.optionDef = this.v6
            ? this.optionsService.findStandardDhcpv6OptionDef(optionCode, this.optionSpace)
            : this.optionsService.findStandardDhcpv4OptionDef(optionCode, this.optionSpace)
        if (!this.optionDef) {
            return
        }
        if (this.optionDef.optionType === 'record') {
            for (let recordType of this.optionDef.recordTypes) {
                this._getFieldCommand(recordType)()
            }
        } else {
            this._getFieldCommand(this.optionDef.optionType)()
        }
    }
}
