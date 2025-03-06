import { ComponentFixture, TestBed } from '@angular/core/testing'
import { FormsModule } from '@angular/forms'
import { By } from '@angular/platform-browser'
import { NoopAnimationsModule } from '@angular/platform-browser/animations'
import { ToggleButtonModule } from 'primeng/togglebutton'
import { IdentifierComponent } from './identifier.component'
import { ByteCharacterComponent } from '../byte-character/byte-character.component'
import { RouterTestingModule } from '@angular/router/testing'

describe('IdentifierComponent', () => {
    let component: IdentifierComponent
    let fixture: ComponentFixture<IdentifierComponent>

    beforeEach(async () => {
        await TestBed.configureTestingModule({
            imports: [FormsModule, NoopAnimationsModule, ToggleButtonModule, RouterTestingModule],
            declarations: [IdentifierComponent, ByteCharacterComponent],
        }).compileComponents()
    })

    beforeEach(() => {
        fixture = TestBed.createComponent(IdentifierComponent)
        component = fixture.componentInstance
        fixture.detectChanges()
    })

    it('should create', () => {
        expect(component).toBeTruthy()
    })

    it('should display convertible value as text', () => {
        // Use a correct flex-id value with all printable characters.
        component.label = 'flex-id'
        component.hexValue = '73:30:6d:45:56:61:4c:75:65'
        component.ngOnInit()
        fixture.detectChanges()

        let identifierEl = fixture.debugElement.query(By.css('div'))
        expect(identifierEl).toBeTruthy()

        // The flex-id value should be converted to string.
        expect(identifierEl.nativeElement.textContent).toContain('flex-id=(s0mEVaLue)')

        // There should be a button allowing to toggle between text and hex
        // identifier format.
        let toggleBtnEl = identifierEl.query(By.css('.p-togglebutton'))
        expect(toggleBtnEl).toBeTruthy()
        expect(toggleBtnEl.nativeElement.textContent).toContain('hex')

        // Click this button.
        toggleBtnEl.nativeElement.click()
        fixture.detectChanges()

        // The identifier should be now converted to hex.
        identifierEl = fixture.debugElement.query(By.css('div'))
        expect(identifierEl).toBeTruthy()
        expect(identifierEl.nativeElement.textContent).toContain('flex-id=(73:30:6d:45:56:61:4c:75:65)')

        // Click again.
        toggleBtnEl.nativeElement.click()
        fixture.detectChanges()

        // It should be back to string format.
        identifierEl = fixture.debugElement.query(By.css('div'))
        expect(identifierEl).toBeTruthy()
        expect(identifierEl.nativeElement.textContent).toContain('flex-id=(s0mEVaLue)')
    })

    it('should display not printable value as hex placeholder', () => {
        // Use the flex-id with unprintable characters.
        component.label = 'flex-id'
        component.hexValue = '01:02:03:04:05:06'
        component.ngOnInit()
        fixture.detectChanges()

        let identifierEl = fixture.debugElement.query(By.css('div'))
        expect(identifierEl).toBeTruthy()

        // The identifier should be displayed in the text format with the
        // hex placeholders for non-printable characters.
        expect(identifierEl.nativeElement.textContent).toContain('flex-id=(\\0x01\\0x02\\0x03\\0x04\\0x05\\0x06)')

        // There should be toggle button.
        let toggleBtnEl = identifierEl.query(By.css('.p-togglebutton'))
        expect(toggleBtnEl).toBeTruthy()
    })

    it('should display error when identifier is not a valid hex', () => {
        component.label = 'client-id'
        component.hexValue = 'invalid'
        component.ngOnInit()
        fixture.detectChanges()

        let identifierEl = fixture.debugElement.query(By.css('div'))
        expect(identifierEl).toBeTruthy()

        // If the specified value is not a valid string of hexadecimal digits
        // NaN value should be displayed.
        expect(identifierEl.nativeElement.textContent).toContain('client-id=(\\0xNaN\\0xNaN\\0xNaN\\0x0d)')

        component.hexFormat = true
        fixture.detectChanges() // Add fixture.detectChanges() to reflect the change
        expect(identifierEl.nativeElement.textContent).toContain('client-id=(in:va:li:d)')

        let toggleBtnEl = identifierEl.query(By.css('.p-togglebutton'))
        expect(toggleBtnEl).toBeTruthy()
    })

    it('should exclude label when it is not specified', () => {
        // Do not specify a label.
        component.hexValue = '01:02:03:04:05:06'
        component.hexFormat = true
        component.ngOnInit()
        fixture.detectChanges()

        let identifierEl = fixture.debugElement.query(By.css('div'))
        expect(identifierEl).toBeTruthy()

        // The label should not be displayed.
        expect(identifierEl.nativeElement.textContent.trim()).toContain('\\0x01\\0x02\\0x03\\0x04\\0x05\\0x06')
    })

    it('should respect default hex format setting', () => {
        // Specify a convertible value but disable default conversion to text.
        component.defaultHexFormat = true
        component.hexValue = '73:30'
        component.ngOnInit()
        fixture.detectChanges()

        let identifierEl = fixture.debugElement.query(By.css('div'))
        expect(identifierEl).toBeTruthy()

        // The identifier should be displayed in hex format.
        expect(identifierEl.nativeElement.textContent.trim()).toContain('73:30')
    })

    it('should parse identifier with spaces', () => {
        component.hexValue = '73 30 6d 45 56 61 4c 75 65'
        component.ngOnInit()
        fixture.detectChanges()

        let identifierEl = fixture.debugElement.query(By.css('div'))
        expect(identifierEl).toBeTruthy()
        expect(identifierEl.nativeElement.textContent.trim()).toContain('s0mEVaLue')
    })

    it('should parse identifier without separators', () => {
        component.hexValue = '73306d4556614c7565'
        component.ngOnInit()
        fixture.detectChanges()

        let identifierEl = fixture.debugElement.query(By.css('div'))
        expect(identifierEl).toBeTruthy()
        expect(identifierEl.nativeElement.textContent.trim()).toContain('s0mEVaLue')
    })

    it('should parse identifier even if it has an odd length', () => {
        component.hexValue = '73:3'
        component.ngOnInit()
        fixture.detectChanges()

        let identifierEl = fixture.debugElement.query(By.css('div'))
        expect(identifierEl).toBeTruthy()
        expect(identifierEl.nativeElement.textContent.trim()).toContain('s\\0x03')
    })

    it('should not parse blank identifier', () => {
        component.hexValue = '   '
        component.ngOnInit()
        fixture.detectChanges()

        let identifierEl = fixture.debugElement.query(By.css('div'))
        expect(identifierEl).toBeTruthy()
        expect(identifierEl.nativeElement.textContent.trim()).toContain('Empty identifier')
    })
})
