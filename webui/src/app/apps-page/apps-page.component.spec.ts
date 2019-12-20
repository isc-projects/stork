import { async, ComponentFixture, TestBed } from '@angular/core/testing'

import { AppsPageComponent } from './apps-page.component'

describe('AppsPageComponent', () => {
    let component: AppsPageComponent
    let fixture: ComponentFixture<AppsPageComponent>

    beforeEach(async(() => {
        TestBed.configureTestingModule({
            declarations: [AppsPageComponent],
        }).compileComponents()
    }))

    beforeEach(() => {
        fixture = TestBed.createComponent(AppsPageComponent)
        component = fixture.componentInstance
        fixture.detectChanges()
    })

    it('should create', () => {
        expect(component).toBeTruthy()
    })
})
