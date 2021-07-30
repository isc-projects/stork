import { ComponentFixture, TestBed, waitForAsync } from '@angular/core/testing'

import { ForbiddenPageComponent } from './forbidden-page.component'

describe('ForbiddenPageComponent', () => {
    let component: ForbiddenPageComponent
    let fixture: ComponentFixture<ForbiddenPageComponent>

    beforeEach(
        waitForAsync(() => {
            TestBed.configureTestingModule({
                declarations: [ForbiddenPageComponent],
            }).compileComponents()
        })
    )

    beforeEach(() => {
        fixture = TestBed.createComponent(ForbiddenPageComponent)
        component = fixture.componentInstance
        fixture.detectChanges()
    })

    it('should create', () => {
        expect(component).toBeTruthy()
    })
})
