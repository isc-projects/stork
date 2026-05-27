import { ComponentFixture, TestBed } from '@angular/core/testing'
import { provideNoopAnimations } from '@angular/platform-browser/animations'

import { DhcpClientClassSetViewComponent } from './dhcp-client-class-set-view.component'
import { By } from '@angular/platform-browser'

describe('DhcpClientClassSetViewComponent', () => {
    let component: DhcpClientClassSetViewComponent
    let fixture: ComponentFixture<DhcpClientClassSetViewComponent>

    beforeEach(async () => {
        await TestBed.configureTestingModule({
            providers: [provideNoopAnimations()],
        }).compileComponents()

        fixture = TestBed.createComponent(DhcpClientClassSetViewComponent)
        component = fixture.componentInstance
    })

    it('should create', () => {
        expect(component).toBeTruthy()
    })

    it('should display client classses', () => {
        fixture.componentRef.setInput('clientClasses', ['access-point', 'router', 'classifier'])
        fixture.detectChanges()

        expect(component.clientClasses.length).toBe(3)
        const chips = fixture.debugElement.queryAll(By.css('p-chip'))
        expect(chips.length).toBe(3)
    })

    it('should display a note that there are no client classes', () => {
        fixture.componentRef.setInput('clientClasses', [])
        fixture.detectChanges()

        expect(component.clientClasses.length).toBe(0)
        const chips = fixture.debugElement.queryAll(By.css('p-chip'))
        expect(chips.length).toBe(0)
        expect(fixture.debugElement.nativeElement.textContent).toContain('No client classes configured.')
    })
})
