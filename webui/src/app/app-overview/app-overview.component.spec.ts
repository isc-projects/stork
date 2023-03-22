import { ComponentFixture, TestBed } from '@angular/core/testing'
import { By } from '@angular/platform-browser'
import { FormsModule } from '@angular/forms'
import { NoopAnimationsModule } from '@angular/platform-browser/animations'
import { RouterTestingModule } from '@angular/router/testing'
import { PanelModule } from 'primeng/panel'
import { AppOverviewComponent } from './app-overview.component'
import { ButtonModule } from 'primeng/button'
import { AuthService } from '../auth.service'
import { HttpClientTestingModule } from '@angular/common/http/testing'
import { MessageService } from 'primeng/api'
import { App } from '../backend'
import { AccessPointKeyComponent } from '../access-point-key/access-point-key.component'

describe('AppOverviewComponent', () => {
    let component: AppOverviewComponent
    let fixture: ComponentFixture<AppOverviewComponent>
    let authService: AuthService

    beforeEach(async () => {
        await TestBed.configureTestingModule({
            imports: [
                FormsModule,
                NoopAnimationsModule,
                RouterTestingModule,
                HttpClientTestingModule,
                PanelModule,
                ButtonModule,
            ],
            providers: [
                MessageService,
                {
                    provide: AuthService,
                    useValue: {
                        superAdmin: () => true,
                    },
                },
            ],
            declarations: [AppOverviewComponent, AccessPointKeyComponent],
        }).compileComponents()
        authService = TestBed.inject(AuthService)
    })

    beforeEach(() => {
        fixture = TestBed.createComponent(AppOverviewComponent)
        component = fixture.componentInstance
        fixture.detectChanges()
    })

    it('should create', () => {
        expect(component).toBeTruthy()
    })

    it('should display access points', () => {
        const fakeApp: App = {
            machine: {
                id: 1,
                address: '192.0.2.1:8080',
            },
            accessPoints: [
                {
                    type: 'control',
                    address: '192.0.3.1',
                    port: 1234,
                    useSecureProtocol: true,
                },
                {
                    type: 'statistics',
                    address: '2001:db8:1::1',
                    port: 2345,
                    useSecureProtocol: false,
                },
            ],
        }
        component.app = fakeApp
        fixture.detectChanges()

        // There should be a table holding access points.
        const tableElement = fixture.debugElement.query(By.css('table'))
        expect(tableElement).toBeTruthy()

        const rows = tableElement.queryAll(By.css('tr'))
        expect(rows.length).toBe(3)

        // The first row holds the machine address.
        let tds = rows[0].queryAll(By.css('td'))
        expect(tds.length).toBe(2)
        expect(tds[0].nativeElement.innerText.trim()).toContain('Hosted on machine:')
        const machineLinkElement = tds[1].query(By.css('a'))
        expect(machineLinkElement).toBeTruthy()
        expect(machineLinkElement.attributes.href).toBe('/machines/1')

        // The second row holds the control access point.
        tds = rows[1].queryAll(By.css('td'))
        expect(tds.length).toBe(2)
        expect(tds[0].nativeElement.innerText.trim()).toContain('Control access point:')
        expect(tds[1].nativeElement.innerText.trim()).toContain('192.0.3.1:1234')

        // Ensure that the icon indicating secured connection is displayed.
        let iconSpan = tds[1].query(By.css('span'))
        expect(iconSpan).toBeTruthy()
        expect(iconSpan.classes.hasOwnProperty('pi-lock')).toBeTruthy()
        expect(iconSpan.attributes.pTooltip).toBe('secured connection')

        // The third row holds the statistics access point.
        tds = rows[2].queryAll(By.css('td'))
        expect(tds.length).toBe(2)
        expect(tds[0].nativeElement.innerText.trim()).toContain('Statistics access point:')
        expect(tds[1].nativeElement.innerText.trim()).toContain('[2001:db8:1::1]:2345')

        // Ensure that the icon indicating unsecured connection is displayed.
        iconSpan = tds[1].query(By.css('span'))
        expect(iconSpan).toBeTruthy()
        expect(iconSpan.classes.hasOwnProperty('pi-lock-open')).toBeTruthy()
        expect(iconSpan.attributes.pTooltip).toBe('unsecured connection')
    })

    it('should format address', () => {
        expect(component.formatAddress('192.0.2.1')).toBe('192.0.2.1')
        expect(component.formatAddress('')).toBe('')
        expect(component.formatAddress('[2001:db8:1::1]')).toBe('[2001:db8:1::1]')
        expect(component.formatAddress('2001:db8:1::2')).toBe('[2001:db8:1::2]')
    })

    it('should hide keys for non-super-admin users', () => {
        component.app = { type: 'bind9' }
        spyOn(authService, 'superAdmin').and.returnValue(false)
        expect(component.canShowKeys).toBeFalse()
    })

    it('should hide keys for non-BIND9 application', () => {
        component.app = { type: 'kea' }
        expect(component.canShowKeys).toBeFalse()
    })

    it('should show keys for BIND9 application and super-admin user', () => {
        component.app = { type: 'bind9' }
        expect(component.canShowKeys).toBeTrue()
    })
})
