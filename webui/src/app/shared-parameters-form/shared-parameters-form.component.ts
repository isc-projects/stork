import { Component, Input } from '@angular/core'
import { AbstractControl, FormArray, FormGroup, UntypedFormGroup } from '@angular/forms'
import { getSeverityByIndex, uncamelCase } from '../utils'
import { SelectableClientClass } from '../forms/selectable-client-class'

/**
 * A component providing a form for editing configuration parameters.
 *
 * This component is useful when it is desired to edit configuration
 * parameters for all servers or for each server individually. A Kea
 * subnet configuration is a good example. The subnet configuration can
 * be shared by multiple Kea servers. Typically, the servers use consistent
 * subnet configurations, but some selected values may differ. This
 * form allows for editing and applying the same values for all servers
 * with optionally unlocking individual values for different servers.
 *
 * The form supports primitive types of parameters (i.e., string, number
 * and boolean), and one array type (i.e., string[]) which should only
 * be used for specifying DHCP client classes. Complex parameter types
 * are not supported by this component.
 */
@Component({
    selector: 'app-shared-parameters-form',
    templateUrl: './shared-parameters-form.component.html',
    styleUrls: ['./shared-parameters-form.component.sass'],
})
export class SharedParametersFormComponent<T extends { [K in keyof T]: AbstractControl<any, any> }> {
    /**
     * An array of server names for which the form is presented.
     *
     * The size of this array must match the number of the form
     * controls for each parameter in the form group. The server
     * names from this array as presented in the tags next to the
     * parameters' input elements.
     */
    @Input() servers: string[]

    /**
     * The form group including all parameters initialized.
     *
     * Use the {@link SubnetSetForm} to instantiate the form.
     */
    @Input() formGroup: FormGroup<T> = null

    /**
     * A list of selectable client classes.
     */
    @Input() clientClasses: SelectableClientClass[]

    /**
     * Returns the names of all parameters in the form group.
     */
    get parameterNames(): string[] {
        let names: string[] = []
        if (!this.formGroup || !this.formGroup.controls) {
            return names
        }
        for (let key of Object.keys(this.formGroup?.controls)) {
            names.push(key)
        }
        return names.sort()
    }

    /**
     * Returns a form group comprising the metadata and controls pertaining
     * to a given parameter.
     *
     * @param parameterName parameter name.
     * @returns A form group comprising the metadata and form controls.
     */
    getParameterFormControls(parameterName: string): UntypedFormGroup {
        return this.formGroup.get(parameterName) as UntypedFormGroup
    }

    /**
     * Checks if different controls for the parameters contain different values.
     *
     * It is used to detect if the parameter should be unlocked for editing
     * different values for different servers.
     *
     * @param parameterName parameter name.
     * @returns `true` if the parameter has different values, `false` otherwise.
     */
    hasDifferentValues(parameterName: string): boolean {
        let controls = this.getParameterFormControls(parameterName)
        return (
            controls &&
            (controls.get('values') as FormArray).controls.length > 1 &&
            (controls.get('values') as FormArray).controls
                .slice(1)
                .some((c) => c.value != (controls.get('values') as FormArray).controls[0].value)
        )
    }

    /**
     * Returns severity of a tag associating an input for a server.
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
     * Converts parameter names from camel case to long names.
     *
     * @param parameterName a name to be converted in camel case notation.
     * @returns converted name.
     */
    uncamelCase(parameterName: string): string {
        return uncamelCase(parameterName)
    }
}
