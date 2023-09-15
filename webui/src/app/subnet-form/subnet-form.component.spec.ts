import { ComponentFixture, TestBed, fakeAsync, tick } from '@angular/core/testing'

import { SubnetFormComponent } from './subnet-form.component'
import { ButtonModule } from 'primeng/button'
import { CheckboxModule } from 'primeng/checkbox'
import { ChipsModule } from 'primeng/chips'
import { DividerModule } from 'primeng/divider'
import { DropdownModule } from 'primeng/dropdown'
import { FieldsetModule } from 'primeng/fieldset'
import { FormGroup, FormsModule, ReactiveFormsModule, UntypedFormArray } from '@angular/forms'
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
import { KeaSubnetParametersForm } from '../forms/subnet-set-form.service'
import { SharedParametersFormComponent } from '../shared-parameters-form/shared-parameters-form.component'
import { DhcpOptionSetFormComponent } from '../dhcp-option-set-form/dhcp-option-set-form.component'
import { DhcpOptionFormComponent } from '../dhcp-option-form/dhcp-option-form.component'
import { DhcpClientClassSetFormComponent } from '../dhcp-client-class-set-form/dhcp-client-class-set-form.component'
import { EntityLinkComponent } from '../entity-link/entity-link.component'
import { HelpTipComponent } from '../help-tip/help-tip.component'
import { By } from '@angular/platform-browser'
import { MessagesModule } from 'primeng/messages'

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
                DhcpClientClassSetFormComponent,
                DhcpOptionFormComponent,
                DhcpOptionSetFormComponent,
                EntityLinkComponent,
                HelpTipComponent,
                SharedParametersFormComponent,
                SubnetFormComponent,
            ],
            providers: [MessageService],
        }).compileComponents()

        fixture = TestBed.createComponent(SubnetFormComponent)
        component = fixture.componentInstance
        dhcpApi = fixture.debugElement.injector.get(DHCPService)
        messageService = fixture.debugElement.injector.get(MessageService)
        fixture.detectChanges()
    })

    it('should create', () => {
        expect(component).toBeTruthy()
    })

    it('should open a form for updating IPv4 subnet', fakeAsync(() => {
        spyOn(dhcpApi, 'updateSubnetBegin').and.returnValue(of(cannedResponseBeginSubnet4))
        component.subnetId = 123
        component.ngOnInit()
        tick()
        expect(component.form).toBeTruthy()
        expect(component.form.preserved).toBeFalse()
        expect(component.form.transactionId).toBe(123)
        expect(component.form.group).toBeTruthy()
        expect(component.form.allDaemons.length).toBe(5)
        expect(component.form.filteredDaemons.length).toBe(2)
        expect(component.form.dhcpv4).toBeTrue()
        expect(component.form.dhcpv6).toBeFalse()

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
                                optionsHash: '',
                            },
                        },
                    ],
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
                                optionsHash: '',
                            },
                        },
                    ],
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
                },
            ],
        }

        expect(dhcpApi.updateSubnetSubmit).toHaveBeenCalledWith(
            component.subnetId,
            component.form.transactionId,
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
        expect(component.form).toBeTruthy()
        expect(component.form.preserved).toBeFalse()
        expect(component.form.transactionId).toBe(345)
        expect(component.form.group).toBeTruthy()
        expect(component.form.allDaemons.length).toBe(5)
        expect(component.form.filteredDaemons.length).toBe(3)
        expect(component.form.dhcpv4).toBeFalse()
        expect(component.form.dhcpv6).toBeTrue()

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
                                optionsHash: '',
                            },
                        },
                    ],
                    prefixDelegationPools: [
                        {
                            prefix: '3000::/48',
                            delegatedLength: 64,
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
                                optionsHash: '',
                            },
                        },
                    ],
                    prefixDelegationPools: [
                        {
                            prefix: '3000::/48',
                            delegatedLength: 64,
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
                        },
                    },
                },
            ],
        }

        expect(dhcpApi.updateSubnetSubmit).toHaveBeenCalledWith(
            component.subnetId,
            component.form.transactionId,
            subnet
        )
        expect(component.formSubmit.emit).toHaveBeenCalled()
        expect(messageService.add).toHaveBeenCalled()
    }))

    it('should initialize the form controls for a subnet', fakeAsync(() => {
        spyOn(dhcpApi, 'updateSubnetBegin').and.returnValue(of(cannedResponseBeginSubnet4))
        component.subnetId = 123
        component.ngOnInit()
        tick()
        // We cannot use contains() function here because it returns false for
        // disabled controls.
        expect(component.form?.group?.get('subnet')).toBeTruthy()
        expect(component.form?.group?.contains('parameters')).toBeTrue()
        expect(component.form?.group?.contains('options')).toBeTrue()

        expect(component.form?.group?.get('subnet').value).toBe('192.0.2.0/24')

        const parameters = component.form?.group?.get('parameters') as FormGroup<KeaSubnetParametersForm>
        expect(parameters.get('allocator.unlocked')?.value).toBeTrue()
        expect(parameters.get('allocator.values')?.value).toEqual(['random', 'iterative'])

        const options = component.form?.group?.get('options')
        expect(options.get('unlocked')?.value).toBeTrue()
        const data = options.get('data') as UntypedFormArray
        expect(data?.length).toBe(2)
        expect(data.get('0.0.optionCode')?.value).toBe(5)
        expect(data.get('1.0.optionCode')?.value).toBe(5)
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

        component.form.group.get('selectedDaemons').setValue([2])
        component.onDaemonsChange({
            itemValue: 1,
        })
        fixture.detectChanges()

        const options = component.form.group.get('options.data') as UntypedFormArray
        expect(options.length).toBe(1)
        expect(options.get('0.0.optionFields.0.control')?.value).toBe('192.0.2.2')

        const parameters = component.form.group.get('parameters') as FormGroup<KeaSubnetParametersForm>
        expect(parameters.get('allocator.unlocked')?.value).toBeFalse()
        expect((parameters.get('allocator.values') as UntypedFormArray).length).toBe(1)
        expect(parameters.get('allocator.values.0')?.value).toBe('iterative')

        expect(component.servers.length).toBe(1)
        expect(component.servers[0]).toBe('second/dhcp4')
    }))

    it('should create the form for the selected server', fakeAsync(() => {
        spyOn(dhcpApi, 'updateSubnetBegin').and.returnValue(of(cannedResponseBeginSubnet6))
        component.subnetId = 234
        component.ngOnInit()
        tick()

        component.form.group.get('selectedDaemons').setValue([3, 4, 5])
        component.onDaemonsChange({
            itemValue: 5,
        })
        fixture.detectChanges()

        const options = component.form.group.get('options.data') as UntypedFormArray
        expect(options.length).toBe(3)
        expect(options.get('0.0.optionFields.0.control')?.value).toBe('2001:db8:2::6789')
        expect(options.get('1.0.optionFields.0.control')?.value).toBe('2001:db8:2::6789')
        expect(options.get('2.0.optionFields.0.control')?.value).toBe('2001:db8:2::6789')

        const parameters = component.form.group.get('parameters') as FormGroup<KeaSubnetParametersForm>
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

        component.form.group.get('selectedDaemons').setValue([2])
        component.onDaemonsChange({
            itemValue: 1,
        })
        fixture.detectChanges()

        let options = component.form.group.get('options.data') as UntypedFormArray
        options.get('0.0.optionFields.0.control')?.setValue('192.0.2.3')

        let parameters = component.form.group.get('parameters') as FormGroup<KeaSubnetParametersForm>
        parameters.get('allocator.values.0')?.setValue('flq')

        component.onRevert()

        options = component.form.group.get('options.data') as UntypedFormArray
        expect(options.length).toBe(2)
        expect(options.get('0.0.optionFields.0.control')?.value).toBe('192.0.2.1')
        expect(options.get('1.0.optionFields.0.control')?.value).toBe('192.0.2.2')

        parameters = component.form.group.get('parameters') as FormGroup<KeaSubnetParametersForm>
        expect(parameters.get('allocator.unlocked')?.value).toBeTrue()
        expect((parameters.get('allocator.values') as UntypedFormArray).length).toBe(2)
        expect(parameters.get('allocator.values.0')?.value).toBe('random')
        expect(parameters.get('allocator.values.1')?.value).toBe('iterative')
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

        expect(component.form.initError).toEqual('status: 404')

        const messagesElement = fixture.debugElement.query(By.css('p-messages'))
        expect(messagesElement).toBeTruthy()
        expect(messagesElement.nativeElement.outerText).toContain(component.form.initError)

        const retryButton = fixture.debugElement.query(By.css('[label="Retry"]'))
        expect(retryButton).toBeTruthy()
        expect(retryButton.nativeElement.outerText).toBe('Retry')

        component.onRetry()
        tick()
        fixture.detectChanges()

        expect(fixture.debugElement.query(By.css('p-messages'))).toBeFalsy()
        expect(fixture.debugElement.query(By.css('[label="Retry"]'))).toBeFalsy()
        expect(fixture.debugElement.query(By.css('[label="Submit"]'))).toBeTruthy()
    }))
})
