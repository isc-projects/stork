import { async, ComponentFixture, TestBed } from '@angular/core/testing'

import { LeaseSearchPageComponent } from './lease-search-page.component'

describe('LeaseSearchPageComponent', () => {
    let component: LeaseSearchPageComponent
    let fixture: ComponentFixture<LeaseSearchPageComponent>

    beforeEach(async(() => {
        TestBed.configureTestingModule({
            declarations: [LeaseSearchPageComponent],
        }).compileComponents()
    }))

    beforeEach(() => {
        fixture = TestBed.createComponent(LeaseSearchPageComponent)
        component = fixture.componentInstance
        fixture.detectChanges()
    })

    it('should create', () => {
        expect(component).toBeTruthy()
    })
})
