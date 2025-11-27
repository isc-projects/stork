import { ComponentFixture, TestBed, waitForAsync } from '@angular/core/testing'

import { ForbiddenPageComponent } from './forbidden-page.component'
import { provideNoopAnimations } from '@angular/platform-browser/animations'
import { provideRouter } from '@angular/router'

describe('ForbiddenPageComponent', () => {
    let component: ForbiddenPageComponent
    let fixture: ComponentFixture<ForbiddenPageComponent>

    beforeEach(waitForAsync(() => {
        TestBed.configureTestingModule({
            providers: [provideNoopAnimations(), provideRouter([])],
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
