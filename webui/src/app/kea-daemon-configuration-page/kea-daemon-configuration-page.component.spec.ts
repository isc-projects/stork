import { HttpErrorResponse } from '@angular/common/http'
import { HttpClientTestingModule } from '@angular/common/http/testing'
import { ComponentFixture, TestBed, waitForAsync } from '@angular/core/testing'
import { By } from '@angular/platform-browser'
import { NoopAnimationsModule } from '@angular/platform-browser/animations'
import { ActivatedRoute, convertToParamMap, Router, RouterModule } from '@angular/router'
import { RouterTestingModule } from '@angular/router/testing'
import { MessageService } from 'primeng/api'
import { BreadcrumbModule } from 'primeng/breadcrumb'
import { ButtonModule } from 'primeng/button'
import { MessageModule } from 'primeng/message'
import { OverlayPanelModule } from 'primeng/overlaypanel'
import { PanelModule } from 'primeng/panel'
import { of, throwError } from 'rxjs'
import { AuthService } from '../auth.service'
import { ServicesService, UsersService } from '../backend'
import { BreadcrumbsComponent } from '../breadcrumbs/breadcrumbs.component'
import { HelpTipComponent } from '../help-tip/help-tip.component'
import { JsonTreeRootComponent } from '../json-tree-root/json-tree-root.component'
import { JsonTreeComponent } from '../json-tree/json-tree.component'
import { ServerDataService } from '../server-data.service'

import { KeaDaemonConfigurationPageComponent } from './kea-daemon-configuration-page.component'

class MockParamMap {
    get(name: string): string | null {
        return null
    }
}

describe('KeaDaemonConfigurationPageComponent', () => {
    let component: KeaDaemonConfigurationPageComponent
    let fixture: ComponentFixture<KeaDaemonConfigurationPageComponent>
    let dataService: ServerDataService
    let userService: UsersService
    let authService: AuthService

    beforeEach(
        waitForAsync(() => {
            TestBed.configureTestingModule({
                imports: [
                    PanelModule,
                    ButtonModule,
                    RouterModule,
                    HttpClientTestingModule,
                    OverlayPanelModule,
                    NoopAnimationsModule,
                    MessageModule,
                    BreadcrumbModule,
                    RouterTestingModule.withRoutes([{ path: 'baz', component: KeaDaemonConfigurationPageComponent }]),
                ],
                declarations: [
                    KeaDaemonConfigurationPageComponent,
                    JsonTreeComponent,
                    JsonTreeRootComponent,
                    BreadcrumbsComponent,
                    HelpTipComponent,
                ],
                providers: [
                    ServicesService,
                    MessageService,
                    UsersService,
                    {
                        provide: ActivatedRoute,
                        useValue: {
                            snapshot: { queryParamMap: new MockParamMap() },
                            queryParamMap: of(new MockParamMap()),
                            paramMap: of(convertToParamMap({ appId: '1', daemonId: '2' })),
                        },
                    },
                ],
            })
            dataService = TestBed.inject(ServerDataService)
            userService = TestBed.inject(UsersService)
            authService = TestBed.inject(AuthService)
        })
    )

    beforeEach(() => {
        const fakeResponse: any = {
            id: 1,
            name: 'foo',
            details: {
                daemons: [
                    {
                        id: 2,
                        name: 'dhcp6',
                    },
                ],
            },
        }
        spyOn(dataService.servicesApi, 'getApp').and.returnValue(of(fakeResponse))
        spyOn(dataService.servicesApi, 'getDaemonConfig').and.returnValues(
            of({} as any),
            of({ foo: 42 } as any),
            of({ secret: 'SECRET', password: 'PASSWORD' } as any),
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

        authService.login('foo', 'bar', 'baz')

        fixture = TestBed.createComponent(KeaDaemonConfigurationPageComponent)
        component = fixture.componentInstance
        fixture.detectChanges()
    })

    it('should create', () => {
        expect(component).toBeTruthy()
    })

    it('should toggle expand nodes', () => {
        expect(component.autoExpand).toBe('none')
        component.onClickToggleNodes()

        expect(component.autoExpand).toBe('all')
        component.onClickToggleNodes()

        expect(component.autoExpand).toBe('none')
    })

    it('should set filename for download file', async () => {
        await fixture.whenStable()
        expect(component.downloadFilename).toBe('foo_DHCPv6.json')
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

        const messageElement = fixture.debugElement.query(By.css('.p-inline-message-text'))
        expect(messageElement).not.toBeNull()
        expect((messageElement.nativeElement as Element).textContent).toBe('Fetching daemon configuration failed')
    })
})
