import { async, ComponentFixture, TestBed } from '@angular/core/testing'

import { PasswordChangePageComponent } from './password-change-page.component'

describe('PasswordChangePageComponent', () => {
    let component: PasswordChangePageComponent
    let fixture: ComponentFixture<PasswordChangePageComponent>

    beforeEach(async(() => {
        TestBed.configureTestingModule({
            declarations: [PasswordChangePageComponent],
        }).compileComponents()
    }))

    beforeEach(() => {
        fixture = TestBed.createComponent(PasswordChangePageComponent)
        component = fixture.componentInstance
        fixture.detectChanges()
    })

    it('should create', () => {
        expect(component).toBeTruthy()
    })
})
