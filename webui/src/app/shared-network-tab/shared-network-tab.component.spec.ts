import { ComponentFixture, TestBed } from '@angular/core/testing'

import { SharedNetworkTabComponent } from './shared-network-tab.component'

describe('SharedNetworkTabComponent', () => {
    let component: SharedNetworkTabComponent
    let fixture: ComponentFixture<SharedNetworkTabComponent>

    beforeEach(async () => {
        await TestBed.configureTestingModule({
            declarations: [SharedNetworkTabComponent],
        }).compileComponents()

        fixture = TestBed.createComponent(SharedNetworkTabComponent)
        component = fixture.componentInstance
        fixture.detectChanges()
    })

    it('should create', () => {
        expect(component).toBeTruthy()
    })
})
