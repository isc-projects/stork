import { ComponentFixture, TestBed, fakeAsync, tick } from '@angular/core/testing'

import { SubnetFormComponent } from './subnet-form.component'
import { ButtonModule } from 'primeng/button'
import { CheckboxModule } from 'primeng/checkbox'
import { ChipsModule } from 'primeng/chips'
import { DividerModule } from 'primeng/divider'
import { DropdownModule } from 'primeng/dropdown'
import { FieldsetModule } from 'primeng/fieldset'
import { FormArray, FormGroup, FormsModule, ReactiveFormsModule, UntypedFormArray } from '@angular/forms'
import { HttpClientModule } from '@angular/common/http'
import { InputNumberModule } from 'primeng/inputnumber'
import { MultiSelectModule } from 'primeng/multiselect'
import { NoopAnimationsModule } from '@angular/platform-browser/animations'
import { TableModule } from 'primeng/table'
import { TagModule } from 'primeng/tag'
import { TriStateCheckboxModule } from 'primeng/tristatecheckbox'
import { OverlayPanelModule } from 'primeng/overlaypanel'
import { ProgressSpinnerModule } from 'primeng/progressspinner'
import { RouterTestingModule } from '@angular/router/testing'
import { SplitButtonModule } from 'primeng/splitbutton'
import { ToastModule } from 'primeng/toast'
import { MessageService } from 'primeng/api'
import { of, throwError } from 'rxjs'
import { DHCPService } from '../backend'
import { AddressPoolForm, KeaSubnetParametersForm, PrefixPoolForm } from '../forms/subnet-set-form.service'
import { SharedParametersFormComponent } from '../shared-parameters-form/shared-parameters-form.component'
import { DhcpOptionSetFormComponent } from '../dhcp-option-set-form/dhcp-option-set-form.component'
import { DhcpOptionFormComponent } from '../dhcp-option-form/dhcp-option-form.component'
import { DhcpClientClassSetFormComponent } from '../dhcp-client-class-set-form/dhcp-client-class-set-form.component'
import { EntityLinkComponent } from '../entity-link/entity-link.component'
import { HelpTipComponent } from '../help-tip/help-tip.component'
import { By } from '@angular/platform-browser'
import { MessagesModule } from 'primeng/messages'
import { AddressPoolFormComponent } from '../address-pool-form/address-pool-form.component'
import { AccordionModule } from 'primeng/accordion'
import { PrefixPoolFormComponent } from '../prefix-pool-form/prefix-pool-form.component'
import { ArrayValueSetFormComponent } from '../array-value-set-form/array-value-set-form.component'

describe('SubnetFormComponent', () => {
    let component: SubnetFormComponent
    let fixture: ComponentFixture<SubnetFormComponent>
    let dhcpApi: DHCPService
    let messageService: MessageService

    let cannedResponseBeginSubnet4: any = {
        id: 123,
        subnet: {
            id: 123,
            subnet: '192.0.2.0/24',
            sharedNetwork: 'floor3',
            sharedNetworkId: 3,
            localSubnets: [
                {
                    id: 123,
                    appId: 234,
                    daemonId: 1,
                    appName: 'server 1',
                    machineAddress: '10.1.1.1.',
                    machineHostname: 'myhost.example.org',
                    pools: [
                        {
                            pool: '192.0.2.10-192.0.2.100',
                            keaConfigPoolParameters: {
                                clientClass: 'foo',
                                requireClientClasses: ['foo', 'bar'],
                                options: [],
                                optionsHash: '',
                            },
                        },
                    ],
                    prefixDelegationPools: [],
                    keaConfigSubnetParameters: {
                        subnetLevelParameters: {
                            allocator: 'random',
                            relay: {
                                ipAddresses: ['192.1.1.1'],
                            },
                            options: [
                                {
                                    alwaysSend: true,
                                    code: 5,
                                    encapsulate: '',
                                    fields: [
                                        {
                                            fieldType: 'ipv4-address',
                                            values: ['192.0.2.1'],
                                        },
                                    ],
                                    options: [],
                                    universe: 4,
                                },
                            ],
                            optionsHash: '123',
                        },
                    },
                },
                {
                    id: 123,
                    appId: 234,
                    daemonId: 2,
                    appName: 'server 2',
                    machineAddress: '10.1.1.1.',
                    machineHostname: 'myhost.example.org',
                    pools: [
                        {
                            pool: '192.0.2.10-192.0.2.100',
                            keaConfigPoolParameters: {
                                clientClass: 'foo',
                                requireClientClasses: ['foo', 'bar'],
                                options: [],
                                optionsHash: '',
                            },
                        },
                    ],
                    prefixDelegationPools: [],
                    keaConfigSubnetParameters: {
                        subnetLevelParameters: {
                            allocator: 'iterative',
                            relay: {
                                ipAddresses: ['192.1.1.1'],
                            },
                            options: [
                                {
                                    alwaysSend: true,
                                    code: 5,
                                    encapsulate: '',
                                    fields: [
                                        {
                                            fieldType: 'ipv4-address',
                                            values: ['192.0.2.2'],
                                        },
                                    ],
                                    options: [],
                                    universe: 4,
                                },
                            ],
                            optionsHash: '234',
                        },
                    },
                },
            ],
        },
        daemons: [
            {
                id: 1,
                name: 'dhcp4',
                app: {
                    name: 'first',
                },
            },
            {
                id: 3,
                name: 'dhcp6',
                app: {
                    name: 'first',
                },
            },
            {
                id: 2,
                name: 'dhcp4',
                app: {
                    name: 'second',
                },
            },
            {
                id: 4,
                name: 'dhcp6',
                app: {
                    name: 'second',
                },
            },
            {
                id: 5,
                name: 'dhcp6',
                app: {
                    name: 'third',
                },
            },
        ],
        sharedNetworks4: [
            {
                id: 1,
                name: 'floor1',
                localSharedNetworks: [
                    {
                        daemonId: 1,
                    },
                ],
            },
            {
                id: 2,
                name: 'floor2',
                localSharedNetworks: [
                    {
                        daemonId: 2,
                    },
                ],
            },
            {
                id: 3,
                name: 'floor3',
                localSharedNetworks: [
                    {
                        daemonId: 1,
                    },
                    {
                        daemonId: 2,
                    },
                ],
            },
        ],
        sharedNetworks6: [],
    }

    let cannedResponseBeginSubnet6: any = {
        id: 345,
        subnet: {
            id: 234,
            subnet: '2001:db8:1::/64',
            localSubnets: [
                {
                    id: 234,
                    appId: 345,
                    daemonId: 3,
                    appName: 'server 1',
                    machineAddress: '10.1.1.1',
                    machineHostname: 'myhost.example.org',
                    pools: [
                        {
                            pool: '2001:db8:1::10-2001:db8:1::100',
                            keaConfigPoolParameters: {
                                clientClass: 'foo',
                                requireClientClasses: ['foo', 'bar'],
                                options: [],
                                optionsHash: '',
                            },
                        },
                    ],
                    prefixDelegationPools: [
                        {
                            prefix: '3000::/48',
                            delegatedLength: 64,
                            keaConfigPoolParameters: {
                                clientClass: 'foo',
                                requireClientClasses: [],
                                options: [],
                            },
                        },
                    ],
                    keaConfigSubnetParameters: {
                        subnetLevelParameters: {
                            allocator: 'random',
                            options: [
                                {
                                    alwaysSend: true,
                                    code: 23,
                                    encapsulate: '',
                                    fields: [
                                        {
                                            fieldType: 'ipv6-address',
                                            values: ['2001:db8:2::6789'],
                                        },
                                    ],
                                    options: [],
                                    universe: 6,
                                },
                            ],
                            optionsHash: '123',
                        },
                    },
                },
                {
                    id: 345,
                    appId: 456,
                    daemonId: 4,
                    appName: 'server 2',
                    machineAddress: '10.1.1.1.',
                    machineHostname: 'myhost.example.org',
                    pools: [
                        {
                            pool: '2001:db8:1::10-2001:db8:1::100',
                            keaConfigPoolParameters: {
                                clientClass: 'foo',
                                requireClientClasses: ['foo', 'bar'],
                                options: [],
                                optionsHash: '',
                            },
                        },
                    ],
                    prefixDelegationPools: [
                        {
                            prefix: '3000::/48',
                            delegatedLength: 64,
                            keaConfigPoolParameters: {
                                clientClass: 'foo',
                                requireClientClasses: [],
                                options: [],
                            },
                        },
                    ],
                    keaConfigSubnetParameters: {
                        subnetLevelParameters: {
                            pdAllocator: 'iterative',
                            options: [
                                {
                                    alwaysSend: true,
                                    code: 23,
                                    encapsulate: '',
                                    fields: [
                                        {
                                            fieldType: 'ipv6-address',
                                            values: ['2001:db8:2::6789'],
                                        },
                                    ],
                                    options: [],
                                    universe: 6,
                                },
                            ],
                            optionsHash: '123',
                        },
                    },
                },
            ],
        },
        daemons: [
            {
                id: 1,
                name: 'dhcp4',
                app: {
                    name: 'first',
                },
            },
            {
                id: 3,
                name: 'dhcp6',
                app: {
                    name: 'first',
                },
            },
            {
                id: 2,
                name: 'dhcp4',
                app: {
                    name: 'second',
                },
            },
            {
                id: 4,
                name: 'dhcp6',
                app: {
                    name: 'second',
                },
            },
            {
                id: 5,
                name: 'dhcp6',
                app: {
                    name: 'third',
                },
            },
        ],
        sharedNetworks4: [],
        sharedNetworks6: [
            {
                id: 1,
                name: 'floor1',
                localSharedNetworks: [
                    {
                        daemonId: 3,
                    },
                ],
            },
            {
                id: 2,
                name: 'floor2',
                localSharedNetworks: [
                    {
                        daemonId: 4,
                    },
                ],
            },
            {
                id: 3,
                name: 'floor3',
                localSharedNetworks: [
                    {
                        daemonId: 5,
                    },
                ],
            },
        ],
    }

    beforeEach(async () => {
        await TestBed.configureTestingModule({
            imports: [
                AccordionModule,
                ButtonModule,
                CheckboxModule,
                ChipsModule,
                DividerModule,
                DropdownModule,
                FieldsetModule,
                FormsModule,
                HttpClientModule,
                InputNumberModule,
                MessagesModule,
                MultiSelectModule,
                NoopAnimationsModule,
                TableModule,
                TagModule,
                TriStateCheckboxModule,
                OverlayPanelModule,
                ProgressSpinnerModule,
                ReactiveFormsModule,
                RouterTestingModule,
                SplitButtonModule,
                ToastModule,
            ],
            declarations: [
                AddressPoolFormComponent,
                ArrayValueSetFormComponent,
                DhcpClientClassSetFormComponent,
                DhcpOptionFormComponent,
                DhcpOptionSetFormComponent,
                EntityLinkComponent,
                HelpTipComponent,
                PrefixPoolFormComponent,
                SharedParametersFormComponent,
                SubnetFormComponent,
            ],
            providers: [MessageService],
        }).compileComponents()

        fixture = TestBed.createComponent(SubnetFormComponent)
        component = fixture.componentInstance
        component.subnetId = 123
        dhcpApi = fixture.debugElement.injector.get(DHCPService)
        messageService = fixture.debugElement.injector.get(MessageService)
        fixture.detectChanges()
    })

    it('should create', () => {
        expect(component).toBeTruthy()
    })

    it('should open a form for creating an IPv4 subnet', fakeAsync(() => {
        spyOn(dhcpApi, 'createSubnetBegin').and.returnValue(of(cannedResponseBeginSubnet4))
        component.subnetId = undefined
        component.ngOnInit()
        tick()
        fixture.detectChanges()
        expect(component.state).toBeTruthy()
        expect(component.state.preserved).toBeFalse()
        expect(component.state.transactionId).toBe(123)
        expect(component.state.group).toBeTruthy()
        expect(component.state.allDaemons.length).toBe(5)
        expect(component.state.filteredDaemons.length).toBe(5)
        expect(component.state.dhcpv4).toBeFalse()
        expect(component.state.dhcpv6).toBeFalse()
        expect(component.state.wizard).toBeTrue()

        expect(fixture.debugElement.query(By.css('[label="Proceed"]'))).toBeTruthy()
        expect(fixture.debugElement.query(By.css('[label="Cancel"]'))).toBeTruthy()
        expect(component.state.group.get('subnet').disabled).toBeFalse()
        expect(component.state.group.get('subnet').invalid).toBeTrue()

        component.state.group.get('subnet').setValue('192.0.2.0/24')
        expect(component.state.group.get('subnet').invalid).toBeFalse()

        component.onSubnetProceed()
        fixture.detectChanges()

        expect(component.state.group.get('subnet').disabled).toBeTrue()
        expect(component.state.wizard).toBeFalse()

        const selectedDaemons = [1, 2]
        component.state.group.get('selectedDaemons').setValue(selectedDaemons)
        selectedDaemons.forEach((id) => {
            component.onDaemonsChange({
                itemValue: id,
            })
        })
        tick()
        fixture.detectChanges()

        // The daemons selection should be enabled, so there should be no helptip.
        expect(fixture.debugElement.query(By.css('[title="Help for disabled servers selection"]'))).toBeFalsy()

        // Set shared network. It should result in disabling the daemons selection.
        component.state.group.get('sharedNetwork').setValue(3)
        component.onSharedNetworkChange({
            value: 3,
        })
        fixture.detectChanges()
        expect(component.state.group.get('selectedDaemons')?.disabled).toBeTrue()

        // It should also display the helptip.
        expect(fixture.debugElement.query(By.css('[title="Help for disabled servers selection"]'))).toBeTruthy()

        const okResp: any = {
            status: 200,
        }
        spyOn(dhcpApi, 'createSubnetSubmit').and.returnValue(of(okResp))
        spyOn(component.formSubmit, 'emit')
        spyOn(messageService, 'add')
        component.onSubmit()
        tick()
        fixture.detectChanges()

        const subnet = {
            subnet: '192.0.2.0/24',
            sharedNetworkId: 3,
            sharedNetwork: 'floor3',
            localSubnets: [
                {
                    daemonId: 1,
                    pools: [],
                    prefixDelegationPools: [],
                    keaConfigSubnetParameters: {
                        subnetLevelParameters: {
                            options: [],
                        },
                    },
                },
                {
                    daemonId: 2,
                    pools: [],
                    prefixDelegationPools: [],
                    keaConfigSubnetParameters: {
                        subnetLevelParameters: {
                            options: [],
                        },
                    },
                },
            ],
        }

        expect(dhcpApi.createSubnetSubmit).toHaveBeenCalledWith(component.state.transactionId, subnet)
        expect(component.formSubmit.emit).toHaveBeenCalled()
        expect(messageService.add).toHaveBeenCalled()
    }))

    it('should open a form for creating an IPv6 subnet', fakeAsync(() => {
        spyOn(dhcpApi, 'createSubnetBegin').and.returnValue(of(cannedResponseBeginSubnet4))
        component.subnetId = undefined
        component.ngOnInit()
        tick()
        fixture.detectChanges()
        expect(component.state).toBeTruthy()
        expect(component.state.preserved).toBeFalse()
        expect(component.state.transactionId).toBe(123)
        expect(component.state.group).toBeTruthy()
        expect(component.state.allDaemons.length).toBe(5)
        expect(component.state.filteredDaemons.length).toBe(5)
        expect(component.state.dhcpv4).toBeFalse()
        expect(component.state.dhcpv6).toBeFalse()
        expect(component.state.wizard).toBeTrue()

        const button = fixture.debugElement.query(By.css('[label="Proceed"]'))
        expect(button).toBeTruthy()
        expect(component.state.group.get('subnet').disabled).toBeFalse()
        expect(component.state.group.get('subnet').invalid).toBeTrue()

        component.state.group.get('subnet').setValue('2001:db8:3::/64')
        expect(component.state.group.get('subnet').invalid).toBeFalse()

        component.onSubnetProceed()
        fixture.detectChanges()

        expect(component.state.group.get('subnet').disabled).toBeTrue()
        expect(component.state.wizard).toBeFalse()

        const selectedDaemons = [3, 4]
        component.state.group.get('selectedDaemons').setValue(selectedDaemons)
        selectedDaemons.forEach((id) => {
            component.onDaemonsChange({
                itemValue: id,
            })
        })
        tick()

        // Ensure there is no shared network selected.
        component.onSharedNetworkChange({
            value: null,
        })
        fixture.detectChanges()

        // Since shared network is not selected, the daemons selection should
        // be enabled.
        expect(component.state.group.get('selectedDaemons')?.disabled).toBeFalse()

        const okResp: any = {
            status: 200,
        }
        spyOn(dhcpApi, 'createSubnetSubmit').and.returnValue(of(okResp))
        spyOn(component.formSubmit, 'emit')
        spyOn(messageService, 'add')
        component.onSubmit()
        tick()
        fixture.detectChanges()

        const subnet = {
            subnet: '2001:db8:3::/64',
            sharedNetworkId: null,
            localSubnets: [
                {
                    daemonId: 3,
                    pools: [],
                    prefixDelegationPools: [],
                    keaConfigSubnetParameters: {
                        subnetLevelParameters: {
                            options: [],
                        },
                    },
                },
                {
                    daemonId: 4,
                    pools: [],
                    prefixDelegationPools: [],
                    keaConfigSubnetParameters: {
                        subnetLevelParameters: {
                            options: [],
                        },
                    },
                },
            ],
        }

        expect(dhcpApi.createSubnetSubmit).toHaveBeenCalledWith(component.state.transactionId, subnet)
        expect(component.formSubmit.emit).toHaveBeenCalled()
        expect(messageService.add).toHaveBeenCalled()
    }))

    it('should open a form for updating IPv4 subnet', fakeAsync(() => {
        spyOn(dhcpApi, 'updateSubnetBegin').and.returnValue(of(cannedResponseBeginSubnet4))
        component.subnetId = 123
        component.ngOnInit()
        tick()
        fixture.detectChanges()

        expect(component.state).toBeTruthy()
        expect(component.state.preserved).toBeFalse()
        expect(component.state.transactionId).toBe(123)
        expect(component.state.group).toBeTruthy()
        expect(component.state.allDaemons.length).toBe(5)
        expect(component.state.filteredDaemons.length).toBe(2)
        expect(component.state.dhcpv4).toBeTrue()
        expect(component.state.dhcpv6).toBeFalse()

        const okResp: any = {
            status: 200,
        }
        spyOn(dhcpApi, 'updateSubnetSubmit').and.returnValue(of(okResp))
        spyOn(component.formSubmit, 'emit')
        spyOn(messageService, 'add')
        component.onSubmit()
        tick()
        fixture.detectChanges()

        const subnet = {
            id: 123,
            subnet: '192.0.2.0/24',
            sharedNetworkId: 3,
            sharedNetwork: 'floor3',
            localSubnets: [
                {
                    id: 123,
                    daemonId: 1,
                    pools: [
                        {
                            pool: '192.0.2.10-192.0.2.100',
                            keaConfigPoolParameters: {
                                clientClass: 'foo',
                                requireClientClasses: ['foo', 'bar'],
                                options: [],
                            },
                        },
                    ],
                    prefixDelegationPools: [],
                    keaConfigSubnetParameters: {
                        subnetLevelParameters: {
                            allocator: 'random',
                            relay: {
                                ipAddresses: ['192.1.1.1'],
                            },
                            options: [
                                {
                                    alwaysSend: true,
                                    code: 5,
                                    encapsulate: '',
                                    fields: [
                                        {
                                            fieldType: 'ipv4-address',
                                            values: ['192.0.2.1'],
                                        },
                                    ],
                                    options: [],
                                    universe: 4,
                                },
                            ],
                        },
                    },
                    userContext: null,
                },
                {
                    id: 123,
                    daemonId: 2,
                    pools: [
                        {
                            pool: '192.0.2.10-192.0.2.100',
                            keaConfigPoolParameters: {
                                clientClass: 'foo',
                                requireClientClasses: ['foo', 'bar'],
                                options: [],
                            },
                        },
                    ],
                    prefixDelegationPools: [],
                    keaConfigSubnetParameters: {
                        subnetLevelParameters: {
                            allocator: 'iterative',
                            relay: {
                                ipAddresses: ['192.1.1.1'],
                            },
                            options: [
                                {
                                    alwaysSend: true,
                                    code: 5,
                                    encapsulate: '',
                                    fields: [
                                        {
                                            fieldType: 'ipv4-address',
                                            values: ['192.0.2.2'],
                                        },
                                    ],
                                    options: [],
                                    universe: 4,
                                },
                            ],
                        },
                    },
                    userContext: null,
                },
            ],
        }

        expect(dhcpApi.updateSubnetSubmit).toHaveBeenCalledWith(
            component.subnetId,
            component.state.transactionId,
            subnet
        )
        expect(component.formSubmit.emit).toHaveBeenCalled()
        expect(messageService.add).toHaveBeenCalled()
    }))

    it('should open a form for updating IPv6 subnet', fakeAsync(() => {
        spyOn(dhcpApi, 'updateSubnetBegin').and.returnValue(of(cannedResponseBeginSubnet6))
        component.subnetId = 234
        component.ngOnInit()
        tick()
        fixture.detectChanges()

        expect(component.state).toBeTruthy()
        expect(component.state.preserved).toBeFalse()
        expect(component.state.transactionId).toBe(345)
        expect(component.state.group).toBeTruthy()
        expect(component.state.allDaemons.length).toBe(5)
        expect(component.state.filteredDaemons.length).toBe(3)
        expect(component.state.dhcpv4).toBeFalse()
        expect(component.state.dhcpv6).toBeTrue()

        const okResp: any = {
            status: 200,
        }
        spyOn(dhcpApi, 'updateSubnetSubmit').and.returnValue(of(okResp))
        spyOn(component.formSubmit, 'emit')
        spyOn(messageService, 'add')
        component.onSubmit()
        tick()
        fixture.detectChanges()

        const subnet = {
            id: 234,
            subnet: '2001:db8:1::/64',
            sharedNetworkId: null,
            localSubnets: [
                {
                    id: 234,
                    daemonId: 3,
                    pools: [
                        {
                            pool: '2001:db8:1::10-2001:db8:1::100',
                            keaConfigPoolParameters: {
                                clientClass: 'foo',
                                requireClientClasses: ['foo', 'bar'],
                                options: [],
                            },
                        },
                    ],
                    prefixDelegationPools: [
                        {
                            prefix: '3000::/48',
                            delegatedLength: 64,
                            excludedPrefix: null,
                            keaConfigPoolParameters: {
                                clientClass: 'foo',
                                requireClientClasses: [],
                                options: [],
                            },
                        },
                    ],
                    userContext: null,
                    keaConfigSubnetParameters: {
                        subnetLevelParameters: {
                            allocator: 'random',
                            options: [
                                {
                                    alwaysSend: true,
                                    code: 23,
                                    encapsulate: '',
                                    fields: [
                                        {
                                            fieldType: 'ipv6-address',
                                            values: ['2001:db8:2::6789'],
                                        },
                                    ],
                                    options: [],
                                    universe: 6,
                                },
                            ],
                        },
                    },
                },
                {
                    id: 345,
                    daemonId: 4,
                    pools: [
                        {
                            pool: '2001:db8:1::10-2001:db8:1::100',
                            keaConfigPoolParameters: {
                                clientClass: 'foo',
                                requireClientClasses: ['foo', 'bar'],
                                options: [],
                            },
                        },
                    ],
                    prefixDelegationPools: [
                        {
                            prefix: '3000::/48',
                            delegatedLength: 64,
                            excludedPrefix: null,
                            keaConfigPoolParameters: {
                                clientClass: 'foo',
                                requireClientClasses: [],
                                options: [],
                            },
                        },
                    ],
                    userContext: null,
                    keaConfigSubnetParameters: {
                        subnetLevelParameters: {
                            pdAllocator: 'iterative',
                            options: [
                                {
                                    alwaysSend: true,
                                    code: 23,
                                    encapsulate: '',
                                    fields: [
                                        {
                                            fieldType: 'ipv6-address',
                                            values: ['2001:db8:2::6789'],
                                        },
                                    ],
                                    options: [],
                                    universe: 6,
                                },
                            ],
                        },
                    },
                },
            ],
        }

        expect(dhcpApi.updateSubnetSubmit).toHaveBeenCalledWith(
            component.subnetId,
            component.state.transactionId,
            subnet
        )
        expect(component.formSubmit.emit).toHaveBeenCalled()
        expect(messageService.add).toHaveBeenCalled()
    }))

    it('should initialize the form controls for an IPv4 subnet', fakeAsync(() => {
        spyOn(dhcpApi, 'updateSubnetBegin').and.returnValue(of(cannedResponseBeginSubnet4))
        component.subnetId = 123
        component.ngOnInit()
        tick()
        // We cannot use contains() function here because it returns false for
        // disabled controls.
        expect(component.state).toBeTruthy()
        expect(component.state.group).toBeTruthy()
        expect(component.state.group.get('subnet')).toBeTruthy()
        expect(component.state.group.get('sharedNetwork')).toBeTruthy()
        expect(component.state.group.get('pools')).toBeTruthy()
        expect(component.state.group.contains('parameters')).toBeTrue()
        expect(component.state.group.contains('options')).toBeTrue()

        expect(component.state.group.get('subnet').value).toBe('192.0.2.0/24')
        expect(component.state.group.get('sharedNetwork').value).toBe(3)

        const pools = component.state.group.get('pools') as FormArray<FormGroup<AddressPoolForm>>
        expect(pools).toBeTruthy()
        expect(pools.length).toBe(1)
        expect(pools.get('0.range.start')?.value).toBe('192.0.2.10')
        expect(pools.get('0.range.end')?.value).toBe('192.0.2.100')

        const parameters = component.state.group.get('parameters') as FormGroup<KeaSubnetParametersForm>
        expect(parameters).toBeTruthy()
        expect(parameters.get('allocator.unlocked')?.value).toBeTrue()
        expect(parameters.get('allocator.values')?.value).toEqual(['random', 'iterative'])

        const options = component.state.group.get('options')
        expect(options).toBeTruthy()
        expect(options.get('unlocked')?.value).toBeTrue()
        const data = options.get('data') as UntypedFormArray
        expect(data).toBeTruthy()
        expect(data.length).toBe(2)
        expect(data.get('0.0.optionCode')?.value).toBe(5)
        expect(data.get('1.0.optionCode')?.value).toBe(5)
    }))

    it('should initialize the form controls for an IPv6 subnet', fakeAsync(() => {
        spyOn(dhcpApi, 'updateSubnetBegin').and.returnValue(of(cannedResponseBeginSubnet6))
        component.subnetId = 234
        component.ngOnInit()
        tick()
        // We cannot use contains() function here because it returns false for
        // disabled controls.
        expect(component.state).toBeTruthy()
        expect(component.state.group).toBeTruthy()
        expect(component.state.group.get('subnet')).toBeTruthy()
        expect(component.state.group.get('sharedNetwork')).toBeTruthy()
        expect(component.state.group.get('pools')).toBeTruthy()
        expect(component.state.group.contains('parameters')).toBeTrue()
        expect(component.state.group.contains('options')).toBeTrue()

        expect(component.state.group.get('subnet').value).toBe('2001:db8:1::/64')
        expect(component.state.group.get('sharedNetwork').value).toBeFalsy()

        const pools = component.state.group.get('pools') as FormArray<FormGroup<AddressPoolForm>>
        expect(pools).toBeTruthy()
        expect(pools.length).toBe(1)
        expect(pools.get('0.range.start')?.value).toBe('2001:db8:1::10')
        expect(pools.get('0.range.end')?.value).toBe('2001:db8:1::100')

        const prefixPools = component.state.group.get('prefixPools') as FormArray<FormGroup<PrefixPoolForm>>
        expect(prefixPools).toBeTruthy()
        expect(prefixPools.length).toBe(1)
        expect(prefixPools.get('0.prefixes.prefix')?.value).toBe('3000::/48')
        expect(prefixPools.get('0.prefixes.delegatedLength')?.value).toBe(64)

        const parameters = component.state.group.get('parameters') as FormGroup<KeaSubnetParametersForm>
        expect(parameters).toBeTruthy()
        expect(parameters.get('allocator.unlocked')?.value).toBeTrue()
        expect(parameters.get('allocator.values')?.value).toEqual(['random', null])

        const options = component.state.group.get('options')
        expect(options).toBeTruthy()
        expect(options.get('unlocked')?.value).toBeFalse()
        const data = options.get('data') as UntypedFormArray
        expect(data?.length).toBe(2)
        expect(data.get('0.0.optionCode')?.value).toBe(23)
        expect(data.get('1.0.optionCode')?.value).toBe(23)
    }))

    it('should return a valid pool header', fakeAsync(() => {
        spyOn(dhcpApi, 'updateSubnetBegin').and.returnValue(of(cannedResponseBeginSubnet6))
        component.subnetId = 234
        component.ngOnInit()
        tick()
        expect(component.getPoolHeader(0)).toEqual(['2001:db8:1::10-2001:db8:1::100', true])
        expect(component.getPoolHeader(1)).toEqual(['New Pool', false])
    }))

    it('should return a valid prefix pool header', fakeAsync(() => {
        spyOn(dhcpApi, 'updateSubnetBegin').and.returnValue(of(cannedResponseBeginSubnet6))
        component.subnetId = 234
        component.ngOnInit()
        tick()
        expect(component.getPrefixPoolHeader(0)).toEqual(['3000::/48', true])
        expect(component.getPrefixPoolHeader(1)).toEqual(['New Pool', false])
    }))

    it('should present the pool in accordion', fakeAsync(() => {
        spyOn(dhcpApi, 'updateSubnetBegin').and.returnValue(of(cannedResponseBeginSubnet4))
        component.subnetId = 123
        component.ngOnInit()
        tick()
        fixture.detectChanges()
        tick()

        const poolsPanel = fixture.debugElement.query(By.css('[legend="Pools"]'))
        expect(poolsPanel).toBeTruthy()

        const poolPanel = poolsPanel.query(By.css('p-accordion'))
        expect(poolsPanel).toBeTruthy()
        expect(poolPanel.nativeElement.innerText).toContain('192.0.2.10-192.0.2.100')
    }))

    it('should present the prefix pool in accordion', fakeAsync(() => {
        spyOn(dhcpApi, 'updateSubnetBegin').and.returnValue(of(cannedResponseBeginSubnet6))
        component.subnetId = 234
        component.ngOnInit()
        tick()
        fixture.detectChanges()
        tick()

        const poolsPanel = fixture.debugElement.query(By.css('[legend="Prefix Delegation Pools"]'))
        expect(poolsPanel).toBeTruthy()

        const poolPanel = poolsPanel.query(By.css('p-accordion'))
        expect(poolsPanel).toBeTruthy()
        expect(poolPanel.nativeElement.innerText).toContain('3000::/48')
    }))

    it('should return correct server tag severity', () => {
        expect(component.getServerTagSeverity(0)).toBe('success')
        expect(component.getServerTagSeverity(1)).toBe('warning')
        expect(component.getServerTagSeverity(2)).toBe('danger')
        expect(component.getServerTagSeverity(3)).toBe('info')
        expect(component.getServerTagSeverity(4)).toBe('info')
    })

    it('should remove the form for the unselected server', fakeAsync(() => {
        spyOn(dhcpApi, 'updateSubnetBegin').and.returnValue(of(cannedResponseBeginSubnet4))
        component.subnetId = 123
        component.ngOnInit()
        tick()
        fixture.detectChanges()

        // Expand the tab.
        const tab = fixture.debugElement.query(By.css('p-accordionTab'))
        expect(tab).toBeTruthy()
        const link = tab.query(By.css('a'))
        expect(link).toBeTruthy()
        link.nativeElement.click()
        fixture.detectChanges()

        expect(component.addressPoolComponents.length).toBe(1)
        spyOn(component.addressPoolComponents.get(0), 'handleDaemonsChange').and.callThrough()

        component.state.group.get('selectedDaemons').setValue([2])
        component.onDaemonsChange({
            itemValue: 1,
        })
        tick()
        fixture.detectChanges()

        expect(component.addressPoolComponents.get(0).handleDaemonsChange).toHaveBeenCalledOnceWith(1)
        expect(component.addressPoolComponents.get(0).selectableDaemons.length).toBe(1)
        expect(component.addressPoolComponents.get(0).selectableDaemons[0].id).toBe(2)

        const options = component.state.group.get('options.data') as UntypedFormArray
        expect(options).toBeTruthy()
        expect(options.length).toBe(1)
        expect(options.get('0.0.optionFields.0.control')?.value).toBe('192.0.2.2')

        const parameters = component.state.group.get('parameters') as FormGroup<KeaSubnetParametersForm>
        expect(parameters.get('allocator.unlocked')?.value).toBeFalse()
        expect((parameters.get('allocator.values') as UntypedFormArray).length).toBe(1)
        expect(parameters.get('allocator.values.0')?.value).toBe('iterative')

        expect(component.state.servers.length).toBe(1)
        expect(component.state.servers[0]).toBe('second/dhcp4')
    }))

    it('should create the form for the selected server', fakeAsync(() => {
        spyOn(dhcpApi, 'updateSubnetBegin').and.returnValue(of(cannedResponseBeginSubnet6))
        component.subnetId = 234
        component.ngOnInit()
        tick()
        fixture.detectChanges()

        // Expand the tab.
        const tab = fixture.debugElement.query(By.css('p-accordionTab'))
        expect(tab).toBeTruthy()
        const link = tab.query(By.css('a'))
        expect(link).toBeTruthy()
        link.nativeElement.click()
        tick()
        fixture.detectChanges()

        expect(component.addressPoolComponents.length).toBe(1)
        spyOn(component.addressPoolComponents.get(0), 'handleDaemonsChange').and.callThrough()

        component.state.group.get('selectedDaemons').setValue([3, 4, 5])
        component.onDaemonsChange({
            itemValue: 5,
        })
        tick()
        fixture.detectChanges()

        expect(component.addressPoolComponents.get(0).handleDaemonsChange).toHaveBeenCalledOnceWith(5)
        expect(component.addressPoolComponents.get(0).selectableDaemons.length).toBe(3)

        const options = component.state.group.get('options.data') as UntypedFormArray
        expect(options.length).toBe(3)
        expect(options.get('0.0.optionFields.0.control')?.value).toBe('2001:db8:2::6789')
        expect(options.get('1.0.optionFields.0.control')?.value).toBe('2001:db8:2::6789')
        expect(options.get('2.0.optionFields.0.control')?.value).toBe('2001:db8:2::6789')

        const parameters = component.state.group.get('parameters') as FormGroup<KeaSubnetParametersForm>
        expect(parameters.get('pdAllocator.unlocked')?.value).toBeTrue()
        expect((parameters.get('pdAllocator.values') as UntypedFormArray).length).toBe(3)
        expect(parameters.get('pdAllocator.values.0')?.value).toBeFalsy()
        expect(parameters.get('pdAllocator.values.1')?.value).toBe('iterative')
        expect(parameters.get('pdAllocator.values.2')?.value).toBeFalsy()
    }))

    it('should revert the changes in the form', fakeAsync(() => {
        spyOn(dhcpApi, 'updateSubnetBegin').and.returnValue(of(cannedResponseBeginSubnet4))
        component.subnetId = 123
        component.ngOnInit()
        tick()
        fixture.detectChanges()

        component.state.group.get('selectedDaemons').setValue([2])
        component.onDaemonsChange({
            itemValue: 1,
        })
        tick()
        fixture.detectChanges()

        let options = component.state.group.get('options.data') as UntypedFormArray
        options.get('0.0.optionFields.0.control')?.setValue('192.0.2.3')

        let parameters = component.state.group.get('parameters') as FormGroup<KeaSubnetParametersForm>
        parameters.get('allocator.values.0')?.setValue('flq')

        component.onRevert()

        options = component.state.group.get('options.data') as UntypedFormArray
        expect(options.length).toBe(2)
        expect(options.get('0.0.optionFields.0.control')?.value).toBe('192.0.2.1')
        expect(options.get('1.0.optionFields.0.control')?.value).toBe('192.0.2.2')

        parameters = component.state.group.get('parameters') as FormGroup<KeaSubnetParametersForm>
        expect(parameters.get('allocator.unlocked')?.value).toBeTrue()
        expect((parameters.get('allocator.values') as UntypedFormArray).length).toBe(2)
        expect(parameters.get('allocator.values.0')?.value).toBe('random')
        expect(parameters.get('allocator.values.1')?.value).toBe('iterative')
    }))

    it('should add and remove the pool', fakeAsync(() => {
        spyOn(dhcpApi, 'updateSubnetBegin').and.returnValue(of(cannedResponseBeginSubnet6))
        component.subnetId = 234
        component.ngOnInit()
        tick()
        fixture.detectChanges()

        component.onAddressPoolAdd()
        tick()
        fixture.detectChanges()

        const poolsPanel = fixture.debugElement.query(By.css('[legend="Pools"]'))
        expect(poolsPanel).toBeTruthy()

        // Expand the tab.
        const tabs = poolsPanel.queryAll(By.css('p-accordionTab'))
        expect(tabs.length).toBe(2)
        const link = tabs[1].query(By.css('a'))
        expect(link).toBeTruthy()
        link.nativeElement.click()
        tick()
        fixture.detectChanges()

        expect(tabs[1].nativeElement.innerText).toContain('New Pool')
        const poolDeleteBtn = tabs[1].query(By.css('[label="Delete Pool"]'))
        expect(poolDeleteBtn).toBeTruthy()

        let pools = component.state.group.get('pools') as FormArray<FormGroup<AddressPoolForm>>
        expect(pools).toBeTruthy()
        expect(pools.length).toBe(2)

        spyOn(messageService, 'add').and.callThrough()
        component.onAddressPoolDelete(1)
        tick()
        fixture.detectChanges()
        pools = component.state.group.get('pools') as FormArray<FormGroup<AddressPoolForm>>
        expect(pools).toBeTruthy()
        expect(pools.length).toBe(1)
        expect(messageService.add).toHaveBeenCalled()
    }))

    it('should add and remove the prefix pool', fakeAsync(() => {
        spyOn(dhcpApi, 'updateSubnetBegin').and.returnValue(of(cannedResponseBeginSubnet6))
        component.subnetId = 234
        component.ngOnInit()
        tick()
        fixture.detectChanges()

        component.onPrefixPoolAdd()
        tick()
        fixture.detectChanges()

        const poolsPanel = fixture.debugElement.query(By.css('[legend="Prefix Delegation Pools"]'))
        expect(poolsPanel).toBeTruthy()

        // Expand the tab.
        const tabs = poolsPanel.queryAll(By.css('p-accordionTab'))
        expect(tabs.length).toBe(2)
        const link = tabs[1].query(By.css('a'))
        expect(link).toBeTruthy()
        link.nativeElement.click()
        tick()
        fixture.detectChanges()

        expect(tabs[1].nativeElement.innerText).toContain('New Pool')
        const poolDeleteBtn = tabs[1].query(By.css('[label="Delete Pool"]'))
        expect(poolDeleteBtn).toBeTruthy()

        let pools = component.state.group.get('prefixPools') as FormArray<FormGroup<PrefixPoolForm>>
        expect(pools).toBeTruthy()
        expect(pools.length).toBe(2)

        spyOn(messageService, 'add').and.callThrough()
        component.onPrefixPoolDelete(1)
        tick()
        fixture.detectChanges()

        pools = component.state.group.get('prefixPools') as FormArray<FormGroup<PrefixPoolForm>>
        expect(pools).toBeTruthy()
        expect(pools.length).toBe(1)
        expect(messageService.add).toHaveBeenCalled()
    }))

    it('should emit cancel event', () => {
        spyOn(component.formCancel, 'emit')
        component.onCancel()
        expect(component.formCancel.emit).toHaveBeenCalled()
    })

    it('should present an error message when begin transaction fails', fakeAsync(() => {
        spyOn(dhcpApi, 'updateSubnetBegin').and.returnValues(
            throwError({ status: 404 }),
            of(cannedResponseBeginSubnet4)
        )
        component.subnetId = 123
        component.ngOnInit()
        tick()
        fixture.detectChanges()

        expect(component.state.initError).toEqual('status: 404')

        const messagesElement = fixture.debugElement.query(By.css('p-messages'))
        expect(messagesElement).toBeTruthy()
        expect(messagesElement.nativeElement.outerText).toContain(component.state.initError)

        const retryButton = fixture.debugElement.query(By.css('[label="Retry"]'))
        expect(retryButton).toBeTruthy()
        expect(retryButton.nativeElement.outerText).toBe('Retry')

        component.onRetry()
        tick()
        fixture.detectChanges()
        tick()

        expect(fixture.debugElement.query(By.css('p-messages'))).toBeFalsy()
        expect(fixture.debugElement.query(By.css('[label="Retry"]'))).toBeFalsy()
        expect(fixture.debugElement.query(By.css('[label="Submit"]'))).toBeTruthy()
    }))
})
