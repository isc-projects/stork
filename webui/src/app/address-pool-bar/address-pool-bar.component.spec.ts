import { ComponentFixture, TestBed } from '@angular/core/testing'

import { AddressPoolBarComponent } from './address-pool-bar.component'

describe('AddressPoolBarComponent', () => {
    let component: AddressPoolBarComponent
    let fixture: ComponentFixture<AddressPoolBarComponent>

    beforeEach(async () => {
        await TestBed.configureTestingModule({
            declarations: [AddressPoolBarComponent],
        }).compileComponents()

        fixture = TestBed.createComponent(AddressPoolBarComponent)
        component = fixture.componentInstance
        fixture.detectChanges()
    })

    it('should create', () => {
        expect(component).toBeTruthy()
    })

    it('should display an address pool', () => {
        component.pool = {
            pool: '192.0.2.0/24',
        }
        fixture.detectChanges()

        expect(fixture.debugElement.nativeElement.innerText).toContain('192.0.2.0/24')
    })
})
