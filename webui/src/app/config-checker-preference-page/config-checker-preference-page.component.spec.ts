import { ComponentFixture, TestBed } from '@angular/core/testing'

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

    it('should have a general help button', () => {
        fail('not implemented')
    })

    it('should display the config review checkers panel with the full layout', () => {
        fail('not implemented')
    })
})
