import { ComponentFixture, TestBed } from '@angular/core/testing'
import { By } from '@angular/platform-browser'
import { provideNoopAnimations } from '@angular/platform-browser/animations'
import { AuthService } from '../auth.service'
import { provideHttpClientTesting } from '@angular/common/http/testing'
import { MessageService } from 'primeng/api'
import { AnyDaemon } from '../backend'
import { AccessPointKeyComponent } from '../access-point-key/access-point-key.component'
import { provideHttpClient, withInterceptorsFromDi } from '@angular/common/http'
import { provideRouter } from '@angular/router'
import { AccessPointsComponent } from './access-points.component'

describe('AccessPointsComponent', () => {
    let component: AccessPointsComponent
    let fixture: ComponentFixture<AccessPointsComponent>
    let authService: AuthService

    beforeEach(async () => {
        await TestBed.configureTestingModule({
            providers: [
                MessageService,
                {
                    provide: AuthService,
                    useValue: {
                        superAdmin: () => true,
                        hasPrivilege: () => true,
                    },
                },
                provideNoopAnimations(),
                provideHttpClient(withInterceptorsFromDi()),
                provideHttpClientTesting(),
                provideRouter([]),
            ],
        }).compileComponents()
        authService = TestBed.inject(AuthService)
    })

    beforeEach(() => {
        fixture = TestBed.createComponent(AccessPointsComponent)
        component = fixture.componentInstance
        component.daemon = { accessPoints: [] }
        fixture.detectChanges()
    })

    it('should create', () => {
        expect(component).toBeTruthy()
    })

    it('should display control access point', () => {
        component.daemon = {
            machine: {
                id: 1,
                address: '192.0.2.1:8080',
            },
            accessPoints: [
                {
                    type: 'control',
                    address: '192.0.3.1',
                    port: 1234,
                    protocol: 'https',
                },
            ],
        }
        fixture.detectChanges()

        const content = (fixture.nativeElement as HTMLElement).innerText

        expect(content).toContain('control')
        expect(content).toContain('192.0.3.1:1234')

        const icon = fixture.debugElement.query(By.css('.pi-lock'))
        expect(icon).toBeTruthy()
        expect(icon.attributes.pTooltip).toBe('secured connection')
    })

    it('should display statistics access point', () => {
        component.daemon = {
            machine: {
                id: 1,
                address: '192.0.2.1:8080',
            },
            accessPoints: [
                {
                    type: 'statistics',
                    address: '2001:db8:1::1',
                    port: 2345,
                    protocol: 'http',
                },
            ],
        }
        fixture.detectChanges()

        const content = (fixture.nativeElement as HTMLElement).innerText

        expect(content).toContain('statistics')
        expect(content).toContain('[2001:db8:1::1]:2345')

        const icon = fixture.debugElement.query(By.css('.pi-lock-open'))
        expect(icon).toBeTruthy()
        expect(icon.attributes.pTooltip).toBe('unsecured connection')
    })

    it('should display multiple access points', () => {
        component.daemon = {
            machine: {
                id: 1,
                address: '192.0.2.1:8080',
            },
            accessPoints: [
                {
                    type: 'control',
                    address: '192.0.3.1',
                    port: 1234,
                    protocol: 'https',
                },
                {
                    type: 'statistics',
                    address: '2001:db8:1::1',
                    port: 2345,
                    protocol: 'http',
                },
            ],
        }
        fixture.detectChanges()

        const content = (fixture.nativeElement as HTMLElement).innerText

        expect(content).toContain('control')
        expect(content).toContain('statistics')
    })

    it('should format address', () => {
        expect(component.formatAddress('192.0.2.1')).toBe('192.0.2.1')
        expect(component.formatAddress('')).toBe('')
        expect(component.formatAddress('[2001:db8:1::1]')).toBe('[2001:db8:1::1]')
        expect(component.formatAddress('2001:db8:1::2')).toBe('[2001:db8:1::2]')
    })

    it('should hide keys for non-super-admin users', () => {
        spyOn(authService, 'hasPrivilege').and.returnValue(false)
        spyOn(authService, 'superAdmin').and.returnValue(false)
        fixture.componentRef.setInput('daemon', {
            name: 'named',
            machine: { id: 1, address: '192.0.2.1:8080' },
            accessPoints: [{ address: '192.0.2.1', port: 8080, protocol: 'rndc', type: 'control' }],
            id: 1,
        } as AnyDaemon)
        fixture.detectChanges()
        expect(authService.hasPrivilege).toHaveBeenCalled()
        const spanDE = fixture.debugElement.query(By.css('span#access-point-key'))
        expect(spanDE).toBeTruthy()
        expect(spanDE.nativeElement.innerText).toBe('')
    })

    it('should hide keys for non-BIND9 daemon', () => {
        component.daemon = { name: 'dhcp4' } as AnyDaemon
        expect(fixture.debugElement.query(By.directive(AccessPointKeyComponent))).toBeFalsy()
    })

    it('should show keys for BIND9 daemon and super-admin user', () => {
        spyOn(authService, 'hasPrivilege').and.returnValue(true)
        fixture.componentRef.setInput('daemon', {
            name: 'named',
            machine: { id: 1, address: '192.0.2.1:8080' },
            accessPoints: [{ address: '192.0.2.1', port: 8080, protocol: 'rndc', type: 'control' }],
            id: 1,
        } as AnyDaemon)
        fixture.detectChanges()
        expect(authService.hasPrivilege).toHaveBeenCalled()
        expect(fixture.debugElement.query(By.directive(AccessPointKeyComponent))).toBeTruthy()
    })
})
