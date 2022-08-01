import { ComponentFixture, TestBed } from '@angular/core/testing'

import { DhcpOptionSetViewComponent } from './dhcp-option-set-view.component'

describe('DhcpOptionSetViewComponent', () => {
    let component: DhcpOptionSetViewComponent
    let fixture: ComponentFixture<DhcpOptionSetViewComponent>

    beforeEach(async () => {
        await TestBed.configureTestingModule({
            declarations: [DhcpOptionSetViewComponent],
        }).compileComponents()
    })

    beforeEach(() => {
        fixture = TestBed.createComponent(DhcpOptionSetViewComponent)
        component = fixture.componentInstance
        fixture.detectChanges()
    })

    it('should create', () => {
        expect(component).toBeTruthy()
    })
})
