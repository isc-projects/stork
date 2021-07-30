import { ComponentFixture, TestBed, waitForAsync } from '@angular/core/testing'

import { SettingsMenuComponent } from './settings-menu.component'

describe('SettingsMenuComponent', () => {
    let component: SettingsMenuComponent
    let fixture: ComponentFixture<SettingsMenuComponent>

    beforeEach(
        waitForAsync(() => {
            TestBed.configureTestingModule({
                declarations: [SettingsMenuComponent],
            }).compileComponents()
        })
    )

    beforeEach(() => {
        fixture = TestBed.createComponent(SettingsMenuComponent)
        component = fixture.componentInstance
        fixture.detectChanges()
    })

    it('should create', () => {
        expect(component).toBeTruthy()
    })
})
