import { AbstractControl, AbstractControlOptions, AsyncValidatorFn, ValidatorFn } from '@angular/forms'
import { v4 as uuidv4 } from 'uuid'
import { LinkedFormGroup } from './linked-form-group'

/**
 * An emum holding all supported DHCP option field types.
 */
export enum DhcpOptionFieldType {
    Binary = 'binary',
    String = 'string',
    Bool = 'bool',
    Uint8 = 'uint8',
    Uint16 = 'uint16',
    Uint32 = 'uint32',
    Int8 = 'int8',
    Int16 = 'int16',
    Int32 = 'int32',
    IPv4Address = 'ipv4-address',
    IPv6Address = 'ipv6-address',
    IPv6Prefix = 'ipv6-prefix',
    Psid = 'psid',
    Fqdn = 'fqdn',
    Suboption = 'suboption',
}

/**
 * Holds a description of the DHCP option field.
 *
 * It includes an option field type and the generated unique identifiers
 * for the form controls.
 */
export class DhcpOptionField {
    /**
     * Option field type.
     */
    fieldType: DhcpOptionFieldType

    /**
     * Generated input identifiers for the controls.
     */
    private _inputIds: string[] = []

    /**
     * Constructor.
     *
     * @param fieldType option field type.
     * @param inputNum a number of controls for which identifiers should be
     * generated.
     */
    constructor(fieldType: DhcpOptionFieldType, inputNum: number) {
        this.fieldType = fieldType
        for (let i = 0; i < inputNum; i++) {
            // Generate UUID identifiers.
            this._inputIds.push(uuidv4())
        }
    }

    /**
     * Returns selected input identifier.
     *
     * @param index control index for which an identifier should be returned.
     * @returns A control identifier.
     */
    getInputId(index: number): string {
        return this._inputIds[index]
    }
}

/**
 * Represents a form group for DHCP option fields.
 */
export class DhcpOptionFieldFormGroup extends LinkedFormGroup<DhcpOptionField> {
    /**
     * Constructor.
     *
     * @param optionFieldType option field type.
     * @param controls form controls belonging to the form group.
     * @param validatorOrOpts validators or control options.
     * @param asyncValidator asynchronous validators.
     */
    constructor(
        optionFieldType: DhcpOptionFieldType,
        controls: { [key: string]: AbstractControl },
        validatorOrOpts?: ValidatorFn | AbstractControlOptions | ValidatorFn[],
        asyncValidator?: AsyncValidatorFn | AsyncValidatorFn[]
    ) {
        let data = new DhcpOptionField(optionFieldType, Object.keys(controls).length)
        super(data, controls, validatorOrOpts, asyncValidator)
    }
}
