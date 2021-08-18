import { HttpErrorResponse } from '@angular/common/http'
import { HttpClientTestingModule } from '@angular/common/http/testing'
import { ComponentFixture, TestBed, waitForAsync } from '@angular/core/testing'
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
                        useValue: {
                            navigate: () => {},
                        },
                    },
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

        component.onClickRefresh()
        fixture.detectChanges()
        await fixture.whenRenderingDone()
        expect(component.configuration).toBeNull()
        expect(component.failedFetch).toBeTrue()

        const messageElement = fixture.debugElement.query(By.css('.p-inline-message-text'))
        expect(messageElement).not.toBeNull()
        expect((messageElement.nativeElement as Element).textContent).toBe('Fetching daemon configuration failed')
    })

    it('should have custom value templates', async () => {
        await fixture.whenStable()
        const jsonElement = fixture.debugElement.query(By.directive(JsonTreeComponent))
        expect(jsonElement).toBeDefined()
        const jsonComponent = jsonElement.componentInstance as JsonTreeComponent
        expect(jsonComponent).not.toBeNull()
        const templates = jsonComponent.customValueTemplates
        expect(Object.keys(templates)).toContain('password')
        expect(Object.keys(templates)).toContain('secret')
    })

    it('should hide the secrets', async () => {
        // Move to configuration with secrets
        await fixture.whenStable()
        for (let i = 0; i < 2; i++) {
            component.onClickRefresh()
            await fixture.whenStable()
        }
        fixture.detectChanges()

        expect(component.configuration).toEqual({ secret: 'SECRET', password: 'PASSWORD' })

        // Extract JSON viewer component
        const jsonElement = fixture.debugElement.query(By.directive(JsonTreeComponent))
        expect(jsonElement).toBeDefined()
        const jsonComponent = jsonElement.componentInstance as JsonTreeComponent
        expect(jsonComponent).not.toBeNull()
        console.log(jsonComponent.value)
        expect(jsonComponent.value).toEqual({ secret: 'SECRET', password: 'PASSWORD' })

        // Extract specific levels
        const valueElements = jsonElement.queryAll(By.css('.tree-level--leaf .tree-level__value'))
        expect(valueElements.length).toBe(2)

        for (const valueElement of valueElements) {
            const valueNativeElement = valueElement.nativeElement as HTMLElement
            // We extract only visible text - the secret should be hidden
            const content = valueNativeElement.innerText
            expect(content).toBeFalsy()
        }
    })

    it('should show the secrets after click when user is super admin', async () => {
        // Log-out and log-in, now user is super admin
        authService.logout()
        await fixture.whenStable()
        authService.login('foo', 'bar', 'baz')
        await fixture.whenStable()
        fixture.detectChanges()

        expect(component.canShowSecrets).toBeTrue()

        // Move to configuration with secrets
        await fixture.whenStable()
        component.onClickRefresh()
        await fixture.whenStable()
        fixture.detectChanges()

        expect(component.configuration).toEqual({ secret: 'SECRET', password: 'PASSWORD' })

        // Extract JSON viewer component
        const jsonElement = fixture.debugElement.query(By.directive(JsonTreeComponent))
        expect(jsonElement).toBeDefined()
        const jsonComponent = jsonElement.componentInstance as JsonTreeComponent
        expect(jsonComponent).not.toBeNull()
        console.log(jsonComponent.value)
        expect(jsonComponent.value).toEqual({ secret: 'SECRET', password: 'PASSWORD' })

        // Extract specific levels
        const keyElements = jsonElement.queryAll(By.css('.tree-level--leaf .tree-level__key'))
        expect(keyElements.length).toBe(2)

        for (const keyElement of keyElements) {
            const key = (keyElement.nativeElement as HTMLElement).textContent.trim()
            const expectedValue = key.toUpperCase()
            const leafElement = keyElement.parent
            const valueElement = leafElement.query(By.css('.tree-level__value'))
            expect(valueElement).not.toBeNull()
            const valueNativeElement = valueElement.nativeElement as HTMLElement

            // We extract only visible text - the secret should be hidden
            let content = valueNativeElement.innerText
            expect(content).toBeFalsy()

            // Click on hidden value
            const summaryElement = valueElement.query(By.css('summary'))
            expect(summaryElement).not.toBeNull()
            const summaryNativeElement = summaryElement.nativeElement as HTMLElement
            summaryNativeElement.click()
            await fixture.whenRenderingDone()
            content = valueNativeElement.innerText.trim()
            expect(content).toBe(expectedValue)
        }
    })

    it('should ignore click on the secret field when user is not a super admin', async () => {
        expect(component.canShowSecrets).toBeFalse()
        // Move to configuration with secrets
        await fixture.whenStable()
        for (let i = 0; i < 2; i++) {
            component.onClickRefresh()
            await fixture.whenStable()
        }
        fixture.detectChanges()

        expect(component.configuration).toEqual({ secret: 'SECRET', password: 'PASSWORD' })

        // Extract JSON viewer component
        const jsonElement = fixture.debugElement.query(By.directive(JsonTreeComponent))
        expect(jsonElement).toBeDefined()
        const jsonComponent = jsonElement.componentInstance as JsonTreeComponent
        expect(jsonComponent).not.toBeNull()
        console.log(jsonComponent.value)
        expect(jsonComponent.value).toEqual({ secret: 'SECRET', password: 'PASSWORD' })

        // Extract specific levels
        const keyElements = jsonElement.queryAll(By.css('.tree-level--leaf .tree-level__key'))
        expect(keyElements.length).toBe(2)

        for (const keyElement of keyElements) {
            const key = (keyElement.nativeElement as HTMLElement).textContent.trim()
            const expectedValue = key.toUpperCase()
            const leafElement = keyElement.parent
            const valueElement = leafElement.query(By.css('.tree-level__value'))
            expect(valueElement).not.toBeNull()
            const valueNativeElement = valueElement.nativeElement as HTMLElement

            // We extract only visible text - the secret should be hidden
            let content = valueNativeElement.innerText
            expect(content).toBeFalsy()

            // Click on hidden value
            const summaryElement = valueElement.query(By.css('summary'))
            expect(summaryElement).not.toBeNull()
            const summaryNativeElement = summaryElement.nativeElement as HTMLElement
            summaryNativeElement.click()
            await fixture.whenRenderingDone()
            content = valueNativeElement.innerText.trim()
            expect(content).toBeFalsy() // Nothing changed
        }
    })
})
