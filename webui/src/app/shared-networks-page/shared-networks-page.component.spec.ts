import { async, ComponentFixture, TestBed } from '@angular/core/testing'

import { SharedNetworksPageComponent } from './shared-networks-page.component'

describe('SharedNetworksPageComponent', () => {
    let component: SharedNetworksPageComponent
    let fixture: ComponentFixture<SharedNetworksPageComponent>

    beforeEach(async(() => {
        TestBed.configureTestingModule({
            declarations: [SharedNetworksPageComponent],
        }).compileComponents()
    }))

    beforeEach(() => {
        fixture = TestBed.createComponent(SharedNetworksPageComponent)
        component = fixture.componentInstance
        fixture.detectChanges()
    })

    it('should create', () => {
        expect(component).toBeTruthy()
    })
})
