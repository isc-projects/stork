import { ComponentFixture, TestBed } from '@angular/core/testing'

import { ManagedAccessComponent } from './managed-access.component'

describe('ManagedAccessComponent', () => {
    let component: ManagedAccessComponent
    let fixture: ComponentFixture<ManagedAccessComponent>

    beforeEach(async () => {
        await TestBed.configureTestingModule({
            imports: [ManagedAccessComponent],
        }).compileComponents()

        fixture = TestBed.createComponent(ManagedAccessComponent)
        component = fixture.componentInstance
        fixture.detectChanges()
    })

    it('should create', () => {
        expect(component).toBeTruthy()
    })
})
