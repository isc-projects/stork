import { ComponentFixture, TestBed } from '@angular/core/testing'
import {
    UntypedFormArray,
    UntypedFormBuilder,
    UntypedFormGroup,
    FormsModule,
    ReactiveFormsModule,
} from '@angular/forms'
import { By } from '@angular/platform-browser'
import { NoopAnimationsModule } from '@angular/platform-browser/animations'
import { CheckboxModule } from 'primeng/checkbox'
import { DropdownModule } from 'primeng/dropdown'
import { InputNumberModule } from 'primeng/inputnumber'
import { OverlayPanelModule } from 'primeng/overlaypanel'
import { ToggleButtonModule } from 'primeng/togglebutton'
import { SplitButtonModule } from 'primeng/splitbutton'
import { createDefaultDhcpOptionFormGroup } from '../forms/dhcp-option-form'
import { DhcpOptionFormComponent } from './dhcp-option-form.component'
import { DhcpOptionSetFormComponent } from '../dhcp-option-set-form/dhcp-option-set-form.component'
import { HelpTipComponent } from '../help-tip/help-tip.component'
import { IPType } from '../iptype'
import { DhcpOptionFieldFormGroup, DhcpOptionFieldType } from '../forms/dhcp-option-field'
import { DividerModule } from 'primeng/divider'

describe('DhcpOptionFormComponent', () => {
    let component: DhcpOptionFormComponent
    let fixture: ComponentFixture<DhcpOptionFormComponent>

    beforeEach(async () => {
        await TestBed.configureTestingModule({
            providers: [UntypedFormBuilder],
            imports: [
                CheckboxModule,
                DropdownModule,
                FormsModule,
                InputNumberModule,
                NoopAnimationsModule,
                OverlayPanelModule,
                ReactiveFormsModule,
                SplitButtonModule,
                ToggleButtonModule,
                DividerModule,
            ],
            declarations: [DhcpOptionFormComponent, DhcpOptionSetFormComponent, HelpTipComponent],
        }).compileComponents()
    })

    beforeEach(() => {
        fixture = TestBed.createComponent(DhcpOptionFormComponent)
        component = fixture.componentInstance
        // Our component needs a form group instance to be initialized.
        component.formGroup = createDefaultDhcpOptionFormGroup(IPType.IPv4)
        fixture.detectChanges()
    })

    it('should create', () => {
        expect(component).toBeTruthy()
        expect(component.formGroup.contains('optionCode')).toBeTrue()
        expect(component.formGroup.contains('optionFields')).toBeTrue()
        expect(component.formGroup.contains('suboptions')).toBeTrue()
    })

    it('should display DHCPv4 options selection', () => {
        // By default, the component should display a dropdown with option codes.
        const dropdownEl = fixture.debugElement.query(By.css('p-dropdown'))
        expect(dropdownEl).toBeTruthy()

        // The dropdown should include a placeholder informing about its purpose.
        const inputEl = dropdownEl.query(By.css('input'))
        expect(inputEl).toBeTruthy()
        expect(inputEl.nativeElement.placeholder).toBe('Select or Type Option Code')

        // By default, we should display DHCPv4 options. Let's get one from the list
        // and ensure it is the DHCPv4 option.
        const nameServer = dropdownEl.componentInstance.options.find((opt) => opt.value === 5)
        expect(nameServer).toBeTruthy()
        expect(nameServer.label).toBe('(5) Name Server')
    })

    it('should display DHCPv6 options selection', () => {
        // Configure the component to display DHCPv6 options.
        component.v6 = true
        fixture.detectChanges()

        // There should be a dropdown.
        const dropdownEl = fixture.debugElement.query(By.css('p-dropdown'))
        expect(dropdownEl).toBeTruthy()

        // The dropdown should include a placeholder informing about its purpose.
        const inputEl = dropdownEl.query(By.css('input'))
        expect(inputEl).toBeTruthy()
        expect(inputEl.nativeElement.placeholder).toBe('Select or Type Option Code')

        // This time the list should comprise DHCPv6 options.
        const nameServer = dropdownEl.componentInstance.options.find((opt) => opt.value === 31)
        expect(nameServer).toBeTruthy()
        expect(nameServer.label).toBe('(31) OPTION_SNTP_SERVERS')
    })

    it('should emit an event to delete an option', () => {
        component.formIndex = 7
        spyOn(component.optionDelete, 'emit')
        component.deleteOption()
        expect(component.optionDelete.emit).toHaveBeenCalledWith(7)
    })

    it('should add default option field when clicked on add payload', () => {
        // Add Payload button should exist.
        const addPayloadBtn = fixture.debugElement.query(By.css('p-splitButton'))
        expect(addPayloadBtn).toBeTruthy()

        // Initially, there should be a tag indicating that the option is empty.
        const emptyTagEl = fixture.debugElement.query(By.css('p-tag'))
        expect(emptyTagEl).toBeTruthy()
        expect(emptyTagEl.attributes.value).toBe('Empty Option')

        // Click the Add Payload button.
        addPayloadBtn.componentInstance.onClick.emit(new Event('click'))
        fixture.detectChanges()

        // It should result in adding a default option field.
        expect(component.optionFields.length).toBe(1)
        expect(component.optionFields.at(0).get('control').value).toBe('')

        // It should be the binary field.
        expect(fixture.debugElement.query(By.css('textarea'))).toBeTruthy()
        expect(fixture.debugElement.query(By.css('p-tag'))).toBeFalsy()

        // It should contain a help-tip for the binary option field.
        expect(fixture.debugElement.query(By.css('[title="Help for binary Option Field"]'))).toBeTruthy()

        // Find the last button. It should delete the option field.
        const allBtns = fixture.debugElement.queryAll(By.css('button'))
        const deleteFieldBtn = allBtns[allBtns.length - 1]
        expect(deleteFieldBtn).toBeTruthy()

        // Click the button to delete the field.
        deleteFieldBtn.nativeElement.dispatchEvent(new Event('click'))
        fixture.detectChanges()

        // There should be no option fields and the tag should be back.
        expect(component.optionFields.length).toBe(0)
        expect(fixture.debugElement.query(By.css('p-tag'))).toBeTruthy()
    })

    it('should remember last added option field', () => {
        // Simulate adding the uint8 option field.
        const uint8Field = component.fieldTypes.find((field) => field.label === 'uint8')
        expect(uint8Field).toBeTruthy()
        uint8Field.command({})
        fixture.detectChanges()

        // The Add Payload button should add another uint8 option field.
        const addPayloadBtn = fixture.debugElement.query(By.css('p-splitButton'))
        expect(addPayloadBtn).toBeTruthy()
        addPayloadBtn.componentInstance.onClick.emit(new Event('click'))
        fixture.detectChanges()

        // We should have two uint8 option field.
        expect(component.optionFields.length).toBe(2)

        // Validate the fields.
        const inputEls = fixture.debugElement.queryAll(By.css('p-inputNumber'))
        expect(inputEls.length).toBe(2)
        for (let i = 0; i < 2; i++) {
            expect(inputEls[i].attributes.hasOwnProperty('max')).toBeTrue()
            expect(inputEls[i].attributes.hasOwnProperty('min')).toBeTrue()
            expect(inputEls[i].attributes.max).toBe('255')
            expect(inputEls[i].attributes.min).toBe('0')
        }
    })

    it('should add many fields', () => {
        // Iterate over the option field types and simulate adding them.
        for (let field of component.fieldTypes) {
            expect(field.command).toBeTruthy()
            field.command({})
        }
        fixture.detectChanges()
        expect(component.optionFields.length).toBe(14)

        // Find the container holding all added option fields.
        const containerEl = fixture.debugElement.query(By.css('[formArrayName="optionFields"]'))
        expect(containerEl).toBeTruthy()

        // Verify that the correct types of option fields have been added.
        const textFieldEls = containerEl.queryAll(By.css('textarea'))
        expect(textFieldEls.length).toBe(1)

        const inputFieldEls = containerEl.queryAll(By.css('input + label'))
        // string, ipv4-addr, ipv6-addr, ipv6-prefix, fqdn
        expect(inputFieldEls.length).toBe(5)

        const boolFieldEls = containerEl.queryAll(By.css('p-toggleButton'))
        expect(boolFieldEls.length).toBe(2)

        let numberFieldEls = containerEl.queryAll(By.css('p-inputNumber'))
        expect(numberFieldEls.length).toBe(9)

        const deleteBtns = containerEl.queryAll(By.css('button'))
        expect(deleteBtns.length).toBe(14)

        // Delete uint32 option field.
        deleteBtns[5].nativeElement.dispatchEvent(new Event('click'))
        fixture.detectChanges()

        // Make sure it is gone.
        numberFieldEls = containerEl.queryAll(By.css('p-inputNumber'))
        expect(numberFieldEls.length).toBe(8)
        expect(component.optionFields.length).toBe(13)
    })

    it('should add and delete suboption form', () => {
        // Add a suboption.
        component.addSuboption()
        fixture.detectChanges()

        expect(fixture.debugElement.query(By.css('p-tag'))).toBeTruthy()

        // Make sure that the component with suboptions has been added.
        const suboptionEl = fixture.debugElement.query(By.css('app-dhcp-option-form'))
        expect(suboptionEl).toBeTruthy()

        // Validate the suboption view.
        const labelEl = suboptionEl.query(By.css('label'))
        expect(labelEl).toBeTruthy()
        expect(labelEl.nativeElement.innerText).toBe('Type Suboption Code')
        expect(suboptionEl.query(By.css('input'))).toBeTruthy()
        const addPayloadBtn = suboptionEl.query(By.css('p-splitButton'))
        expect(addPayloadBtn).toBeTruthy()
        const deleteBtn = suboptionEl.query(By.css('button .pi-times'))
        expect(deleteBtn).toBeTruthy()
        expect(suboptionEl.query(By.css('p-toggleButton'))).toBeFalsy()

        // Simulate clicking the Add Payload button within the suboption.
        addPayloadBtn.componentInstance.onClick.emit(new Event('click'))
        fixture.detectChanges()
        expect(component.suboptions.length).toBe(1)
        expect((component.suboptions.at(0) as UntypedFormGroup).contains('optionFields')).toBeTrue()
        expect(((component.suboptions.at(0) as UntypedFormGroup).get('optionFields') as UntypedFormArray).length).toBe(
            1
        )
        expect(suboptionEl.query(By.css('textarea'))).toBeTruthy()

        // Simulate deleting the suboption.
        deleteBtn.parent.nativeElement.dispatchEvent(new Event('click'))
        fixture.detectChanges()

        // It should be gone.
        expect(fixture.debugElement.query(By.css('app-dhcp-option-form'))).toBeFalsy()
        expect(component.suboptions.length).toBe(0)
    })

    it('should require a valid DHCPv4 option code', () => {
        component.formGroup.get('optionCode').setValue(7)
        expect(component.formGroup.valid).toBeTrue()

        // Option code must be specified.
        component.formGroup.get('optionCode').setValue(null)
        expect(component.formGroup.valid).toBeFalse()

        // Out of range value is invalid.
        component.formGroup.get('optionCode').setValue(256)
        expect(component.formGroup.valid).toBeFalse()

        // String value is invalid.
        component.formGroup.get('optionCode').setValue('abc')
        expect(component.formGroup.valid).toBeFalse()
    })

    it('should require a valid DHCPv6 option code', () => {
        component.v6 = true
        component.formGroup = createDefaultDhcpOptionFormGroup(IPType.IPv6)
        fixture.detectChanges()

        component.formGroup.get('optionCode').setValue(7)
        expect(component.formGroup.valid).toBeTrue()

        // Option code must be specified.
        component.formGroup.get('optionCode').setValue(null)
        expect(component.formGroup.valid).toBeFalse()

        // The value that is out of range for DHCPv4 should be fine for
        // DHCPv6 case.
        component.formGroup.get('optionCode').setValue(256)
        expect(component.formGroup.valid).toBeTrue()

        // Out of range value is invalid.
        component.formGroup.get('optionCode').setValue(65537)
        expect(component.formGroup.valid).toBeFalse()

        // String value is invalid.
        component.formGroup.get('optionCode').setValue('abc')
        expect(component.formGroup.valid).toBeFalse()
    })

    it('should require a valid ipv4 address', () => {
        component.formGroup.get('optionCode').setValue(5)
        expect(component.formGroup.valid).toBeTrue()

        component.addIPv4AddressField()
        component.optionFields.at(0).get('control').setValue('192x56x45x1')
        expect(component.formGroup.valid).toBeFalse()

        component.optionFields.at(0).get('control').setValue('192.56.45.1')
        expect(component.formGroup.valid).toBeTrue()
    })

    it('should require a valid ipv6 address', () => {
        component.formGroup.get('optionCode').setValue(5)
        expect(component.formGroup.valid).toBeTrue()

        component.addIPv6AddressField()
        component.optionFields.at(0).get('control').setValue('2001')
        expect(component.formGroup.valid).toBeFalse()

        component.optionFields.at(0).get('control').setValue('2001::')
        expect(component.formGroup.valid).toBeTrue()
    })

    it('should require a valid ipv6 prefix', () => {
        component.formGroup.get('optionCode').setValue(5)
        expect(component.formGroup.valid).toBeTrue()

        component.addIPv6PrefixField()
        component.optionFields.at(0).get('prefix').setValue('3000')
        component.optionFields.at(0).get('prefixLength').setValue('24')
        expect(component.formGroup.valid).toBeFalse()

        component.optionFields.at(0).get('prefix').setValue('3000::')
        component.optionFields.at(0).get('prefixLength').setValue(null)
        expect(component.formGroup.valid).toBeFalse()

        component.optionFields.at(0).get('prefixLength').setValue(24)
        expect(component.formGroup.valid).toBeTrue()
    })

    it('should require psid', () => {
        component.formGroup.get('optionCode').setValue(5)
        expect(component.formGroup.valid).toBeTrue()

        component.addPsidField()
        component.optionFields.at(0).get('psid').setValue(null)
        component.optionFields.at(0).get('psidLength').setValue(10)
        expect(component.formGroup.valid).toBeFalse()

        component.optionFields.at(0).get('psid').setValue(12)
        expect(component.formGroup.valid).toBeTrue()
    })

    it('should require a valid fqdn', () => {
        component.formGroup.get('optionCode').setValue(5)
        expect(component.formGroup.valid).toBeTrue()

        component.addFqdnField()

        const input = component.optionFields.at(0).get('control')
        const toggleButton = component.optionFields.at(0).get('isPartialFqdn')

        // By default, the FQDN field is set to be full.
        const isPartialFQDN = toggleButton.getRawValue()
        expect(isPartialFQDN).toBeFalse()

        // Invalid FQDN.
        input.setValue('fqdn..invalid.')
        expect(component.formGroup.valid).toBeFalse()

        // Null FQDN.
        input.setValue(null)
        expect(component.formGroup.valid).toBeFalse()

        // Partial FQDN.
        input.setValue('foo.bar-baz')
        expect(component.formGroup.valid).toBeFalse()

        // Full FQDN.
        input.setValue('fqdn.valid.')
        expect(component.formGroup.valid).toBeTrue()

        // Switch to partial FQDN.
        toggleButton.setValue(true)
        component.togglePartialFqdn({ checked: true }, 0)
        fixture.detectChanges()

        // Invalid FQDN.
        input.setValue('fqdn..invalid.')
        expect(component.formGroup.valid).toBeFalse()

        // Null FQDN.
        input.setValue(null)
        expect(component.formGroup.valid).toBeFalse()

        // Partial FQDN.
        input.setValue('foo.bar-baz')
        expect(component.formGroup.valid).toBeTrue()

        // Full FQDN.
        input.setValue('fqdn.valid.')
        expect(component.formGroup.valid).toBeFalse()

        // Switch back to full FQDN.
        toggleButton.setValue(false)
        component.togglePartialFqdn({ checked: false }, 0)
        fixture.detectChanges()

        // Invalid FQDN.
        input.setValue('fqdn..invalid.')
        expect(component.formGroup.valid).toBeFalse()

        // Null FQDN.
        input.setValue(null)
        expect(component.formGroup.valid).toBeFalse()

        // Partial FQDN.
        input.setValue('foo.bar-baz')
        expect(component.formGroup.valid).toBeFalse()

        // Full FQDN.
        input.setValue('fqdn.valid.')
        expect(component.formGroup.valid).toBeTrue()
    })

    it('should only add suboptions to top-level options and first level suboptions', () => {
        component.nestLevel = 0
        component.ngOnInit()
        expect(component.fieldTypes.find((field) => field.label === 'suboption')).toBeTruthy()

        component.nestLevel = 1
        component.ngOnInit()
        expect(component.fieldTypes.find((field) => field.label === 'suboption')).toBeTruthy()

        component.nestLevel = 2
        component.ngOnInit()
        expect(component.fieldTypes.find((field) => field.label === 'suboption')).toBeFalsy()
    })

    it('should set the corresponding form layout for simple option type ', () => {
        component.formGroup = createDefaultDhcpOptionFormGroup(IPType.IPv4)
        component.formGroup.get('optionCode').setValue(3)
        fixture.detectChanges()

        const event = {
            value: 3,
        }
        component.onOptionCodeChange(event)
        expect(component.optionFields.length).toBe(1)

        let field = component.optionFields.at(0) as DhcpOptionFieldFormGroup
        expect(field).toBeTruthy()
        expect(field.data.fieldType).toBe(DhcpOptionFieldType.IPv4Address)

        event.value = 17
        component.onOptionCodeChange(event)
        expect(component.optionFields.length).toBe(1)

        field = component.optionFields.at(0) as DhcpOptionFieldFormGroup
        expect(field).toBeTruthy()
        expect(field.data.fieldType).toBe(DhcpOptionFieldType.String)
    })

    it('should set the corresponding form layout for a record suboption type', () => {
        component.v6 = true
        component.nestLevel = 1
        component.optionSpace = 's46-cont-mape-options'
        component.formGroup = createDefaultDhcpOptionFormGroup(IPType.IPv6)
        component.formGroup.get('optionCode').setValue(89)
        fixture.detectChanges()

        const event = {
            value: 89,
        }
        component.onOptionCodeChange(event)
        expect(component.optionFields.length).toBe(5)

        const expectedFieldTypes = [
            DhcpOptionFieldType.Uint8,
            DhcpOptionFieldType.Uint8,
            DhcpOptionFieldType.Uint8,
            DhcpOptionFieldType.IPv4Address,
            DhcpOptionFieldType.IPv6Prefix,
        ]
        for (let i = 0; i < expectedFieldTypes.length; i++) {
            const field = component.optionFields.at(i) as DhcpOptionFieldFormGroup
            expect(field).toBeTruthy()
            expect(field.data.fieldType).toBe(expectedFieldTypes[i])
        }
    })

    it('should set the default form layout for an option without definition', () => {
        component.formGroup = createDefaultDhcpOptionFormGroup(IPType.IPv4)
        component.formGroup.get('optionCode').setValue(254)
        fixture.detectChanges()

        const event = {
            value: 254,
        }
        component.onOptionCodeChange(event)
        expect(component.optionFields.length).toBe(0)
    })

    it('should display help tip for an option with the definition', () => {
        component.formGroup = createDefaultDhcpOptionFormGroup(IPType.IPv4)
        let helptip = fixture.debugElement.query(By.css('app-help-tip'))
        expect(helptip).toBeFalsy()

        component.formGroup.get('optionCode').setValue(85)
        const event = {
            value: 85,
        }
        component.onOptionCodeChange(event)
        fixture.detectChanges()

        helptip = fixture.debugElement.query(By.css('app-help-tip'))
        expect(helptip).toBeTruthy()
    })

    it('should return standard DHCPv4 option definition codes for a known option space', () => {
        component.optionDef = {
            code: 82,
            name: 'dhcp-agent-options',
            space: 'dhcp4',
            optionType: 'empty',
            array: false,
            encapsulate: 'dhcp-agent-options-space',
        }
        const codes = component.getStandardDhcpOptionDefCodes()
        expect(codes.length).toBe(20)

        const expectedCodes = [1, 2, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 151, 152]
        expect(codes).toEqual(expectedCodes)
    })

    it('should return standard DHCPv6 option definition codes for a known option space', () => {
        component.optionDef = {
            code: 89,
            name: 's46-rule',
            space: 's46-cont-mape-options',
            optionType: 'record',
            array: false,
            encapsulate: 's46-rule-options',
            recordTypes: ['uint8', 'uint8', 'uint8', 'ipv4-address', 'ipv6-prefix'],
        }
        component.v6 = true
        const codes = component.getStandardDhcpOptionDefCodes()
        expect(codes.length).toBe(1)
        expect(codes[0]).toBe(93)
    })

    it('should return an empty array of option definition codes when option definition is unavailable', () => {
        const codes = component.getStandardDhcpOptionDefCodes()
        expect(codes).toBeTruthy()
        expect(codes.length).toBe(0)
    })
})
