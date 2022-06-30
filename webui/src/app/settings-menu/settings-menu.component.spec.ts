import { ComponentFixture, TestBed, waitForAsync } from '@angular/core/testing'
import { NoopAnimationsModule } from '@angular/platform-browser/animations'
import { ActivatedRoute, Router } from '@angular/router'
import { RouterTestingModule } from '@angular/router/testing'
import { MenuModule } from 'primeng/menu'

import { SettingsMenuComponent } from './settings-menu.component'

describe('SettingsMenuComponent', () => {
    let component: SettingsMenuComponent
    let fixture: ComponentFixture<SettingsMenuComponent>

    beforeEach(waitForAsync(() => {
        TestBed.configureTestingModule({
            imports: [MenuModule, NoopAnimationsModule, RouterTestingModule],
            declarations: [SettingsMenuComponent],
            providers: [
                {
                    provide: ActivatedRoute,
                    useValue: {},
                },
            ],
        }).compileComponents()
    }))

    beforeEach(() => {
        fixture = TestBed.createComponent(SettingsMenuComponent)
        component = fixture.componentInstance
        fixture.detectChanges()
    })

    it('should create', () => {
        expect(component).toBeTruthy()
    })
})
