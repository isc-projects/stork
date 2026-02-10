import { HttpErrorResponse, provideHttpClient, withInterceptorsFromDi } from '@angular/common/http'
import { provideHttpClientTesting } from '@angular/common/http/testing'
import { ComponentFixture, TestBed, waitForAsync } from '@angular/core/testing'
import { By } from '@angular/platform-browser'
import { provideNoopAnimations } from '@angular/platform-browser/animations'
import { provideRouter } from '@angular/router'
import { MessageService } from 'primeng/api'
import { of, throwError } from 'rxjs'
import { AuthService } from '../auth.service'
import { UsersService } from '../backend'
import { BreadcrumbsComponent } from '../breadcrumbs/breadcrumbs.component'
import { ServerDataService } from '../server-data.service'

import { KeaDaemonConfigurationPageComponent } from './kea-daemon-configuration-page.component'
import { RouterTestingHarness } from '@angular/router/testing'

describe('KeaDaemonConfigurationPageComponent', () => {
    let component: KeaDaemonConfigurationPageComponent
    let fixture: ComponentFixture<unknown>
    let dataService: ServerDataService
    let userService: UsersService
    let authService: AuthService

    beforeEach(waitForAsync(() => {
        TestBed.configureTestingModule({
            providers: [
                MessageService,
                provideHttpClient(withInterceptorsFromDi()),
                provideHttpClientTesting(),
                provideNoopAnimations(),
                provideRouter([{ path: 'daemons/:daemonId/config', component: KeaDaemonConfigurationPageComponent }]),
            ],
        })
        dataService = TestBed.inject(ServerDataService)
        userService = TestBed.inject(UsersService)
        authService = TestBed.inject(AuthService)
    }))

    beforeEach(async () => {
        spyOn(dataService.servicesApi, 'getDaemonConfig').and.returnValues(
            of({
                config: {},
            } as any),
            of({
                config: {
                    foo: 42,
                },
            } as any),
            of({
                config: {
                    secret: 'SECRET',
                    password: 'PASSWORD',
                },
            } as any),
            throwError(new HttpErrorResponse({ status: 400 }))
        )

        spyOn(userService, 'createSession').and.returnValues(
            of({
                id: 1,
                login: 'foo',
                email: 'foo@bar.baz',
                name: 'foo',
                lastname: 'bar',
                groups: [],
            } as any),
            of({
                id: 1,
                login: 'foo',
                email: 'foo@bar.baz',
                name: 'foo',
                lastname: 'bar',
                groups: [1],
            } as any)
        )

        authService.login('boz', 'foo', 'bar', 'baz')

        const harness = await RouterTestingHarness.create()
        component = await harness.navigateByUrl('/daemons/2/config', KeaDaemonConfigurationPageComponent)
        fixture = harness.fixture

        fixture.detectChanges()
    })

    it('should create', () => {
        expect(component).toBeTruthy()
    })

    it('should toggle expand nodes', async () => {
        await fixture.whenStable()
        component.onClickRefresh()
        fixture.detectChanges()

        expect(component.autoExpand).toBe('none')
        let expectedButtonCount = fixture.debugElement
            .queryAll(By.css('.p-panel-header > div:not(.p-panel-icons) > *'))
            .map((b) => (b.nativeElement as HTMLElement).textContent.trim())
            .filter((t) => t === 'Expand').length
        expect(expectedButtonCount).toBe(1)

        component.onClickToggleNodes()
        fixture.detectChanges()
        await fixture.whenRenderingDone()
        expect(component.autoExpand).toBe('all')
        expectedButtonCount = fixture.debugElement
            .queryAll(By.css('.p-panel-header > div:not(.p-panel-icons) > *'))
            .map((b) => (b.nativeElement as HTMLElement).textContent.trim())
            .filter((t) => t === 'Collapse').length
        expect(expectedButtonCount).toBe(1)

        component.onClickToggleNodes()
        fixture.detectChanges()
        await fixture.whenRenderingDone()
        expect(component.autoExpand).toBe('none')
        expectedButtonCount = fixture.debugElement
            .queryAll(By.css('.p-panel-header > div:not(.p-panel-icons) > *'))
            .map((b) => (b.nativeElement as HTMLElement).textContent.trim())
            .filter((t) => t === 'Expand').length
        expect(expectedButtonCount).toBe(1)
    })

    it('should set filename for download file', async () => {
        await fixture.whenStable()
        expect(component.downloadFilename).toBe('daemon_2.json')
    })

    it('should set daemon id', async () => {
        await fixture.whenStable()
        expect(component.daemonId).toBe(2)
    })

    it('should refresh configuration', async () => {
        await fixture.whenStable()
        expect(component.configuration).toEqual({})

        component.onClickRefresh()

        expect(component.configuration).toEqual({ foo: 42 })
    })

    it('should handle error fetch', async () => {
        await fixture.whenStable()
        expect(component.failedFetch).toBeFalse()

        component.onClickRefresh()
        await fixture.whenStable()
        expect(component.failedFetch).toBeFalse()

        component.onClickRefresh()
        fixture.detectChanges()
        await fixture.whenRenderingDone()

        component.onClickRefresh()
        fixture.detectChanges()
        await fixture.whenRenderingDone()
        expect(component.configuration).toBeNull()
        expect(component.failedFetch).toBeTrue()

        const messageElement = fixture.debugElement.query(By.css('.p-message'))
        expect(messageElement).not.toBeNull()
        expect((messageElement.nativeElement as Element).textContent).toBe('Fetching daemon configuration failed')
    })

    it('should have breadcrumbs', async () => {
        await fixture.whenStable()
        const breadcrumbsElement = fixture.debugElement.query(By.directive(BreadcrumbsComponent))
        expect(breadcrumbsElement).not.toBeNull()
        const breadcrumbsComponent = breadcrumbsElement.componentInstance as BreadcrumbsComponent
        expect(breadcrumbsComponent).not.toBeNull()
        expect(breadcrumbsComponent.items).toHaveSize(4)
        expect(breadcrumbsComponent.items[0].label).toEqual('Services')
        expect(breadcrumbsComponent.items[1].label).toEqual('Kea Daemons')
        expect(breadcrumbsComponent.items[2].label).toEqual('Daemon')
        expect(breadcrumbsComponent.items[3].label).toEqual('Configuration')
    })
})
