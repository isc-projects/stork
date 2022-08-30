import { ComponentFixture, TestBed } from '@angular/core/testing'
import { By } from '@angular/platform-browser'
import { ConfigCheckerPreferencePickerComponent } from '../config-checker-preference-picker/config-checker-preference-picker.component'

import { ConfigCheckerPreferencePageComponent } from './config-checker-preference-page.component'

describe('ConfigCheckerPreferencePageComponent', () => {
    let component: ConfigCheckerPreferencePageComponent
    let fixture: ComponentFixture<ConfigCheckerPreferencePageComponent>

    beforeEach(async () => {
        await TestBed.configureTestingModule({
            declarations: [ConfigCheckerPreferencePageComponent],
        }).compileComponents()
    })

    beforeEach(() => {
        fixture = TestBed.createComponent(ConfigCheckerPreferencePageComponent)
        component = fixture.componentInstance
        fixture.detectChanges()
    })

    it('should create', () => {
        expect(component).toBeTruthy()
    })

    it('should display the config review checkers panel with the full layout', () => {
        const element = fixture.debugElement.query(By.directive(ConfigCheckerPreferencePickerComponent))
        expect(element).toBeDefined()
        const picker = element.componentInstance as ConfigCheckerPreferencePickerComponent

        expect(picker.allowInheritState).toBeFalse()
        expect(picker.minimal).toBeFalse()
    })
})
