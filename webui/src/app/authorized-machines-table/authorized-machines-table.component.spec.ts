import { ComponentFixture, TestBed } from '@angular/core/testing'

import { AuthorizedMachinesTableComponent } from './authorized-machines-table.component'

describe('AuthorizedMachinesTableComponent', () => {
    let component: AuthorizedMachinesTableComponent
    let fixture: ComponentFixture<AuthorizedMachinesTableComponent>

    beforeEach(async () => {
        await TestBed.configureTestingModule({
            imports: [AuthorizedMachinesTableComponent],
        }).compileComponents()

        fixture = TestBed.createComponent(AuthorizedMachinesTableComponent)
        component = fixture.componentInstance
        fixture.detectChanges()
    })

    it('should create', () => {
        expect(component).toBeTruthy()
    })
})
