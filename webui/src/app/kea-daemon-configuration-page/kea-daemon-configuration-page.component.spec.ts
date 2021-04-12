import { HttpErrorResponse } from '@angular/common/http'
import { HttpClientTestingModule } from '@angular/common/http/testing'
import { async, ComponentFixture, TestBed } from '@angular/core/testing'
import { By } from '@angular/platform-browser'
import { NoopAnimationsModule } from '@angular/platform-browser/animations'
import { ActivatedRoute, convertToParamMap, Router, RouterModule } from '@angular/router'
import { MessageService } from 'primeng/api'
import { ButtonModule } from 'primeng/button'
import { MessageModule } from 'primeng/message'
import { OverlayPanelModule } from 'primeng/overlaypanel'
import { PanelModule } from 'primeng/panel'
import { of, throwError } from 'rxjs'
import { AuthService } from '../auth.service'
import { ServicesService, UsersService } from '../backend'
import { BreadcrumbsComponent } from '../breadcrumbs/breadcrumbs.component'
import { HelpTipComponent } from '../help-tip/help-tip.component'
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
    let service: ServerDataService

    beforeEach(async(() => {
        TestBed.configureTestingModule({
            imports: [
                PanelModule,
                ButtonModule,
                RouterModule,
                HttpClientTestingModule,
                OverlayPanelModule,
                NoopAnimationsModule,
                MessageModule,
            ],
            declarations: [
                KeaDaemonConfigurationPageComponent,
                JsonTreeComponent,
                BreadcrumbsComponent,
                HelpTipComponent,
            ],
            providers: [
                ServicesService,
                MessageService,
                UsersService,
                {
                    provide: Router,
                    useValue: {},
                },
                {
                    provide: ActivatedRoute,
                    useValue: {
                        snapshot: { queryParamMap: new MockParamMap() },
                        queryParamMap: of(new MockParamMap()),
                        paramMap: of(convertToParamMap({ appId: '1', daemonId: '2' })),
                    },
                },
                {
                    provide: AuthService,
                    useValue: {
                        currentUser: of({}),
                    },
                },
            ],
        })
        service = TestBed.inject(ServerDataService)
    }))

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
        spyOn(service.servicesApi, 'getApp').and.returnValue(of(fakeResponse))
        spyOn(service.servicesApi, 'getDaemonConfig').and.returnValues(
            of({} as any),
            of({ foo: 42 } as any),
            throwError(new HttpErrorResponse({ status: 400 }))
        )

        fixture = TestBed.createComponent(KeaDaemonConfigurationPageComponent)
        component = fixture.componentInstance
        fixture.detectChanges()
    })

    it('should create', () => {
        expect(component).toBeTruthy()
    })

    it('should toggle expand nodes', () => {
        expect(component.autoExpandNodeCount).toBe(0)
        expect(component.currentAction).toBe('expand')
        component.onClickToggleNodes()

        expect(component.autoExpandNodeCount).toBe(Number.MAX_SAFE_INTEGER)
        expect(component.currentAction).toBe('collapse')
        component.onClickToggleNodes()

        expect(component.autoExpandNodeCount).toBe(0)
        expect(component.currentAction).toBe('expand')
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
        expect(component.configuration).toBeNull()
        expect(component.failedFetch).toBeTrue()

        const messageElement = fixture.debugElement.query(By.css('.ui-message-text'))
        expect(messageElement).not.toBeNull()
        expect((messageElement.nativeElement as Element).textContent).toBe('Fetching daemon configuration failed')
    })
})
