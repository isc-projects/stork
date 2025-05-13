import { ComponentFixture, TestBed } from '@angular/core/testing'

import { UtilizationBarComponent } from './utilization-bar.component'
import { TooltipModule } from 'primeng/tooltip'

describe('UtilizationBarComponent', () => {
    let component: UtilizationBarComponent
    let fixture: ComponentFixture<UtilizationBarComponent>

    beforeEach(async () => {
        await TestBed.configureTestingModule({
            imports: [TooltipModule],
            declarations: [UtilizationBarComponent],
        }).compileComponents()

        fixture = TestBed.createComponent(UtilizationBarComponent)
        component = fixture.componentInstance
        fixture.detectChanges()
    })

    it('should create', () => {
        expect(component).toBeTruthy()
    })
})
