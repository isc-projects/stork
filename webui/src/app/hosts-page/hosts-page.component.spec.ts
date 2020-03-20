import { async, ComponentFixture, TestBed } from '@angular/core/testing'

import { HostsPageComponent } from './hosts-page.component'

describe('HostsPageComponent', () => {
    let component: HostsPageComponent
    let fixture: ComponentFixture<HostsPageComponent>

    beforeEach(async(() => {
        TestBed.configureTestingModule({
            declarations: [HostsPageComponent],
        }).compileComponents()
    }))

    beforeEach(() => {
        fixture = TestBed.createComponent(HostsPageComponent)
        component = fixture.componentInstance
        fixture.detectChanges()
    })

    it('should create', () => {
        expect(component).toBeTruthy()
    })
})
