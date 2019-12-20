import { async, ComponentFixture, TestBed } from '@angular/core/testing'

import { KeaAppTabComponent } from './kea-app-tab.component'

describe('KeaAppTabComponent', () => {
    let component: KeaAppTabComponent
    let fixture: ComponentFixture<KeaAppTabComponent>

    beforeEach(async(() => {
        TestBed.configureTestingModule({
            declarations: [KeaAppTabComponent],
        }).compileComponents()
    }))

    beforeEach(() => {
        fixture = TestBed.createComponent(KeaAppTabComponent)
        component = fixture.componentInstance
        fixture.detectChanges()
    })

    it('should create', () => {
        expect(component).toBeTruthy()
    })
})
