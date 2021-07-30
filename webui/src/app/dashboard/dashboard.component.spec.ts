import { ComponentFixture, TestBed, waitForAsync } from '@angular/core/testing'

import { DashboardComponent } from './dashboard.component'
import { PanelModule } from 'primeng/panel'
import { ButtonModule } from 'primeng/button'
import { Router, RouterModule, ActivatedRoute } from '@angular/router'
import { ServicesService, DHCPService, SettingsService, UsersService } from '../backend'
import { HttpClientTestingModule } from '@angular/common/http/testing'
import { MessageService } from 'primeng/api'
import { LocationStrategy } from '@angular/common'

describe('DashboardComponent', () => {
    let component: DashboardComponent
    let fixture: ComponentFixture<DashboardComponent>

    beforeEach(
        waitForAsync(() => {
            TestBed.configureTestingModule({
                imports: [PanelModule, ButtonModule, RouterModule, HttpClientTestingModule],
                declarations: [DashboardComponent],
                providers: [
                    ServicesService,
                    LocationStrategy,
                    DHCPService,
                    MessageService,
                    UsersService,
                    SettingsService,
                    {
                        provide: Router,
                        useValue: {},
                    },
                    {
                        provide: ActivatedRoute,
                        useValue: {},
                    },
                ],
            }).compileComponents()
        })
    )

    beforeEach(() => {
        fixture = TestBed.createComponent(DashboardComponent)
        component = fixture.componentInstance
        fixture.detectChanges()
    })

    it('should create', () => {
        expect(component).toBeTruthy()
    })

    it('should indicate that HA is not enabled', () => {
        // This test doesn't check that the state is rendered correctly
        // as HTML, because the table listing daemons is dynamic and
        // finding the right table cell is going to be involved. Instead
        // we test it indirectly by making sure that the functions used
        // to render the content return expected values.
        const daemon = { haState: 'load-balancing', haFailureAt: '2014-06-01T12:00:00Z' }
        expect(component.showHAState(daemon)).toBe('not configured')
        expect(component.showHAFailureTime(daemon)).toBe('')
        expect(component.haStateIcon(daemon)).toBe('ban')

        const daemon2 = { haEnabled: false, haState: 'load-balancing', haFailureAt: '2014-06-01T12:00:00Z' }
        expect(component.showHAState(daemon2)).toBe('not configured')
        expect(component.showHAFailureTime(daemon2)).toBe('')
        expect(component.haStateIcon(daemon2)).toBe('ban')

        const daemon3 = { haEnabled: true, haState: '', haFailureAt: '0001-01-01' }
        expect(component.showHAState(daemon3)).toBe('fetching...')
        expect(component.showHAFailureTime(daemon3)).toBe('')
        expect(component.haStateIcon(daemon3)).toBe('spin pi-spinner')

        const daemon4 = { haEnabled: true, haFailureAt: '0001-01-01' }
        expect(component.showHAState(daemon4)).toBe('fetching...')
        expect(component.showHAFailureTime(daemon4)).toBe('')
        expect(component.haStateIcon(daemon4)).toBe('spin pi-spinner')
    })
})
