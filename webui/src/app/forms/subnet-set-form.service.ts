import { Injectable } from '@angular/core'
import { FormControl, FormGroup, UntypedFormArray, UntypedFormGroup, Validators } from '@angular/forms'
import { SharedParameterFormGroup } from './shared-parameter-form-group'
import { KeaConfigSubnetDerivedParameters, LocalSubnet, Subnet } from '../backend'
import { StorkValidators } from '../validators'
import { DhcpOptionSetFormService } from './dhcp-option-set-form.service'
import { IPType } from '../iptype'
import { hasDifferentSubnetLevelOptions } from '../subnets'

/**
 * A type of the subnet form for editing Kea-specific parameters using
 * the {@link SharedParametersForm} component.
 */
export interface KeaSubnetParametersForm {
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

export interface OptionsForm {
    unlocked: FormControl<boolean>
    data: UntypedFormArray
}

/**
 * An interface describing the form for editing a subnet.
 */
export interface SubnetForm {
    /**
     * Subnet prefix.
     */
    subnet: FormControl<string>

    /**
     * Kea-specific parameters for a subnet.
     */
    parameters?: FormGroup<KeaSubnetParametersForm>

    /**
     * DHCP options in a subnet.
     */
    options?: FormGroup<OptionsForm>

    /**
     * Daemon IDs selected with a multi-select component.
     *
     * Selected daemons are associated with the subnet.
     */
    selectedDaemons: FormControl<number[]>
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
     *
     * @param optionService a service for manipulating DHCP options.
     */
    constructor(public optionService: DhcpOptionSetFormService) {}

    /**
     * Converts subnet data to a form.
     *
     * @param ipType universe (i.e., IPv4 or IPv6 subnet)
     * @param subnet subnet data.
     * @returns A form created for a subnet.
     */
    convertSubnetToForm(ipType: IPType, subnet: Subnet): FormGroup<SubnetForm> {
        let formGroup = new FormGroup<SubnetForm>({
            subnet: new FormControl({ value: subnet.subnet, disabled: true }),
            parameters: this.convertKeaParametersToForm(
                ipType,
                subnet.localSubnets?.map((ls) => ls.keaConfigSubnetParameters.subnetLevelParameters) || []
            ),
            options: new FormGroup({
                unlocked: new FormControl(hasDifferentSubnetLevelOptions(subnet)),
                data: new UntypedFormArray(
                    subnet.localSubnets?.map((ls) =>
                        this.optionService.convertOptionsToForm(
                            ipType,
                            ls.keaConfigSubnetParameters.subnetLevelParameters.options
                        )
                    ) || []
                ),
            }),
            selectedDaemons: new FormControl<number[]>(
                subnet.localSubnets?.map((ls) => ls.daemonId) || [],
                Validators.required
            ),
        })
        return formGroup
    }

    /**
     * Converts Kea subnet parameters to a form.
     *
     * The created form is used in the {@link SharedParametersForm} for editing
     * the subnet parameters. It comprises the metadata describing each parameter.
     *
     * @param ipType subnet universe (IPv4 or IPv6).
     * @param parameters Kea-specific subnet parameters.
     * @returns Created form group instance.
     */
    convertKeaParametersToForm(
        ipType: IPType,
        parameters: KeaConfigSubnetDerivedParameters[]
    ): FormGroup<KeaSubnetParametersForm> {
        // Common parameters.
        let form: KeaSubnetParametersForm = {
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
                    max: 1,
                    fractionDigits: 2,
                },
                parameters.map((params) => new FormControl<number>(params.t1Percent))
            ),
            t2Percent: new SharedParameterFormGroup<number>(
                {
                    type: 'number',
                    min: 0,
                    max: 1,
                    fractionDigits: 2,
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
                parameters.map((params) => new FormControl<number>(params.validLifetime))
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
            storeExtendedInfo: new SharedParameterFormGroup<boolean>(
                {
                    type: 'boolean',
                },
                parameters.map((params) => new FormControl<boolean>(params.storeExtendedInfo))
            ),
        }
        // DHCPv4 parameters.
        switch (ipType) {
            case IPType.IPv4:
                form.fourOverSixInterface = new SharedParameterFormGroup<string>(
                    {
                        type: 'string',
                    },
                    parameters.map((params) => new FormControl<string>(params.fourOverSixInterface))
                )
                form.fourOverSixInterfaceID = new SharedParameterFormGroup<string>(
                    {
                        type: 'string',
                    },
                    parameters.map((params) => new FormControl<string>(params.fourOverSixInterfaceID))
                )
                form.fourOverSixSubnet = new SharedParameterFormGroup<string>(
                    {
                        type: 'string',
                    },
                    parameters.map(
                        (params) => new FormControl<string>(params.fourOverSixSubnet, StorkValidators.ipv6Prefix())
                    )
                )
                form.bootFileName = new SharedParameterFormGroup<string>(
                    {
                        type: 'string',
                    },
                    parameters.map((params) => new FormControl<string>(params.bootFileName))
                )
                form.matchClientID = new SharedParameterFormGroup<boolean>(
                    {
                        type: 'boolean',
                    },
                    parameters.map((params) => new FormControl<boolean>(params.matchClientID))
                )
                form.nextServer = new SharedParameterFormGroup<string>(
                    {
                        type: 'string',
                        invalidText: 'Please specify an IPv4 address.',
                    },
                    parameters.map((params) => new FormControl<string>(params.nextServer, StorkValidators.ipv4()))
                )
                form.serverHostname = new SharedParameterFormGroup<string>(
                    {
                        type: 'string',
                        invalidText: 'Please specify a valid hostname.',
                    },
                    parameters.map((params) => new FormControl<string>(params.serverHostname, StorkValidators.fqdn))
                )
                break

            // DHCPv6 parameters.
            default:
                form.preferredLifetime = new SharedParameterFormGroup<number>(
                    {
                        type: 'number',
                    },
                    parameters.map((params) => new FormControl<number>(params.preferredLifetime))
                )
                form.minPreferredLifetime = new SharedParameterFormGroup<number>(
                    {
                        type: 'number',
                    },
                    parameters.map((params) => new FormControl<number>(params.minPreferredLifetime))
                )
                form.maxPreferredLifetime = new SharedParameterFormGroup<number>(
                    {
                        type: 'number',
                    },
                    parameters.map((params) => new FormControl<number>(params.maxPreferredLifetime))
                )
                form.pdAllocator = new SharedParameterFormGroup<string>(
                    {
                        type: 'string',
                        values: ['iterative', 'random', 'flq'],
                    },
                    parameters.map((params) => new FormControl<string>(params.pdAllocator))
                )
                form.rapidCommit = new SharedParameterFormGroup<boolean>(
                    {
                        type: 'boolean',
                    },
                    parameters.map((params) => new FormControl<boolean>(params.rapidCommit))
                )
        }
        let formGroup = new FormGroup<KeaSubnetParametersForm>(form)
        return formGroup
    }

    /**
     * Converts a form holding subnet data to a subnet instance.
     *
     * It currently only converts the simple DHCP parameters and options. It
     * excludes complex parameters, such as relay specification or pools.
     *
     * @param form a form comprising subnet data.
     * @returns A subnet instance converted from the form.
     */
    convertFormToSubnet(form: FormGroup<SubnetForm>): Subnet {
        let subnet: Subnet = {
            subnet: form.get('subnet')?.value,
            localSubnets:
                form.get('selectedDaemons')?.value.map((sd) => {
                    let ls: LocalSubnet = {
                        daemonId: sd,
                    }
                    return ls
                }) || [],
        }
        // Convert the simple DHCP parameters and options.
        const params = this.convertFormToKeaParameters(form.get('parameters') as FormGroup<KeaSubnetParametersForm>)
        const options = form.get('options') as UntypedFormGroup
        for (let i = 0; i < subnet.localSubnets.length; i++) {
            subnet.localSubnets[i].keaConfigSubnetParameters = {
                subnetLevelParameters: {},
            }
            if (params?.length > i) {
                subnet.localSubnets[i].keaConfigSubnetParameters = {
                    subnetLevelParameters: params[i],
                }
            }
            const data = options.get('data') as UntypedFormArray
            if (data?.length > i) {
                subnet.localSubnets[i].keaConfigSubnetParameters.subnetLevelParameters.options =
                    this.optionService.convertFormToOptions(
                        subnet.subnet?.includes(':') ? IPType.IPv6 : IPType.IPv4,
                        data.at(!!options.get('unlocked')?.value ? i : 0) as UntypedFormArray
                    )
            }
        }
        return subnet
    }

    /**
     * Converts a form holding DHCP parameters to a set of parameters assignable
     * to a subnet instance.
     *
     * @param form a form holding DHCP parameters for a subnet.
     * @returns An array of parameter sets for different servers.
     */
    convertFormToKeaParameters(form: FormGroup<KeaSubnetParametersForm>): KeaConfigSubnetDerivedParameters[] {
        const params: KeaConfigSubnetDerivedParameters[] = []

        for (let key in form.controls) {
            const unlocked = form.get(key).get('unlocked')?.value
            const values = form.get(key).get('values') as UntypedFormArray
            for (let i = 0; i < values.length; i++) {
                if (params.length <= i) {
                    params.push({})
                }
                if (values.at(!!unlocked ? i : 0).value != null) {
                    params[i][key] = values.at(!!unlocked ? i : 0).value
                }
            }
        }
        return params
    }

    /**
     * Creates a default form for an empty subnet.
     *
     * @param ipType subnet universe (IPv4 or IPv6).
     * @returns A default form group for a subnet.
     */
    createDefaultKeaParametersForm(ipType: IPType): UntypedFormGroup {
        let parameters: KeaConfigSubnetDerivedParameters[] = [{}]
        return this.convertKeaParametersToForm(ipType, parameters)
    }
}
