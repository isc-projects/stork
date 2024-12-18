import { Injectable } from '@angular/core'
import {
    FormArray,
    FormControl,
    FormGroup,
    UntypedFormArray,
    UntypedFormControl,
    UntypedFormGroup,
    Validators,
} from '@angular/forms'
import { gte, lt, valid } from 'semver'
import { SharedParameterFormGroup } from './shared-parameter-form-group'
import {
    DelegatedPrefixPool,
    KeaConfigPoolParameters,
    KeaConfigSubnetDerivedParameters,
    KeaConfigurableGlobalParameters,
    KeaDaemonConfig,
    LocalSharedNetwork,
    LocalSubnet,
    Pool,
    SharedNetwork,
    Subnet,
} from '../backend'
import { StorkValidators } from '../validators'
import { DhcpOptionSetFormService } from './dhcp-option-set-form.service'
import { IPType } from '../iptype'
import {
    extractUniqueSubnetPools,
    hasDifferentGlobalLevelOptions,
    hasDifferentLocalPoolOptions,
    hasDifferentSharedNetworkLevelOptions,
    hasDifferentSubnetLevelOptions,
    hasDifferentSubnetUserContexts,
} from '../subnets'
import { AddressRange } from '../address-range'
import { GenericFormService } from './generic-form.service'

/**
 * An interface to a {@link LocalSubnet}, {@link LocalPool} etc.
 */
interface LocalDaemonData {
    daemonId?: number
}

/**
 * A type of a form holding DHCP options.
 */
export interface OptionsForm {
    unlocked: FormControl<boolean>
    data: UntypedFormArray
}

/**
 * A type of a form holding user-contexts.
 */
export interface UserContextsForm {
    unlocked: FormControl<boolean>
    // An original user context data. Not editable now.
    contexts: FormArray<FormControl<object>>
    // The subnet names extracted from the user contexts.
    names: FormArray<FormControl<string>>
}

/**
 * A type of the form for editing Kea-specific pool parameters using
 * the {@link SharedParametersForm} component.
 */
export interface KeaPoolParametersForm {
    clientClass?: SharedParameterFormGroup<string>
    poolID?: SharedParameterFormGroup<number>
    requireClientClasses?: SharedParameterFormGroup<string[]>
}

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
    ddnsConflictResolutionMode?: SharedParameterFormGroup<string>
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
    relayAddresses?: SharedParameterFormGroup<string[]>
}

/**
 * A form group for editing pool address range.
 */
export interface AddressRangeForm {
    /**
     * Lower pool boundary.
     */
    start: FormControl<string>

    /**
     * Upper pool boundary.
     */
    end: FormControl<string>
}

/**
 * A form for editing an address pool.
 */
export interface AddressPoolForm {
    /**
     * Pool address range.
     */
    range: FormGroup<AddressRangeForm>

    /**
     * Kea-specific parameters for a pool.
     */
    parameters: FormGroup<KeaPoolParametersForm>

    /**
     * DHCP options in an address pool.
     */
    options: FormGroup<OptionsForm>

    /**
     * Daemon IDs selected with a multi-select component.
     *
     * Selected daemons are associated with the pool.
     */
    selectedDaemons: FormControl<number[]>
}

/**
 * A form for editing a prefix delegation pool.
 */
export interface PrefixForm {
    /**
     * A prefix in a CIDR notation.
     */
    prefix: FormControl<string>

    /**
     * A delegated prefix length.
     */
    delegatedLength: FormControl<number>

    /**
     * An excluded prefix in a CIDR notation.
     */
    excludedPrefix: FormControl<string>
}

/**
 * A form for editing a delegated prefix pool.
 */
export interface PrefixPoolForm {
    /**
     * Delegated and excluded prefixes.
     */
    prefixes: FormGroup<PrefixForm>

    /**
     * Kea-specific parameters for a pool.
     */
    parameters: FormGroup<KeaPoolParametersForm>

    /**
     * DHCP options in the pool.
     */
    options: FormGroup<OptionsForm>

    /**
     * Daemon IDs selected with a multi-select component.
     *
     * Selected daemons are associated with the pool.
     */
    selectedDaemons: FormControl<number[]>
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
     * Association with a shared network.
     */
    sharedNetwork: FormControl<number>

    /**
     * An array of the address pools.
     */
    pools: FormArray<FormGroup<AddressPoolForm>>

    /**
     * An array of the delegated prefix pools.
     */
    prefixPools: FormArray<FormGroup<PrefixPoolForm>>

    /**
     * Kea-specific parameters for a subnet.
     */
    parameters: FormGroup<KeaSubnetParametersForm>

    /**
     * DHCP options in a subnet.
     */
    options: FormGroup<OptionsForm>

    /**
     * Daemon IDs selected with a multi-select component.
     *
     * Selected daemons are associated with the subnet.
     */
    selectedDaemons: FormControl<number[]>

    /**
     * User contexts for a subnet.
     */
    userContexts: FormGroup<UserContextsForm>
}

/**
 * An interface describing the form for editing a shared network.
 */
export interface SharedNetworkForm {
    /**
     * Shared network name.
     */
    name: FormControl<string>

    /**
     * Kea-specific parameters for a shared network.
     */
    parameters: FormGroup<KeaSubnetParametersForm>

    /**
     * DHCP options in a shared network.
     */
    options: FormGroup<OptionsForm>

    /**
     * Daemon IDs selected with a multi-select component.
     *
     * Selected daemons are associated with the shared network.
     */
    selectedDaemons: FormControl<number[]>
}

/**
 * An interface describing the form for editing global Kea parameters.
 */
export interface KeaGlobalParametersForm {
    allocator?: SharedParameterFormGroup<string>
    authoritative?: SharedParameterFormGroup<boolean>
    cacheThreshold?: SharedParameterFormGroup<number>
    ddnsGeneratedPrefix?: SharedParameterFormGroup<string>
    ddnsOverrideClientUpdate?: SharedParameterFormGroup<boolean>
    ddnsOverrideNoUpdate?: SharedParameterFormGroup<boolean>
    ddnsQualifyingSuffix?: SharedParameterFormGroup<string>
    ddnsReplaceClientName?: SharedParameterFormGroup<string>
    ddnsSendUpdates?: SharedParameterFormGroup<boolean>
    ddnsUpdateOnRenew?: SharedParameterFormGroup<boolean>
    ddnsUseConflictResolution?: SharedParameterFormGroup<boolean>
    ddnsConflictResolutionMode?: SharedParameterFormGroup<string>
    dhcpDdnsEnableUpdates?: SharedParameterFormGroup<boolean>
    dhcpDdnsServerIP?: SharedParameterFormGroup<string>
    dhcpDdnsServerPort?: SharedParameterFormGroup<number>
    dhcpDdnsSenderIP?: SharedParameterFormGroup<string>
    dhcpDdnsSenderPort?: SharedParameterFormGroup<number>
    dhcpDdnsMaxQueueSize?: SharedParameterFormGroup<number>
    dhcpDdnsNcrProtocol?: SharedParameterFormGroup<string>
    dhcpDdnsNcrFormat?: SharedParameterFormGroup<string>
    earlyGlobalReservationsLookup?: SharedParameterFormGroup<boolean>
    echoClientId?: SharedParameterFormGroup<boolean>
    expiredFlushReclaimedTimerWaitTime?: SharedParameterFormGroup<number>
    expiredHoldReclaimedTime?: SharedParameterFormGroup<number>
    expiredMaxReclaimLeases?: SharedParameterFormGroup<number>
    expiredMaxReclaimTime?: SharedParameterFormGroup<number>
    expiredReclaimTimerWaitTime?: SharedParameterFormGroup<number>
    expiredUnwarnedReclaimCycles?: SharedParameterFormGroup<number>
    hostReservationIdentifiers?: SharedParameterFormGroup<string[]>
    reservationsGlobal?: SharedParameterFormGroup<boolean>
    reservationsInSubnet?: SharedParameterFormGroup<boolean>
    reservationsOutOfPool?: SharedParameterFormGroup<boolean>
    pdAllocator?: SharedParameterFormGroup<string>
}

/**
 * An interface describing the form for editing global Kea configuration.
 */
export interface KeaGlobalConfigurationForm {
    /**
     * Kea global parameters.
     */
    parameters: FormGroup<KeaGlobalParametersForm>

    /**
     * Kea global DHCP options.
     */
    options: FormGroup<OptionsForm>
}

/**
 * An interface for retrieving a version of a daemon.
 *
 * It is used in the form conversion functions where some of the
 * parameters must be excluded when the daemons do not meet version
 * requirements for the selected parameters.
 */
export interface VersionedDaemon {
    id: number
    version: string
}

/**
 * Raw Kea configuration type.
 *
 * It is an alias for a raw configuration returned by the Kea server
 * in the {@link KeaDaemonConfig}.
 */
export type KeaRawConfig = { [key: string]: any }

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
     * @param genericFormService a generic form service used to clone controls.
     * @param optionService a service for manipulating DHCP options.
     */
    constructor(
        private genericFormService: GenericFormService,
        private optionService: DhcpOptionSetFormService
    ) {}

    /**
     * Extract the index from the array by matching the daemon id.
     *
     * @param localData A {@link LocalSubnet} or {@link LocalPool} etc.
     * @param selectedDaemons an array with identifiers of the selected daemons.
     * @returns An index in the array.
     */
    private getDaemonIndex(localData: LocalDaemonData, selectedDaemons: number[]) {
        return selectedDaemons.findIndex((sd) => sd === localData.daemonId)
    }

    /**
     * Generic function converting a form to Kea-specific parameters.
     *
     * It can be used for different parameter sets, e.g. subnet-specific parameters,
     * pool-specific parameters etc.
     *
     * @typeParam FormType a type of the form holding the parameters.
     * @typeParam ParamsType a type of the parameter set returned by this function.
     * @param daemons an array of daemons including their software versions.
     * @param form a form group holding the parameters set by the {@link SharedParametersForm}
     * component.
     * @returns An array of the parameter sets.
     */
    private convertFormToKeaParameters<
        FormType extends { [K in keyof FormType]: SharedParameterFormGroup<any, any> },
        ParamsType extends { [K in keyof ParamsType]: ParamsType[K] },
    >(daemons: VersionedDaemon[], form: FormGroup<FormType>): ParamsType[] {
        const params: ParamsType[] = []
        // Iterate over all parameters.
        for (let key in form.controls) {
            const unlocked = form.get(key).get('unlocked')?.value
            // Get the values of the parameter for different servers.
            const values = form.get(key).get('values') as UntypedFormArray
            const data = (form.controls[key] as SharedParameterFormGroup<any, any>)?.data
            // For each server-specific value of the parameter.
            for (let i = 0; i < values?.length; i++) {
                // If we haven't added the parameter set for the current index let's add one.
                if (params.length <= i) {
                    params.push({} as ParamsType)
                }
                // Some of the configured parameters may not be applicable to certain Kea
                // server versions. We need to compare the daemon versions with the upper
                // and lower bound versions potentially set for each parameter.
                if (
                    valid(daemons?.[i]?.version) &&
                    ((valid(data?.versionLowerBound) && lt(daemons[i].version, data.versionLowerBound)) ||
                        (valid(data?.versionUpperBound) && gte(daemons[i].version, data.versionUpperBound)))
                ) {
                    // The parameter is not supported by the current daemon version.
                    continue
                }
                // If the parameter is unlocked, there should be a value dedicated
                // for each server. Otherwise, we add the first (common) value.
                if (values.at(!!unlocked ? i : 0).value != null) {
                    params[i][key] = values.at(!!unlocked ? i : 0).value
                }
            }
        }
        return params
    }

    /**
     * Create default form for editing DHCP options.
     *
     * @returns A default form group for DHCP options.
     */
    createDefaultOptionsForm(): FormGroup<OptionsForm> {
        return new FormGroup({
            unlocked: new FormControl({ value: false, disabled: true }),
            data: new UntypedFormArray([new UntypedFormArray([])]),
        })
    }

    /**
     * Convert Kea pool parameters to a form.
     *
     * @param parameters Kea-specific pool parameters.
     * @returns Created form group instance.
     */
    convertKeaPoolParametersToForm(parameters: KeaConfigPoolParameters[]): FormGroup<KeaPoolParametersForm> {
        let form: KeaPoolParametersForm = {
            clientClass: new SharedParameterFormGroup<string>(
                {
                    type: 'string',
                },
                parameters?.map((params) => new FormControl<string>(params?.clientClass ?? null))
            ),
            poolID: new SharedParameterFormGroup<number>(
                {
                    type: 'number',
                    min: 1,
                    invalidText: 'Please specify non-overlapping numeric pool identifiers.',
                },
                parameters?.map((params) => new FormControl<number>(params?.poolID ?? null))
            ),
            requireClientClasses: new SharedParameterFormGroup<string[]>(
                {
                    type: 'client-classes',
                },
                parameters?.map((params) => new FormControl<string[]>(params?.requireClientClasses ?? []))
            ),
        }
        let formGroup = new FormGroup<KeaPoolParametersForm>(form)
        return formGroup
    }

    /**
     * Creates a default parameters form for an empty pool.
     *
     * @param ipType subnet universe (IPv4 or IPv6).
     * @returns A default form group for a subnet.
     */
    createDefaultKeaPoolParametersForm(): UntypedFormGroup {
        let parameters: KeaConfigPoolParameters[] = [{}]
        return this.convertKeaPoolParametersToForm(parameters)
    }

    /**
     * Creates a default form for an address pool.
     *
     * @param subnet subnet prefix.
     * @returns A default form group for an address pool.
     */
    createDefaultAddressPoolForm(subnet: string): FormGroup<AddressPoolForm> {
        let formGroup = new FormGroup<AddressPoolForm>({
            range: new FormGroup<AddressRangeForm>(
                {
                    start: new FormControl('', StorkValidators.ipInSubnet(subnet)),
                    end: new FormControl('', StorkValidators.ipInSubnet(subnet)),
                },
                StorkValidators.ipRangeBounds
            ),
            parameters: this.createDefaultKeaPoolParametersForm(),
            options: this.createDefaultOptionsForm(),
            selectedDaemons: new FormControl([], Validators.required),
        })
        return formGroup
    }

    /**
     * Creates a default form for a prefix pool.
     *
     * @returns A default form group for a prefix pool.
     */
    createDefaultPrefixPoolForm(): FormGroup<PrefixPoolForm> {
        let formGroup = new FormGroup<PrefixPoolForm>({
            prefixes: new FormGroup<PrefixForm>(
                {
                    prefix: new FormControl('', StorkValidators.ipv6Prefix),
                    delegatedLength: new FormControl(null, Validators.required),
                    excludedPrefix: new FormControl('', StorkValidators.ipv6Prefix),
                },
                Validators.compose([
                    StorkValidators.ipv6PrefixDelegatedLength,
                    StorkValidators.ipv6ExcludedPrefixDelegatedLength,
                    StorkValidators.ipv6ExcludedPrefix,
                ])
            ),
            parameters: this.createDefaultKeaPoolParametersForm(),
            options: this.createDefaultOptionsForm(),
            selectedDaemons: new FormControl([], Validators.required),
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
     * @param keaVersionRange a tuple with the earliest and the latest Kea version
     *        for the configured daemons.
     * @param level level in the configuration hierarchy.
     * @param parameters Kea-specific subnet parameters.
     * @returns Created form group instance.
     */
    convertKeaSubnetParametersToForm(
        ipType: IPType,
        keaVersionRange: [string, string],
        level: 'subnet' | 'shared-network',
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
            relayAddresses: new SharedParameterFormGroup<string[]>(
                {
                    type: 'string',
                    isArray: true,
                    invalidText:
                        ipType === IPType.IPv4
                            ? 'Please specify valid IPv4 addresses.'
                            : 'Please specify valid IPv6 addresses.',
                },
                parameters.map(
                    (params) =>
                        new FormControl<string[]>(
                            params.relay?.ipAddresses,
                            ipType === IPType.IPv4 ? StorkValidators.ipv4() : StorkValidators.ipv6()
                        )
                )
            ),
        }
        if (!keaVersionRange || lt(keaVersionRange[0], '2.5.0')) {
            form.ddnsUseConflictResolution = new SharedParameterFormGroup<boolean>(
                {
                    type: 'boolean',
                    versionUpperBound: !keaVersionRange || gte(keaVersionRange[1], '2.5.0') ? '2.5.0' : undefined,
                },
                parameters.map((params) => new FormControl<boolean>(params.ddnsUseConflictResolution))
            )
        }
        if (!keaVersionRange || gte(keaVersionRange[1], '2.5.0')) {
            form.ddnsConflictResolutionMode = new SharedParameterFormGroup<string>(
                {
                    type: 'string',
                    values: [
                        'check-with-dhcid',
                        'no-check-with-dhcid',
                        'check-exists-with-dhcid',
                        'no-check-without-dhcid',
                    ],
                    versionLowerBound: !keaVersionRange || lt(keaVersionRange[0], '2.5.0') ? '2.5.0' : undefined,
                },
                parameters.map((params) => new FormControl<string>(params.ddnsConflictResolutionMode))
            )
        }
        // DHCPv4 parameters.
        switch (ipType) {
            case IPType.IPv4:
                // These parameters are only valid for a subnet.
                if (level === 'subnet') {
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
                            (params) => new FormControl<string>(params.fourOverSixSubnet, StorkValidators.ipv6Prefix)
                        )
                    )
                }
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
     * Converts a form holding DHCP parameters to a set of parameters assignable
     * to a subnet instance.
     *
     * @param daemons an array of daemons comprising their software versions.
     * @param form a form holding DHCP parameters for a subnet.
     * @returns An array of parameter sets for different servers.
     */
    convertFormToKeaSubnetParameters(
        daemons: VersionedDaemon[],
        form: FormGroup<KeaSubnetParametersForm>
    ): KeaConfigSubnetDerivedParameters[] {
        const convertedParameters = this.convertFormToKeaParameters<
            KeaSubnetParametersForm,
            KeaConfigSubnetDerivedParameters
        >(daemons, form)
        for (let parameters of convertedParameters) {
            if ('relayAddresses' in parameters) {
                parameters.relay = {
                    ipAddresses: parameters.relayAddresses as string[],
                }
                delete parameters['relayAddresses']
            }
        }
        return convertedParameters
    }

    /**
     * Creates a default parameters form for an empty subnet.

     * @param ipType shared network universe (IPv4 or IPv6).
     * @param keaVersionRange a tuple with the earliest and the latest Kea version
     *        for the configured daemons.
     * @returns A default form group for a shared network.
     */
    createDefaultKeaSharedNetworkParametersForm(ipType: IPType, keaVersionRange: [string, string]): UntypedFormGroup {
        let parameters: KeaConfigSubnetDerivedParameters[] = [{}]
        return this.convertKeaSubnetParametersToForm(ipType, keaVersionRange, 'shared-network', parameters)
    }

    /**
     * Creates a default parameters form for an empty subnet.
     *
     * @param ipType subnet universe (IPv4 or IPv6).
     * @param keaVersionRange a tuple with the earliest and the latest Kea version
     *        for the configured daemons.
     * @returns A default form group for a subnet.
     */
    createDefaultKeaSubnetParametersForm(ipType: IPType, keaVersionRange: [string, string]): UntypedFormGroup {
        let parameters: KeaConfigSubnetDerivedParameters[] = [{}]
        return this.convertKeaSubnetParametersToForm(ipType, keaVersionRange, 'subnet', parameters)
    }

    /**
     * Converts a set of address pools in a subnet to a form.
     *
     * @param subnet a subnet instance holding the converted pools.
     * @returns An array of form groups representing address pools.
     */
    convertAddressPoolsToForm(subnet: Subnet): FormArray<FormGroup<AddressPoolForm>> {
        const formArray = new FormArray<FormGroup<AddressPoolForm>>(
            [],
            [StorkValidators.ipRangeOverlaps, StorkValidators.poolIDOverlaps]
        )
        // A subnet can be associated with many servers. Each server may contain
        // the same or different address pools. Some of the pools may overlap.
        // This call extracts the pools and combines those that are the same for
        // different servers. It makes it easier to later convert the extracted pools
        // to a form.
        const subnetWithUniquePools = extractUniqueSubnetPools(subnet)
        if (subnetWithUniquePools.length === 0) {
            return formArray
        }
        // Iterate over the extracted pools and convert them to a form.
        for (const pool of subnetWithUniquePools[0]?.pools) {
            // Attempt to validate and convert the pool range specified
            // as a string to an address range. It may throw.
            const addressRange = AddressRange.fromStringRange(pool.pool)
            formArray.push(
                new FormGroup<AddressPoolForm>({
                    range: new FormGroup<AddressRangeForm>(
                        {
                            start: new FormControl(addressRange.getFirst(), StorkValidators.ipInSubnet(subnet.subnet)),
                            end: new FormControl(addressRange.getLast(), StorkValidators.ipInSubnet(subnet.subnet)),
                        },
                        StorkValidators.ipRangeBounds
                    ),
                    // Local pools contain Kea-specific pool parameters for different servers.
                    // Extract them from the local pools and pass as an array to the conversion
                    // function.
                    parameters: this.convertKeaPoolParametersToForm(
                        pool.localPools?.map((lp) => lp.keaConfigPoolParameters) || []
                    ),
                    // Convert the options to a form.
                    options: new FormGroup({
                        unlocked: new FormControl(hasDifferentLocalPoolOptions(pool)),
                        data: new UntypedFormArray(
                            pool.localPools?.map((lp) =>
                                this.optionService.convertOptionsToForm(
                                    subnet.subnet?.includes('.') ? IPType.IPv4 : IPType.IPv6,
                                    lp.keaConfigPoolParameters?.options
                                )
                            ) || []
                        ),
                    }),
                    selectedDaemons: new FormControl<number[]>(
                        pool.localPools?.map((lp) => lp.daemonId) || [],
                        Validators.required
                    ),
                })
            )
        }
        return formArray
    }

    /**
     * Converts a form holding pool data to a pool instance.
     *
     * @param daemons an array of daemons including their software data.
     * @param localData an interface pointing to a local subnet, pool or shared
     * network for which the data should be converted.
     * @param form form a form comprising pool data.
     * @returns A pool instance converted from the form.
     */
    convertFormToAddressPools(
        daemons: VersionedDaemon[],
        localData: LocalDaemonData,
        form: FormArray<FormGroup<AddressPoolForm>>
    ): Pool[] {
        const pools: Pool[] = []
        for (let poolCtrl of form.controls) {
            const selectedDaemons = poolCtrl.get('selectedDaemons')?.value
            const index = this.getDaemonIndex(localData, selectedDaemons)
            if (index < 0) {
                continue
            }
            const range = `${poolCtrl.get('range.start').value}-${poolCtrl.get('range.end').value}`
            const params = this.convertFormToKeaParameters(
                daemons.filter((d) => selectedDaemons.includes(d.id)),
                poolCtrl.get('parameters') as FormGroup<KeaPoolParametersForm>
            )
            const options = poolCtrl.get('options') as UntypedFormGroup
            const pool: Pool = {
                pool: range,
                keaConfigPoolParameters: params?.length > index ? params[index] : null,
            }
            const data = options.get('data') as UntypedFormArray
            if (data?.length > index) {
                if (!pool.keaConfigPoolParameters) {
                    pool.keaConfigPoolParameters = {}
                }
                pool.keaConfigPoolParameters.options = this.optionService.convertFormToOptions(
                    range.includes(':') ? IPType.IPv6 : IPType.IPv4,
                    data.at(!!options.get('unlocked')?.value ? index : 0) as UntypedFormArray
                )
            }
            pools.push(pool)
        }
        return pools
    }

    /**
     * Converts a set of delegated prefix pools in a subnet to a form.
     *
     * @param subnet a subnet instance holding the converted pools.
     * @returns An array of form groups representing the pools.
     */
    convertPrefixPoolsToForm(subnet: Subnet): FormArray<FormGroup<PrefixPoolForm>> {
        const formArray = new FormArray<FormGroup<PrefixPoolForm>>(
            [],
            [StorkValidators.ipv6PrefixOverlaps, StorkValidators.poolIDOverlaps]
        )
        // A subnet can be associated with many servers. Each server may contain
        // the same or different prefix pools. Some of the pools may overlap.
        // This call extracts the pools and combines those that are the same for
        // different servers. It makes it easier to later convert the extracted pools
        // to a form.
        const subnetWithUniquePools = extractUniqueSubnetPools(subnet)
        if (subnetWithUniquePools.length === 0) {
            return formArray
        }
        // Iterate over the extracted pools and convert them to a form.
        for (const pool of subnetWithUniquePools[0]?.prefixDelegationPools) {
            formArray.push(
                new FormGroup<PrefixPoolForm>({
                    prefixes: new FormGroup<PrefixForm>(
                        {
                            prefix: new FormControl(
                                pool.prefix,
                                Validators.compose([Validators.required, StorkValidators.ipv6Prefix])
                            ),
                            delegatedLength: new FormControl(pool.delegatedLength, Validators.required),
                            excludedPrefix: new FormControl(pool.excludedPrefix, StorkValidators.ipv6Prefix),
                        },
                        Validators.compose([
                            StorkValidators.ipv6PrefixDelegatedLength,
                            StorkValidators.ipv6ExcludedPrefixDelegatedLength,
                            StorkValidators.ipv6ExcludedPrefix,
                        ])
                    ),
                    // Local pools contain Kea-specific pool parameters for different servers.
                    // Extract them from the local pools and pass as an array to the conversion
                    // function.
                    parameters: this.convertKeaPoolParametersToForm(
                        pool.localPools?.map((lp) => lp.keaConfigPoolParameters) || []
                    ),
                    // Convert the options to a form.
                    options: new FormGroup({
                        unlocked: new FormControl(hasDifferentLocalPoolOptions(pool)),
                        data: new UntypedFormArray(
                            pool.localPools?.map((lp) =>
                                this.optionService.convertOptionsToForm(
                                    subnet.subnet?.includes('.') ? IPType.IPv4 : IPType.IPv6,
                                    lp.keaConfigPoolParameters?.options
                                )
                            ) || []
                        ),
                    }),
                    selectedDaemons: new FormControl<number[]>(
                        pool.localPools?.map((lp) => lp.daemonId) || [],
                        Validators.required
                    ),
                })
            )
        }
        return formArray
    }

    /**
     * Converts a form holding delegated prefix pool data to a pool instance.
     *
     * @param daemons an array of daemons including their software versions.
     * @param localData an interface pointing to a local subnet, pool or shared
     * network for which the data should be converted.
     * @param form form a form comprising pool data.
     * @returns A pool instance converted from the form.
     */
    convertFormToPrefixPools(
        daemons: VersionedDaemon[],
        localData: LocalDaemonData,
        form: FormArray<FormGroup<PrefixPoolForm>>
    ): DelegatedPrefixPool[] {
        const pools: DelegatedPrefixPool[] = []
        for (let poolCtrl of form.controls) {
            const selectedDaemons = poolCtrl.get('selectedDaemons')?.value
            const index = this.getDaemonIndex(localData, selectedDaemons)
            if (index < 0) {
                continue
            }
            const prefix = poolCtrl.get('prefixes.prefix').value
            const params = this.convertFormToKeaParameters(
                daemons.filter((d) => selectedDaemons.includes(d.id)),
                poolCtrl.get('parameters') as FormGroup<KeaPoolParametersForm>
            )
            const options = poolCtrl.get('options') as UntypedFormGroup
            const pool: DelegatedPrefixPool = {
                prefix: prefix || null,
                delegatedLength: poolCtrl.get('prefixes.delegatedLength').value || null,
                excludedPrefix: poolCtrl.get('prefixes.excludedPrefix').value || null,
                keaConfigPoolParameters: params?.length > index ? params[index] : null,
            }
            const data = options.get('data') as UntypedFormArray
            if (data?.length > index) {
                if (!pool.keaConfigPoolParameters) {
                    pool.keaConfigPoolParameters = {}
                }
                pool.keaConfigPoolParameters.options = this.optionService.convertFormToOptions(
                    prefix?.includes(':') ? IPType.IPv6 : IPType.IPv4,
                    data.at(!!options.get('unlocked')?.value ? index : 0) as UntypedFormArray
                )
            }
            pools.push(pool)
        }
        return pools
    }

    /**
     * Converts subnet data to a form.
     *
     * @param ipType universe (i.e., IPv4 or IPv6 subnet)
     * @param keaVersionRange a tuple with the earliest and the latest Kea version
     *        for the configured daemons.
     * @param subnet subnet data.
     * @returns A form created for a subnet.
     */
    convertSubnetToForm(ipType: IPType, keaVersionRange: [string, string], subnet: Subnet): FormGroup<SubnetForm> {
        let formGroup = new FormGroup<SubnetForm>({
            subnet: new FormControl({ value: subnet.subnet, disabled: true }),
            sharedNetwork: new FormControl(subnet.sharedNetworkId),
            pools: this.convertAddressPoolsToForm(subnet),
            prefixPools: this.convertPrefixPoolsToForm(subnet),
            parameters: this.convertKeaSubnetParametersToForm(
                ipType,
                keaVersionRange,
                'subnet',
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
                {
                    value: subnet.localSubnets?.map((ls) => ls.daemonId) || [],
                    disabled: !!subnet.sharedNetworkId,
                },
                Validators.required
            ),
            userContexts: new FormGroup({
                unlocked: new FormControl(hasDifferentSubnetUserContexts(subnet)),
                contexts: new FormArray<FormControl<object>>(
                    subnet.localSubnets?.map((ls) => new FormControl(ls.userContext)) ?? []
                ),
                names: new FormArray<FormControl<string>>(
                    subnet.localSubnets?.map((ls) => new FormControl(ls.userContext?.['subnet-name'])) ?? []
                ),
            }),
        })
        return formGroup
    }

    /**
     * Converts shared network data to a form.
     *
     * @param keaVersionRange a tuple with the earliest and the latest Kea version
     *        for the configured daemons.
     * @param sharedNetwork shared network data.
     * @param sharedNetworkNames list of names of the existing shared networks.
     * @returns A form created for a shared network.
     */
    convertSharedNetworkToForm(
        keaVersionRange: [string, string],
        sharedNetwork: SharedNetwork,
        sharedNetworkNames: string[]
    ): FormGroup<SharedNetworkForm> {
        let formGroup = new FormGroup<SharedNetworkForm>({
            name: new FormControl(
                sharedNetwork.name,
                Validators.compose([Validators.required, StorkValidators.valueInList(sharedNetworkNames)])
            ),
            parameters: this.convertKeaSubnetParametersToForm(
                sharedNetwork.universe,
                keaVersionRange,
                'shared-network',
                sharedNetwork.localSharedNetworks?.map(
                    (lsn) => lsn.keaConfigSharedNetworkParameters.sharedNetworkLevelParameters
                ) || []
            ),
            options: new FormGroup({
                unlocked: new FormControl(hasDifferentSharedNetworkLevelOptions(sharedNetwork)),
                data: new UntypedFormArray(
                    sharedNetwork.localSharedNetworks?.map((lsn) =>
                        this.optionService.convertOptionsToForm(
                            sharedNetwork.universe,
                            lsn.keaConfigSharedNetworkParameters.sharedNetworkLevelParameters.options
                        )
                    ) || []
                ),
            }),
            selectedDaemons: new FormControl<number[]>(
                sharedNetwork.localSharedNetworks?.map((lsn) => lsn.daemonId) || [],
                Validators.required
            ),
        })
        return formGroup
    }

    /**
     * Creates a default form for a shared network.
     *
     * @param ipType a shared network universe (IPv4 or IPv6).
     * @param keaVersionRange a tuple with the earliest and the latest Kea version
     *        for the configured daemons.
     * @param sharedNetworks existing shared networks' names.
     * @returns A default form group for a shared network.
     */
    createDefaultSharedNetworkForm(
        ipType: IPType,
        keaVersionRange: [string, string],
        sharedNetworks: string[]
    ): FormGroup<SharedNetworkForm> {
        let formGroup = new FormGroup<SharedNetworkForm>({
            name: new FormControl('', [Validators.required, StorkValidators.valueInList(sharedNetworks)]),
            parameters: this.createDefaultKeaSharedNetworkParametersForm(ipType, keaVersionRange),
            options: this.createDefaultOptionsForm(),
            selectedDaemons: new FormControl<number[]>([], Validators.required),
        })
        return formGroup
    }

    /**
     * Creates a default form for a subnet.
     *
     * When the subnet is specified, the subnet control is disabled and the subnet
     * cannot be modified. That's because the validation of the remaining parameters
     * depends on the exact subnet value. If the subnet is not specified, the subnet
     * control remains enabled.
     *
     * @param keaVersionRange a tuple with the earliest and the latest Kea version
     *        for the configured daemons.
     * @param subnets an array of existing subnets that are compared with the subnet
     * in the form for overlaps, or a specified subnet.
     * @returns A default form group for a subnet.
     */
    createDefaultSubnetForm(keaVersionRange: [string, string], subnets: string[] | string): FormGroup<SubnetForm> {
        const isArray = Array.isArray(subnets)
        let formGroup = new FormGroup<SubnetForm>({
            subnet: new FormControl({ value: isArray ? null : subnets, disabled: !isArray }, [
                Validators.required,
                StorkValidators.ipPrefix,
                StorkValidators.prefixInList(isArray ? subnets : []),
            ]),
            sharedNetwork: new FormControl(null),
            pools: new FormArray<FormGroup<AddressPoolForm>>([], StorkValidators.ipRangeOverlaps),
            prefixPools: new FormArray<FormGroup<PrefixPoolForm>>([], StorkValidators.ipv6PrefixOverlaps),
            parameters: this.createDefaultKeaSubnetParametersForm(IPType.IPv4, keaVersionRange),
            options: this.createDefaultOptionsForm(),
            selectedDaemons: new FormControl<number[]>([], Validators.required),
            userContexts: new FormGroup({
                unlocked: new FormControl(false),
                contexts: new FormArray([]),
                names: new FormArray([]),
            }),
        })
        return formGroup
    }

    /**
     * Converts a form holding subnet data to a subnet instance.
     *
     * It currently only converts the simple DHCP parameters and options.
     *
     * @param daemons an array of daemons comprising their software versions.
     * @param form a form comprising subnet data.
     * @returns A subnet instance converted from the form.
     */
    convertFormToSubnet(daemons: VersionedDaemon[], form: FormGroup<SubnetForm>): Subnet {
        let subnet: Subnet = {
            subnet: form.get('subnet')?.value,
            sharedNetworkId: form.get('sharedNetwork')?.value,
            localSubnets:
                form.get('selectedDaemons')?.value.map((sd) => {
                    let ls: LocalSubnet = {
                        daemonId: sd,
                    }
                    return ls
                }) || [],
        }
        // Convert the simple DHCP parameters and options.
        const params = this.convertFormToKeaSubnetParameters(
            daemons,
            form.get('parameters') as FormGroup<KeaSubnetParametersForm>
        )
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
            subnet.localSubnets[i].pools = this.convertFormToAddressPools(
                daemons,
                subnet.localSubnets[i],
                form.get('pools') as FormArray<FormGroup<AddressPoolForm>>
            )
            subnet.localSubnets[i].prefixDelegationPools = this.convertFormToPrefixPools(
                daemons,
                subnet.localSubnets[i],
                form.get('prefixPools') as FormArray<FormGroup<PrefixPoolForm>>
            )
            const data = options.get('data') as UntypedFormArray
            if (data?.length > i) {
                subnet.localSubnets[i].keaConfigSubnetParameters.subnetLevelParameters.options =
                    this.optionService.convertFormToOptions(
                        subnet.subnet?.includes(':') ? IPType.IPv6 : IPType.IPv4,
                        data.at(!!options.get('unlocked')?.value ? i : 0) as UntypedFormArray
                    )
            }
        }
        const group = form.get('userContexts') as FormGroup<UserContextsForm>
        const contexts = group.get('contexts')?.value
        const names = group.get('names')?.value
        const unlocked = !!group.get('unlocked')?.value
        for (let i = 0; i < subnet.localSubnets.length; i++) {
            if (contexts?.length <= i) {
                break
            }
            let context = contexts[unlocked ? i : 0]
            const name = names[unlocked ? i : 0]

            if (!context && name) {
                context = {}
            }
            if (name) {
                context['subnet-name'] = name
            } else if (context) {
                delete context['subnet-name']
            }

            subnet.localSubnets[i].userContext = context
        }
        return subnet
    }

    /**
     * Converts a form holding shared network data to a shared network instance.
     *
     * @param daemons an array of daemons including their software versions.
     * @param ipType universe (i.e., IPv4 or IPv6 shared network)
     * @param form a form comprising subnet data.
     * @returns A subnet instance converted from the form.
     */
    convertFormToSharedNetwork(
        daemons: VersionedDaemon[],
        ipType: IPType,
        form: FormGroup<SharedNetworkForm>
    ): SharedNetwork {
        let sharedNetwork: SharedNetwork = {
            name: form.get('name')?.value,
            universe: ipType === IPType.IPv6 ? 6 : 4,
            localSharedNetworks:
                form.get('selectedDaemons')?.value.map((sd) => {
                    let lsn: LocalSharedNetwork = {
                        daemonId: sd,
                    }
                    return lsn
                }) || [],
        }
        // Convert the simple DHCP parameters and options.
        const params = this.convertFormToKeaSubnetParameters(
            daemons,
            form.get('parameters') as FormGroup<KeaSubnetParametersForm>
        )
        const options = form.get('options') as UntypedFormGroup
        for (let i = 0; i < sharedNetwork.localSharedNetworks.length; i++) {
            sharedNetwork.localSharedNetworks[i].keaConfigSharedNetworkParameters = {
                sharedNetworkLevelParameters: {},
            }
            if (params?.length > i) {
                sharedNetwork.localSharedNetworks[i].keaConfigSharedNetworkParameters = {
                    sharedNetworkLevelParameters: params[i],
                }
            }
            const data = options.get('data') as UntypedFormArray
            if (data?.length > i) {
                sharedNetwork.localSharedNetworks[
                    i
                ].keaConfigSharedNetworkParameters.sharedNetworkLevelParameters.options =
                    this.optionService.convertFormToOptions(
                        ipType,
                        data.at(!!options.get('unlocked')?.value ? i : 0) as UntypedFormArray
                    )
            }
        }
        return sharedNetwork
    }

    /**
     * Converts Kea global parameters to a form.
     *
     * The created form is used in the {@link SharedParametersForm} for editing
     * the global Kea parameters. It comprises the metadata describing each
     * parameter.
     *
     * @param keaVersionRange a tuple with the earliest and the latest Kea version
     *        for the configured daemons.
     * @param topLevelKey: top level key in the Kea configuration, used to determine
     *        the configured server type.
     * @param configs Kea-specific global parameters parameters.
     * @returns Created form group instance.
     */
    convertKeaGlobalParametersToForm(
        keaVersionRange: [string, string],
        topLevelKey: 'Dhcp4' | 'Dhcp6',
        configs: KeaRawConfig[]
    ): FormGroup<KeaGlobalParametersForm> {
        // Common parameters.
        let form: KeaGlobalParametersForm = {
            cacheThreshold: new SharedParameterFormGroup<number>(
                {
                    type: 'number',
                    min: 0,
                    max: 1,
                    fractionDigits: 2,
                },
                configs.map((params) => new FormControl<number>(params['cache-threshold']))
            ),
            ddnsGeneratedPrefix: new SharedParameterFormGroup<string>(
                {
                    type: 'string',
                    invalidText: 'Please specify a valid prefix.',
                },
                configs.map((params) => new FormControl<string>(params['ddns-generated-prefix'], StorkValidators.fqdn))
            ),
            ddnsOverrideClientUpdate: new SharedParameterFormGroup<boolean>(
                {
                    type: 'boolean',
                },
                configs.map((params) => new FormControl<boolean>(params['ddns-override-client-update']))
            ),
            ddnsOverrideNoUpdate: new SharedParameterFormGroup<boolean>(
                {
                    type: 'boolean',
                },
                configs.map((params) => new FormControl<boolean>(params['ddns-override-no-update']))
            ),
            ddnsQualifyingSuffix: new SharedParameterFormGroup<string>(
                {
                    type: 'string',
                    invalidText: 'Please specify a valid suffix.',
                },
                configs.map((params) => new FormControl<string>(params['ddns-qualifying-suffix'], StorkValidators.fqdn))
            ),
            ddnsReplaceClientName: new SharedParameterFormGroup<string>(
                {
                    type: 'string',
                    values: ['never', 'always', 'when-not-present'],
                },
                configs.map((params) => new FormControl<string>(params['ddns-replace-client-name']))
            ),
            ddnsSendUpdates: new SharedParameterFormGroup<boolean>(
                {
                    type: 'boolean',
                },
                configs.map((params) => new FormControl<boolean>(params['ddns-send-updates']))
            ),
            ddnsUpdateOnRenew: new SharedParameterFormGroup<boolean>(
                {
                    type: 'boolean',
                },
                configs.map((params) => new FormControl<boolean>(params['ddns-update-on-renew']))
            ),
            dhcpDdnsEnableUpdates: new SharedParameterFormGroup<boolean>(
                {
                    type: 'boolean',
                    required: true,
                    clearValue: false,
                },
                configs.map((params) => new FormControl<boolean>(params?.['dhcp-ddns']?.['enable-updates']))
            ),
            dhcpDdnsServerIP: new SharedParameterFormGroup<string>(
                {
                    type: 'string',
                },
                configs.map((params) => new FormControl<string>(params?.['dhcp-ddns']?.['server-ip']))
            ),
            dhcpDdnsServerPort: new SharedParameterFormGroup<number>(
                {
                    type: 'number',
                },
                configs.map((params) => new FormControl<number>(params?.['dhcp-ddns']?.['server-port']))
            ),
            dhcpDdnsSenderIP: new SharedParameterFormGroup<string>(
                {
                    type: 'string',
                },
                configs.map((params) => new FormControl<string>(params?.['dhcp-ddns']?.['sender-ip']))
            ),
            dhcpDdnsSenderPort: new SharedParameterFormGroup<number>(
                {
                    type: 'number',
                },
                configs.map((params) => new FormControl<number>(params?.['dhcp-ddns']?.['sender-port']))
            ),
            dhcpDdnsMaxQueueSize: new SharedParameterFormGroup<number>(
                {
                    type: 'number',
                },
                configs.map((params) => new FormControl<number>(params?.['dhcp-ddns']?.['max-queue-size']))
            ),
            dhcpDdnsNcrProtocol: new SharedParameterFormGroup<string>(
                {
                    type: 'string',
                    values: ['UDP'],
                },
                configs.map((params) => new FormControl<string>(params?.['dhcp-ddns']?.['ncr-protocol']))
            ),
            dhcpDdnsNcrFormat: new SharedParameterFormGroup<string>(
                {
                    type: 'string',
                    values: ['JSON'],
                },
                configs.map((params) => new FormControl<string>(params?.['dhcp-ddns']?.['ncr-format']))
            ),
            earlyGlobalReservationsLookup: new SharedParameterFormGroup<boolean>(
                {
                    type: 'boolean',
                },
                configs.map((params) => new FormControl<boolean>(params['early-global-reservations-lookup']))
            ),
            echoClientId: new SharedParameterFormGroup<boolean>(
                {
                    type: 'boolean',
                },
                configs.map((params) => new FormControl<boolean>(params['echo-client-id']))
            ),
            expiredFlushReclaimedTimerWaitTime: new SharedParameterFormGroup<number>(
                {
                    type: 'number',
                },
                configs.map(
                    (params) =>
                        new FormControl<number>(
                            params?.['expired-leases-processing']?.['flush-reclaimed-timer-wait-time']
                        )
                )
            ),
            expiredHoldReclaimedTime: new SharedParameterFormGroup<number>(
                {
                    type: 'number',
                },
                configs.map(
                    (params) => new FormControl<number>(params?.['expired-leases-processing']?.['hold-reclaimed-time'])
                )
            ),
            expiredMaxReclaimLeases: new SharedParameterFormGroup<number>(
                {
                    type: 'number',
                },
                configs.map(
                    (params) => new FormControl<number>(params?.['expired-leases-processing']?.['max-reclaim-leases'])
                )
            ),
            expiredMaxReclaimTime: new SharedParameterFormGroup<number>(
                {
                    type: 'number',
                },
                configs.map(
                    (params) => new FormControl<number>(params?.['expired-leases-processing']?.['max-reclaim-time'])
                )
            ),
            expiredReclaimTimerWaitTime: new SharedParameterFormGroup<number>(
                {
                    type: 'number',
                },
                configs.map(
                    (params) =>
                        new FormControl<number>(params?.['expired-leases-processing']?.['reclaim-timer-wait-time'])
                )
            ),
            expiredUnwarnedReclaimCycles: new SharedParameterFormGroup<number>(
                {
                    type: 'number',
                },
                configs.map(
                    (params) =>
                        new FormControl<number>(params?.['expired-leases-processing']?.['unwarned-reclaim-cycles'])
                )
            ),
            hostReservationIdentifiers: new SharedParameterFormGroup<string[]>(
                {
                    type: 'string',
                    isArray: true,
                    values: ['circuit-id', 'hw-address', 'duid', 'client-id'],
                },
                configs.map((params) => new FormControl<string[]>(params['host-reservation-identifiers']))
            ),
            reservationsGlobal: new SharedParameterFormGroup<boolean>(
                {
                    type: 'boolean',
                },
                configs.map((params) => new FormControl<boolean>(params['reservations-global']))
            ),
            reservationsInSubnet: new SharedParameterFormGroup<boolean>(
                {
                    type: 'boolean',
                },
                configs.map((params) => new FormControl<boolean>(params['reservations-in-subnet']))
            ),
            reservationsOutOfPool: new SharedParameterFormGroup<boolean>(
                {
                    type: 'boolean',
                },
                configs.map((params) => new FormControl<boolean>(params['reservations-out-of-pool']))
            ),
            allocator: new SharedParameterFormGroup<string>(
                {
                    type: 'string',
                    values: ['iterative', 'random', 'flq'],
                },
                configs.map((params) => new FormControl<string>(params['allocator']))
            ),
        }
        if (!keaVersionRange || lt(keaVersionRange[0], '2.5.0')) {
            form.ddnsUseConflictResolution = new SharedParameterFormGroup<boolean>(
                {
                    type: 'boolean',
                    versionUpperBound: !keaVersionRange || gte(keaVersionRange[1], '2.5.0') ? '2.5.0' : undefined,
                },
                configs.map((params) => new FormControl<boolean>(params['ddns-use-conflict-resolution']))
            )
        }
        if (!keaVersionRange || gte(keaVersionRange[1], '2.5.0')) {
            form.ddnsConflictResolutionMode = new SharedParameterFormGroup<string>(
                {
                    type: 'string',
                    values: [
                        'check-with-dhcid',
                        'no-check-with-dhcid',
                        'check-exists-with-dhcid',
                        'no-check-without-dhcid',
                    ],
                    versionLowerBound: !keaVersionRange || lt(keaVersionRange[0], '2.5.0') ? '2.5.0' : undefined,
                },
                configs.map((params) => new FormControl<string>(params['ddns-conflict-resolution-mode']))
            )
        }
        switch (topLevelKey) {
            // DHCPv4 parameters.
            case 'Dhcp4':
                form.authoritative = new SharedParameterFormGroup<boolean>(
                    {
                        type: 'boolean',
                    },
                    configs.map((params) => new FormControl<boolean>(params['authoritative']))
                )
                break
            // DHCPv6 parameters.
            default:
                form.pdAllocator = new SharedParameterFormGroup<string>(
                    {
                        type: 'string',
                        values: ['iterative', 'random', 'flq'],
                    },
                    configs.map((params) => new FormControl<string>(params['pd-allocator']))
                )
        }
        let formGroup = new FormGroup<KeaGlobalParametersForm>(form)
        return formGroup
    }

    /**
     * Converts Kea global configuration to a form.
     *
     * The created form is used in the {@link SharedParametersForm} for editing
     * the global Kea configuration. It comprises the metadata describing each
     * parameter.
     *
     * @param keaVersionRange a tuple with the earliest and the latest Kea version
     *        for the configured daemons.
     * @param parameters Kea-specific global parameters parameters.
     * @returns Created form group instance.
     */
    convertKeaGlobalConfigurationToForm(
        keaVersionRange: [string, string],
        configs: KeaDaemonConfig[]
    ): FormGroup<KeaGlobalConfigurationForm> {
        if (!configs?.length) {
            return null
        }
        const topLevelKeys = configs.map((config) => {
            if (config.config?.hasOwnProperty('Dhcp4')) {
                return 'Dhcp4'
            }
            if (config.config?.hasOwnProperty('Dhcp6')) {
                return 'Dhcp6'
            }
            return null
        })
        if (!topLevelKeys.every((key) => key === topLevelKeys[0]) || topLevelKeys.every((key) => key == null)) {
            return null
        }
        const innerConfigs = configs.map((config) => config.config[topLevelKeys[0]])
        const formGroup = new FormGroup<KeaGlobalConfigurationForm>({
            parameters: this.convertKeaGlobalParametersToForm(keaVersionRange, topLevelKeys[0], innerConfigs),
            options: new FormGroup({
                unlocked: new FormControl(hasDifferentGlobalLevelOptions(configs)),
                data: new UntypedFormArray(
                    configs.map((c) =>
                        this.optionService.convertOptionsToForm(
                            topLevelKeys[0] === 'Dhcp4' ? IPType.IPv4 : IPType.IPv6,
                            c.options?.options
                        )
                    ) || []
                ),
            }),
        })
        return formGroup
    }

    /**
     * Converts a form holding global configuration data to an API instance.
     *
     * It currently only converts the simple DHCP parameters and options.
     * @param daemons an array of daemons including their software versions.
     * @param form A form comprising global configuration data.
     * @returns A configuration instance converted from the form.
     */
    convertFormToKeaGlobalParameters(
        daemons: VersionedDaemon[],
        form: FormGroup<KeaGlobalConfigurationForm>,
        ipType: IPType
    ): KeaConfigurableGlobalParameters[] {
        const parametersForm = form.get('parameters') as FormGroup<KeaGlobalParametersForm>

        const convertedParameters = this.convertFormToKeaParameters<
            KeaGlobalParametersForm,
            KeaConfigurableGlobalParameters
        >(daemons, parametersForm)

        const options = form.get('options') as UntypedFormGroup
        for (let i = 0; i < convertedParameters.length; i++) {
            const data = options.get('data') as UntypedFormArray
            if (data?.length > i) {
                convertedParameters[i].options = this.optionService.convertFormToOptions(
                    ipType,
                    data.at(!!options.get('unlocked')?.value ? i : 0) as UntypedFormArray
                )
            }
        }

        return convertedParameters
    }

    /**
     * Adjusts the form to the new daemons selection.
     *
     * This function is invoked when a user selected or unselected daemons
     * associated with a subnet or a pool. New form controls are added when
     * new daemons are selected. Existing form controls are removed when the
     * daemons are unselected.
     *
     * @param formGroup form group holding the subnet or pool data.
     * @param toggledDaemonIndex index of the selected or unselected daemon.
     * @param prevSelectedDaemonsNum a number of previously selected daemons.
     */
    adjustFormForSelectedDaemons(
        formGroup: UntypedFormGroup,
        toggledDaemonIndex: number,
        prevSelectedDaemonsNum: number
    ): void {
        // If the number of daemons hasn't changed, there is nothing more to do.
        const selectedDaemons = formGroup.get('selectedDaemons').value ?? []
        if (prevSelectedDaemonsNum === selectedDaemons.length) {
            return
        }
        const pools = formGroup.get('pools') as FormArray<FormGroup<AddressPoolForm>>
        if (pools) {
            for (const pool of pools.controls) {
                pool.get('selectedDaemons').setValue(
                    pool.get('selectedDaemons').value.filter((sd) => selectedDaemons.find((found) => found === sd))
                )
            }
        }

        const prefixPools = formGroup.get('prefixPools') as FormArray<FormGroup<PrefixPoolForm>>
        if (prefixPools) {
            for (const pool of prefixPools.controls) {
                pool.get('selectedDaemons').setValue(
                    pool.get('selectedDaemons').value.filter((sd) => selectedDaemons.find((found) => found === sd))
                )
            }
        }

        // Get form controls pertaining to the servers before the selection change.
        const parameters = formGroup.get('parameters') as FormGroup<KeaSubnetParametersForm | KeaPoolParametersForm>

        // Iterate over the controls holding the configuration parameters.
        for (const key of Object.keys(parameters?.controls)) {
            const values = parameters.get(key).get('values') as UntypedFormArray
            const unlocked = parameters.get(key).get('unlocked') as UntypedFormControl
            if (selectedDaemons.length < prevSelectedDaemonsNum) {
                // We have removed a daemon from a list. Let's remove the
                // controls pertaining to the removed daemon.
                if (values?.length > selectedDaemons.length) {
                    // If we have the index of the removed daemon let's remove the
                    // controls appropriate for this daemon. This will preserve the
                    // values specified for any other daemons. Otherwise, let's remove
                    // the last control.
                    if (toggledDaemonIndex >= 0 && toggledDaemonIndex < values.controls.length) {
                        values.controls.splice(toggledDaemonIndex, 1)
                    } else {
                        values.controls.splice(selectedDaemons.length)
                    }
                }
                // Clear the unlock flag when there is only one server left.
                if (values?.length < 2) {
                    unlocked?.setValue(false)
                    unlocked?.disable()
                }
            } else {
                // If we have added a new server we should populate some values
                // for this server. Let's use the values associated with the first
                // server. We should have at least one server at this point but
                // let's double check.
                if (values?.length > 0 && values.length < selectedDaemons.length) {
                    values.push(this.genericFormService.cloneControl(values.at(0)))
                    unlocked?.enable()
                }
            }
        }

        // Handle the daemons selection change for the DHCP options.
        const data = formGroup.get('options.data') as UntypedFormArray
        if (data?.controls?.length > 0) {
            const unlocked = formGroup.get('options')?.get('unlocked') as UntypedFormControl
            if (selectedDaemons.length < prevSelectedDaemonsNum) {
                // If we have the index of the removed daemon let's remove the
                // controls appropriate for this daemon. This will preserve the
                // values specified for any other daemons. Otherwise, let's remove
                // the last control.
                if (toggledDaemonIndex >= 0 && toggledDaemonIndex < data.controls.length && unlocked.value) {
                    data.controls.splice(toggledDaemonIndex, 1)
                } else {
                    data.controls.splice(selectedDaemons.length)
                }
                // Clear the unlock flag when there is only one server left.
                if (data.controls.length < 2) {
                    unlocked?.setValue(false)
                    unlocked?.disable()
                }
            } else {
                if (data.controls.length > 0 && data.controls.length < selectedDaemons.length) {
                    data.push(this.optionService.cloneControl(data.controls[0]))
                    unlocked?.enable()
                }
            }
        }
    }
}
