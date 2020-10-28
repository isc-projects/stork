import { async, ComponentFixture, TestBed } from '@angular/core/testing'
import { By } from '@angular/platform-browser'

import { HaStatusPanelComponent } from './ha-status-panel.component'

import { of } from 'rxjs'

describe('HaStatusPanelComponent', () => {
    let component: HaStatusPanelComponent
    let fixture: ComponentFixture<HaStatusPanelComponent>

    function itHasStatusIconAndText(icon, color, text) {
        // The table cell holding control status has an id. Let's access
        // that cell by id to verify that it contains both the icon and
        // the status text.
        const controlStatusTd = fixture.debugElement.query(By.css('#ha-control-status'))
        expect(controlStatusTd.children.length).toBe(2)

        // Both the icon and the text are within the <i> tags.
        const controlStatusTdChildren = controlStatusTd.queryAll(By.css('i'))
        expect(controlStatusTdChildren.length).toBe(2)

        // The first element should be an icon. Make sure that the appropriate
        // icon is displayed for the given status.
        const controlStatusIcon = controlStatusTdChildren[0]
        expect(controlStatusIcon.classes.hasOwnProperty('pi')).toBeTrue()
        expect(controlStatusIcon.classes.hasOwnProperty(icon)).toBeTrue()

        // The icon should have expected color.
        expect(controlStatusIcon.styles.hasOwnProperty('color')).toBeTrue()
        expect(controlStatusIcon.styles.color).toBe(color)

        // Finally, the status text should be present.
        const controlStatusNative = controlStatusTdChildren[1].nativeElement
        expect(controlStatusNative.textContent).toBe(text)
    }

    function itHasStateIconAndText(icon, color, text) {
        // The table cell holding server state has an id. Let's acces this
        // cell by id to verify that it contains both the icon and the
        // state text.
        const serverStateTd = fixture.debugElement.query(By.css('#ha-server-state'))
        expect(serverStateTd.children.length).toBe(2)

        // Both the icon and the text are within the <i> tags.
        const serverStateTdChildren = serverStateTd.queryAll(By.css('i'))
        expect(serverStateTdChildren.length).toBe(2)

        // The first element should be an icon. Make sure that the appropriate
        // icon is displayed for the given server state.
        const serverStateIcon = serverStateTdChildren[0]
        expect(serverStateIcon.classes.hasOwnProperty('pi')).toBeTrue()
        expect(serverStateIcon.classes.hasOwnProperty(icon)).toBeTrue()

        // The icon should have expected color.
        expect(serverStateIcon.styles.hasOwnProperty('color')).toBeTrue()
        expect(serverStateIcon.styles.color).toBe(color)

        // Finally, the state name should be present.
        const serverStateNative = serverStateTdChildren[1].nativeElement
        expect(serverStateNative.textContent).toBe(text)
    }

    beforeEach(async(() => {
        TestBed.configureTestingModule({
            declarations: [HaStatusPanelComponent],
        }).compileComponents()
    }))

    beforeEach(() => {
        fixture = TestBed.createComponent(HaStatusPanelComponent)
        component = fixture.componentInstance
    })

    it('should create', () => {
        component.serverStatus = of({ state: 'unavailable' })
        fixture.detectChanges()
        expect(component).toBeTruthy()
    })

    it('should present default HA status information when status was not fetched yet', () => {
        // The inTouch flag indicates that the communication with the server
        // via control channel is ok, but the HA state hasn't been fetched yet.
        component.serverStatus = { inTouch: true, role: 'primary' }
        fixture.detectChanges()

        // Control status testing.
        itHasStatusIconAndText('pi-check', 'rgb(0, 168, 0)', 'online')

        // Server state testing.
        itHasStateIconAndText('pi-spinner', 'gray', 'fetching...')

        // Scopes testing.
        const scopesList = fixture.debugElement.query(By.css('#ha-local-scopes'))
        expect(scopesList.nativeElement.textContent).toBe('(none)')
    })

    it('should present offline control status and unavailable server state', () => {
        // The inTouch flag indicates that there is no communication with the
        // server via control channel.
        component.serverStatus = { inTouch: false, role: 'primary', state: 'unavailable' }
        fixture.detectChanges()

        // Control status testing.
        itHasStatusIconAndText('pi-times', 'rgb(255, 17, 17)', 'offline')

        // Server state testing.
        itHasStateIconAndText('pi-times', 'rgb(255, 17, 17)', 'unavailable')
    })

    it('should present online control status and normal server state', () => {
        // Simulate the case when the communication with the server is fine and
        // when the server is working normally in the load-balancing state with
        // some scopes served.
        component.serverStatus = {
            inTouch: true,
            role: 'primary',
            state: 'load-balancing',
            scopes: ['server1', 'server2'],
        }
        fixture.detectChanges()

        // Control status testing.
        itHasStatusIconAndText('pi-check', 'rgb(0, 168, 0)', 'online')

        // Server state testing.
        itHasStateIconAndText('pi-check', 'rgb(0, 168, 0)', 'load-balancing')

        // Scopes testing.
        const scopesList = fixture.debugElement.query(By.css('#ha-local-scopes'))
        expect(scopesList.nativeElement.textContent).toBe('server1, server2')
    })
})
