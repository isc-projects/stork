import { ComponentFixture, fakeAsync, flush, TestBed, tick } from '@angular/core/testing'
import { RouterTestingModule } from '@angular/router/testing'
import { UntypedFormArray, UntypedFormBuilder, FormsModule, ReactiveFormsModule } from '@angular/forms'
import { HttpClientTestingModule } from '@angular/common/http/testing'
import { By } from '@angular/platform-browser'
import { NoopAnimationsModule } from '@angular/platform-browser/animations'
import { of, throwError } from 'rxjs'
import { MessageService } from 'primeng/api'
import { ButtonModule } from 'primeng/button'
import { CheckboxModule } from 'primeng/checkbox'
import { DropdownModule } from 'primeng/dropdown'
import { FieldsetModule } from 'primeng/fieldset'
import { InputNumberModule } from 'primeng/inputnumber'
import { InputSwitchModule } from 'primeng/inputswitch'
import { MessagesModule } from 'primeng/messages'
import { MultiSelectModule } from 'primeng/multiselect'
import { OverlayPanelModule } from 'primeng/overlaypanel'
import { HostFormComponent } from './host-form.component'
import { ToggleButtonModule } from 'primeng/togglebutton'
import { SplitButtonModule } from 'primeng/splitbutton'
import { DhcpOptionFormComponent } from '../dhcp-option-form/dhcp-option-form.component'
import { DhcpOptionSetFormComponent } from '../dhcp-option-set-form/dhcp-option-set-form.component'
import { DhcpOptionFieldFormGroup, DhcpOptionFieldType } from '../forms/dhcp-option-field'
import { HelpTipComponent } from '../help-tip/help-tip.component'
import { DHCPService } from '../backend'
import { DhcpClientClassSetFormComponent } from '../dhcp-client-class-set-form/dhcp-client-class-set-form.component'
import { ChipsModule } from 'primeng/chips'
import { TableModule } from 'primeng/table'
import { ProgressSpinnerModule } from 'primeng/progressspinner'

describe('HostFormComponent', () => {
    let component: HostFormComponent
    let fixture: ComponentFixture<HostFormComponent>
    let dhcpApi: DHCPService
    let messageService: MessageService
    let formBuilder: UntypedFormBuilder = new UntypedFormBuilder()

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
        clientClasses: ['router', 'cable-modem'],
    }

    beforeEach(async () => {
        await TestBed.configureTestingModule({
            providers: [UntypedFormBuilder, DHCPService, MessageService],
            imports: [
                ButtonModule,
                CheckboxModule,
                ChipsModule,
                DropdownModule,
                FieldsetModule,
                FormsModule,
                HttpClientTestingModule,
                InputNumberModule,
                InputSwitchModule,
                MessagesModule,
                MultiSelectModule,
                NoopAnimationsModule,
                OverlayPanelModule,
                ReactiveFormsModule,
                RouterTestingModule,
                SplitButtonModule,
                TableModule,
                ToggleButtonModule,
                ProgressSpinnerModule,
            ],
            declarations: [
                DhcpClientClassSetFormComponent,
                DhcpOptionFormComponent,
                DhcpOptionSetFormComponent,
                HelpTipComponent,
                HostFormComponent,
            ],
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

        component.formGroup.get('selectedDaemons').setValue([1])
        component.onDaemonsChange()
        expect(component.form.filteredDaemons.length).toBe(2)
        expect(component.form.filteredDaemons[0].id).toBe(1)
        expect(component.form.filteredDaemons[1].id).toBe(2)

        component.formGroup.get('selectedDaemons').setValue([2])
        component.onDaemonsChange()
        expect(component.form.filteredDaemons.length).toBe(2)
        expect(component.form.filteredDaemons[0].id).toBe(1)
        expect(component.form.filteredDaemons[1].id).toBe(2)
    }))

    it('should enable specific controls for dhcpv6 servers', fakeAsync(() => {
        spyOn(dhcpApi, 'createHostBegin').and.returnValue(of(cannedResponseBegin))
        component.ngOnInit()
        tick()
        fixture.detectChanges()

        component.formGroup.get('selectedDaemons').setValue([3, 4, 5])
        component.onDaemonsChange()
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

        component.formGroup.get('selectedDaemons').setValue([2])
        component.onDaemonsChange()
        expect(component.form.filteredSubnets.length).toBe(2)

        component.formGroup.get('selectedDaemons').setValue([1, 2])
        component.onDaemonsChange()
        expect(component.form.filteredSubnets.length).toBe(1)
    }))

    it('should show filter overlapping ipv6 subnets for selected servers', fakeAsync(() => {
        spyOn(dhcpApi, 'createHostBegin').and.returnValue(of(cannedResponseBegin))
        component.ngOnInit()
        tick()
        fixture.detectChanges()
        expect(component.form.filteredSubnets.length).toBe(4)

        component.formGroup.get('selectedDaemons').setValue([4])
        component.onDaemonsChange()
        expect(component.form.filteredSubnets.length).toBe(1)

        component.formGroup.get('selectedDaemons').setValue([4, 5])
        component.onDaemonsChange()
        expect(component.form.filteredSubnets.length).toBe(0)
    }))

    it('should require server specification', fakeAsync(() => {
        spyOn(dhcpApi, 'createHostBegin').and.returnValue(of(cannedResponseBegin))
        component.ngOnInit()
        tick()
        fixture.detectChanges()

        component.formGroup.get('selectedDaemons').setValue([4])
        component.onDaemonsChange()
        fixture.detectChanges()
        expect(component.formGroup.get('selectedDaemons').valid).toBeTrue()

        component.formGroup.get('selectedDaemons').setValue([])
        component.onDaemonsChange()
        fixture.detectChanges()
        expect(component.formGroup.get('selectedDaemons').valid).toBeFalse()

        component.formGroup.get('selectedDaemons').markAsTouched()
        component.formGroup.get('selectedDaemons').markAsDirty()
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

        component.formGroup.get('selectedSubnet').markAsTouched()
        component.formGroup.get('selectedSubnet').markAsDirty()
        component.formGroup.get('selectedSubnet').setValue(null)
        fixture.detectChanges()
        expect(component.formGroup.get('selectedSubnet').valid).toBeFalse()

        component.formGroup.get('selectedSubnet').markAsTouched()
        component.formGroup.get('selectedSubnet').markAsDirty()
        fixture.detectChanges()

        let errmsg = fixture.debugElement.query(By.css('small'))
        expect(errmsg).toBeTruthy()
        expect(errmsg.nativeElement.innerText).toContain('A subnet must be selected if the reservation is not global.')
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

        component.formGroup.get('selectedDaemons').setValue([1])
        component.onDaemonsChange()

        expect(component.hostIdTypes.length).toBe(5)
        expect(component.hostIdTypes[0].label).toBe('hw-address')
        expect(component.hostIdTypes[1].label).toBe('client-id')
        expect(component.hostIdTypes[2].label).toBe('circuit-id')
        expect(component.hostIdTypes[3].label).toBe('duid')
        expect(component.hostIdTypes[4].label).toBe('flex-id')

        component.formGroup.get('selectedDaemons').setValue([3])
        component.onDaemonsChange()

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

        component.formGroup.get('hostIdGroup.idInputHex').markAsTouched()
        component.formGroup.get('hostIdGroup.idInputHex').markAsDirty()
        fixture.detectChanges()

        const errmsg = fixture.debugElement.query(By.css('small'))
        expect(errmsg).toBeTruthy()
        expect(errmsg.nativeElement.innerText).toContain('Please specify valid hexadecimal digits (e.g., ab:09:ef:01).')
    }))

    it('should validate hw-address hex identifier length', fakeAsync(() => {
        spyOn(dhcpApi, 'createHostBegin').and.returnValue(of(cannedResponseBegin))
        component.ngOnInit()
        tick()
        fixture.detectChanges()

        expect(component.formGroup.get('hostIdGroup.idFormat').value).toBe('hex')

        const pattern = '11'
        component.formGroup.get('hostIdGroup.idInputHex').setValue(pattern.repeat(21))
        expect(component.formGroup.get('hostIdGroup.idInputHex').valid).toBeFalse()

        component.formGroup.get('hostIdGroup.idInputHex').markAsTouched()
        component.formGroup.get('hostIdGroup.idInputHex').markAsDirty()
        fixture.detectChanges()

        const errmsg = fixture.debugElement.query(By.css('small'))
        expect(errmsg).toBeTruthy()
        expect(errmsg.nativeElement.innerText).toContain(
            'The number of hexadecimal digits exceeds the maximum value of 40.'
        )
    }))

    it('should validate other hex identifier length', fakeAsync(() => {
        spyOn(dhcpApi, 'createHostBegin').and.returnValue(of(cannedResponseBegin))
        component.ngOnInit()
        tick()
        fixture.detectChanges()

        expect(component.formGroup.get('hostIdGroup.idFormat').value).toBe('hex')

        component.formGroup.get('hostIdGroup.idType').setValue('client-id')
        component.onSelectedIdentifierChange()
        const pattern = '11'
        component.formGroup.get('hostIdGroup.idInputHex').setValue(pattern.repeat(129))
        expect(component.formGroup.get('hostIdGroup.idInputHex').valid).toBeFalse()

        component.formGroup.get('hostIdGroup.idInputHex').markAsTouched()
        component.formGroup.get('hostIdGroup.idInputHex').markAsDirty()
        fixture.detectChanges()

        const errmsg = fixture.debugElement.query(By.css('small'))
        expect(errmsg).toBeTruthy()
        expect(errmsg.nativeElement.innerText).toContain(
            'The number of hexadecimal digits exceeds the maximum value of 256.'
        )
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

        component.formGroup.get('hostIdGroup.idInputText').markAsTouched()
        component.formGroup.get('hostIdGroup.idInputText').markAsDirty()
        fixture.detectChanges()

        const errmsg = fixture.debugElement.query(By.css('small'))
        expect(errmsg).toBeTruthy()
        expect(errmsg.nativeElement.innerText).toContain('DHCP identifier is required.')
    }))

    it('should validate hw-address text identifier length', fakeAsync(() => {
        spyOn(dhcpApi, 'createHostBegin').and.returnValue(of(cannedResponseBegin))
        component.ngOnInit()
        tick()
        fixture.detectChanges()

        component.formGroup.get('hostIdGroup.idFormat').setValue('text')
        fixture.detectChanges()
        expect(component.formGroup.get('hostIdGroup.idFormat').value).toBe('text')

        const pattern = 'a'
        component.formGroup.get('hostIdGroup.idInputText').setValue(pattern.repeat(21))
        expect(component.formGroup.get('hostIdGroup.idInputText').valid).toBeFalse()

        component.formGroup.get('hostIdGroup.idInputText').markAsTouched()
        component.formGroup.get('hostIdGroup.idInputText').markAsDirty()
        fixture.detectChanges()

        const errmsg = fixture.debugElement.query(By.css('small'))
        expect(errmsg).toBeTruthy()
        expect(errmsg.nativeElement.innerText).toContain('The identifier length exceeds the maximum value of 20.')
    }))

    it('should validate other text identifier length', fakeAsync(() => {
        spyOn(dhcpApi, 'createHostBegin').and.returnValue(of(cannedResponseBegin))
        component.ngOnInit()
        tick()
        fixture.detectChanges()

        component.formGroup.get('hostIdGroup.idFormat').setValue('text')
        fixture.detectChanges()
        expect(component.formGroup.get('hostIdGroup.idFormat').value).toBe('text')

        component.formGroup.get('hostIdGroup.idType').setValue('duid')
        component.onSelectedIdentifierChange()
        const pattern = 'a'
        component.formGroup.get('hostIdGroup.idInputText').setValue(pattern.repeat(129))
        expect(component.formGroup.get('hostIdGroup.idInputText').valid).toBeFalse()

        component.formGroup.get('hostIdGroup.idInputText').markAsTouched()
        component.formGroup.get('hostIdGroup.idInputText').markAsDirty()
        fixture.detectChanges()

        const errmsg = fixture.debugElement.query(By.css('small'))
        expect(errmsg).toBeTruthy()
        expect(errmsg.nativeElement.innerText).toContain('The identifier length exceeds the maximum value of 128.')
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

        component.formGroup.get('selectedDaemons').setValue([3])
        component.onDaemonsChange()
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

        component.formGroup.get('selectedDaemons').setValue([3])
        component.onDaemonsChange()
        component.addIPInput()
        fixture.detectChanges()

        expect(component.ipGroups.length).toBe(2)
        component.ipGroups.at(0).get('ipType').setValue('ia_pd')
        component.ipGroups.at(0).get('inputPD').setValue('3001::')
        component.ipGroups.at(0).get('inputPDLength').setValue(96)
        component.ipGroups.at(1).get('ipType').setValue('ia_na')
        component.ipGroups.at(1).get('inputNA').setValue('2001:db8:1::1')

        component.formGroup.get('selectedDaemons').setValue([1])
        component.onDaemonsChange()
        fixture.detectChanges()

        expect(component.ipGroups.length).toBe(1)
        expect(component.ipGroups.at(0).get('ipType').value).toBe('ipv4')
        expect(component.ipGroups.at(0).get('inputIPv4').value).toBe('')

        component.deleteIPInput(0)

        component.formGroup.get('selectedDaemons').setValue([3])
        component.onDaemonsChange()
        fixture.detectChanges()

        expect(component.ipGroups.length).toBe(0)
    }))

    it('should show the button for adding next IP reservation when there are none', fakeAsync(() => {
        spyOn(dhcpApi, 'createHostBegin').and.returnValue(of(cannedResponseBegin))
        component.ngOnInit()
        tick()
        fixture.detectChanges()

        component.deleteIPInput(0)
        fixture.detectChanges()

        expect(fixture.debugElement.query(By.css('[label="Add IP Reservation"]'))).toBeTruthy()
    }))

    it('should show the button for adding next ip reservation for dhcpv6 server', fakeAsync(() => {
        spyOn(dhcpApi, 'createHostBegin').and.returnValue(of(cannedResponseBegin))
        component.ngOnInit()
        tick()
        fixture.detectChanges()

        component.formGroup.get('selectedDaemons').setValue([3])
        component.onDaemonsChange()
        fixture.detectChanges()

        expect(fixture.debugElement.query(By.css('[label="Add IP Reservation"]'))).toBeTruthy()
    }))

    it('should hide the button for adding next ip reservation for dhcpv4 server', fakeAsync(() => {
        spyOn(dhcpApi, 'createHostBegin').and.returnValue(of(cannedResponseBegin))
        component.ngOnInit()
        tick()
        fixture.detectChanges()

        expect(fixture.debugElement.query(By.css('[label="Add IP Reservation"]'))).toBeFalsy()
    }))

    it('should validate ipv4 reservation', fakeAsync(() => {
        spyOn(dhcpApi, 'createHostBegin').and.returnValue(of(cannedResponseBegin))
        component.ngOnInit()
        tick()
        fixture.detectChanges()

        expect(component.ipGroups.length).toBe(1)
        component.ipGroups.at(0).get('inputIPv4').setValue('invalid')
        component.ipGroups.at(0).get('inputIPv4').markAsTouched()
        component.ipGroups.at(0).get('inputIPv4').markAsDirty()
        fixture.detectChanges()

        expect(component.ipGroups.at(0).get('inputIPv4').valid).toBeFalse()

        const errmsg = fixture.debugElement.query(By.css('small'))
        expect(errmsg).toBeTruthy()
        expect(errmsg.nativeElement.innerText).toContain('Please specify a valid IPv4 address.')
    }))

    it('should check that ipv4 reservation is within the subnet boundaries', fakeAsync(() => {
        spyOn(dhcpApi, 'createHostBegin').and.returnValue(of(cannedResponseBegin))
        component.ngOnInit()
        tick()
        fixture.detectChanges()

        expect(component.ipGroups.length).toBe(1)
        component.ipGroups.at(0).get('inputIPv4').setValue('192.0.3.2')
        component.ipGroups.at(0).get('inputIPv4').markAsTouched()
        component.ipGroups.at(0).get('inputIPv4').markAsDirty()
        fixture.detectChanges()

        // Initially, the specified address it not matched with the subnet prefix
        // because the subnet is not selected.
        expect(component.ipGroups.at(0).get('inputIPv4').valid).toBeTrue()

        component.formGroup.get('selectedSubnet').setValue(1)
        component.onSelectedSubnetChange()
        fixture.detectChanges()
        expect(component.formGroup.get('selectedSubnet').valid).toBeTrue()

        // The subnet has been selected. This time the address should match
        // the subnet prefix.
        expect(component.ipGroups.at(0).get('inputIPv4').valid).toBeFalse()

        const errmsg = fixture.debugElement.query(By.css('small'))
        expect(errmsg).toBeTruthy()
        expect(errmsg.nativeElement.innerText).toContain('IP address is not in the subnet 192.0.2.0/24 range.')
    }))

    it('should replace ipv4 placeholder after subnet selection', fakeAsync(() => {
        spyOn(dhcpApi, 'createHostBegin').and.returnValue(of(cannedResponseBegin))
        component.ngOnInit()
        tick()
        fixture.detectChanges()

        component.formGroup.get('selectedSubnet').setValue(1)
        component.onSelectedSubnetChange()
        expect(component.ipv4Placeholder).toBe('in range of 192.0.2.0 - 192.0.2.255')

        component.formGroup.get('selectedSubnet').setValue(null)
        component.onSelectedSubnetChange()
        expect(component.ipv4Placeholder).toBe('?.?.?.?')
    }))

    it('should validate ipv6 address reservation', fakeAsync(() => {
        spyOn(dhcpApi, 'createHostBegin').and.returnValue(of(cannedResponseBegin))
        component.ngOnInit()
        tick()
        fixture.detectChanges()

        component.formGroup.get('selectedDaemons').setValue([3])
        component.onDaemonsChange()

        component.ipGroups.at(0).get('ipType').setValue('ia_na')
        fixture.detectChanges()

        expect(component.ipGroups.length).toBe(1)

        component.ipGroups.at(0).get('inputNA').setValue('192.0.2.1')
        component.ipGroups.at(0).get('inputNA').markAsTouched()
        component.ipGroups.at(0).get('inputNA').markAsDirty()
        fixture.detectChanges()

        expect(component.ipGroups.at(0).get('inputNA').valid).toBeFalse()

        const errmsg = fixture.debugElement.query(By.css('small'))
        expect(errmsg).toBeTruthy()
        expect(errmsg.nativeElement.innerText).toContain('Please specify a valid IPv6 address.')
    }))

    it('should check that ipv6 address reservation is within the subnet boundaries', fakeAsync(() => {
        spyOn(dhcpApi, 'createHostBegin').and.returnValue(of(cannedResponseBegin))
        component.ngOnInit()
        tick()
        fixture.detectChanges()

        component.formGroup.get('selectedDaemons').setValue([4])
        component.onDaemonsChange()

        component.ipGroups.at(0).get('ipType').setValue('ia_na')
        fixture.detectChanges()

        expect(component.ipGroups.length).toBe(1)

        component.ipGroups.at(0).get('inputNA').setValue('2001:db8:2::56')
        component.ipGroups.at(0).get('inputNA').markAsTouched()
        component.ipGroups.at(0).get('inputNA').markAsDirty()
        fixture.detectChanges()

        // Initially, the specified address it not matched with the subnet prefix
        // because the subnet is not selected.
        expect(component.ipGroups.at(0).get('inputNA').valid).toBeTrue()

        component.formGroup.get('selectedSubnet').setValue(3)
        component.onSelectedSubnetChange()
        fixture.detectChanges()
        expect(component.formGroup.get('selectedSubnet').valid).toBeTrue()

        // The subnet has been selected. This time the address should match
        // the subnet prefix.
        expect(component.ipGroups.at(0).get('inputNA').valid).toBeFalse()

        const errmsg = fixture.debugElement.query(By.css('small'))
        expect(errmsg).toBeTruthy()
        expect(errmsg.nativeElement.innerText).toContain('IP address is not in the subnet 2001:db8:1::/64 range.')
    }))

    it('should replace ipv6 placeholder after subnet selection', fakeAsync(() => {
        spyOn(dhcpApi, 'createHostBegin').and.returnValue(of(cannedResponseBegin))
        component.ngOnInit()
        tick()
        fixture.detectChanges()

        component.formGroup.get('selectedDaemons').setValue([5])
        component.onDaemonsChange()

        component.ipGroups.at(0).get('ipType').setValue('ia_na')
        fixture.detectChanges()

        expect(component.ipGroups.length).toBe(1)

        component.formGroup.get('selectedSubnet').setValue(4)
        component.onSelectedSubnetChange()
        expect(component.ipv6Placeholder).toBe('2001:db8:2::')

        component.formGroup.get('selectedSubnet').setValue(null)
        component.onSelectedSubnetChange()
        expect(component.ipv6Placeholder).toBe('e.g. 2001:db8:1::')
    }))

    it('should validate ipv6 prefix reservation', fakeAsync(() => {
        spyOn(dhcpApi, 'createHostBegin').and.returnValue(of(cannedResponseBegin))
        component.ngOnInit()
        tick()
        fixture.detectChanges()

        component.formGroup.get('selectedDaemons').setValue([3])
        component.onDaemonsChange()

        component.ipGroups.at(0).get('ipType').setValue('ia_pd')
        fixture.detectChanges()

        expect(component.ipGroups.length).toBe(1)

        component.ipGroups.at(0).get('inputPD').setValue('invalid')
        component.ipGroups.at(0).get('inputPD').markAsTouched()
        component.ipGroups.at(0).get('inputPD').markAsDirty()
        fixture.detectChanges()

        expect(component.ipGroups.at(0).get('inputPD').valid).toBeFalse()

        const errmsg = fixture.debugElement.query(By.css('small'))
        expect(errmsg).toBeTruthy()
        expect(errmsg.nativeElement.innerText).toContain('Please specify a valid IPv6 prefix.')
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

    it('should submit new dhcpv4 host', fakeAsync(() => {
        spyOn(dhcpApi, 'createHostBegin').and.returnValue(of(cannedResponseBegin))
        component.ngOnInit()
        tick()
        fixture.detectChanges()

        component.formGroup.get('selectedDaemons').setValue([1])
        component.onDaemonsChange()
        component.formGroup.get('selectedSubnet').setValue(1)
        component.formGroup.get('hostIdGroup.idInputHex').setValue('01:02:03:04:05:06')
        component.ipGroups.at(0).get('inputIPv4').setValue('192.0.2.4')
        component.formGroup.get('hostname').setValue('example.org')
        component.getClientClassSetControl(0).setValue(['cable-modem', 'router'])
        component.getOptionSetArray(0).push(
            formBuilder.group({
                optionCode: [5],
                alwaysSend: true,
                optionFields: formBuilder.array([
                    new DhcpOptionFieldFormGroup(DhcpOptionFieldType.IPv4Address, {
                        control: formBuilder.control('192.0.2.1'),
                    }),
                ]),
                suboptions: formBuilder.array([]),
            })
        )
        fixture.detectChanges()

        expect(component.formGroup.valid).toBeTrue()

        const okResp: any = {
            status: 200,
        }
        spyOn(dhcpApi, 'createHostSubmit').and.returnValue(of(okResp))
        spyOn(component.formSubmit, 'emit')
        spyOn(messageService, 'add')
        component.onSubmit()
        tick()
        fixture.detectChanges()

        const host: any = {
            subnetId: 1,
            hostIdentifiers: [
                {
                    idType: 'hw-address',
                    idHexValue: '01:02:03:04:05:06',
                },
            ],
            addressReservations: [
                {
                    address: '192.0.2.4/32',
                },
            ],
            prefixReservations: [],
            hostname: 'example.org',
            localHosts: [
                {
                    clientClasses: ['cable-modem', 'router'],
                    daemonId: 1,
                    dataSource: 'api',
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
            ],
        }
        expect(dhcpApi.createHostSubmit).toHaveBeenCalledWith(component.form.transactionId, host)
        expect(component.formSubmit.emit).toHaveBeenCalled()
        expect(messageService.add).toHaveBeenCalled()
    }))

    it('should submit new dhcpv4 host with no reservations', fakeAsync(() => {
        spyOn(dhcpApi, 'createHostBegin').and.returnValue(of(cannedResponseBegin))
        component.ngOnInit()
        tick()
        fixture.detectChanges()

        component.formGroup.get('selectedDaemons').setValue([1])
        component.formGroup.get('selectedSubnet').setValue(1)
        component.formGroup.get('hostIdGroup.idInputHex').setValue('01:02:03:04:05:06')
        fixture.detectChanges()

        expect(component.formGroup.valid).toBeTrue()

        const okResp: any = {
            status: 200,
        }
        spyOn(dhcpApi, 'createHostSubmit').and.returnValue(of(okResp))
        spyOn(component.formSubmit, 'emit')
        spyOn(messageService, 'add')
        component.onSubmit()
        tick()
        fixture.detectChanges()

        const host: any = {
            subnetId: 1,
            hostIdentifiers: [
                {
                    idType: 'hw-address',
                    idHexValue: '01:02:03:04:05:06',
                },
            ],
            addressReservations: [],
            prefixReservations: [],
            hostname: '',
            localHosts: [
                {
                    daemonId: 1,
                    dataSource: 'api',
                    clientClasses: [],
                    options: [],
                },
            ],
        }
        expect(dhcpApi.createHostSubmit).toHaveBeenCalledWith(component.form.transactionId, host)
        expect(component.formSubmit.emit).toHaveBeenCalled()
        expect(messageService.add).toHaveBeenCalled()
    }))

    it('should submit new dhcpv6 host', fakeAsync(() => {
        spyOn(dhcpApi, 'createHostBegin').and.returnValue(of(cannedResponseBegin))
        component.ngOnInit()
        tick()
        fixture.detectChanges()

        component.formGroup.get('selectedDaemons').setValue([5])
        component.formGroup.get('selectedSubnet').setValue(4)
        component.onDaemonsChange()
        component.onSelectedSubnetChange()
        fixture.detectChanges()

        component.addIPInput()

        component.formGroup.get('hostIdGroup.idType').setValue('flex-id')
        component.formGroup.get('hostIdGroup.idFormat').setValue('text')
        component.formGroup.get('hostIdGroup.idInputText').setValue(' foobar ')
        component.ipGroups.at(0).get('ipType').setValue('ia_na')
        component.ipGroups.at(0).get('inputNA').setValue('2001:db8:2::100')
        component.ipGroups.at(1).get('ipType').setValue('ia_pd')
        component.ipGroups.at(1).get('inputPD').setValue('3000::')
        component.ipGroups.at(1).get('inputPDLength').setValue('56')

        expect(component.formGroup.valid).toBeTrue()

        const okResp: any = {
            status: 200,
        }
        spyOn(dhcpApi, 'createHostSubmit').and.returnValue(of(okResp))
        spyOn(component.formSubmit, 'emit')
        spyOn(messageService, 'add')
        component.onSubmit()
        tick()
        fixture.detectChanges()

        const host: any = {
            subnetId: 4,
            hostIdentifiers: [
                {
                    idType: 'flex-id',
                    idHexValue: '66:6f:6f:62:61:72',
                },
            ],
            hostname: '',
            addressReservations: [
                {
                    address: '2001:db8:2::100/128',
                },
            ],
            prefixReservations: [
                {
                    address: '3000::/56',
                },
            ],
            localHosts: [
                {
                    daemonId: 5,
                    dataSource: 'api',
                    clientClasses: [],
                    options: [],
                },
            ],
        }
        expect(dhcpApi.createHostSubmit).toHaveBeenCalledWith(component.form.transactionId, host)
        expect(component.formSubmit.emit).toHaveBeenCalled()
        expect(messageService.add).toHaveBeenCalled()
    }))

    it('should submit new dhcpv6 host with no reservations', fakeAsync(() => {
        spyOn(dhcpApi, 'createHostBegin').and.returnValue(of(cannedResponseBegin))
        component.ngOnInit()
        tick()
        fixture.detectChanges()

        component.formGroup.get('selectedDaemons').setValue([5])
        component.formGroup.get('selectedSubnet').setValue(4)
        component.onDaemonsChange()
        fixture.detectChanges()

        component.formGroup.get('hostIdGroup.idType').setValue('flex-id')
        component.formGroup.get('hostIdGroup.idFormat').setValue('text')
        component.formGroup.get('hostIdGroup.idInputText').setValue(' foobar ')

        expect(component.formGroup.valid).toBeTrue()

        const okResp: any = {
            status: 200,
        }
        spyOn(dhcpApi, 'createHostSubmit').and.returnValue(of(okResp))
        spyOn(component.formSubmit, 'emit')
        spyOn(messageService, 'add')
        component.onSubmit()
        tick()
        fixture.detectChanges()

        const host: any = {
            subnetId: 4,
            hostIdentifiers: [
                {
                    idType: 'flex-id',
                    idHexValue: '66:6f:6f:62:61:72',
                },
            ],
            hostname: '',
            addressReservations: [],
            prefixReservations: [],
            localHosts: [
                {
                    daemonId: 5,
                    dataSource: 'api',
                    clientClasses: [],
                    options: [],
                },
            ],
        }
        expect(dhcpApi.createHostSubmit).toHaveBeenCalledWith(component.form.transactionId, host)
        expect(component.formSubmit.emit).toHaveBeenCalled()
        expect(messageService.add).toHaveBeenCalled()
    }))

    it('should submit new dhcpv6 host with different local hosts', fakeAsync(() => {
        spyOn(dhcpApi, 'createHostBegin').and.returnValue(of(cannedResponseBegin))
        component.ngOnInit()
        tick()
        fixture.detectChanges()

        component.formGroup.get('selectedDaemons').setValue([4, 5])
        component.formGroup.get('globalReservation').setValue(true)
        component.onDaemonsChange()
        component.onSelectedSubnetChange()
        fixture.detectChanges()

        component.addIPInput()

        component.formGroup.get('hostIdGroup.idType').setValue('flex-id')
        component.formGroup.get('hostIdGroup.idFormat').setValue('text')
        component.formGroup.get('hostIdGroup.idInputText').setValue(' foobar ')

        component.formGroup.get('splitFormMode').setValue(true)
        component.onSplitModeChange()
        fixture.detectChanges()

        expect(component.optionsArray.length).toBe(2)

        component.getOptionSetArray(0).push(
            formBuilder.group({
                optionCode: [23],
                alwaysSend: true,
                optionFields: formBuilder.array([
                    new DhcpOptionFieldFormGroup(DhcpOptionFieldType.IPv6Address, {
                        control: formBuilder.control('2001:db8:1::1'),
                    }),
                ]),
                suboptions: formBuilder.array([]),
            })
        )

        component.getOptionSetArray(1).push(
            formBuilder.group({
                optionCode: [23],
                alwaysSend: true,
                optionFields: formBuilder.array([
                    new DhcpOptionFieldFormGroup(DhcpOptionFieldType.IPv6Address, {
                        control: formBuilder.control('2001:db8:1::2'),
                    }),
                ]),
                suboptions: formBuilder.array([]),
            })
        )

        component.getClientClassSetControl(0).setValue(['foo', 'bar'])
        component.getClientClassSetControl(1).setValue(['baz', 'bar'])

        expect(component.formGroup.valid).toBeTrue()

        const okResp: any = {
            status: 200,
        }
        spyOn(dhcpApi, 'createHostSubmit').and.returnValue(of(okResp))
        spyOn(component.formSubmit, 'emit')
        spyOn(messageService, 'add')
        component.onSubmit()
        tick()
        fixture.detectChanges()

        const host: any = {
            subnetId: 0,
            hostIdentifiers: [
                {
                    idType: 'flex-id',
                    idHexValue: '66:6f:6f:62:61:72',
                },
            ],
            addressReservations: [],
            prefixReservations: [],
            hostname: '',
            localHosts: [
                {
                    daemonId: 4,
                    dataSource: 'api',
                    clientClasses: ['foo', 'bar'],
                    options: [
                        {
                            alwaysSend: true,
                            code: 23,
                            encapsulate: '',
                            fields: [
                                {
                                    fieldType: 'ipv6-address',
                                    values: ['2001:db8:1::1'],
                                },
                            ],
                            options: [],
                            universe: 6,
                        },
                    ],
                },
                {
                    daemonId: 5,
                    dataSource: 'api',
                    clientClasses: ['baz', 'bar'],
                    options: [
                        {
                            alwaysSend: true,
                            code: 23,
                            encapsulate: '',
                            fields: [
                                {
                                    fieldType: 'ipv6-address',
                                    values: ['2001:db8:1::2'],
                                },
                            ],
                            options: [],
                            universe: 6,
                        },
                    ],
                },
            ],
        }
        expect(dhcpApi.createHostSubmit).toHaveBeenCalledWith(component.form.transactionId, host)
        expect(component.formSubmit.emit).toHaveBeenCalled()
        expect(messageService.add).toHaveBeenCalled()

        flush()
    }))

    it('should present an error message when processing options fails', fakeAsync(() => {
        spyOn(dhcpApi, 'createHostBegin').and.returnValue(of(cannedResponseBegin))
        component.ngOnInit()
        tick()
        fixture.detectChanges()

        component.formGroup.get('selectedDaemons').setValue([1])
        component.formGroup.get('selectedSubnet').setValue(1)
        component.formGroup.get('hostIdGroup.idInputHex').setValue('01:02:03:04:05:06')
        component.ipGroups.at(0).get('inputIPv4').setValue('192.0.2.4')
        component.getOptionSetArray(0).push(
            formBuilder.group({
                optionCode: ['abc'],
                alwaysSend: false,
                optionFields: formBuilder.array([
                    new DhcpOptionFieldFormGroup(DhcpOptionFieldType.IPv4Address, {
                        control: formBuilder.control('192.0.2.1'),
                    }),
                ]),
                suboptions: formBuilder.array([]),
            })
        )
        fixture.detectChanges()

        const okResp: any = {
            status: 200,
        }
        spyOn(dhcpApi, 'createHostSubmit').and.returnValue(of(okResp))
        spyOn(messageService, 'add')
        component.onSubmit()
        tick()
        fixture.detectChanges()

        expect(dhcpApi.createHostSubmit).not.toHaveBeenCalled()
        expect(messageService.add).toHaveBeenCalled()
    }))

    it('should present an error message when submit fails', fakeAsync(() => {
        spyOn(dhcpApi, 'createHostBegin').and.returnValue(of(cannedResponseBegin))
        component.ngOnInit()
        tick()
        fixture.detectChanges()

        spyOn(dhcpApi, 'createHostSubmit').and.returnValue(throwError({ status: 404 }))
        spyOn(component.formSubmit, 'emit')
        spyOn(messageService, 'add')
        component.onSubmit()
        tick()
        fixture.detectChanges()

        expect(component.formSubmit.emit).not.toHaveBeenCalled()
        expect(messageService.add).toHaveBeenCalled()
    }))

    it('should include dhcpv4 options form', fakeAsync(() => {
        spyOn(dhcpApi, 'createHostBegin').and.returnValue(of(cannedResponseBegin))
        component.ngOnInit()
        tick()
        fixture.detectChanges()

        const optionsForm = fixture.debugElement.query(By.css('app-dhcp-option-set-form'))
        expect(optionsForm).toBeTruthy()
        expect(optionsForm.componentInstance.v6).toBeFalse()
    }))

    it('should include dhcpv6 options form', fakeAsync(() => {
        spyOn(dhcpApi, 'createHostBegin').and.returnValue(of(cannedResponseBegin))
        component.ngOnInit()
        tick()
        fixture.detectChanges()

        component.formGroup.get('selectedDaemons').setValue([3])
        component.onDaemonsChange()
        fixture.detectChanges()

        const optionsForm = fixture.debugElement.query(By.css('app-dhcp-option-set-form'))
        expect(optionsForm).toBeTruthy()
        expect(optionsForm.componentInstance.v6).toBeTrue()
    }))

    it('should include client classes form', fakeAsync(() => {
        spyOn(dhcpApi, 'createHostBegin').and.returnValue(of(cannedResponseBegin))
        component.ngOnInit()
        tick()
        fixture.detectChanges()

        const clientClassesForm = fixture.debugElement.query(By.css('app-dhcp-client-class-set-form'))
        expect(clientClassesForm).toBeTruthy()
    }))

    it('should include boot field inputs for dhcpv4', fakeAsync(() => {
        spyOn(dhcpApi, 'createHostBegin').and.returnValue(of(cannedResponseBegin))
        component.ngOnInit()
        tick()
        fixture.detectChanges()

        component.formGroup.get('selectedDaemons').setValue([0])
        component.onDaemonsChange()
        fixture.detectChanges()

        const nextServerInput = fixture.debugElement.query(By.css('[formControlName="nextServer"]'))
        expect(nextServerInput).toBeTruthy()
    }))

    it('should not include boot field inputs for dhcpv6', fakeAsync(() => {
        spyOn(dhcpApi, 'createHostBegin').and.returnValue(of(cannedResponseBegin))
        component.ngOnInit()
        tick()
        fixture.detectChanges()

        component.formGroup.get('selectedDaemons').setValue([3])
        component.onDaemonsChange()
        fixture.detectChanges()

        const nextServerInput = fixture.debugElement.query(By.css('[formControlName="nextServer"]'))
        expect(nextServerInput).toBeFalsy()
    }))

    it('should enable split editing mode', fakeAsync(() => {
        spyOn(dhcpApi, 'createHostBegin').and.returnValue(of(cannedResponseBegin))
        component.ngOnInit()
        tick()
        fixture.detectChanges()

        component.formGroup.get('splitFormMode').setValue(true)
        component.onSplitModeChange()
        fixture.detectChanges()

        let optionForms = fixture.debugElement.queryAll(By.css('app-dhcp-option-set-form'))
        expect(optionForms).toBeTruthy()
        expect(optionForms.length).toBe(1)

        let clientClassForms = fixture.debugElement.queryAll(By.css('app-dhcp-client-class-set-form'))
        expect(clientClassForms).toBeTruthy()
        expect(clientClassForms.length).toBe(1)

        let nextServerInputs = fixture.debugElement.queryAll(By.css('[formControlName="nextServer"]'))
        expect(nextServerInputs).toBeTruthy()
        expect(nextServerInputs.length).toBe(1)

        component.formGroup.get('selectedDaemons').setValue([2])
        component.onDaemonsChange()
        fixture.detectChanges()

        optionForms = fixture.debugElement.queryAll(By.css('app-dhcp-option-set-form'))
        expect(optionForms).toBeTruthy()
        expect(optionForms.length).toBe(1)

        clientClassForms = fixture.debugElement.queryAll(By.css('app-dhcp-client-class-set-form'))
        expect(clientClassForms).toBeTruthy()
        expect(clientClassForms.length).toBe(1)

        nextServerInputs = fixture.debugElement.queryAll(By.css('[formControlName="nextServer"]'))
        expect(nextServerInputs).toBeTruthy()
        expect(nextServerInputs.length).toBe(1)

        component.formGroup.get('selectedDaemons').setValue([2, 1])
        component.onDaemonsChange()
        fixture.detectChanges()

        optionForms = fixture.debugElement.queryAll(By.css('app-dhcp-option-set-form'))
        expect(optionForms).toBeTruthy()
        expect(optionForms.length).toBe(2)

        clientClassForms = fixture.debugElement.queryAll(By.css('app-dhcp-client-class-set-form'))
        expect(clientClassForms).toBeTruthy()
        expect(clientClassForms.length).toBe(2)

        nextServerInputs = fixture.debugElement.queryAll(By.css('[formControlName="nextServer"]'))
        expect(nextServerInputs).toBeTruthy()
        expect(nextServerInputs.length).toBe(2)

        expect(component.optionsArray.length).toBe(2)
        expect((component.optionsArray.at(0) as UntypedFormArray).length).toBe(0)
        expect((component.optionsArray.at(1) as UntypedFormArray).length).toBe(0)

        expect(component.clientClassesArray.length).toBe(2)
        expect(component.clientClassesArray.get('0').value).toBeFalsy()

        expect(component.bootFieldsArray.length).toBe(2)
        expect(component.bootFieldsArray.get('0.nextServer').value).toBeFalsy()
        expect(component.bootFieldsArray.get('1.nextServer').value).toBeFalsy()

        component.formGroup.get('selectedDaemons').setValue([1])
        component.onDaemonsChange()
        fixture.detectChanges()

        optionForms = fixture.debugElement.queryAll(By.css('app-dhcp-option-set-form'))
        expect(optionForms).toBeTruthy()
        expect(optionForms.length).toBe(1)

        clientClassForms = fixture.debugElement.queryAll(By.css('app-dhcp-client-class-set-form'))
        expect(clientClassForms).toBeTruthy()
        expect(clientClassForms.length).toBe(1)

        nextServerInputs = fixture.debugElement.queryAll(By.css('[formControlName="nextServer"]'))
        expect(nextServerInputs).toBeTruthy()
        expect(nextServerInputs.length).toBe(1)

        component.formGroup.get('selectedDaemons').setValue([])
        component.onDaemonsChange()
        fixture.detectChanges()

        optionForms = fixture.debugElement.queryAll(By.css('app-dhcp-option-set-form'))
        expect(optionForms).toBeTruthy()
        expect(optionForms.length).toBe(1)

        clientClassForms = fixture.debugElement.queryAll(By.css('app-dhcp-client-class-set-form'))
        expect(clientClassForms).toBeTruthy()
        expect(clientClassForms.length).toBe(1)

        nextServerInputs = fixture.debugElement.queryAll(By.css('[formControlName="nextServer"]'))
        expect(nextServerInputs).toBeTruthy()
        expect(nextServerInputs.length).toBe(1)
    }))

    it('should toggle split editing mode', fakeAsync(() => {
        spyOn(dhcpApi, 'createHostBegin').and.returnValue(of(cannedResponseBegin))
        component.ngOnInit()
        tick()
        fixture.detectChanges()

        component.formGroup.get('selectedDaemons').setValue([3, 4, 5])
        component.onDaemonsChange()
        fixture.detectChanges()
        expect(component.optionsArray.length).toBe(1)
        expect(component.formGroup.get('selectedDaemons').value.length).toBe(3)

        component.formGroup.get('splitFormMode').setValue(true)
        component.onSplitModeChange()
        fixture.detectChanges()

        let optionForms = fixture.debugElement.queryAll(By.css('app-dhcp-option-set-form'))
        expect(optionForms).toBeTruthy()
        expect(optionForms.length).toBe(3)

        const optionSetLeft = component.optionsArray.at(0)
        component.formGroup.get('splitFormMode').setValue(false)
        component.onSplitModeChange()
        fixture.detectChanges()

        optionForms = fixture.debugElement.queryAll(By.css('app-dhcp-option-set-form'))
        expect(optionForms).toBeTruthy()
        expect(optionForms.length).toBe(1)
        expect(component.optionsArray.at(0)).toBe(optionSetLeft)
    }))

    it('should clone local host values upon switching to split mode', fakeAsync(() => {
        spyOn(dhcpApi, 'createHostBegin').and.returnValue(of(cannedResponseBegin))
        component.ngOnInit()
        tick()
        fixture.detectChanges()

        component.formGroup.get('selectedDaemons').setValue([1, 2])
        component.onDaemonsChange()
        component.formGroup.get('selectedSubnet').setValue(1)
        component.formGroup.get('hostIdGroup.idInputHex').setValue('01:02:03:04:05:06')
        component.getOptionSetArray(0).push(
            formBuilder.group({
                optionCode: [5],
                alwaysSend: true,
                optionFields: formBuilder.array([
                    new DhcpOptionFieldFormGroup(DhcpOptionFieldType.IPv4Address, {
                        control: formBuilder.control('192.0.2.1'),
                    }),
                ]),
                suboptions: formBuilder.array([]),
            })
        )
        component.getClientClassSetControl(0).setValue(['foo', 'bar'])
        component.getBootFieldsGroup(0).get('nextServer').setValue('192.0.2.1')
        component.getBootFieldsGroup(0).get('serverHostname').setValue('myserver')
        component.getBootFieldsGroup(0).get('bootFileName').setValue('/tmp/boot')

        fixture.detectChanges()

        component.formGroup.get('splitFormMode').setValue(true)
        component.onSplitModeChange()
        fixture.detectChanges()

        expect(component.formGroup.valid).toBeTrue()
        expect(component.optionsArray.length).toBe(2)

        expect(component.getOptionSetArray(0).get('0.optionCode')).toBeTruthy()
        expect(component.getOptionSetArray(1).get('0.optionCode')).toBeTruthy()
        expect(component.getOptionSetArray(0).get('0.optionCode').value).toBe(5)
        expect(component.getOptionSetArray(1).get('0.optionCode').value).toBe(5)

        expect(component.clientClassesArray.length).toBe(2)
        expect(component.getClientClassSetControl(0).value).toEqual(['foo', 'bar'])
        expect(component.getClientClassSetControl(1).value).toEqual(['foo', 'bar'])

        expect(component.bootFieldsArray.length).toBe(2)
        expect(component.getBootFieldsGroup(0).get('nextServer')).toBeTruthy()
        expect(component.getBootFieldsGroup(1).get('nextServer')).toBeTruthy()
        expect(component.getBootFieldsGroup(0).get('serverHostname')).toBeTruthy()
        expect(component.getBootFieldsGroup(1).get('serverHostname')).toBeTruthy()
        expect(component.getBootFieldsGroup(0).get('bootFileName')).toBeTruthy()
        expect(component.getBootFieldsGroup(1).get('bootFileName')).toBeTruthy()
        expect(component.getBootFieldsGroup(0).get('nextServer').value).toBe('192.0.2.1')
        expect(component.getBootFieldsGroup(1).get('nextServer').value).toBe('192.0.2.1')
        expect(component.getBootFieldsGroup(0).get('serverHostname').value).toBe('myserver')
        expect(component.getBootFieldsGroup(1).get('serverHostname').value).toBe('myserver')
        expect(component.getBootFieldsGroup(0).get('bootFileName').value).toBe('/tmp/boot')
        expect(component.getBootFieldsGroup(1).get('bootFileName').value).toBe('/tmp/boot')

        flush()
    }))

    it('should open a form for editing dhcpv4 host', fakeAsync(() => {
        component.hostId = 123

        let beginResponse = cannedResponseBegin
        beginResponse.host = {
            id: 123,
            subnetId: 1,
            subnetPrefix: '192.0.2.0/24',
            hostIdentifiers: [
                {
                    idType: 'hw-address',
                    idHexValue: '01:02:03:04:05:06',
                },
            ],
            addressReservations: [
                {
                    address: '192.0.2.4',
                },
            ],
            prefixReservations: [],
            hostname: 'foo.example.org',
            localHosts: [
                {
                    daemonId: 1,
                    dataSource: 'api',
                    nextServer: '192.2.2.1',
                    serverHostname: 'myserver.example.org',
                    bootFileName: '/tmp/boot1',
                    clientClasses: ['foo'],
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
                {
                    daemonId: 2,
                    dataSource: 'api',
                    nextServer: '192.2.2.1',
                    serverHostname: 'myserver.example.org',
                    bootFileName: '/tmp/boot2',
                    clientClasses: ['bar'],
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
            ],
        }
        spyOn(dhcpApi, 'updateHostBegin').and.returnValue(of(beginResponse))
        component.ngOnInit()
        tick()
        fixture.detectChanges()

        expect(dhcpApi.updateHostBegin).toHaveBeenCalled()
        expect(component.formGroup.valid).toBeTrue()
        expect(component.formGroup.get('splitFormMode').value).toBeTrue()
        expect(component.formGroup.get('globalReservation').value).toBeFalse()
        expect(component.formGroup.get('selectedDaemons').value.length).toBe(2)
        expect(component.formGroup.get('selectedSubnet').value).toBe(1)
        expect(component.ipGroups.length).toBe(1)
        expect(component.ipGroups.get('0.inputIPv4').value).toBe('192.0.2.4')
        expect(component.formGroup.get('hostname').value).toBe('foo.example.org')
        expect(component.optionsArray.length).toBe(2)
        expect(component.getOptionSetArray(0).length).toBe(1)
        expect(component.getOptionSetArray(0).get('0.alwaysSend').value).toBeTrue()
        expect(component.getOptionSetArray(0).get('0.optionCode').value).toBe(5)
        let optionFields = component.getOptionSetArray(0).get('0.optionFields') as UntypedFormArray
        expect(optionFields.length).toBe(1)
        expect(optionFields.get('0.control').value).toBe('192.0.2.1')
        expect(component.getOptionSetArray(1).length).toBe(1)
        expect(component.getOptionSetArray(1).get('0.alwaysSend').value).toBeTrue()
        expect(component.getOptionSetArray(1).get('0.optionCode').value).toBe(5)
        optionFields = component.getOptionSetArray(1).get('0.optionFields') as UntypedFormArray
        expect(optionFields.length).toBe(1)
        expect(optionFields.get('0.control').value).toBe('192.0.2.2')
        expect(component.clientClassesArray.length).toBe(2)
        expect(component.getClientClassSetControl(0).value).toEqual(['foo'])
        expect(component.getClientClassSetControl(1).value).toEqual(['bar'])
        expect(component.bootFieldsArray.length).toBe(2)
        expect(component.getBootFieldsGroup(0).get('nextServer').value).toEqual('192.2.2.1')
        expect(component.getBootFieldsGroup(1).get('nextServer').value).toEqual('192.2.2.1')
        expect(component.getBootFieldsGroup(0).get('serverHostname').value).toEqual('myserver.example.org')
        expect(component.getBootFieldsGroup(1).get('serverHostname').value).toEqual('myserver.example.org')
        expect(component.getBootFieldsGroup(0).get('bootFileName').value).toEqual('/tmp/boot1')
        expect(component.getBootFieldsGroup(1).get('bootFileName').value).toEqual('/tmp/boot2')

        const okResp: any = {
            status: 200,
        }
        spyOn(dhcpApi, 'updateHostSubmit').and.returnValue(of(okResp))
        spyOn(component.formSubmit, 'emit')
        spyOn(messageService, 'add')
        component.onSubmit()
        tick()
        fixture.detectChanges()

        let host = {
            id: 123,
            subnetId: 1,
            hostIdentifiers: [
                {
                    idType: 'hw-address',
                    idHexValue: '01:02:03:04:05:06',
                },
            ],
            addressReservations: [
                {
                    address: '192.0.2.4/32',
                },
            ],
            prefixReservations: [],
            hostname: 'foo.example.org',
            localHosts: [
                {
                    daemonId: 1,
                    dataSource: 'api',
                    nextServer: '192.2.2.1',
                    serverHostname: 'myserver.example.org',
                    bootFileName: '/tmp/boot1',
                    clientClasses: ['foo'],
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
                {
                    daemonId: 2,
                    dataSource: 'api',
                    nextServer: '192.2.2.1',
                    serverHostname: 'myserver.example.org',
                    bootFileName: '/tmp/boot2',
                    clientClasses: ['bar'],
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
            ],
        }
        expect(dhcpApi.updateHostSubmit).toHaveBeenCalledWith(component.hostId, component.form.transactionId, host)
        expect(component.formSubmit.emit).toHaveBeenCalled()
        expect(messageService.add).toHaveBeenCalled()
    }))

    it('should revert host changes', fakeAsync(() => {
        component.hostId = 123

        let beginResponse = cannedResponseBegin
        beginResponse.host = {
            id: 123,
            subnetId: 1,
            subnetPrefix: '192.0.2.0/24',
            hostIdentifiers: [
                {
                    idType: 'hw-address',
                    idHexValue: '01:02:03:04:05:06',
                },
            ],
            addressReservations: [
                {
                    address: '192.0.2.4',
                },
            ],
            prefixReservations: [],
            hostname: 'foo.example.org',
            localHosts: [
                {
                    daemonId: 1,
                    dataSource: 'api',
                    options: [],
                    optionsHash: '',
                },
            ],
        }
        spyOn(dhcpApi, 'updateHostBegin').and.returnValue(of(beginResponse))
        component.ngOnInit()
        tick()
        fixture.detectChanges()

        expect(component.ipGroups.length).toBe(1)
        expect(component.ipGroups.get('0.inputIPv4').value).toBe('192.0.2.4')
        expect(component.formGroup.get('hostname').value).toBe('foo.example.org')
        expect(component.formGroup.valid).toBeTrue()

        // Apply some changes.
        component.ipGroups.get('0.inputIPv4').setValue('192.0.')
        component.formGroup.get('hostname').setValue('xyz')
        fixture.detectChanges()
        expect(component.formGroup.valid).toBeFalse()

        // Revert the changes.
        component.onRevert()
        fixture.detectChanges()

        // Ensure that the changes have been reverted.
        expect(component.ipGroups.length).toBe(1)
        expect(component.ipGroups.get('0.inputIPv4').value).toBe('192.0.2.4')
        expect(component.formGroup.get('hostname').value).toBe('foo.example.org')
        expect(component.formGroup.valid).toBeTrue()
    }))

    it('should emit cancel event', () => {
        spyOn(component.formCancel, 'emit')
        component.onCancel()
        expect(component.formCancel.emit).toHaveBeenCalled()
    })
})
