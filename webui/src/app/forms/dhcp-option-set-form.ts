import { FormArray, FormGroup } from '@angular/forms'
import { DhcpOptionFieldFormGroup, DhcpOptionFieldType } from './dhcp-option-field'
import { Universe } from '../universe'

/**
 * A class processing DHCP options forms.
 *
 * The main purpose of this class is to extract the DHCP options
 * from the FormArray object. The FormArray must contain a
 * collection of the DhcpOptionFieldFormGroup objects, each
 * representing a single option. These options may contain
 * suboptions.
 *
 * @todo Extend this class to detect option definitions from the
 * specified options.
 */
export class DhcpOptionSetForm {
    /**
     * Extracted DHCP options into the REST API format.
     */
    private _serializedOptions: any[]

    /**
     * Constructor.
     *
     * @param _formArray input form array holding DHCP options.
     */
    constructor(private _formArray: FormArray) {}

    /**
     * DHCP options form processing implementation.
     *
     * @param universe options universe (i.e., IPv4 or IPv6).
     * @param nestingLevel nesting level of the currently processed options.
     * Its value is 0 for top-level options, 1 for top-level option suboptions etc.
     * @param optionSpace option space encapsulated by a parent option.
     * @throw An error for nesting level higher than 1 or if option data is invalid
     * or missing.
     */
    private _process(universe: Universe, nestingLevel: number, optionSpace: string = '') {
        // To avoid too much recursion, we only parse first level of suboptions.
        if (this._formArray.length > 0 && nestingLevel > 1) {
            throw new Error('options serialization supports up to two nesting levels')
        }
        const serialized = []
        for (let o of this._formArray.controls) {
            const option = o as FormGroup
            // Option code is mandatory.
            if (!option.contains('optionCode') || !option.get('optionCode').value) {
                throw new Error('form group does not contain control with an option code')
            }
            const item = {
                alwaysSend: option.get('alwaysSend').value,
                code: option.get('optionCode').value,
                encapsulate: '',
                fields: [],
                universe: universe,
                options: [],
            }
            const optionFieldsArray = option.get('optionFields') as FormArray
            // Option fields are not mandatory. It is possible to have an empty option.
            if (optionFieldsArray) {
                for (const f of optionFieldsArray.controls) {
                    const field = f as DhcpOptionFieldFormGroup
                    let values: any[]
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
            const suboptions = option.get('suboptions') as FormArray
            // Suboptions are not mandatory.
            if (suboptions && suboptions.length > 0) {
                item.encapsulate = optionSpace.length > 0 ? `${optionSpace}.${item.code}` : `option-${item.code}`
                const suboptionsForm = new DhcpOptionSetForm(suboptions)
                suboptionsForm._process(universe, nestingLevel + 1, item.encapsulate)
                item.options = suboptionsForm.getSerializedOptions()
            }
            // Done extracting an option.
            serialized.push(item)
        }
        // All options extracted. Save the result.
        this._serializedOptions = serialized
    }

    /**
     * Processes top-level DHCP options with their suboptions.
     *
     * The result of the processing can be retrieved with getSerializedOptions().
     *
     * @param universe options universe (i.e., IPv4 or IPv6).
     */
    process(universe: Universe) {
        this._process(universe, 0)
    }

    /**
     * Returns serialized DHCP options.
     *
     * @returns serialized options (in the REST API format).
     * @throws an error when process() function hasn't been called.
     */
    getSerializedOptions(): any[] {
        if (!this._serializedOptions) {
            throw new Error('options form has not been processed')
        }
        return this._serializedOptions
    }
}
