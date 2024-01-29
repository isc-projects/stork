import { ComponentFixture, TestBed } from '@angular/core/testing'
import { ChipModule } from 'primeng/chip'
import { NoopAnimationsModule } from '@angular/platform-browser/animations'

import { DhcpClientClassSetViewComponent } from './dhcp-client-class-set-view.component'
import { By } from '@angular/platform-browser'

describe('DhcpClientClassSetViewComponent', () => {
    let component: DhcpClientClassSetViewComponent
    let fixture: ComponentFixture<DhcpClientClassSetViewComponent>

    beforeEach(async () => {
        await TestBed.configureTestingModule({
            imports: [ChipModule, NoopAnimationsModule],
            declarations: [DhcpClientClassSetViewComponent],
        }).compileComponents()

        fixture = TestBed.createComponent(DhcpClientClassSetViewComponent)
        component = fixture.componentInstance
        fixture.detectChanges()
    })

    it('should create', () => {
        expect(component).toBeTruthy()
    })

    it('should display client classses', () => {
        component.clientClasses = ['access-point', 'router', 'classifier']
        fixture.detectChanges()

        const chips = fixture.debugElement.queryAll(By.css('p-chip'))
        expect(chips.length).toBe(3)
    })

    it('should display a note that there are no client classes', () => {
        component.clientClasses = []
        fixture.detectChanges()

        const chips = fixture.debugElement.queryAll(By.css('p-chip'))
        expect(chips.length).toBe(0)
        expect(fixture.debugElement.nativeElement.innerText).toContain('No client classes configured.')
    })
})
