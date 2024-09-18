import { Injectable } from '@angular/core'
import { AbstractControl, UntypedFormArray, UntypedFormControl, UntypedFormGroup, Validators } from '@angular/forms'
import { createDefaultDhcpOptionFormGroup } from './dhcp-option-form'
import { DhcpOptionFieldFormGroup, DhcpOptionFieldType } from './dhcp-option-field'
import { IPType } from '../iptype'
import { DHCPOption } from '../backend/model/dHCPOption'
import { DHCPOptionField } from '../backend/model/dHCPOptionField'
import { StorkValidators } from '../validators'
import { FormProcessor } from './form-processor'

/**
 * A service for converting reactive forms with DHCP options to the REST API
 * format and vice-versa.
 */
@Injectable({
    providedIn: 'root',
})
export class DhcpOptionSetFormService extends FormProcessor {
    /**
     * Constructor.
     *
     * Creates form builder instance.
     */
    constructor() {
        super()
    }

    /**
     * Performs deep copy of the form array holding DHCP options or its fragment.
     *
     * It copies all controls, including DhcpOptionFieldFormGroup, with their
     * validators. Controls belonging to forms or arrays are copied recursively.
     *
     * It extends the generic function implemented in the base class with the
     * support to copy DhcpOptionFieldFormGroup instances.
     *
     * @param control top-level control to be copied.
     * @returns copied control instance.
     */
    public cloneControl<T extends AbstractControl>(control: T): T {
        let newControl: T

        if (control instanceof DhcpOptionFieldFormGroup) {
            const formGroup = new DhcpOptionFieldFormGroup(
                (control as DhcpOptionFieldFormGroup).data.fieldType,
                {},
                control.validator,
                control.asyncValidator
            )

            const controls = control.controls

            Object.keys(controls).forEach((key) => {
                formGroup.addControl(key, this.cloneControl(controls[key]))
            })

            newControl = formGroup as any

            if (control.disabled) {
                newControl.disable({ emitEvent: false })
            }
            return newControl
        }

        return super.cloneControl(control)
    }

    /**
     * Implements conversion of the DHCP options from the reactive form to
     * the REST API format.
     *
     * @param universe options universe (i.e., IPv4 or IPv6).
     * @param nestingLevel nesting level of the currently processed options.
     * @param formArray form array containing the options.
     * @returns options in the REST API format.
     * @throw An error for nesting level higher than 2 or if option data is invalid
     * or missing.
     */
    private _convertFormToOptions(
        universe: IPType,
        formArray: UntypedFormArray,
        nestingLevel: number,
        optionSpace?: string
    ): Array<DHCPOption> {
        // To avoid too much recursion, we only parse first and second level of suboptions.
        if (formArray.length > 0 && nestingLevel > 2) {
            throw new Error('options serialization supports up to two nesting levels')
        }
        let serialized = new Array<DHCPOption>()
        for (let o of formArray.controls) {
            const option = o as UntypedFormGroup
            // Option code is mandatory.
            if (!option.contains('optionCode') || option.get('optionCode').value === null) {
                throw new Error('form group does not contain control with an option code')
            }
            let optionCode = 0
            if (typeof option.get('optionCode').value === 'string') {
                optionCode = parseInt(option.get('optionCode').value, 10)
                if (isNaN(optionCode)) {
                    throw new Error(`specified option code ${option.get('optionCode').value} is not a valid number`)
                }
            } else {
                optionCode = option.get('optionCode').value
            }
            const item: DHCPOption = {
                alwaysSend: option.get('alwaysSend').value,
                code: optionCode,
                encapsulate: '',
                fields: new Array<DHCPOptionField>(),
                universe: universe,
                options: new Array<DHCPOption>(),
            }
            const optionFieldsArray = option.get('optionFields') as UntypedFormArray
            // Option fields are not mandatory. It is possible to have an empty option.
            if (optionFieldsArray) {
                for (const f of optionFieldsArray.controls) {
                    const field = f as DhcpOptionFieldFormGroup
                    let values: Array<string> = []
                    switch (field.data.fieldType) {
                        case DhcpOptionFieldType.Bool:
                            if (!field.contains('control')) {
                                throw new Error(field.data.fieldType + ' option field must contain control')
                            }
                            let value = field.get('control').value.toString()
                            if (value.length === 0) {
                                value = 'false'
                            }
                            values = [value]
                            break
                        case DhcpOptionFieldType.IPv6Prefix:
                            // IPv6 prefix field contains a prefix and length.
                            if (!field.contains('prefix') || !field.contains('prefixLength')) {
                                throw new Error(
                                    'IPv6 prefix option field must contain prefix and prefixLength controls'
                                )
                            }
                            values = [field.get('prefix').value.trim(), field.get('prefixLength').value.toString()]
                            break
                        case DhcpOptionFieldType.Psid:
                            // PSID field contains PSID and PSID length.
                            if (!field.contains('psid') || !field.contains('psidLength')) {
                                throw new Error('psid option field must contain psid and psidLength controls')
                            }
                            values = [field.get('psid').value.toString(), field.get('psidLength').value.toString()]
                            break
                        default:
                            // Other fields contain a single value.
                            if (!field.contains('control')) {
                                throw new Error(field.data.fieldType + ' option field must contain control')
                            }
                            values = [field.get('control').value.toString().trim()]
                            break
                    }
                    item.fields.push({
                        fieldType: field.data.fieldType,
                        values: values,
                    })
                }
            }
            const suboptions = option.get('suboptions') as UntypedFormArray
            // Suboptions are not mandatory.
            if (suboptions && suboptions.length > 0) {
                item.encapsulate = optionSpace ? `${optionSpace}.${item.code}` : `option-${item.code}`
                item.options = this._convertFormToOptions(universe, suboptions, nestingLevel + 1, item.encapsulate)
            }
            // Done extracting an option.
            serialized.push(item)
        }
        return serialized
    }

    /**
     * Converts top-level DHCP options with suboptions contained in the reactive form
     * to the REST API format.
     *
     * @param universe options universe (i.e., IPv4 or IPv6).
     * @param formArray form array containing the options.
     * @returns options in the REST API format.
     */
    public convertFormToOptions(universe: IPType, formArray: UntypedFormArray): Array<DHCPOption> {
        return this._convertFormToOptions(universe, formArray, 0)
    }

    /**
     * Implements conversion of the DHCP options from the REST API format to a reactive form.
     *
     * @param universe options universe (i.e., IPv4 or IPv6).
     * @param nestingLevel nesting level of the currently processed options.
     * @param options a set of DHCP options at certain nesting level.
     * @returns form array comprising converted options.
     * @throw an error when parsed option field contain an invalid number of
     * values. Typically, they contain a single value. They contain two values
     * when they are IPv6 prefixes or PSIDs.
     */
    private _convertOptionsToForm(
        universe: IPType,
        nestingLevel: number,
        options: Array<DHCPOption>
    ): UntypedFormArray {
        // To avoid too much recursion, we only convert first and second level of suboptions.
        if (options?.length > 0 && nestingLevel > 2) {
            throw new Error('options serialization supports up to two nesting levels')
        }
        let formArray = this._formBuilder.array([])
        if (!options || options.length === 0) {
            return formArray
        }
        for (let option of options) {
            let optionFormGroup = createDefaultDhcpOptionFormGroup(universe)
            if (!isNaN(option.code)) {
                optionFormGroup.get('optionCode').setValue(option.code)
            }
            if (option.alwaysSend) {
                optionFormGroup.get('alwaysSend').setValue(option.alwaysSend)
            }
            for (let field of option.fields ?? []) {
                // Sanity check option field values.
                if (
                    field.fieldType === DhcpOptionFieldType.IPv6Prefix ||
                    field.fieldType === DhcpOptionFieldType.Psid
                ) {
                    if (field.values.length !== 2) {
                        throw new Error(`expected two option field values for the option field type ${field.fieldType}`)
                    }
                } else if (field.values.length !== 1) {
                    throw new Error(`expected one option field value for the option field type ${field.fieldType}`)
                }
                // For each option field create an appropriate form group.
                let fieldGroup: DhcpOptionFieldFormGroup
                switch (field.fieldType as DhcpOptionFieldType) {
                    case DhcpOptionFieldType.Binary:
                        fieldGroup = this.createBinaryField(field.values[0])
                        break
                    case DhcpOptionFieldType.String:
                        fieldGroup = this.createStringField(field.values[0])
                        break
                    case DhcpOptionFieldType.Bool:
                        fieldGroup = this.createBoolField(field.values[0])
                        break
                    case DhcpOptionFieldType.Uint8:
                        fieldGroup = this.createUint8Field(field.values[0])
                        break
                    case DhcpOptionFieldType.Uint16:
                        fieldGroup = this.createUint16Field(field.values[0])
                        break
                    case DhcpOptionFieldType.Uint32:
                        fieldGroup = this.createUint32Field(field.values[0])
                        break
                    case DhcpOptionFieldType.Int8:
                        fieldGroup = this.createInt8Field(field.values[0])
                        break
                    case DhcpOptionFieldType.Int16:
                        fieldGroup = this.createInt16Field(field.values[0])
                        break
                    case DhcpOptionFieldType.Int32:
                        fieldGroup = this.createInt32Field(field.values[0])
                        break
                    case DhcpOptionFieldType.IPv4Address:
                        fieldGroup = this.createIPv4AddressField(field.values[0])
                        break
                    case DhcpOptionFieldType.IPv6Address:
                        fieldGroup = this.createIPv6AddressField(field.values[0])
                        break
                    case DhcpOptionFieldType.IPv6Prefix:
                        fieldGroup = this.createIPv6PrefixField(field.values[0], field.values[1])
                        break
                    case DhcpOptionFieldType.Psid:
                        fieldGroup = this.createPsidField(field.values[0], field.values[1])
                        break
                    case DhcpOptionFieldType.Fqdn:
                        fieldGroup = this.createFqdnField(field.values[0])
                        break
                    default:
                        continue
                }
                ;(optionFormGroup.get('optionFields') as UntypedFormArray).push(fieldGroup)
            }
            if (option.options?.length > 0) {
                optionFormGroup.setControl(
                    'suboptions',
                    this._convertOptionsToForm(universe, nestingLevel + 1, option.options)
                )
            }
            formArray.push(optionFormGroup)
        }
        return formArray
    }

    /**
     * Converts DHCP options from the REST API format to the reactive form.
     *
     * @param universe options universe (i.e., IPv4 or IPv6).
     * @param options a set of DHCP options at certain nesting level.
     * @returns form array comprising converted options.
     */
    public convertOptionsToForm(universe: IPType, options: Array<DHCPOption>): UntypedFormArray {
        return this._convertOptionsToForm(universe, 0, options)
    }

    /**
     * Creates a form group instance comprising one control representing
     * an option field.
     *
     * It is called for option fields which require a single control,
     * e.g. a string option fields.
     *
     * @param fieldType DHCP option field type.
     * @param control a control associated with the option field.
     * @returns created form group instance.
     */
    private _createSimpleField(fieldType: DhcpOptionFieldType, control: UntypedFormControl): DhcpOptionFieldFormGroup {
        return new DhcpOptionFieldFormGroup(fieldType, { control: control })
    }

    /**
     * Creates a form group instance comprising multiple controls representing
     * an option field.
     *
     * It is called for the option fields which require multiple controls,
     * e.g. delegated prefix option field requires an input for the prefix
     * and another input for the prefix length.
     *
     * @param fieldType DHCP option field type.
     * @param controls the controls associated with the option field.
     * @returns created form group instance.
     */
    private _createComplexField(
        fieldType: DhcpOptionFieldType,
        controls: { [key: string]: AbstractControl }
    ): DhcpOptionFieldFormGroup {
        return new DhcpOptionFieldFormGroup(fieldType, controls)
    }

    /**
     * Creates a control for option field using binary format.
     *
     * @param value option field value to set.
     * @returns created form group instance.
     */
    createBinaryField(value: string = ''): DhcpOptionFieldFormGroup {
        return this._createSimpleField(
            DhcpOptionFieldType.Binary,
            this._formBuilder.control(value, StorkValidators.hexIdentifier())
        )
    }

    /**
     * Creates a control for option field using string format.
     *
     * @param value option field value to set.
     * @returns created form group instance.
     */
    createStringField(value: string = ''): DhcpOptionFieldFormGroup {
        return this._createSimpleField(
            DhcpOptionFieldType.String,
            this._formBuilder.control(value, Validators.required)
        )
    }

    /**
     * Creates a control for option field using boolean format.
     *
     * @param value option field value to set.
     * @returns created form group instance.
     */
    createBoolField(value: string | boolean = false): DhcpOptionFieldFormGroup {
        let boolValue = false
        switch (value) {
            case 'true':
            case 'TRUE':
            case true:
                boolValue = true
                break
            default:
                break
        }
        return this._createSimpleField(DhcpOptionFieldType.Bool, this._formBuilder.control(boolValue))
    }

    /**
     * Creates a control for option field using uint8 format.
     *
     * @param value option field value to set.
     * @returns created form group instance.
     */
    createUint8Field(value: string | number | null = null): DhcpOptionFieldFormGroup {
        return this._createSimpleField(DhcpOptionFieldType.Uint8, this._formBuilder.control(value, Validators.required))
    }

    /**
     * Creates a control for option field using uint16 format.
     *
     * @param value option field value to set.
     * @returns created form group instance.
     */
    createUint16Field(value: string | number | null = null): DhcpOptionFieldFormGroup {
        return this._createSimpleField(
            DhcpOptionFieldType.Uint16,
            this._formBuilder.control(value, Validators.required)
        )
    }

    /**
     * Creates a control for option field using uint32 format.
     *
     * @param value option field value to set.
     * @returns created form group instance.
     */
    createUint32Field(value: string | number | null = null): DhcpOptionFieldFormGroup {
        return this._createSimpleField(
            DhcpOptionFieldType.Uint32,
            this._formBuilder.control(value, Validators.required)
        )
    }

    /**
     * Creates a control for option field using int8 format.
     *
     * @param value option field value to set.
     * @returns created form group instance.
     */
    createInt8Field(value: string | number | null = null): DhcpOptionFieldFormGroup {
        return this._createSimpleField(DhcpOptionFieldType.Int8, this._formBuilder.control(value, Validators.required))
    }

    /**
     * Creates a control for option field using int16 format.
     *
     * @param value option field value to set.
     * @returns created form group instance.
     */
    createInt16Field(value: string | number | null = null): DhcpOptionFieldFormGroup {
        return this._createSimpleField(DhcpOptionFieldType.Int16, this._formBuilder.control(value, Validators.required))
    }

    /**
     * Creates a control for option field using int32 format.
     *
     * @param value option field value to set.
     * @returns created form group instance.
     */
    createInt32Field(value: string | number | null = null): DhcpOptionFieldFormGroup {
        return this._createSimpleField(DhcpOptionFieldType.Int32, this._formBuilder.control(value, Validators.required))
    }

    /**
     * Creates a control for option field using IPv4 address format.
     *
     * @param value option field value to set.
     * @returns created form group instance.
     */
    createIPv4AddressField(value: string = ''): DhcpOptionFieldFormGroup {
        return this._createSimpleField(
            DhcpOptionFieldType.IPv4Address,
            this._formBuilder.control(value, [Validators.required, StorkValidators.ipv4()])
        )
    }

    /**
     * Creates a control for option field using IPv6 address format.
     *
     * @param value option field value to set.
     * @returns created form group instance.
     */
    createIPv6AddressField(value: string = ''): DhcpOptionFieldFormGroup {
        return this._createSimpleField(
            DhcpOptionFieldType.IPv6Address,
            this._formBuilder.control(value, [Validators.required, StorkValidators.ipv6()])
        )
    }

    /**
     * Creates a control for option field using IPv6 prefix format.
     *
     * @param value option field value to set.
     * @returns created form group instance.
     */
    createIPv6PrefixField(prefix: string = '', prefixLen: string | number | null = null): DhcpOptionFieldFormGroup {
        return this._createComplexField(DhcpOptionFieldType.IPv6Prefix, {
            prefix: this._formBuilder.control(prefix, [Validators.required, StorkValidators.ipv6()]),
            prefixLength: this._formBuilder.control(prefixLen, Validators.required),
        })
    }

    /**
     * Creates a control for option field using PSID format.
     *
     * @param value option field value to set.
     * @returns created form group instance.
     */
    createPsidField(
        psid: string | number | null = null,
        psidLen: string | number | null = null
    ): DhcpOptionFieldFormGroup {
        return this._createComplexField(DhcpOptionFieldType.Psid, {
            psid: this._formBuilder.control(psid, Validators.required),
            psidLength: this._formBuilder.control(psidLen, Validators.required),
        })
    }

    /**
     * Creates a control for option field using FQDN format.
     *
     * @param value option field value to set.
     * @returns created form group instance.
     */
    createFqdnField(value: string = ''): DhcpOptionFieldFormGroup {
        let control = this._formBuilder.control(value, [Validators.required, StorkValidators.fullFqdn])
        let isPartialFqdn = false
        if (value != '' && StorkValidators.partialFqdn(control) == null) {
            control.setValidators([Validators.required, StorkValidators.partialFqdn])
            isPartialFqdn = true
        }

        return this._createComplexField(DhcpOptionFieldType.Fqdn, {
            control: control,
            isPartialFqdn: this._formBuilder.control(isPartialFqdn),
        })
    }
}
