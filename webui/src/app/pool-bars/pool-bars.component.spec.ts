import { ComponentFixture, TestBed } from '@angular/core/testing'

import { PoolBarsComponent } from './pool-bars.component'

describe('PoolBarsComponent', () => {
    let component: PoolBarsComponent
    let fixture: ComponentFixture<PoolBarsComponent>

    beforeEach(async () => {
        await TestBed.configureTestingModule({
            declarations: [PoolBarsComponent],
        }).compileComponents()

        fixture = TestBed.createComponent(PoolBarsComponent)
        component = fixture.componentInstance
        fixture.detectChanges()
    })

    it('should create', () => {
        expect(component).toBeTruthy()
    })
})
