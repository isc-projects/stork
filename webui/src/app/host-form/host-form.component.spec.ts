import { ComponentFixture, TestBed } from '@angular/core/testing'
import { FormBuilder, FormsModule, ReactiveFormsModule } from '@angular/forms'
import { HttpClientTestingModule } from '@angular/common/http/testing'
import { By } from '@angular/platform-browser'
import { NoopAnimationsModule } from '@angular/platform-browser/animations'
import { of, throwError } from 'rxjs'
import { MessageService } from 'primeng/api'
import { ButtonModule } from 'primeng/button'
import { CheckboxModule } from 'primeng/checkbox'
import { DropdownModule } from 'primeng/dropdown'
import { FieldsetModule } from 'primeng/fieldset'
import { MessagesModule } from 'primeng/messages'
import { MultiSelectModule } from 'primeng/multiselect'
import { HostFormComponent } from './host-form.component'
import { DHCPService } from '../backend'

import { fakeAsync, tick } from '@angular/core/testing'

describe('HostFormComponent', () => {
    let component: HostFormComponent
    let fixture: ComponentFixture<HostFormComponent>
    let dhcpApi: DHCPService
    let messageService: MessageService

    let cannedResponseBegin: any = {
        id: 123,
        subnets: [
            {
                id: 1,
                subnet: '192.0.2.0/24',
                localSubnets: [
                    {
                        daemonId: 1,
                    },
                    {
                        daemonId: 2,
                    },
                ],
            },
            {
                id: 2,
                subnet: '192.0.3.0/24',
                localSubnets: [
                    {
                        daemonId: 2,
                    },
                    {
                        daemonId: 3,
                    },
                ],
            },
            {
                id: 3,
                subnet: '2001:db8:1::/64',
                localSubnets: [
                    {
                        daemonId: 4,
                    },
                ],
            },
            {
                id: 4,
                subnet: '2001:db8:2::/64',
                localSubnets: [
                    {
                        daemonId: 5,
                    },
                ],
            },
        ],
        apps: [
            {
                id: 1,
                name: 'first',
                details: {
                    daemons: [
                        {
                            id: 1,
                            name: 'dhcp4',
                        },
                        {
                            id: 3,
                            name: 'dhcp6',
                        },
                    ],
                },
            },
            {
                id: 2,
                name: 'second',
                details: {
                    daemons: [
                        {
                            id: 2,
                            name: 'dhcp4',
                        },
                        {
                            id: 4,
                            name: 'dhcp6',
                        },
                    ],
                },
            },
            {
                id: 3,
                name: 'third',
                details: {
                    daemons: [
                        {
                            id: 5,
                            name: 'dhcp6',
                        },
                    ],
                },
            },
        ],
    }

    beforeEach(async () => {
        await TestBed.configureTestingModule({
            providers: [FormBuilder, DHCPService, MessageService],
            imports: [
                ButtonModule,
                CheckboxModule,
                DropdownModule,
                FieldsetModule,
                FormsModule,
                HttpClientTestingModule,
                MessagesModule,
                MultiSelectModule,
                NoopAnimationsModule,
                ReactiveFormsModule,
            ],
            declarations: [HostFormComponent],
        }).compileComponents()
    })

    beforeEach(() => {
        fixture = TestBed.createComponent(HostFormComponent)
        component = fixture.componentInstance
        dhcpApi = fixture.debugElement.injector.get(DHCPService)
        messageService = fixture.debugElement.injector.get(MessageService)
        fixture.detectChanges()
    })

    it('should create', () => {
        expect(component).toBeTruthy()
    })

    it('should begin new transaction', fakeAsync(() => {
        spyOn(dhcpApi, 'createHostBegin').and.returnValue(of(cannedResponseBegin))
        component.ngOnInit()
        tick()
        expect(component.form).toBeTruthy()
        expect(component.form.preserved).toBeFalse()
        expect(component.form.transactionId).toBe(123)
        expect(component.form.group).toBeTruthy()
        expect(component.form.allSubnets.length).toBe(4)
        expect(component.form.filteredSubnets.length).toBe(4)
        expect(component.form.allDaemons.length).toBe(5)
        expect(component.form.filteredDaemons.length).toBe(5)
        expect(component.form.dhcpv4).toBeFalse()
        expect(component.form.dhcpv6).toBeFalse()
    }))

    it('should enable specific controls for dhcpv4 servers', fakeAsync(() => {
        spyOn(dhcpApi, 'createHostBegin').and.returnValue(of(cannedResponseBegin))
        component.ngOnInit()
        tick()
        fixture.detectChanges()

        component.formGroup.get('selectedServers').setValue([1])
        component.onServersChange()
        expect(component.form.filteredDaemons.length).toBe(2)
        expect(component.form.filteredDaemons[0].id).toBe(1)
        expect(component.form.filteredDaemons[1].id).toBe(2)

        component.formGroup.get('selectedServers').setValue([2])
        component.onServersChange()
        expect(component.form.filteredDaemons.length).toBe(2)
        expect(component.form.filteredDaemons[0].id).toBe(1)
        expect(component.form.filteredDaemons[1].id).toBe(2)
    }))

    it('should enable specific controls for dhcpv6 servers', fakeAsync(() => {
        spyOn(dhcpApi, 'createHostBegin').and.returnValue(of(cannedResponseBegin))
        component.ngOnInit()
        tick()
        fixture.detectChanges()

        component.formGroup.get('selectedServers').setValue([3, 4, 5])
        component.onServersChange()
        expect(component.form.filteredDaemons.length).toBe(3)
        expect(component.form.filteredDaemons[0].id).toBe(3)
        expect(component.form.filteredDaemons[1].id).toBe(4)
        expect(component.form.filteredDaemons[2].id).toBe(5)
    }))

    it('should show filter overlapping ipv4 subnets for selected servers', fakeAsync(() => {
        spyOn(dhcpApi, 'createHostBegin').and.returnValue(of(cannedResponseBegin))
        component.ngOnInit()
        tick()
        fixture.detectChanges()
        expect(component.form.filteredSubnets.length).toBe(4)

        component.formGroup.get('selectedServers').setValue([2])
        component.onServersChange()
        expect(component.form.filteredSubnets.length).toBe(2)

        component.formGroup.get('selectedServers').setValue([1, 2])
        component.onServersChange()
        expect(component.form.filteredSubnets.length).toBe(1)
    }))

    it('should show filter overlapping ipv6 subnets for selected servers', fakeAsync(() => {
        spyOn(dhcpApi, 'createHostBegin').and.returnValue(of(cannedResponseBegin))
        component.ngOnInit()
        tick()
        fixture.detectChanges()
        expect(component.form.filteredSubnets.length).toBe(4)

        component.formGroup.get('selectedServers').setValue([4])
        component.onServersChange()
        expect(component.form.filteredSubnets.length).toBe(1)

        component.formGroup.get('selectedServers').setValue([4, 5])
        component.onServersChange()
        expect(component.form.filteredSubnets.length).toBe(0)
    }))

    it('should require server specification', fakeAsync(() => {
        spyOn(dhcpApi, 'createHostBegin').and.returnValue(of(cannedResponseBegin))
        component.ngOnInit()
        tick()
        fixture.detectChanges()

        component.formGroup.get('selectedServers').setValue([4])
        component.onServersChange()
        fixture.detectChanges()
        expect(component.formGroup.get('selectedServers').valid).toBeTrue()

        component.formGroup.get('selectedServers').setValue([])
        component.onServersChange()
        fixture.detectChanges()
        expect(component.formGroup.get('selectedServers').valid).toBeFalse()

        component.formGroup.get('selectedServers').markAsTouched()
        component.formGroup.get('selectedServers').markAsDirty()
        fixture.detectChanges()

        let errmsg = fixture.debugElement.query(By.css('small'))
        expect(errmsg).toBeTruthy()
        expect(errmsg.nativeElement.innerText).toContain('At least one server must be selected.')
    }))

    it('should require subnet specification', fakeAsync(() => {
        spyOn(dhcpApi, 'createHostBegin').and.returnValue(of(cannedResponseBegin))
        component.ngOnInit()
        tick()
        fixture.detectChanges()

        component.formGroup.get('selectedSubnet').setValue(1)
        fixture.detectChanges()
        expect(component.formGroup.get('selectedSubnet').valid).toBeTrue()

        component.formGroup.get('selectedSubnet').setValue(null)
        fixture.detectChanges()
        expect(component.formGroup.get('selectedSubnet').valid).toBeFalse()

        component.formGroup.get('selectedSubnet').markAsTouched()
        component.formGroup.get('selectedSubnet').markAsDirty()
        fixture.detectChanges()

        let errmsg = fixture.debugElement.query(By.css('small'))
        expect(errmsg).toBeTruthy()
        expect(errmsg.nativeElement.innerText).toContain('Subnet must be selected if the reservation is not global.')
    }))

    it('should disable subnet selection for global reservations', fakeAsync(() => {
        spyOn(dhcpApi, 'createHostBegin').and.returnValue(of(cannedResponseBegin))
        component.ngOnInit()
        tick()
        fixture.detectChanges()
        let subnetsDropdown = fixture.debugElement.query(By.css('[inputId="subnets-dropdown"]'))
        expect(subnetsDropdown).toBeTruthy()

        component.formGroup.get('globalReservation').setValue(true)
        fixture.detectChanges()

        subnetsDropdown = fixture.debugElement.query(By.css('[inputId="subnets-dropdown"]'))
        expect(subnetsDropdown).toBeFalsy()
    }))

    it('should list identifier types for a server type', fakeAsync(() => {
        spyOn(dhcpApi, 'createHostBegin').and.returnValue(of(cannedResponseBegin))
        component.ngOnInit()
        tick()
        fixture.detectChanges()

        component.formGroup.get('selectedServers').setValue([1])
        component.onServersChange()

        expect(component.hostIdTypes.length).toBe(5)
        expect(component.hostIdTypes[0].label).toBe('hw-address')
        expect(component.hostIdTypes[1].label).toBe('client-id')
        expect(component.hostIdTypes[2].label).toBe('circuit-id')
        expect(component.hostIdTypes[3].label).toBe('duid')
        expect(component.hostIdTypes[4].label).toBe('flex-id')

        component.formGroup.get('selectedServers').setValue([3])
        component.onServersChange()

        expect(component.hostIdTypes.length).toBe(3)
        expect(component.hostIdTypes[0].label).toBe('hw-address')
        expect(component.hostIdTypes[1].label).toBe('duid')
        expect(component.hostIdTypes[2].label).toBe('flex-id')
    }))

    it('should validate hex identifier', fakeAsync(() => {
        spyOn(dhcpApi, 'createHostBegin').and.returnValue(of(cannedResponseBegin))
        component.ngOnInit()
        tick()
        fixture.detectChanges()

        expect(component.formGroup.get('hostIdGroup.idFormat').value).toBe('hex')

        component.formGroup.get('hostIdGroup.idInputHex').setValue('01:02:03')
        expect(component.formGroup.get('hostIdGroup.idInputHex').valid).toBeTrue()

        component.formGroup.get('hostIdGroup.idInputHex').setValue('invalid')
        expect(component.formGroup.get('hostIdGroup.idInputHex').valid).toBeFalse()
    }))

    it('should validate text identifier', fakeAsync(() => {
        spyOn(dhcpApi, 'createHostBegin').and.returnValue(of(cannedResponseBegin))
        component.ngOnInit()
        tick()
        fixture.detectChanges()

        component.formGroup.get('hostIdGroup.idFormat').setValue('text')
        fixture.detectChanges()
        expect(component.formGroup.get('hostIdGroup.idFormat').value).toBe('text')

        component.formGroup.get('hostIdGroup.idInputText').setValue('valid')
        expect(component.formGroup.get('hostIdGroup.idInputText').valid).toBeTrue()

        component.formGroup.get('hostIdGroup.idInputText').setValue('')
        expect(component.formGroup.get('hostIdGroup.idInputText').valid).toBeFalse()
    }))

    it('should list ip reservation types for a server type', fakeAsync(() => {
        spyOn(dhcpApi, 'createHostBegin').and.returnValue(of(cannedResponseBegin))
        component.ngOnInit()
        tick()
        fixture.detectChanges()

        expect(component.ipGroups.length).toBe(1)
        expect(component.ipGroups.at(0).get('ipType').value).toBe('ipv4')
        expect(component.ipTypes.length).toBe(1)
        expect(component.ipTypes[0].label).toBe('IPv4 address')

        component.formGroup.get('selectedServers').setValue([3])
        component.onServersChange()
        fixture.detectChanges()

        expect(component.ipGroups.length).toBe(1)
        expect(component.ipGroups.at(0).get('ipType').value).toBe('ia_na')
        expect(component.ipTypes.length).toBe(2)
        expect(component.ipTypes[0].label).toBe('IPv6 address')
        expect(component.ipTypes[1].label).toBe('IPv6 prefix')
    }))

    it('should clear selected ip reservations on server type change', fakeAsync(() => {
        spyOn(dhcpApi, 'createHostBegin').and.returnValue(of(cannedResponseBegin))
        component.ngOnInit()
        tick()
        fixture.detectChanges()

        component.formGroup.get('selectedServers').setValue([3])
        component.onServersChange()
        component.addIPInput()
        fixture.detectChanges()

        expect(component.ipGroups.length).toBe(2)
        component.ipGroups.at(0).get('ipType').setValue('ia_pd')
        component.ipGroups.at(0).get('inputPD').setValue('3001::')
        component.ipGroups.at(0).get('inputPDLength').setValue(96)
        component.ipGroups.at(1).get('ipType').setValue('ia_na')
        component.ipGroups.at(1).get('inputNA').setValue('2001:db8:1::1')

        component.formGroup.get('selectedServers').setValue([1])
        component.onServersChange()
        fixture.detectChanges()

        expect(component.ipGroups.length).toBe(1)
        expect(component.ipGroups.at(0).get('ipType').value).toBe('ipv4')
        expect(component.ipGroups.at(0).get('inputIPv4').value).toBe('')

        component.deleteIPInput(0)

        component.formGroup.get('selectedServers').setValue([3])
        component.onServersChange()
        fixture.detectChanges()

        expect(component.ipGroups.length).toBe(0)
    }))

    it('should validate ipv4 reservation', fakeAsync(() => {
        spyOn(dhcpApi, 'createHostBegin').and.returnValue(of(cannedResponseBegin))
        component.ngOnInit()
        tick()
        fixture.detectChanges()

        expect(component.ipGroups.length).toBe(1)
        component.ipGroups.at(0).get('inputIPv4').setValue('invalid')
        fixture.detectChanges()
        expect(component.ipGroups.at(0).get('inputIPv4').valid).toBeFalse()
    }))

    it('should validate ipv6 address reservation', fakeAsync(() => {
        spyOn(dhcpApi, 'createHostBegin').and.returnValue(of(cannedResponseBegin))
        component.ngOnInit()
        tick()
        fixture.detectChanges()

        component.ipGroups.at(0).get('ipType').setValue('ia_na')
        fixture.detectChanges()

        expect(component.ipGroups.length).toBe(1)
        component.ipGroups.at(0).get('inputNA').setValue('invalid')
        fixture.detectChanges()
        expect(component.ipGroups.at(0).get('inputNA').valid).toBeFalse()
    }))

    it('should validate ipv6 prefix reservation', fakeAsync(() => {
        spyOn(dhcpApi, 'createHostBegin').and.returnValue(of(cannedResponseBegin))
        component.ngOnInit()
        tick()
        fixture.detectChanges()

        component.ipGroups.at(0).get('ipType').setValue('ia_pd')
        fixture.detectChanges()

        expect(component.ipGroups.length).toBe(1)
        component.ipGroups.at(0).get('inputPD').setValue('invalid')
        fixture.detectChanges()
        expect(component.ipGroups.at(0).get('inputPD').valid).toBeFalse()
    }))

    it('should expect at least one reservation', fakeAsync(() => {
        spyOn(dhcpApi, 'createHostBegin').and.returnValue(of(cannedResponseBegin))
        component.ngOnInit()
        tick()
        fixture.detectChanges()

        component.formGroup.get('selectedServers').setValue([1])
        component.onServersChange()
        fixture.detectChanges()

        component.formGroup.get('selectedSubnet').setValue(1)
        fixture.detectChanges()

        component.formGroup.get('hostIdGroup.idInputHex').setValue('01:02:03:04:05:06')
        fixture.detectChanges()

        expect(component.formGroup.valid).toBeFalse()

        component.ipGroups.at(0).get('inputIPv4').setValue('10.0.0.1')
        fixture.detectChanges()

        expect(component.formGroup.valid).toBeTrue()

        component.deleteIPInput(0)
        fixture.detectChanges()

        expect(component.formGroup.valid).toBeFalse()

        component.formGroup.get('hostname').setValue('example.org')
        fixture.detectChanges()

        expect(component.formGroup.valid).toBeTrue()
    }))

    it('should present an error message when begin transaction fails', fakeAsync(() => {
        spyOn(dhcpApi, 'createHostBegin').and.returnValues(throwError({ status: 404 }), of(cannedResponseBegin))
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

    it('should present an error message when submit fails', fakeAsync(() => {
        spyOn(dhcpApi, 'createHostBegin').and.returnValue(of(cannedResponseBegin))
        component.ngOnInit()
        tick()
        fixture.detectChanges()

        spyOn(dhcpApi, 'createHostSubmit').and.returnValue(throwError({ status: 404 }))
        spyOn(messageService, 'add')
        component.onSubmit()
        tick()
        fixture.detectChanges()

        expect(messageService.add).toHaveBeenCalled()
    }))
})
