import { ComponentFixture, TestBed, waitForAsync } from '@angular/core/testing'

import { HelpTipComponent } from './help-tip.component'

describe('HelpTipComponent', () => {
    let component: HelpTipComponent
    let fixture: ComponentFixture<HelpTipComponent>

    beforeEach(waitForAsync(() => {
        TestBed.configureTestingModule({}).compileComponents()
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
