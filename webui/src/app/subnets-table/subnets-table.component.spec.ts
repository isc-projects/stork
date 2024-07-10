import { ComponentFixture, TestBed } from '@angular/core/testing'

import { SubnetsTableComponent } from './subnets-table.component'

describe('SubnetsTableComponent', () => {
    let component: SubnetsTableComponent
    let fixture: ComponentFixture<SubnetsTableComponent>

    beforeEach(async () => {
        await TestBed.configureTestingModule({
            imports: [SubnetsTableComponent],
        }).compileComponents()

        fixture = TestBed.createComponent(SubnetsTableComponent)
        component = fixture.componentInstance
        fixture.detectChanges()
    })

    it('should create', () => {
        expect(component).toBeTruthy()
    })
})
