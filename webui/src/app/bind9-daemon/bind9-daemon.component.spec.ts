import { ComponentFixture, TestBed } from '@angular/core/testing'

import { Bind9DaemonComponent } from './bind9-daemon.component'
import { AppsVersions, Bind9DaemonView, Daemon } from '../backend'
import { By } from '@angular/platform-browser'
import { VersionStatusComponent } from '../version-status/version-status.component'
import { Severity, VersionService } from '../version.service'
import { ConfirmationService, MessageService } from 'primeng/api'
import { of } from 'rxjs'
import { provideRouter } from '@angular/router'
import { provideHttpClientTesting } from '@angular/common/http/testing'
import { provideHttpClient, withInterceptorsFromDi } from '@angular/common/http'

describe('Bind9DaemonComponent', () => {
    let component: Bind9DaemonComponent
    let fixture: ComponentFixture<Bind9DaemonComponent>
    let versionServiceStub: Partial<VersionService>

    beforeEach(async () => {
        versionServiceStub = {
            sanitizeSemver: () => '9.18.30',
            getCurrentData: () => of({} as AppsVersions),
            getSoftwareVersionFeedback: () => ({ severity: Severity.success, messages: ['test feedback'] }),
        }
        await TestBed.configureTestingModule({
            providers: [
                { provide: VersionService, useValue: versionServiceStub },
                MessageService,
                ConfirmationService,
                provideRouter([]),
                provideHttpClient(withInterceptorsFromDi()),
                provideHttpClientTesting(),
            ],
        }).compileComponents()

        fixture = TestBed.createComponent(Bind9DaemonComponent)
        fixture.debugElement.injector.get(VersionService)
        component = fixture.componentInstance
        const daemon: Daemon = {
            id: 1,
            name: Daemon.NameEnum.Named,
            version: '9.18.30'
        }
        component.daemon = daemon
        fixture.detectChanges()
    })

    it('should create', () => {
        expect(component).toBeTruthy()
    })

    it('should return 0 when queryHitRatio is undefined', () => {
        const view = {} as Bind9DaemonView
        expect(component.getQueryUtilization(view)).toBe(0)
    })

    it('should calculate correct utilization percentage', () => {
        const view = { queryHitRatio: 0.756 } as Bind9DaemonView
        expect(component.getQueryUtilization(view)).toBe(75)
    })

    it('should display version status component', () => {
        // One VersionStatus BIND9.
        let versionStatus = fixture.debugElement.queryAll(By.directive(VersionStatusComponent))
        expect(versionStatus).toBeTruthy()
        expect(versionStatus.length).toEqual(1)
        // Stubbed success icon for BIND 9.18.30 is expected.
        expect(versionStatus[0].nativeElement.innerHTML).toContain('9.18.30')
        expect(versionStatus[0].nativeElement.innerHTML).not.toContain('named')
        expect(versionStatus[0].nativeElement.innerHTML).toContain('text-green-500')
        expect(versionStatus[0].nativeElement.innerHTML).toContain('test feedback')
    })
})
