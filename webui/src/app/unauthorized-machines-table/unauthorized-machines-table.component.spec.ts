import { ComponentFixture, TestBed } from '@angular/core/testing'

import { UnauthorizedMachinesTableComponent } from './unauthorized-machines-table.component'

describe('UnauthorizedMachinesTableComponent', () => {
    let component: UnauthorizedMachinesTableComponent
    let fixture: ComponentFixture<UnauthorizedMachinesTableComponent>

    beforeEach(async () => {
        await TestBed.configureTestingModule({
            imports: [UnauthorizedMachinesTableComponent],
        }).compileComponents()

        fixture = TestBed.createComponent(UnauthorizedMachinesTableComponent)
        component = fixture.componentInstance
        fixture.detectChanges()
    })

    it('should create', () => {
        expect(component).toBeTruthy()
    })
})
