import { ComponentFixture, TestBed } from '@angular/core/testing'

import { SharedNetworkFormComponent } from './shared-network-form.component'
import { MessagesModule } from 'primeng/messages'
import { MessageService } from 'primeng/api'
import { ProgressSpinnerModule } from 'primeng/progressspinner'
import { of, throwError } from 'rxjs'
import { DHCPService } from '../backend'
import { IPType } from '../iptype'
import { FormGroup, FormsModule, ReactiveFormsModule, UntypedFormArray } from '@angular/forms'
import { ButtonModule } from 'primeng/button'
import { CheckboxModule } from 'primeng/checkbox'
import { ChipsModule } from 'primeng/chips'
import { DividerModule } from 'primeng/divider'
import { DropdownModule } from 'primeng/dropdown'
import { FieldsetModule } from 'primeng/fieldset'
import { HttpClientModule } from '@angular/common/http'
import { InputNumberModule } from 'primeng/inputnumber'
import { MultiSelectModule } from 'primeng/multiselect'
import { TableModule } from 'primeng/table'
import { TagModule } from 'primeng/tag'
import { TriStateCheckboxModule } from 'primeng/tristatecheckbox'
import { OverlayPanelModule } from 'primeng/overlaypanel'
import { RouterTestingModule } from '@angular/router/testing'
import { SplitButtonModule } from 'primeng/splitbutton'
import { ToastModule } from 'primeng/toast'
import { ArrayValueSetFormComponent } from '../array-value-set-form/array-value-set-form.component'
import { DhcpClientClassSetFormComponent } from '../dhcp-client-class-set-form/dhcp-client-class-set-form.component'
import { DhcpOptionFormComponent } from '../dhcp-option-form/dhcp-option-form.component'
import { DhcpOptionSetFormComponent } from '../dhcp-option-set-form/dhcp-option-set-form.component'
import { EntityLinkComponent } from '../entity-link/entity-link.component'
import { HelpTipComponent } from '../help-tip/help-tip.component'
import { SharedParametersFormComponent } from '../shared-parameters-form/shared-parameters-form.component'
import { NoopAnimationsModule } from '@angular/platform-browser/animations'
import { KeaSubnetParametersForm } from '../forms/subnet-set-form.service'
import { By } from '@angular/platform-browser'
import { SharedNetworkFormState } from '../forms/shared-network-form'

describe('SharedNetworkFormComponent', () => {
    let component: SharedNetworkFormComponent
    let fixture: ComponentFixture<SharedNetworkFormComponent>
    let dhcpApi: DHCPService
    let messageService: MessageService

    let cannedResponseBeginSharedNetwork4: any = {
        id: 123,
        sharedNetwork: {
            id: 123,
            name: 'stanza',
            universe: 4,
            localSharedNetworks: [
                {
                    appId: 234,
                    daemonId: 1,
                    appName: 'server 1',
                    keaConfigSharedNetworkParameters: {
                        sharedNetworkLevelParameters: {
                            allocator: 'random',
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
                    appId: 345,
                    daemonId: 2,
                    appName: 'server 2',
                    keaConfigSharedNetworkParameters: {
                        sharedNetworkLevelParameters: {
                            allocator: 'iterative',
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
            subnets: [
                {
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
                        },
                        {
                            id: 234,
                            appId: 345,
                            daemonId: 2,
                            appName: 'server 2',
                        },
                    ],
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
            {
                id: 6,
                name: 'dhcp4',
                app: {
                    name: 'fifth',
                },
            },
        ],
        sharedNetworks4: ['floor1', 'floor2', 'floor3', 'stanza'],
        sharedNetworks6: [],
        clientClasses: ['foo', 'bar'],
    }

    let cannedResponseBeginSharedNetwork6: any = {
        id: 234,
        sharedNetwork: {
            id: 234,
            name: 'bella',
            universe: 6,
            localSharedNetworks: [
                {
                    appId: 234,
                    daemonId: 4,
                    appName: 'server 1',
                    keaConfigSharedNetworkParameters: {
                        sharedNetworkLevelParameters: {
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
                    appId: 345,
                    daemonId: 5,
                    appName: 'server 2',
                    keaConfigSharedNetworkParameters: {
                        sharedNetworkLevelParameters: {
                            allocator: 'random',
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
        sharedNetworks6: ['floor1', 'floor2', 'floor3', 'bella'],
        clientClasses: ['foo', 'bar'],
    }

    beforeEach(async () => {
        await TestBed.configureTestingModule({
            imports: [
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
                ArrayValueSetFormComponent,
                DhcpClientClassSetFormComponent,
                DhcpOptionFormComponent,
                DhcpOptionSetFormComponent,
                EntityLinkComponent,
                HelpTipComponent,
                SharedNetworkFormComponent,
                SharedParametersFormComponent,
            ],
            providers: [DHCPService, MessageService],
        }).compileComponents()

        fixture = TestBed.createComponent(SharedNetworkFormComponent)
        component = fixture.componentInstance
        dhcpApi = fixture.debugElement.injector.get(DHCPService)
        messageService = fixture.debugElement.injector.get(MessageService)
        fixture.detectChanges()
    })

    it('should create', () => {
        expect(component).toBeTruthy()
    })

    it('should open a form for creating an IPv4 shared network', async () => {
        spyOn(dhcpApi, 'createSharedNetworkBegin').and.returnValue(of(cannedResponseBeginSharedNetwork4))
        component.ngOnInit()
        await fixture.whenStable()
        fixture.detectChanges()

        expect(component.state).toBeTruthy()
        expect(component.state.loaded).toBeTrue()
        expect(component.state.transactionId).toBe(123)
        expect(component.state.ipType).toBe(IPType.IPv4)
        expect(component.state.group).toBeTruthy()
        expect(component.state.filteredDaemons.length).toBe(3)

        component.state.group.get('name').setValue('stanza')

        const selectedDaemons = [1, 2]
        component.state.group.get('selectedDaemons').setValue(selectedDaemons)
        selectedDaemons.forEach((id) => {
            component.onDaemonsChange({
                itemValue: id,
            })
        })
        await fixture.whenStable()
        fixture.detectChanges()

        const okResp: any = {
            status: 200,
        }
        spyOn(dhcpApi, 'createSharedNetworkSubmit').and.returnValue(of(okResp))
        spyOn(component.formSubmit, 'emit')
        spyOn(messageService, 'add')
        component.onSubmit()
        await fixture.whenStable()
        fixture.detectChanges()

        const sharedNetwork = {
            name: 'stanza',
            universe: 4,
            subnets: [],
            localSharedNetworks: [
                {
                    daemonId: 1,
                    keaConfigSharedNetworkParameters: {
                        sharedNetworkLevelParameters: {
                            options: [],
                        },
                    },
                },
                {
                    daemonId: 2,
                    keaConfigSharedNetworkParameters: {
                        sharedNetworkLevelParameters: {
                            options: [],
                        },
                    },
                },
            ],
        }

        expect(dhcpApi.createSharedNetworkSubmit).toHaveBeenCalledWith(component.state.transactionId, sharedNetwork)
        expect(component.formSubmit.emit).toHaveBeenCalled()
        expect(messageService.add).toHaveBeenCalled()
    })

    it('should open a form for creating an IPv6 shared network', async () => {
        spyOn(dhcpApi, 'createSharedNetworkBegin').and.returnValue(of(cannedResponseBeginSharedNetwork6))
        component.ngOnInit()
        await fixture.whenStable()
        fixture.detectChanges()

        expect(component.state).toBeTruthy()
        expect(component.state.loaded).toBeTrue()
        expect(component.state.transactionId).toBe(234)
        expect(component.state.ipType).toBe(IPType.IPv4)
        expect(component.state.group).toBeTruthy()
        expect(component.state.filteredDaemons.length).toBe(3)

        component.state.group.get('name').setValue('hola')

        const selectedDaemons = [4, 5]
        component.state.group.get('selectedDaemons').setValue(selectedDaemons)
        selectedDaemons.forEach((id) => {
            component.onDaemonsChange({
                itemValue: id,
            })
        })
        await fixture.whenStable()
        fixture.detectChanges()

        const okResp: any = {
            status: 200,
        }
        spyOn(dhcpApi, 'createSharedNetworkSubmit').and.returnValue(of(okResp))
        spyOn(component.formSubmit, 'emit')
        spyOn(messageService, 'add')
        component.onSubmit()
        await fixture.whenStable()
        fixture.detectChanges()

        const sharedNetwork = {
            name: 'hola',
            universe: 6,
            subnets: [],
            localSharedNetworks: [
                {
                    daemonId: 4,
                    keaConfigSharedNetworkParameters: {
                        sharedNetworkLevelParameters: {
                            options: [],
                        },
                    },
                },
                {
                    daemonId: 5,
                    keaConfigSharedNetworkParameters: {
                        sharedNetworkLevelParameters: {
                            options: [],
                        },
                    },
                },
            ],
        }

        expect(dhcpApi.createSharedNetworkSubmit).toHaveBeenCalledWith(component.state.transactionId, sharedNetwork)
        expect(component.formSubmit.emit).toHaveBeenCalled()
        expect(messageService.add).toHaveBeenCalled()
    })

    it('should open a form for updating IPv4 shared network', async () => {
        spyOn(dhcpApi, 'updateSharedNetworkBegin').and.returnValue(of(cannedResponseBeginSharedNetwork4))
        component.sharedNetworkId = 123
        component.ngOnInit()
        await fixture.whenStable()
        fixture.detectChanges()

        expect(component.state).toBeTruthy()
        expect(component.state.loaded).toBeTrue()
        expect(component.state.transactionId).toBe(123)
        expect(component.state.ipType).toBe(IPType.IPv4)
        expect(component.state.group).toBeTruthy()
        expect(component.state.filteredDaemons.length).toBe(3)

        component.state.group.get('selectedDaemons').setValue([1, 6])
        component.state.updateFormForSelectedDaemons([1, 6])
        component.state.updateServers([1, 6])

        const okResp: any = {
            status: 200,
        }
        spyOn(dhcpApi, 'updateSharedNetworkSubmit').and.returnValue(of(okResp))
        spyOn(component.formSubmit, 'emit')
        spyOn(messageService, 'add')
        component.onSubmit()
        await fixture.whenStable()
        fixture.detectChanges()

        const sharedNetwork = {
            id: 123,
            name: 'stanza',
            universe: 4,
            subnets: [
                {
                    id: 123,
                    subnet: '192.0.2.0/24',
                    sharedNetwork: 'floor3',
                    sharedNetworkId: 3,
                    localSubnets: [
                        {
                            id: 123,
                            daemonId: 1,
                        },
                        {
                            id: 123,
                            daemonId: 6,
                        },
                    ],
                },
            ],
            localSharedNetworks: [
                {
                    daemonId: 1,
                    keaConfigSharedNetworkParameters: {
                        sharedNetworkLevelParameters: {
                            allocator: 'random',
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
                },
                {
                    daemonId: 6,
                    keaConfigSharedNetworkParameters: {
                        sharedNetworkLevelParameters: {
                            allocator: 'iterative',
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
                },
            ],
        }

        expect(dhcpApi.updateSharedNetworkSubmit).toHaveBeenCalledWith(
            component.sharedNetworkId,
            component.state.transactionId,
            sharedNetwork
        )
        expect(component.formSubmit.emit).toHaveBeenCalled()
        expect(messageService.add).toHaveBeenCalled()
    })

    it('should open a form for updating IPv6 shared network', async () => {
        spyOn(dhcpApi, 'updateSharedNetworkBegin').and.returnValue(of(cannedResponseBeginSharedNetwork6))
        component.sharedNetworkId = 234
        component.ngOnInit()
        await fixture.whenStable()
        fixture.detectChanges()

        expect(component.state).toBeTruthy()
        expect(component.state.loaded).toBeTrue()
        expect(component.state.transactionId).toBe(234)
        expect(component.state.ipType).toBe(IPType.IPv6)
        expect(component.state.group).toBeTruthy()
        expect(component.state.filteredDaemons.length).toBe(3)

        const okResp: any = {
            status: 200,
        }
        spyOn(dhcpApi, 'updateSharedNetworkSubmit').and.returnValue(of(okResp))
        spyOn(component.formSubmit, 'emit')
        spyOn(messageService, 'add')
        component.onSubmit()
        await fixture.whenStable()
        fixture.detectChanges()

        const sharedNetwork = {
            id: 234,
            name: 'bella',
            universe: 6,
            subnets: [],
            localSharedNetworks: [
                {
                    daemonId: 4,
                    keaConfigSharedNetworkParameters: {
                        sharedNetworkLevelParameters: {
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
                    daemonId: 5,
                    keaConfigSharedNetworkParameters: {
                        sharedNetworkLevelParameters: {
                            allocator: 'random',
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

        expect(dhcpApi.updateSharedNetworkSubmit).toHaveBeenCalledWith(
            component.sharedNetworkId,
            component.state.transactionId,
            sharedNetwork
        )
        expect(component.formSubmit.emit).toHaveBeenCalled()
        expect(messageService.add).toHaveBeenCalled()
    })

    it('should initialize the form controls for an IPv4 shared network', async () => {
        spyOn(dhcpApi, 'updateSharedNetworkBegin').and.returnValue(of(cannedResponseBeginSharedNetwork4))
        component.sharedNetworkId = 123
        component.ngOnInit()
        await fixture.whenStable()
        fixture.detectChanges()
        // We cannot use contains() function here because it returns false for
        // disabled controls.
        expect(component.state).toBeTruthy()
        expect(component.state.group).toBeTruthy()
        expect(component.state.group.get('name')).toBeTruthy()
        expect(component.state.group.contains('parameters')).toBeTrue()
        expect(component.state.group.contains('options')).toBeTrue()

        expect(component.state.group.get('name').value).toBe('stanza')

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

        await fixture.whenStable()
    })

    it('should initialize the form controls for an IPv6 subnet', async () => {
        spyOn(dhcpApi, 'updateSharedNetworkBegin').and.returnValue(of(cannedResponseBeginSharedNetwork6))
        component.sharedNetworkId = 234
        component.ngOnInit()
        await fixture.whenStable()
        fixture.detectChanges()
        // We cannot use contains() function here because it returns false for
        // disabled controls.
        expect(component.state).toBeTruthy()
        expect(component.state.group).toBeTruthy()
        expect(component.state.group.get('name')).toBeTruthy()
        expect(component.state.group.contains('parameters')).toBeTrue()
        expect(component.state.group.contains('options')).toBeTrue()

        expect(component.state.group.get('name').value).toBe('bella')

        const parameters = component.state.group.get('parameters') as FormGroup<KeaSubnetParametersForm>
        expect(parameters).toBeTruthy()
        expect(parameters.get('allocator.unlocked')?.value).toBeFalse()
        expect(parameters.get('allocator.values')?.value).toEqual(['random', 'random'])

        const options = component.state.group.get('options')
        expect(options).toBeTruthy()
        expect(options.get('unlocked')?.value).toBeFalse()
        const data = options.get('data') as UntypedFormArray
        expect(data?.length).toBe(2)
        expect(data.get('0.0.optionCode')?.value).toBe(23)
        expect(data.get('1.0.optionCode')?.value).toBe(23)

        await fixture.whenStable()
    })

    it('should return correct server tag severity', () => {
        expect(component.getServerTagSeverity(0)).toBe('success')
        expect(component.getServerTagSeverity(1)).toBe('warning')
        expect(component.getServerTagSeverity(2)).toBe('danger')
        expect(component.getServerTagSeverity(3)).toBe('info')
        expect(component.getServerTagSeverity(4)).toBe('info')
    })

    it('should remove the form for the unselected server', async () => {
        spyOn(dhcpApi, 'updateSharedNetworkBegin').and.returnValue(of(cannedResponseBeginSharedNetwork4))
        component.sharedNetworkId = 123
        component.ngOnInit()
        await fixture.whenStable()
        fixture.detectChanges()

        component.state.group.get('selectedDaemons').setValue([2])
        component.onDaemonsChange({
            itemValue: 1,
        })
        await fixture.whenStable()
        fixture.detectChanges()

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
    })

    it('should create the form for the selected server', async () => {
        spyOn(dhcpApi, 'updateSharedNetworkBegin').and.returnValue(of(cannedResponseBeginSharedNetwork6))
        component.sharedNetworkId = 234
        component.ngOnInit()
        await fixture.whenStable()
        fixture.detectChanges()

        component.state.group.get('selectedDaemons').setValue([3, 4, 5])
        component.onDaemonsChange({
            itemValue: 5,
        })
        await fixture.whenStable()
        fixture.detectChanges()

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
    })

    it('should revert the changes in the form', async () => {
        spyOn(dhcpApi, 'updateSharedNetworkBegin').and.returnValue(of(cannedResponseBeginSharedNetwork4))
        component.sharedNetworkId = 123
        component.ngOnInit()
        await fixture.whenStable()
        fixture.detectChanges()

        component.state.group.get('selectedDaemons').setValue([2])
        component.onDaemonsChange({
            itemValue: 1,
        })
        await fixture.whenStable()
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
    })

    it('should emit cancel event', () => {
        spyOn(component.formCancel, 'emit')
        component.onCancel()
        expect(component.formCancel.emit).toHaveBeenCalled()
    })

    it('should present an error message when begin transaction fails', async () => {
        spyOn(dhcpApi, 'updateSharedNetworkBegin').and.returnValues(
            throwError(() => new Error('status: 404')),
            of(cannedResponseBeginSharedNetwork4)
        )
        spyOn(messageService, 'add')
        component.sharedNetworkId = 123
        component.state = new SharedNetworkFormState()
        component.ngOnInit()
        await fixture.whenStable()
        fixture.detectChanges()

        expect(messageService.add).toHaveBeenCalled()
        expect(component.state.initError.length).not.toBe(0)

        const messagesElement = fixture.debugElement.query(By.css('p-messages'))
        expect(messagesElement).toBeTruthy()
        expect(messagesElement.nativeElement.outerText).toContain(component.state.initError)

        const retryButton = fixture.debugElement.query(By.css('[label="Retry"]'))
        expect(retryButton).toBeTruthy()
        expect(retryButton.nativeElement.outerText).toBe('Retry')

        component.onRetry()
        await fixture.whenStable()
        fixture.detectChanges()

        expect(fixture.debugElement.query(By.css('p-messages'))).toBeFalsy()
        expect(fixture.debugElement.query(By.css('[label="Retry"]'))).toBeFalsy()
        expect(fixture.debugElement.query(By.css('[label="Submit"]'))).toBeTruthy()

        await fixture.whenStable()
    })
})
