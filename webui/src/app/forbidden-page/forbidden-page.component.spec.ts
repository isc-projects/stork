import { async, ComponentFixture, TestBed } from '@angular/core/testing'

import { ForbiddenPageComponent } from './forbidden-page.component'

describe('ForbiddenPageComponent', () => {
    let component: ForbiddenPageComponent
    let fixture: ComponentFixture<ForbiddenPageComponent>

    beforeEach(async(() => {
        TestBed.configureTestingModule({
            declarations: [ForbiddenPageComponent],
        }).compileComponents()
    }))

    beforeEach(() => {
        fixture = TestBed.createComponent(ForbiddenPageComponent)
        component = fixture.componentInstance
        fixture.detectChanges()
    })

    it('should create', () => {
        expect(component).toBeTruthy()
    })
})
