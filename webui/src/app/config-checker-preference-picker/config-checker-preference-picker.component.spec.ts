import { ComponentFixture, TestBed } from '@angular/core/testing'
import { NoopAnimationsModule } from '@angular/platform-browser/animations'
import { ButtonModule } from 'primeng/button'
import { ChipModule } from 'primeng/chip'
import { OverlayPanelModule } from 'primeng/overlaypanel'
import { TableModule } from 'primeng/table'
import { HelpTipComponent } from '../help-tip/help-tip.component'
import { ConfigCheckerPreferencePickerComponent } from './config-checker-preference-picker.component'

describe('ConfigCheckerPreferencePickerComponent', () => {
    let component: ConfigCheckerPreferencePickerComponent
    let fixture: ComponentFixture<ConfigCheckerPreferencePickerComponent>

    beforeEach(async () => {
        await TestBed.configureTestingModule({
            imports: [TableModule, ChipModule, OverlayPanelModule, NoopAnimationsModule, ButtonModule],
            declarations: [HelpTipComponent],
        }).compileComponents()
    })

    beforeEach(() => {
        fixture = TestBed.createComponent(ConfigCheckerPreferencePickerComponent)
        component = fixture.componentInstance
        fixture.detectChanges()
    })

    it('should create', () => {
        expect(component).toBeTruthy()
    })

    it('should handle loading state', () => {
        fail("not implemented")
    })

    it('should handle an empty state and display no buttons', () => {
        fail("not implemented")
    })

    it('should display full layout by default', () => {
        fail("not implemented")
    })

    it('should display only necessary columns in a minimal layout', () => {
        fail("not implemented")
    })

    it('should correctly cycle the checker state', () => {
        fail("not implemented")
    })

    it('should display the checker description', () => {
        fail("not implemented")
    })

    it('should display the checker selectors', () => {
        fail("not implemented")
    })

    it('should display the checker triggers', () => {
        fail("not implemented")
    })

    it('should activate the submit button only if any changes were provided', () => {
        fail("not implemented")
    })

    it('should detect reverting changes', () => {
        fail("not implemented")
    })

    it('should display the checker state using a color and a proper checkbox state', () => {
        fail("not implemented")
    })

    it('should display inherit state with a global enabled status', () => {
        fail("not implemented")
    })

    it('should handle submitting and set the loading state', () => {
        fail("not implemented")
    })

    it('should handle the reset button', () => {
        fail("not implemented")
    })

    it('should present the help button', () => {
        fail("not implemented")
    })
})