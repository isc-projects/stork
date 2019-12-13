import { async, ComponentFixture, TestBed } from '@angular/core/testing'

import { KeaServiceTabComponent } from './kea-service-tab.component'

describe('KeaServiceTabComponent', () => {
    let component: KeaServiceTabComponent
    let fixture: ComponentFixture<KeaServiceTabComponent>

    beforeEach(async(() => {
        TestBed.configureTestingModule({
            declarations: [KeaServiceTabComponent],
        }).compileComponents()
    }))

    beforeEach(() => {
        fixture = TestBed.createComponent(KeaServiceTabComponent)
        component = fixture.componentInstance
        fixture.detectChanges()
    })

    it('should create', () => {
        expect(component).toBeTruthy()
    })
})
