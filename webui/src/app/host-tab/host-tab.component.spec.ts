import { async, ComponentFixture, TestBed } from '@angular/core/testing'

import { HostTabComponent } from './host-tab.component'

describe('HostTabComponent', () => {
    let component: HostTabComponent
    let fixture: ComponentFixture<HostTabComponent>

    beforeEach(async(() => {
        TestBed.configureTestingModule({
            declarations: [HostTabComponent],
        }).compileComponents()
    }))

    beforeEach(() => {
        fixture = TestBed.createComponent(HostTabComponent)
        component = fixture.componentInstance
        fixture.detectChanges()
    })

    it('should create', () => {
        expect(component).toBeTruthy()
    })
})
