import { async, ComponentFixture, TestBed } from '@angular/core/testing'

import { SubnetsPageComponent } from './subnets-page.component'

describe('SubnetsPageComponent', () => {
    let component: SubnetsPageComponent
    let fixture: ComponentFixture<SubnetsPageComponent>

    beforeEach(async(() => {
        TestBed.configureTestingModule({
            declarations: [SubnetsPageComponent],
        }).compileComponents()
    }))

    beforeEach(() => {
        fixture = TestBed.createComponent(SubnetsPageComponent)
        component = fixture.componentInstance
        fixture.detectChanges()
    })

    it('should create', () => {
        expect(component).toBeTruthy()
    })
})
