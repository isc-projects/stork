import { Injectable } from '@angular/core'
import { FormControl, FormGroup } from '@angular/forms'
import { SharedParameterFormGroup } from './shared-parameter-form-group'
import { KeaConfigSubnetDerivedParameters } from '../backend'
import { StorkValidators } from '../validators'
import { Validator } from 'ip-num/Validator'

/**
 * A type of the subnet form for editing Kea-specific parameters using
 * the {@link SharedParametersForm} component.
 */
interface KeaSubnetParametersForm {
    cacheMaxAge?: SharedParameterFormGroup<number>
    cacheThreshold?: SharedParameterFormGroup<number>
    clientClass?: SharedParameterFormGroup<string>
    requireClientClasses?: SharedParameterFormGroup<string[]>
    ddnsGeneratedPrefix?: SharedParameterFormGroup<string>
    ddnsOverrideClientUpdate?: SharedParameterFormGroup<boolean>
    ddnsOverrideNoUpdate?: SharedParameterFormGroup<boolean>
    ddnsQualifyingSuffix?: SharedParameterFormGroup<string>
    ddnsReplaceClientName?: SharedParameterFormGroup<string>
    ddnsSendUpdates?: SharedParameterFormGroup<boolean>
    ddnsUpdateOnRenew?: SharedParameterFormGroup<boolean>
    ddnsUseConflictResolution?: SharedParameterFormGroup<boolean>
    fourOverSixInterface?: SharedParameterFormGroup<string>
    fourOverSixInterfaceID?: SharedParameterFormGroup<string>
    fourOverSixSubnet?: SharedParameterFormGroup<string>
    hostnameCharReplacement?: SharedParameterFormGroup<string>
    hostnameCharSet?: SharedParameterFormGroup<string>
    preferredLifetime?: SharedParameterFormGroup<number>
    minPreferredLifetime?: SharedParameterFormGroup<number>
    maxPreferredLifetime?: SharedParameterFormGroup<number>
    reservationsGlobal?: SharedParameterFormGroup<boolean>
    reservationsInSubnet?: SharedParameterFormGroup<boolean>
    reservationsOutOfPool?: SharedParameterFormGroup<boolean>
    renewTimer?: SharedParameterFormGroup<number>
    rebindTimer?: SharedParameterFormGroup<number>
    t1Percent?: SharedParameterFormGroup<number>
    t2Percent?: SharedParameterFormGroup<number>
    calculateTeeTimes?: SharedParameterFormGroup<boolean>
    validLifetime?: SharedParameterFormGroup<number>
    minValidLifetime?: SharedParameterFormGroup<number>
    maxValidLifetime?: SharedParameterFormGroup<number>
    allocator?: SharedParameterFormGroup<string>
    authoritative?: SharedParameterFormGroup<boolean>
    bootFileName?: SharedParameterFormGroup<string>
    interface?: SharedParameterFormGroup<string>
    interfaceID?: SharedParameterFormGroup<string>
    matchClientID?: SharedParameterFormGroup<boolean>
    nextServer?: SharedParameterFormGroup<string>
    pdAllocator?: SharedParameterFormGroup<string>
    rapidCommit?: SharedParameterFormGroup<boolean>
    serverHostname?: SharedParameterFormGroup<string>
    storeExtendedInfo?: SharedParameterFormGroup<boolean>
}

/**
 * A service exposing functions converting subnet data to a form and
 * vice versa.
 */
@Injectable({
    providedIn: 'root',
})
export class SubnetSetFormService {
    /**
     * Empty constructor.
     */
    constructor() {}

    /**
     * Converts Kea subnet parameters to a form.
     *
     * The created form is used in the {@link SharedParametersForm} for editing
     * the subnet parameters. It comprises the metadata describing each parameter.
     *
     * @param parameters Kea-specific subnet parameters.
     * @returns Created form group instance.
     */
    convertKeaParametersToForm(parameters: KeaConfigSubnetDerivedParameters[]): FormGroup<KeaSubnetParametersForm> {
        let formGroup = new FormGroup<KeaSubnetParametersForm>({
            cacheThreshold: new SharedParameterFormGroup<number>(
                {
                    type: 'number',
                    min: 0,
                    max: 1,
                    fractionDigits: 2,
                },
                parameters.map((params) => new FormControl<number>(params.cacheThreshold))
            ),
            cacheMaxAge: new SharedParameterFormGroup<number>(
                {
                    type: 'number',
                },
                parameters.map((params) => new FormControl<number>(params.cacheMaxAge))
            ),
            clientClass: new SharedParameterFormGroup<string>(
                {
                    type: 'string',
                },
                parameters.map((params) => new FormControl<string>(params.clientClass))
            ),
            requireClientClasses: new SharedParameterFormGroup<string[]>(
                {
                    type: 'client-classes',
                },
                parameters.map((params) => new FormControl<string[]>(params.requireClientClasses))
            ),
            ddnsGeneratedPrefix: new SharedParameterFormGroup<string>(
                {
                    type: 'string',
                    invalidText: 'Please specify a valid prefix.',
                },
                parameters.map((params) => new FormControl<string>(params.ddnsGeneratedPrefix, StorkValidators.fqdn))
            ),
            ddnsOverrideClientUpdate: new SharedParameterFormGroup<boolean>(
                {
                    type: 'boolean',
                },
                parameters.map((params) => new FormControl<boolean>(params.ddnsOverrideClientUpdate))
            ),
            ddnsOverrideNoUpdate: new SharedParameterFormGroup<boolean>(
                {
                    type: 'boolean',
                },
                parameters.map((params) => new FormControl<boolean>(params.ddnsOverrideNoUpdate))
            ),
            ddnsQualifyingSuffix: new SharedParameterFormGroup<string>(
                {
                    type: 'string',
                    invalidText: 'Please specify a valid suffix.',
                },
                parameters.map((params) => new FormControl<string>(params.ddnsQualifyingSuffix, StorkValidators.fqdn))
            ),
            ddnsReplaceClientName: new SharedParameterFormGroup<string>(
                {
                    type: 'string',
                    values: ['never', 'always', 'when-not-present'],
                },
                parameters.map((params) => new FormControl<string>(params.ddnsReplaceClientName))
            ),
            ddnsSendUpdates: new SharedParameterFormGroup<boolean>(
                {
                    type: 'boolean',
                },
                parameters.map((params) => new FormControl<boolean>(params.ddnsSendUpdates))
            ),
            ddnsUpdateOnRenew: new SharedParameterFormGroup<boolean>(
                {
                    type: 'boolean',
                },
                parameters.map((params) => new FormControl<boolean>(params.ddnsUpdateOnRenew))
            ),
            ddnsUseConflictResolution: new SharedParameterFormGroup<boolean>(
                {
                    type: 'boolean',
                },
                parameters.map((params) => new FormControl<boolean>(params.ddnsUseConflictResolution))
            ),
            fourOverSixInterface: new SharedParameterFormGroup<string>(
                {
                    type: 'string',
                },
                parameters.map((params) => new FormControl<string>(params.fourOverSixInterface))
            ),
            fourOverSixInterfaceID: new SharedParameterFormGroup<string>(
                {
                    type: 'string',
                },
                parameters.map((params) => new FormControl<string>(params.fourOverSixInterfaceID))
            ),
            fourOverSixSubnet: new SharedParameterFormGroup<string>(
                {
                    type: 'string',
                },
                parameters.map(
                    (params) => new FormControl<string>(params.fourOverSixSubnet, StorkValidators.ipv6Prefix)
                )
            ),
            hostnameCharReplacement: new SharedParameterFormGroup<string>(
                {
                    type: 'string',
                },
                parameters.map((params) => new FormControl<string>(params.hostnameCharReplacement))
            ),
            hostnameCharSet: new SharedParameterFormGroup<string>(
                {
                    type: 'string',
                },
                parameters.map((params) => new FormControl<string>(params.hostnameCharSet))
            ),
            preferredLifetime: new SharedParameterFormGroup<number>(
                {
                    type: 'number',
                },
                parameters.map((params) => new FormControl<number>(params.preferredLifetime))
            ),
            minPreferredLifetime: new SharedParameterFormGroup<number>(
                {
                    type: 'number',
                },
                parameters.map((params) => new FormControl<number>(params.minPreferredLifetime))
            ),
            maxPreferredLifetime: new SharedParameterFormGroup<number>(
                {
                    type: 'number',
                },
                parameters.map((params) => new FormControl<number>(params.maxPreferredLifetime))
            ),
            reservationsGlobal: new SharedParameterFormGroup<boolean>(
                {
                    type: 'boolean',
                },
                parameters.map((params) => new FormControl<boolean>(params.reservationsGlobal))
            ),
            reservationsInSubnet: new SharedParameterFormGroup<boolean>(
                {
                    type: 'boolean',
                },
                parameters.map((params) => new FormControl<boolean>(params.reservationsInSubnet))
            ),
            reservationsOutOfPool: new SharedParameterFormGroup<boolean>(
                {
                    type: 'boolean',
                },
                parameters.map((params) => new FormControl<boolean>(params.reservationsOutOfPool))
            ),
            renewTimer: new SharedParameterFormGroup<number>(
                {
                    type: 'number',
                },
                parameters.map((params) => new FormControl<number>(params.renewTimer))
            ),
            rebindTimer: new SharedParameterFormGroup<number>(
                {
                    type: 'number',
                },
                parameters.map((params) => new FormControl<number>(params.rebindTimer))
            ),
            t1Percent: new SharedParameterFormGroup<number>(
                {
                    type: 'number',
                    min: 0,
                    max: 100,
                },
                parameters.map((params) => new FormControl<number>(params.t1Percent))
            ),
            t2Percent: new SharedParameterFormGroup<number>(
                {
                    type: 'number',
                    min: 0,
                    max: 100,
                },
                parameters.map((params) => new FormControl<number>(params.t2Percent))
            ),
            calculateTeeTimes: new SharedParameterFormGroup<boolean>(
                {
                    type: 'boolean',
                },
                parameters.map((params) => new FormControl<boolean>(params.calculateTeeTimes))
            ),
            validLifetime: new SharedParameterFormGroup<number>(
                {
                    type: 'number',
                },
                parameters.map((params) => new FormControl<number>(params.preferredLifetime))
            ),
            minValidLifetime: new SharedParameterFormGroup<number>(
                {
                    type: 'number',
                },
                parameters.map((params) => new FormControl<number>(params.minValidLifetime))
            ),
            maxValidLifetime: new SharedParameterFormGroup<number>(
                {
                    type: 'number',
                },
                parameters.map((params) => new FormControl<number>(params.maxValidLifetime))
            ),
            allocator: new SharedParameterFormGroup<string>(
                {
                    type: 'string',
                    values: ['iterative', 'random', 'flq'],
                },
                parameters.map((params) => new FormControl<string>(params.allocator))
            ),
            authoritative: new SharedParameterFormGroup<boolean>(
                {
                    type: 'boolean',
                },
                parameters.map((params) => new FormControl<boolean>(params.authoritative))
            ),
            bootFileName: new SharedParameterFormGroup<string>(
                {
                    type: 'string',
                },
                parameters.map((params) => new FormControl<string>(params.bootFileName))
            ),
            interface: new SharedParameterFormGroup<string>(
                {
                    type: 'string',
                },
                parameters.map((params) => new FormControl<string>(params._interface))
            ),
            interfaceID: new SharedParameterFormGroup<string>(
                {
                    type: 'string',
                },
                parameters.map((params) => new FormControl<string>(params.interfaceID))
            ),
            matchClientID: new SharedParameterFormGroup<boolean>(
                {
                    type: 'boolean',
                },
                parameters.map((params) => new FormControl<boolean>(params.matchClientID))
            ),
            nextServer: new SharedParameterFormGroup<string>(
                {
                    type: 'string',
                    invalidText: 'Please specify an IPv4 address.',
                },
                parameters.map((params) => new FormControl<string>(params.nextServer, StorkValidators.ipv4))
            ),
            pdAllocator: new SharedParameterFormGroup<string>(
                {
                    type: 'string',
                    values: ['iterative', 'random', 'flq'],
                },
                parameters.map((params) => new FormControl<string>(params.pdAllocator))
            ),
            rapidCommit: new SharedParameterFormGroup<boolean>(
                {
                    type: 'boolean',
                },
                parameters.map((params) => new FormControl<boolean>(params.rapidCommit))
            ),
            serverHostname: new SharedParameterFormGroup<string>(
                {
                    type: 'string',
                    invalidText: 'Please specify a valid hostname.',
                },
                parameters.map((params) => new FormControl<string>(params.serverHostname, StorkValidators.fqdn))
            ),
            storeExtendedInfo: new SharedParameterFormGroup<boolean>(
                {
                    type: 'boolean',
                },
                parameters.map((params) => new FormControl<boolean>(params.storeExtendedInfo))
            ),
        })
        return formGroup
    }
}
