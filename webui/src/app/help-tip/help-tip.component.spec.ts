import { ComponentFixture, TestBed, waitForAsync } from '@angular/core/testing'

import { HelpTipComponent } from './help-tip.component'
import { OverlayPanelModule } from 'primeng/overlaypanel'

describe('HelpTipComponent', () => {
    let component: HelpTipComponent
    let fixture: ComponentFixture<HelpTipComponent>

    beforeEach(waitForAsync(() => {
        TestBed.configureTestingModule({
            imports: [OverlayPanelModule],
            declarations: [HelpTipComponent],
        }).compileComponents()
    }))

    beforeEach(() => {
        fixture = TestBed.createComponent(HelpTipComponent)
        component = fixture.componentInstance
        fixture.detectChanges()
    })

    it('should create', () => {
        expect(component).toBeTruthy()
    })
})
